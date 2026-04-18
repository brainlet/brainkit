// Command storage-vectors demonstrates persistent KV storage +
// vector search from a deployed .ts package. One Kit, two
// backends: a SQLite storage ("default") for key/value state and
// a SQLite vector store ("docs") for embeddings + similarity
// search.
//
// Without OPENAI_API_KEY the example still exercises the KV
// path and reports that the vector path was skipped. With a key
// present it embeds three documents via embeddingModel("openai")
// and queries by semantic similarity.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/storage-vectors
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("storage-vectors: %v", err)
	}
}

func run() error {
	tmp, err := os.MkdirTemp("", "brainkit-storage-vectors-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	cfg := brainkit.Config{
		Namespace: "storage-vectors-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmp, "kv.db")),
		},
		Vectors: map[string]brainkit.VectorConfig{
			"default": brainkit.SQLiteVector(filepath.Join(tmp, "vectors.db")),
		},
	}

	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.Providers = []brainkit.ProviderConfig{brainkit.OpenAI(key)}
	}

	kit, err := brainkit.New(cfg)
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	if err := demoKV(ctx, kit); err != nil {
		return fmt.Errorf("kv demo: %w", err)
	}

	if _, hasKey := os.LookupEnv("OPENAI_API_KEY"); !hasKey {
		fmt.Println()
		fmt.Println("OPENAI_API_KEY not set — skipping vector similarity demo.")
		fmt.Println("Set it and re-run to see embeddings + similaritySearch.")
		return nil
	}
	if err := demoVectors(ctx, kit); err != nil {
		// Vector extensions (libsql_vector_idx) aren't in every
		// libsql build. Surface the miss without failing the
		// example — the KV half still shows real persistence.
		fmt.Println()
		fmt.Printf("vector similarity demo skipped: %v\n", err)
		fmt.Println("(libsql vector extensions need to be available on the embedded build)")
	}
	return nil
}

// demoKV deploys a .ts that saves and retrieves a thread through
// the Mastra Memory surface backed by the SQLite storage. Uses
// Memory's saveThread / getThreadById — the two methods exposed
// across every Mastra storage implementation.
func demoKV(ctx context.Context, kit *brainkit.Kit) error {
	code := `
		import { storage } from "kit";
		import { Memory } from "agent";

		const store = storage("default");
		const mem = new Memory({ storage: store });

		bus.on("put", async (msg) => {
			const thread = {
				id: msg.payload.id,
				title: msg.payload.title,
				resourceId: "demo",
				createdAt: new Date(),
				updatedAt: new Date(),
			};
			await mem.saveThread({ thread });
			msg.reply({ saved: msg.payload.id });
		});

		bus.on("get", async (msg) => {
			const thread = await mem.getThreadById({ threadId: msg.payload.id });
			msg.reply({ found: thread !== null, thread });
		});
	`
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("kv-demo", "kv.ts", code)); err != nil {
		return fmt.Errorf("deploy kv.ts: %w", err)
	}

	putPayload := json.RawMessage(`{"id":"t-1","title":"first thread"}`)
	if _, err := call(ctx, kit, "ts.kv-demo.put", putPayload); err != nil {
		return fmt.Errorf("put: %w", err)
	}

	getRes, err := call(ctx, kit, "ts.kv-demo.get", json.RawMessage(`{"id":"t-1"}`))
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	fmt.Println("KV round-trip:")
	fmt.Println(string(getRes))
	return nil
}

// demoVectors deploys a .ts that embeds three strings, upserts
// them into the "docs" vector store, and runs a similarity
// search. Requires an embedding-capable provider.
func demoVectors(ctx context.Context, kit *brainkit.Kit) error {
	code := `
		import { vectorStore, embeddingModel } from "kit";

		const store = vectorStore("default");
		const embed = embeddingModel("openai", "text-embedding-3-small");

		let indexReady = false;
		async function ensureIndex(dim) {
			if (indexReady) return;
			await store.createIndex({ indexName: "demo", dimension: dim });
			indexReady = true;
		}

		bus.on("seed", async (msg) => {
			const docs = msg.payload.docs;
			const { embeddings } = await embed.doEmbed({ values: docs.map((d) => d.text) });
			await ensureIndex(embeddings[0].length);
			await store.upsert({
				indexName: "demo",
				vectors: embeddings,
				ids: docs.map((d) => d.id),
				metadata: docs.map((d) => ({ text: d.text })),
			});
			msg.reply({ inserted: docs.length });
		});

		bus.on("query", async (msg) => {
			const { embeddings } = await embed.doEmbed({ values: [msg.payload.query] });
			const hits = await store.query({
				indexName: "demo",
				queryVector: embeddings[0],
				topK: msg.payload.k || 2,
			});
			msg.reply({ hits: hits.map((h) => ({ id: h.id, score: h.score, text: h.metadata.text })) });
		});
	`
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("vec-demo", "vec.ts", code)); err != nil {
		return fmt.Errorf("deploy vec.ts: %w", err)
	}

	seed := json.RawMessage(`{"docs":[
		{"id":"d1","text":"brainkit is an embeddable runtime for AI agent teams"},
		{"id":"d2","text":"the gateway module exposes HTTP endpoints onto bus topics"},
		{"id":"d3","text":"bananas are yellow fruit that ripen quickly"}
	]}`)
	if _, err := call(ctx, kit, "ts.vec-demo.seed", seed); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	fmt.Println()
	fmt.Println("vector similarity round-trip:")
	q := json.RawMessage(`{"query":"what is brainkit","k":2}`)
	hits, err := call(ctx, kit, "ts.vec-demo.query", q)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	fmt.Println(string(hits))
	return nil
}

func call(ctx context.Context, kit *brainkit.Kit, topic string, payload json.RawMessage) (json.RawMessage, error) {
	return brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   topic,
		Payload: payload,
	}, brainkit.WithCallTimeout(30*time.Second))
}
