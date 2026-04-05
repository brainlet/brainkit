# FS Test Map

**Purpose:** Verifies the Node.js fs polyfill (writeFileSync, readFileSync, mkdirSync, statSync, unlinkSync, readdirSync) exposed to JS/TS code
**Tests:** 12 functions across 1 file
**Entry point:** `fs_test.go` → `Run(t, env)`
**Campaigns:** transport (all 5)

## Files

### operations.go — Filesystem polyfill operations via EvalTS

| Function | Purpose |
|----------|---------|
| testWriteReadRoundtrip | Writes "hello fs" to test.txt via writeFileSync and reads it back via readFileSync, asserts exact match |
| testWriteOverwrite | Writes "v1" then "v2" to the same file, reads back, asserts "v2" (overwrite semantics) |
| testMkdirRecursive | Creates nested dirs a/b/c with recursive:true, writes and reads a file at the deepest level |
| testStatFile | Writes "12345" to a file, calls statSync, asserts size=5 and isDirectory()=false |
| testStatDirectory | Creates a directory, calls statSync, asserts isDirectory()=true |
| testDelete | Writes a file, deletes it with unlinkSync, attempts readFileSync and asserts it throws an error |
| testDeleteNotFound | Calls unlinkSync on a nonexistent file and asserts it throws (no panic) |
| testReadNotFound | Calls readFileSync on a nonexistent file and asserts it throws (no panic) |
| testPathTraversalRejected | Attempts readFileSync("../../etc/passwd") and asserts the workspace escape is rejected with an error |
| testLargeFileWrite | Writes a 1MB string to a file, stats it, asserts size equals 1048576 bytes |
| testFSListWithPattern | Creates 3 files in a directory, calls readdirSync, asserts 3 entries returned |
| testFSFromTS | Deploys .ts that exercises write, read, stat, readdir, unlink in sequence and verifies all via output() |

## Cross-references

- **Campaigns:** `transport/{sqlite,nats,postgres,redis,amqp}_test.go`
- **Related domains:** deploy (fs operations during deploy), bus/cross_feature (testCrossHandlerWritesFS)
- **Fixtures:** none
