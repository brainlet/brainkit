// Package evalmsg contains the light typed bus API for modules/eval.
package evalmsg

// KitEvalMsg is the single unified eval command. Mode selects the
// evaluation strategy; when empty, it is inferred from Source's file
// extension (".ts" -> "ts", else "script").
//
// Mode values:
//   - "script" deploys Code as a temp .ts, then reads
//     globalThis.__module_result.
//   - "ts" evaluates TS source directly in the current runtime context.
//   - "module" evaluates as an ES module and supports import statements.
type KitEvalMsg struct {
	Source string `json:"source,omitempty"`
	Code   string `json:"code"`
	Mode   string `json:"mode,omitempty"`
}

func (KitEvalMsg) BusTopic() string { return "kit.eval" }

// KitEvalResp reports the string result produced by eval.
type KitEvalResp struct {
	Result string `json:"result"`
}
