package crony

import (
	"context"
	"testing"
	"time"

	"github.com/0xrawsec/toast"
)

var (
	asyncVar    = 0
	asyncTicker = time.Millisecond * 100
)

type testStruct struct{}

func (t *testStruct) HelloWorld(T *testing.T) {
	T.Log("Hello world from testStruct")
}

func empty() {}

func helloWorld(t *testing.T) {
	t.Log("Hello World")
}

func log(t *testing.T, s string) {
	t.Log(s)
}

func asyncFunc() {
	if asyncVar != 0 {
		panic("should be 0")
	}
	asyncVar++
	defer func() { asyncVar-- }()
	time.Sleep(asyncTicker * 2)

}

func TestTask(t *testing.T) {
	tick := time.Millisecond * 100
	tt := toast.FromT(t)

	tt.CheckErr(new(Task).Func(func() { t.Logf("Running task") }).Args().Run())
	tt.ExpectErr(new(Task).Func(func() { t.Logf("Running task") }).Args("test", 2).Run(), ErrTaskWrongNumOfArg)
	tt.ExpectErr(new(Task).Run(), ErrTaskNotFunc)

	tt.CheckErr(new(Task).Func(func(s string) { t.Logf(s) }).Args("string passed as argument").Run())
	tt.ExpectErr(new(Task).Func(func(s int) { t.Log(s) }).Args("string argument").Run(), ErrTaskWrongArgType)

	// calling a method of a structure
	tt.CheckErr(new(Task).Func(new(testStruct).HelloWorld).Args(t).Run())

	// test schedule
	task := new(Task).Func(empty).Schedule(time.Now().Add(tick))
	tt.Assert(!task.ShouldRun())
	time.Sleep(tick)
	tt.Assert(task.ShouldRun())
	tt.CheckErr(task.Run())
	// task should not be runnable
	tt.Assert(!task.ShouldRun())

	// test ticker
	task = new(Task).Func(empty).Ticker(tick)
	tt.Assert(!task.ShouldRun())
	time.Sleep(tick)
	tt.Assert(task.ShouldRun())
	task.Run()
	// task should not be runnable
	tt.Assert(!task.ShouldRun())
	time.Sleep(tick)
	// task should be runnable again
	tt.Assert(task.ShouldRun())
}

func TestCrony(t *testing.T) {
	tick := time.Second
	tt := toast.FromT(t)

	tt.ShouldPanic(func() {
		c := New()
		c.Schedule(new(Task).Ticker(time.Millisecond*200), PrioMedium)
		c.start()
	})

	t.Log("Testing basic crony")
	c := New().Sleep(time.Millisecond * 100)
	c.Schedule(NewTask("HelloWorld").Func(helloWorld).Args(t).Ticker(time.Millisecond*200), PrioMedium)
	c.Start()
	time.Sleep(tick * 2)
	c.Stop()

	t.Log("Testing crony with context")
	ctx, cancel := context.WithTimeout(context.Background(), tick*2)
	c = NewWithContext(ctx).Sleep(time.Millisecond * 100)
	c.Schedule(NewTask("HelloWorld").Func(helloWorld).Args(t).Ticker(time.Millisecond*200), PrioMedium)
	c.Start()
	c.Wait()
	cancel()

	t.Log("Testing crony with priorities")
	ctx, cancel = context.WithTimeout(context.Background(), tick*2)
	c = NewWithContext(ctx).Sleep(time.Millisecond * 100)
	c.Schedule(NewTask("Log high prio").Func(log).Args(t, "High prio").Ticker(time.Millisecond*200), PrioHigh)
	c.Schedule(NewTask("Log medium prio").Func(log).Args(t, "Medium prio").Ticker(time.Millisecond*400), PrioMedium)
	c.Schedule(NewTask("Log low prio").Func(log).Args(t, "Low prio").Ticker(time.Millisecond*600), PrioLow)
	for _, tk := range c.Tasks() {
		t.Logf("Scheduled task: %s", tk.Name)
	}
	c.Start()
	c.Wait()
	cancel()

}

func TestAsyncTask(t *testing.T) {
	tick := time.Second
	ctx, cancel := context.WithTimeout(context.Background(), tick*2)
	c := NewWithContext(ctx).Sleep(asyncTicker)
	c.Schedule(NewAsyncTask("Async Task").Func(asyncFunc).Ticker(asyncTicker), PrioHigh)
	c.Start()
	c.Wait()
	cancel()
}
