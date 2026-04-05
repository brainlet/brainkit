// Test: fs.writeFileSync() then fs.readFileSync() roundtrip
import { fs, output } from "kit";
fs.writeFileSync("test-file.txt", "hello from fixture");
const data = fs.readFileSync("test-file.txt", "utf8");
output({ written: true, matches: data === "hello from fixture" });
