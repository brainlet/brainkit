// Command rag-pipeline demonstrates the full Mastra RAG flow on
// brainkit: ingest a seed corpus, chunk with MDocument, embed,
// upsert into pgvector, expose the knowledge base through
// createVectorQueryTool on an Agent, ask questions that can only
// be answered from the corpus, then ask one that can't — and
// verify the agent declines instead of hallucinating.
//
// Requires OPENAI_API_KEY + a running pgvector (see
// docker-compose.yml shipped alongside).
//
// Run from the repo root:
//
//	docker compose -f examples/rag-pipeline/docker-compose.yml up -d
//	export OPENAI_API_KEY=sk-...
//	export PGVECTOR_URL="postgres://brainkit:brainkit@127.0.0.1:5434/brainkit?sslmode=disable"
//	go run ./examples/rag-pipeline
//	docker compose -f examples/rag-pipeline/docker-compose.yml down -v
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

// seedDocs are deliberately fictional so answers must come from
// the corpus — not from the model's training data.
var seedDocs = []doc{
	{ID: "flr-001", Text: "Floria GridLamp is a ceiling lamp that draws 18 watts and comes with an integrated dimmer rated to 2000 lumens. It ships with a 12-foot power cable."},
	{ID: "flr-002", Text: "The Floria MossDesk has a walnut top and four adjustable legs; its maximum load capacity is 85 kilograms. Floria assembles every MossDesk in Porto, Portugal."},
	{ID: "flr-003", Text: "Vintlo Mk-7 is an espresso machine with a 58mm portafilter, dual 800 mL boilers, and a PID temperature controller accurate to ±0.2°C. Grind size dial has 60 steps."},
	{ID: "flr-004", Text: "The Vintlo Mk-7 uses a rotary pump powered by a 165-watt motor and produces 9 bars of brew pressure at the group head. Cold water tank holds 2.4 liters."},
	{ID: "flr-005", Text: "Albra TreadCloud is a running shoe whose midsole is a 70/30 blend of supercritical EVA foam and recycled rubber. The outsole has a 4.5 millimeter lug depth."},
	{ID: "flr-006", Text: "Albra TreadCloud comes in Moss, Ember, and Harbor colorways. All three weigh 248 grams in a men's US size 10. The upper is a jacquard-knit polyester."},
}

type doc struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type ingestReply struct {
	Inserted int `json:"inserted"`
}

type askReply struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"`
}

func main() {
	rerank := flag.Bool("rerank", false, "route the ambiguous question through rerankWithScorer before the agent sees the sources")
	flag.Parse()
	if err := run(*rerank); err != nil {
		log.Fatalf("rag-pipeline: %v", err)
	}
}

func run(rerank bool) error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	pgURL := os.Getenv("PGVECTOR_URL")
	if pgURL == "" {
		return fmt.Errorf("PGVECTOR_URL is required — start the compose stack first:\n" +
			"  docker compose -f examples/rag-pipeline/docker-compose.yml up -d\n" +
			"  export PGVECTOR_URL=\"postgres://brainkit:brainkit@127.0.0.1:5434/brainkit?sslmode=disable\"")
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "rag-pipeline-demo",
		Transport: brainkit.Memory(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
		Vectors: map[string]brainkit.VectorConfig{
			"docs": brainkit.PgVectorStore(pgURL),
		},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Println("[1/4] deploying rag-pipeline")
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("rag-pipeline", "rag.ts", ragSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	fmt.Printf("[2/4] ingesting %d seed documents (chunk → embed → upsert)\n", len(seedDocs))
	ingestPayload, _ := json.Marshal(map[string]any{"docs": seedDocs})
	ingest, err := brainkit.Call[sdk.CustomMsg, ingestReply](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.rag-pipeline.ingest",
		Payload: ingestPayload,
	}, brainkit.WithCallTimeout(90*time.Second))
	if err != nil {
		return fmt.Errorf("ingest: %w", err)
	}
	fmt.Printf("        inserted %d chunks across %d docs\n\n", ingest.Inserted, len(seedDocs))

	fmt.Println("[3/4] positive question (answer lives in the corpus)")
	posQ := "What is the brew pressure of the Vintlo Mk-7, and what powers its pump?"
	posRes, err := ask(kit, ctx, posQ, rerank)
	if err != nil {
		return fmt.Errorf("positive ask: %w", err)
	}
	printAsk(posQ, posRes)
	wantFacts := []string{"9 bars", "165", "rotary"}
	missing := missingFacts(posRes.Answer, wantFacts)
	if len(missing) > 0 {
		fmt.Printf("        ⚠ expected facts missing from the answer: %v\n", missing)
	} else {
		fmt.Println("        ✓ answer cites the corpus facts")
	}
	if !hasSource(posRes.Sources, "flr-004") {
		fmt.Println("        ⚠ expected source flr-004 in citations")
	} else {
		fmt.Println("        ✓ cited flr-004")
	}
	fmt.Println()

	fmt.Println("[4/4] negative question (not in corpus — agent should decline)")
	negQ := "What flavor of ice cream does the Vintlo Mk-7 pair best with?"
	negRes, err := ask(kit, ctx, negQ, rerank)
	if err != nil {
		return fmt.Errorf("negative ask: %w", err)
	}
	printAsk(negQ, negRes)
	if looksLikeDecline(negRes.Answer) {
		fmt.Println("        ✓ agent declined rather than hallucinating")
	} else {
		fmt.Println("        ⚠ agent did not visibly decline — review the answer above")
	}
	return nil
}

func ask(kit *brainkit.Kit, ctx context.Context, question string, rerank bool) (askReply, error) {
	payload, _ := json.Marshal(map[string]any{"question": question, "rerank": rerank})
	return brainkit.Call[sdk.CustomMsg, askReply](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.rag-pipeline.ask",
		Payload: payload,
	}, brainkit.WithCallTimeout(60*time.Second))
}

func printAsk(question string, r askReply) {
	fmt.Printf("        Q: %s\n", question)
	fmt.Printf("        A: %s\n", r.Answer)
	if len(r.Sources) > 0 {
		fmt.Printf("        sources: %v\n", r.Sources)
	}
}

func missingFacts(answer string, want []string) []string {
	lower := strings.ToLower(answer)
	var missing []string
	for _, w := range want {
		if !strings.Contains(lower, strings.ToLower(w)) {
			missing = append(missing, w)
		}
	}
	return missing
}

func hasSource(sources []string, id string) bool {
	for _, s := range sources {
		if s == id {
			return true
		}
	}
	return false
}

func looksLikeDecline(answer string) bool {
	lower := strings.ToLower(answer)
	for _, marker := range []string{"don't know", "do not know", "cannot answer", "can't answer", "no information", "not in the", "not mentioned", "no mention", "doesn't mention", "does not mention", "not available", "not found"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// ragSource runs the ingest + ask topics inside a SES compartment.
//
// createVectorQueryTool uses the direct-instance shape
// (`vectorStore: vectorStore("docs")`) because brainkit doesn't
// wire a Mastra instance into the compartment, so the
// `vectorStoreName: "docs"` shape (which calls
// `mastra.getVector(...)`) has nothing to look up.
const ragSource = `
import { vectorStore, embeddingModel } from "kit";
import { Agent, model, MDocument, createVectorQueryTool, rerankWithScorer } from "agent";

const INDEX_NAME = "docs";
const EMBED_MODEL = "text-embedding-3-small";

const store = vectorStore("docs");
const embedder = embeddingModel("openai", EMBED_MODEL);

let indexReady = false;
async function ensureIndex(dim) {
    if (indexReady) return;
    await store.createIndex({ indexName: INDEX_NAME, dimension: dim });
    indexReady = true;
}

bus.on("ingest", async (msg) => {
    const docs = (msg.payload && msg.payload.docs) || [];

    // MDocument.fromText accepts (text, meta?) — meta flows
    // through to every chunk as chunk.metadata.
    const allChunks = [];
    for (const d of docs) {
        const doc = MDocument.fromText(d.text, { docId: d.id });
        const chunks = await doc.chunk({
            strategy: "recursive",
            maxSize: 400,
            overlap: 40,
        });
        for (const c of chunks) {
            allChunks.push({
                text: c.text,
                metadata: { ...c.metadata, docId: d.id, text: c.text },
            });
        }
    }
    if (allChunks.length === 0) {
        msg.reply({ inserted: 0 });
        return;
    }

    const { embeddings } = await embedder.doEmbed({
        values: allChunks.map((c) => c.text),
    });
    await ensureIndex(embeddings[0].length);

    await store.upsert({
        indexName: INDEX_NAME,
        vectors: embeddings,
        ids: allChunks.map((_, i) => "chunk-" + i),
        metadata: allChunks.map((c) => c.metadata),
    });
    msg.reply({ inserted: allChunks.length });
});

const vectorQueryTool = createVectorQueryTool({
    vectorStore: store,
    indexName: INDEX_NAME,
    model: embedder,
    topK: 4,
    description: "Search the product knowledge base for facts about Floria, Vintlo, and Albra products.",
});

const ragAgent = new Agent({
    name: "rag-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: [
        "You are a product knowledge assistant. Answer questions about Floria, Vintlo, and Albra products using ONLY the context returned by the vectorQueryTool.",
        "If the tool output does not contain the answer, reply exactly: 'I don't know based on the available documents.' Do not invent facts.",
        "When you cite, include the docId values you used from the tool output.",
    ].join(" "),
    tools: { vectorQueryTool },
});
kit.register("agent", "rag-agent", ragAgent);

// extractSources reads the Mastra step stream and collects every
// docId the vectorQueryTool returned on this turn.
function extractSources(steps) {
    const ids = new Set();
    for (const step of steps || []) {
        for (const part of step.content || []) {
            if (part.type === "tool-result" && part.output && Array.isArray(part.output.relevantContext)) {
                for (const hit of part.output.relevantContext) {
                    const id = hit.metadata && hit.metadata.docId;
                    if (id) ids.add(id);
                }
            }
        }
    }
    return Array.from(ids);
}

bus.on("ask", async (msg) => {
    const question = (msg.payload && msg.payload.question) || "";
    const useRerank = !!(msg.payload && msg.payload.rerank);

    // Rerank path: fetch a wider candidate set directly, rerank
    // with the model-as-judge, hand the top-3 back to the agent
    // as an explicit system preamble. The vectorQueryTool is
    // still available; the agent may re-query if needed.
    let rerankPreamble = "";
    if (useRerank) {
        const { embeddings } = await embedder.doEmbed({ values: [question] });
        const candidates = await store.query({
            indexName: INDEX_NAME,
            queryVector: embeddings[0],
            topK: 8,
        });
        const reranked = await rerankWithScorer({
            results: candidates,
            query: question,
            scorer: embedder,
            options: {
                weights: { semantic: 0.5, vector: 0.3, position: 0.2 },
                topK: 3,
                queryEmbedding: embeddings[0],
            },
        });
        rerankPreamble = "Reranked top context:\n" +
            reranked.map((r, i) =>
                "[" + (i+1) + "] docId=" + (r.result.metadata && r.result.metadata.docId) +
                " — " + (r.result.metadata && r.result.metadata.text)
            ).join("\n") + "\n\n";
    }

    const gen = await ragAgent.generate(rerankPreamble + question);
    const sources = extractSources(gen.steps);
    msg.reply({ answer: gen.text || "", sources });
});
`
