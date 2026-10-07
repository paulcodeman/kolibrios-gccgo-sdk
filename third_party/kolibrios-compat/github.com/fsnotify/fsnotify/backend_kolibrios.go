//go:build kolibrios

// Native backend for upstream fsnotify. KolibriOS has no inotify/kqueue API;
// use the upstream radovskyb/watcher polling implementation instead.
package fsnotify

import (
	"path/filepath"
	"sync"
	"time"

	poll "github.com/radovskyb/watcher"
)

type nativePolling struct {
	operation sync.Mutex
	mu        sync.Mutex
	watcher   *poll.Watcher
	events    chan Event
	errors    chan error
	stop      chan struct{}
	done      chan struct{}
	closed    bool
	paths     map[string]Op
}

func newBackend(events chan Event, errors chan error) (backend, error) {
	p := &nativePolling{watcher: poll.New(), events: events, errors: errors,
		stop: make(chan struct{}), done: make(chan struct{}), paths: make(map[string]Op)}
	go p.forward()
	go func() {
		if err := p.watcher.Start(time.Second); err != nil {
			select {
			case errors <- err:
			case <-p.stop:
			}
		}
	}()
	p.watcher.Wait()
	return p, nil
}

func newBufferedBackend(size uint, events chan Event, errors chan error) (backend, error) {
	return newBackend(events, errors)
}

func (p *nativePolling) send(event Event) {
	p.mu.Lock()
	mask, watched := p.paths[event.Name]
	if !watched {
		mask, watched = p.paths[filepath.Dir(event.Name)]
	}
	p.mu.Unlock()
	if watched && event.Op&mask != 0 {
		select {
		case p.events <- event:
		case <-p.stop:
		}
	}
}

func (p *nativePolling) forward() {
	defer close(p.events)
	defer close(p.errors)
	defer close(p.done)
	for {
		select {
		case event := <-p.watcher.Event:
			switch event.Op {
			case poll.Create:
				p.send(Event{Name: event.Path, Op: Create})
			case poll.Write:
				p.send(Event{Name: event.Path, Op: Write})
			case poll.Remove:
				p.send(Event{Name: event.Path, Op: Remove})
			case poll.Rename, poll.Move:
				p.send(Event{Name: event.OldPath, Op: Rename})
				p.send(Event{Name: event.Path, Op: Create})
			case poll.Chmod:
				p.send(Event{Name: event.Path, Op: Chmod})
			}
		case err := <-p.watcher.Error:
			select {
			case p.errors <- err:
			case <-p.stop:
			}
		case <-p.watcher.Closed:
			return
		case <-p.stop:
			// Continue draining upstream channels while its Close completes.
			go p.watcher.Close()
			for {
				select {
				case <-p.watcher.Event:
				case <-p.watcher.Error:
				case <-p.watcher.Closed:
					return
				}
			}
		}
	}
}

func (p *nativePolling) Add(name string) error { return p.AddWith(name) }
func (p *nativePolling) AddWith(name string, options ...addOpt) error {
	p.operation.Lock()
	defer p.operation.Unlock()
	path, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	with := getOptions(options...)
	if !p.xSupports(with.op) {
		return xErrUnsupported
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	if _, exists := p.paths[path]; exists {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()
	if err = p.watcher.Add(path); err != nil {
		return err
	}
	p.mu.Lock()
	p.paths[path] = with.op
	p.mu.Unlock()
	return nil
}

func (p *nativePolling) Remove(name string) error {
	p.operation.Lock()
	defer p.operation.Unlock()
	path, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	if _, exists := p.paths[path]; !exists {
		p.mu.Unlock()
		return ErrNonExistentWatch
	}
	delete(p.paths, path)
	p.mu.Unlock()
	return p.watcher.Remove(path)
}

func (p *nativePolling) WatchList() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	paths := make([]string, 0, len(p.paths))
	for path := range p.paths {
		paths = append(paths, path)
	}
	return paths
}

func (p *nativePolling) Close() error {
	p.operation.Lock()
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.stop)
	}
	p.mu.Unlock()
	p.operation.Unlock()
	<-p.done
	return nil
}

func (p *nativePolling) xSupports(op Op) bool {
	return op & ^(Create|Write|Remove|Rename|Chmod) == 0
}
