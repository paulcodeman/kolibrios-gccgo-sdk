package sync

import "kos"

// Locker represents an object that can be locked and unlocked.
type Locker interface {
	Lock()
	Unlock()
}

// Mutex is a minimal mutex suitable for single-threaded use.
type Mutex struct {
	state uint32
}

func (m *Mutex) Lock() {
	for !m.TryLock() {
		yield()
	}
}

func (m *Mutex) TryLock() bool {
	if m.state != 0 {
		return false
	}
	m.state = 1
	return true
}

func (m *Mutex) Unlock() {
	if m.state == 0 {
		panic("sync: unlock of unlocked mutex")
	}
	m.state = 0
}

// RWMutex is a minimal reader/writer mutex that serializes all access.
type RWMutex struct {
	mu Mutex
}

func (rw *RWMutex) Lock()   { rw.mu.Lock() }
func (rw *RWMutex) Unlock() { rw.mu.Unlock() }
func (rw *RWMutex) RLock()  { rw.mu.Lock() }
func (rw *RWMutex) RUnlock() {
	rw.mu.Unlock()
}

type rlocker RWMutex

func (r *rlocker) Lock()   { (*RWMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*RWMutex)(r).RUnlock() }

// RLocker returns a Locker interface that locks and unlocks rw for reading.
func (rw *RWMutex) RLocker() Locker {
	return (*rlocker)(rw)
}

// Once is a minimal implementation of sync.Once.
type Once struct {
	done uint32
}

func (o *Once) Do(f func()) {
	if o.done != 0 {
		return
	}
	f()
	o.done = 1
}

// WaitGroup is a minimal wait group implementation.
type WaitGroup struct {
	mu      Mutex
	counter int
}

func (wg *WaitGroup) Add(delta int) {
	wg.mu.Lock()
	wg.counter += delta
	if wg.counter < 0 {
		wg.mu.Unlock()
		panic("sync: negative WaitGroup counter")
	}
	wg.mu.Unlock()
}

func (wg *WaitGroup) Done() {
	wg.Add(-1)
}

func (wg *WaitGroup) Wait() {
	for {
		wg.mu.Lock()
		remaining := wg.counter
		wg.mu.Unlock()
		if remaining == 0 {
			return
		}
		yield()
	}
}

// Cond is a minimal condition variable.
type Cond struct {
	L Locker
}

func NewCond(l Locker) *Cond {
	return &Cond{L: l}
}

func (c *Cond) Wait() {
	if c == nil || c.L == nil {
		return
	}
	c.L.Unlock()
	yield()
	c.L.Lock()
}

func (c *Cond) Signal() {
}

func (c *Cond) Broadcast() {
}

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

// Map is a minimal, non-concurrent map implementation.
type Map struct {
	mu    Mutex
	items []mapEntry
}

type mapEntry struct {
	key     any
	value   any
	deleted bool
}

func (m *Map) Load(key any) (value any, ok bool) {
	m.mu.Lock()
	for i := range m.items {
		item := &m.items[i]
		if item.deleted {
			continue
		}
		if item.key == key {
			value = item.value
			ok = true
			break
		}
	}
	m.mu.Unlock()
	return
}

func (m *Map) Store(key, value any) {
	m.mu.Lock()
	for i := range m.items {
		item := &m.items[i]
		if item.deleted {
			continue
		}
		if item.key == key {
			item.value = value
			m.mu.Unlock()
			return
		}
	}
	m.items = append(m.items, mapEntry{key: key, value: value})
	m.mu.Unlock()
}

func (m *Map) LoadOrStore(key, value any) (actual any, loaded bool) {
	m.mu.Lock()
	for i := range m.items {
		item := &m.items[i]
		if item.deleted {
			continue
		}
		if item.key == key {
			actual = item.value
			loaded = true
			m.mu.Unlock()
			return
		}
	}
	m.items = append(m.items, mapEntry{key: key, value: value})
	m.mu.Unlock()
	return value, false
}

func (m *Map) LoadAndDelete(key any) (value any, loaded bool) {
	m.mu.Lock()
	for i := range m.items {
		item := &m.items[i]
		if item.deleted {
			continue
		}
		if item.key == key {
			item.deleted = true
			value = item.value
			item.value = nil
			loaded = true
			break
		}
	}
	m.mu.Unlock()
	return
}

func (m *Map) Delete(key any) {
	m.mu.Lock()
	for i := range m.items {
		item := &m.items[i]
		if item.deleted {
			continue
		}
		if item.key == key {
			item.deleted = true
			item.value = nil
			break
		}
	}
	m.mu.Unlock()
}

func (m *Map) Range(f func(key, value any) bool) {
	if f == nil {
		return
	}
	m.mu.Lock()
	items := make([]mapEntry, 0, len(m.items))
	for _, item := range m.items {
		if item.deleted {
			continue
		}
		items = append(items, item)
	}
	m.mu.Unlock()
	for _, item := range items {
		if !f(item.key, item.value) {
			return
		}
	}
}

func yield() {
	var regs kos.SyscallRegs
	regs.EAX = 68
	regs.EBX = 1
	kos.SyscallRaw(&regs)
}
