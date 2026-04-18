// Package server composes a brainkit.Kit with the service-mode
// module set (gateway, probes, tracing, audit, optional plugins) and
// manages a single lifecycle on top.
//
// Use New(Config) for explicit configuration, QuickStart for sensible
// demo defaults, or LoadConfig for YAML-driven setup.
//
//	srv, err := server.QuickStart("my-app", "/var/brainkit",
//	    server.WithSecretKey(os.Getenv("SECRET_KEY")))
//	if err != nil { log.Fatal(err) }
//	defer srv.Close()
//
//	ctx, cancel := signal.NotifyContext(context.Background(),
//	    syscall.SIGINT, syscall.SIGTERM)
//	defer cancel()
//	_ = srv.Start(ctx)
package server
