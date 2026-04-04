package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brainlet/brainkit/packages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startTestRegistry(t *testing.T) *httptest.Server {
	t.Helper()

	index := packages.RegistryIndex{
		Plugins: []packages.PluginSummary{
			{Name: "echo", Owner: "brainlet", Version: "1.0.0", Description: "Echo test plugin", Capabilities: []string{"tools", "testing"}},
			{Name: "telegram-gateway", Owner: "brainlet", Version: "1.2.0", Description: "Telegram bot gateway", Capabilities: []string{"gateway", "telegram", "messaging"}},
		},
	}
	echoManifest := packages.PluginManifest{
		Name: "echo", Owner: "brainlet", Version: "1.0.0",
		Description: "Echo test plugin", Capabilities: []string{"tools", "testing"},
		Platforms: map[string]packages.PlatformBinary{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/index.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	})
	mux.HandleFunc("/v1/plugins/brainlet/echo/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(echoManifest)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func testSearchByName(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	results, err := client.Search("echo", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "echo", results[0].Name)
}

func testSearchByCapability(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	results, err := client.Search("", []string{"gateway"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "telegram-gateway", results[0].Name)
}

func testSearchMultipleCapabilities(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	results, err := client.Search("", []string{"tools", "testing"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "echo", results[0].Name)
}

func testSearchNoResults(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	results, err := client.Search("nonexistent-xyz", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func testSearchAllPlugins(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	results, err := client.Search("", nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func testFetchManifest(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	manifest, err := client.FetchManifest("brainlet", "echo", "")
	require.NoError(t, err)
	assert.Equal(t, "echo", manifest.Name)
	assert.Equal(t, "brainlet", manifest.Owner)
	assert.Contains(t, manifest.Capabilities, "tools")
}

func testFetchManifestSpecificVersion(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	manifest, err := client.FetchManifest("brainlet", "echo", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", manifest.Version)
}

func testFetchManifestWrongVersion(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	_, err := client.FetchManifest("brainlet", "echo", "99.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func testFetchManifestNotFound(t *testing.T, _ *suite.TestEnv) {
	srv := startTestRegistry(t)
	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "test", URL: srv.URL + "/v1"}})
	_, err := client.FetchManifest("brainlet", "nonexistent", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func testMultipleRegistries(t *testing.T, _ *suite.TestEnv) {
	srv1 := startTestRegistry(t)
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/v1/index.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(packages.RegistryIndex{
			Plugins: []packages.PluginSummary{{Name: "custom-tool", Owner: "acme", Version: "2.0.0", Description: "Custom tool"}},
		})
	})
	srv2 := httptest.NewServer(mux2)
	t.Cleanup(srv2.Close)

	client := packages.NewRegistryClient([]packages.RegistrySource{
		{Name: "official", URL: srv1.URL + "/v1"},
		{Name: "company", URL: srv2.URL + "/v1"},
	})
	results, err := client.Search("", nil)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func testRegistryWithAuth(t *testing.T, _ *suite.TestEnv) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/index.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token-123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(packages.RegistryIndex{
			Plugins: []packages.PluginSummary{{Name: "private-plugin", Owner: "acme", Version: "1.0.0"}},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := packages.NewRegistryClient([]packages.RegistrySource{{Name: "private", URL: srv.URL + "/v1"}})
	results, _ := client.Search("", nil)
	assert.Empty(t, results)

	clientAuth := packages.NewRegistryClient([]packages.RegistrySource{{Name: "private", URL: srv.URL + "/v1", AuthToken: "test-token-123"}})
	results, err := clientAuth.Search("", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "private-plugin", results[0].Name)
}
