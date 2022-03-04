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

func New() (c *Crony) {
	c = newCrony()
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return
}

func NewWithContext(ctx context.Context) (c *Crony) {
	c = newCrony()
	c.ctx = ctx
	return
}

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

func (c *Crony) Tasks() []*Task {
	return c.tasks.ordered()
}

func (c *Crony) start() {
	for {
		for _, t := range c.Tasks() {
			if c.ctx.Err() != nil {
				return
			}
			if t.ShouldRun() {
				if err := t.Run(); err != nil {
					panic(err)
				}
			}
		}
		time.Sleep(c.sleep)
	}
}

func (c *Crony) Start() {
	c.Add(1)
	go func() {
		defer c.Done()
		c.start()
	}()
}

func (c *Crony) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.Wait()
}
