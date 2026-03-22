package messages

// ── Requests ──

type VectorUpsertMsg struct {
	Index   string   `json:"index"`
	Vectors []Vector `json:"vectors"`
}

func (VectorUpsertMsg) BusTopic() string { return "vectors.upsert" }

type VectorQueryMsg struct {
	Index     string    `json:"index"`
	Embedding []float64 `json:"embedding"`
	TopK      int       `json:"topK"`
	Filter    any       `json:"filter,omitempty"`
}

func (VectorQueryMsg) BusTopic() string { return "vectors.query" }

type VectorCreateIndexMsg struct {
	Name      string `json:"name"`
	Dimension int    `json:"dimension"`
	Metric    string `json:"metric"`
}

func (VectorCreateIndexMsg) BusTopic() string { return "vectors.createIndex" }

type VectorDeleteIndexMsg struct {
	Name string `json:"name"`
}

func (VectorDeleteIndexMsg) BusTopic() string { return "vectors.deleteIndex" }

type VectorListIndexesMsg struct{}

func (VectorListIndexesMsg) BusTopic() string { return "vectors.listIndexes" }

// ── Responses ──

type VectorUpsertResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (VectorUpsertResp) BusTopic() string { return "vectors.upsert.result" }

type VectorQueryResp struct {
	ResultMeta
	Matches []VectorMatch `json:"matches"`
}

func (VectorQueryResp) BusTopic() string { return "vectors.query.result" }

type VectorCreateIndexResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (VectorCreateIndexResp) BusTopic() string { return "vectors.createIndex.result" }

type VectorDeleteIndexResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (VectorDeleteIndexResp) BusTopic() string { return "vectors.deleteIndex.result" }

type VectorListIndexesResp struct {
	ResultMeta
	Indexes []VectorIndexInfo `json:"indexes"`
}

func (VectorListIndexesResp) BusTopic() string { return "vectors.listIndexes.result" }

// ── Shared types ──

type Vector struct {
	ID       string            `json:"id"`
	Values   []float64         `json:"values"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type VectorMatch struct {
	ID       string            `json:"id"`
	Score    float64           `json:"score"`
	Values   []float64         `json:"values,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type VectorIndexInfo struct {
	Name      string `json:"name"`
	Dimension int    `json:"dimension"`
}
