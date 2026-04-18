# hello-embedded

The smallest useful brainkit program: embed a Kit in a Go process,
deploy an inline `.ts` handler, call it, print the reply.

```sh
go run ./examples/hello-embedded
```

Expected output:

```
{"greeting":"hello, world"}
```

Wire the same pattern into a real service by:

- Swapping `brainkit.Memory()` for `brainkit.EmbeddedNATS()` or
  `brainkit.NATS(url)` so other Kits on the same transport can call
  your handlers.
- Loading `.ts` packages from disk with
  `brainkit.PackageFromDir("./agents/support")` instead of the
  inline helper.
- Registering AI providers via `brainkit.Config.Providers` or
  `kit.Providers().Register(...)` and calling them from `.ts` with
  `model("openai", "gpt-4o-mini")`.
