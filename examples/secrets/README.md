# secrets

Encrypted secret store lifecycle: `Set` -> `Get` -> `Rotate` ->
`List` -> `Delete` through the `modules/secrets` typed bus API.

## Run

```sh
go run ./examples/secrets
```

Expected output:

```
set API_KEY = sk-demo-v1
get API_KEY -> sk-demo-v1
rotate API_KEY -> sk-demo-v2
list: 1 secret(s)
  API_KEY (version=2, updated=2026-04-18T00:40:58-04:00)
delete API_KEY -> Get returns ""
```

## What it shows

- `brainkit.Config.SecretKey` activates the encrypted KV secret
  store. Without it, secrets are written cleartext on top of the
  Kit store (a warning is logged).
- Mount `modules/secrets` and call
  `secretmsg.CallSecretsSet/Get/Rotate/List/Delete` from Go. Each
  entry carries a version counter that bumps on Set / Rotate.
- `Rotate` is sugar for `Set` with a rotation audit event — pair
  it with `modules/plugins` for auto-restart of plugins whose
  env refers to the rotated secret (see
  `modules/plugins/README.md` §"Rotation").

## Accessor vs `$secret:NAME` resolution

Two surfaces share the same underlying store:

- **Go-side:** `secretmsg.CallSecretsGet(kit, ctx,
  secretmsg.SecretsGetMsg{Name: "API_KEY"})` — bus command read.
- **Config-side:** a plugin / module config string `$secret:API_KEY`
  is replaced with the current value at Kit init time. Used for
  `plugins.PluginConfig{Env: map[string]string{"API_KEY":
  "$secret:API_KEY"}}` and similar.

Either surface shares the same encryption + versioning — pick
whichever fits the caller's shape.
