// Test: Basic filesystem operations via Go bridge (globalThis.fs from jsbridge/fs.go)
import { output } from "brainlet";

try {
  // globalThis.fs is set by jsbridge/fs.go
  const fs = globalThis.fs;
  if (!fs) throw new Error("globalThis.fs not available");

  // path is available as a local helper (not a module at runtime)
  function pathJoin(...parts) { return parts.filter(Boolean).join("/").replace(/\/\/+/g, "/"); }

  const tmpDir = globalThis.process?.env?.TEST_TMPDIR || "/tmp/brainkit-fs-test-" + Date.now();
  await fs.mkdir(tmpDir, { recursive: true });

  // Write
  const testFile = pathJoin(tmpDir, "hello.txt");
  await fs.writeFile(testFile, "Hello from brainkit!");

  // Read
  const content = await fs.readFile(testFile, "utf8");

  // Stat
  const stat = await fs.stat(testFile);

  // Readdir
  const entries = await fs.readdir(tmpDir);

  // Now test the new bridges via __go_fs_* directly
  // appendFile
  await __go_fs_appendFile(testFile, " Extra.");
  const appended = await fs.readFile(testFile, "utf8");

  // copyFile
  const copyDest = pathJoin(tmpDir, "copy.txt");
  await __go_fs_copyFile(testFile, copyDest);
  const copyContent = await fs.readFile(copyDest, "utf8");

  // rename
  const renamedDest = pathJoin(tmpDir, "renamed.txt");
  await __go_fs_rename(copyDest, renamedDest);
  const renamedContent = await fs.readFile(renamedDest, "utf8");

  // lstat
  const lstatResult = JSON.parse(await __go_fs_lstat(testFile));

  // realpath
  const realp = await __go_fs_realpath(tmpDir);

  // access (should not throw for existing file)
  await __go_fs_access(testFile);

  // unlink
  await fs.unlink(renamedDest);
  const entriesAfter = await fs.readdir(tmpDir);

  // rm recursive
  await fs.rm(tmpDir, { recursive: true });

  output({
    content: content,
    appendedContent: appended,
    statIsFile: stat.isFile,
    statIsDirectory: stat.isDirectory,
    statSize: stat.size,
    entries: entries.map(e => typeof e === "string" ? e : e.name),
    copyContent: copyContent,
    renamedContent: renamedContent,
    lstatIsFile: lstatResult.isFile,
    realpath: realp,
    entriesAfterDelete: entriesAfter.map(e => typeof e === "string" ? e : e.name),
    success: content === "Hello from brainkit!" && appended === "Hello from brainkit! Extra.",
  });
} catch(e) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 1500) });
}
