package engine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/google/uuid"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// --- Scheduling ---

func parseScheduleExpression(expr string) (time.Duration, bool, error) {
	if strings.HasPrefix(expr, "every ") {
		d, err := time.ParseDuration(strings.TrimPrefix(expr, "every "))
		return d, false, err
	}
	if strings.HasPrefix(expr, "in ") {
		d, err := time.ParseDuration(strings.TrimPrefix(expr, "in "))
		return d, true, err
	}
	return 0, false, fmt.Errorf("unsupported schedule expression: %q (use 'every <duration>' or 'in <duration>')", expr)
}

func (k *Kernel) addSchedule(ps types.PersistedSchedule) {
	delay := time.Until(ps.NextFire)
	if delay < 0 {
		delay = 0
	}
	entry := &scheduleEntry{PersistedSchedule: ps}
	entry.timer = time.AfterFunc(delay, func() {
		k.fireSchedule(entry)
	})
	k.mu.Lock()
	k.schedules[ps.ID] = entry
	k.mu.Unlock()
}

func (k *Kernel) fireSchedule(entry *scheduleEntry) {
	if k.IsDraining() {
		return
	}

	// Schedule deduplication: if shared store exists, try to claim this fire.
	// Only one replica wins — others skip. On error, fire anyway (single-replica safety).
	if k.config.Store != nil {
		claimed, err := k.config.Store.ClaimScheduleFire(entry.ID, time.Now())
		if err == nil && !claimed {
			return
		}
	}

	k.publish(context.Background(), entry.Topic, entry.Payload)

	if entry.OneTime {
		k.mu.Lock()
		delete(k.schedules, entry.ID)
		k.mu.Unlock()
		if k.config.Store != nil {
			k.config.Store.DeleteSchedule(entry.ID)
		}
		return
	}

	entry.NextFire = time.Now().Add(entry.Duration)
	entry.timer.Reset(entry.Duration)
	if k.config.Store != nil {
		if err := k.config.Store.SaveSchedule(entry.PersistedSchedule); err != nil {
			k.persistenceError(context.Background(), "SaveSchedule", entry.ID, err)
		}
	}
}

func (k *Kernel) removeSchedule(id string) {
	k.mu.Lock()
	entry, ok := k.schedules[id]
	if ok {
		entry.timer.Stop()
		delete(k.schedules, id)
	}
	k.mu.Unlock()
	if ok && k.config.Store != nil {
		k.config.Store.DeleteSchedule(id)
	}
}

// Schedule creates a new scheduled bus message.
func (k *Kernel) Schedule(ctx context.Context, cfg types.ScheduleConfig) (string, error) {
	// Block scheduling to command topics — scheduled messages fire from Go
	// with no RBAC context, so they'd bypass all permission checks.
	if commandCatalog().HasCommand(cfg.Topic) {
		return "", &sdkerrors.ValidationError{Field: "topic", Message: cfg.Topic + " is a command topic; schedules cannot target commands"}
	}

	duration, oneTime, err := parseScheduleExpression(cfg.Expression)
	if err != nil {
		return "", err
	}
	id := cfg.ID
	if id == "" {
		id = uuid.NewString()
	}
	ps := types.PersistedSchedule{
		ID:         id,
		Expression: cfg.Expression,
		Duration:   duration,
		Topic:      cfg.Topic,
		Payload:    cfg.Payload,
		Source:     cfg.Source,
		CreatedAt:  time.Now(),
		NextFire:   time.Now().Add(duration),
		OneTime:    oneTime,
	}
	k.addSchedule(ps)
	if k.config.Store != nil {
		if err := k.config.Store.SaveSchedule(ps); err != nil {
			k.persistenceError(ctx, "SaveSchedule", id, err)
		}
	}
	return id, nil
}

// Unschedule cancels and removes a schedule.
func (k *Kernel) Unschedule(ctx context.Context, id string) error {
	k.removeSchedule(id)
	return nil
}

// ListSchedules returns all active schedules.
func (k *Kernel) ListSchedules() []types.PersistedSchedule {
	k.mu.Lock()
	defer k.mu.Unlock()
	result := make([]types.PersistedSchedule, 0, len(k.schedules))
	for _, entry := range k.schedules {
		result = append(result, entry.PersistedSchedule)
	}
	return result
}

func (k *Kernel) restoreSchedules() {
	schedules, err := k.config.Store.LoadSchedules()
	if err != nil {
		types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "LoadSchedules", Cause: err,
		}, types.ErrorContext{Operation: "LoadSchedules", Component: "kernel"})
		return
	}
	now := time.Now()
	restored := 0
	for _, s := range schedules {
		if s.OneTime {
			if s.NextFire.Before(now) {
				if err := k.publish(context.Background(), s.Topic, s.Payload); err != nil {
					types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
						Operation: "RestoreSchedule.CatchUp", Source: s.ID, Cause: err,
					}, types.ErrorContext{Operation: "RestoreSchedule", Component: "kernel", Source: s.ID})
				}
				k.config.Store.DeleteSchedule(s.ID)
				continue
			}
		} else {
			if s.NextFire.Before(now) {
				if err := k.publish(context.Background(), s.Topic, s.Payload); err != nil {
					types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
						Operation: "RestoreSchedule.CatchUp", Source: s.ID, Cause: err,
					}, types.ErrorContext{Operation: "RestoreSchedule", Component: "kernel", Source: s.ID})
				}
				s.NextFire = now.Add(s.Duration)
				k.config.Store.SaveSchedule(s)
			}
		}
		k.addSchedule(s)
		restored++
	}
	if restored > 0 {
		k.logger.Info("restored persisted schedules", slog.Int("count", restored))
	}
}
