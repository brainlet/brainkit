package schedules

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/google/uuid"
)

// Scheduler owns the live schedule set. It implements types.ScheduleHandler
// so the kernel's QuickJS bridges (bus.schedule / bus.unschedule) dispatch
// through it. It publishes fires via the runtime's PublishRaw.
type Scheduler struct {
	runtime sdk.Runtime
	store   Store
	logger  *slog.Logger

	// Topic-catalog guard — schedules cannot target command topics.
	isCommand func(topic string) bool
	// IsDraining is checked before each fire; schedules go silent while the
	// kit is draining (matches the pre-module behavior).
	isDraining func() bool

	// ReportError fans a persistence failure to the Kit's configured
	// ErrorHandler. Matches the pre-module behavior.
	reportError func(err error)

	mu        sync.Mutex
	schedules map[string]*entry
}

type entry struct {
	types.PersistedSchedule
	timer *time.Timer
}

// newScheduler constructs a Scheduler wired to a Kit. Exported types flow in
// via the module so this file stays free of brainkit imports (no cycle).
func newScheduler(runtime sdk.Runtime, store Store, logger *slog.Logger,
	isCommand func(string) bool, isDraining func() bool, reportError func(error)) *Scheduler {
	return &Scheduler{
		runtime:     runtime,
		store:       store,
		logger:      logger,
		isCommand:   isCommand,
		isDraining:  isDraining,
		reportError: reportError,
		schedules:   make(map[string]*entry),
	}
}

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

// Schedule creates a new scheduled bus message. Implements types.ScheduleHandler.
func (s *Scheduler) Schedule(ctx context.Context, cfg types.ScheduleConfig) (string, error) {
	if s.isCommand != nil && s.isCommand(cfg.Topic) {
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
	s.add(ps)
	if s.store != nil {
		if err := s.store.SaveSchedule(ps); err != nil {
			s.persistenceError(ctx, "SaveSchedule", id, err)
		}
	}
	return id, nil
}

// Unschedule cancels and removes a schedule. Implements types.ScheduleHandler.
func (s *Scheduler) Unschedule(ctx context.Context, id string) error {
	s.remove(id)
	return nil
}

// List returns all active schedules. Implements types.ScheduleHandler.
func (s *Scheduler) List() []types.PersistedSchedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]types.PersistedSchedule, 0, len(s.schedules))
	for _, e := range s.schedules {
		result = append(result, e.PersistedSchedule)
	}
	return result
}

// Close stops all schedule timers. Safe to call more than once.
func (s *Scheduler) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.schedules {
		e.timer.Stop()
	}
	s.schedules = map[string]*entry{}
	return nil
}

// Restore replays persisted schedules. Called by the module during Init
// when a Store is configured. Catches up missed fires.
func (s *Scheduler) Restore() {
	if s.store == nil {
		return
	}
	loaded, err := s.store.LoadSchedules()
	if err != nil {
		if s.reportError != nil {
			s.reportError(&sdkerrors.PersistenceError{Operation: "LoadSchedules", Cause: err})
		}
		return
	}
	now := time.Now()
	restored := 0
	for _, ps := range loaded {
		if ps.NextFire.Before(now) {
			if err := s.publish(context.Background(), ps.Topic, ps.Payload); err != nil && s.reportError != nil {
				s.reportError(&sdkerrors.PersistenceError{Operation: "RestoreSchedule.CatchUp", Source: ps.ID, Cause: err})
			}
			if ps.OneTime {
				_ = s.store.DeleteSchedule(ps.ID)
				continue
			}
			ps.NextFire = now.Add(ps.Duration)
			_ = s.store.SaveSchedule(ps)
		}
		s.add(ps)
		restored++
	}
	if restored > 0 && s.logger != nil {
		s.logger.Info("restored persisted schedules", slog.Int("count", restored))
	}
}

func (s *Scheduler) add(ps types.PersistedSchedule) {
	delay := time.Until(ps.NextFire)
	if delay < 0 {
		delay = 0
	}
	e := &entry{PersistedSchedule: ps}
	e.timer = time.AfterFunc(delay, func() { s.fire(e) })
	s.mu.Lock()
	s.schedules[ps.ID] = e
	s.mu.Unlock()
}

func (s *Scheduler) remove(id string) {
	s.mu.Lock()
	e, ok := s.schedules[id]
	if ok {
		e.timer.Stop()
		delete(s.schedules, id)
	}
	s.mu.Unlock()
	if ok && s.store != nil {
		_ = s.store.DeleteSchedule(id)
	}
}

func (s *Scheduler) fire(e *entry) {
	if s.isDraining != nil && s.isDraining() {
		return
	}
	// Multi-replica dedup: only one claimant per fire.
	if s.store != nil {
		claimed, err := s.store.ClaimScheduleFire(e.ID, time.Now())
		if err == nil && !claimed {
			return
		}
	}

	_ = s.publish(context.Background(), e.Topic, e.Payload)

	if e.OneTime {
		s.mu.Lock()
		delete(s.schedules, e.ID)
		s.mu.Unlock()
		if s.store != nil {
			_ = s.store.DeleteSchedule(e.ID)
		}
		return
	}

	// NextFire is observed by List() under s.mu — mutate it inside
	// the same lock so -race doesn't see a torn read on the
	// re-arm path.
	s.mu.Lock()
	e.NextFire = time.Now().Add(e.Duration)
	snapshot := e.PersistedSchedule
	s.mu.Unlock()

	e.timer.Reset(e.Duration)
	if s.store != nil {
		if err := s.store.SaveSchedule(snapshot); err != nil {
			s.persistenceError(context.Background(), "SaveSchedule", e.ID, err)
		}
	}
}

func (s *Scheduler) publish(ctx context.Context, topic string, payload json.RawMessage) error {
	_, err := s.runtime.PublishRaw(ctx, topic, payload)
	return err
}

func (s *Scheduler) persistenceError(ctx context.Context, operation, source string, err error) {
	if s.reportError != nil {
		s.reportError(&sdkerrors.PersistenceError{Operation: operation, Source: source, Cause: err})
	}
	// Best-effort bus event for observers. Matches the pre-module behavior.
	payload, _ := json.Marshal(map[string]any{
		"operation": operation,
		"source":    source,
		"error":     err.Error(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
	_, _ = s.runtime.PublishRaw(ctx, "kit.persistence.error", payload)
}
