package brainkit_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	metricsmod "github.com/brainlet/brainkit/modules/metrics"
	"github.com/stretchr/testify/require"
)

type metricsProbeModule struct{}

func (m *metricsProbeModule) ID() string { return "metrics-probe" }

func (m *metricsProbeModule) Mount(_ context.Context, host bkmodule.Host) error {
	_, err := host.Commands().Handle(bkmodule.Command(m.Ping))
	return err
}

func (m *metricsProbeModule) Ping(context.Context, metricsPingMsg) (*metricsPingResp, error) {
	return &metricsPingResp{OK: true}, nil
}

type metricsPingMsg struct{}

func (metricsPingMsg) BusTopic() string { return "metrics.ping" }

type metricsPingResp struct {
	OK bool `json:"ok"`
}

func TestMetricsGetIncludesStableBusAndTransportObservability(t *testing.T) {
	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "metrics-observability-test",
		CallerID:  "metrics-observability-test",
		FSRoot:    t.TempDir(),
		Modules: []bkmodule.Module{
			metricsmod.New(),
			&metricsProbeModule{},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, kit.Close()) })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ping, err := brainkit.Call[metricsPingMsg, metricsPingResp](kit, ctx, metricsPingMsg{})
	require.NoError(t, err)
	require.True(t, ping.OK)

	var snapshot kernelMetricsView
	var metricsJSON json.RawMessage
	require.Eventually(t, func() bool {
		resp, err := metricsmod.CallMetricsGet(kit, ctx, metricsmod.MetricsGetMsg{})
		if err != nil {
			return false
		}
		metricsJSON = append(metricsJSON[:0], resp.Metrics...)
		snapshot = kernelMetricsView{}
		if err := json.Unmarshal(resp.Metrics, &snapshot); err != nil {
			return false
		}
		return snapshot.Bus != nil &&
			snapshot.Bus.Published["metrics.ping"] >= 1 &&
			snapshot.Bus.Handled["metrics.ping"] >= 1 &&
			snapshot.Bus.HandleDurationCount["metrics.ping"] >= 1
	}, time.Second, 10*time.Millisecond)

	require.NotNil(t, snapshot.Transport)
	require.Equal(t, "memory", snapshot.Transport.Kind)
	require.Greater(t, snapshot.Transport.RouterHandlers, 0)
	require.Greater(t, snapshot.Transport.RouterStartedHandlers, 0)
	require.GreaterOrEqual(t, snapshot.Transport.ActiveSubscriptions, int64(0))
	require.GreaterOrEqual(t, snapshot.Transport.ActiveStreamHeartbeats, 0)

	require.NotNil(t, snapshot.Bus)
	require.GreaterOrEqual(t, snapshot.Bus.Published["metrics.get"], 1)
	require.GreaterOrEqual(t, snapshot.Bus.HandleDurationTotal["metrics.ping"], time.Duration(0))
	require.GreaterOrEqual(t, snapshot.Bus.HandleDurationMax["metrics.ping"], time.Duration(0))

	var raw struct {
		Transport map[string]json.RawMessage `json:"transport"`
	}
	require.NoError(t, json.Unmarshal(metricsJSON, &raw))
	require.NotContains(t, raw.Transport, "ownsTransport")
	require.NotContains(t, raw.Transport, "closingRouter")
	require.NotContains(t, raw.Transport, "closingCaller")
	require.NotContains(t, raw.Transport, "closingTransport")
	require.NotContains(t, raw.Transport, "closedRouter")
	require.NotContains(t, raw.Transport, "closedCaller")
	require.NotContains(t, raw.Transport, "closedTransport")
}

type kernelMetricsView struct {
	Bus       *busMetricsView       `json:"bus"`
	Transport *transportMetricsView `json:"transport"`
}

type busMetricsView struct {
	Published           map[string]int           `json:"published"`
	Handled             map[string]int           `json:"handled"`
	Errors              map[string]int           `json:"errors"`
	HandleDurationTotal map[string]time.Duration `json:"handleDurationTotal"`
	HandleDurationMax   map[string]time.Duration `json:"handleDurationMax"`
	HandleDurationCount map[string]int           `json:"handleDurationCount"`
}

type transportMetricsView struct {
	Kind                   string `json:"kind"`
	ActiveSubscriptions    int64  `json:"activeSubscriptions"`
	RouterHandlers         int    `json:"routerHandlers"`
	RouterStartedHandlers  int    `json:"routerStartedHandlers"`
	RouterStoppedHandlers  int    `json:"routerStoppedHandlers"`
	ActiveStreamHeartbeats int    `json:"activeStreamHeartbeats"`
}
