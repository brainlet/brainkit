import { tools, output } from "kit";
const result = await tools.call("echo", { message: "from-deploy-init" });
output({ toolResult: result, calledDuringInit: true });
