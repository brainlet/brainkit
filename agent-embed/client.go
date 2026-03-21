package agentembed

import internalagent "github.com/brainlet/brainkit/internal/embed/agent"

type ClientConfig = internalagent.ClientConfig
type Client = internalagent.Client
type QuickGenerateParams = internalagent.QuickGenerateParams
type QuickStreamParams = internalagent.QuickStreamParams

func NewClient(cfg ClientConfig) *Client {
	return internalagent.NewClient(cfg)
}
