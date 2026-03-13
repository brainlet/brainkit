package asembed

import (
	"fmt"
	"testing"
)

func TestCumulativeCrashPoint(t *testing.T) {
	c, err := NewCompiler()
	if err != nil {
		t.Fatalf("NewCompiler: %v", err)
	}
	defer c.Close()

	src := `class A {
  x: i64 = 0;
  y: i64 = 0;
}
export function test(): void {
  let x = new A();
  let cnt = 0;
  x.x = x.y = cnt++;
  assert(cnt == 1);
}
test();
`

	for i := 0; i < 50; i++ {
		t.Run(fmt.Sprintf("iteration_%02d", i), func(t *testing.T) {
			result, cerr := c.Compile(map[string]string{
				"test.ts": src,
			}, CompileOptions{
				OptimizeLevel: 0,
				ShrinkLevel:   0,
				Debug:         true,
				Runtime:       "incremental",
			})
			if cerr != nil {
				t.Fatalf("Compile #%d: %v", i, cerr)
			}
			t.Logf("OK #%d: %d bytes", i, len(result.Binary))
		})
	}
}
