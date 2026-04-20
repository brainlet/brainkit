// `parsePartialJson(raw)` attempts to decode JSON that may be
// incomplete (mid-stream from a model) and returns
// `{value, state}` where state is "successful-parse" |
// "repaired-parse" | "failed-parse" | "undefined-input".
// Used to surface progressive object deltas to a UI while an LLM
// is still emitting tokens.
import { parsePartialJson } from "ai";
import { output } from "kit";

const complete = await parsePartialJson('{"name":"brainkit","version":"1.0"}');
const partial = await parsePartialJson('{"name":"brainkit","versi');
const empty = await parsePartialJson("");
const broken = await parsePartialJson("not json");

output({
  completeState: complete.state,
  completeName: (complete.value as any)?.name,
  partialState: partial.state,
  partialName: (partial.value as any)?.name,
  emptyState: empty.state,
  brokenState: broken.state,
});
