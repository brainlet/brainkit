import { JSONValue, JSONObject, JSONArray, log } from "wasm";

export function run(): i32 {
  // 1. Build source object with a users array
  const alice = new JSONObject().setString("name", "Alice");
  const bob = new JSONObject().setString("name", "Bob");
  const users = new JSONArray().pushObject(alice).pushObject(bob);
  const source = new JSONObject().setArray("users", users);

  // 2. Serialize to string
  const json = source.toString();
  log("source: " + json);
  if (!json.includes("Alice")) return 1;
  if (!json.includes("Bob")) return 2;

  // 3. Parse back from string
  const parsed = JSONValue.parse(json);
  if (parsed.isNull()) return 3;

  // 4. Extract the users array from parsed object
  const obj = parsed.asObject();
  const parsedUsers = obj.getArray("users");
  if (parsedUsers.length != 2) return 4;

  // 5. Build a new array of just names
  const names = new JSONArray();
  for (let i: i32 = 0; i < parsedUsers.length; i++) {
    const user = parsedUsers.at(i).asObject();
    const name = user.getString("name");
    names.pushString(name);
  }

  // 6. Verify the transformed result
  const namesJson = names.toString();
  log("names: " + namesJson);
  if (!namesJson.includes("Alice")) return 5;
  if (!namesJson.includes("Bob")) return 6;

  return 0;
}
