import {
  Agent,
  Harness,
  InMemoryStore,
  Memory,
  taskCheckTool,
  taskCompleteTool,
  taskUpdateTool,
  taskWriteTool,
} from "agent";
import { model, output } from "kit";

const events: any[] = [];
const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const agent = new Agent({
  name: "harness-task-tools-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a fixture agent for Harness task tool coverage.",
});

const harness = new Harness({
  id: "harness-task-tools",
  storage,
  memory,
  modes: [
    {
      id: "default",
      name: "Default",
      default: true,
      agent,
    },
  ],
});

const unsubscribe = harness.subscribe((event) => {
  events.push(event);
});

try {
  await harness.init();
  const requestContext = await (harness as any).buildRequestContext();

  const writeResult = await (taskWriteTool as any).execute(
    {
      tasks: [
        {
          id: "research",
          content: "Inspect Harness task tools",
          status: "completed",
          activeForm: "Inspecting Harness task tools",
        },
        {
          id: "tests",
          content: "Write Harness task fixture",
          status: "pending",
          activeForm: "Writing Harness task fixture",
        },
      ],
    },
    { requestContext },
  );

  const updateResult = await (taskUpdateTool as any).execute(
    {
      id: "tests",
      status: "in_progress",
    },
    { requestContext },
  );

  const rejectedResult = await (taskUpdateTool as any).execute(
    {
      id: "research",
      status: "in_progress",
    },
    { requestContext },
  );

  const completeResult = await (taskCompleteTool as any).execute(
    {
      id: "tests",
    },
    { requestContext },
  );

  const checkResult = await (taskCheckTool as any).execute({}, { requestContext });
  const state = harness.getState() as any;
  const displayState = harness.getDisplayState() as any;
  const tasks = Array.isArray(state.tasks) ? state.tasks : [];

  output({
    writeOk: writeResult.isError === false,
    updateOk: updateResult.isError === false,
    rejectedMultipleInProgress:
      rejectedResult.isError === true && String(rejectedResult.content).includes("Only one task can be in_progress"),
    completeOk: completeResult.isError === false,
    taskCount: tasks.length,
    finalStatuses: tasks.map((task: any) => `${task.id}:${task.status}`).join(","),
    checkAllCompleted: checkResult.summary?.allCompleted === true,
    checkCompleted: checkResult.summary?.completed,
    checkIncomplete: checkResult.summary?.incomplete,
    incompleteCount: checkResult.incompleteTasks?.length,
    taskUpdatedEvents: events.filter((event) => event.type === "task_updated").length,
    displayTaskCount: Array.isArray(displayState.tasks) ? displayState.tasks.length : 0,
    displayAllCompleted:
      Array.isArray(displayState.tasks) && displayState.tasks.every((task: any) => task.status === "completed"),
  });
} finally {
  unsubscribe();
  await harness.destroy();
}
