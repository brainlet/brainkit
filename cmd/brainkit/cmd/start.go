package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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
			controlSrv := &http.Server{Handler: mux}
			go controlSrv.Serve(ln)

			// Write pidfile with control port so CLI commands can discover it
			pidDir := "data"
			os.MkdirAll(pidDir, 0755)
			pidFile := filepath.Join(pidDir, "brainkit.pid")
			os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", port)), 0644)
			defer os.Remove(pidFile)

			cmd.Println("brainkit started")
			cmd.Printf("  namespace:  %s\n", node.Kernel.Namespace())
			cmd.Printf("  transport:  %s\n", cfg.Transport)
			cmd.Printf("  control:    http://127.0.0.1:%d\n", port)
			if cfg.FSRoot != "" {
				cmd.Printf("  workspace:  %s\n", cfg.FSRoot)
			}
			cmd.Println("\nPress Ctrl+C to stop.")

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			cmd.Println("\nShutting down...")
			controlSrv.Shutdown(context.Background())
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			return node.Shutdown(ctx)
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
