// Package contract defines the Go↔JS interface — every globalThis name shared
// between Go bridge code and JS runtime files.
//
// Go code uses these constants for qctx.Globals().Set() and fmt.Sprintf references.
// JS runtime files reference the same names by convention — a mismatch crashes at startup.
//
// Importable from any internal package (jsbridge, harness) and the root brainkit package.
package contract

// ── Bridge Functions (Go registers on globalThis, JS calls directly) ──

const (
	// Request/response bridge — sync and async command invocation
	JSBridgeRequest      = "__go_brainkit_request"
	JSBridgeRequestAsync = "__go_brainkit_request_async"

	// Control bridge — local-only registration operations
	JSBridgeControl = "__go_brainkit_control"

	// Bus bridges — publish, emit, send, reply, subscribe
	JSBridgeBusSend      = "__go_brainkit_bus_send"
	JSBridgeBusPublish   = "__go_brainkit_bus_publish"
	JSBridgeBusEmit      = "__go_brainkit_bus_emit"
	JSBridgeBusReply     = "__go_brainkit_bus_reply"
	JSBridgeSubscribe    = "__go_brainkit_subscribe"
	JSBridgeUnsubscribe  = "__go_brainkit_unsubscribe"

	// Scheduling bridges
	JSBridgeBusSchedule   = "__go_brainkit_bus_schedule"
	JSBridgeBusUnschedule = "__go_brainkit_bus_unschedule"

	// Registry bridges — provider/storage/vector resolution
	JSBridgeRegistryResolve = "__go_registry_resolve"
	JSBridgeRegistryHas     = "__go_registry_has"
	JSBridgeRegistryList    = "__go_registry_list"

	// Resource tracking bridge — Go-native resource registry
	JSBridgeResourceRegister = "__go_resource_register"

	// Approval bridge — HITL tool approval
	JSBridgeAwaitApproval = "__go_brainkit_await_approval"

	// Secrets bridge
	JSBridgeSecretGet = "__go_brainkit_secret_get"

	// Logging bridge
	JSBridgeConsoleLogTagged = "__go_console_log_tagged"
)

// ── Bridge State (Go sets on globalThis, JS reads) ──

const (
	JSSandboxID        = "__brainkit_sandbox_id"
	JSSandboxNamespace = "__brainkit_sandbox_namespace"
	JSSandboxCallerID  = "__brainkit_sandbox_callerID"
	JSObsConfig        = "__brainkit_obs_config"
	JSProviders        = "__kit_providers"
)

// ── Runtime State (JS creates on globalThis, Go references) ──

const (
	JSCompartments = "__kit_compartments"
	JSBusSubs      = "__bus_subs"
	JSDispatch     = "__brainkit"
)

// ── Harness Bridge Functions ──

const (
	JSHarnessEvent       = "__go_harness_event"
	JSHarnessLockAcquire = "__go_harness_lock_acquire"
	JSHarnessLockRelease = "__go_harness_lock_release"
)

// ── jsbridge Polyfill Functions ──
// These map 1:1 to Node.js APIs. Grouped by polyfill module.

// Console
const (
	JSConsoleLog   = "__go_console_log"
	JSConsoleWarn  = "__go_console_warn"
	JSConsoleError = "__go_console_error"
	JSConsoleInfo  = "__go_console_info"
	JSConsoleDebug = "__go_console_debug"
)

// Crypto
const (
	JSCryptoRandomUUID    = "__go_crypto_randomUUID"
	JSCryptoHash          = "__go_crypto_hash"
	JSCryptoHmac          = "__go_crypto_hmac"
	JSCryptoSubtleDigest  = "__go_crypto_subtle_digest"
	JSCryptoSubtleSign    = "__go_crypto_subtle_sign"
	JSCryptoDeriveBits    = "__go_crypto_subtle_deriveBits"
	JSCryptoGetRandomVals = "__go_crypto_getRandomValues"
)

// DNS
const (
	JSDNSLookup      = "__go_dns_lookup"
	JSDNSLookupAsync = "__go_dns_lookup_async"
)

// Encoding
const (
	JSTextEncode = "__go_text_encode"
	JSTextDecode = "__go_text_decode"
	JSAtob       = "__go_atob"
	JSBtoa       = "__go_btoa"
)

// Fetch
const (
	JSFetch = "__go_fetch"
)

// Filesystem
const (
	JSFSReadFileSync     = "__go_fs_readFileSync"
	JSFSWriteFileSync    = "__go_fs_writeFileSync"
	JSFSAppendFileSync   = "__go_fs_appendFileSync"
	JSFSReaddirSync      = "__go_fs_readdirSync"
	JSFSStatSync         = "__go_fs_statSync"
	JSFSLstatSync        = "__go_fs_lstatSync"
	JSFSMkdirSync        = "__go_fs_mkdirSync"
	JSFSMkdtempSync      = "__go_fs_mkdtempSync"
	JSFSRmdirSync        = "__go_fs_rmdirSync"
	JSFSRmSync           = "__go_fs_rmSync"
	JSFSUnlinkSync       = "__go_fs_unlinkSync"
	JSFSRenameSync       = "__go_fs_renameSync"
	JSFSCopyFileSync     = "__go_fs_copyFileSync"
	JSFSCpSync           = "__go_fs_cpSync"
	JSFSLinkSync         = "__go_fs_linkSync"
	JSFSSymlinkSync      = "__go_fs_symlinkSync"
	JSFSReadlinkSync     = "__go_fs_readlinkSync"
	JSFSRealpathSync     = "__go_fs_realpathSync"
	JSFSChmodSync        = "__go_fs_chmodSync"
	JSFSChownSync        = "__go_fs_chownSync"
	JSFSTruncateSync     = "__go_fs_truncateSync"
	JSFSUtimesSync       = "__go_fs_utimesSync"
	JSFSAccessSync       = "__go_fs_accessSync"
	JSFSExistsSync       = "__go_fs_existsSync"
	JSFSAsync            = "__go_fs_async"
	JSFSConstantsJSON    = "__go_fs_constants_json"
	JSFSCreateReadStream = "__go_fs_createReadStream"
	JSFSCreateWriteStream = "__go_fs_createWriteStream"
	JSFSWatch            = "__go_fs_watch"
	JSFSWsWrite          = "__go_fs_ws_write"
	JSFSWsClose          = "__go_fs_ws_close"
	JSFSFhRead           = "__go_fs_fh_read"
	JSFSFhReadFile       = "__go_fs_fh_readFile"
	JSFSFhWrite          = "__go_fs_fh_write"
	JSFSFhWriteFile      = "__go_fs_fh_writeFile"
	JSFSFhStat           = "__go_fs_fh_stat"
	JSFSFhTruncate       = "__go_fs_fh_truncate"
	JSFSFhClose          = "__go_fs_fh_close"
)

// Net
const (
	JSNetConnect    = "__go_net_connect"
	JSNetWrite      = "__go_net_write"
	JSNetEnd        = "__go_net_end"
	JSNetTLSUpgrade = "__go_net_tls_upgrade"
)

// OS
const (
	JSOSPlatform = "__go_os_platform"
	JSOSArch     = "__go_os_arch"
	JSOSHomedir  = "__go_os_homedir"
	JSOSTmpdir   = "__go_os_tmpdir"
	JSOSHostname = "__go_os_hostname"
	JSOSType     = "__go_os_type"
	JSOSCpus     = "__go_os_cpus"
)

// Path
const (
	JSPathJoin     = "__go_path_join"
	JSPathResolve  = "__go_path_resolve"
	JSPathDirname  = "__go_path_dirname"
	JSPathBasename = "__go_path_basename"
	JSPathExtname  = "__go_path_extname"
)

// Process
const (
	JSProcessCwd    = "__go_process_cwd"
	JSProcessEnv    = "__go_process_env"
	JSProcessEnvSet = "__go_process_env_set"
)

// Exec / child_process
const (
	JSExec          = "__go_exec"
	JSExecSync      = "__go_exec_sync"
	JSExecFileSync  = "__go_exec_file_sync"
	JSSpawn         = "__go_spawn"
	JSSpawnWrite    = "__go_spawn_write"
	JSSpawnRead     = "__go_spawn_read"
	JSSpawnReadChunk = "__go_spawn_read_chunk"
	JSSpawnWait     = "__go_spawn_wait"
	JSSpawnKill     = "__go_spawn_kill"
)

// Timers
const (
	JSSetTimeout      = "__go_set_timeout"
	JSClearTimeout    = "__go_clear_timeout"
	JSScheduleTimeout = "__go_schedule_timeout"
)

// URL
const (
	JSURLParse        = "__go_url_parse"
	JSURLSearchParams = "__go_url_search_params"
)

// WebAssembly
const (
	JSWasmInstantiate = "__go_wasm_instantiate"
)

// Zlib
const (
	JSZlibInflate      = "__go_zlib_inflate"
	JSZlibDeflate      = "__go_zlib_deflate"
	JSGzipDecompress   = "__go_gzip_decompress"
	JSGzipCompress     = "__go_gzip_compress"
	JSRawInflate       = "__go_raw_inflate"
	JSRawDeflate       = "__go_raw_deflate"
)
