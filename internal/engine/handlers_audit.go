package engine

import (
	"context"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/sdk"
)

// AuditDomain handles audit log queries.
type AuditDomain struct {
	store auditpkg.Store
}

func newAuditDomain(store auditpkg.Store) *AuditDomain {
	if store == nil {
		return &AuditDomain{} // nil-safe — returns empty results
	}
	return &AuditDomain{store: store}
}

// Query returns audit events matching the filter.
func (d *AuditDomain) Query(_ context.Context, req sdk.AuditQueryMsg) (*sdk.AuditQueryResp, error) {
	if d.store == nil {
		return &sdk.AuditQueryResp{Events: []sdk.AuditEvent{}}, nil
	}

	events, err := d.store.Query(auditpkg.Query{
		Category:  req.Category,
		Type:      req.Type,
		Source:    req.Source,
		Since:     req.Since,
		Until:     req.Until,
		Limit:     req.Limit,
		RuntimeID: req.RuntimeID,
	})
	if err != nil {
		return nil, err
	}

	total, _ := d.store.Count()

	sdkEvents := make([]sdk.AuditEvent, len(events))
	for i, e := range events {
		sdkEvents[i] = sdk.AuditEvent{
			ID:        e.ID,
			Timestamp: e.Timestamp,
			Category:  e.Category,
			Type:      e.Type,
			Source:    e.Source,
			RuntimeID: e.RuntimeID,
			Namespace: e.Namespace,
			Data:      e.Data,
			Duration:  e.Duration,
			Error:     e.Error,
		}
	}

	return &sdk.AuditQueryResp{Events: sdkEvents, Total: total}, nil
}

// Stats returns audit store statistics.
func (d *AuditDomain) Stats(_ context.Context, _ sdk.AuditStatsMsg) (*sdk.AuditStatsResp, error) {
	if d.store == nil {
		return &sdk.AuditStatsResp{EventsByCategory: map[string]int64{}}, nil
	}

	total, _ := d.store.Count()
	byCat, _ := d.store.CountByCategory()

	return &sdk.AuditStatsResp{TotalEvents: total, EventsByCategory: byCat}, nil
}

// Prune deletes old audit events.
func (d *AuditDomain) Prune(_ context.Context, req sdk.AuditPruneMsg) (*sdk.AuditPruneResp, error) {
	if d.store == nil {
		return &sdk.AuditPruneResp{Pruned: false}, nil
	}
	hours := req.OlderThanHours
	if hours <= 0 {
		hours = 24 * 7 // default: prune events older than 1 week
	}
	err := d.store.Prune(time.Duration(hours) * time.Hour)
	if err != nil {
		return nil, err
	}
	return &sdk.AuditPruneResp{Pruned: true}, nil
}

// auditStoreFromKernel extracts the audit store from kernel state.
// Returns nil if no audit store is configured.
func auditStoreFromKernel(k *Kernel) auditpkg.Store {
	return k.auditStore
}
