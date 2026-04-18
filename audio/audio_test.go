package audio

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNullSinkPlayIsNoOp(t *testing.T) {
	if err := Null().Play(context.Background(), []byte{1, 2, 3}, "audio/mpeg"); err != nil {
		t.Fatalf("Null().Play returned %v, want nil", err)
	}
}

func TestFuncSinkInvokesCallback(t *testing.T) {
	var gotMime string
	var gotBytes int
	sink := Func(func(ctx context.Context, audio []byte, mime string) error {
		gotMime = mime
		gotBytes = len(audio)
		return nil
	})
	if err := sink.Play(context.Background(), []byte{1, 2, 3, 4}, "audio/wav"); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if gotMime != "audio/wav" || gotBytes != 4 {
		t.Errorf("callback saw mime=%q bytes=%d, want audio/wav/4", gotMime, gotBytes)
	}
}

func TestFuncSinkPropagatesError(t *testing.T) {
	boom := errors.New("boom")
	sink := Func(func(context.Context, []byte, string) error { return boom })
	if err := sink.Play(context.Background(), nil, ""); !errors.Is(err, boom) {
		t.Errorf("Play err = %v, want %v", err, boom)
	}
}

func TestCompositeFansOut(t *testing.T) {
	var plays atomic.Int32
	s1 := Func(func(context.Context, []byte, string) error { plays.Add(1); return nil })
	s2 := Func(func(context.Context, []byte, string) error { plays.Add(1); return nil })
	s3 := Func(func(context.Context, []byte, string) error { plays.Add(1); return nil })
	sink := Composite(s1, s2, s3)
	if err := sink.Play(context.Background(), []byte{1}, "audio/mpeg"); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if got := plays.Load(); got != 3 {
		t.Errorf("plays = %d, want 3", got)
	}
}

func TestCompositeJoinsErrors(t *testing.T) {
	errA := errors.New("a failed")
	errB := errors.New("b failed")
	sink := Composite(
		Func(func(context.Context, []byte, string) error { return errA }),
		Func(func(context.Context, []byte, string) error { return errB }),
		Func(func(context.Context, []byte, string) error { return nil }),
	)
	err := sink.Play(context.Background(), nil, "")
	if err == nil {
		t.Fatal("expected joined error")
	}
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Errorf("joined err = %v, want both a/b wrapped", err)
	}
	if !strings.Contains(err.Error(), "a failed") || !strings.Contains(err.Error(), "b failed") {
		t.Errorf("joined err message missing both causes: %q", err.Error())
	}
}

func TestCompositeSkipsNils(t *testing.T) {
	var plays atomic.Int32
	ok := Func(func(context.Context, []byte, string) error { plays.Add(1); return nil })
	sink := Composite(nil, ok, nil)
	if err := sink.Play(context.Background(), nil, ""); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if got := plays.Load(); got != 1 {
		t.Errorf("plays = %d, want 1", got)
	}
}

func TestCompositeEmptyReturnsNull(t *testing.T) {
	// No sinks → Null (no error, no panic).
	if err := Composite().Play(context.Background(), nil, ""); err != nil {
		t.Fatalf("empty Composite should no-op, got %v", err)
	}
}
