package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderModulesShowsCapabilityDirectionCounts(t *testing.T) {
	payload := json.RawMessage(`{
		"modules": [
			{
				"name": "grouped",
				"status": "stable",
				"provides": ["runtime"],
				"requires": ["jsruntime"],
				"commands": [{}],
				"events": [{}],
				"capabilityGroups": {
					"required": [{"name": "cap.required"}],
					"optional": [{"name": "cap.optional"}],
					"provided": [{"name": "cap.provided"}]
				},
				"resources": [{}],
				"summary": "grouped"
			},
			{
				"name": "flat",
				"capabilities": [
					{"name": "cap.required", "direction": "required"},
					{"name": "cap.optional", "direction": "optional"},
					{"name": "cap.provided", "direction": "provided"}
				]
			}
		],
		"preflights": {
			"grouped": {
				"ready": false,
				"missingRequiredCapabilities": [
					{"module": "grouped", "name": "cap.required", "direction": "required", "available": false}
				]
			},
			"flat": {
				"ready": true
			}
		}
	}`)

	var out bytes.Buffer
	require.NoError(t, renderModules(&out, payload))

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 3)
	require.Equal(t, []string{
		"NAME", "STATUS", "PROVIDES", "REQUIRES", "COMMANDS", "EVENTS",
		"REQCAPS", "OPTCAPS", "PROVCAPS", "READY", "MISSING", "RES", "SUMMARY",
	}, strings.Fields(lines[0]))
	require.Equal(t, []string{
		"grouped", "stable", "runtime", "jsruntime", "1", "1", "1", "1", "1", "false", "cap:cap.required", "1", "grouped",
	}, strings.Fields(lines[1]))
	require.Equal(t, []string{
		"flat", "-", "-", "-", "0", "0", "1", "1", "1", "true", "-", "0", "-",
	}, strings.Fields(lines[2]))
}

func TestRenderModuleShowsPreflightDetails(t *testing.T) {
	payload := json.RawMessage(`{
		"module": {
			"name": "workflow",
			"status": "stable",
			"requires": ["jsruntime"],
			"commands": [{}, {}],
			"capabilityGroups": {
				"required": [{"name": "brainkit.core.call_js", "type": "func"}],
				"optional": [{"name": "brainkit.core.kit_store", "type": "store"}],
				"provided": [{"name": "workflow.runtime", "type": "runtime"}]
			},
			"summary": "Workflow commands."
		},
		"mounted": false,
		"preflight": {
			"ready": false,
			"requiredModules": [
				{"name": "jsruntime", "requestedBy": "workflow", "mounted": false, "registered": true, "available": true}
			],
			"requiredCapabilities": [
				{"module": "workflow", "name": "brainkit.core.call_js", "direction": "required", "type": "func", "available": false}
			],
			"optionalCapabilities": [
				{"module": "workflow", "name": "brainkit.core.kit_store", "direction": "optional", "type": "store", "available": true, "source": "core", "provider": "brainkit.core"}
			],
			"missingRequiredCapabilities": [
				{"module": "workflow", "name": "brainkit.core.call_js", "direction": "required", "type": "func", "available": false}
			]
		}
	}`)

	var out bytes.Buffer
	require.NoError(t, renderModule(&out, payload))
	text := out.String()
	require.Contains(t, text, "name")
	require.Contains(t, text, "workflow")
	require.Contains(t, text, "ready")
	require.Contains(t, text, "false")
	require.Contains(t, text, "DEPENDENCY")
	require.Contains(t, text, "jsruntime")
	require.Contains(t, text, "brainkit.core.call_js")
	require.Contains(t, text, "brainkit.core.kit_store")
	require.Contains(t, text, "SOURCE")
	require.Contains(t, text, "PROVIDER")
	require.Contains(t, text, "brainkit.core")
	require.Contains(t, text, "self")
	require.Contains(t, text, "workflow.runtime")
}

func TestRenderLifecycleShowsRuntimeAndComponentCounts(t *testing.T) {
	payload := json.RawMessage(`{
		"lifecycle": {
			"runtime": {
				"runtimeId": "rt-1",
				"namespace": "user",
				"callerId": "kit",
				"mountedModules": 2,
				"activeHandlers": 1,
				"draining": false,
				"provider": {
					"aiProviders": 1,
					"vectorStores": 2,
					"storages": 3,
					"closing": true,
					"closed": true,
					"activeProbes": 4,
					"activeOperations": 5
				},
				"storage": {
					"bridgeCount": 1,
					"bridgeNames": ["main"],
					"activeCloses": 1
				},
				"transport": {
					"kind": "memory",
					"ownsTransport": true,
					"activeSubscriptions": 5,
					"activeStreamHeartbeats": 7,
					"closingRouter": false,
					"closingCaller": false,
					"closingTransport": true,
					"closedRouter": true,
					"closedCaller": true,
					"closedTransport": false,
					"router": {
						"handlers": 6,
						"startedHandlers": 6,
						"stoppedHandlers": 0
					}
				}
			},
			"components": [
				{
					"name": "gateway",
					"data": {
						"routeSubscriptions": 4,
						"streamSessions": 0
					}
				}
			]
		}
	}`)

	var out bytes.Buffer
	require.NoError(t, renderLifecycle(&out, payload))
	text := out.String()
	require.Contains(t, text, "runtimeId")
	require.Contains(t, text, "rt-1")
	require.Contains(t, text, "activeProbes")
	require.Contains(t, text, "4")
	require.Contains(t, text, "closing")
	require.Contains(t, text, "closed")
	require.Contains(t, text, "activeOperations")
	require.Contains(t, text, "5")
	require.Contains(t, text, "activeStreamHeartbeats")
	require.Contains(t, text, "7")
	require.Contains(t, text, "ownsTransport")
	require.Contains(t, text, "closedRouter")
	require.Contains(t, text, "closedCaller")
	require.Contains(t, text, "closedTransport")
	require.Contains(t, text, "activeCloses")
	require.Contains(t, text, "gateway")
	require.Contains(t, text, "routeSubscriptions")
}
