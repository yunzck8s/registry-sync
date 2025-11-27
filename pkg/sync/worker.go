package sync

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of workers for concurrent task execution
type WorkerPool struct {
	workers     int
	taskCh      chan Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	errors      []error
	errorsMux   sync.Mutex
	totalTasks  int64
	doneTasks   int64
	failedTasks int64
}

// Task represents a unit of work
type Task interface {
	Execute(ctx context.Context) error
	Description() string
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(ctx context.Context, workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)

	return &WorkerPool{
		workers: workers,
		taskCh:  make(chan Task, workers*2), // Buffer to reduce blocking
		ctx:     ctx,
		cancel:  cancel,
		errors:  make([]error, 0),
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the worker goroutine
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskCh:
			if !ok {
				return
			}

			// Execute task
			if err := task.Execute(p.ctx); err != nil {
				p.addError(fmt.Errorf("worker %d: task %s failed: %w", id, task.Description(), err))
				atomic.AddInt64(&p.failedTasks, 1)
			}

			atomic.AddInt64(&p.doneTasks, 1)
		}
	}
}

// Submit submits a task to the pool
func (p *WorkerPool) Submit(task Task) error {
	atomic.AddInt64(&p.totalTasks, 1)

	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case p.taskCh <- task:
		return nil
	}
}

// Wait waits for all tasks to complete and returns any errors
func (p *WorkerPool) Wait() error {
	close(p.taskCh)
	p.wg.Wait()

	if len(p.errors) > 0 {
		return fmt.Errorf("worker pool encountered %d errors: %v", len(p.errors), p.errors[0])
	}

	return nil
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	p.cancel()
}

// addError adds an error to the error list
func (p *WorkerPool) addError(err error) {
	p.errorsMux.Lock()
	defer p.errorsMux.Unlock()
	p.errors = append(p.errors, err)
}

// GetErrors returns all errors encountered
func (p *WorkerPool) GetErrors() []error {
	p.errorsMux.Lock()
	defer p.errorsMux.Unlock()
	return append([]error{}, p.errors...)
}

// GetProgress returns the current progress
func (p *WorkerPool) GetProgress() (total, done, failed int64) {
	return atomic.LoadInt64(&p.totalTasks),
		atomic.LoadInt64(&p.doneTasks),
		atomic.LoadInt64(&p.failedTasks)
}

// IsComplete checks if all tasks are complete
func (p *WorkerPool) IsComplete() bool {
	total := atomic.LoadInt64(&p.totalTasks)
	done := atomic.LoadInt64(&p.doneTasks)
	return total > 0 && total == done
}

// ProgressStats contains progress statistics
type ProgressStats struct {
	TotalTasks  int64
	DoneTasks   int64
	FailedTasks int64
	Percentage  float64
}

// GetProgressStats returns detailed progress statistics
func (p *WorkerPool) GetProgressStats() ProgressStats {
	total := atomic.LoadInt64(&p.totalTasks)
	done := atomic.LoadInt64(&p.doneTasks)
	failed := atomic.LoadInt64(&p.failedTasks)

	percentage := 0.0
	if total > 0 {
		percentage = float64(done) / float64(total) * 100
	}

	return ProgressStats{
		TotalTasks:  total,
		DoneTasks:   done,
		FailedTasks: failed,
		Percentage:  percentage,
	}
}

// BatchSubmit submits multiple tasks at once
func (p *WorkerPool) BatchSubmit(tasks []Task) error {
	for _, task := range tasks {
		if err := p.Submit(task); err != nil {
			return err
		}
	}
	return nil
}

// WaitWithProgress waits for completion and calls the progress callback periodically
func (p *WorkerPool) WaitWithProgress(interval int64, callback func(stats ProgressStats)) error {
	// Start a goroutine to report progress
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if callback != nil {
					callback(p.GetProgressStats())
				}
			}
		}
	}()

	err := p.Wait()
	close(done)

	// Report final progress
	if callback != nil {
		callback(p.GetProgressStats())
	}

	return err
}
