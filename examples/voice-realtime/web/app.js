// Browser side of brainkit's voice-realtime example.
//
// What it does:
//
//  1. captureLoop — opens the mic via getUserMedia, feeds the
//     raw samples through an AudioWorklet that downsamples the
//     browser's native sample rate (usually 48 kHz) to the
//     24 kHz PCM16 mono frames OpenAI's realtime API expects,
//     and ships each frame as a binary WS message.
//
//  2. playLoop — incoming binary frames are PCM16 24 kHz; we
//     create a short AudioBuffer for each chunk, queue them on
//     a single AudioContext so playback is seamless.
//
//  3. transcripts — JSON text frames from the agent update the
//     DOM log.
//
// Audio format constraints come from the OpenAI realtime spec;
// the inline worklet is ~20 lines and keeps the client
// dependency-free.

const SAMPLE_RATE = 24000;
const FRAME_SAMPLES = 2400; // 100 ms per frame at 24 kHz

const toggle = document.getElementById("toggle");
const label = toggle.querySelector(".label");
const logEl = document.getElementById("log");
const status = document.getElementById("status");

let ws = null;
let micContext = null;
let micStream = null;
let worklet = null;

let playContext = null;
let playheadTime = 0;

function setStatus(state, text) {
  status.className = "status " + state;
  status.textContent = text;
}

function appendLog(role, text) {
  if (!text) return;
  let last = logEl.lastElementChild;
  if (last && last.dataset.role === role && last.dataset.open === "true") {
    last.textContent += text;
    return;
  }
  const li = document.createElement("li");
  li.className = "role-" + role;
  li.dataset.role = role;
  li.dataset.open = "true";
  li.textContent = text;
  logEl.appendChild(li);
  logEl.scrollTop = logEl.scrollHeight;
}

function closeLog(role) {
  const last = logEl.lastElementChild;
  if (last && last.dataset.role === role) last.dataset.open = "false";
}

async function start() {
  setStatus("connecting", "connecting");

  const wsURL = (location.protocol === "https:" ? "wss://" : "ws://") + location.host + "/ws/voice";
  ws = new WebSocket(wsURL);
  ws.binaryType = "arraybuffer";

  ws.addEventListener("open", async () => {
    setStatus("live", "live");
    toggle.classList.add("live");
    label.textContent = "Stop";
    await openMic();
  });

  ws.addEventListener("message", (ev) => {
    if (typeof ev.data === "string") {
      try {
        const msg = JSON.parse(ev.data);
        if (msg.type === "transcript") {
          if (!msg.text || msg.text === "\n") closeLog(msg.role);
          else appendLog(msg.role, msg.text);
        }
      } catch (_) {}
      return;
    }
    enqueuePlayback(new Uint8Array(ev.data));
  });

  ws.addEventListener("close", stop);
  ws.addEventListener("error", stop);
}

async function stop() {
  if (ws && ws.readyState !== WebSocket.CLOSED) {
    try { ws.close(); } catch (_) {}
  }
  ws = null;
  if (worklet) { try { worklet.disconnect(); } catch (_) {} worklet = null; }
  if (micStream) {
    micStream.getTracks().forEach((t) => t.stop());
    micStream = null;
  }
  if (micContext) { try { await micContext.close(); } catch (_) {} micContext = null; }
  toggle.classList.remove("live");
  label.textContent = "Start";
  setStatus("idle", "idle");
  playheadTime = 0;
}

async function openMic() {
  micStream = await navigator.mediaDevices.getUserMedia({ audio: {
    channelCount: 1,
    echoCancellation: true,
    noiseSuppression: true,
  } });
  // Use the hardware sample rate; the worklet downsamples to 24 kHz.
  micContext = new AudioContext();
  const source = micContext.createMediaStreamSource(micStream);

  const workletCode = `
class DownsampleProcessor extends AudioWorkletProcessor {
  constructor(opts) {
    super();
    this.dstRate = opts.processorOptions.dstRate;
    this.frameSamples = opts.processorOptions.frameSamples;
    this.buf = new Float32Array(this.frameSamples * 8);
    this.buffered = 0;
    this.srcCursor = 0;
  }

  process(inputs) {
    const input = inputs[0];
    if (!input || !input[0]) return true;
    const src = input[0];
    const srcRate = sampleRate;  // AudioWorkletGlobalScope const
    const ratio = srcRate / this.dstRate;

    for (let i = 0; i < src.length; i++) {
      this.srcCursor += 1;
      if (this.srcCursor >= ratio) {
        this.srcCursor -= ratio;
        this.buf[this.buffered++] = src[i];
        if (this.buffered >= this.frameSamples) {
          const pcm = new Int16Array(this.buffered);
          for (let j = 0; j < this.buffered; j++) {
            const s = Math.max(-1, Math.min(1, this.buf[j]));
            pcm[j] = s < 0 ? s * 0x8000 : s * 0x7FFF;
          }
          this.port.postMessage(pcm.buffer, [pcm.buffer]);
          this.buffered = 0;
        }
      }
    }
    return true;
  }
}
registerProcessor("downsample", DownsampleProcessor);
`;
  const blob = new Blob([workletCode], { type: "application/javascript" });
  await micContext.audioWorklet.addModule(URL.createObjectURL(blob));

  worklet = new AudioWorkletNode(micContext, "downsample", {
    processorOptions: { dstRate: SAMPLE_RATE, frameSamples: FRAME_SAMPLES },
  });
  worklet.port.onmessage = (ev) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    // ev.data is an ArrayBuffer of PCM16 samples.
    ws.send(ev.data);
  };
  source.connect(worklet);
  // Do NOT connect the worklet to destination — we don't want
  // to hear the local mic loopback.
}

function enqueuePlayback(bytes) {
  if (!playContext) {
    playContext = new AudioContext({ sampleRate: SAMPLE_RATE });
    playheadTime = playContext.currentTime + 0.08;
  }
  const samples = bytes.byteLength / 2;
  if (samples === 0) return;
  const pcm = new Int16Array(bytes.buffer, bytes.byteOffset, samples);
  const float = new Float32Array(samples);
  for (let i = 0; i < samples; i++) float[i] = pcm[i] / 0x8000;

  const buffer = playContext.createBuffer(1, samples, SAMPLE_RATE);
  buffer.copyToChannel(float, 0);
  const src = playContext.createBufferSource();
  src.buffer = buffer;
  src.connect(playContext.destination);

  const startAt = Math.max(playContext.currentTime, playheadTime);
  src.start(startAt);
  playheadTime = startAt + buffer.duration;
}

toggle.addEventListener("click", async () => {
  try {
    if (ws && ws.readyState === WebSocket.OPEN) {
      await stop();
    } else {
      await start();
    }
  } catch (err) {
    console.error(err);
    setStatus("idle", "error");
    await stop();
  }
});

setStatus("idle", "idle");
