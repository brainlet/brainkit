//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestFixture_TS_OMBasic(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/om-basic.js")
	result, err := kit.EvalModule(context.Background(), "om-basic.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		MessagesCount float64 `json:"messagesCount"`
		LastResponse  string  `json:"lastResponse"`
		Success       bool    `json:"success"`
		Error         string  `json:"error"`
		Stack         string  `json:"stack"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Fatalf("fixture error: %s\nstack: %s", out.Error, out.Stack)
	}
	if !out.Success {
		t.Fatal("expected success=true")
	}
	if out.MessagesCount != 8 {
		t.Errorf("messagesCount = %v, want 8", out.MessagesCount)
	}
	t.Logf("om-basic: %d messages, lastResponse=%q", int(out.MessagesCount), out.LastResponse)
}

func TestFixture_TS_OMRetrieval(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/tursodatabase/libsql-server:latest",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/health").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("could not start LibSQL container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8080")
	libsqlURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"LIBSQL_URL": libsqlURL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/om-retrieval.js")
	result, err := kit.EvalModule(context.Background(), "om-retrieval.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Recall1      string `json:"recall1"`
		Recall2      string `json:"recall2"`
		Recall3      string `json:"recall3"`
		HasDogName   bool   `json:"hasDogName"`
		HasBirthCity bool   `json:"hasBirthCity"`
		HasCompany   bool   `json:"hasCompany"`
		Error        string `json:"error"`
		Stack        string `json:"stack"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Fatalf("fixture error: %s\nstack: %s", out.Error, out.Stack)
	}

	// At least 2 of 3 facts should be recalled (LLM is non-deterministic)
	recalled := 0
	if out.HasDogName {
		recalled++
	}
	if out.HasBirthCity {
		recalled++
	}
	if out.HasCompany {
		recalled++
	}

	t.Logf("Recalled %d/3 facts", recalled)
	t.Logf("Dog: %s", out.Recall1)
	t.Logf("Birth: %s", out.Recall2)
	t.Logf("Company: %s", out.Recall3)

	if recalled < 2 {
		t.Errorf("Agent should recall at least 2/3 facts from observations, got %d", recalled)
	}
}

func TestFixture_TS_OMThreadIsolation(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/tursodatabase/libsql-server:latest",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/health").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("could not start LibSQL container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8080")
	libsqlURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"LIBSQL_URL": libsqlURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/om-thread-isolation.js")
	result, err := kit.EvalModule(context.Background(), "om-thread-isolation.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		RecallA   string `json:"recallA"`
		RecallB   string `json:"recallB"`
		AHasAlpha bool   `json:"aHasAlpha"`
		AHasBeta  bool   `json:"aHasBeta"`
		BHasBeta  bool   `json:"bHasBeta"`
		BHasAlpha bool   `json:"bHasAlpha"`
		Error     string `json:"error"`
		Stack     string `json:"stack"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Fatalf("fixture error: %s\nstack: %s", out.Error, out.Stack)
	}

	t.Logf("Thread A recall: %s", out.RecallA)
	t.Logf("Thread B recall: %s", out.RecallB)

	if !out.AHasAlpha {
		t.Error("Thread A should recall ALPHA-7742")
	}
	if out.AHasBeta {
		t.Error("Thread A should NOT see BETA-9901")
	}
	if !out.BHasBeta {
		t.Error("Thread B should recall BETA-9901")
	}
	if out.BHasAlpha {
		t.Error("Thread B should NOT see ALPHA-7742")
	}
}

func TestFixture_TS_OMEndToEnd(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/tursodatabase/libsql-server:latest",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/health").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("could not start LibSQL container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8080")
	libsqlURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"LIBSQL_URL": libsqlURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/om-end-to-end.js")
	result, err := kit.EvalModule(context.Background(), "om-end-to-end.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Summary    string `json:"summary"`
		HasTokyo   bool   `json:"hasTokyo"`
		HasBudget  bool   `json:"hasBudget"`
		HasAllergy bool   `json:"hasAllergy"`
		HasHotel   bool   `json:"hasHotel"`
		HasFlight  bool   `json:"hasFlight"`
		Error      string `json:"error"`
		Stack      string `json:"stack"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Fatalf("fixture error: %s\nstack: %s", out.Error, out.Stack)
	}

	t.Logf("Summary: %s", out.Summary)

	recalled := 0
	facts := map[string]bool{
		"hasTokyo": out.HasTokyo, "hasBudget": out.HasBudget,
		"hasAllergy": out.HasAllergy, "hasHotel": out.HasHotel, "hasFlight": out.HasFlight,
	}
	for name, has := range facts {
		if has {
			recalled++
			t.Logf("  + %s", name)
		} else {
			t.Logf("  - %s", name)
		}
	}

	if recalled < 3 {
		t.Errorf("Agent should recall at least 3/5 trip facts, got %d", recalled)
	}
}
