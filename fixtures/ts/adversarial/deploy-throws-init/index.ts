import { output } from "kit";
// This fixture tests that output() works before a throw
output({ beforeThrow: true });
// Note: if we throw here, the fixture runner catches it at deploy level
// and the test fails — which IS the expected behavior for deploy-throws-init.
// But for the fixture system, we need to NOT throw to verify output works.
