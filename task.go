package crony

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

type Task struct {
	sync.Mutex
	fun     interface{}
	args    []interface{}
	nextRun time.Time
	lastRun time.Time
	tick    time.Duration
}

var (
	ErrTaskNotFunc       = errors.New("not a function")
	ErrTaskWrongNumOfArg = errors.New("wrong number of arguments")
	ErrTaskWrongArgType  = errors.New("wrong argument type")
)

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

func (t *Task) Func(f interface{}) *Task {
	t.fun = f
	return t
}

func (t *Task) Args(args ...interface{}) *Task {
	t.args = args
	return t
}

func (t *Task) Schedule(next time.Time) *Task {
	t.nextRun = next
	return t
}

func (t *Task) Ticker(d time.Duration) *Task {
	t.tick = d
	if t.nextRun.IsZero() {
		t.nextRun = time.Now().Add(d)
	}
	return t
}

func (t *Task) Run() (err error) {
	t.Lock()
	defer t.Unlock()

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
	v.Call(vargs)

	return
}

func (t *Task) ShouldRun() bool {
	return !t.nextRun.IsZero() && (t.nextRun.Before(time.Now()) || t.nextRun.Equal(time.Now()))
}
