package main

import (
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	p := sdk.New("brainlet", "cron", "1.0.0",
		sdk.WithDescription("Cron scheduling plugin for brainkit"),
	)

	sdk.Tool(p, "create", "Create a cron job", handleCreate)
	sdk.Tool(p, "list", "List all cron jobs", handleList)
	sdk.Tool(p, "remove", "Remove a cron job", handleRemove)
	sdk.Tool(p, "pause", "Pause a cron job", handlePause)
	sdk.Tool(p, "resume", "Resume a paused cron job", handleResume)

	sdk.Event[CronFiredEvent](p, "Emitted when a cron job fires")

	p.OnStart(onStart)
	p.OnStop(onStop)

	p.Run()
}
