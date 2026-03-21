package bus

// BusMetrics is the top-level metrics snapshot.
type BusMetrics struct {
	Transport   TransportMetrics `json:"transport"`
	ActiveJobs  int              `json:"activeJobs"`
	TotalJobs   int              `json:"totalJobs"`
	Subscribers int              `json:"subscribers"`
}

// TransportMetrics are reported by the transport layer.
type TransportMetrics struct {
	Topics  map[string]TopicMetrics       `json:"topics"`
	Workers map[string]WorkerGroupMetrics `json:"workers"`
}

// TopicMetrics tracks per-topic stats.
type TopicMetrics struct {
	Pending int     `json:"pending"`
	Rate    float64 `json:"rate"` // messages/sec
}

// WorkerGroupMetrics tracks per-worker-group stats.
type WorkerGroupMetrics struct {
	Name       string  `json:"name"`
	Members    int     `json:"members"`
	Pending    int     `json:"pending"`
	Throughput float64 `json:"throughput"`
}
