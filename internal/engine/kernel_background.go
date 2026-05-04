package engine

import "context"

func (k *Kernel) startBackground(fn func(context.Context)) {
	if k == nil || fn == nil {
		return
	}
	ctx := k.shutdownCtx
	if ctx == nil {
		ctx = context.Background()
	}
	k.background.Add(1)
	go func() {
		defer k.background.Done()
		fn(ctx)
	}()
}

func (k *Kernel) waitBackground(ctx context.Context) error {
	if k == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	k.mu.Lock()
	done := k.backgroundDone
	if done == nil {
		done = make(chan struct{})
		k.backgroundDone = done
	}
	k.backgroundWait.Do(func() {
		go func() {
			k.background.Wait()
			close(done)
		}()
	})
	k.mu.Unlock()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
