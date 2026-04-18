package jsbridge

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordingSink captures payloads handed to play so tests can
// assert what the polyfill ships out.
type recordingSink struct {
	mu      sync.Mutex
	payload []byte
	mime    string
	delay   time.Duration
	err     error
	calls   int
}

func (r *recordingSink) Play(ctx context.Context, audio []byte, mime string) error {
	r.mu.Lock()
	r.payload = append([]byte(nil), audio...)
	r.mime = mime
	r.calls++
	delay := r.delay
	err := r.err
	r.mu.Unlock()
	if delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return err
}

func TestAudioPlayWithSink(t *testing.T) {
	sink := &recordingSink{}
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Audio(AudioWithSink(sink)))
	val, err := b.EvalAsync("audio.js", `(async function() {
		// Crafted MP3 ID3 header + payload.
		var bytes = new Uint8Array([0x49, 0x44, 0x33, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10]);
		var audio = new Audio(bytes);
		var ended = false;
		audio.addEventListener("ended", function() { ended = true; });
		await audio.play();
		return JSON.stringify({
			ended: ended,
			paused: audio.paused,
			endedFlag: audio.ended,
		});
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	if !strings.Contains(val.String(), `"ended":true`) {
		t.Errorf("expected ended event, got %s", val.String())
	}
	if !strings.Contains(val.String(), `"paused":true`) {
		t.Errorf("expected paused after play, got %s", val.String())
	}
	if sink.calls != 1 {
		t.Errorf("sink calls = %d, want 1", sink.calls)
	}
	if sink.mime != "audio/mpeg" {
		t.Errorf("sniffed mime = %q, want audio/mpeg", sink.mime)
	}
	if len(sink.payload) != 10 {
		t.Errorf("payload bytes = %d, want 10", len(sink.payload))
	}
}

func TestAudioPauseCancelsPlayback(t *testing.T) {
	sink := &recordingSink{delay: 5 * time.Second}
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Timers(), Audio(AudioWithSink(sink)))
	val, err := b.EvalAsync("audio.js", `(async function() {
		var bytes = new Uint8Array([0x49, 0x44, 0x33, 0x04, 0x00]);
		var audio = new Audio(bytes);
		setTimeout(function() { audio.pause(); }, 50);
		await audio.play();
		return JSON.stringify({
			paused: audio.paused,
			ended: audio.ended,
		});
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	if !strings.Contains(val.String(), `"ended":false`) {
		t.Errorf("pause should leave ended=false, got %s", val.String())
	}
}

func TestAudioDefaultsToNullSink(t *testing.T) {
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Audio())
	val, err := b.EvalAsync("audio.js", `(async function() {
		var bytes = new Uint8Array([1,2,3,4]);
		var audio = new Audio(bytes);
		await audio.play();
		return JSON.stringify({ paused: audio.paused, ended: audio.ended });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	if !strings.Contains(val.String(), `"ended":true`) {
		t.Errorf("null sink should still resolve to ended, got %s", val.String())
	}
}
