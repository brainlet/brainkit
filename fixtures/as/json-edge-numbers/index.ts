import { JSONValue } from "brainkit";

export function run(): i32 {
  // 1. Zero
  const v0 = JSONValue.parse("0");
  if (!v0.isNumber()) return 1;
  if (v0.asNumber() != 0.0) return 2;

  // 2. Negative integer
  const v1 = JSONValue.parse("-1");
  if (!v1.isNumber()) return 3;
  if (v1.asInt() != -1) return 4;

  // 3. Positive integer
  const v2 = JSONValue.parse("42");
  if (!v2.isNumber()) return 5;
  if (v2.asInt() != 42) return 6;

  // 4. Decimal
  const v3 = JSONValue.parse("3.14");
  if (!v3.isNumber()) return 7;
  const n3 = v3.asNumber();
  if (n3 < 3.13 || n3 > 3.15) return 8;

  // 5. Negative decimal
  const v4 = JSONValue.parse("-0.5");
  if (!v4.isNumber()) return 9;
  if (v4.asNumber() != -0.5) return 10;

  // 6. Scientific notation (large)
  const v5 = JSONValue.parse("1e10");
  if (!v5.isNumber()) return 11;
  if (v5.asNumber() != 1e10) return 12;

  // 7. Scientific notation (small)
  const v6 = JSONValue.parse("1.5e-3");
  if (!v6.isNumber()) return 13;
  const n6 = v6.asNumber();
  if (n6 < 0.0014 || n6 > 0.0016) return 14;

  return 0;
}
