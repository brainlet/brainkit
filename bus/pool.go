package bus

// PoolConfig describes a pool of Kit instances.
type PoolConfig struct {
	Name     string // "order-processors"
	Template any    // base Kit config (typed by brainkit, opaque here)
	Min      int    // minimum instances
	Max      int    // maximum instances
}

// PoolInfo describes an active pool.
type PoolInfo struct {
	Name      string `json:"name"`
	Instances int    `json:"instances"`
	Min       int    `json:"min"`
	Max       int    `json:"max"`
}

// InstanceManager manages pools of Kit instances.
// Implementation is future — this defines the interface for scaling.
type InstanceManager interface {
	SpawnPool(cfg PoolConfig) error
	Scale(poolName string, count int) error
	KillPool(poolName string) error
	Pools() []PoolInfo
}
