import { createStep, createWorkflow, z } from "agent";
import { output } from "kit";

const writerStep = createStep({
  id: "writer-step",
  inputSchema: z.object({ value: z.string() }),
  outputSchema: z.object({ value: z.string() }),
  execute: async ({ inputData, writer }) => {
    await writer?.write({
      type: "custom-status",
      data: { message: `write:${inputData.value}` },
    });
    await writer?.custom?.({
      type: "custom-status",
      data: { message: `custom:${inputData.value}` },
    });
    return { value: inputData.value.toUpperCase() };
  },
});

const workflow = createWorkflow({
  id: "stream-writer-wf",
  inputSchema: z.object({ value: z.string() }),
  outputSchema: z.object({ value: z.string() }),
}).then(writerStep).commit();

const run = await workflow.createRun();
const streamResult = await (run as any).stream({ inputData: { value: "alpha" } });
const iterable = streamResult?.fullStream
  ? streamResult.fullStream
  : streamResult?.[Symbol.asyncIterator]
  ? streamResult
  : streamResult?.stream ?? streamResult?.fullStream;

const events: any[] = [];
for await (const event of iterable) {
  events.push(event);
}

const writeEvent = events.find(
  (event) => event.type === "custom-status" && event.data?.message === "write:alpha",
);
const customEvent = events.find(
  (event) => event.type === "custom-status" && event.data?.message === "custom:alpha",
);

output({
  eventCount: events.length,
  eventTypes: events.map((event) => event.type).join(","),
  customMessages: events
    .filter((event) => event.type === "custom-status")
    .map((event) => event.data?.message)
    .join(","),
  writerWriteBubbled: Boolean(writeEvent),
  sawCustomEvent: Boolean(customEvent),
  completed: events.some(
    (event) =>
      event.type === "finish" ||
      event.type === "watch" ||
      event.type === "workflow-result" ||
      event.type === "workflow-finish",
  ),
});
