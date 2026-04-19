// Test: globalThis.FormData polyfill — append, get, getAll,
// has, delete, entries iteration.
import { output } from "kit";

const fd = new FormData();
fd.append("name", "alice");
fd.append("tags", "first");
fd.append("tags", "second");
fd.set("name", "bob");

const entries: Array<[string, any]> = [];
for (const [k, v] of fd.entries()) entries.push([k, v]);

output({
  hasClass: typeof FormData === "function",
  nameValue: fd.get("name"),
  tagsAll: fd.getAll("tags").length === 2,
  hasName: fd.has("name"),
  entryCount: entries.length,
  deleteWorks: (fd.delete("name"), fd.has("name") === false),
});
