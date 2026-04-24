package handlers

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type capabilityDispatchRequest struct {
	ctx      context.Context
	interval time.Duration
	ch       chan struct{}
}

type capabilityDispatcherEntry struct {
	dispatcher *CapabilityTestDispatcher
}

type CapabilityTestDispatcher struct {
	mu       sync.RWMutex
	queue    chan capabilityDispatchRequest
	closed   atomic.Bool
	lastUsed atomic.Int64
	idleTTL  time.Duration
}

type CapabilityTestDispatcherPool struct {
	mu          sync.RWMutex
	dispatchers map[string]*capabilityDispatcherEntry
	idleTTL     time.Duration
}

const capabilityDispatcherIdleTTL = 30 * time.Minute

var capabilityTestDispatcherPool = newCapabilityTestDispatcherPool()

func newCapabilityTestDispatcher() *CapabilityTestDispatcher {
	d := &CapabilityTestDispatcher{
		queue:   make(chan capabilityDispatchRequest, 4096),
		idleTTL: capabilityDispatcherIdleTTL,
	}
	d.touch()
	go d.run()
	return d
}

func (d *CapabilityTestDispatcher) touch() {
	d.lastUsed.Store(time.Now().UnixNano())
}

func (d *CapabilityTestDispatcher) lastUsedTime() time.Time {
	lastUsed := d.lastUsed.Load()
	if lastUsed == 0 {
		return time.Time{}
	}
	return time.Unix(0, lastUsed)
}

func newCapabilityTestDispatcherPool() *CapabilityTestDispatcherPool {
	p := &CapabilityTestDispatcherPool{
		dispatchers: make(map[string]*capabilityDispatcherEntry),
		idleTTL:     capabilityDispatcherIdleTTL,
	}
	go p.gcLoop()
	return p
}

func GetCapabilityTestDispatcher(identityKey string) *CapabilityTestDispatcher {
	return capabilityTestDispatcherPool.Get(identityKey)
}

func (p *CapabilityTestDispatcherPool) Get(identityKey string) *CapabilityTestDispatcher {
	key := identityKey
	if key == "" {
		key = "default"
	}

	p.mu.RLock()
	entry, ok := p.dispatchers[key]
	if ok && entry != nil && entry.dispatcher != nil && !entry.dispatcher.closed.Load() {
		dispatcher := entry.dispatcher
		p.mu.RUnlock()
		dispatcher.touch()
		return dispatcher
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if entry, ok := p.dispatchers[key]; ok && entry != nil && entry.dispatcher != nil {
		if !entry.dispatcher.closed.Load() {
			entry.dispatcher.touch()
			return entry.dispatcher
		}
		delete(p.dispatchers, key)
	}

	dispatcher := newCapabilityTestDispatcher()
	p.dispatchers[key] = &capabilityDispatcherEntry{dispatcher: dispatcher}
	return dispatcher
}

func (p *CapabilityTestDispatcherPool) gcLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		p.gc()
	}
}

func (p *CapabilityTestDispatcherPool) gc() {
	cutoff := time.Now().Add(-p.idleTTL)
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, entry := range p.dispatchers {
		if entry == nil || entry.dispatcher == nil {
			delete(p.dispatchers, key)
			continue
		}
		dispatcher := entry.dispatcher
		if dispatcher.closed.Load() || dispatcher.lastUsedTime().Before(cutoff) {
			delete(p.dispatchers, key)
		}
	}
}

func (d *CapabilityTestDispatcher) AcquireSendSlot(ctx context.Context, interval time.Duration) error {
	d.mu.RLock()
	if d.closed.Load() {
		d.mu.RUnlock()
		return errors.New("dispatcher closed")
	}

	readyCh := make(chan struct{}, 1)
	request := capabilityDispatchRequest{ctx: ctx, interval: interval, ch: readyCh}

	select {
	case <-ctx.Done():
		d.mu.RUnlock()
		return ctx.Err()
	case d.queue <- request:
		d.touch()
		d.mu.RUnlock()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-readyCh:
		d.touch()
		return nil
	}
}

func (d *CapabilityTestDispatcher) run() {
	nextAvailable := time.Now()
	idleTimer := time.NewTimer(d.idleTTL)
	defer idleTimer.Stop()

	for {
		select {
		case <-idleTimer.C:
			d.mu.Lock()
			lastUsed := d.lastUsedTime()
			idleFor := time.Since(lastUsed)
			if idleFor >= d.idleTTL && len(d.queue) == 0 {
				d.closed.Store(true)
				d.mu.Unlock()
				return
			}
			d.mu.Unlock()
			resetCapabilityDispatcherTimer(idleTimer, d.idleTTL)
		case request := <-d.queue:
			d.touch()
			resetCapabilityDispatcherTimer(idleTimer, d.idleTTL)
			if request.ctx.Err() != nil {
				continue
			}

			now := time.Now()
			if wait := nextAvailable.Sub(now); wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-timer.C:
				case <-request.ctx.Done():
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					continue
				}
			}

			select {
			case request.ch <- struct{}{}:
			default:
			}
			d.touch()

			interval := request.interval
			if interval <= 0 {
				interval = time.Minute / 10
			}
			log.Printf("[CapabilityTest-Dispatch] 放行一个能力测试请求，间隔=%s", interval)
			nextAvailable = time.Now().Add(interval)
		}
	}
}

func resetCapabilityDispatcherTimer(timer *time.Timer, d time.Duration) {
	if d <= 0 {
		d = time.Second
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(d)
}
