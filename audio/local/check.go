package local

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"strings"
	"time"
)

// CheckResult reports what audio/local could observe about the
// host's playback pipeline without needing a human in the loop.
// A fully-passing run means: system volume is non-zero, an
// output device is enumerated, and oto accepted + drained a
// known-good synthetic tone. It does NOT mean "a human heard
// it" — the wire from the OS sink to physical speakers is
// outside what we can observe from a Go process.
type CheckResult struct {
	Platform      string        // runtime.GOOS
	OutputDevice  string        // "MacBook Pro Speakers", "" on non-darwin
	VolumePercent int           // 0–100; -1 if unknown
	Muted         bool          // true if the OS reports muted output
	SampleRate    int           // rate opened on the oto.Context
	Channels      int           // channels opened on the oto.Context
	ToneDuration  time.Duration // how long the tone ran
	BytesWritten  int           // bytes the sink wrote to oto
	PeakSample    int16         // max |sample| in the played PCM — 0 means silent output path
	Warnings      []string      // non-fatal observations
	Err           error         // fatal error from the run
}

// OK returns true when the check completed with no fatal
// errors + no warnings a human would want to see first.
func (r CheckResult) OK() bool { return r.Err == nil && len(r.Warnings) == 0 }

// String summarises the result on one line per field for
// copy-paste into CI logs.
func (r CheckResult) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "platform:       %s\n", r.Platform)
	if r.OutputDevice != "" {
		fmt.Fprintf(&b, "output device:  %s\n", r.OutputDevice)
	} else {
		fmt.Fprintf(&b, "output device:  (not probed on %s)\n", r.Platform)
	}
	if r.VolumePercent >= 0 {
		fmt.Fprintf(&b, "system volume:  %d%% (muted=%v)\n", r.VolumePercent, r.Muted)
	} else {
		fmt.Fprintf(&b, "system volume:  (unknown on %s)\n", r.Platform)
	}
	fmt.Fprintf(&b, "oto context:    %d Hz × %d ch\n", r.SampleRate, r.Channels)
	fmt.Fprintf(&b, "tone duration:  %s\n", r.ToneDuration)
	fmt.Fprintf(&b, "bytes written:  %d\n", r.BytesWritten)
	fmt.Fprintf(&b, "peak sample:    %d / %d (%.1f%%)\n", r.PeakSample, int16(math.MaxInt16), float64(r.PeakSample)/float64(math.MaxInt16)*100)
	for _, w := range r.Warnings {
		fmt.Fprintf(&b, "warning:        %s\n", w)
	}
	if r.Err != nil {
		fmt.Fprintf(&b, "error:          %v\n", r.Err)
	}
	return b.String()
}

// Check plays a 1-second 440 Hz sine wave through the sink +
// reports what it observed. Intended as a headless smoke test
// — you can run it in CI or before a voice demo to confirm
// the pipeline is wired end-to-end even without ears on.
//
// s may be the Sink that will later handle real audio, or a
// fresh local.New() — the check reuses the lazy oto.Context
// init so opening the device counts as part of the smoke.
func (s *Sink) Check(ctx context.Context) CheckResult {
	result := CheckResult{
		Platform:      runtime.GOOS,
		VolumePercent: -1,
		SampleRate:    s.sampleRate,
		Channels:      s.channels,
	}

	// Platform probe — volume, mute, output device. Purely
	// observational; never blocks the play path.
	if device, vol, muted, err := probeDevice(ctx); err == nil {
		result.OutputDevice = device
		result.VolumePercent = vol
		result.Muted = muted
		if muted {
			result.Warnings = append(result.Warnings, "system output is muted; hardware playback suppressed (bytes will still flow through oto)")
		}
		if vol == 0 {
			result.Warnings = append(result.Warnings, "system output volume is 0%; sink will drain but nothing audible")
		}
	} else {
		result.Warnings = append(result.Warnings, fmt.Sprintf("device probe skipped: %v", err))
	}

	// Synthesize a short, well-known tone — 440 Hz sine for
	// 1 second at the sink's native rate so no resample runs.
	tone := synthesizeSineWAV(440.0, time.Second, s.sampleRate, s.channels)

	started := time.Now()
	playErr := s.Play(ctx, tone, "audio/wav")
	result.ToneDuration = time.Since(started)
	result.BytesWritten = len(tone)
	result.PeakSample = peakPCM16(tone[44:]) // skip WAV header
	if playErr != nil {
		result.Err = fmt.Errorf("tone play: %w", playErr)
		return result
	}
	// Duration sanity — a 1-second tone shouldn't return in
	// 10 ms (oto never queued it) or block forever.
	if result.ToneDuration < 500*time.Millisecond {
		result.Warnings = append(result.Warnings, fmt.Sprintf("play returned in %s; expected ≥1s for a 1-second tone — oto may not have blocked on drain", result.ToneDuration))
	}
	return result
}

// synthesizeSineWAV builds a WAV (RIFF + fmt + data) with a
// pure-tone PCM16 payload so Check()'s Play call takes the
// same decode path as a real TTS clip.
func synthesizeSineWAV(freq float64, duration time.Duration, rate, channels int) []byte {
	samples := int(float64(rate) * duration.Seconds())
	pcm := make([]byte, samples*channels*2)
	amp := 0.3 * float64(math.MaxInt16) // -10 dBFS; audible but not painful
	for i := 0; i < samples; i++ {
		v := int16(math.Sin(2*math.Pi*freq*float64(i)/float64(rate)) * amp)
		for c := 0; c < channels; c++ {
			binary.LittleEndian.PutUint16(pcm[(i*channels+c)*2:], uint16(v))
		}
	}
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+len(pcm)))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(rate*channels*2))
	binary.LittleEndian.PutUint16(header[32:34], uint16(channels*2))
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(len(pcm)))
	return append(header, pcm...)
}

// peakPCM16 scans a PCM16 LE payload for the maximum absolute
// sample value. Returns 0 for an all-silence buffer; returns
// int16 max (32767) for a fully-clipped one. Used as a rough
// "did the sink actually receive audio or did something zero
// it out" signal.
func peakPCM16(pcm []byte) int16 {
	var peak int16
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(binary.LittleEndian.Uint16(pcm[i:]))
		abs := v
		if abs < 0 {
			abs = -abs
		}
		if abs > peak {
			peak = abs
		}
	}
	return peak
}
