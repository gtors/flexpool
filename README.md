# FlexPool

A flexible worker pool implementation in Go that supports dynamic resizing

## Features

- Dynamic worker pool resizing
- Graceful termination
- Task buffering with configurable capacity
- Thread-safe operations

## Usage

```go
package main

import (
	"fmt"
	"time"
  "context"

	"github.com/gtors/flexpool"
)

func main() {
  ctx := context.Background()

	pool := flexpool.New(
    flexpool.WithContext(ctx),
    flexpool.WithPoolSize(5),
    flexpool.WithTasksBufferSize(20),
  )

	// Send tasks
	for range 10 {
		pool.SendTask(func() {
			fmt.Println("Processing task")
			time.Sleep(100 * time.Millisecond)
		})
	}

	// Resize pool if needed
	pool.Resize(3)

	// Close pool and wait for workers
	pool.Close()
}
```
