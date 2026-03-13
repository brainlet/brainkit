package jsbridge

import quickjs "github.com/buke/quickjs-go"

// StreamsPolyfill provides ReadableStream, WritableStream, TransformStream,
// TextDecoderStream, and TextEncoderStream.
//
// IMPORTANT: The Encoding polyfill must be loaded before Streams,
// since TextDecoderStream/TextEncoderStream depend on TextEncoder/TextDecoder.
type StreamsPolyfill struct{}

// Streams creates a Web Streams API polyfill.
func Streams() *StreamsPolyfill { return &StreamsPolyfill{} }

func (p *StreamsPolyfill) Name() string { return "streams" }

func (p *StreamsPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, streamsJS)
}

const streamsJS = `
// --- ReadableStream ---

globalThis.ReadableStream = class ReadableStream {
  constructor(underlyingSource, strategy) {
    this._queue = [];
    this._closeRequested = false;
    this._errored = false;
    this._storedError = undefined;
    this._locked = false;
    this._source = underlyingSource || {};
    this._pendingRead = null;
    this._controller = {
      enqueue: (chunk) => {
        if (this._pendingRead) {
          const { resolve } = this._pendingRead;
          this._pendingRead = null;
          resolve({ done: false, value: chunk });
        } else {
          this._queue.push(chunk);
        }
      },
      close: () => {
        this._closeRequested = true;
        if (this._pendingRead) {
          const { resolve } = this._pendingRead;
          this._pendingRead = null;
          resolve({ done: true, value: undefined });
        }
      },
      error: (e) => {
        this._errored = true;
        this._storedError = e;
        if (this._pendingRead) {
          const { reject } = this._pendingRead;
          this._pendingRead = null;
          reject(e);
        }
      },
      desiredSize: 1,
    };
    if (this._source.start) {
      this._source.start(this._controller);
    }
  }

  get locked() { return this._locked; }

  getReader() {
    if (this._locked) throw new TypeError('ReadableStream is already locked');
    this._locked = true;
    const stream = this;
    const reader = {
      _stream: stream,
      _closed: false,
      get closed() { return Promise.resolve(this._closed); },
      async read() {
        const s = this._stream;
        if (s._queue.length > 0) {
          return { done: false, value: s._queue.shift() };
        }
        if (s._closeRequested) {
          this._closed = true;
          return { done: true, value: undefined };
        }
        if (s._errored) throw s._storedError;
        if (s._source.pull) {
          await s._source.pull(s._controller);
          if (s._queue.length > 0) {
            return { done: false, value: s._queue.shift() };
          }
          if (s._closeRequested) {
            this._closed = true;
            return { done: true, value: undefined };
          }
        }
        // No data yet and no pull — wait for enqueue/close/error
        return new Promise((resolve, reject) => {
          s._pendingRead = { resolve, reject };
        });
      },
      cancel(reason) {
        if (this._stream._source.cancel) this._stream._source.cancel(reason);
        this.releaseLock();
        return Promise.resolve();
      },
      releaseLock() {
        this._stream._locked = false;
      },
    };
    return reader;
  }

  tee() {
    const reader = this.getReader();
    let ctrl1, ctrl2;
    const s1 = new ReadableStream({ start(c) { ctrl1 = c; } });
    const s2 = new ReadableStream({ start(c) { ctrl2 = c; } });
    (async () => {
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) { ctrl1.close(); ctrl2.close(); break; }
          ctrl1.enqueue(value);
          ctrl2.enqueue(value);
        }
      } catch (e) {
        try { ctrl1.error(e); } catch(_) {}
        try { ctrl2.error(e); } catch(_) {}
      }
    })();
    return [s1, s2];
  }

  pipeThrough(transform, options) {
    const reader = this.getReader();
    const writer = transform.writable.getWriter();
    (async () => {
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) { await writer.close(); break; }
          await writer.write(value);
        }
      } catch (e) {
        try { await writer.abort(e); } catch(_) {}
      }
    })();
    return transform.readable;
  }

  async pipeTo(dest, options) {
    const reader = this.getReader();
    const writer = dest.getWriter();
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) { await writer.close(); break; }
        await writer.write(value);
      }
    } catch (e) {
      try { await writer.abort(e); } catch(_) {}
    }
  }

  [Symbol.asyncIterator]() {
    const reader = this.getReader();
    return {
      next() { return reader.read(); },
      return() { reader.releaseLock(); return Promise.resolve({ done: true, value: undefined }); },
    };
  }
};

// --- WritableStream ---

globalThis.WritableStream = class WritableStream {
  constructor(underlyingSink, strategy) {
    this._sink = underlyingSink || {};
    this._locked = false;
    this._controller = { error(e) {} };
    if (this._sink.start) this._sink.start(this._controller);
  }

  get locked() { return this._locked; }

  getWriter() {
    if (this._locked) throw new TypeError('WritableStream is already locked');
    this._locked = true;
    const stream = this;
    return {
      _stream: stream,
      ready: Promise.resolve(),
      async write(chunk) {
        if (stream._sink.write) await stream._sink.write(chunk, stream._controller);
      },
      async close() {
        if (stream._sink.close) await stream._sink.close();
      },
      async abort(reason) {
        if (stream._sink.abort) await stream._sink.abort(reason);
      },
      releaseLock() { stream._locked = false; },
    };
  }
};

// --- TransformStream ---

globalThis.TransformStream = class TransformStream {
  constructor(transformer, writableStrategy, readableStrategy) {
    const xf = transformer || {};
    let readableController;

    this.readable = new ReadableStream({
      start(controller) { readableController = controller; },
    });

    this.writable = new WritableStream({
      async write(chunk) {
        if (xf.transform) {
          await xf.transform(chunk, readableController);
        } else {
          readableController.enqueue(chunk);
        }
      },
      async close() {
        if (xf.flush) await xf.flush(readableController);
        readableController.close();
      },
    });
  }
};

// --- TextDecoderStream ---

globalThis.TextDecoderStream = class TextDecoderStream extends TransformStream {
  constructor(encoding, options) {
    const dec = new TextDecoder(encoding || 'utf-8', options);
    super({
      transform(chunk, controller) {
        const text = typeof chunk === 'string' ? chunk : dec.decode(chunk, { stream: true });
        if (text) controller.enqueue(text);
      },
      flush(controller) {
        const text = dec.decode();
        if (text) controller.enqueue(text);
      },
    });
    this.encoding = encoding || 'utf-8';
  }
};

// --- TextEncoderStream ---

globalThis.TextEncoderStream = class TextEncoderStream extends TransformStream {
  constructor() {
    const enc = new TextEncoder();
    super({
      transform(chunk, controller) {
        controller.enqueue(enc.encode(chunk));
      },
    });
    this.encoding = 'utf-8';
  }
};
`
