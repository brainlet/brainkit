# hello-server

The smallest server-mode brainkit program: load a YAML config, build
the composed runtime, run until signal.

```sh
go run ./examples/hello-server
```

Expected output:

```
listening on :8080
```

Curl the gateway's health endpoint to verify it's up:

```sh
curl http://127.0.0.1:8080/health
```

## What it shows

- `configfile.Load` reads the YAML, substitutes `$VAR` / `${VAR}`
  against `os.Environ`, and projects onto the runtime `Config`.
- The example imports `server/configfile/storebackends/sqlite` because
  `kit_store_path` defaults to `<fs_root>/kit.db`.
- The example imports `server/configfile/packageboot` so top-level
  `packages:` entries can auto-deploy when you scale the YAML up.
- `server.New` composes the YAML-selected modules behind a single
  lifecycle. This example imports the `server/standard/commands`,
  `server/standard/packages`, and `server/standard/server` profiles,
  enough for command modules, package deploy, gateway, and probes.
- `brainkit new server <name>` stamps this same shape into a new
  module when you want to ship it as a service.

## Scaling up

Add providers, storages, or auto-deployed packages by uncommenting
the corresponding sections in `brainkit.yaml`. See
`brainkit/server/testdata/example.yaml` for the full field set.
