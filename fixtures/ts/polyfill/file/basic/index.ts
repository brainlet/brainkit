// Test: File extends Blob, carries a name + lastModified.
import { output } from "kit";

const file = new File(["abc"], "hello.txt", { type: "text/plain", lastModified: 1700000000000 });
const text = await file.text();

output({
  hasFile: typeof File === "function",
  extendsBlob: file instanceof Blob,
  name: file.name,
  type: file.type,
  lastModified: file.lastModified,
  text,
});
