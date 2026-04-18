package audit

import (
	"context"
	"time"

	"github.com/brainlet/brainkit/sdk"
)

// domain wraps the module's Store for the three bus commands. Each
// call is nil-safe on a missing store so that a module created with
// no Store still answers queries with empty results.
type domain struct {
	store Store
}

func newDomain(store Store) *domain { return &domain{store: store} }

func (d *domain) Query(_ context.Context, req sdk.AuditQueryMsg) (*sdk.AuditQueryResp, error) {
	if d.store == nil {
		return &sdk.AuditQueryResp{Events: []sdk.AuditEvent{}}, nil
	}
	events, err := d.store.Query(Query{
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

func (d *domain) Stats(_ context.Context, _ sdk.AuditStatsMsg) (*sdk.AuditStatsResp, error) {
	if d.store == nil {
		return &sdk.AuditStatsResp{EventsByCategory: map[string]int64{}}, nil
	}
	total, _ := d.store.Count()
	byCat, _ := d.store.CountByCategory()
	return &sdk.AuditStatsResp{TotalEvents: total, EventsByCategory: byCat}, nil
}

func (d *domain) Prune(_ context.Context, req sdk.AuditPruneMsg) (*sdk.AuditPruneResp, error) {
	if d.store == nil {
		return &sdk.AuditPruneResp{Pruned: false}, nil
	}
	hours := req.OlderThanHours
	if hours <= 0 {
		hours = 24 * 7 // default: prune events older than 1 week
	}
	if err := d.store.Prune(time.Duration(hours) * time.Hour); err != nil {
		return nil, err
	}
	return &sdk.AuditPruneResp{Pruned: true}, nil
}
