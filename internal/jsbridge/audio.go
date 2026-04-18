package jsbridge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	quickjs "github.com/buke/quickjs-go"
)

// AudioSink plays an audio buffer. Implementations decide what
// "play" means for their environment: a desktop sink writes the
// decoded PCM to the system audio device; a bus sink emits the
// bytes to subscribers; a server sink writes them to an HTTP /
// WebSocket response. NullSink (default) discards the bytes so
// agent code that calls `new Audio(stream).play()` is portable
// across environments.
//
// Format hint is the MIME type when known ("audio/mpeg",
// "audio/wav", "audio/ogg", "audio/opus", "audio/webm",
// "application/octet-stream"). Implementations should sniff the
// actual format from the bytes if the hint is unknown.
//
// Play is called from a Go goroutine; it must respect ctx for
// cancellation (e.g. when audio.pause() is invoked or the kit
// drains).
type AudioSink interface {
	Play(ctx context.Context, audio []byte, mime string) error
}

// NullSink is the zero-value sink — discards audio silently.
// Useful for headless tests, server-side kits, and as the
// default when the host hasn't wired anything else.
type NullSink struct{}

// Play satisfies AudioSink and discards the audio.
func (NullSink) Play(_ context.Context, _ []byte, _ string) error { return nil }

// AudioPolyfill provides a web-standard `Audio` class on
// globalThis. The class is always installed; what `play()`
// actually does is delegated to the configured AudioSink.
//
// Construct via Audio(...). Defaults to NullSink so deployments
// without an explicit sink resolve play() as a no-op rather
// than throwing.
type AudioPolyfill struct {
	sink   AudioSink
	bridge *Bridge

	mu      sync.Mutex
	nextID  uint64
	playing map[uint64]context.CancelFunc
	dones   map[uint64]audioDone
}

type audioDone struct {
	ch  chan error
	ctx context.Context
}

// AudioOption configures an AudioPolyfill.
type AudioOption func(*AudioPolyfill)

// AudioWithSink wires a sink for playback. Without this, plays
// resolve immediately as a no-op (NullSink).
func AudioWithSink(sink AudioSink) AudioOption {
	return func(p *AudioPolyfill) {
		if sink != nil {
			p.sink = sink
		}
	}
}

// Audio creates an audio polyfill. Zero options = NullSink.
func Audio(opts ...AudioOption) *AudioPolyfill {
	p := &AudioPolyfill{
		sink:    NullSink{},
		playing: make(map[uint64]context.CancelFunc),
		dones:   make(map[uint64]audioDone),
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *AudioPolyfill) attachDone(handle uint64, ch chan error, ctx context.Context) {
	p.mu.Lock()
	p.dones[handle] = audioDone{ch: ch, ctx: ctx}
	p.mu.Unlock()
}

func (p *AudioPolyfill) takeDone(handle uint64) (chan error, context.Context) {
	p.mu.Lock()
	d, ok := p.dones[handle]
	delete(p.dones, handle)
	p.mu.Unlock()
	if !ok {
		return nil, nil
	}
	return d.ch, d.ctx
}

// Name implements Polyfill.
func (p *AudioPolyfill) Name() string { return "audio" }

// SetBridge implements polyfills that need access to Bridge.Go.
func (p *AudioPolyfill) SetBridge(b *Bridge) { p.bridge = b }

// Setup installs globalThis.Audio + the bridge functions.
func (p *AudioPolyfill) Setup(ctx *quickjs.Context) error {
	polyfill := p

	// __go_audio_start(b64, mime) → handle (sync).
	// Spawns the sink playback in a goroutine and returns the
	// handle immediately so JS can call __go_audio_stop() while
	// playback is in flight.
	ctx.Globals().Set("__go_audio_start", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("audio.start: bytes required"))
		}
		b64 := args[0].ToString()
		mime := ""
		if len(args) >= 2 {
			mime = args[1].ToString()
		}
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("audio.start: decode bytes: %w", err))
		}

		handle := atomic.AddUint64(&polyfill.nextID, 1)
		playCtx, cancel := context.WithCancel(context.Background())
		doneCh := make(chan error, 1)

		polyfill.mu.Lock()
		polyfill.playing[handle] = cancel
		polyfill.mu.Unlock()

		polyfill.bridge.Go(func(goCtx context.Context) {
			result := make(chan error, 1)
			go func() { result <- polyfill.sink.Play(playCtx, raw, mime) }()
			var perr error
			select {
			case perr = <-result:
			case <-goCtx.Done():
				cancel()
				perr = <-result
			}
			polyfill.mu.Lock()
			delete(polyfill.playing, handle)
			polyfill.mu.Unlock()
			doneCh <- perr
			close(doneCh)
		})

		// Stash the channel under a JS-visible global so the
		// matching __go_audio_wait can await it.
		polyfill.attachDone(handle, doneCh, playCtx)

		return qctx.NewInt64(int64(handle))
	}))

	// __go_audio_wait(handle) → Promise<JSON> — resolves with
	// {status:"ended"|"cancelled"|"error", error?:string} when the
	// sink returns. Resolves only once per handle.
	ctx.Globals().Set("__go_audio_wait", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("audio.wait: handle required"))
		}
		handle := uint64(args[0].ToInt64())
		doneCh, playCtx := polyfill.takeDone(handle)
		if doneCh == nil {
			return qctx.ThrowError(fmt.Errorf("audio.wait: unknown handle %d", handle))
		}
		return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			polyfill.bridge.Go(func(_ context.Context) {
				perr := <-doneCh
				status := "ended"
				errMsg := ""
				if playCtx.Err() != nil {
					status = "cancelled"
				} else if perr != nil {
					status = "error"
					errMsg = perr.Error()
				}
				payload, _ := json.Marshal(map[string]interface{}{
					"handle": handle,
					"status": status,
					"error":  errMsg,
				})
				out := string(payload)
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolve(qctx.NewString(out))
				})
			})
		})
	}))

	// __go_audio_stop(handle) → void — cancels playback.
	ctx.Globals().Set("__go_audio_stop", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.NewUndefined()
		}
		handle := uint64(args[0].ToInt64())
		polyfill.mu.Lock()
		cancel, ok := polyfill.playing[handle]
		polyfill.mu.Unlock()
		if ok {
			cancel()
		}
		return qctx.NewUndefined()
	}))

	return evalJS(ctx, audioJS)
}

// audioJS implements globalThis.Audio.
//
// Mirrors the HTMLAudioElement subset agent code typically uses:
//
//   const audio = new Audio(src);    // src: URL | path | Buffer | Uint8Array | Blob | Node Readable
//   audio.src = "...";                // late-bind
//   await audio.play();               // resolves when playback ends
//   audio.pause();                    // cancels in-flight playback
//   audio.addEventListener("ended", fn);
//   audio.onended = fn;
//
// `play()` materializes the source into bytes, sniffs the MIME
// type, and dispatches to the wired AudioSink. Without a sink
// (NullSink), play resolves immediately so portable agent code
// runs everywhere without conditionals.
const audioJS = `
(function() {
  "use strict";

  function inferMime(bytes, srcHint) {
    // Sniff well-known audio container magic numbers; fall back
    // to a hint extracted from the source URL/path/extension.
    if (bytes && bytes.length >= 4) {
      // ID3 tag (MP3 with metadata).
      if (bytes[0] === 0x49 && bytes[1] === 0x44 && bytes[2] === 0x33) return "audio/mpeg";
      // Raw MP3 sync word.
      if (bytes[0] === 0xFF && (bytes[1] & 0xE0) === 0xE0) return "audio/mpeg";
      // RIFF .... WAVE.
      if (bytes[0] === 0x52 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x46) return "audio/wav";
      // OggS (vorbis or opus).
      if (bytes[0] === 0x4F && bytes[1] === 0x67 && bytes[2] === 0x67 && bytes[3] === 0x53) return "audio/ogg";
      // FLAC (fLaC).
      if (bytes[0] === 0x66 && bytes[1] === 0x4C && bytes[2] === 0x61 && bytes[3] === 0x43) return "audio/flac";
    }
    if (typeof srcHint === "string") {
      var lower = srcHint.toLowerCase();
      if (lower.endsWith(".mp3")) return "audio/mpeg";
      if (lower.endsWith(".wav")) return "audio/wav";
      if (lower.endsWith(".ogg")) return "audio/ogg";
      if (lower.endsWith(".opus")) return "audio/opus";
      if (lower.endsWith(".flac")) return "audio/flac";
      if (lower.endsWith(".m4a") || lower.endsWith(".aac")) return "audio/aac";
      if (lower.endsWith(".webm")) return "audio/webm";
    }
    return "application/octet-stream";
  }

  function bytesToBase64(u8) {
    var bin = "";
    for (var i = 0; i < u8.length; i++) bin += String.fromCharCode(u8[i] & 0xFF);
    return btoa(bin);
  }

  // Drain whatever shape the caller passed in (URL string, file
  // path, Buffer/Uint8Array, ArrayBuffer, Blob, Node Readable,
  // Web ReadableStream) into a Uint8Array.
  async function _resolveBytes(src) {
    if (src == null) return new Uint8Array(0);
    if (src instanceof Uint8Array) return src;
    if (typeof Buffer !== "undefined" && Buffer.isBuffer && Buffer.isBuffer(src)) {
      return new Uint8Array(src.buffer, src.byteOffset, src.byteLength);
    }
    if (src instanceof ArrayBuffer) return new Uint8Array(src);
    if (ArrayBuffer.isView(src)) return new Uint8Array(src.buffer, src.byteOffset, src.byteLength);
    if (typeof Blob !== "undefined" && src instanceof Blob) {
      return new Uint8Array(await src.arrayBuffer());
    }
    if (typeof src === "string") {
      // URL → fetch. Path or relative path → fs.readFile.
      if (/^(https?:|data:|file:|blob:)/i.test(src)) {
        var resp = await fetch(src);
        return new Uint8Array(await resp.arrayBuffer());
      }
      if (typeof globalThis.fs !== "undefined" && globalThis.fs.readFile) {
        var data = await globalThis.fs.readFile(src);
        if (data instanceof Uint8Array) return data;
        if (typeof data === "string") return new TextEncoder().encode(data);
        return new Uint8Array(data);
      }
      throw new Error("Audio: cannot resolve src '" + src + "' — no fetch or fs available");
    }
    // Node Readable: drain via for-await.
    if (src && typeof src[Symbol.asyncIterator] === "function") {
      var chunks = [];
      var total = 0;
      for await (var chunk of src) {
        var u8 = chunk instanceof Uint8Array ? chunk :
                 (typeof chunk === "string" ? new TextEncoder().encode(chunk) : new Uint8Array(chunk));
        chunks.push(u8);
        total += u8.byteLength;
      }
      var out = new Uint8Array(total);
      var off = 0;
      for (var c of chunks) { out.set(c, off); off += c.byteLength; }
      return out;
    }
    // Web ReadableStream.
    if (src && typeof src.getReader === "function") {
      var reader = src.getReader();
      var chunks2 = [];
      var total2 = 0;
      while (true) {
        var step = await reader.read();
        if (step.done) break;
        var u82 = step.value instanceof Uint8Array ? step.value : new Uint8Array(step.value);
        chunks2.push(u82);
        total2 += u82.byteLength;
      }
      var out2 = new Uint8Array(total2);
      var off2 = 0;
      for (var c2 of chunks2) { out2.set(c2, off2); off2 += c2.byteLength; }
      return out2;
    }
    throw new Error("Audio: unsupported src type");
  }

  class _AudioListeners {
    constructor() { this._m = {}; }
    add(ev, fn) { (this._m[ev] = this._m[ev] || []).push(fn); }
    remove(ev, fn) { var ls = this._m[ev]; if (!ls) return; this._m[ev] = ls.filter(function(x) { return x !== fn; }); }
    fire(ev, payload) {
      var ls = (this._m[ev] || []).slice();
      for (var i = 0; i < ls.length; i++) {
        try { ls[i](payload); } catch(_) {}
      }
    }
  }

  class Audio {
    constructor(src) {
      this._src = src != null ? src : null;
      this._handle = 0;
      this._listeners = new _AudioListeners();
      this._format = "";
      this.paused = true;
      this.ended = false;
      this.currentTime = 0;
      this.duration = NaN;
      this.volume = 1;
      this.muted = false;
      this.loop = false;
      this.autoplay = false;
      this.preload = "auto";
      // on<event> handlers — fire alongside addEventListener.
      this.onplay = null;
      this.onpause = null;
      this.onended = null;
      this.onerror = null;
    }

    get src() { return this._src; }
    set src(v) { this._src = v; this.ended = false; }

    addEventListener(ev, fn) { this._listeners.add(ev, fn); }
    removeEventListener(ev, fn) { this._listeners.remove(ev, fn); }
    _fire(ev, payload) {
      var on = this["on" + ev];
      if (typeof on === "function") {
        try { on.call(this, payload); } catch(_) {}
      }
      this._listeners.fire(ev, payload);
    }

    async play() {
      if (this._src == null) {
        // HTML semantics: play() with no src rejects.
        var err = new Error("Audio: src not set");
        this._fire("error", err);
        throw err;
      }
      var bytes = await _resolveBytes(this._src);
      this._format = inferMime(bytes, typeof this._src === "string" ? this._src : "");
      this.paused = false;
      this.ended = false;
      this._fire("play");
      var b64 = bytesToBase64(bytes);
      var raw;
      try {
        // start() returns the handle synchronously so pause()
        // has something to cancel while playback is in flight.
        this._handle = __go_audio_start(b64, this._format);
        raw = await __go_audio_wait(this._handle);
      } catch (e) {
        this.paused = true;
        this._handle = 0;
        this._fire("error", e);
        throw e;
      }
      this.paused = true;
      this._handle = 0;
      var data = {};
      try { data = JSON.parse(raw); } catch (_) {}
      if (data.status === "error" && data.error) {
        var err2 = new Error(data.error);
        this._fire("error", err2);
        throw err2;
      }
      // "cancelled" (audio.pause()) and "ended" both terminate
      // the play promise; only "ended" fires the ended event.
      if (data.status === "ended") {
        this.ended = true;
        this._fire("ended");
      }
      return undefined;
    }

    pause() {
      if (this._handle) __go_audio_stop(this._handle);
      this.paused = true;
      this._fire("pause");
    }

    load() { /* no-op — bytes are resolved lazily on play() */ }

    canPlayType(type) {
      // Optimistic: report "maybe" for common audio mimes; the
      // sink decides what it actually supports at play time.
      if (typeof type !== "string") return "";
      var t = type.toLowerCase();
      if (t.indexOf("audio/") === 0) return "maybe";
      return "";
    }
  }

  globalThis.Audio = Audio;
})();
`
