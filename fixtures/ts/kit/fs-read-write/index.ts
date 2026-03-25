// Test: fs.write() then fs.read() roundtrip
import { fs, output } from "kit";
await fs.write("test-file.txt", "hello from fixture");
const result = await fs.read("test-file.txt");
output({ written: true, data: result.data, matches: result.data === "hello from fixture" });
