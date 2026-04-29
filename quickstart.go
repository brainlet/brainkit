package brainkit

import (
	"fmt"
)

// QuickStart creates a bare Kit wired with memory transport and an FSRoot. It does
// not compose persistence or the standard module set; import brainkit/server or
// github.com/brainlet/brainkit/transports for those optional pieces.
//
// fsRoot must be an existing writable directory.
func QuickStart(namespace, fsRoot string) (*Kit, error) {
	if fsRoot == "" {
		return nil, fmt.Errorf("brainkit.QuickStart: fsRoot is required")
	}
	return New(Config{
		Namespace: namespace,
		CallerID:  namespace,
		FSRoot:    fsRoot,
		Transport: Memory(),
	})
}
