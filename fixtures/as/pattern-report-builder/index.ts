import { setState, getState, emit, log, JSONObject, JSONArray } from "brainkit";

export function run(): i32 {
  // 1. Seed state with data points
  setState("app.name", "brainkit");
  setState("app.version", "1.0.0");
  setState("app.status", "healthy");

  // 2. Read state back to build report
  const name = getState("app.name");
  const version = getState("app.version");
  const status = getState("app.status");

  if (name != "brainkit") return 1;
  if (version != "1.0.0") return 2;
  if (status != "healthy") return 3;

  // 3. Build the report object with aggregated data
  const checks = new JSONArray()
    .pushString("state-ok")
    .pushString("tools-ok")
    .pushString("bus-ok");

  const report = new JSONObject()
    .setString("name", name)
    .setString("version", version)
    .setString("status", status)
    .setInt("checkCount", 3)
    .setArray("checks", checks)
    .setBool("passing", true);

  // 4. Serialize and verify
  const json = report.toString();
  log("report: " + json);
  if (!json.includes("brainkit")) return 4;
  if (!json.includes("healthy")) return 5;

  // 5. Publish the report
  emit("report.generated", report.toString());

  return 0;
}
