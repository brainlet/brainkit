package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show instance health status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.KitHealthMsg{},
			func(resp *messages.KitHealthResp) {
				var health struct {
					Healthy bool   `json:"healthy"`
					Status  string `json:"status"`
					Uptime  int64  `json:"uptime"`
					Checks  []struct {
						Name    string `json:"name"`
						Healthy bool   `json:"healthy"`
						Latency int64  `json:"latency,omitempty"`
						Error   string `json:"error,omitempty"`
					} `json:"checks"`
				}
				json.Unmarshal(resp.Health, &health)

				uptime := time.Duration(health.Uptime)
				fmt.Printf("Status: %s\n", health.Status)
				fmt.Printf("Uptime: %s\n", uptime.Round(time.Second))
				for _, c := range health.Checks {
					if c.Healthy {
						latency := ""
						if c.Latency > 0 {
							latency = fmt.Sprintf(" (%s)", time.Duration(c.Latency).Round(time.Millisecond))
						}
						fmt.Printf("  %s: healthy%s\n", c.Name, latency)
					} else {
						fmt.Printf("  %s: unhealthy (%s)\n", c.Name, c.Error)
					}
				}
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
