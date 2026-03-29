package workflow

import (
	"context"
	"encoding/json"
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

			// Live execution
			start := time.Now()
			text, err := e.ai.GenerateText(ctx, prompt)
			duration := time.Since(start)

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

		Instantiate(ctx)
}
