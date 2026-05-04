package schedules

import (
	"time"

	"github.com/brainlet/brainkit/internal/types"
)

// ScheduleConfig configures a scheduled bus message.
type ScheduleConfig = types.ScheduleConfig

// Store is the narrow persistence surface the module needs. brainkit's
// KitStore (returned by stores/sqlite.New) satisfies it structurally,
// so the common case is `Config{Store: kitStore}`.
type Store interface {
	SaveSchedule(s types.PersistedSchedule) error
	LoadSchedules() ([]types.PersistedSchedule, error)
	DeleteSchedule(id string) error
	// ClaimScheduleFire atomically claims a schedule fire across replicas.
	// Returns true if this replica claimed it, false if another already did.
	ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error)
}

// Config configures the schedules module.
type Config struct {
	// Store is optional. When nil: schedules are in-memory only and do not
	// survive restart. When provided: schedules are persisted and restored on
	// module Mount, and ClaimScheduleFire is used for multi-replica dedup.
	Store Store

	// OwnStore tells the module to close Store during unmount. Leave false
	// when the store is borrowed from the Kit or another owner.
	OwnStore bool
}
