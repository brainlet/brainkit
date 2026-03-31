package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// registerAIHostFunctions registers the "ai" module with generate and embed functions.
// These route to the AIGenerator interface (implemented by the Kernel via EvalTS).
func (e *Engine) registerAIHostFunctions(ctx context.Context, rt wazero.Runtime, ar *activeRun) {
	rt.NewHostModuleBuilder("ai").

		// ai.generate(prompt: string) → string
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, promptPtr uint32) uint32 {
			prompt := readASString(m, promptPtr)

			// Check journal for replay
			argsJSON, _ := json.Marshal(prompt)
			if result, ok := ar.journal.GetRecordedResult("ai", "generate", argsJSON); ok {
				var text string
				json.Unmarshal(result, &text)
				ptr, _ := writeASString(ctx, m, text)
				return ptr
			}

			// Live execution with step-level timeout (30s)
			callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
			start := time.Now()
			text, err := e.ai.GenerateText(callCtx, prompt)
			duration := time.Since(start)
			callCancel()

			resultJSON, _ := json.Marshal(text)
			if err != nil {
				ar.journal.RecordCall("ai", "generate", argsJSON, nil, err, duration)
				log.Printf("[workflow:ai] generate failed: %v", err)
				ptr, _ := writeASString(ctx, m, "")
				return ptr
			}

			ar.journal.RecordCall("ai", "generate", argsJSON, resultJSON, nil, duration)
			ptr, _ := writeASString(ctx, m, text)
			return ptr
		}).Export("generate").

		// ai.embed(text: string) → string (JSON array of floats)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, textPtr uint32) uint32 {
			text := readASString(m, textPtr)

			argsJSON, _ := json.Marshal(text)
			if result, ok := ar.journal.GetRecordedResult("ai", "embed", argsJSON); ok {
				ptr, _ := writeASString(ctx, m, string(result))
				return ptr
			}

			if e.ai == nil {
				ar.journal.RecordCall("ai", "embed", argsJSON, nil, fmt.Errorf("no AI provider"), 0)
				ptr, _ := writeASString(ctx, m, "[]")
				return ptr
			}

			callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
			start := time.Now()
			result, err := e.ai.EmbedText(callCtx, text)
			duration := time.Since(start)
			callCancel()

			if err != nil {
				ar.journal.RecordCall("ai", "embed", argsJSON, nil, err, duration)
				log.Printf("[workflow:ai] embed failed: %v", err)
				ptr, _ := writeASString(ctx, m, "[]")
				return ptr
			}

			ar.journal.RecordCall("ai", "embed", argsJSON, json.RawMessage(result), nil, duration)
			ptr, _ := writeASString(ctx, m, result)
			return ptr
		}).Export("embed").

		Instantiate(ctx)
}
