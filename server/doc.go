// Package server composes a brainkit.Kit with an explicit service-mode module
// set and manages a single lifecycle on top.
//
// Use New(Config) for explicit configuration, package server/configfile for
// YAML-driven setup, or package server/quickstart for sensible demo defaults.
//
//	srv, err := quickstart.New("my-app", "/var/brainkit",
//	    quickstart.WithSecretKey(os.Getenv("SECRET_KEY")))
//	if err != nil { log.Fatal(err) }
//	defer srv.Close()
//
//	ctx, cancel := signal.NotifyContext(context.Background(),
//	    syscall.SIGINT, syscall.SIGTERM)
//	defer cancel()
//	_ = srv.Start(ctx)
package server
