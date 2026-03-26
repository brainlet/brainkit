/**
 * Global types available inside brainkit SES Compartments.
 * These are injected as endowments — not imported via ES modules.
 */

// ── Node.js globals ────────────────────────────────────────────

declare var Buffer: {
  from(data: string | ArrayBuffer | Uint8Array | number[] | ArrayBufferView, encoding?: string, length?: number): any;
  alloc(size: number, fill?: number): any;
  allocUnsafe(size: number): any;
  isBuffer(obj: any): boolean;
  isEncoding(enc: string): boolean;
  byteLength(str: string | Uint8Array | ArrayBuffer, encoding?: string): number;
  concat(bufs: any[], totalLength?: number): any;
  compare(a: any, b: any): number;
  poolSize: number;
};

declare var process: {
  env: Record<string, string | undefined>;
  cwd(): string;
  version: string;
  versions: Record<string, string>;
  platform: string;
  arch: string;
  pid: number;
  argv: string[];
  execPath: string;
  title: string;
  nextTick(fn: (...args: any[]) => void, ...args: any[]): void;
  hrtime(prev?: [number, number]): [number, number];
  stdout: { write(s: string): boolean };
  stderr: { write(s: string): boolean };
  on(event: string, listener: (...args: any[]) => void): typeof process;
  once(event: string, listener: (...args: any[]) => void): typeof process;
  off(event: string, listener: (...args: any[]) => void): typeof process;
  emit(event: string, ...args: any[]): boolean;
  emitWarning(msg: string): void;
  getuid(): number;
  getgid(): number;
  geteuid(): number;
  getegid(): number;
  exit(code?: number): void;
  umask(mask?: number): number;
  uptime(): number;
  memoryUsage(): { rss: number; heapTotal: number; heapUsed: number; external: number };
  cpuUsage(): { user: number; system: number };
};

declare class EventEmitter {
  constructor();
  on(event: string, listener: (...args: any[]) => void): this;
  addListener(event: string, listener: (...args: any[]) => void): this;
  prependListener(event: string, listener: (...args: any[]) => void): this;
  once(event: string, listener: (...args: any[]) => void): this;
  prependOnceListener(event: string, listener: (...args: any[]) => void): this;
  emit(event: string, ...args: any[]): boolean;
  removeListener(event: string, listener: (...args: any[]) => void): this;
  off(event: string, listener: (...args: any[]) => void): this;
  removeAllListeners(event?: string): this;
  setMaxListeners(n: number): this;
  getMaxListeners(): number;
  listenerCount(event: string): number;
  listeners(event: string): Function[];
  rawListeners(event: string): Function[];
  eventNames(): string[];
  static captureRejections: boolean;
  static defaultMaxListeners: number;
  static setMaxListeners(...args: any[]): void;
  static listenerCount(emitter: EventEmitter, event: string): number;
}

// ── Node.js module globals (available in SES Compartments) ─────
// These match real Node.js module names. Internal __node_* aliases also exist.

declare var stream: {
  Readable: any;
  Writable: any;
  Duplex: any;
  Transform: any;
  PassThrough: any;
  pipeline: (...args: any[]) => void;
  finished: (stream: any, cb?: (err?: Error) => void) => void;
  Stream: any;
};

declare var crypto: {
  createHash(alg: string): { update(data: any, enc?: string): any; digest(enc?: string): any; copy(): any };
  createHmac(alg: string, key: any): { update(data: any, enc?: string): any; digest(enc?: string): any };
  pbkdf2Sync(password: any, salt: any, iterations: number, keylen: number, hash: string): any;
  pbkdf2(password: any, salt: any, iterations: number, keylen: number, hash: string, cb: (err: any, key: any) => void): void;
  randomBytes(n: number, cb?: (err: any, buf: any) => void): any;
  randomFillSync(buf: any): any;
  randomInt(min: number, max?: number): number;
  timingSafeEqual(a: any, b: any): boolean;
  getHashes(): string[];
  getCiphers(): string[];
  getFips(): number;
  webcrypto: typeof crypto;
};

declare var net: {
  Socket: any;
  createConnection: (...args: any[]) => any;
  connect: (...args: any[]) => any;
  createServer: (...args: any[]) => any;
  Server: any;
  isIP(input: string): number;
  isIPv4(input: string): boolean;
  isIPv6(input: string): boolean;
};

declare var os: {
  platform(): string;
  arch(): string;
  tmpdir(): string;
  homedir(): string;
  hostname(): string;
  type(): string;
  cpus(): Array<{ model: string; speed: number }>;
  EOL: string;
  endianness(): string;
  release(): string;
  totalmem(): number;
  freemem(): number;
  uptime(): number;
  loadavg(): [number, number, number];
  networkInterfaces(): Record<string, any>;
  userInfo(): { username: string; uid: number; gid: number; shell: string; homedir: string };
};

declare var dns: {
  lookup(hostname: string, cb: (err: any, addr: string, family: number) => void): void;
  lookup(hostname: string, options: any, cb: (err: any, addr: string, family: number) => void): void;
  resolve4(hostname: string, cb: (err: any, addrs: string[]) => void): void;
  Resolver: any;
  promises: {
    lookup(hostname: string): Promise<{ address: string; family: number }>;
    resolve4(hostname: string): Promise<string[]>;
    resolveSrv(hostname: string): Promise<any[]>;
    resolveCname(hostname: string): Promise<string[]>;
    resolvePtr(hostname: string): Promise<string[]>;
  };
};

declare var zlib: {
  inflate(buf: any, cb: (err: any, result: any) => void): void;
  deflate(buf: any, cb: (err: any, result: any) => void): void;
  deflate(buf: any, opts: any, cb: (err: any, result: any) => void): void;
  gunzip(buf: any, cb: (err: any, result: any) => void): void;
  gzip(buf: any, cb: (err: any, result: any) => void): void;
  inflateSync(buf: any): any;
  deflateSync(buf: any, opts?: any): any;
  gunzipSync(buf: any): any;
  gzipSync(buf: any): any;
  inflateRaw(buf: any, cb: (err: any, result: any) => void): void;
  deflateRaw(buf: any, cb: (err: any, result: any) => void): void;
  inflateRawSync(buf: any): any;
  deflateRawSync(buf: any, opts?: any): any;
  createGzip(): any;
  createGunzip(): any;
  createDeflate(opts?: any): any;
  createInflate(): any;
  constants: {
    Z_NO_COMPRESSION: number;
    Z_BEST_SPEED: number;
    Z_BEST_COMPRESSION: number;
    Z_DEFAULT_COMPRESSION: number;
    Z_DEFAULT_STRATEGY: number;
  };
};

declare var child_process: {
  exec(command: string): Promise<{ stdout: string; stderr: string; exitCode: number }>;
  execSync(command: string): any;
  execFileSync(file: string, args?: string[], options?: { cwd?: string }): any;
  spawnSync(command: string, args?: string[], options?: { cwd?: string }): { stdout: string; stderr: string; status: number; error: any };
  spawn(command: string, args?: string[], cwd?: string): {
    pid: number;
    readLine(): Promise<string | null>;
    readChunk(): Promise<string | null>;
    write(data: string): Promise<boolean>;
    wait(): Promise<number>;
    kill(): void;
  };
};

declare class GoSocket {
  connect(portOrOpts: number | { host?: string; port?: number; tls?: boolean }, host?: string): this;
  write(data: any, encoding?: string, cb?: (err?: Error) => void): boolean;
  end(data?: any, encoding?: string, cb?: () => void): void;
  destroy(err?: Error): this;
  pipe(dest: any, opts?: { end?: boolean }): any;
  on(event: string, listener: (...args: any[]) => void): this;
  once(event: string, listener: (...args: any[]) => void): this;
  removeListener(event: string, listener: (...args: any[]) => void): this;
  emit(event: string, ...args: any[]): boolean;
  setNoDelay(noDelay?: boolean): this;
  setKeepAlive(enable?: boolean, delay?: number): this;
  setTimeout(ms: number, cb?: () => void): this;
  readonly remoteAddress: string;
  readonly remotePort: number;
}
