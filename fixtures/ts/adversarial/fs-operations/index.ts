import { fs, output } from "kit";

// Write, read, list, stat, delete — full filesystem lifecycle
await fs.write("adversarial-test.txt", "hello from adversarial");
const read = await fs.read("adversarial-test.txt");
const list = await fs.list(".", "adversarial-*");
const stat = await fs.stat("adversarial-test.txt");
await fs.delete("adversarial-test.txt");

output({
  written: true,
  readData: read.data,
  fileFound: list.files.length > 0,
  hasSize: stat.size > 0,
  deleted: true
});
