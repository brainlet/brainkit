package jsbridge

import (
	"fmt"
	"os"
	"runtime"

	quickjs "github.com/buke/quickjs-go"
)

// OSPolyfill provides Node.js os module (platform, arch, tmpdir, homedir, etc.).
// Go-backed — returns real system values from Go's runtime and os packages.
type OSPolyfill struct{}

// OS creates a Node.js os module polyfill.
func OS() *OSPolyfill { return &OSPolyfill{} }

func (p *OSPolyfill) Name() string { return "os" }

func (p *OSPolyfill) Setup(ctx *quickjs.Context) error {
	// Go-backed functions for real system values
	ctx.Globals().Set("__go_os_platform", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewString(runtime.GOOS)
	}))
	ctx.Globals().Set("__go_os_arch", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		goarch := runtime.GOARCH
		// Map Go arch names to Node.js arch names
		switch goarch {
		case "amd64":
			return ctx.NewString("x64")
		case "386":
			return ctx.NewString("ia32")
		case "arm64":
			return ctx.NewString("arm64")
		case "arm":
			return ctx.NewString("arm")
		default:
			return ctx.NewString(goarch)
		}
	}))
	ctx.Globals().Set("__go_os_tmpdir", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewString(os.TempDir())
	}))
	ctx.Globals().Set("__go_os_homedir", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		dir, err := os.UserHomeDir()
		if err != nil {
			return ctx.NewString("")
		}
		return ctx.NewString(dir)
	}))
	ctx.Globals().Set("__go_os_hostname", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		name, err := os.Hostname()
		if err != nil {
			return ctx.NewString("localhost")
		}
		return ctx.NewString(name)
	}))
	ctx.Globals().Set("__go_os_cpus", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewInt32(int32(runtime.NumCPU()))
	}))
	ctx.Globals().Set("__go_os_type", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		switch runtime.GOOS {
		case "darwin":
			return ctx.NewString("Darwin")
		case "linux":
			return ctx.NewString("Linux")
		case "windows":
			return ctx.NewString("Windows_NT")
		default:
			return ctx.NewString(runtime.GOOS)
		}
	}))

	eol := "\n"
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}

	return evalJS(ctx, fmt.Sprintf(`
globalThis.__node_os = {
  platform: function() { return __go_os_platform(); },
  arch: function() { return __go_os_arch(); },
  tmpdir: function() { return __go_os_tmpdir(); },
  homedir: function() { return __go_os_homedir(); },
  hostname: function() { return __go_os_hostname(); },
  type: function() { return __go_os_type(); },
  cpus: function() {
    var n = __go_os_cpus();
    var arr = [];
    for (var i = 0; i < n; i++) arr.push({ model: "cpu", speed: 0 });
    return arr;
  },
  EOL: %q,
  endianness: function() { return "LE"; },
  release: function() { return "0.0.0"; },
  totalmem: function() { return 0; },
  freemem: function() { return 0; },
  uptime: function() { return 0; },
  loadavg: function() { return [0, 0, 0]; },
  networkInterfaces: function() { return {}; },
  userInfo: function() { return { username: "", uid: -1, gid: -1, shell: "", homedir: __go_os_homedir() }; },
};
`, eol))
}
