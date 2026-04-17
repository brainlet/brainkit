# Secrets Test Map

**Purpose:** Verifies the encrypted secrets store: CRUD operations, rotation, JS bridge access, audit events, concurrent access, dev mode, persistence with encryption, and input abuse resilience.
**Tests:** 22 functions across 5 files
**Entry point:** `secrets_test.go` → `Run(t, env)`
**Campaigns:** transport (amqp, redis, postgres, nats, sqlite)

## Files

### crud.go — Core secrets CRUD and bridge operations

| Function | Purpose |
|----------|---------|
| testSetAndGet | Sets a secret via SecretsSetMsg, verifies stored=true and version=1, gets it back via SecretsGetMsg, verifies value matches |
| testDelete | Sets a secret, deletes via SecretsDeleteMsg, verifies deleted=true, gets it, verifies empty value |
| testList | Sets 2 secrets, lists via SecretsListMsg, verifies 2 entries with non-empty names |
| testRotate | Sets a secret, rotates via SecretsRotateMsg, verifies rotated=true and version=2, gets it, verifies new value |
| testJSBridge | Sets a secret via bus, reads it from EvalTS using secrets.get(), verifies the value matches |
| testAuditEvents | Subscribes to secrets.stored event, sets a secret, verifies the audit event contains the secret name and version |
| testConcurrentAccess | Sets a secret, spawns 10 goroutines that all read it simultaneously, verifies all get the correct value without race |
| testDevModeNoEncryption | Creates kernel without secret key (dev mode), sets and gets a secret, verifies it works without encryption |
| testListNeverLeaksValues | Sets a secret, lists secrets, marshals the list response to JSON, verifies the secret value string never appears in the output |

### matrix.go — Adversarial secrets matrix tests

| Function | Purpose |
|----------|---------|
| testMatrixSetGetDeleteList | Full lifecycle: set, get (verify value), list (verify present), delete (verify deleted), get again (verify empty) |
| testMatrixRotate | Sets v1, rotates to v2, gets, verifies v2 returned |
| testMatrixManySecrets | Sets 20 secrets with unique names, lists all, verifies all 20 appear |
| testMatrixEncryptedPersistence | Sets encrypted secret, closes kernel, reopens with same key, verifies secret decrypts correctly |
| testMatrixWrongKeyCannotDecrypt | Sets encrypted secret with key A, reopens with key B, verifies the returned value is not the original plaintext |
| testMatrixAuditEvents | Subscribes to secrets.stored/secrets.accessed/secrets.deleted, performs set/get/delete, verifies all 3 audit events fire |
| testMatrixFromTS | Sets a secret via bus, reads it from deployed .ts using secrets.get(), verifies the value matches |

### input_abuse.go — Secrets input abuse

| Function | Purpose |
|----------|---------|
| testInputAbuseEmptyName | Sends SecretsSetMsg with empty name, verifies VALIDATION_ERROR response code |
| testInputAbuseLargeValue | Stores a 100KB secret value, verifies stored=true (no crash or timeout) |
| testInputAbuseSpecialCharsInName | Sets secrets with names containing slashes, dots, spaces, equals, verifies no panic per name |
| testInputAbuseBulkOperations | Sets 20 secrets with varying names, lists them all, verifies the list returns without hang |

### integration.go — Secrets rotation integration

| Function | Purpose |
|----------|---------|
| testSecretsRotation | Sets a secret, rotates it, verifies the get returns the new value (typed SDK responses) |
| testE2ESecretsRotateAndVerify | Sets secret v1, rotates to v2, gets via raw bus, verifies v2 value in JSON response |

### backend_advanced.go — Transport-level secrets tests

| Function | Purpose |
|----------|---------|
| testSecretsOnTransport | Sets a secret via SDK, gets it via raw bus subscribe, verifies the payload contains the value |

## Cross-references

- **Campaigns:** transport/{amqp,redis,postgres,nats,sqlite}_test.go
- **Related domains:** persistence (encrypted secret persistence), security (secret exfiltration tests)
- **Fixtures:** secrets-related TS fixtures
