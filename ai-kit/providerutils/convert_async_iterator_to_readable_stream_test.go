// Ported from: packages/provider-utils/src/convert-async-iterator-to-readable-stream.test.ts
package providerutils

import "testing"

func TestConvertAsyncIteratorToReadableStream_Basic(t *testing.T) {
	input := make(chan int, 3)
	input <- 1
	input <- 2
	input <- 3
	close(input)

	output := ConvertAsyncIteratorToReadableStream(input)

	var collected []int
	for val := range output {
		collected = append(collected, val)
	}

	if len(collected) != 3 {
		t.Fatalf("expected 3 items, got %d", len(collected))
	}
	for i, v := range collected {
		if v != i+1 {
			t.Errorf("expected %d at index %d, got %d", i+1, i, v)
		}
	}
}

func TestConvertAsyncIteratorToReadableStream_Empty(t *testing.T) {
	input := make(chan string)
	close(input)

	output := ConvertAsyncIteratorToReadableStream(input)

	var collected []string
	for val := range output {
		collected = append(collected, val)
	}

	if len(collected) != 0 {
		t.Errorf("expected 0 items, got %d", len(collected))
	}
}

func TestConvertAsyncIteratorToReadableStream_Strings(t *testing.T) {
	input := make(chan string, 2)
	input <- "hello"
	input <- "world"
	close(input)

	output := ConvertAsyncIteratorToReadableStream(input)

	var collected []string
	for val := range output {
		collected = append(collected, val)
	}

	if len(collected) != 2 {
		t.Fatalf("expected 2 items, got %d", len(collected))
	}
	if collected[0] != "hello" || collected[1] != "world" {
		t.Errorf("unexpected values: %v", collected)
	}
}
