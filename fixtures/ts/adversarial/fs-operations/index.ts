import { fs, output } from "kit";

// Write, read, list, stat, delete — full filesystem lifecycle
fs.writeFileSync("adversarial-test.txt", "hello from adversarial");
const readData = fs.readFileSync("adversarial-test.txt", "utf8");
const list = fs.readdirSync(".");
const stat = fs.statSync("adversarial-test.txt");
fs.unlinkSync("adversarial-test.txt");

output({
  written: true,
  readData: readData,
  fileFound: list.length > 0,
  hasSize: stat.size > 0,
  deleted: true
});
