package esbuild

import "strings"

const npmPreviewPackageStubNamespace = "brainkit-package-stub"

var npmPreviewPackageStubs = map[string]string{
	"ws": `
var NativeWebSocket = globalThis.WebSocket;
if (typeof NativeWebSocket !== "function") {
	throw new Error("ws: Brainkit WebSocket global is not available");
}
function BrainkitWebSocket(url, protocols, options) {
	return new NativeWebSocket(url, protocols, options);
}
BrainkitWebSocket.prototype = NativeWebSocket.prototype;
try { Object.defineProperty(BrainkitWebSocket, "CONNECTING", { value: NativeWebSocket.CONNECTING }); } catch (_) {}
try { Object.defineProperty(BrainkitWebSocket, "OPEN", { value: NativeWebSocket.OPEN }); } catch (_) {}
try { Object.defineProperty(BrainkitWebSocket, "CLOSING", { value: NativeWebSocket.CLOSING }); } catch (_) {}
try { Object.defineProperty(BrainkitWebSocket, "CLOSED", { value: NativeWebSocket.CLOSED }); } catch (_) {}
function unsupported(name) {
	return function() {
		throw globalThis.__brainkit_unsupported_boundary({
			api: "ws." + name,
			importPath: "ws",
			boundaryClass: "server-listener",
			suggestedOwner: "modules/gateway or future server runtime profile",
			owner: "modules/packages/bundlers/esbuild npm-preview ws adapter",
		});
	};
}
BrainkitWebSocket.WebSocket = BrainkitWebSocket;
BrainkitWebSocket.Server = unsupported("Server");
BrainkitWebSocket.WebSocketServer = BrainkitWebSocket.Server;
BrainkitWebSocket.createWebSocketStream = unsupported("createWebSocketStream");
module.exports = BrainkitWebSocket;
module.exports.default = BrainkitWebSocket;
module.exports.WebSocket = BrainkitWebSocket;
module.exports.Server = BrainkitWebSocket.Server;
module.exports.WebSocketServer = BrainkitWebSocket.WebSocketServer;
module.exports.createWebSocketStream = BrainkitWebSocket.createWebSocketStream;
`,
}

func npmPreviewPackageStub(specifier string) (string, string, bool) {
	specifier = strings.TrimPrefix(specifier, "node:")
	if _, ok := npmPreviewPackageStubs[specifier]; ok {
		return specifier, specifier, true
	}
	return "", "", false
}
