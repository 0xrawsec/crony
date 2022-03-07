package crony

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Task structure describing a task to execute
// by a Crony
type Task struct {
	sync.Mutex
	fun     interface{}
	args    []interface{}
	nextRun time.Time
	lastRun time.Time
	tick    time.Duration
	running bool
	async   bool

	Name string
}

var (
	ErrTaskNotFunc       = errors.New("not a function")
	ErrTaskWrongNumOfArg = errors.New("wrong number of arguments")
	ErrTaskWrongArgType  = errors.New("wrong argument type")
)

// NewTask creates a new Task with a name
func NewTask(name string) *Task {
	return &Task{Name: name}
}

// NewAsyncTask creates a new asynchronous Task with a name.
// An asynchronous task will run in a separate go routine and
// will de facto not be blocking
func NewAsyncTask(name string) *Task {
	return NewTask(name).Async()
}

func (t *Task) control() error {
	v := reflect.ValueOf(t.fun)
	ft := reflect.TypeOf(t.fun)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("%w %T", ErrTaskNotFunc, t.fun)
	}

	if ft.NumIn() != len(t.args) {
		return fmt.Errorf("%w got %d expecting %d", ErrTaskWrongNumOfArg, len(t.args), ft.NumIn())
	}

	for i := 0; i < ft.NumIn(); i++ {
		funcArgType := ft.In(i)
		argType := reflect.TypeOf(t.args[i])
		if funcArgType.Kind() != argType.Kind() {
			return fmt.Errorf("%w got %s expecting %s", ErrTaskWrongArgType, argType.Name(), funcArgType.Name())
		}
	}

	return nil
}

func (t *Task) updateSchedule() {
	t.lastRun = time.Now()
	t.nextRun = time.Time{}
	if t.tick > 0 {
		t.nextRun = t.lastRun.Add(t.tick)
	}
}

// Func sets the fuction to execute by the Task
func (t *Task) Func(f interface{}) *Task {
	t.fun = f
	return t
}

// Args sets the arguments used by the function to execute
func (t *Task) Args(args ...interface{}) *Task {
	t.args = args
	return t
}

// Schedule schedule a time at which a Task must be ran
func (t *Task) Schedule(next time.Time) *Task {
	t.nextRun = next
	return t
}

// Ticker sets ticker's duration to run Task at every tick
func (t *Task) Ticker(d time.Duration) *Task {
	t.tick = d
	if t.nextRun.IsZero() {
		t.nextRun = time.Now().Add(d)
	}
	return t
}

// Async makes the Task asynchronous
func (t *Task) Async() *Task {
	t.async = true
	return t
}

// IsRunning returns true if the Task is running
func (t *Task) IsRunning() bool {
	return t.running
}

// Run runs a Task and returns a error in case the
// function cannot be ran
func (t *Task) Run() (err error) {
	t.Lock()
	defer t.Unlock()
	t.running = true

	v := reflect.ValueOf(t.fun)
	vargs := make([]reflect.Value, 0, len(t.args))

	if err = t.control(); err != nil {
		return
	}

	for _, arg := range t.args {
		vargs = append(vargs, reflect.ValueOf(arg))
	}

	// we schedule next run
	t.updateSchedule()
	if t.async {
		go func() {
			defer func() { t.running = false }()
			v.Call(vargs)
		}()
	} else {
		defer func() { t.running = false }()
		v.Call(vargs)
	}

	return
}

// ShouldRun returns true if the Task has to run
func (t *Task) ShouldRun() bool {
	return !t.nextRun.IsZero() && (t.nextRun.Before(time.Now()) || t.nextRun.Equal(time.Now()))
}
