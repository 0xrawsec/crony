package crony

import (
	"context"
	"sync"
	"time"
)

const (
	PrioLow = iota << 1
	PrioMedium
	PrioHigh
)

var (
	DefaultSleep = time.Duration(500 * time.Millisecond)
)

type tasks struct {
	low    []*Task
	medium []*Task
	high   []*Task
}

func (t *tasks) ordered() (out []*Task) {
	out = make([]*Task, 0, len(t.low)+len(t.medium)+len(t.high))
	out = append(out, t.high...)
	out = append(out, t.medium...)
	out = append(out, t.low...)
	return
}

// Crony structure used to schedule tasks
type Crony struct {
	sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	tasks  *tasks
	sleep  time.Duration
}

func newCrony() *Crony {
	return &Crony{
		tasks: &tasks{
			make([]*Task, 0),
			make([]*Task, 0),
			make([]*Task, 0),
		},
		sleep: DefaultSleep,
	}
}

// New creates a new Crony structure
func New() (c *Crony) {
	c = newCrony()
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return
}

// New creates a new Crony structure from a context
func NewWithContext(ctx context.Context) (c *Crony) {
	c = newCrony()
	c.ctx = ctx
	return
}

// Sleep configures the time the Crony sleeps between
// every loop checking for tasks to execute
func (c *Crony) Sleep(d time.Duration) *Crony {
	c.sleep = d
	return c
}

func (c *Crony) Schedule(t *Task, prio int) {
	switch prio {
	case PrioHigh:
		c.tasks.high = append(c.tasks.high, t)
	case PrioLow:
		c.tasks.low = append(c.tasks.low, t)
	default:
		c.tasks.medium = append(c.tasks.medium, t)
	}
}

// Tasks returns the list of Tasks loaded in the Crony
// ordered by priority
func (c *Crony) Tasks() []*Task {
	return c.tasks.ordered()
}

func (c *Crony) start() {
	for {
		for _, t := range c.Tasks() {
			if c.ctx.Err() != nil {
				return
			}
			if t.ShouldRun() && !t.IsRunning() {
				if err := t.Run(); err != nil {
					panic(err)
				}
			}
		}
		time.Sleep(c.sleep)
	}
}

// Start starts the Crony's scheduling loop
func (c *Crony) Start() {
	c.Add(1)
	go func() {
		defer c.Done()
		c.start()
	}()
}

// Stop stops the Crony
func (c *Crony) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.Wait()
}
