package kit

import (
	"context"
	"encoding/json"
	"fmt"
)

type LocalInvoker struct {
	kernel *Kernel
}

func newLocalInvoker(kernel *Kernel) *LocalInvoker {
	return &LocalInvoker{kernel: kernel}
}

func (i *LocalInvoker) Invoke(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error) {
	spec, ok := commandCatalog().Lookup(topic)
	if !ok || spec.invokeKernel == nil {
		return nil, fmt.Errorf("unknown topic: %s", topic)
	}
	return spec.invokeKernel(ctx, i.kernel, payload)
}
