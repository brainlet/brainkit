import { emit, JSONObject, log } from "brainkit";

export function run(): i32 {
  // Publish 5 events with incrementing index
  for (let i: i32 = 0; i < 5; i++) {
    const payload = new JSONObject()
      .setInt("index", i)
      .setString("ts", "now");
    emit("event." + i.toString(), payload.toString());
    log("published event." + i.toString());
  }

  return 0;
}
