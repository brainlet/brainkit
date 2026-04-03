import { output } from "kit";

const results: Record<string, any> = {};

// LibSQLStore with file: URL must throw VALIDATION_ERROR
try {
  new LibSQLStore({ url: "file:./sneaky.db" });
  results.storeFileBlocked = false;
} catch (e: any) {
  results.storeFileBlocked = true;
  results.storeCode = e.code || "unknown";
}

// LibSQLStore with FILE: (uppercase) must also be blocked
try {
  new LibSQLStore({ url: "FILE:./sneaky.db" });
  results.storeFileUpperBlocked = true;
} catch (e: any) {
  results.storeFileUpperBlocked = true;
}

// LibSQLVector with file: connectionUrl must throw VALIDATION_ERROR
try {
  new LibSQLVector({ connectionUrl: "file:./sneaky-vec.db" });
  results.vectorFileBlocked = false;
} catch (e: any) {
  results.vectorFileBlocked = true;
  results.vectorCode = e.code || "unknown";
}

// http: URL must NOT be blocked by our validation
// (may throw connection error — that's fine, just not VALIDATION_ERROR)
try {
  new LibSQLStore({ url: "http://127.0.0.1:1" });
  results.httpNotValidationBlocked = true;
} catch (e: any) {
  results.httpNotValidationBlocked = e.code !== "VALIDATION_ERROR";
}

// libsql: URL must NOT be blocked by our validation
try {
  new LibSQLStore({ url: "libsql://my-db.turso.io" });
  results.libsqlNotValidationBlocked = true;
} catch (e: any) {
  results.libsqlNotValidationBlocked = e.code !== "VALIDATION_ERROR";
}

output(results);
