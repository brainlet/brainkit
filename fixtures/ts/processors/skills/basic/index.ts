import { SkillsProcessor } from "agent";
import { output } from "kit";

const proto: any = (SkillsProcessor as any).prototype;
output({
  id: "skills-processor",
  hasProcessInputStep: typeof proto.processInputStep === "function",
});
