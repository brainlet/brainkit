import { SkillSearchProcessor } from "agent";
import { output } from "kit";

const proto: any = (SkillSearchProcessor as any).prototype;
output({
  id: "skill-search",
  hasProcessInputStep: typeof proto.processInputStep === "function",
});
