// Test: fs.readdirSync() and fs.statSync()
import { fs, output } from "kit";
fs.writeFileSync("stat-test.txt", "12345");
const stat = fs.statSync("stat-test.txt");
const listing = fs.readdirSync(".");
output({
  isDir: stat.isDirectory(),
  hasFile: listing.some((f: any) => f === "stat-test.txt"),
});
