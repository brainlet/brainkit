// Package audio is the public surface for brainkit's audio
// playback story. It names the Sink contract that `.ts` code
// targets through the web-standard `Audio` polyfill, and ships
// the small handful of helpers users reach for most often:
// Null, Func, and Composite.
//
// Concrete implementations live in subpackages:
//
//	import "github.com/brainlet/brainkit/audio/local"  // desktop (oto)
//
// Wire on the kit:
//
//	kit, _ := brainkit.New(brainkit.Config{
//	    Audio: local.New(),                            // or
//	    // Audio: audio.Composite(local.New(), myBusSink), // fan-out
//	})
//
// Sinks mirror jsbridge.AudioSink signature-for-signature, so
// any Sink satisfies the internal polyfill contract without a
// shim. Keeping the public interface here (and not in the
// internal package) means user code never imports internals.
package audio

import (
	"context"
	"errors"
	"sync"
)

// Sink plays an audio buffer. Implementations decide what
// "play" means for their environment: a desktop sink writes
// the decoded PCM to the system audio device; a bus sink
// publishes the bytes for another subscriber to handle; a
// gateway sink streams them to a connected browser.
//
// mime is the inferred MIME type ("audio/mpeg", "audio/wav",
// "audio/ogg", "audio/opus", "audio/webm",
// "application/octet-stream"). Implementations should still
// sniff if the hint is unknown — the JS polyfill probes
// container magic numbers already but passes along its best
// guess.
//
// Play must respect ctx. It's called on a goroutine owned by
// the bridge; cancel happens when `.ts` code calls audio.pause()
// or the kit drains.
type Sink interface {
	Play(ctx context.Context, audio []byte, mime string) error
}

// Null is the silent sink — discards bytes. It's the default
// when Config.Audio is nil, so portable agent code calling
// `new Audio(stream).play()` runs everywhere without guards.
func Null() Sink { return nullSink{} }

type nullSink struct{}

func (nullSink) Play(context.Context, []byte, string) error { return nil }

// SinkFunc lifts a plain function into a Sink. Convenient for
// one-off adapters (publishing to a bus, writing to an HTTP
// response, handing bytes to a test capture):
//
//	audio.Func(func(ctx context.Context, buf []byte, mime string) error {
//	    return kit.PublishRaw(ctx, "audio.played", buf)
//	})
type SinkFunc func(ctx context.Context, audio []byte, mime string) error

// Play implements Sink.
func (f SinkFunc) Play(ctx context.Context, audio []byte, mime string) error {
	if f == nil {
		return nil
	}
	return f(ctx, audio, mime)
}

// Func is a shorthand constructor matching the naming users
// expect (audio.Func(fn)) — returns a Sink bound to fn.
func Func(fn func(ctx context.Context, audio []byte, mime string) error) Sink {
	return SinkFunc(fn)
}

// Composite plays the same bytes through every sink in order.
// Typical shape: local playback + bus broadcast so the desktop
// operator hears the audio while subscribed agents see it too.
//
// All sinks run concurrently; Composite returns when all have
// returned. Any sink that returns an error has its error joined
// into the result; the rest still play.
func Composite(sinks ...Sink) Sink {
	// Drop nils so callers can pass conditionally-constructed
	// sinks without a trail of `if s != nil { ... }`.
	clean := sinks[:0]
	for _, s := range sinks {
		if s != nil {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		return Null()
	}
	if len(clean) == 1 {
		return clean[0]
	}
	return composite(append([]Sink(nil), clean...))
}

type composite []Sink

func (c composite) Play(ctx context.Context, buf []byte, mime string) error {
	errCh := make(chan error, len(c))
	var wg sync.WaitGroup
	for _, s := range c {
		wg.Add(1)
		go func(sink Sink) {
			defer wg.Done()
			errCh <- sink.Play(ctx, buf, mime)
		}(s)
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for e := range errCh {
		if e != nil {
			errs = append(errs, e)
		}
	}
	return errors.Join(errs...)
}
