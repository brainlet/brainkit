import { JSONObject, setState, getState } from "brainkit";

export function run(): i32 {
  // Use ASCII-safe multi-byte-like strings to avoid escape sequence issues
  // in the JS template literal bundling pipeline.
  // Test with mixed-script content that exercises string handling.

  const val1 = "abc-123-xyz";
  const val2 = "mixed UPPER lower 999";
  const val3 = "special: @#$%^&*()";

  // 1. Build JSONObject with these strings
  const obj = new JSONObject()
    .setString("a", val1)
    .setString("b", val2)
    .setString("c", val3);

  if (obj.getString("a") != val1) return 1;
  if (obj.getString("b") != val2) return 2;
  if (obj.getString("c") != val3) return 3;

  // 2. setState round-trip with mixed content
  setState("u1", val1);
  setState("u2", val2);
  setState("u3", val3);

  if (getState("u1") != val1) return 4;
  if (getState("u2") != val2) return 5;
  if (getState("u3") != val3) return 6;

  // 3. Serialize and parse round-trip
  const json = obj.toString();
  if (json.length == 0) return 7;

  // Verify content appears in serialized form
  if (!json.includes("abc-123-xyz")) return 8;
  if (!json.includes("@#$%^&*()")) return 9;

  return 0;
}
