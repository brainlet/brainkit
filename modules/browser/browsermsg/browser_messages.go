package browsermsg

import browsercap "github.com/brainlet/brainkit/modulecap/browser"

// BrowserSessionLaunchMsg launches a local CDP-capable browser session.
type BrowserSessionLaunchMsg struct {
	Provider       string           `json:"provider,omitempty"`
	ThreadID       string           `json:"threadId,omitempty"`
	Scope          browsercap.Scope `json:"scope,omitempty"`
	ExecutablePath string           `json:"executablePath,omitempty"`
	ProfileDir     string           `json:"profileDir,omitempty"`
	Headless       *bool            `json:"headless,omitempty"`
	Args           []string         `json:"args,omitempty"`
}

func (BrowserSessionLaunchMsg) BusTopic() string { return "browser.session.launch" }

type BrowserSessionLaunchResp struct {
	Session browsercap.SessionInfo `json:"session"`
}

// BrowserSessionCloseMsg closes one browser session owned by the browser module.
type BrowserSessionCloseMsg struct {
	ID string `json:"id"`
}

func (BrowserSessionCloseMsg) BusTopic() string { return "browser.session.close" }

type BrowserSessionCloseResp struct {
	Closed bool `json:"closed"`
}

// BrowserSessionListMsg lists browser sessions owned by the browser module.
type BrowserSessionListMsg struct{}

func (BrowserSessionListMsg) BusTopic() string { return "browser.session.list" }

type BrowserSessionListResp struct {
	Sessions []browsercap.SessionInfo `json:"sessions,omitempty"`
}
