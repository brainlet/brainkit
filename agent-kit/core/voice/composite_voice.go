// Ported from: packages/core/src/voice/composite-voice.ts
package voice

import (
	"io"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	sm "github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	tm "github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// ---------------------------------------------------------------------------
// AI SDK model types — imported from ai-kit
// ---------------------------------------------------------------------------

// TranscriptionModel is the AI SDK TranscriptionModel interface (V3 spec).
// Imported from brainlink/experiments/ai-kit/provider/transcriptionmodel.
type TranscriptionModel = tm.TranscriptionModel

// SpeechModel is the AI SDK SpeechModel interface (V3 spec).
// Imported from brainlink/experiments/ai-kit/provider/speechmodel.
type SpeechModel = sm.SpeechModel

// supportedSpecificationVersions lists the AI SDK spec versions we accept.
var supportedSpecificationVersions = []string{"v2", "v3"}

// isTranscriptionModel checks if an interface value satisfies TranscriptionModel
// and has a supported specification version.
func isTranscriptionModel(obj any) (TranscriptionModel, bool) {
	m, ok := obj.(TranscriptionModel)
	if !ok {
		return nil, false
	}
	for _, v := range supportedSpecificationVersions {
		if m.SpecificationVersion() == v {
			return m, true
		}
	}
	return nil, false
}

// isSpeechModel checks if an interface value satisfies SpeechModel
// and has a supported specification version.
func isSpeechModel(obj any) (SpeechModel, bool) {
	m, ok := obj.(SpeechModel)
	if !ok {
		return nil, false
	}
	for _, v := range supportedSpecificationVersions {
		if m.SpecificationVersion() == v {
			return m, true
		}
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// CompositeVoice
// ---------------------------------------------------------------------------

// CompositeVoiceConfig holds the configuration for creating a CompositeVoice.
type CompositeVoiceConfig struct {
	// Input is the listen/transcription provider. Can be a MastraVoice or TranscriptionModel.
	Input any
	// Output is the speak/TTS provider. Can be a MastraVoice or SpeechModel.
	Output any
	// Realtime is an optional real-time voice provider.
	Realtime MastraVoice
}

// CompositeVoice delegates to separate providers for speaking, listening,
// and real-time communication. It auto-wraps AI SDK models into the
// appropriate AISDK voice adapters.
type CompositeVoice struct {
	*MastraVoiceBase
	speakProvider    MastraVoice
	listenProvider   MastraVoice
	realtimeProvider MastraVoice
}

// NewCompositeVoice creates a new CompositeVoice from the given config.
func NewCompositeVoice(cfg CompositeVoiceConfig) *CompositeVoice {
	cv := &CompositeVoice{
		MastraVoiceBase:  NewMastraVoiceBase(nil),
		realtimeProvider: cfg.Realtime,
	}

	// Auto-wrap AI SDK models for input (transcription)
	if cfg.Input != nil {
		if mv, ok := cfg.Input.(MastraVoice); ok {
			cv.listenProvider = mv
		} else if tm, ok := isTranscriptionModel(cfg.Input); ok {
			// TODO: replace with NewAISDKTranscription(tm) once aisdk package is wired up
			_ = tm
			cv.listenProvider = nil // Will be set by aisdk package integration
		}
	}

	// Auto-wrap AI SDK models for output (speech)
	if cfg.Output != nil {
		if mv, ok := cfg.Output.(MastraVoice); ok {
			cv.speakProvider = mv
		} else if sm, ok := isSpeechModel(cfg.Output); ok {
			// TODO: replace with NewAISDKSpeech(sm) once aisdk package is wired up
			_ = sm
			cv.speakProvider = nil // Will be set by aisdk package integration
		}
	}

	return cv
}

// SetSpeakProvider sets the speak provider (used by aisdk auto-wrap integration).
func (cv *CompositeVoice) SetSpeakProvider(p MastraVoice) {
	cv.speakProvider = p
}

// SetListenProvider sets the listen provider (used by aisdk auto-wrap integration).
func (cv *CompositeVoice) SetListenProvider(p MastraVoice) {
	cv.listenProvider = p
}

// Speak converts text to speech using the configured provider.
func (cv *CompositeVoice) Speak(input string, options *SpeakOptions) (io.Reader, error) {
	if cv.realtimeProvider != nil {
		return cv.realtimeProvider.Speak(input, options)
	}
	if cv.speakProvider != nil {
		return cv.speakProvider.Speak(input, options)
	}

	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_COMPOSITE_NO_SPEAK_PROVIDER",
		Text:     "No speak provider or realtime provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// SpeakStream converts a text stream to speech using the configured provider.
func (cv *CompositeVoice) SpeakStream(input io.Reader, options *SpeakOptions) (io.Reader, error) {
	if cv.realtimeProvider != nil {
		return cv.realtimeProvider.SpeakStream(input, options)
	}
	if cv.speakProvider != nil {
		return cv.speakProvider.SpeakStream(input, options)
	}

	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_COMPOSITE_NO_SPEAK_PROVIDER",
		Text:     "No speak provider or realtime provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// Listen converts speech to text using the configured provider.
func (cv *CompositeVoice) Listen(audioStream io.Reader, options any) (string, error) {
	if cv.realtimeProvider != nil {
		return cv.realtimeProvider.Listen(audioStream, options)
	}
	if cv.listenProvider != nil {
		return cv.listenProvider.Listen(audioStream, options)
	}

	return "", mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_COMPOSITE_NO_LISTEN_PROVIDER",
		Text:     "No listen provider or realtime provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// GetSpeakers returns available speakers from the configured provider.
func (cv *CompositeVoice) GetSpeakers() ([]SpeakerInfo, error) {
	if cv.realtimeProvider != nil {
		return cv.realtimeProvider.GetSpeakers()
	}
	if cv.speakProvider != nil {
		return cv.speakProvider.GetSpeakers()
	}

	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_COMPOSITE_NO_SPEAKERS_PROVIDER",
		Text:     "No speak provider or realtime provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// GetListener returns the listener status from the configured provider.
func (cv *CompositeVoice) GetListener() (*ListenerInfo, error) {
	if cv.realtimeProvider != nil {
		return cv.realtimeProvider.GetListener()
	}
	if cv.listenProvider != nil {
		return cv.listenProvider.GetListener()
	}

	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_COMPOSITE_NO_LISTENER_PROVIDER",
		Text:     "No listener provider or realtime provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// UpdateConfig updates configuration on the realtime provider, if present.
func (cv *CompositeVoice) UpdateConfig(options map[string]any) {
	if cv.realtimeProvider == nil {
		return
	}
	cv.realtimeProvider.UpdateConfig(options)
}

// Connect initializes a WebSocket or WebRTC connection for real-time communication.
func (cv *CompositeVoice) Connect(options map[string]any) error {
	if cv.realtimeProvider == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VOICE_COMPOSITE_NO_REALTIME_PROVIDER_CONNECT",
			Text:     "No realtime provider configured",
			Domain:   mastraerror.ErrorDomainMastraVoice,
			Category: mastraerror.ErrorCategoryUser,
		})
	}
	return cv.realtimeProvider.Connect(options)
}

// Send relays audio data to the voice provider for real-time processing.
func (cv *CompositeVoice) Send(audioData io.Reader) error {
	if cv.realtimeProvider == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VOICE_COMPOSITE_NO_REALTIME_PROVIDER_SEND",
			Text:     "No realtime provider configured",
			Domain:   mastraerror.ErrorDomainMastraVoice,
			Category: mastraerror.ErrorCategoryUser,
		})
	}
	return cv.realtimeProvider.Send(audioData)
}

// SendInt16 relays Int16 PCM audio data to the voice provider for real-time processing.
func (cv *CompositeVoice) SendInt16(audioData []int16) error {
	if cv.realtimeProvider == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VOICE_COMPOSITE_NO_REALTIME_PROVIDER_SEND",
			Text:     "No realtime provider configured",
			Domain:   mastraerror.ErrorDomainMastraVoice,
			Category: mastraerror.ErrorCategoryUser,
		})
	}
	return cv.realtimeProvider.SendInt16(audioData)
}

// Answer triggers voice providers to respond.
func (cv *CompositeVoice) Answer(options map[string]any) error {
	if cv.realtimeProvider == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VOICE_COMPOSITE_NO_REALTIME_PROVIDER_ANSWER",
			Text:     "No realtime provider configured",
			Domain:   mastraerror.ErrorDomainMastraVoice,
			Category: mastraerror.ErrorCategoryUser,
		})
	}
	return cv.realtimeProvider.Answer(options)
}

// AddInstructions equips the voice provider with instructions.
func (cv *CompositeVoice) AddInstructions(instructions string) {
	if cv.realtimeProvider == nil {
		return
	}
	cv.realtimeProvider.AddInstructions(instructions)
}

// AddTools equips the voice provider with tools.
func (cv *CompositeVoice) AddTools(tools ToolsInput) {
	if cv.realtimeProvider == nil {
		return
	}
	cv.realtimeProvider.AddTools(tools)
}

// Close disconnects from the WebSocket or WebRTC connection.
func (cv *CompositeVoice) Close() {
	if cv.realtimeProvider == nil {
		// In TS this throws; in Go we match that behavior with a panic
		// since the TS version also throws synchronously.
		panic(mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VOICE_COMPOSITE_NO_REALTIME_PROVIDER_CLOSE",
			Text:     "No realtime provider configured",
			Domain:   mastraerror.ErrorDomainMastraVoice,
			Category: mastraerror.ErrorCategoryUser,
		}).Error())
	}
	cv.realtimeProvider.Close()
}

// On registers an event listener on the realtime provider.
func (cv *CompositeVoice) On(event VoiceEventType, callback VoiceEventCallback) {
	if cv.realtimeProvider == nil {
		panic(mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VOICE_COMPOSITE_NO_REALTIME_PROVIDER_ON",
			Text:     "No realtime provider configured",
			Domain:   mastraerror.ErrorDomainMastraVoice,
			Category: mastraerror.ErrorCategoryUser,
		}).Error())
	}
	cv.realtimeProvider.On(event, callback)
}

// Off removes an event listener from the realtime provider.
func (cv *CompositeVoice) Off(event VoiceEventType, callback VoiceEventCallback) {
	if cv.realtimeProvider == nil {
		panic(mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VOICE_COMPOSITE_NO_REALTIME_PROVIDER_OFF",
			Text:     "No realtime provider configured",
			Domain:   mastraerror.ErrorDomainMastraVoice,
			Category: mastraerror.ErrorCategoryUser,
		}).Error())
	}
	cv.realtimeProvider.Off(event, callback)
}
