package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/brainlet/brainkit"
	cliconfig "github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start a brainkit instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cliconfig.LoadConfig()
			if err != nil {
				return err
			}
			nodeCfg, err := cliconfig.BuildNodeConfig(cfg)
			if err != nil {
				return fmt.Errorf("build config: %w", err)
			}
			node, err := brainkit.NewNode(nodeCfg)
			if err != nil {
				return fmt.Errorf("create node: %w", err)
			}
			if err := node.Start(context.Background()); err != nil {
				node.Close()
				return fmt.Errorf("start: %w", err)
			}

			// Start control API server on a random local port
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				node.Close()
				return fmt.Errorf("control api listen: %w", err)
			}
			port := ln.Addr().(*net.TCPAddr).Port

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/bus", controlBusHandler(node))
			mux.HandleFunc("POST /api/stream", controlStreamHandler(node))
			controlSrv := &http.Server{Handler: mux}
			go controlSrv.Serve(ln)

			// Write pidfile with control port so CLI commands can discover it
			pidDir := "data"
			os.MkdirAll(pidDir, 0755)
			pidFile := filepath.Join(pidDir, "brainkit.pid")
			os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", port)), 0644)
			defer os.Remove(pidFile)

			logger := node.Kernel.Logger()
			logger.Info("brainkit started",
				slog.String("namespace", node.Kernel.Namespace()),
				slog.String("transport", cfg.Transport),
				slog.String("control", fmt.Sprintf("http://127.0.0.1:%d", port)),
				slog.String("workspace", cfg.FSRoot),
			)

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			logger.Info("shutting down")

			// Shutdown with a hard deadline — don't hang on stuck connections.
			// Second Ctrl+C force-exits immediately.
			shutdownDone := make(chan error, 1)
			go func() {
				shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer shutCancel()
				controlSrv.Shutdown(shutCtx)
				shutdownDone <- node.Shutdown(shutCtx)
			}()

			select {
			case err := <-shutdownDone:
				return err
			case <-sigCh:
				logger.Warn("force exit")
				os.Exit(1)
				return nil
			}
		},
	}
}

// controlBusHandler handles POST /api/bus — generic bus request-reply over HTTP.
// Body: {"topic":"kit.health","payload":{}}
// Response: {"payload":{...}} or {"error":"..."}
func controlBusHandler(node *brainkit.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
			return
		}
		if req.Topic == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "topic is required"})
			return
		}

		// Use the HTTP request context — inherits the client's timeout.
		// No server-side timeout cap — the client controls how long to wait.
		ctx := r.Context()

		correlationID := uuid.NewString()
		replyTo := req.Topic + ".reply." + correlationID

		replyCh := make(chan messages.Message, 1)
		unsub, err := node.Kernel.SubscribeRaw(ctx, replyTo, func(msg messages.Message) {
			select {
			case replyCh <- msg:
			default:
			}
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "subscribe: " + err.Error()})
			return
		}
		defer unsub()

		pubCtx := messaging.WithPublishMeta(ctx, correlationID, replyTo)
		if _, err := node.Kernel.PublishRaw(pubCtx, req.Topic, req.Payload); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "publish: " + err.Error()})
			return
		}

		select {
		case msg := <-replyCh:
			writeJSON(w, http.StatusOK, map[string]json.RawMessage{"payload": msg.Payload})
		case <-ctx.Done():
			writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "timeout waiting for response"})
		}
	}
}

// controlStreamHandler handles POST /api/stream — bus publish + stream all events as NDJSON.
// Each intermediate message (done=false) is written as a JSON line and flushed.
// The terminal message (done=true) is written last, then the response closes.
func controlStreamHandler(node *brainkit.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
			return
		}
		if req.Topic == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "topic is required"})
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
			return
		}

		ctx := r.Context()
		correlationID := uuid.NewString()
		replyTo := req.Topic + ".reply." + correlationID

		eventCh := make(chan messages.Message, 100)
		unsub, err := node.Kernel.SubscribeRaw(ctx, replyTo, func(msg messages.Message) {
			select {
			case eventCh <- msg:
			default:
			}
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "subscribe: " + err.Error()})
			return
		}
		defer unsub()

		pubCtx := messaging.WithPublishMeta(ctx, correlationID, replyTo)
		if _, err := node.Kernel.PublishRaw(pubCtx, req.Topic, req.Payload); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "publish: " + err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		enc := json.NewEncoder(w)
		for {
			select {
			case msg := <-eventCh:
				done := msg.Metadata != nil && msg.Metadata["done"] == "true"
				evt := map[string]any{
					"payload": json.RawMessage(msg.Payload),
					"done":    done,
				}
				enc.Encode(evt)
				flusher.Flush()
				if done {
					return
				}
			case <-ctx.Done():
				enc.Encode(map[string]string{"error": "timeout"})
				flusher.Flush()
				return
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
