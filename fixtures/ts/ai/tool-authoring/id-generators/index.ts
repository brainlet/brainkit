// AI SDK id generation: `generateId()` produces a random id on each
// call; `createIdGenerator({prefix, size, alphabet})` builds a
// configurable factory. Used for correlating tool calls, messages,
// and cache keys across retries.
import { generateId, createIdGenerator } from "ai";
import { output } from "kit";

const id1 = generateId();
const id2 = generateId();

const genPrefixed = createIdGenerator({ prefix: "msg", size: 12 });
const pid1 = genPrefixed();
const pid2 = genPrefixed();

// Custom alphabet — lowercase hex only.
const genHex = createIdGenerator({ alphabet: "0123456789abcdef", size: 16 });
const hex1 = genHex();
const hex2 = genHex();

output({
  hasTwoIds: !!id1 && !!id2 && id1 !== id2,
  id1Length: id1.length,
  prefixedStartsWithPrefix:
    pid1.startsWith("msg") && pid2.startsWith("msg"),
  prefixedUnique: pid1 !== pid2,
  hexMatchesAlphabet: /^[0-9a-f]+$/.test(hex1) && /^[0-9a-f]+$/.test(hex2),
  hexLength: hex1.length,
});
