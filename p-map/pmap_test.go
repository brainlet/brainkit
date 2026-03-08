package pmap

// Faithful port of p-map test.js with minimal Go adaptations.
// TS source: https://github.com/sindresorhus/p-map/blob/main/test.js

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type delayedInput struct {
	value any
	delay time.Duration
}

type indexedValue struct {
	Value int
	Index int
}

func sharedInput() []any {
	return []any{
		delayedInput{
			value: func() (int, error) { return 10, nil },
			delay: 300 * time.Millisecond,
		},
		delayedInput{
			value: 20,
			delay: 200 * time.Millisecond,
		},
		AwaitableFunc[delayedInput](func(context.Context) (delayedInput, error) {
			return delayedInput{
				value: 30,
				delay: 100 * time.Millisecond,
			}, nil
		}),
	}
}

func longerSharedInput() []any {
	return []any{
		delayedInput{value: 10, delay: 300 * time.Millisecond},
		delayedInput{value: 20, delay: 200 * time.Millisecond},
		delayedInput{value: 30, delay: 100 * time.Millisecond},
		delayedInput{value: 40, delay: 50 * time.Millisecond},
		delayedInput{value: 50, delay: 25 * time.Millisecond},
	}
}

func errorInput1() []any {
	return []any{
		delayedInput{value: 20, delay: 200 * time.Millisecond},
		delayedInput{value: 30, delay: 100 * time.Millisecond},
		delayedInput{
			value: func() (int, error) { return 0, errors.New("foo") },
			delay: 10 * time.Millisecond,
		},
		delayedInput{
			value: func() (int, error) { return 0, errors.New("bar") },
			delay: 10 * time.Millisecond,
		},
	}
}

func errorInput2() []any {
	return []any{
		delayedInput{value: 20, delay: 200 * time.Millisecond},
		delayedInput{
			value: func() (int, error) { return 0, errors.New("bar") },
			delay: 10 * time.Millisecond,
		},
		delayedInput{value: 30, delay: 100 * time.Millisecond},
		delayedInput{
			value: func() (int, error) { return 0, errors.New("foo") },
			delay: 10 * time.Millisecond,
		},
	}
}

func errorInput3() []any {
	return []any{
		delayedInput{value: 20, delay: 10 * time.Millisecond},
		delayedInput{
			value: func() (int, error) { return 0, errors.New("bar") },
			delay: 100 * time.Millisecond,
		},
		delayedInput{value: 30, delay: 100 * time.Millisecond},
	}
}

func mapper(input delayedInput, _ int) (any, error) {
	time.Sleep(input.delay)

	switch value := input.value.(type) {
	case func() (int, error):
		return value()
	case int:
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected mapper input value type %T", input.value)
	}
}

func mapperWithIndex(input delayedInput, index int) (any, error) {
	value, err := mapper(input, index)
	if err != nil {
		return nil, err
	}

	return indexedValue{
		Value: value.(int),
		Index: index,
	}, nil
}

type asyncTestData struct {
	data  []any
	index int
}

func newAsyncTestData(data []any) *asyncTestData {
	return &asyncTestData{data: data}
}

func (a *asyncTestData) Next(context.Context) (value any, done bool, err error) {
	if a.index >= len(a.data) {
		return nil, true, nil
	}

	time.Sleep(10 * time.Millisecond)
	value = a.data[a.index]
	a.index++
	return value, false, nil
}

type throwingIterator struct {
	max          int
	throwOnIndex int
	index        int
}

func newThrowingIterator(max, throwOnIndex int) *throwingIterator {
	return &throwingIterator{
		max:          max,
		throwOnIndex: throwOnIndex,
	}
}

func (t *throwingIterator) Next(context.Context) (value any, done bool, err error) {
	current := t.index
	defer func() {
		t.index++
	}()

	if current == t.throwOnIndex {
		return nil, false, fmt.Errorf("throwing on index %d", current)
	}

	return current, current == t.max, nil
}

func collectAsyncIterable[T any](t *testing.T, iterable AsyncIterable[T]) ([]T, error) {
	t.Helper()

	values := []T{}
	for {
		value, done, err := iterable.Next(context.Background())
		if err != nil {
			return values, err
		}
		if done {
			return values, nil
		}
		values = append(values, value)
	}
}

func assertEqual[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func assertInRange(t *testing.T, got float64, start float64, end float64) {
	t.Helper()
	if got < start || got > end {
		t.Fatalf("expected %.2f to be between %.2f and %.2f", got, start, end)
	}
}

func elapsedMS(start time.Time) float64 {
	return float64(time.Since(start)) / float64(time.Millisecond)
}

func TestPMapMain(t *testing.T) {
	start := time.Now()
	result, err := PMap[delayedInput, int](sharedInput(), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, []int{10, 20, 30})
	assertInRange(t, elapsedMS(start), 290, 430)
}

func TestPMapConcurrencyOne(t *testing.T) {
	start := time.Now()
	result, err := PMap[delayedInput, int](sharedInput(), mapper, Options{Concurrency: Int(1)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, []int{10, 20, 30})
	assertInRange(t, elapsedMS(start), 590, 800)
}

func TestPMapConcurrencyFour(t *testing.T) {
	concurrency := 4
	var running atomic.Int32
	input := make([]int, 100)
	mapper := func(_ int, _ int) (any, error) {
		current := running.Add(1)
		if current > int32(concurrency) {
			t.Fatalf("running=%d exceeded concurrency=%d", current, concurrency)
		}
		time.Sleep(time.Duration(rand.IntN(171)+30) * time.Millisecond)
		running.Add(-1)
		return nil, nil
	}

	if _, err := PMap[int, any](input, mapper, Options{Concurrency: Int(concurrency)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPMapHandlesEmptyIterable(t *testing.T) {
	result, err := PMap[delayedInput, int]([]any{}, mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{})
}

func TestPMapConcurrencyTwoRandomTimeSequence(t *testing.T) {
	input := make([]int, 10)
	for index := range input {
		input[index] = rand.IntN(101)
	}

	mapper := func(value int, _ int) (any, error) {
		time.Sleep(time.Duration(value) * time.Millisecond)
		return value, nil
	}

	result, err := PMap[int, int](input, mapper, Options{Concurrency: Int(2)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, input)
}

func TestPMapConcurrencyTwoProblematicSequence(t *testing.T) {
	input := []int{100, 200, 10, 36, 13, 45}
	mapper := func(value int, _ int) (any, error) {
		time.Sleep(time.Duration(value) * time.Millisecond)
		return value, nil
	}

	result, err := PMap[int, int](input, mapper, Options{Concurrency: Int(2)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, input)
}

func TestPMapConcurrencyTwoOutOfOrderSequence(t *testing.T) {
	input := []int{200, 100, 50}
	mapper := func(value int, _ int) (any, error) {
		time.Sleep(time.Duration(value) * time.Millisecond)
		return value, nil
	}

	result, err := PMap[int, int](input, mapper, Options{Concurrency: Int(2)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, input)
}

func TestPMapValidateConcurrency(t *testing.T) {
	_, err := PMap[int, int]([]int{}, func(int, int) (any, error) { return 0, nil }, Options{Concurrency: Int(0)})
	var typeErr *TypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("expected TypeError, got %v", err)
	}

	if _, err := PMap[int, int]([]int{}, func(int, int) (any, error) { return 0, nil }, Options{Concurrency: Int(1)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := PMap[int, int]([]int{}, func(int, int) (any, error) { return 0, nil }, Options{Concurrency: Int(10)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := PMap[int, int]([]int{}, func(int, int) (any, error) { return 0, nil }, Options{Concurrency: Int(Infinity)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPMapImmediateRejectWhenStopOnErrorTrue(t *testing.T) {
	if _, err := PMap[delayedInput, int](errorInput1(), mapper, Options{Concurrency: Int(1)}); err == nil || err.Error() != "foo" {
		t.Fatalf("expected foo error, got %v", err)
	}

	if _, err := PMap[delayedInput, int](errorInput2(), mapper, Options{Concurrency: Int(1)}); err == nil || err.Error() != "bar" {
		t.Fatalf("expected bar error, got %v", err)
	}
}

func TestPMapAggregateErrorsWhenStopOnErrorFalse(t *testing.T) {
	if _, err := PMap[delayedInput, int](sharedInput(), mapper, Options{Concurrency: Int(1), StopOnError: Bool(false)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := PMap[delayedInput, int](errorInput1(), mapper, Options{Concurrency: Int(1), StopOnError: Bool(false)})
	var aggregateErr *AggregateError
	if !errors.As(err, &aggregateErr) {
		t.Fatalf("expected AggregateError, got %v", err)
	}

	_, err = PMap[delayedInput, int](errorInput2(), mapper, Options{Concurrency: Int(1), StopOnError: Bool(false)})
	if !errors.As(err, &aggregateErr) {
		t.Fatalf("expected AggregateError, got %v", err)
	}
}

func TestPMapSkip(t *testing.T) {
	input := []any{1, PMapSkip, 2}
	mapper := func(value any, _ int) (any, error) {
		if value == PMapSkip {
			return PMapSkip, nil
		}
		return value, nil
	}

	result, err := PMap[any, int](input, mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{1, 2})
}

func TestPMapMultipleSkips(t *testing.T) {
	input := []any{1, PMapSkip, 2, PMapSkip, 3, PMapSkip, PMapSkip, 4}
	mapper := func(value any, _ int) (any, error) {
		if value == PMapSkip {
			return PMapSkip, nil
		}
		return value, nil
	}

	result, err := PMap[any, int](input, mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{1, 2, 3, 4})
}

func TestPMapAllSkips(t *testing.T) {
	input := []any{PMapSkip, PMapSkip, PMapSkip, PMapSkip}
	mapper := func(value any, _ int) (any, error) {
		if value == PMapSkip {
			return PMapSkip, nil
		}
		return value, nil
	}

	result, err := PMap[any, int](input, mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{})
}

func TestPMapAllMappersRunWithInfiniteConcurrencyAfterError(t *testing.T) {
	input := []any{
		1,
		func() (int, error) {
			time.Sleep(300 * time.Millisecond)
			return 2, nil
		},
		3,
	}

	var mu sync.Mutex
	mappedValues := []int{}
	_, err := PMap[any, int](input, func(value any, _ int) (any, error) {
		switch resolved := value.(type) {
		case func() (int, error):
			var err error
			value, err = resolved()
			if err != nil {
				return nil, err
			}
		}

		mu.Lock()
		mappedValues = append(mappedValues, value.(int))
		mu.Unlock()
		if value.(int) == 1 {
			time.Sleep(100 * time.Millisecond)
			return nil, errors.New("Oops!")
		}
		return value.(int), nil
	})
	if err == nil || err.Error() != "Oops!" {
		t.Fatalf("expected Oops! error, got %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assertEqual(t, mappedValues, []int{1, 3, 2})
}

func TestPMapAsyncIteratorMain(t *testing.T) {
	start := time.Now()
	result, err := PMap[delayedInput, int](newAsyncTestData(sharedInput()), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, []int{10, 20, 30})
	assertInRange(t, elapsedMS(start), 290, 450)
}

func TestPMapAsyncIteratorConcurrencyOne(t *testing.T) {
	start := time.Now()
	result, err := PMap[delayedInput, int](newAsyncTestData(sharedInput()), mapper, Options{Concurrency: Int(1)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, []int{10, 20, 30})
	assertInRange(t, elapsedMS(start), 590, 800)
}

func TestPMapAsyncIteratorConcurrencyFour(t *testing.T) {
	concurrency := 4
	var running atomic.Int32
	input := make([]any, 100)
	for index := range input {
		input[index] = 0
	}

	if _, err := PMap[int, any](newAsyncTestData(input), func(_ int, _ int) (any, error) {
		current := running.Add(1)
		if current > int32(concurrency) {
			t.Fatalf("running=%d exceeded concurrency=%d", current, concurrency)
		}
		time.Sleep(time.Duration(rand.IntN(171)+30) * time.Millisecond)
		running.Add(-1)
		return nil, nil
	}, Options{Concurrency: Int(concurrency)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPMapAsyncIteratorEmpty(t *testing.T) {
	result, err := PMap[delayedInput, int](newAsyncTestData([]any{}), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{})
}

func TestPMapAsyncIteratorConcurrencyTwoRandomTimeSequence(t *testing.T) {
	input := make([]any, 10)
	expected := make([]int, 10)
	for i := range input {
		v := rand.IntN(101)
		input[i] = v
		expected[i] = v
	}

	mapper := func(value int, _ int) (any, error) {
		time.Sleep(time.Duration(value) * time.Millisecond)
		return value, nil
	}

	result, err := PMap[int, int](newAsyncTestData(input), mapper, Options{Concurrency: Int(2)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, expected)
}

func TestPMapAsyncIteratorConcurrencyTwoProblematicSequence(t *testing.T) {
	input := []any{100, 200, 10, 36, 13, 45}
	mapper := func(value int, _ int) (any, error) {
		time.Sleep(time.Duration(value) * time.Millisecond)
		return value, nil
	}

	result, err := PMap[int, int](newAsyncTestData(input), mapper, Options{Concurrency: Int(2)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, []int{100, 200, 10, 36, 13, 45})
}

func TestPMapAsyncIteratorConcurrencyTwoOutOfOrderSequence(t *testing.T) {
	input := []any{200, 100, 50}
	mapper := func(value int, _ int) (any, error) {
		time.Sleep(time.Duration(value) * time.Millisecond)
		return value, nil
	}

	result, err := PMap[int, int](newAsyncTestData(input), mapper, Options{Concurrency: Int(2)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, result, []int{200, 100, 50})
}

func TestPMapAsyncIteratorValidateConcurrency(t *testing.T) {
	_, err := PMap[int, int](newAsyncTestData([]any{}), func(int, int) (any, error) { return 0, nil }, Options{Concurrency: Int(0)})
	var typeErr *TypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("expected TypeError, got %v", err)
	}
}

func TestPMapAsyncIteratorImmediateRejectWhenStopOnErrorTrue(t *testing.T) {
	if _, err := PMap[delayedInput, int](newAsyncTestData(errorInput1()), mapper, Options{Concurrency: Int(1)}); err == nil || err.Error() != "foo" {
		t.Fatalf("expected foo error, got %v", err)
	}

	if _, err := PMap[delayedInput, int](newAsyncTestData(errorInput2()), mapper, Options{Concurrency: Int(1)}); err == nil || err.Error() != "bar" {
		t.Fatalf("expected bar error, got %v", err)
	}
}

func TestPMapAsyncIteratorAggregateErrorsWhenStopOnErrorFalse(t *testing.T) {
	_, err := PMap[delayedInput, int](newAsyncTestData(errorInput1()), mapper, Options{Concurrency: Int(1), StopOnError: Bool(false)})
	var aggregateErr *AggregateError
	if !errors.As(err, &aggregateErr) {
		t.Fatalf("expected AggregateError, got %v", err)
	}

	_, err = PMap[delayedInput, int](newAsyncTestData(errorInput2()), mapper, Options{Concurrency: Int(1), StopOnError: Bool(false)})
	if !errors.As(err, &aggregateErr) {
		t.Fatalf("expected AggregateError, got %v", err)
	}
}

func TestPMapAsyncIteratorPMapSkip(t *testing.T) {
	input := []any{1, PMapSkip, 2}
	mapper := func(value any, _ int) (any, error) {
		if value == PMapSkip {
			return PMapSkip, nil
		}
		return value, nil
	}

	result, err := PMap[any, int](newAsyncTestData(input), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{1, 2})
}

func TestPMapAsyncIteratorMultipleSkips(t *testing.T) {
	input := []any{1, PMapSkip, 2, PMapSkip, 3, PMapSkip, PMapSkip, 4}
	mapper := func(value any, _ int) (any, error) {
		if value == PMapSkip {
			return PMapSkip, nil
		}
		return value, nil
	}

	result, err := PMap[any, int](newAsyncTestData(input), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{1, 2, 3, 4})
}

func TestPMapAsyncIteratorAllSkips(t *testing.T) {
	input := []any{PMapSkip, PMapSkip, PMapSkip, PMapSkip}
	mapper := func(value any, _ int) (any, error) {
		if value == PMapSkip {
			return PMapSkip, nil
		}
		return value, nil
	}

	result, err := PMap[any, int](newAsyncTestData(input), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, result, []int{})
}

func TestPMapAsyncIteratorAllMappersRunWithInfiniteConcurrencyAfterError(t *testing.T) {
	input := []any{
		1,
		func() (int, error) {
			time.Sleep(300 * time.Millisecond)
			return 2, nil
		},
		3,
	}

	var mu sync.Mutex
	mappedValues := []int{}
	_, err := PMap[any, int](newAsyncTestData(input), func(value any, _ int) (any, error) {
		switch resolved := value.(type) {
		case func() (int, error):
			var err error
			value, err = resolved()
			if err != nil {
				return nil, err
			}
		}

		mu.Lock()
		mappedValues = append(mappedValues, value.(int))
		mu.Unlock()
		if value.(int) == 1 {
			time.Sleep(100 * time.Millisecond)
			return nil, fmt.Errorf("Oops! %d", value.(int))
		}
		return value.(int), nil
	})
	if err == nil || err.Error() != "Oops! 1" {
		t.Fatalf("expected Oops! 1 error, got %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assertEqual(t, mappedValues, []int{1, 3, 2})
}

func TestPMapCatchesIteratorExceptionFirstItem(t *testing.T) {
	input := newThrowingIterator(100, 0)
	mappedValues := []int{}
	_, err := PMap[int, int](input, func(value int, _ int) (any, error) {
		mappedValues = append(mappedValues, value)
		time.Sleep(100 * time.Millisecond)
		return value, nil
	}, Options{Concurrency: Int(1)})
	if err == nil || err.Error() != "throwing on index 0" {
		t.Fatalf("expected iterator error, got %v", err)
	}

	if input.index != 1 {
		t.Fatalf("expected index=1, got %d", input.index)
	}

	time.Sleep(300 * time.Millisecond)
	assertEqual(t, mappedValues, []int{})
}

func TestPMapCatchesIteratorExceptionSecondItem(t *testing.T) {
	input := newThrowingIterator(100, 1)
	mappedValues := []int{}
	_, err := PMap[int, int](input, func(value int, _ int) (any, error) {
		mappedValues = append(mappedValues, value)
		time.Sleep(100 * time.Millisecond)
		return value, nil
	}, Options{Concurrency: Int(1)})
	if err == nil || err.Error() != "throwing on index 1" {
		t.Fatalf("expected iterator error, got %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	if input.index != 2 {
		t.Fatalf("expected index=2, got %d", input.index)
	}
	assertEqual(t, mappedValues, []int{0})
}

func TestPMapCatchesIteratorExceptionSecondItemAfterMapperThrow(t *testing.T) {
	input := newThrowingIterator(100, 1)
	mappedValues := []int{}
	_, err := PMap[int, int](input, func(value int, _ int) (any, error) {
		mappedValues = append(mappedValues, value)
		time.Sleep(100 * time.Millisecond)
		return nil, errors.New("mapper threw error")
	}, Options{Concurrency: Int(1), StopOnError: Bool(false)})
	if err == nil || err.Error() != "throwing on index 1" {
		t.Fatalf("expected iterator error, got %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	if input.index != 2 {
		t.Fatalf("expected index=2, got %d", input.index)
	}
	assertEqual(t, mappedValues, []int{0})
}

func TestPMapAsyncIteratorCorrectExceptionAfterStopOnError(t *testing.T) {
	input := []any{
		1,
		func() (int, error) {
			time.Sleep(200 * time.Millisecond)
			return 2, nil
		},
		func() (int, error) {
			time.Sleep(300 * time.Millisecond)
			return 3, nil
		},
	}

	var mu sync.Mutex
	mappedValues := []int{}
	taskDone := make(chan error, 1)
	go func() {
		_, err := PMap[any, int](newAsyncTestData(input), func(value any, _ int) (any, error) {
			switch resolved := value.(type) {
			case func() (int, error):
				var err error
				value, err = resolved()
				if err != nil {
					return nil, err
				}
			}

			mu.Lock()
			mappedValues = append(mappedValues, value.(int))
			mu.Unlock()
			time.Sleep(100 * time.Millisecond)
			return nil, fmt.Errorf("Oops! %d", value.(int))
		})
		taskDone <- err
	}()

	time.Sleep(500 * time.Millisecond)
	err := <-taskDone
	if err == nil || err.Error() != "Oops! 1" {
		t.Fatalf("expected Oops! 1 error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	assertEqual(t, mappedValues, []int{1, 2, 3})
}

func TestPMapIncorrectInputType(t *testing.T) {
	mapperCalled := false
	_, err := PMap[int, int](123456, func(_ int, _ int) (any, error) {
		mapperCalled = true
		time.Sleep(100 * time.Millisecond)
		return 0, nil
	})
	if err == nil || err.Error() != "Expected `input` to be either an `Iterable` or `AsyncIterable`, got (int)" {
		t.Fatalf("expected input type error, got %v", err)
	}
	if mapperCalled {
		t.Fatalf("mapper should not have been called")
	}
}

func TestPMapNoUnhandledMapperThrowsInfiniteConcurrency(t *testing.T) {
	input := []int{1, 2, 3}
	var mu sync.Mutex
	mappedValues := []int{}
	_, err := PMap[int, int](input, func(value int, _ int) (any, error) {
		mu.Lock()
		mappedValues = append(mappedValues, value)
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		return nil, fmt.Errorf("Oops! %d", value)
	})
	if err == nil || !strings.HasPrefix(err.Error(), "Oops!") {
		t.Fatalf("expected Oops! error, got %v", err)
	}
	mu.Lock()
	got := append([]int(nil), mappedValues...)
	mu.Unlock()
	sort.Ints(got)
	assertEqual(t, got, []int{1, 2, 3})
}

func TestPMapNoUnhandledMapperThrowsConcurrencyOne(t *testing.T) {
	input := []int{1, 2, 3}
	var mu sync.Mutex
	mappedValues := []int{}
	_, err := PMap[int, int](input, func(value int, _ int) (any, error) {
		mu.Lock()
		mappedValues = append(mappedValues, value)
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		return nil, fmt.Errorf("Oops! %d", value)
	}, Options{Concurrency: Int(1)})
	if err == nil || err.Error() != "Oops! 1" {
		t.Fatalf("expected Oops! 1 error, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	assertEqual(t, mappedValues, []int{1})
}

func TestPMapAbortByContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	abortErr := errors.New("AbortError")
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel(abortErr)
	}()

	input := []any{
		AwaitableFunc[int](func(context.Context) (int, error) {
			time.Sleep(time.Second)
			return 1, nil
		}),
		AwaitableFunc[int](func(context.Context) (int, error) {
			time.Sleep(time.Second)
			return 2, nil
		}),
	}

	_, err := PMap[int, int](input, func(value int, _ int) (any, error) {
		return value, nil
	}, Options{Context: ctx})
	if !errors.Is(err, abortErr) {
		t.Fatalf("expected abort error, got %v", err)
	}
}

func TestPMapAlreadyAbortedContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	abortErr := errors.New("AbortError")
	cancel(abortErr)

	input := []any{
		AwaitableFunc[int](func(context.Context) (int, error) {
			time.Sleep(time.Second)
			return 1, nil
		}),
	}

	_, err := PMap[int, int](input, func(value int, _ int) (any, error) {
		return value, nil
	}, Options{Context: ctx})
	if !errors.Is(err, abortErr) {
		t.Fatalf("expected abort error, got %v", err)
	}
}

func TestPMapInvalidMapper(t *testing.T) {
	t.Skip("Go's static type system prevents passing a non-function mapper to PMap")
}

func TestPMapIterableMain(t *testing.T) {
	iterable, err := PMapIterable[delayedInput, int](sharedInput(), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, err := collectAsyncIterable(t, iterable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, values, []int{10, 20, 30})
}

func TestPMapIterableIndexInMapper(t *testing.T) {
	iterable, err := PMapIterable[delayedInput, indexedValue](sharedInput(), mapperWithIndex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, err := collectAsyncIterable(t, iterable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, values, []indexedValue{
		{Value: 10, Index: 0},
		{Value: 20, Index: 1},
		{Value: 30, Index: 2},
	})

	iterable, err = PMapIterable[delayedInput, indexedValue](longerSharedInput(), mapperWithIndex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, err = collectAsyncIterable(t, iterable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, values, []indexedValue{
		{Value: 10, Index: 0},
		{Value: 20, Index: 1},
		{Value: 30, Index: 2},
		{Value: 40, Index: 3},
		{Value: 50, Index: 4},
	})
}

func TestPMapIterableEmpty(t *testing.T) {
	iterable, err := PMapIterable[delayedInput, int]([]any{}, mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, err := collectAsyncIterable(t, iterable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, values, []int{})
}

func TestPMapIterableIterableThatThrows(t *testing.T) {
	isFirstNextCall := true
	iterable, err := PMapIterable[delayedInput, int](IteratorFunc(func(context.Context) (any, bool, error) {
		if !isFirstNextCall {
			return nil, true, nil
		}
		isFirstNextCall = false
		return nil, false, errors.New("foo")
	}), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, err = iterable.Next(context.Background())
	if err == nil || err.Error() != "foo" {
		t.Fatalf("expected foo error, got %v", err)
	}
}

func TestPMapIterableMapperThatThrows(t *testing.T) {
	iterable, err := PMapIterable[delayedInput, int](sharedInput(), func(delayedInput, int) (any, error) {
		return nil, errors.New("foo")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = collectAsyncIterable(t, iterable)
	if err == nil || err.Error() != "foo" {
		t.Fatalf("expected foo error, got %v", err)
	}
}

func TestPMapIterableStopOnError(t *testing.T) {
	iterable, err := PMapIterable[delayedInput, int](errorInput3(), mapper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := []int{}
	for {
		value, done, err := iterable.Next(context.Background())
		if err != nil {
			if err.Error() != "bar" {
				t.Fatalf("expected bar error, got %v", err)
			}
			break
		}
		if done {
			break
		}
		output = append(output, value)
	}

	assertEqual(t, output, []int{20})
}

func TestPMapIterableConcurrencyOne(t *testing.T) {
	start := time.Now()
	iterable, err := PMapIterable[delayedInput, int](sharedInput(), mapper, IterableOptions{
		Concurrency:  Int(1),
		Backpressure: Int(Infinity),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, err := collectAsyncIterable(t, iterable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, values, []int{10, 20, 30})
	assertInRange(t, elapsedMS(start), 590, 800)
}

func TestPMapIterableConcurrencyTwo(t *testing.T) {
	var mu sync.Mutex
	times := map[int]float64{}
	start := time.Now()
	iterable, err := PMapIterable[delayedInput, int](longerSharedInput(), func(input delayedInput, index int) (any, error) {
		switch value := input.value.(type) {
		case int:
			mu.Lock()
			times[value] = elapsedMS(start)
			mu.Unlock()
		}
		return mapper(input, index)
	}, IterableOptions{
		Concurrency:  Int(2),
		Backpressure: Int(Infinity),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, err := collectAsyncIterable(t, iterable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, values, []int{10, 20, 30, 40, 50})
	mu.Lock()
	defer mu.Unlock()
	assertInRange(t, times[10], 0, 50)
	assertInRange(t, times[20], 0, 50)
	assertInRange(t, times[30], 190, 280)
	assertInRange(t, times[40], 280, 380)
	assertInRange(t, times[50], 280, 380)
}

func TestPMapIterableBackpressure(t *testing.T) {
	var currentValue atomic.Int64
	iterable, err := PMapIterable[delayedInput, int](longerSharedInput(), func(input delayedInput, index int) (any, error) {
		value, err := mapper(input, index)
		if err != nil {
			return nil, err
		}
		currentValue.Store(int64(value.(int)))
		return value, nil
	}, IterableOptions{
		Backpressure: Int(2),
		Concurrency:  Int(2),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value1, done, err := iterable.Next(context.Background())
	if err != nil || done {
		t.Fatalf("unexpected next result: value=%v done=%v err=%v", value1, done, err)
	}
	if value1 != 10 {
		t.Fatalf("expected first value 10, got %d", value1)
	}

	time.Sleep(600 * time.Millisecond)
	if currentValue.Load() != 30 {
		t.Fatalf("expected currentValue=30, got %d", currentValue.Load())
	}

	value2, done, err := iterable.Next(context.Background())
	if err != nil || done {
		t.Fatalf("unexpected next result: value=%v done=%v err=%v", value2, done, err)
	}
	if value2 != 20 {
		t.Fatalf("expected second value 20, got %d", value2)
	}

	time.Sleep(100 * time.Millisecond)
	if currentValue.Load() != 40 {
		t.Fatalf("expected currentValue=40, got %d", currentValue.Load())
	}
}

func TestPMapIterableAsyncInputBackpressureGreaterThanConcurrency(t *testing.T) {
	source := newAsyncTestData([]any{1, 2, 3})
	var mu sync.Mutex
	log := []int{}

	iterable, err := PMapIterable[int, int](source, func(n int, _ int) (any, error) {
		mu.Lock()
		log = append(log, n)
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		log = append(log, n)
		mu.Unlock()
		return n, nil
	}, IterableOptions{
		Concurrency:  Int(1),
		Backpressure: Int(2),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := collectAsyncIterable(t, iterable); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	assertEqual(t, log, []int{1, 1, 2, 2, 3, 3})
}

func TestPMapIterableSkip(t *testing.T) {
	iterable, err := PMapIterable[any, int]([]any{1, PMapSkip, 2}, func(value any, _ int) (any, error) {
		if value == PMapSkip {
			return PMapSkip, nil
		}
		return value, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, err := collectAsyncIterable(t, iterable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, values, []int{1, 2})
}

func TestPMapMultipleSkipsAlgorithmicComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping algorithmic complexity test in short mode")
	}

	generateSkipData := func(length int) []any {
		data := make([]any, length)
		for index := range data {
			data[index] = PMapSkip
		}
		return data
	}

	testData := [][]any{
		generateSkipData(1000),
		generateSkipData(5000),
		generateSkipData(25000),
	}
	testDurationsMS := make([]float64, 0, len(testData))

	for _, data := range testData {
		start := time.Now()
		_, err := PMap[any, int](data, func(value any, _ int) (any, error) {
			if value == PMapSkip {
				return PMapSkip, nil
			}
			return value, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		testDurationsMS = append(testDurationsMS, elapsedMS(start))
	}

	for index := 0; index < len(testDurationsMS)-1; index++ {
		smaller := testDurationsMS[index]
		larger := testDurationsMS[index+1]
		assertInRange(t, larger, 1.05*smaller, 25*smaller)
	}
}
