package pmap

// Go port of p-map.
// JS source: https://github.com/sindresorhus/p-map/blob/main/index.js

import (
	"container/list"
	"context"
	"fmt"
	"reflect"
	"sync"
)

// Infinity mirrors JavaScript's Number.POSITIVE_INFINITY for concurrency and
// backpressure options.
const Infinity = -1

// PMapSkip mirrors the upstream sentinel used to exclude a mapper result from
// the final output.
var PMapSkip = &skipToken{}

type skipToken struct{}

// Int returns a pointer to the provided int.
func Int(v int) *int { return &v }

// Bool returns a pointer to the provided bool.
func Bool(v bool) *bool { return &v }

// TypeError mirrors the upstream runtime validation failures.
type TypeError struct {
	message string
}

func (e *TypeError) Error() string { return e.message }

func newTypeError(message string) *TypeError {
	return &TypeError{message: message}
}

// AggregateError mirrors the upstream error returned when stopOnError is false.
type AggregateError struct {
	Errors []error
}

func (e *AggregateError) Error() string { return "" }

func (e *AggregateError) Unwrap() []error { return e.Errors }

// Options configures PMap.
type Options struct {
	Concurrency *int
	StopOnError *bool
	Context     context.Context
}

// IterableOptions configures PMapIterable.
type IterableOptions struct {
	Concurrency  *int
	Backpressure *int
}

// Mapper mirrors the JS mapper signature while allowing PMapSkip to be
// returned directly.
type Mapper[T any] func(element T, index int) (any, error)

// Iterator is the Go equivalent of the sync/async iterator consumed by p-map.
type Iterator interface {
	Next(ctx context.Context) (value any, done bool, err error)
}

// IteratorFunc adapts a function into an Iterator.
type IteratorFunc func(ctx context.Context) (value any, done bool, err error)

func (f IteratorFunc) Next(ctx context.Context) (value any, done bool, err error) {
	return f(ctx)
}

// AsyncIterable is the Go equivalent of the async iterable returned by
// pMapIterable.
type AsyncIterable[T any] interface {
	Next(ctx context.Context) (value T, done bool, err error)
}

// Awaitable mirrors an input item that must be awaited before invoking the
// mapper.
type Awaitable[T any] interface {
	Await(ctx context.Context) (T, error)
}

// AwaitableFunc adapts a function into an Awaitable.
type AwaitableFunc[T any] func(ctx context.Context) (T, error)

func (f AwaitableFunc[T]) Await(ctx context.Context) (T, error) {
	return f(ctx)
}

type resolvedValue[T any] struct {
	value T
}

func (r resolvedValue[T]) Await(context.Context) (T, error) {
	return r.value, nil
}

// Resolved wraps a direct value so it can be mixed with awaitable input items.
func Resolved[T any](value T) Awaitable[T] {
	return resolvedValue[T]{value: value}
}

// PMap faithfully ports the upstream pMap behavior.
func PMap[T, R any](input any, mapper Mapper[T], opts ...Options) ([]R, error) {
	iterator, err := toIterator(input)
	if err != nil {
		return nil, err
	}

	if mapper == nil {
		return nil, newTypeError("Mapper function is required")
	}

	options := resolveOptions(opts)
	if err := validateConcurrency(options.concurrency); err != nil {
		return nil, err
	}

	ctx := options.context
	if cause := context.Cause(ctx); cause != nil {
		return nil, cause
	}

	type outcome struct {
		values []R
		err    error
	}

	var state struct {
		mu             sync.Mutex
		result         []any
		errors         []error
		skippedIndexes map[int]struct{}
		isRejected     bool
		isResolved     bool
		isIterableDone bool
		pendingNext    int
		startedCount   int
		completedCount int
	}

	orderedIterator := newOrderedIterator(iterator)
	outcomeCh := make(chan outcome, 1)
	var settleOnce sync.Once

	reject := func(reason error) {
		settleOnce.Do(func() {
			state.mu.Lock()
			state.isRejected = true
			state.isResolved = true
			state.mu.Unlock()
			outcomeCh <- outcome{err: reason}
		})
	}

	resolve := func(values []R) {
		settleOnce.Do(func() {
			state.mu.Lock()
			state.isResolved = true
			state.mu.Unlock()
			outcomeCh <- outcome{values: values}
		})
	}

	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			reject(context.Cause(ctx))
		}()
	}

	var next func() error
	var nextCounted func() error

	spawnWorker := func(index int, rawItem any) {
		go func() {
			handleWorkerError := func(workerErr error) {
				if options.stopOnError {
					reject(workerErr)
					return
				}

				state.mu.Lock()
				state.errors = append(state.errors, workerErr)
				state.completedCount++
				state.pendingNext++
				state.mu.Unlock()

				if err := nextCounted(); err != nil {
					reject(err)
				}
			}

			element, err := resolveInputValue[T](ctx, rawItem)
			if err != nil {
				handleWorkerError(err)
				return
			}

			value, err := mapper(element, index)
			if err != nil {
				handleWorkerError(err)
				return
			}

			if value != PMapSkip {
				if _, ok := castValue[R](value); !ok {
					handleWorkerError(newTypeError(
						fmt.Sprintf("Expected mapper result to be %s or PMapSkip, got (%T)", typeName[R](), value),
					))
					return
				}
			}

			state.mu.Lock()
			if value == PMapSkip {
				if state.skippedIndexes == nil {
					state.skippedIndexes = make(map[int]struct{})
				}
				state.skippedIndexes[index] = struct{}{}
			}
			ensureResultLen(&state.result, index+1)
			state.result[index] = value
			state.completedCount++
			state.pendingNext++
			state.mu.Unlock()

			if err := nextCounted(); err != nil {
				if options.stopOnError {
					reject(err)
					return
				}

				state.mu.Lock()
				state.errors = append(state.errors, err)
				state.pendingNext++
				state.mu.Unlock()

				if err := nextCounted(); err != nil {
					reject(err)
				}
			}
		}()
	}

	consumeNext := func(precounted bool) error {
		state.mu.Lock()
		if state.isResolved {
			if precounted {
				state.pendingNext--
			}
			state.mu.Unlock()
			return nil
		}
		if !precounted {
			state.pendingNext++
		}
		state.mu.Unlock()

		response, err := orderedIterator.NextOrdered(ctx)

		state.mu.Lock()
		state.pendingNext--
		if err != nil {
			state.mu.Unlock()
			return err
		}

		if response.done {
			state.isIterableDone = true

			if state.completedCount == state.startedCount && state.pendingNext == 0 && !state.isResolved {
				if !options.stopOnError && len(state.errors) > 0 {
					errorsCopy := append([]error(nil), state.errors...)
					state.mu.Unlock()
					reject(&AggregateError{Errors: errorsCopy})
					return nil
				}

				state.isResolved = true

				resultsCopy := append([]any(nil), state.result...)
				skippedCopy := make(map[int]struct{}, len(state.skippedIndexes))
				for skippedIndex := range state.skippedIndexes {
					skippedCopy[skippedIndex] = struct{}{}
				}
				state.mu.Unlock()

				values, castErr := compactResults[R](resultsCopy, skippedCopy)
				if castErr != nil {
					reject(castErr)
					return nil
				}

				resolve(values)
			} else {
				state.mu.Unlock()
			}

			return nil
		}

		state.startedCount++
		state.mu.Unlock()

		spawnWorker(response.index, response.value)
		return nil
	}

	next = func() error {
		return consumeNext(false)
	}

	nextCounted = func() error {
		return consumeNext(true)
	}

	go func() {
		if options.concurrency == Infinity {
			for {
				if err := next(); err != nil {
					reject(err)
					return
				}

				state.mu.Lock()
				stop := state.isIterableDone || state.isRejected
				state.mu.Unlock()
				if stop {
					return
				}
			}
		}

		for index := 0; index < options.concurrency; index++ {
			if err := next(); err != nil {
				reject(err)
				return
			}

			state.mu.Lock()
			stop := state.isIterableDone || state.isRejected
			state.mu.Unlock()
			if stop {
				return
			}
		}
	}()

	result := <-outcomeCh
	return result.values, result.err
}

// PMapIterable faithfully ports the upstream pMapIterable behavior.
func PMapIterable[T, R any](input any, mapper Mapper[T], opts ...IterableOptions) (AsyncIterable[R], error) {
	iterator, err := toIterator(input)
	if err != nil {
		return nil, err
	}

	if mapper == nil {
		return nil, newTypeError("Mapper function is required")
	}

	options := resolveIterableOptions(opts)
	if err := validateConcurrency(options.concurrency); err != nil {
		return nil, err
	}

	if err := validateBackpressure(options.concurrency, options.backpressure); err != nil {
		return nil, err
	}

	return &mappedIterable[T, R]{
		iterator: newOrderedIterator(iterator),
		mapper:   mapper,
		options:  options,
	}, nil
}

type resolvedOptions struct {
	concurrency int
	stopOnError bool
	context     context.Context
}

func resolveOptions(opts []Options) resolvedOptions {
	result := resolvedOptions{
		concurrency: Infinity,
		stopOnError: true,
		context:     context.Background(),
	}

	if len(opts) == 0 {
		return result
	}

	opt := opts[0]
	if opt.Concurrency != nil {
		result.concurrency = *opt.Concurrency
	}
	if opt.StopOnError != nil {
		result.stopOnError = *opt.StopOnError
	}
	if opt.Context != nil {
		result.context = opt.Context
	}

	return result
}

type resolvedIterableOptions struct {
	concurrency  int
	backpressure int
}

func resolveIterableOptions(opts []IterableOptions) resolvedIterableOptions {
	result := resolvedIterableOptions{
		concurrency:  Infinity,
		backpressure: Infinity,
	}

	if len(opts) == 0 {
		return result
	}

	opt := opts[0]
	if opt.Concurrency != nil {
		result.concurrency = *opt.Concurrency
	}
	if opt.Backpressure != nil {
		result.backpressure = *opt.Backpressure
	} else {
		result.backpressure = result.concurrency
	}

	return result
}

func validateConcurrency(concurrency int) error {
	if concurrency == Infinity {
		return nil
	}
	if concurrency >= 1 {
		return nil
	}

	return newTypeError(
		fmt.Sprintf("Expected `concurrency` to be an integer from 1 and up or `Infinity`, got `%d` (int)", concurrency),
	)
}

func validateBackpressure(concurrency, backpressure int) error {
	if backpressure == Infinity {
		return nil
	}
	if backpressure < 1 {
		return newTypeError(
			fmt.Sprintf("Expected `backpressure` to be an integer from `concurrency` (%s) and up or `Infinity`, got `%d` (int)", formatLimit(concurrency), backpressure),
		)
	}

	if concurrency == Infinity {
		return newTypeError(
			fmt.Sprintf("Expected `backpressure` to be an integer from `concurrency` (%s) and up or `Infinity`, got `%d` (int)", formatLimit(concurrency), backpressure),
		)
	}

	if backpressure < concurrency {
		return newTypeError(
			fmt.Sprintf("Expected `backpressure` to be an integer from `concurrency` (%d) and up or `Infinity`, got `%d` (int)", concurrency, backpressure),
		)
	}

	return nil
}

func formatLimit(limit int) string {
	if limit == Infinity {
		return "Infinity"
	}
	return fmt.Sprintf("%d", limit)
}

func toIterator(input any) (Iterator, error) {
	if input == nil {
		return nil, newTypeError("Expected `input` to be either an `Iterable` or `AsyncIterable`, got (nil)")
	}

	if iterator, ok := input.(Iterator); ok {
		return iterator, nil
	}

	value := reflect.ValueOf(input)
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		return &sliceIterator{value: value}, nil
	default:
		return nil, newTypeError(
			fmt.Sprintf("Expected `input` to be either an `Iterable` or `AsyncIterable`, got (%s)", value.Kind()),
		)
	}
}

type sliceIterator struct {
	value reflect.Value
	index int
}

func (s *sliceIterator) Next(context.Context) (value any, done bool, err error) {
	if s.index >= s.value.Len() {
		return nil, true, nil
	}

	item := s.value.Index(s.index).Interface()
	s.index++
	return item, false, nil
}

func resolveInputValue[T any](ctx context.Context, value any) (T, error) {
	if awaitable, ok := value.(Awaitable[T]); ok {
		return awaitable.Await(ctx)
	}

	if resolved, ok := castValue[T](value); ok {
		return resolved, nil
	}

	var zero T
	return zero, newTypeError(
		fmt.Sprintf("Expected input item to resolve to %s, got (%T)", typeName[T](), value),
	)
}

func compactResults[R any](values []any, skipped map[int]struct{}) ([]R, error) {
	result := make([]R, 0, len(values))
	for index, value := range values {
		if _, skippedValue := skipped[index]; skippedValue {
			continue
		}

		resolved, ok := castValue[R](value)
		if !ok {
			return nil, newTypeError(
				fmt.Sprintf("Expected mapper result to be %s or PMapSkip, got (%T)", typeName[R](), value),
			)
		}

		result = append(result, resolved)
	}

	return result, nil
}

func ensureResultLen(values *[]any, size int) {
	for len(*values) < size {
		*values = append(*values, nil)
	}
}

func castValue[T any](value any) (T, bool) {
	var zero T

	if value == nil {
		if allowsNil(reflect.TypeFor[T]()) {
			return zero, true
		}
		return zero, false
	}

	resolved, ok := value.(T)
	return resolved, ok
}

func allowsNil(typ reflect.Type) bool {
	if typ == nil {
		return true
	}

	switch typ.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		return true
	default:
		return false
	}
}

func typeName[T any]() string {
	typ := reflect.TypeFor[T]()
	if typ == nil {
		return "interface{}"
	}
	return typ.String()
}

type mappedIterable[T, R any] struct {
	iterator *orderedIterator
	mapper   Mapper[T]
	options  resolvedIterableOptions

	initOnce sync.Once

	mu                   sync.Mutex
	promises             list.List
	pendingPromisesCount int
	isDone               bool
}

type iterableFuture struct {
	value any
	done  bool
	err   error
	index int
	ready chan struct{}
	elem  *list.Element
}

func (m *mappedIterable[T, R]) Next(ctx context.Context) (value R, done bool, err error) {
	var zero R

	m.initOnce.Do(func() {
		m.trySpawn()
	})

	for {
		m.mu.Lock()
		front := m.promises.Front()
		if front == nil {
			m.mu.Unlock()
			return zero, true, nil
		}
		future := front.Value.(*iterableFuture)
		ready := future.ready
		m.mu.Unlock()

		select {
		case <-ready:
		case <-ctx.Done():
			return zero, false, context.Cause(ctx)
		}

		m.mu.Lock()
		currentFront := m.promises.Front()
		if currentFront != front {
			m.mu.Unlock()
			continue
		}
		if currentFront == front {
			m.promises.Remove(front)
			future.elem = nil
		}
		m.mu.Unlock()

		if future.err != nil {
			return zero, false, future.err
		}

		if future.done {
			return zero, true, nil
		}

		m.trySpawn()

		if future.value == PMapSkip {
			continue
		}

		resolved, ok := castValue[R](future.value)
		if !ok {
			return zero, false, newTypeError(
				fmt.Sprintf("Expected mapper result to be %s or PMapSkip, got (%T)", typeName[R](), future.value),
			)
		}

		return resolved, false, nil
	}
}

func (m *mappedIterable[T, R]) trySpawn() {
	m.mu.Lock()
	if m.isDone || !m.canSpawnLocked() {
		m.mu.Unlock()
		return
	}

	future := &iterableFuture{
		index: -1,
		ready: make(chan struct{}),
	}
	future.elem = m.promises.PushBack(future)
	m.pendingPromisesCount++
	m.mu.Unlock()

	go m.runFuture(future)
}

func (m *mappedIterable[T, R]) canSpawnLocked() bool {
	if m.options.concurrency != Infinity && m.pendingPromisesCount >= m.options.concurrency {
		return false
	}
	if m.options.backpressure != Infinity && m.promises.Len() >= m.options.backpressure {
		return false
	}
	return true
}

func (m *mappedIterable[T, R]) runFuture(future *iterableFuture) {
	response, err := m.nextInput()
	if err != nil {
		m.mu.Lock()
		m.pendingPromisesCount--
		m.isDone = true
		future.index = response.index
		m.reorderFutureLocked(future)
		m.mu.Unlock()
		future.err = err
		close(future.ready)
		return
	}

	future.index = response.index

	if response.done {
		m.mu.Lock()
		m.pendingPromisesCount--
		m.isDone = true
		m.reorderFutureLocked(future)
		m.mu.Unlock()
		future.done = true
		close(future.ready)
		return
	}

	m.trySpawn()

	element, err := resolveInputValue[T](context.Background(), response.value)
	if err != nil {
		m.mu.Lock()
		m.pendingPromisesCount--
		m.isDone = true
		m.reorderFutureLocked(future)
		m.mu.Unlock()
		future.err = err
		close(future.ready)
		return
	}

	returnValue, err := m.mapper(element, response.index)
	if err != nil {
		m.mu.Lock()
		m.pendingPromisesCount--
		m.isDone = true
		m.reorderFutureLocked(future)
		m.mu.Unlock()
		future.err = err
		close(future.ready)
		return
	}

	if returnValue != PMapSkip {
		if _, ok := castValue[R](returnValue); !ok {
			m.mu.Lock()
			m.pendingPromisesCount--
			m.isDone = true
			m.reorderFutureLocked(future)
			m.mu.Unlock()
			future.err = newTypeError(
				fmt.Sprintf("Expected mapper result to be %s or PMapSkip, got (%T)", typeName[R](), returnValue),
			)
			close(future.ready)
			return
		}
	}

	m.mu.Lock()
	m.pendingPromisesCount--
	future.value = returnValue
	m.reorderFutureLocked(future)
	if returnValue == PMapSkip {
		if elem := future.elem; elem != nil && elem != m.promises.Front() {
			m.promises.Remove(elem)
			future.elem = nil
		}
	}
	m.mu.Unlock()

	m.trySpawn()
	close(future.ready)
}

func (m *mappedIterable[T, R]) nextInput() (iteratorResponse, error) {
	return m.iterator.NextOrdered(context.Background())
}

func (m *mappedIterable[T, R]) reorderFutureLocked(future *iterableFuture) {
	elem := future.elem
	if elem == nil {
		return
	}

	for previous := elem.Prev(); previous != nil; previous = elem.Prev() {
		previousFuture := previous.Value.(*iterableFuture)
		if previousFuture.index == -1 || previousFuture.index <= future.index {
			break
		}
		m.promises.MoveBefore(elem, previous)
	}

	for next := elem.Next(); next != nil; next = elem.Next() {
		nextFuture := next.Value.(*iterableFuture)
		if nextFuture.index == -1 {
			m.promises.MoveAfter(elem, next)
			continue
		}
		if nextFuture.index >= future.index {
			break
		}
		m.promises.MoveAfter(elem, next)
	}
}

type orderedIterator struct {
	requests chan iteratorRequest
}

type iteratorRequest struct {
	ctx      context.Context
	response chan iteratorResponse
}

type iteratorResponse struct {
	value any
	done  bool
	err   error
	index int
}

func newOrderedIterator(iterator Iterator) *orderedIterator {
	ordered := &orderedIterator{
		requests: make(chan iteratorRequest),
	}

	go func() {
		index := 0
		for request := range ordered.requests {
			value, done, err := iterator.Next(request.ctx)
			itemIndex := index
			if err == nil && !done {
				index++
			}
			request.response <- iteratorResponse{
				value: value,
				done:  done,
				err:   err,
				index: itemIndex,
			}
		}
	}()

	return ordered
}

func (o *orderedIterator) NextOrdered(ctx context.Context) (iteratorResponse, error) {
	response := make(chan iteratorResponse, 1)
	request := iteratorRequest{
		ctx:      ctx,
		response: response,
	}

	select {
	case o.requests <- request:
	case <-ctx.Done():
		return iteratorResponse{}, context.Cause(ctx)
	}

	select {
	case result := <-response:
		if result.err != nil {
			return iteratorResponse{}, result.err
		}
		return result, nil
	case <-ctx.Done():
		return iteratorResponse{}, context.Cause(ctx)
	}
}

func (o *orderedIterator) Next(ctx context.Context) (value any, done bool, err error) {
	result, err := o.NextOrdered(ctx)
	if err != nil {
		return nil, false, err
	}
	return result.value, result.done, nil
}
