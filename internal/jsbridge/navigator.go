package jsbridge

import quickjs "github.com/buke/quickjs-go"

// NavigatorPolyfill provides globalThis.navigator for environment detection.
// Various libraries check navigator.userAgent, navigator.onLine, etc.
type NavigatorPolyfill struct{}

func Navigator() *NavigatorPolyfill { return &NavigatorPolyfill{} }

func (p *NavigatorPolyfill) Name() string { return "navigator" }

func (p *NavigatorPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
if (typeof navigator === "undefined") {
  globalThis.navigator = {
    userAgent: "Mozilla/5.0 (compatible; QuickJS/0.1; Go)",
    language: "en-US",
    languages: ["en-US", "en"],
    platform: "Linux x86_64",
    hardwareConcurrency: 1,
    onLine: true,
    cookieEnabled: false,
    maxTouchPoints: 0,
    mediaDevices: {},
    permissions: {},
    clipboard: {},
    locks: { request: function() { return Promise.resolve(); } },
  };
}
`)
}
