// Ported from: packages/core/src/processors/processors/tool-search.ts
package concreteprocessors

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// Tool is a stub for ../../tools.Tool.
// TODO: import from tools package once ported.
type Tool struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}


// ---------------------------------------------------------------------------
// ThreadState
// ---------------------------------------------------------------------------

// threadState holds thread-scoped tool state with timestamp for TTL management.
type threadState struct {
	tools        map[string]bool
	lastAccessed time.Time
}

// ---------------------------------------------------------------------------
// SearchResult
// ---------------------------------------------------------------------------

// ToolSearchResult holds a search result with ranking score.
type ToolSearchResult struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// ---------------------------------------------------------------------------
// BM25 Index (inline implementation)
// ---------------------------------------------------------------------------

// bm25Document holds a single indexed document.
type bm25Document struct {
	id      string
	tokens  []string
	termFreq map[string]int
	length  int
}

// bm25Index is a simple BM25 search index for tool discovery.
type bm25Index struct {
	mu        sync.RWMutex
	documents map[string]*bm25Document
	docFreq   map[string]int
	totalDocs int
	avgDL     float64
	k1        float64
	b         float64
}

func newBM25Index() *bm25Index {
	return &bm25Index{
		documents: make(map[string]*bm25Document),
		docFreq:   make(map[string]int),
		k1:        1.5,
		b:         0.75,
	}
}

// tokenize splits text into tokens suitable for tool names and descriptions.
func (idx *bm25Index) tokenize(text string) []string {
	lower := strings.ToLower(text)

	// Split on whitespace, hyphens, underscores, and common punctuation.
	var tokens []string
	var current strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() >= 2 {
				tokens = append(tokens, current.String())
			}
			current.Reset()
		}
	}
	if current.Len() >= 2 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// add indexes a document.
func (idx *bm25Index) add(id, content string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	tokens := idx.tokenize(content)
	termFreq := make(map[string]int)
	for _, token := range tokens {
		termFreq[token]++
	}

	// Track document frequency.
	seenTerms := make(map[string]bool)
	for _, token := range tokens {
		if !seenTerms[token] {
			seenTerms[token] = true
			idx.docFreq[token]++
		}
	}

	idx.documents[id] = &bm25Document{
		id:       id,
		tokens:   tokens,
		termFreq: termFreq,
		length:   len(tokens),
	}

	idx.totalDocs++

	// Update average document length.
	totalLen := 0
	for _, doc := range idx.documents {
		totalLen += doc.length
	}
	idx.avgDL = float64(totalLen) / float64(idx.totalDocs)
}

// search performs a BM25 search and returns results sorted by score.
func (idx *bm25Index) search(query string, topK int, minScore float64) []ToolSearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.totalDocs == 0 {
		return nil
	}

	queryTokens := idx.tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	type scoredDoc struct {
		id    string
		score float64
	}
	var scored []scoredDoc

	for docID, doc := range idx.documents {
		score := 0.0

		for _, term := range queryTokens {
			tf, exists := doc.termFreq[term]
			if !exists {
				continue
			}

			df := idx.docFreq[term]
			idf := math.Log(1 + (float64(idx.totalDocs)-float64(df)+0.5)/(float64(df)+0.5))

			tfNorm := (float64(tf) * (idx.k1 + 1)) /
				(float64(tf) + idx.k1*(1-idx.b+idx.b*float64(doc.length)/idx.avgDL))

			score += idf * tfNorm
		}

		if score > minScore {
			scored = append(scored, scoredDoc{id: docID, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if len(scored) > topK {
		scored = scored[:topK]
	}

	var results []ToolSearchResult
	for _, s := range scored {
		results = append(results, ToolSearchResult{
			Name:  s.id,
			Score: s.score,
		})
	}

	return results
}

func (idx *bm25Index) size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.totalDocs
}

// ---------------------------------------------------------------------------
// ToolSearchProcessorOptions
// ---------------------------------------------------------------------------

// ToolSearchProcessorOptions configures the ToolSearchProcessor.
type ToolSearchProcessorOptions struct {
	// Tools are all tools that can be searched and loaded dynamically.
	// These tools are not immediately available -- they must be discovered via search and loaded on demand.
	Tools map[string]*Tool

	// SearchTopK is the maximum number of tools to return in search results. Default: 5.
	SearchTopK int

	// SearchMinScore is the minimum relevance score (0-1) for including a tool. Default: 0.
	SearchMinScore float64

	// TTL is the time-to-live for thread state in milliseconds.
	// After this duration of inactivity, thread state will be eligible for cleanup.
	// Set to 0 to disable TTL cleanup. Default: 3600000 (1 hour).
	TTL int64
}

// ---------------------------------------------------------------------------
// ToolSearchProcessor
// ---------------------------------------------------------------------------

// ToolSearchProcessor enables dynamic tool discovery and loading.
//
// Instead of providing all tools to the agent upfront, this processor:
//  1. Gives the agent two meta-tools: search_tools and load_tool
//  2. Agent searches for relevant tools using keywords
//  3. Agent loads specific tools into the conversation on demand
//  4. Loaded tools become immediately available for use
//
// This pattern dramatically reduces context usage when working with many tools (100+).
type ToolSearchProcessor struct {
	processors.BaseProcessor
	allTools         map[string]*Tool
	searchTopK       int
	searchMinScore   float64
	ttl              int64
	index            *bm25Index
	toolDescriptions map[string]string

	mu                sync.Mutex
	threadLoadedTools map[string]*threadState
}

// NewToolSearchProcessor creates a new ToolSearchProcessor.
func NewToolSearchProcessor(opts ToolSearchProcessorOptions) *ToolSearchProcessor {
	topK := opts.SearchTopK
	if topK == 0 {
		topK = 5
	}

	ttl := opts.TTL
	if ttl == 0 {
		ttl = 3600000 // 1 hour in ms
	}

	tsp := &ToolSearchProcessor{
		BaseProcessor:     processors.NewBaseProcessor("tool-search", "Tool Search Processor"),
		allTools:          opts.Tools,
		searchTopK:        topK,
		searchMinScore:    opts.SearchMinScore,
		ttl:               ttl,
		index:             newBM25Index(),
		toolDescriptions:  make(map[string]string),
		threadLoadedTools: make(map[string]*threadState),
	}

	// Index all tools.
	tsp.indexTools()

	// Start periodic cleanup if TTL is enabled.
	if tsp.ttl > 0 {
		go tsp.scheduleCleanup()
	}

	return tsp
}

// ProcessInputStep provides search and load meta-tools and loaded tools.
func (tsp *ToolSearchProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	threadID := tsp.getThreadID(args)
	loadedToolNames := tsp.getLoadedToolNames(threadID)

	// Build system messages with tool search instructions.
	systemMessages := []processors.CoreMessageV4{
		{
			Role: "system",
			Content: "To discover available tools, call search_tools with a keyword query. " +
				"To add a tool to the conversation, call load_tool with the tool name. " +
				"Tools must be loaded before they can be used.",
		},
	}

	// Get loaded tools for this thread.
	loadedTools := tsp.getLoadedTools(threadID)

	// Merge: meta-tools + existing tools + loaded tools.
	// TODO: Once createTool is ported, create actual search_tools and load_tool Tool objects.
	// For now, we provide the loaded tools and system messages.
	_ = loadedToolNames // Used by meta-tool execute functions (not yet ported).

	tools := make(map[string]any)
	if args.Tools != nil {
		for k, v := range args.Tools {
			tools[k] = v
		}
	}
	for k, v := range loadedTools {
		tools[k] = v
	}

	result := &processors.ProcessInputStepResult{
		SystemMessages: systemMessages,
		Tools:          tools,
	}

	return result, nil, nil
}

// ProcessInput is not implemented for this processor.
func (tsp *ToolSearchProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (tsp *ToolSearchProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (tsp *ToolSearchProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (tsp *ToolSearchProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ClearState clears loaded tools for a specific thread.
func (tsp *ToolSearchProcessor) ClearState(threadID string) {
	if threadID == "" {
		threadID = "default"
	}
	tsp.mu.Lock()
	defer tsp.mu.Unlock()
	delete(tsp.threadLoadedTools, threadID)
}

// ClearAllState clears all thread state.
func (tsp *ToolSearchProcessor) ClearAllState() {
	tsp.mu.Lock()
	defer tsp.mu.Unlock()
	tsp.threadLoadedTools = make(map[string]*threadState)
}

// CleanupNow manually triggers cleanup of stale state.
// Returns the number of threads cleaned up.
func (tsp *ToolSearchProcessor) CleanupNow() int {
	return tsp.cleanupStaleState()
}

// GetStateStats returns statistics about current thread state.
func (tsp *ToolSearchProcessor) GetStateStats() (threadCount int, oldestAccessTime *time.Time) {
	tsp.mu.Lock()
	defer tsp.mu.Unlock()

	if len(tsp.threadLoadedTools) == 0 {
		return 0, nil
	}

	var oldest time.Time
	first := true
	for _, state := range tsp.threadLoadedTools {
		if first || state.lastAccessed.Before(oldest) {
			oldest = state.lastAccessed
			first = false
		}
	}

	return len(tsp.threadLoadedTools), &oldest
}

// SearchTools searches for tools matching the query using BM25 ranking
// with name-match boosting.
func (tsp *ToolSearchProcessor) SearchTools(query string) []ToolSearchResult {
	if tsp.index.size() == 0 {
		return nil
	}

	// Get BM25 results (request more than topK to allow for re-ranking after boosting).
	bm25Results := tsp.index.search(query, tsp.searchTopK*2, 0)

	if len(bm25Results) == 0 {
		return nil
	}

	// Apply name-match boosting on top of BM25 scores.
	queryTokens := tsp.index.tokenize(query)

	type boostedResult struct {
		name  string
		score float64
	}
	var boosted []boostedResult

	for _, result := range bm25Results {
		score := result.Score
		nameLower := strings.ToLower(result.Name)

		for _, term := range queryTokens {
			if nameLower == term {
				score += 5
			} else if strings.Contains(nameLower, term) {
				score += 2
			}
		}

		boosted = append(boosted, boostedResult{name: result.Name, score: score})
	}

	// Re-sort after boosting, filter by minScore, apply topK.
	sort.Slice(boosted, func(i, j int) bool {
		return boosted[i].score > boosted[j].score
	})

	var results []ToolSearchResult
	for _, r := range boosted {
		if r.score <= tsp.searchMinScore {
			continue
		}
		if len(results) >= tsp.searchTopK {
			break
		}

		description := tsp.toolDescriptions[r.name]
		if len(description) > 150 {
			description = description[:147] + "..."
		}

		results = append(results, ToolSearchResult{
			Name:        r.name,
			Description: description,
			Score:       math.Round(r.score*100) / 100,
		})
	}

	return results
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// getThreadID extracts the thread ID from request context.
func (tsp *ToolSearchProcessor) getThreadID(args processors.ProcessInputStepArgs) string {
	if args.RequestContext != nil {
		if val := args.RequestContext.Get(requestcontext.MastraThreadIDKey); val != nil {
			if threadID, ok := val.(string); ok && threadID != "" {
				return threadID
			}
		}
	}
	return "default"
}

// getLoadedToolNames gets the set of loaded tool names for the current thread.
func (tsp *ToolSearchProcessor) getLoadedToolNames(threadID string) map[string]bool {
	tsp.mu.Lock()
	defer tsp.mu.Unlock()

	state, ok := tsp.threadLoadedTools[threadID]
	if !ok {
		state = &threadState{
			tools:        make(map[string]bool),
			lastAccessed: time.Now(),
		}
		tsp.threadLoadedTools[threadID] = state
	}
	state.lastAccessed = time.Now()
	return state.tools
}

// getLoadedTools gets loaded tools as Tool objects for the current thread.
func (tsp *ToolSearchProcessor) getLoadedTools(threadID string) map[string]*Tool {
	loadedNames := tsp.getLoadedToolNames(threadID)
	loadedTools := make(map[string]*Tool)

	for toolName := range loadedNames {
		if tool, ok := tsp.allTools[toolName]; ok {
			loadedTools[toolName] = tool
		} else {
			// Try matching by tool.ID.
			for _, t := range tsp.allTools {
				if t.ID == toolName {
					loadedTools[toolName] = t
					break
				}
			}
		}
	}

	return loadedTools
}

// indexTools indexes all tools into the BM25 index.
func (tsp *ToolSearchProcessor) indexTools() {
	for key, tool := range tsp.allTools {
		name := tool.ID
		if name == "" {
			name = key
		}
		description := tool.Description
		tsp.index.add(name, name+" "+description)
		tsp.toolDescriptions[name] = description
	}
}

// cleanupStaleState removes threads that haven't been accessed within the TTL period.
func (tsp *ToolSearchProcessor) cleanupStaleState() int {
	if tsp.ttl <= 0 {
		return 0
	}

	tsp.mu.Lock()
	defer tsp.mu.Unlock()

	now := time.Now()
	ttlDuration := time.Duration(tsp.ttl) * time.Millisecond
	cleanedCount := 0

	for threadID, state := range tsp.threadLoadedTools {
		if now.Sub(state.lastAccessed) > ttlDuration {
			delete(tsp.threadLoadedTools, threadID)
			cleanedCount++
		}
	}

	return cleanedCount
}

// scheduleCleanup periodically cleans up stale thread state.
func (tsp *ToolSearchProcessor) scheduleCleanup() {
	cleanupInterval := time.Duration(tsp.ttl/2) * time.Millisecond
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		tsp.cleanupStaleState()
	}
}
