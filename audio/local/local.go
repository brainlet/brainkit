// Package local plays audio bytes through the host's default
// audio device using oto + a small pool of pure-Go decoders.
//
// This is the opt-in desktop backend for brainkit's Audio
// polyfill. Wire it on Config.Audio:
//
//	import "github.com/brainlet/brainkit/audio/local"
//
//	kit, _ := brainkit.New(brainkit.Config{
//	    Audio: local.New(),
//	})
//
// Inside `.ts` deployments, web-standard `new Audio(stream).play()`
// calls now actually play through the speaker. With no Audio
// configured, the polyfill is a no-op so portable agent code
// runs unchanged on headless / server kits.
//
// Format support:
//   - audio/mpeg (MP3) — decoded via hajimehoshi/go-mp3
//   - audio/wav — minimal RIFF parser (PCM payload only)
//
// Anything else returns an error from Play; the polyfill
// surfaces it as an "error" event on the JS Audio object.
package local

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/brainlet/brainkit/audio"
	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

// Sink plays audio through the host's default device.
type Sink struct {
	once    sync.Once
	initErr error
	ctx     *oto.Context
	ready   chan struct{}

	// Output device parameters used when (re)initializing the
	// shared oto.Context. The first non-zero MP3 sample rate
	// observed wins; subsequent payloads must resample if they
	// don't match (oto only supports one context per process).
	sampleRate int
	channels   int
}

// Option configures a Sink.
type Option func(*Sink)

// WithSampleRate pins the output context to a specific rate.
// Default: 24000 (matches OpenAI TTS MP3 output).
func WithSampleRate(rate int) Option {
	return func(s *Sink) { s.sampleRate = rate }
}

// WithChannels pins the channel count. Default: 2 (stereo).
// Mono MP3s are upmixed at decode time.
func WithChannels(ch int) Option {
	return func(s *Sink) { s.channels = ch }
}

// New constructs a desktop audio sink. Initialization of the
// underlying oto.Context happens on first Play (lazy) so kits
// that never play audio don't hold the audio device open.
func New(opts ...Option) *Sink {
	s := &Sink{sampleRate: 24000, channels: 2, ready: make(chan struct{})}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Sink) initOnce() error {
	s.once.Do(func() {
		ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   s.sampleRate,
			ChannelCount: s.channels,
			Format:       oto.FormatSignedInt16LE,
		})
		if err != nil {
			s.initErr = fmt.Errorf("local-audio: init oto: %w", err)
			close(s.ready)
			return
		}
		s.ctx = ctx
		go func() {
			<-ready
			close(s.ready)
		}()
	})
	if s.initErr != nil {
		return s.initErr
	}
	<-s.ready
	return nil
}

// Play satisfies audio.Sink. Decodes the audio and blocks until
// playback finishes (or ctx is cancelled).
func (s *Sink) Play(ctx context.Context, audio []byte, mime string) error {
	if err := s.initOnce(); err != nil {
		return err
	}

	pcm, srcRate, srcChannels, err := decode(audio, mime)
	if err != nil {
		return err
	}
	pcm = matchOutput(pcm, srcRate, srcChannels, s.sampleRate, s.channels)

	player := s.ctx.NewPlayer(bytes.NewReader(pcm))
	player.Play()

	// Poll IsPlaying with cancellation. oto doesn't expose a
	// done channel, but the player drains the byte reader and
	// stops on its own.
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			if !player.IsPlaying() {
				return nil
			}
		}
	}
}

// decode dispatches on MIME / sniffed magic and returns
// signed-16-bit-LE interleaved PCM with its native sample rate
// and channel count.
func decode(data []byte, mime string) ([]byte, int, int, error) {
	if len(data) == 0 {
		return nil, 0, 0, errors.New("local-audio: empty payload")
	}
	switch {
	case mime == "audio/mpeg" || mime == "audio/mp3" ||
		(len(data) >= 3 && data[0] == 'I' && data[1] == 'D' && data[2] == '3') ||
		(len(data) >= 2 && data[0] == 0xFF && (data[1]&0xE0) == 0xE0):
		return decodeMP3(data)
	case mime == "audio/wav" || mime == "audio/x-wav" ||
		(len(data) >= 4 && data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F'):
		return decodeWAV(data)
	default:
		return nil, 0, 0, fmt.Errorf("local-audio: unsupported format %q", mime)
	}
}

func decodeMP3(data []byte) ([]byte, int, int, error) {
	dec, err := mp3.NewDecoder(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("local-audio: decode mp3 header: %w", err)
	}
	pcm, err := io.ReadAll(dec)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("local-audio: decode mp3 payload: %w", err)
	}
	// go-mp3 always emits 16-bit stereo PCM at the source rate.
	return pcm, dec.SampleRate(), 2, nil
}

func decodeWAV(data []byte) ([]byte, int, int, error) {
	// Minimal RIFF/WAVE PCM parser — enough for OpenAI's WAV
	// output (16-bit signed LE PCM). Skips through subchunks
	// looking for "fmt " then "data".
	if len(data) < 44 {
		return nil, 0, 0, errors.New("local-audio: wav too short")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, 0, errors.New("local-audio: not a RIFF/WAVE container")
	}
	var (
		sampleRate uint32
		channels   uint16
		bits       uint16
		pcm        []byte
	)
	off := 12
	for off+8 <= len(data) {
		id := string(data[off : off+4])
		size := binary.LittleEndian.Uint32(data[off+4 : off+8])
		body := data[off+8 : min(off+8+int(size), len(data))]
		switch id {
		case "fmt ":
			if len(body) >= 16 {
				channels = binary.LittleEndian.Uint16(body[2:4])
				sampleRate = binary.LittleEndian.Uint32(body[4:8])
				bits = binary.LittleEndian.Uint16(body[14:16])
			}
		case "data":
			pcm = body
		}
		// Subchunks are word-aligned.
		next := off + 8 + int(size)
		if size%2 == 1 {
			next++
		}
		off = next
	}
	if pcm == nil || sampleRate == 0 {
		return nil, 0, 0, errors.New("local-audio: malformed wav")
	}
	if bits != 16 {
		return nil, 0, 0, fmt.Errorf("local-audio: wav bit depth %d not supported (need 16-bit PCM)", bits)
	}
	return pcm, int(sampleRate), int(channels), nil
}

// matchOutput coerces decoded PCM to the (rate, channels) the
// oto context was opened with. The first Play wins the rate, so
// later mismatches resample with a cheap nearest-neighbor pass.
// Channel adjustment is upmix mono→stereo by duplication or
// downmix stereo→mono by averaging — good enough for voice
// playback; not a substitute for a real DSP pipeline.
func matchOutput(pcm []byte, srcRate, srcCh, dstRate, dstCh int) []byte {
	if srcCh != dstCh {
		pcm = remapChannels(pcm, srcCh, dstCh)
		srcCh = dstCh
	}
	if srcRate != dstRate && dstRate > 0 {
		pcm = resampleNearest(pcm, srcRate, dstRate, srcCh)
	}
	return pcm
}

func remapChannels(pcm []byte, src, dst int) []byte {
	if src == dst || src == 0 || dst == 0 {
		return pcm
	}
	frames := len(pcm) / (2 * src)
	out := make([]byte, frames*2*dst)
	for i := 0; i < frames; i++ {
		// Average the source channels into a single sample.
		var sum int32
		for c := 0; c < src; c++ {
			s := int16(binary.LittleEndian.Uint16(pcm[(i*src+c)*2:]))
			sum += int32(s)
		}
		mono := int16(sum / int32(src))
		// Fan that mono frame across the destination channels.
		for c := 0; c < dst; c++ {
			binary.LittleEndian.PutUint16(out[(i*dst+c)*2:], uint16(mono))
		}
	}
	return out
}

func resampleNearest(pcm []byte, srcRate, dstRate, channels int) []byte {
	if srcRate == dstRate || srcRate == 0 || channels == 0 {
		return pcm
	}
	frame := 2 * channels
	srcFrames := len(pcm) / frame
	dstFrames := int(int64(srcFrames) * int64(dstRate) / int64(srcRate))
	out := make([]byte, dstFrames*frame)
	for i := 0; i < dstFrames; i++ {
		j := int(int64(i) * int64(srcRate) / int64(dstRate))
		if j >= srcFrames {
			j = srcFrames - 1
		}
		copy(out[i*frame:(i+1)*frame], pcm[j*frame:(j+1)*frame])
	}
	return out
}

// Compile-time assertion that Sink implements the contract.
var _ audio.Sink = (*Sink)(nil)
