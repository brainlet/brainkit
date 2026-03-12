// Experiment: QuickJS System Bridge
//
// Goal: Prove that JS running in QuickJS can call Go-provided file system,
// path, process, and child_process bridges. This validates the foundation
// for embedding Mastra (which needs Node.js-like system APIs) in Go via QuickJS.
//
// Tests:
//  1.  fs.readFile       - Read file contents from JS via Go os.ReadFile
//  2.  fs.writeFile      - Write file contents from JS via Go os.WriteFile
//  3.  fs.readdir        - List directory entries from JS via Go os.ReadDir
//  4.  fs.stat           - Get file metadata from JS via Go os.Stat
//  5.  fs.mkdir          - Create directories from JS via Go os.MkdirAll
//  6.  fs.unlink / fs.rm - Remove files/directories from JS via Go os.Remove/RemoveAll
//  7.  path.join         - Join path segments from JS via Go filepath.Join
//  8.  path.resolve      - Resolve absolute paths from JS via Go filepath.Abs
//  9.  path.dirname/basename/extname - Path component extraction
// 10.  child_process.exec - Execute commands and capture output
// 11.  child_process.spawn with streaming - Streaming process output line by line
// 12.  process.env        - Get/set environment variables
// 13.  process.cwd        - Get current working directory

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fastschema/qjs"
)

// spawnedProcess tracks a spawned child process for streaming reads.
type spawnedProcess struct {
	cmd       *exec.Cmd
	lines     chan string // buffered lines from stdout
	linesDone chan struct{}
	waitErr   chan error
}

var (
	spawnMu     sync.Mutex
	spawnNextID int
	spawnProcs  = map[int]*spawnedProcess{}
)

func main() {
	fmt.Println("=== QuickJS System Bridge Experiment ===")
	fmt.Println()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"fs.readFile", testFsReadFile},
		{"fs.writeFile", testFsWriteFile},
		{"fs.readdir", testFsReaddir},
		{"fs.stat", testFsStat},
		{"fs.mkdir", testFsMkdir},
		{"fs.unlink / fs.rm", testFsUnlinkRm},
		{"path.join", testPathJoin},
		{"path.resolve", testPathResolve},
		{"path.dirname/basename/extname", testPathComponents},
		{"child_process.exec", testExec},
		{"child_process.spawn streaming", testSpawnStreaming},
		{"process.env", testProcessEnv},
		{"process.cwd", testProcessCwd},
	}

	for i, t := range tests {
		fmt.Printf("--- Test %d: %s ---\n", i+1, t.name)
		if err := t.fn(); err != nil {
			log.Fatalf("FAILED: %v\n", err)
		}
		fmt.Println("PASS")
		fmt.Println()
	}

	fmt.Println("=== ALL TESTS PASSED ===")
}

// --------------------------------------------------------------------------
// Bridge registration helpers
// --------------------------------------------------------------------------

// registerFsBridges registers all __go_fs_* functions on the given context.
func registerFsBridges(ctx *qjs.Context) {
	ctx.SetFunc("__go_fs_readFile", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("readFile: path argument required")
		}
		path := args[0].String()
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("readFile: %w", err)
		}
		return this.Context().NewString(string(data)), nil
	})

	ctx.SetFunc("__go_fs_writeFile", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("writeFile: path and data arguments required")
		}
		path := args[0].String()
		data := args[1].String()
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			return nil, fmt.Errorf("writeFile: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_fs_readdir", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("readdir: path argument required")
		}
		dirPath := args[0].String()
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("readdir: %w", err)
		}
		type dirEntry struct {
			Name        string `json:"name"`
			IsDirectory bool   `json:"isDirectory"`
		}
		result := make([]dirEntry, len(entries))
		for i, e := range entries {
			result[i] = dirEntry{Name: e.Name(), IsDirectory: e.IsDir()}
		}
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("readdir: json marshal: %w", err)
		}
		return this.Context().NewString(string(jsonBytes)), nil
	})

	ctx.SetFunc("__go_fs_stat", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("stat: path argument required")
		}
		path := args[0].String()
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat: %w", err)
		}
		statResult := map[string]interface{}{
			"size":        info.Size(),
			"isFile":      info.Mode().IsRegular(),
			"isDirectory": info.IsDir(),
			"modTime":     info.ModTime().Unix(),
		}
		jsonBytes, err := json.Marshal(statResult)
		if err != nil {
			return nil, fmt.Errorf("stat: json marshal: %w", err)
		}
		return this.Context().NewString(string(jsonBytes)), nil
	})

	ctx.SetFunc("__go_fs_mkdir", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("mkdir: path argument required")
		}
		dirPath := args[0].String()
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("mkdir: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_fs_unlink", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("unlink: path argument required")
		}
		path := args[0].String()
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("unlink: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_fs_rm", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("rm: path argument required")
		}
		path := args[0].String()
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("rm: %w", err)
		}
		return this.Context().NewBool(true), nil
	})
}

// registerPathBridges registers all __go_path_* functions on the given context.
func registerPathBridges(ctx *qjs.Context) {
	ctx.SetFunc("__go_path_join", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.join: parts_json argument required")
		}
		partsJSON := args[0].String()
		var parts []string
		if err := json.Unmarshal([]byte(partsJSON), &parts); err != nil {
			return nil, fmt.Errorf("path.join: json unmarshal: %w", err)
		}
		return this.Context().NewString(filepath.Join(parts...)), nil
	})

	ctx.SetFunc("__go_path_resolve", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.resolve: parts_json argument required")
		}
		partsJSON := args[0].String()
		var parts []string
		if err := json.Unmarshal([]byte(partsJSON), &parts); err != nil {
			return nil, fmt.Errorf("path.resolve: json unmarshal: %w", err)
		}
		// Join then resolve to absolute
		joined := filepath.Join(parts...)
		abs, err := filepath.Abs(joined)
		if err != nil {
			return nil, fmt.Errorf("path.resolve: %w", err)
		}
		return this.Context().NewString(abs), nil
	})

	ctx.SetFunc("__go_path_dirname", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.dirname: path argument required")
		}
		return this.Context().NewString(filepath.Dir(args[0].String())), nil
	})

	ctx.SetFunc("__go_path_basename", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.basename: path argument required")
		}
		return this.Context().NewString(filepath.Base(args[0].String())), nil
	})

	ctx.SetFunc("__go_path_extname", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.extname: path argument required")
		}
		return this.Context().NewString(filepath.Ext(args[0].String())), nil
	})
}

// registerExecBridges registers __go_exec on the given context.
func registerExecBridges(ctx *qjs.Context) {
	ctx.SetFunc("__go_exec", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("exec: command argument required")
		}
		command := args[0].String()

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}

		var stdoutBuf, stderrBuf strings.Builder
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		exitCode := 0
		err := cmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, fmt.Errorf("exec: %w", err)
			}
		}

		result := map[string]interface{}{
			"stdout":   stdoutBuf.String(),
			"stderr":   stderrBuf.String(),
			"exitCode": exitCode,
		}
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("exec: json marshal: %w", err)
		}
		return this.Context().NewString(string(jsonBytes)), nil
	})
}

// registerSpawnBridges registers __go_spawn, __go_spawn_read, __go_spawn_wait.
func registerSpawnBridges(ctx *qjs.Context) {
	ctx.SetFunc("__go_spawn", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("spawn: command argument required")
		}
		command := args[0].String()
		var cmdArgs []string
		if len(args) >= 2 {
			argsJSON := args[1].String()
			if err := json.Unmarshal([]byte(argsJSON), &cmdArgs); err != nil {
				return nil, fmt.Errorf("spawn: json unmarshal args: %w", err)
			}
		}

		cmd := exec.Command(command, cmdArgs...)
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("spawn: stdout pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("spawn: start: %w", err)
		}

		proc := &spawnedProcess{
			cmd:       cmd,
			lines:     make(chan string, 256),
			linesDone: make(chan struct{}),
			waitErr:   make(chan error, 1),
		}

		// Read all stdout lines into the channel in a goroutine.
		// This must complete before cmd.Wait() is called, because
		// Wait closes the pipes.
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				proc.lines <- scanner.Text()
			}
			close(proc.lines)
			close(proc.linesDone)
		}()

		// Wait for the process in another goroutine, but only after
		// stdout has been fully drained.
		go func() {
			<-proc.linesDone
			proc.waitErr <- cmd.Wait()
		}()

		spawnMu.Lock()
		id := spawnNextID
		spawnNextID++
		spawnProcs[id] = proc
		spawnMu.Unlock()

		return this.Context().NewInt32(int32(id)), nil
	})

	ctx.SetFunc("__go_spawn_read", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("spawn_read: id argument required")
		}
		id := int(args[0].Int32())

		spawnMu.Lock()
		proc, ok := spawnProcs[id]
		spawnMu.Unlock()
		if !ok {
			return nil, fmt.Errorf("spawn_read: no process with id %d", id)
		}

		line, ok := <-proc.lines
		if !ok {
			// Channel closed, no more lines
			return this.Context().NewNull(), nil
		}
		return this.Context().NewString(line), nil
	})

	ctx.SetFunc("__go_spawn_wait", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("spawn_wait: id argument required")
		}
		id := int(args[0].Int32())

		spawnMu.Lock()
		proc, ok := spawnProcs[id]
		spawnMu.Unlock()
		if !ok {
			return nil, fmt.Errorf("spawn_wait: no process with id %d", id)
		}

		waitErr := <-proc.waitErr

		exitCode := 0
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		// Clean up
		spawnMu.Lock()
		delete(spawnProcs, id)
		spawnMu.Unlock()

		return this.Context().NewInt32(int32(exitCode)), nil
	})
}

// registerProcessBridges registers __go_process_env, __go_process_env_set, __go_process_cwd.
func registerProcessBridges(ctx *qjs.Context) {
	ctx.SetFunc("__go_process_env", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("process.env: key argument required")
		}
		key := args[0].String()
		val := os.Getenv(key)
		return this.Context().NewString(val), nil
	})

	ctx.SetFunc("__go_process_env_set", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("process.env.set: key and value arguments required")
		}
		key := args[0].String()
		val := args[1].String()
		if err := os.Setenv(key, val); err != nil {
			return nil, fmt.Errorf("process.env.set: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_process_cwd", func(this *qjs.This) (*qjs.Value, error) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("process.cwd: %w", err)
		}
		return this.Context().NewString(cwd), nil
	})
}

// registerAllBridges registers every bridge function on the context.
func registerAllBridges(ctx *qjs.Context) {
	registerFsBridges(ctx)
	registerPathBridges(ctx)
	registerExecBridges(ctx)
	registerSpawnBridges(ctx)
	registerProcessBridges(ctx)
}

// newRuntime creates a new qjs runtime+context with all bridges registered.
func newRuntime() (*qjs.Runtime, *qjs.Context, error) {
	rt, err := qjs.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create runtime: %w", err)
	}
	ctx := rt.Context()
	registerAllBridges(ctx)
	return rt, ctx, nil
}

// --------------------------------------------------------------------------
// Test 1: fs.readFile
// --------------------------------------------------------------------------
func testFsReadFile() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	// Create a temp file with known contents
	tmpDir, err := os.MkdirTemp("", "qjs-test-readfile-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "Hello from Go! Line 1\nLine 2\nLine 3"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}

	// JS reads the file
	jsCode := fmt.Sprintf(`
		const content = __go_fs_readFile(%q, "utf8");
		content;
	`, testFile)

	result, err := ctx.Eval("test_readfile.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	got := result.String()
	if got != testContent {
		return fmt.Errorf("content mismatch:\n  expected: %q\n  got:      %q", testContent, got)
	}
	fmt.Printf("  Read %d bytes, content matches\n", len(got))
	return nil
}

// --------------------------------------------------------------------------
// Test 2: fs.writeFile
// --------------------------------------------------------------------------
func testFsWriteFile() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	tmpDir, err := os.MkdirTemp("", "qjs-test-writefile-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "output.txt")
	testContent := "Written from JavaScript via Go bridge!"

	jsCode := fmt.Sprintf(`
		__go_fs_writeFile(%q, %q, "utf8");
	`, testFile, testContent)

	result, err := ctx.Eval("test_writefile.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	// Verify from Go side
	data, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Errorf("failed to read back file: %w", err)
	}
	if string(data) != testContent {
		return fmt.Errorf("content mismatch:\n  expected: %q\n  got:      %q", testContent, string(data))
	}
	fmt.Printf("  Wrote %d bytes, content verified\n", len(data))
	return nil
}

// --------------------------------------------------------------------------
// Test 3: fs.readdir
// --------------------------------------------------------------------------
func testFsReaddir() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	tmpDir, err := os.MkdirTemp("", "qjs-test-readdir-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files and subdirectories
	os.WriteFile(filepath.Join(tmpDir, "alpha.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "beta.txt"), []byte("b"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	jsCode := fmt.Sprintf(`
		const entriesJSON = __go_fs_readdir(%q);
		const entries = JSON.parse(entriesJSON);
		JSON.stringify({
			count: entries.length,
			names: entries.map(e => e.name).sort(),
			hasDirs: entries.some(e => e.isDirectory),
			hasFiles: entries.some(e => !e.isDirectory),
			subdirIsDir: entries.find(e => e.name === "subdir").isDirectory,
			alphaIsFile: !entries.find(e => e.name === "alpha.txt").isDirectory,
		});
	`, tmpDir)

	result, err := ctx.Eval("test_readdir.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}
	if parsed["count"].(float64) != 3 {
		return fmt.Errorf("expected 3 entries, got %v", parsed["count"])
	}
	if parsed["subdirIsDir"] != true {
		return fmt.Errorf("expected subdir to be a directory")
	}
	if parsed["alphaIsFile"] != true {
		return fmt.Errorf("expected alpha.txt to be a file")
	}
	return nil
}

// --------------------------------------------------------------------------
// Test 4: fs.stat
// --------------------------------------------------------------------------
func testFsStat() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	tmpDir, err := os.MkdirTemp("", "qjs-test-stat-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "stat test content with known size"
	testFile := filepath.Join(tmpDir, "statfile.txt")
	os.WriteFile(testFile, []byte(testContent), 0644)

	jsCode := fmt.Sprintf(`
		const statJSON = __go_fs_stat(%q);
		const stat = JSON.parse(statJSON);
		JSON.stringify({
			size: stat.size,
			isFile: stat.isFile,
			isDirectory: stat.isDirectory,
			hasModTime: stat.modTime > 0,
		});
	`, testFile)

	result, err := ctx.Eval("test_stat.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}
	expectedSize := float64(len(testContent))
	if parsed["size"].(float64) != expectedSize {
		return fmt.Errorf("expected size %v, got %v", expectedSize, parsed["size"])
	}
	if parsed["isFile"] != true {
		return fmt.Errorf("expected isFile=true")
	}
	if parsed["isDirectory"] != false {
		return fmt.Errorf("expected isDirectory=false")
	}
	if parsed["hasModTime"] != true {
		return fmt.Errorf("expected modTime > 0")
	}

	// Also test stat on directory
	dirStatCode := fmt.Sprintf(`
		const ds = JSON.parse(__go_fs_stat(%q));
		JSON.stringify({ isDir: ds.isDirectory, isFile: ds.isFile });
	`, tmpDir)
	result2, err := ctx.Eval("test_stat_dir.js", qjs.Code(dirStatCode))
	if err != nil {
		return fmt.Errorf("dir stat eval failed: %w", err)
	}
	defer result2.Free()

	var dirParsed map[string]interface{}
	json.Unmarshal([]byte(result2.String()), &dirParsed)
	if dirParsed["isDir"] != true {
		return fmt.Errorf("expected directory isDir=true")
	}
	fmt.Printf("  Directory stat: %s\n", result2.String())
	return nil
}

// --------------------------------------------------------------------------
// Test 5: fs.mkdir
// --------------------------------------------------------------------------
func testFsMkdir() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	tmpDir, err := os.MkdirTemp("", "qjs-test-mkdir-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	nestedPath := filepath.Join(tmpDir, "a", "b", "c")
	jsCode := fmt.Sprintf(`
		__go_fs_mkdir(%q, true);
		const stat = JSON.parse(__go_fs_stat(%q));
		JSON.stringify({ exists: stat.isDirectory });
	`, nestedPath, nestedPath)

	result, err := ctx.Eval("test_mkdir.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)
	if parsed["exists"] != true {
		return fmt.Errorf("expected nested directory to exist")
	}

	// Verify from Go side
	info, err := os.Stat(nestedPath)
	if err != nil {
		return fmt.Errorf("nested dir does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory, got file")
	}
	fmt.Printf("  Created nested path: %s\n", nestedPath)
	return nil
}

// --------------------------------------------------------------------------
// Test 6: fs.unlink / fs.rm
// --------------------------------------------------------------------------
func testFsUnlinkRm() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	tmpDir, err := os.MkdirTemp("", "qjs-test-rm-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file and a directory with contents
	unlinkFile := filepath.Join(tmpDir, "to_unlink.txt")
	os.WriteFile(unlinkFile, []byte("delete me"), 0644)

	rmDir := filepath.Join(tmpDir, "to_remove")
	os.MkdirAll(filepath.Join(rmDir, "nested"), 0755)
	os.WriteFile(filepath.Join(rmDir, "nested", "file.txt"), []byte("deep"), 0644)

	// Test unlink (single file)
	jsCode := fmt.Sprintf(`
		__go_fs_unlink(%q);
		let unlinkGone = false;
		try {
			__go_fs_stat(%q);
		} catch(e) {
			unlinkGone = true;
		}

		// Test rm (recursive directory)
		__go_fs_rm(%q);
		let rmGone = false;
		try {
			__go_fs_stat(%q);
		} catch(e) {
			rmGone = true;
		}

		JSON.stringify({ unlinkGone: unlinkGone, rmGone: rmGone });
	`, unlinkFile, unlinkFile, rmDir, rmDir)

	result, err := ctx.Eval("test_rm.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)
	if parsed["unlinkGone"] != true {
		return fmt.Errorf("expected unlinked file to be gone")
	}
	if parsed["rmGone"] != true {
		return fmt.Errorf("expected rm'd directory to be gone")
	}

	// Verify from Go side
	if _, err := os.Stat(unlinkFile); !os.IsNotExist(err) {
		return fmt.Errorf("unlinked file still exists")
	}
	if _, err := os.Stat(rmDir); !os.IsNotExist(err) {
		return fmt.Errorf("rm'd directory still exists")
	}
	return nil
}

// --------------------------------------------------------------------------
// Test 7: path.join
// --------------------------------------------------------------------------
func testPathJoin() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	jsCode := `
		const tests = [
			{ parts: ["/usr", "local", "bin"], expected: "/usr/local/bin" },
			{ parts: ["foo", "bar", "baz.txt"], expected: "foo/bar/baz.txt" },
			{ parts: ["/a", "b", "..", "c"], expected: "/a/c" },
			{ parts: [".", "src", "main.go"], expected: "src/main.go" },
		];

		const results = tests.map(t => {
			const got = __go_path_join(JSON.stringify(t.parts));
			return { parts: t.parts, expected: t.expected, got: got, pass: got === t.expected };
		});

		JSON.stringify({ all_pass: results.every(r => r.pass), results: results });
	`

	result, err := ctx.Eval("test_path_join.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)

	results := parsed["results"].([]interface{})
	for _, r := range results {
		m := r.(map[string]interface{})
		status := "PASS"
		if m["pass"] != true {
			status = "FAIL"
		}
		fmt.Printf("  [%s] join(%v) = %q (expected %q)\n", status, m["parts"], m["got"], m["expected"])
	}

	if parsed["all_pass"] != true {
		return fmt.Errorf("some path.join tests failed")
	}
	return nil
}

// --------------------------------------------------------------------------
// Test 8: path.resolve
// --------------------------------------------------------------------------
func testPathResolve() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	cwd, _ := os.Getwd()

	jsCode := `
		const r1 = __go_path_resolve(JSON.stringify([".", "src"]));
		const r2 = __go_path_resolve(JSON.stringify(["/absolute", "path"]));
		JSON.stringify({ relative: r1, absolute: r2 });
	`

	result, err := ctx.Eval("test_path_resolve.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)

	expectedRelative := filepath.Join(cwd, "src")
	if parsed["relative"].(string) != expectedRelative {
		return fmt.Errorf("expected relative resolve to %q, got %q", expectedRelative, parsed["relative"])
	}
	if parsed["absolute"].(string) != filepath.Join("/absolute", "path") {
		return fmt.Errorf("expected absolute path, got %q", parsed["absolute"])
	}
	return nil
}

// --------------------------------------------------------------------------
// Test 9: path.dirname/basename/extname
// --------------------------------------------------------------------------
func testPathComponents() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	jsCode := `
		const testPath = "/usr/local/bin/program.tar.gz";
		const dir = __go_path_dirname(testPath);
		const base = __go_path_basename(testPath);
		const ext = __go_path_extname(testPath);

		const testPath2 = "/just/a/directory/";
		const dir2 = __go_path_dirname(testPath2);
		const base2 = __go_path_basename(testPath2);
		const ext2 = __go_path_extname(testPath2);

		JSON.stringify({
			dir: dir, base: base, ext: ext,
			dir2: dir2, base2: base2, ext2: ext2,
		});
	`

	result, err := ctx.Eval("test_path_components.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)

	checks := []struct {
		key      string
		expected string
	}{
		{"dir", "/usr/local/bin"},
		{"base", "program.tar.gz"},
		{"ext", ".gz"},
		{"dir2", "/just/a/directory"},
		{"base2", "directory"},
		{"ext2", ""},
	}

	for _, c := range checks {
		got := parsed[c.key].(string)
		if got != c.expected {
			return fmt.Errorf("  %s: expected %q, got %q", c.key, c.expected, got)
		}
		fmt.Printf("  %s = %q\n", c.key, got)
	}
	return nil
}

// --------------------------------------------------------------------------
// Test 10: child_process.exec
// --------------------------------------------------------------------------
func testExec() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	// Test echo command
	jsCode := `
		const r1 = JSON.parse(__go_exec("echo hello world"));
		const r2 = JSON.parse(__go_exec("ls /tmp"));

		JSON.stringify({
			echoStdout: r1.stdout.trim(),
			echoExit: r1.exitCode,
			lsHasOutput: r2.stdout.length > 0,
			lsExit: r2.exitCode,
		});
	`

	result, err := ctx.Eval("test_exec.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)

	if parsed["echoStdout"].(string) != "hello world" {
		return fmt.Errorf("expected echo output 'hello world', got %q", parsed["echoStdout"])
	}
	if parsed["echoExit"].(float64) != 0 {
		return fmt.Errorf("expected exit code 0, got %v", parsed["echoExit"])
	}
	if parsed["lsHasOutput"] != true {
		return fmt.Errorf("expected ls to produce output")
	}

	// Test non-zero exit code
	jsCode2 := `
		const r = JSON.parse(__go_exec("exit 42"));
		JSON.stringify({ exitCode: r.exitCode });
	`
	result2, err := ctx.Eval("test_exec2.js", qjs.Code(jsCode2))
	if err != nil {
		return fmt.Errorf("eval failed for exit code test: %w", err)
	}
	defer result2.Free()

	var parsed2 map[string]interface{}
	json.Unmarshal([]byte(result2.String()), &parsed2)
	if parsed2["exitCode"].(float64) != 42 {
		return fmt.Errorf("expected exit code 42, got %v", parsed2["exitCode"])
	}
	fmt.Printf("  Non-zero exit code: %v\n", parsed2["exitCode"])
	return nil
}

// --------------------------------------------------------------------------
// Test 11: child_process.spawn with streaming
// --------------------------------------------------------------------------
func testSpawnStreaming() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	// Use printf to produce multiple lines (portable across macOS/Linux)
	jsCode := `
		const id = __go_spawn("sh", JSON.stringify(["-c", "for i in 1 2 3 4 5; do echo line_$i; done"]));
		const lines = [];
		while (true) {
			const line = __go_spawn_read(id);
			if (line === null) break;
			lines.push(line);
		}
		const exitCode = __go_spawn_wait(id);
		JSON.stringify({ lines: lines, exitCode: exitCode, count: lines.length });
	`

	result, err := ctx.Eval("test_spawn.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)

	if parsed["count"].(float64) != 5 {
		return fmt.Errorf("expected 5 lines, got %v", parsed["count"])
	}

	lines := parsed["lines"].([]interface{})
	for i, l := range lines {
		expected := fmt.Sprintf("line_%d", i+1)
		if l.(string) != expected {
			return fmt.Errorf("line %d: expected %q, got %q", i, expected, l)
		}
	}

	if parsed["exitCode"].(float64) != 0 {
		return fmt.Errorf("expected exit code 0, got %v", parsed["exitCode"])
	}
	fmt.Printf("  Streamed %v lines successfully\n", parsed["count"])
	return nil
}

// --------------------------------------------------------------------------
// Test 12: process.env
// --------------------------------------------------------------------------
func testProcessEnv() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	jsCode := `
		// Read existing env var
		const pathVal = __go_process_env("PATH");

		// Set a custom env var from JS
		__go_process_env_set("QJS_TEST_VAR", "hello_from_js");
		const customVal = __go_process_env("QJS_TEST_VAR");

		// Read a var that doesn't exist
		const missingVal = __go_process_env("QJS_NONEXISTENT_VAR_12345");

		JSON.stringify({
			pathNotEmpty: pathVal.length > 0,
			customVal: customVal,
			missingVal: missingVal,
		});
	`

	result, err := ctx.Eval("test_env.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(str), &parsed)

	if parsed["pathNotEmpty"] != true {
		return fmt.Errorf("expected PATH to be non-empty")
	}
	if parsed["customVal"].(string) != "hello_from_js" {
		return fmt.Errorf("expected custom var to be 'hello_from_js', got %q", parsed["customVal"])
	}
	if parsed["missingVal"].(string) != "" {
		return fmt.Errorf("expected missing var to be empty string, got %q", parsed["missingVal"])
	}

	// Verify from Go side
	if os.Getenv("QJS_TEST_VAR") != "hello_from_js" {
		return fmt.Errorf("Go side: QJS_TEST_VAR not set correctly")
	}
	// Clean up
	os.Unsetenv("QJS_TEST_VAR")
	return nil
}

// --------------------------------------------------------------------------
// Test 13: process.cwd
// --------------------------------------------------------------------------
func testProcessCwd() error {
	rt, ctx, err := newRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()

	jsCode := `
		const cwd = __go_process_cwd();
		cwd;
	`

	result, err := ctx.Eval("test_cwd.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	jsCwd := result.String()
	goCwd, _ := os.Getwd()

	fmt.Printf("  JS cwd:  %s\n", jsCwd)
	fmt.Printf("  Go cwd:  %s\n", goCwd)

	if jsCwd != goCwd {
		return fmt.Errorf("cwd mismatch: JS=%q Go=%q", jsCwd, goCwd)
	}
	return nil
}
