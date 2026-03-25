// Test: fs.list() and fs.stat()
import { fs, output } from "kit";
await fs.write("stat-test.txt", "12345");
const stat = await fs.stat("stat-test.txt");
const listing = await fs.list(".");
output({
  size: stat.size,
  isDir: stat.isDir,
  fileCount: listing.files.length,
  hasFile: listing.files.some((f: any) => f.name === "stat-test.txt"),
});
