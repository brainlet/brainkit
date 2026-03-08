// Ported from: packages/core/src/voice/voice.ts
package voice

import (
	"io"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ToolsInput is a stub for ../agent.ToolsInput.
// TODO: import from agent package once ported.
type ToolsInput = map[string]any

// ---------------------------------------------------------------------------
// Event types
// ---------------------------------------------------------------------------

// VoiceEventType represents the type of voice event.
// Core values are "speaking", "writing", "error", but any string is valid.
type VoiceEventType = string

// Predefined voice event types.
const (
	VoiceEventSpeaker VoiceEventType = "speaker"
	VoiceEventSpeaking VoiceEventType = "speaking"
	VoiceEventWriting  VoiceEventType = "writing"
	VoiceEventError    VoiceEventType = "error"
)

// SpeakerEventData is the data emitted with a "speaker" event.
type SpeakerEventData struct {
	Stream io.Reader
}

// SpeakingEventData is the data emitted with a "speaking" event.
type SpeakingEventData struct {
	Audio *string
}

// WritingEventData is the data emitted with a "writing" event.
type WritingEventData struct {
	Text string
	Role string // "assistant" or "user"
}

// ErrorEventData is the data emitted with an "error" event.
type ErrorEventData struct {
	Message string
	Code    *string
	Details any
}

// ---------------------------------------------------------------------------
// Configuration types
// ---------------------------------------------------------------------------

// BuiltInModelConfig holds the name and optional API key for a built-in model.
type BuiltInModelConfig struct {
	Name   string
	APIKey string
}

// RealtimeConfig holds configuration for real-time voice connections.
type RealtimeConfig struct {
	Model   string
	APIKey  string
	Options any
}

// VoiceConfig holds configuration for constructing a MastraVoice.
type VoiceConfig struct {
	ListeningModel *BuiltInModelConfig
	SpeechModel    *BuiltInModelConfig
	Speaker        string
	Name           string
	RealtimeConfig *RealtimeConfig
}

// ---------------------------------------------------------------------------
// Speaker metadata
// ---------------------------------------------------------------------------

// SpeakerInfo represents a single available voice/speaker.
type SpeakerInfo struct {
	VoiceID  string
	Metadata any
}

// ListenerInfo represents the listener status.
type ListenerInfo struct {
	Enabled bool
}

// ---------------------------------------------------------------------------
// SpeakOptions
// ---------------------------------------------------------------------------

// SpeakOptions holds options for the Speak method.
type SpeakOptions struct {
	Speaker string
	Extra   map[string]any
}

// ---------------------------------------------------------------------------
// VoiceEventCallback
// ---------------------------------------------------------------------------

// VoiceEventCallback is a callback function for voice events.
type VoiceEventCallback func(data any)

// ---------------------------------------------------------------------------
// MastraVoice interface
// ---------------------------------------------------------------------------

// MastraVoice defines the abstract interface that all voice providers must implement.
// This corresponds to the abstract class MastraVoice in the TypeScript source.
type MastraVoice interface {
	// Speak converts text to speech.
	// input is either a plain string or an io.Reader for streaming text.
	// Returns an audio stream (io.Reader) or nil if in chat mode.
	Speak(input string, options *SpeakOptions) (io.Reader, error)

	// SpeakStream converts a text stream to speech.
	// Returns an audio stream (io.Reader) or nil if in chat mode.
	SpeakStream(input io.Reader, options *SpeakOptions) (io.Reader, error)

	// Listen converts speech to text.
	// audioStream is the audio input to transcribe.
	// options is provider-specific.
	// Returns transcribed text.
	Listen(audioStream io.Reader, options any) (string, error)

	// UpdateConfig updates the provider configuration.
	UpdateConfig(options map[string]any)

	// Connect initializes a WebSocket or WebRTC connection for real-time communication.
	Connect(options map[string]any) error

	// Send relays audio data to the voice provider for real-time processing.
	Send(audioData io.Reader) error

	// SendInt16 relays Int16 PCM audio data to the voice provider for real-time processing.
	SendInt16(audioData []int16) error

	// Answer triggers voice providers to respond.
	Answer(options map[string]any) error

	// AddInstructions equips the voice provider with instructions.
	AddInstructions(instructions string)

	// AddTools equips the voice provider with tools.
	AddTools(tools ToolsInput)

	// Close disconnects from the WebSocket or WebRTC connection.
	Close()

	// On registers an event listener.
	On(event VoiceEventType, callback VoiceEventCallback)

	// Off removes an event listener.
	Off(event VoiceEventType, callback VoiceEventCallback)

	// GetSpeakers returns available speakers/voices.
	GetSpeakers() ([]SpeakerInfo, error)

	// GetListener returns the listener status.
	GetListener() (*ListenerInfo, error)
}

// ---------------------------------------------------------------------------
// MastraVoiceBase provides default implementations for MastraVoice methods.
// Concrete voice providers should embed this struct and override methods
// they support.
// ---------------------------------------------------------------------------

// MastraVoiceBase provides shared state and default (warn-only) implementations
// for all MastraVoice methods. It mirrors the non-abstract method bodies from
// the TypeScript abstract class.
type MastraVoiceBase struct {
	*agentkit.MastraBase
	ListeningModel *BuiltInModelConfig
	SpeechModel    *BuiltInModelConfig
	Speaker        string
	Realtime       *RealtimeConfig
}

// NewMastraVoiceBase creates a new MastraVoiceBase with the given config.
func NewMastraVoiceBase(cfg *VoiceConfig) *MastraVoiceBase {
	var c VoiceConfig
	if cfg != nil {
		c = *cfg
	}

	base := agentkit.NewMastraBase(agentkit.MastraBaseOptions{
		Component: logger.RegisteredLoggerVoice,
		Name:      c.Name,
	})

	return &MastraVoiceBase{
		MastraBase:     base,
		ListeningModel: c.ListeningModel,
		SpeechModel:    c.SpeechModel,
		Speaker:        c.Speaker,
		Realtime:       c.RealtimeConfig,
	}
}

// Speak is a default implementation that warns the provider does not support speech.
func (v *MastraVoiceBase) Speak(_ string, _ *SpeakOptions) (io.Reader, error) {
	v.Logger().Warn("speak not implemented by this voice provider")
	return nil, nil
}

// SpeakStream is a default implementation that warns the provider does not support speech.
func (v *MastraVoiceBase) SpeakStream(_ io.Reader, _ *SpeakOptions) (io.Reader, error) {
	v.Logger().Warn("speakStream not implemented by this voice provider")
	return nil, nil
}

// Listen is a default implementation that warns the provider does not support listening.
func (v *MastraVoiceBase) Listen(_ io.Reader, _ any) (string, error) {
	v.Logger().Warn("listen not implemented by this voice provider")
	return "", nil
}

// UpdateConfig is a default implementation that warns the provider does not support config updates.
func (v *MastraVoiceBase) UpdateConfig(_ map[string]any) {
	v.Logger().Warn("updateConfig not implemented by this voice provider")
}

// Connect is a default implementation that warns the provider does not support connections.
func (v *MastraVoiceBase) Connect(_ map[string]any) error {
	v.Logger().Warn("connect not implemented by this voice provider")
	return nil
}

// Send is a default implementation that warns the provider does not support sending audio.
func (v *MastraVoiceBase) Send(_ io.Reader) error {
	v.Logger().Warn("send not implemented by this voice provider")
	return nil
}

// SendInt16 is a default implementation that warns the provider does not support sending Int16 audio.
func (v *MastraVoiceBase) SendInt16(_ []int16) error {
	v.Logger().Warn("send not implemented by this voice provider")
	return nil
}

// Answer is a default implementation that warns the provider does not support answering.
func (v *MastraVoiceBase) Answer(_ map[string]any) error {
	v.Logger().Warn("answer not implemented by this voice provider")
	return nil
}

// AddInstructions is a default no-op implementation.
func (v *MastraVoiceBase) AddInstructions(_ string) {
	// Default implementation - voice providers can override if they support this feature
}

// AddTools is a default no-op implementation.
func (v *MastraVoiceBase) AddTools(_ ToolsInput) {
	// Default implementation - voice providers can override if they support this feature
}

// Close is a default implementation that warns the provider does not support closing.
func (v *MastraVoiceBase) Close() {
	v.Logger().Warn("close not implemented by this voice provider")
}

// On is a default implementation that warns the provider does not support event listeners.
func (v *MastraVoiceBase) On(_ VoiceEventType, _ VoiceEventCallback) {
	v.Logger().Warn("on not implemented by this voice provider")
}

// Off is a default implementation that warns the provider does not support removing event listeners.
func (v *MastraVoiceBase) Off(_ VoiceEventType, _ VoiceEventCallback) {
	v.Logger().Warn("off not implemented by this voice provider")
}

// GetSpeakers is a default implementation that returns an empty list.
func (v *MastraVoiceBase) GetSpeakers() ([]SpeakerInfo, error) {
	v.Logger().Warn("getSpeakers not implemented by this voice provider")
	return []SpeakerInfo{}, nil
}

// GetListener is a default implementation that returns disabled.
func (v *MastraVoiceBase) GetListener() (*ListenerInfo, error) {
	v.Logger().Warn("getListener not implemented by this voice provider")
	return &ListenerInfo{Enabled: false}, nil
}
