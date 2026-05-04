package engine

type hookLease[T any] struct {
	token uint64
	value T
}

type hookLeaseSlot[T any] struct {
	base    T
	baseSet bool
	next    uint64
	active  []hookLease[T]
}

func (s *hookLeaseSlot[T]) acquire(base T, value T) uint64 {
	if !s.baseSet {
		s.base = base
		s.baseSet = true
	}
	s.next++
	token := s.next
	s.active = append(s.active, hookLease[T]{token: token, value: value})
	return token
}

func (s *hookLeaseSlot[T]) release(token uint64) T {
	for i, lease := range s.active {
		if lease.token != token {
			continue
		}
		copy(s.active[i:], s.active[i+1:])
		var zero hookLease[T]
		s.active[len(s.active)-1] = zero
		s.active = s.active[:len(s.active)-1]
		break
	}
	return s.current()
}

func (s *hookLeaseSlot[T]) current() T {
	if len(s.active) > 0 {
		return s.active[len(s.active)-1].value
	}
	return s.base
}
