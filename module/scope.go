package module

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Cleanup releases a resource owned by a Scope.
type Cleanup func(context.Context) error

// Scope owns resources created while mounting a module.
type Scope interface {
	ID() string
	Child(id string) Scope
	Defer(Cleanup)
	Resource(ResourceDescriptor)
	Resources() []ResourceDescriptor
	Close(context.Context) error
}

type scope struct {
	id        string
	mu        sync.Mutex
	cleanups  []Cleanup
	resources []ResourceDescriptor
	children  []*scope
	closed    bool
}

// NewScope creates an empty resource scope.
func NewScope(id string) Scope {
	return &scope{id: id}
}

func (s *scope) ID() string { return s.id }

func (s *scope) Child(id string) Scope {
	childID := id
	if s.id != "" && id != "" {
		childID = s.id + "/" + id
	}
	child := &scope{id: childID}
	s.mu.Lock()
	if !s.closed {
		s.children = append(s.children, child)
		s.cleanups = append(s.cleanups, child.Close)
	}
	s.mu.Unlock()
	return child
}

func (s *scope) Defer(fn Cleanup) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.cleanups = append(s.cleanups, fn)
}

func (s *scope) Resource(desc ResourceDescriptor) {
	if desc.Kind == "" || desc.Name == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.resources = append(s.resources, desc)
}

func (s *scope) Resources() []ResourceDescriptor {
	s.mu.Lock()
	resources := append([]ResourceDescriptor(nil), s.resources...)
	children := append([]*scope(nil), s.children...)
	s.mu.Unlock()

	for _, child := range children {
		resources = append(resources, child.Resources()...)
	}
	return normalizeResources(resources)
}

func (s *scope) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed && len(s.cleanups) == 0 {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	cleanups := append([]Cleanup(nil), s.cleanups...)
	s.cleanups = nil
	s.mu.Unlock()

	var err error
	var failed []Cleanup
	for i := len(cleanups) - 1; i >= 0; i-- {
		if e := cleanups[i](ctx); e != nil {
			err = errors.Join(err, e)
			failed = append(failed, cleanups[i])
		}
	}
	if err != nil && s.id != "" {
		s.mu.Lock()
		for i := len(failed) - 1; i >= 0; i-- {
			s.cleanups = append(s.cleanups, failed[i])
		}
		s.mu.Unlock()
		return fmt.Errorf("scope %s: %w", s.id, err)
	}
	return err
}

// Handle is a closeable lease returned by scoped hosts.
type Handle interface {
	Close(context.Context) error
}

// HandleFunc adapts a function to Handle.
type HandleFunc func(context.Context) error

// Close satisfies Handle.
func (f HandleFunc) Close(ctx context.Context) error { return f(ctx) }
