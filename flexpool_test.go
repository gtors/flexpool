package flexpool

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPoolBasicFunctionality(t *testing.T) {
	p := New(WithPoolSize(2), WithTasksBufferSize(10))
	defer p.Close()

	var counter int
	var m sync.Mutex

	for range 10 {
		p.SendTask(func() {
			m.Lock()
			defer m.Unlock()
			counter++
		})
	}

	time.Sleep(100 * time.Millisecond)
	p.Close()

	m.Lock()
	assert.Equal(t, 10, counter)
	m.Unlock()
}

func TestTermination(t *testing.T) {
	p := New(WithPoolSize(2), WithTasksBufferSize(10))
	p.Close()
	err := p.SendTask(func() {})
	assert.Equal(t, ErrPoolClosed, err, "Sending task should fail after the pool is closed")
}

// TestResizeUp tests that resizing the pool increases the number of workers
func TestResizeUp(t *testing.T) {
	// Create a pool with a small initial size and task buffer
	pool := New(
		WithPoolSize(2), // Initial pool size of 2 workers
		WithResizeInterval(50*time.Millisecond),
	)
	defer pool.Close()

	// Resize the pool to have more workers
	err := pool.Resize(10)
	assert.NoError(t, err)

	// Wait a bit to allow the resizer to scale the workers
	time.Sleep(100 * time.Millisecond)

	// Check if the pool size is increased to 10 workers
	activeWorkers := pool.GetActiveWorkersNumber()
	assert.Equal(t, activeWorkers, uint64(10), "Active workers should be 10 after resizing up")
}

// TestResizeDown tests that resizing the pool decreases the number of workers
func TestResizeDown(t *testing.T) {
	// Create a pool with a large initial size and task buffer
	pool := New(
		WithPoolSize(100), // Initial pool size of 10 workers
		WithResizeInterval(50*time.Millisecond),
	)
	defer pool.Close()

	// Wait a bit to allow the resizer to scale the workers
	time.Sleep(100 * time.Millisecond)

	workersAtStart := pool.GetActiveWorkersNumber()
	assert.Equal(t, workersAtStart, uint64(100), "At the start, active workers number should be 100")

	// Resize the pool to have fewer workers
	err := pool.Resize(5)
	assert.NoError(t, err)

	// Wait a bit to allow the resizer to scale down the workers
	time.Sleep(200 * time.Millisecond)

	// Check if the pool size is decreased to 5 workers
	activeWorkers := pool.GetActiveWorkersNumber()
	assert.Equal(t, activeWorkers, uint64(5), "Active workers should be 5 after resizing down")
}

// TestResizeAfterClose tests that resizing does not work after the pool is closed
func TestResizeAfterClose(t *testing.T) {
	// Create a pool with an initial size
	pool := New(WithPoolSize(5))

	// Close the pool
	pool.Close()

	// Attempt to resize the pool after closing
	err := pool.Resize(10)
	assert.Equal(t, ErrPoolClosed, err, "Resizing should fail after the pool is closed")
}

// TestResizeWithMultipleTasks tests that resizing works while tasks are being processed
func TestResizeWithMultipleTasks(t *testing.T) {
	// Create a pool with a small initial size and task buffer
	pool := New(
		WithPoolSize(2), // Initial pool size of 2 workers
		WithResizeInterval(50*time.Millisecond),
	)
	defer pool.Close()

	// Send tasks to utilize workers
	for range 10 {
		err := pool.SendTask(func() {
			// Simulate a task
			time.Sleep(10 * time.Millisecond)
		})
		assert.NoError(t, err)
	}

	// Resize the pool to have more workers while tasks are being processed
	err := pool.Resize(6)
	assert.NoError(t, err)

	// Wait a bit to allow the resizer to scale the workers
	time.Sleep(100 * time.Millisecond)

	// Check if the pool size is increased to 6 workers
	activeWorkers := pool.GetActiveWorkersNumber()
	assert.Equal(t, activeWorkers, uint64(6), "Active workers should be at least 6 after resizing up")
}
