// Test: DefaultExporter — construct with batch knobs, assert flush +
// shutdown surface. The exporter is storage-backed and wiring it into
// a real trace belongs in a Go suite test; here we lock the shape.
import { DefaultExporter } from "agent";
import { output } from "kit";

const exporter = new DefaultExporter({
  maxBatchSize: 10,
  maxBufferSize: 100,
  maxBatchWaitMs: 1000,
} as any);

output({
  constructed: exporter !== null && exporter !== undefined,
  hasFlush: typeof (exporter as any).flush === "function",
  hasShutdown: typeof (exporter as any).shutdown === "function",
});
