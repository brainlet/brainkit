package jsbridge

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

type conformanceCase struct {
	group string
	name  string
	code  string
	want  string
}

func TestJSBridgeConformancePacks(t *testing.T) {
	root := t.TempDir()
	b := newTestBridge(t,
		Inspect(),
		Console(),
		Process(),
		Encoding(),
		Streams(),
		Crypto(),
		URL(),
		Timers(),
		Scheduling(),
		Abort(),
		Events(),
		DOMEvents(),
		StructuredClone(),
		Navigator(),
		Performance(),
		Intl(),
		ErrorCompat(),
		NodeStreams(),
		Buffer(),
		Path(),
		NodeCompat(),
		OS(),
		Net(),
		DNS(),
		Zlib(),
		WebAssembly(),
		FS(root),
		Exec(root),
		Fetch(),
		Audio(),
	)

	for _, tc := range []conformanceCase{
		{
			group: "web",
			name:  "url-searchparams",
			code: `
				const u = new URL("/v1?q=1", "https://example.test/base");
				u.searchParams.append("q", "2");
				return JSON.stringify({ href: u.href, q: u.searchParams.getAll("q").join("|") });
			`,
			want: `{"href":"https://example.test/v1?q=1&q=2","q":"1|2"}`,
		},
		{
			group: "web",
			name:  "abort-eventtarget",
			code: `
				const ac = new AbortController();
				let seen = 0;
				ac.signal.addEventListener("abort", () => { seen += 1; });
				ac.abort("done");
				const target = new EventTarget();
				target.addEventListener("custom", () => { seen += 2; });
				target.dispatchEvent(new Event("custom"));
				return JSON.stringify({ aborted: ac.signal.aborted, seen });
			`,
			want: `{"aborted":true,"seen":3}`,
		},
		{
			group: "web",
			name:  "headers-request-response-formdata-file",
			code: `
				const headers = new Headers({ "X-Test": "1" });
				headers.append("x-test", "2");
				const file = new File(["abc"], "a.txt", { type: "text/plain" });
				const form = new FormData();
				form.append("field", "value");
				form.append("file", file);
				const formText = await new Response(form).text();
				const req = new Request("https://example.test/post", { method: "POST", body: "body" });
				const blobText = await new Blob(["hi"]).text();
				return JSON.stringify({
					header: headers.get("x-test"),
					method: req.method,
					boundary: formText.includes("brainkitResponseBoundary"),
					file: file.name + ":" + file.type,
					blobText
				});
			`,
			want: `{"header":"1, 2","method":"POST","boundary":true,"file":"a.txt:text/plain","blobText":"hi"}`,
		},
		{
			group: "web",
			name:  "readable-transform-response",
			code: `
				const upper = new TransformStream({
					transform(chunk, controller) {
						controller.enqueue(new TextEncoder().encode(new TextDecoder().decode(chunk).toUpperCase()));
					}
				});
				const stream = new ReadableStream({
					start(controller) {
						controller.enqueue(new TextEncoder().encode("ok"));
						controller.close();
					}
				}).pipeThrough(upper);
				return JSON.stringify({ text: await new Response(stream).text() });
			`,
			want: `{"text":"OK"}`,
		},
		{
			group: "web",
			name:  "audio-shape",
			code: `
				const audio = new Audio(new Uint8Array([1, 2, 3]));
				return JSON.stringify({
					paused: audio.paused,
					play: typeof audio.play,
					pause: typeof audio.pause,
					addEventListener: typeof audio.addEventListener
				});
			`,
			want: `{"paused":true,"play":"function","pause":"function","addEventListener":"function"}`,
		},
		{
			group: "node-core",
			name:  "process-buffer-crypto-os-path",
			code: `
				const hash = crypto.createHash("sha256").update("abc").digest("hex").slice(0, 8);
				return JSON.stringify({
					node: typeof process.versions.node,
					buffer: Buffer.from("ok").toString("hex"),
					hash,
					platform: typeof os.platform(),
					path: path.join("a", "b")
				});
			`,
			want: `{"node":"string","buffer":"6f6b","hash":"ba7816bf","platform":"string","path":"a/b"}`,
		},
		{
			group: "node-core",
			name:  "events-streams-timers",
			code: `
				const emitter = new EventEmitter();
				let seen = 0;
				emitter.once("value", (n) => { seen = n; });
				emitter.emit("value", 7);
				await timersPromises.setTimeout(1);
				const chunks = [];
				for await (const chunk of stream.Readable.from(["a", "b"])) {
					chunks.push(String(chunk));
				}
				return JSON.stringify({ seen, chunks: chunks.join("") });
			`,
			want: `{"seen":7,"chunks":"ab"}`,
		},
		{
			group: "node-core",
			name:  "fs-child-process",
			code: `
				await fs.promises.writeFile("conf.txt", "file-ok");
				const text = await fs.promises.readFile("conf.txt", { encoding: "utf8" });
				const result = await child_process.execFile("sh", ["-c", "printf exec-ok"]);
				return JSON.stringify({ text, stdout: result.stdout });
			`,
			want: `{"text":"file-ok","stdout":"exec-ok"}`,
		},
		{
			group: "node-core",
			name:  "util-assert-querystring-stringdecoder-zlib-dns",
			code: `
				assert.strictEqual(querystring.stringify({ a: "b" }), "a=b");
				const decoder = new StringDecoder("utf8");
				const decoded = decoder.write(Buffer.from([0xe2, 0x82, 0xac]));
				const gzip = zlib.gzipSync(Buffer.from("zip"));
				const unzipped = zlib.gunzipSync(gzip).toString();
				const lookup = await dns.promises.lookup("localhost");
				return JSON.stringify({
					decoded,
					promise: util.types.isPromise(Promise.resolve()),
					unzipped,
					lookupFamily: typeof lookup.family
				});
			`,
			want: `{"decoded":"€","promise":true,"unzipped":"zip","lookupFamily":"number"}`,
		},
		{
			group: "unsupported",
			name:  "typed-failures",
			code: `
				const failures = [];
				for (const fn of [
					() => http.createServer(),
					() => https.createServer(),
					() => net.createServer(),
					() => tls.createServer(),
					() => new worker_threads.Worker("worker.js"),
					() => crypto.createCipheriv("aes-128-cbc", "bad", "bad"),
					() => zlib.brotliCompressSync(Buffer.from("x")),
				]) {
					try { fn(); } catch (err) { failures.push({
						code: err && err.code || "",
						boundaryClass: err && err.boundaryClass || "",
						message: String(err && err.message || err)
					}); }
				}
				return JSON.stringify({
					count: failures.length,
					boundaryCount: failures.filter((err) => err.code === "BRAINKIT_UNSUPPORTED_BOUNDARY").length,
					serverBoundaryCount: failures.filter((err) => err.boundaryClass === "server-listener").length,
					workerBoundaryCount: failures.filter((err) => err.boundaryClass === "worker").length,
					typed: failures.every((err) => err.code === "BRAINKIT_UNSUPPORTED_BOUNDARY" || err.message.includes("not available") || err.message.includes("requires") || err.message.includes("unsupported"))
				});
			`,
			want: `{"count":7,"boundaryCount":5,"serverBoundaryCount":4,"workerBoundaryCount":1,"typed":true}`,
		},
	} {
		t.Run(filepath.Join(tc.group, tc.name), func(t *testing.T) {
			got := runConformanceCase(t, b, tc)
			if got != tc.want {
				t.Fatalf("conformance result = %s, want %s", got, tc.want)
			}
		})
	}
}

func runConformanceCase(t *testing.T, b *Bridge, tc conformanceCase) string {
	t.Helper()
	val, err := b.EvalAsync("conformance/"+tc.group+"/"+tc.name+".js", `(async () => {`+tc.code+`})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	return val.String()
}

func TestJSBridgeConformanceResourcesDrain(t *testing.T) {
	root := t.TempDir()
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Buffer(), FS(root), Exec(root), Timers(), Fetch(), Audio())

	val, err := b.EvalAsync("conformance/resources.js", `(async () => {
		globalThis.__handle = await fs.promises.open("tracked.txt", "w+");
		setTimeout(function(){}, 10000);
		return "tracked";
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	val.Free()
	waitForBridgeResource(t, b, "fs.fileHandles", 1)
	waitForBridgeResource(t, b, "timers.timeouts", 1)

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v snapshot=%+v", err, b.DebugSnapshot())
	}
	if len(b.DebugSnapshot().Resources) != 0 {
		t.Fatalf("resources after conformance bridge close = %+v", b.DebugSnapshot().Resources)
	}
}
