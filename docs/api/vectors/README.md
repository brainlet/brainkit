# Vectors API Reference

Vector index management, upsert, and similarity search.

---

## Bus Topics

All vector operations use Ask (request/response) over the bus.

### vectors.createIndex

Create a new vector index.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"name": string, "dimension": int, "metric": string}` |
| **Response** | *(ack)* |

`metric` is the distance metric (e.g. `"cosine"`, `"euclidean"`, `"dotProduct"`).

### vectors.deleteIndex

Delete a vector index by name.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"name": string}` |
| **Response** | *(ack)* |

### vectors.listIndexes

List all vector indexes.

| Direction | Shape |
|-----------|-------|
| **Request** | `{}` |
| **Response** | `VectorIndexInfo[]` |

### vectors.upsert

Insert or update vectors in an index.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"index": string, "vectors": Vector[]}` |
| **Response** | *(ack)* |

### vectors.query

Query an index for nearest neighbors.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"index": string, "embedding": float64[], "topK": int, "filter": any}` |
| **Response** | `{"matches": VectorMatch[]}` |

`filter` is optional. Format is index-implementation-specific.

---

## SDK Messages

Typed messages for bus interactions. All implement `BusMessage`.

```go
import "github.com/brainlet/brainkit/sdk/messages"
```

### Request Messages

| Message | Fields | BusTopic() |
|---------|--------|------------|
| `VectorCreateIndexMsg` | `Name string`, `Dimension int`, `Metric string` | `"vectors.createIndex"` |
| `VectorDeleteIndexMsg` | `Name string` | `"vectors.deleteIndex"` |
| `VectorListIndexesMsg` | *(none)* | `"vectors.listIndexes"` |
| `VectorUpsertMsg` | `Index string`, `Vectors []Vector` | `"vectors.upsert"` |
| `VectorQueryMsg` | `Index string`, `Embedding []float64`, `TopK int`, `Filter any` | `"vectors.query"` |

### Response Messages

| Message | Fields |
|---------|--------|
| `VectorQueryResp` | `Matches []VectorMatch` |

---

## Types

### Vector

```go
type Vector struct {
    ID       string            `json:"id"`
    Values   []float64         `json:"values"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique vector identifier |
| `Values` | `[]float64` | Embedding values |
| `Metadata` | `map[string]string` | Optional key-value metadata |

### VectorMatch

```go
type VectorMatch struct {
    ID       string            `json:"id"`
    Score    float64           `json:"score"`
    Values   []float64         `json:"values,omitempty"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Matched vector ID |
| `Score` | `float64` | Similarity score |
| `Values` | `[]float64` | Embedding values (included when available) |
| `Metadata` | `map[string]string` | Vector metadata (included when available) |

### VectorIndexInfo

```go
type VectorIndexInfo struct {
    Name      string `json:"name"`
    Dimension int    `json:"dimension"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Index name |
| `Dimension` | `int` | Vector dimension |
