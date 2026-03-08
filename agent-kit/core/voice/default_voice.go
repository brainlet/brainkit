// Ported from: packages/core/src/voice/default-voice.ts
package voice

import (
	"io"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// DefaultVoice is a voice provider that always returns errors,
// indicating that no real voice provider has been configured.
type DefaultVoice struct {
	*MastraVoiceBase
}

// NewDefaultVoice creates a new DefaultVoice.
func NewDefaultVoice() *DefaultVoice {
	return &DefaultVoice{
		MastraVoiceBase: NewMastraVoiceBase(nil),
	}
}

// Speak always returns an error indicating no voice provider is configured.
func (d *DefaultVoice) Speak(_ string, _ *SpeakOptions) (io.Reader, error) {
	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_DEFAULT_NO_SPEAK_PROVIDER",
		Text:     "No voice provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// SpeakStream always returns an error indicating no voice provider is configured.
func (d *DefaultVoice) SpeakStream(_ io.Reader, _ *SpeakOptions) (io.Reader, error) {
	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_DEFAULT_NO_SPEAK_PROVIDER",
		Text:     "No voice provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// Listen always returns an error indicating no voice provider is configured.
func (d *DefaultVoice) Listen(_ io.Reader, _ any) (string, error) {
	return "", mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_DEFAULT_NO_LISTEN_PROVIDER",
		Text:     "No voice provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// GetSpeakers always returns an error indicating no voice provider is configured.
func (d *DefaultVoice) GetSpeakers() ([]SpeakerInfo, error) {
	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_DEFAULT_NO_SPEAKERS_PROVIDER",
		Text:     "No voice provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}

// GetListener always returns an error indicating no voice provider is configured.
func (d *DefaultVoice) GetListener() (*ListenerInfo, error) {
	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VOICE_DEFAULT_NO_LISTENER_PROVIDER",
		Text:     "No voice provider configured",
		Domain:   mastraerror.ErrorDomainMastraVoice,
		Category: mastraerror.ErrorCategoryUser,
	})
}
