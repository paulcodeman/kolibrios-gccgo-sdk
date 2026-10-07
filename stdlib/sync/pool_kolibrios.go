package sync

// Pool is a minimal object pool.
type Pool struct {
	New   func() any
	mu    Mutex
	items []any
}

func (p *Pool) Get() any {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	n := len(p.items)
	if n != 0 {
		item := p.items[n-1]
		p.items = p.items[:n-1]
		p.mu.Unlock()
		return item
	}
	p.mu.Unlock()
	if p.New != nil {
		return p.New()
	}
	return nil
}

func (p *Pool) Put(x any) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.items = append(p.items, x)
	p.mu.Unlock()
}

