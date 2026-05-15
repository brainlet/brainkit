import { Agent, Harness, InMemoryStore, Memory, askUserTool, submitPlanTool } from "agent";
import { model, output } from "kit";

const events: any[] = [];
const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const agent = new Agent({
  name: "harness-interactive-tools-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a fixture agent for Harness interactive tool coverage.",
});

const harness = new Harness({
  id: "harness-interactive-tools",
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

function lastEvent(type: string) {
  return [...events].reverse().find((event) => event.type === type);
}

try {
  await harness.init();
  const requestContext = await (harness as any).buildRequestContext();

  const questionResultPromise = (askUserTool as any).execute(
    {
      question: "Pick supported Harness options.",
      options: [
        { label: "task tools", description: "Task state tools" },
        { label: "subagents", description: "Delegated subagents" },
        { label: "plans", description: "Plan approval" },
      ],
      selectionMode: "multi_select",
    },
    { requestContext },
  );

  const questionEvent = lastEvent("ask_question");
  const displayWithQuestion = harness.getDisplayState() as any;
  const displayQuestionMatched =
    displayWithQuestion.pendingQuestion?.questionId === questionEvent?.questionId &&
    displayWithQuestion.pendingQuestion?.selectionMode === "multi_select";
  harness.respondToQuestion({
    questionId: questionEvent?.questionId,
    answer: ["task tools", "plans"],
  });
  const questionResult = await questionResultPromise;

  const approvalResultPromise = (submitPlanTool as any).execute(
    {
      title: "Fixture plan",
      plan: "1. Exercise Harness interactive tools\n2. Validate resolver results",
    },
    { requestContext },
  );
  const approvalEvent = lastEvent("plan_approval_required");
  const displayWithPlan = harness.getDisplayState() as any;
  const displayPlanMatched =
    displayWithPlan.pendingPlanApproval?.planId === approvalEvent?.planId &&
    displayWithPlan.pendingPlanApproval?.title === "Fixture plan";
  await harness.respondToPlanApproval({
    planId: approvalEvent?.planId,
    response: { action: "approved" },
  });
  const approvalResult = await approvalResultPromise;

  const rejectionResultPromise = (submitPlanTool as any).execute(
    {
      title: "Rejected fixture plan",
      plan: "1. Submit an intentionally rejected plan",
    },
    { requestContext },
  );
  const rejectionEvent = lastEvent("plan_approval_required");
  await harness.respondToPlanApproval({
    planId: rejectionEvent?.planId,
    response: { action: "rejected", feedback: "Add verification." },
  });
  const rejectionResult = await rejectionResultPromise;

  output({
    questionEvent: questionEvent?.type === "ask_question",
    questionSelectionMode: questionEvent?.selectionMode,
    questionOptionCount: questionEvent?.options?.length,
    displayQuestion: displayQuestionMatched,
    questionAnswered:
      questionResult.isError === false && String(questionResult.content).includes("task tools, plans"),
    approvalEvent: approvalEvent?.type === "plan_approval_required",
    approvalTitle: approvalEvent?.title,
    displayPlan: displayPlanMatched,
    approvalAccepted:
      approvalResult.isError === false && String(approvalResult.content).includes("Plan approved"),
    rejectionEvent: rejectionEvent?.type === "plan_approval_required",
    rejectionReturned:
      rejectionResult.isError === false &&
      String(rejectionResult.content).includes("Plan was not approved") &&
      String(rejectionResult.content).includes("Add verification."),
    askEvents: events.filter((event) => event.type === "ask_question").length,
    planEvents: events.filter((event) => event.type === "plan_approval_required").length,
  });
} finally {
  unsubscribe();
  await harness.destroy();
}
