package task

import (
	"testing"
)

type TestTaskTrace struct {
	NumTasks int
	Trace    []int
}

func (trace *TestTaskTrace) MakeTask(ok bool) *TestTaskImpl {
	task := &TestTaskImpl{Trace: trace, UID: trace.NumTasks, OK: ok}
	trace.NumTasks += 1
	return task
}

type TestTaskImpl struct {
	Trace *TestTaskTrace
	UID   int
	OK    bool
}

func (task *TestTaskImpl) Run(log TaskLog) bool {
	task.Trace.Trace = append(task.Trace.Trace, task.UID)
	return task.OK
}

func checkTrace(expected []int, trace *TestTaskTrace, t *testing.T) {
	actual := trace.Trace
	if len(expected) != len(actual) {
		t.Fatal(actual)
	}
	for i := 0; i < len(expected); i++ {
		if actual[i] != expected[i] {
			t.Fatal(i, expected[i], actual[i])
		}
	}
}

func runAndCheck(task TaskDecl, trace *TestTaskTrace, expectedResult bool, expectedTrace []int, t *testing.T) {
	log := &NullLog{}
	actualResult := task.Run(log)
	if actualResult != expectedResult {
		t.Fatal(expectedResult, actualResult)
	}
	checkTrace(expectedTrace, trace, t)
}

func TestBasicSanity(t *testing.T) {
	trace := &TestTaskTrace{}
	t0 := trace.MakeTask(false)
	runAndCheck(t0, trace, false, []int{0}, t)
}

func TestCommandTrue(t *testing.T) {
	task := &CommandTask{
		Args: []string{"true"},
	}
	log := &NullLog{}
	result := task.Run(log)
	if !result {
		t.Fatal(task.Args)
	}
}

func TestCommandFalse(t *testing.T) {
	task := &CommandTask{
		Args: []string{"false"},
	}
	log := &NullLog{}
	result := task.Run(log)
	if result {
		t.Fatal(task.Args)
	}
}
