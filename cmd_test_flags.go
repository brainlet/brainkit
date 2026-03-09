//go:build ignore

package main

import (
	"fmt"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

func main() {
	fmt.Printf("None:            %d (expect 0)\n", program.DecoratorFlagsNone)
	fmt.Printf("Global:          %d (expect 1)\n", program.DecoratorFlagsGlobal)
	fmt.Printf("OperatorBinary:  %d (expect 2)\n", program.DecoratorFlagsOperatorBinary)
	fmt.Printf("OperatorPrefix:  %d (expect 4)\n", program.DecoratorFlagsOperatorPrefix)
	fmt.Printf("OperatorPostfix: %d (expect 8)\n", program.DecoratorFlagsOperatorPostfix)
	fmt.Printf("Unmanaged:       %d (expect 16)\n", program.DecoratorFlagsUnmanaged)
	fmt.Printf("Final:           %d (expect 32)\n", program.DecoratorFlagsFinal)
	fmt.Printf("Inline:          %d (expect 64)\n", program.DecoratorFlagsInline)
	fmt.Printf("External:        %d (expect 128)\n", program.DecoratorFlagsExternal)
	fmt.Printf("ExternalJs:      %d (expect 256)\n", program.DecoratorFlagsExternalJs)
	fmt.Printf("Builtin:         %d (expect 512)\n", program.DecoratorFlagsBuiltin)
	fmt.Printf("Lazy:            %d (expect 1024)\n", program.DecoratorFlagsLazy)
	fmt.Printf("Unsafe:          %d (expect 2048)\n", program.DecoratorFlagsUnsafe)
}
