// Ported from: packages/core/src/vector/vector.ts
package vector

import (
	"context"
	"fmt"
	"strings"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/vector/filter"
)

// EmbeddingModel is a stub interface for AI SDK embedding models.
// TODO: Replace with a proper embedding model interface when the AI SDK
// integration layer is ported. In TypeScript this covers EmbeddingModelV1,
// EmbeddingModelV2, and EmbeddingModelV3 from @internal/ai-sdk-v4, v5, and ai-v6.
type EmbeddingModel interface{}

// EmbeddingOptions holds configuration for embedding operations.
// Ported from MastraEmbeddingOptions in vector.ts.
type EmbeddingOptions struct {
	// MaxRetries is the maximum number of retries for embedding requests.
	MaxRetries int `json:"maxRetries,omitempty"`
	// Headers contains optional HTTP headers for embedding requests.
	Headers map[string]string `json:"headers,omitempty"`
	// MaxParallelCalls is the maximum number of parallel embedding calls.
	MaxParallelCalls int `json:"maxParallelCalls,omitempty"`
	// TODO: TelemetrySettings and ProviderOptions are omitted until AI SDK types are ported.
}

// SupportedEmbeddingModelSpecifications lists the spec versions for modern embedding models.
// Ported from: supportedEmbeddingModelSpecifications = ['v2', 'v3'] as const.
var SupportedEmbeddingModelSpecifications = []string{"v2", "v3"}

// MastraVector defines the interface for all vector store implementations.
// Ported from the abstract class MastraVector<Filter> in vector.ts.
//
// The Filter type parameter from TypeScript is not replicated here;
// all filter parameters use filter.VectorFilter (map[string]any).
type MastraVector interface {
	// ID returns the unique identifier for this vector store instance.
	ID() string

	// IndexSeparator returns the separator used in index names. Default is "_".
	IndexSeparator() string

	// Query performs a similarity search against the named index.
	Query(ctx context.Context, params QueryVectorParams) ([]QueryResult, error)

	// Upsert inserts or updates vectors in the named index.
	// Returns the IDs of the upserted vectors.
	Upsert(ctx context.Context, params UpsertVectorParams) ([]string, error)

	// CreateIndex creates a new vector index with the specified parameters.
	CreateIndex(ctx context.Context, params CreateIndexParams) error

	// ListIndexes returns the names of all existing indexes.
	ListIndexes(ctx context.Context) ([]string, error)

	// DescribeIndex returns statistics about the named index.
	DescribeIndex(ctx context.Context, params DescribeIndexParams) (IndexStats, error)

	// DeleteIndex deletes the named index.
	DeleteIndex(ctx context.Context, params DeleteIndexParams) error

	// UpdateVector updates a single vector by ID or multiple vectors by filter.
	UpdateVector(ctx context.Context, params UpdateVectorParams) error

	// DeleteVector deletes a single vector by ID.
	DeleteVector(ctx context.Context, params DeleteVectorParams) error

	// DeleteVectors deletes multiple vectors by IDs or metadata filter.
	// Implementations should return an error if the operation is not supported.
	DeleteVectors(ctx context.Context, params DeleteVectorsParams) error
}

// Ensure filter package is used (it is referenced by types.go, but this
// prevents a "declared and not used" error if types.go is built independently).
var _ filter.VectorFilter

// MastraVectorBase provides shared functionality for vector store implementations.
// It embeds MastraBase and provides the ID, IndexSeparator, and ValidateExistingIndex
// helper that was part of the abstract TypeScript class.
//
// Concrete implementations should embed this struct and implement the MastraVector interface.
type MastraVectorBase struct {
	*agentkit.MastraBase
	id string
}

// MastraVectorBaseConfig holds the configuration for creating a MastraVectorBase.
type MastraVectorBaseConfig struct {
	ID string
}

// NewMastraVectorBase creates a new MastraVectorBase.
// Returns an error if id is empty.
func NewMastraVectorBase(cfg MastraVectorBaseConfig) (*MastraVectorBase, error) {
	id := strings.TrimSpace(cfg.ID)
	if id == "" {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VECTOR_INVALID_ID",
			Text:     "Vector id must be provided and cannot be empty",
			Domain:   mastraerror.ErrorDomainMastraVector,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	base := agentkit.NewMastraBase(agentkit.MastraBaseOptions{
		Name:      "MastraVector",
		Component: logger.RegisteredLoggerVector,
	})

	return &MastraVectorBase{
		MastraBase: base,
		id:         id,
	}, nil
}

// ID returns the unique identifier for this vector store instance.
func (v *MastraVectorBase) ID() string {
	return v.id
}

// IndexSeparator returns the separator used in index names. Default is "_".
func (v *MastraVectorBase) IndexSeparator() string {
	return "_"
}

// ValidateExistingIndex checks an existing index against the expected dimension and metric.
// It is a protected helper (called by concrete implementations during CreateIndex).
//
// describeIndex is a function that describes the index — typically the implementation's
// own DescribeIndex method, passed to avoid circular interface references.
func (v *MastraVectorBase) ValidateExistingIndex(
	ctx context.Context,
	indexName string,
	dimension int,
	metric string,
	describeIndex func(ctx context.Context, params DescribeIndexParams) (IndexStats, error),
) error {
	info, err := describeIndex(ctx, DescribeIndexParams{IndexName: indexName})
	if err != nil {
		mErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "VECTOR_VALIDATE_INDEX_FETCH_FAILED",
			Text:     fmt.Sprintf("Index %q already exists, but failed to fetch index info for dimension check.", indexName),
			Domain:   mastraerror.ErrorDomainMastraVector,
			Category: mastraerror.ErrorCategorySystem,
			Details:  map[string]any{"indexName": indexName},
		}, err)
		l := v.Logger()
		if l != nil {
			l.Error(mErr.Error())
		}
		return mErr
	}

	existingDim := info.Dimension
	existingMetric := info.Metric

	l := v.Logger()

	if existingDim == dimension {
		if l != nil {
			l.Info(fmt.Sprintf(
				"Index %q already exists with %d dimensions and metric %s, skipping creation.",
				indexName, existingDim, existingMetric,
			))
		}
		if string(existingMetric) != metric {
			if l != nil {
				l.Warn(fmt.Sprintf(
					"Attempted to create index with metric %q, but index already exists with metric %q. To use a different metric, delete and recreate the index.",
					metric, existingMetric,
				))
			}
		}
		return nil
	}

	if existingDim != 0 {
		mErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:   "VECTOR_VALIDATE_INDEX_DIMENSION_MISMATCH",
			Text: fmt.Sprintf("Index %q already exists with %d dimensions, but %d dimensions were requested", indexName, existingDim, dimension),
			Domain:   mastraerror.ErrorDomainMastraVector,
			Category: mastraerror.ErrorCategoryUser,
			Details: map[string]any{
				"indexName":    indexName,
				"existingDim":  existingDim,
				"requestedDim": dimension,
			},
		})
		if l != nil {
			l.Error(mErr.Error())
		}
		return mErr
	}

	mErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "VECTOR_VALIDATE_INDEX_NO_DIMENSION",
		Text:     fmt.Sprintf("Index %q already exists, but could not retrieve its dimensions for validation.", indexName),
		Domain:   mastraerror.ErrorDomainMastraVector,
		Category: mastraerror.ErrorCategorySystem,
		Details:  map[string]any{"indexName": indexName},
	})
	if l != nil {
		l.Error(mErr.Error())
	}
	return mErr
}
