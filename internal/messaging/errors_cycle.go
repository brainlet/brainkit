package messaging

import "fmt"

// ErrCycleDetected is returned when a message exceeds the maximum cascade depth.
var ErrCycleDetected = fmt.Errorf("messaging: cycle detected (maximum depth exceeded)")
