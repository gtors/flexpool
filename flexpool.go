// Package flexpool implements a flexible goroutine worker pool that can resize dynamically
package flexpool

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrPoolClosed is returned when attempting to send a task to a closed pool
	ErrPoolClosed = errors.New("pool has been closed")
)

const (
	defaultResizeInterval  = 1 * time.Second
	defaultTasksBufferSize = 256
)

// Pool manages a dynamic number of worker goroutines that process tasks
type Pool struct {
	ctx            context.Context
	wg             sync.WaitGroup
	activeWorkers  atomic.Uint64
	maxWorkers     atomic.Uint64
	resizeInterval time.Duration
	stopSignal     chan struct{}
	tasksQueue     chan func()
	closed         atomic.Bool
}

// Option defines a function signature for configuring a Pool
type Option func(p *Pool)

// WithContext sets the context for the Pool
func WithContext(ctx context.Context) Option {
	return func(p *Pool) {
		p.ctx = ctx
	}
}

// WithPoolSize sets the maximum number of workers in the Pool
func WithPoolSize(sz uint) Option {
	return func(p *Pool) {
		p.maxWorkers.Store(uint64(sz))
	}
}

// WithTasksBufferSize sets the buffer size for the task queue
func WithTasksBufferSize(sz uint) Option {
	return func(p *Pool) {
		p.tasksQueue = make(chan func(), sz)
	}
}

// WithResizeInterval sets the interval for resizing the pool of workers
func WithResizeInterval(interval time.Duration) Option {
	return func(p *Pool) {
		p.resizeInterval = interval
	}
}

// New creates a new worker pool with the specified configuration options
func New(options ...Option) *Pool {
	p := &Pool{
		tasksQueue:     make(chan func(), defaultTasksBufferSize),
		stopSignal:     make(chan struct{}),
		resizeInterval: defaultResizeInterval,
	}

	// Default to the number of CPUs.
	p.maxWorkers.Store(uint64(runtime.NumCPU()))

	for _, opt := range options {
		opt(p)
	}

	if p.ctx == nil {
		// Default context if none is provided
		p.ctx = context.Background()
	}

	// Start the resizer goroutine. It will manage the count of workers
	go p.startResizer()
	return p
}

// GetActiveWorkersNumber returns current number of the workers
func (p *Pool) GetActiveWorkersNumber() uint64 {
	return p.activeWorkers.Load()
}

// GetPoolSize returns current pool size
func (p *Pool) GetPoolSize() uint64 {
	return p.maxWorkers.Load()
}

// Resize adjusts the number of workers in the pool
func (p *Pool) Resize(size uint) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}
	p.maxWorkers.Store(uint64(size))
	return nil
}

// SendTask adds a task to the pool
func (p *Pool) SendTask(task func()) error {
	if p.closed.Load() {
		return ErrPoolClosed
	} else {
		p.tasksQueue <- task
		return nil
	}
}

// IsClosed returns true if the pool already closed
func (p *Pool) IsClosed() bool {
	return p.closed.Load()
}

// Close stops all workers and waits for them to finish
func (p *Pool) Close() {
	// Only close if the pool wasn't already closed
	if !p.closed.Swap(true) {
		close(p.stopSignal)
	}
	// Wait for all workers to finish
	p.wg.Wait()
}

// Wait blocks until all workers finish processing (after Close is called)
func (p *Pool) Wait() {
	p.wg.Wait()
}

// spawnWorker starts a new worker goroutine to process tasks
func (p *Pool) spawnWorker() {
	if p.closed.Load() {
		return
	}

	p.wg.Add(1)
	p.activeWorkers.Add(1)

	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-p.stopSignal:
				return
			case task, ok := <-p.tasksQueue:
				if !ok {
					return
				}
				task()
			}
		}
	}()
}

// startResizer continuously adjusts the number of workers based on demand
func (p *Pool) startResizer() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			if p.closed.Load() {
				return
			}
		}

		activeWorkers := p.activeWorkers.Load()
		maxWorkers := p.maxWorkers.Load()

		if maxWorkers > activeWorkers {
			toSpawn := maxWorkers - activeWorkers
			for range toSpawn {
				p.spawnWorker()
			}
		} else if activeWorkers > maxWorkers {
			toKill := activeWorkers - maxWorkers
			for range toKill {
				select {
				case p.stopSignal <- struct{}{}: // Request worker to stop
				default:
				}
			}
		}

		time.Sleep(p.resizeInterval)
	}
}
