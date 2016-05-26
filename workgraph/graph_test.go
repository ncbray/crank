package workgraph

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type FakeWorkManager struct {
	Trace      []int
	CurrentUID int
}

func (m *FakeWorkManager) Create(result bool) *FakeWork {
	w := &FakeWork{Manager: m, UID: m.CurrentUID, Result: result}
	m.CurrentUID += 1
	return w
}

type FakeWork struct {
	Manager *FakeWorkManager
	UID     int
	Result  bool
}

func (w *FakeWork) Invalidated() {
}

func (w *FakeWork) Run() bool {
	w.Manager.Trace = append(w.Manager.Trace, w.UID)
	return w.Result
}

func checkCounts(t *testing.T, g *WorkGraph, live NodeCounts, dead NodeCounts) {
	assert.Equal(t, g.LiveNodes.Waiting, live.Waiting, "live waiting count wrong")
	assert.Equal(t, g.LiveNodes.Pending, live.Pending, "live pending count wrong")
	assert.Equal(t, g.LiveNodes.Running, live.Running, "live running count wrong")
	assert.Equal(t, g.LiveNodes.Success, live.Success, "live success count wrong")
	assert.Equal(t, g.LiveNodes.Error, live.Error, "live error count wrong")
	assert.Equal(t, g.LiveNodes.WaitCount, live.WaitCount, "live total wait count wrong")

	assert.Equal(t, g.DeadNodes.Waiting, dead.Waiting, "dead waiting count wrong")
	assert.Equal(t, g.DeadNodes.Pending, dead.Pending, "dead pending count wrong")
	assert.Equal(t, g.DeadNodes.Running, dead.Running, "dead running count wrong")
	assert.Equal(t, g.DeadNodes.Success, dead.Success, "dead success count wrong")
	assert.Equal(t, g.DeadNodes.Error, dead.Error, "dead error count wrong")
	assert.Equal(t, g.DeadNodes.WaitCount, dead.WaitCount, "dead total wait count wrong")
}

func TestNodeReadySingle(t *testing.T) {
	m := &FakeWorkManager{}
	g := &WorkGraph{}
	n0 := g.CreateNode(m.Create(true))
	g.MarkLive(n0)

	assert.Equal(t, PENDING, n0.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 0, 0, 0}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)

	checkCounts(t, g, NodeCounts{0, 0, 0, 1, 0, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 0, 0, 0}, NodeCounts{})

	g.markError(n0)
	assert.Equal(t, ERROR, n0.state)

	checkCounts(t, g, NodeCounts{0, 0, 0, 0, 1, 0}, NodeCounts{})
}

func TestNodeReadyOne(t *testing.T) {
	m := &FakeWorkManager{}
	g := &WorkGraph{}
	n0 := g.CreateNode(m.Create(true))
	n1 := g.CreateNode(m.Create(true))
	g.CreateEdge(n0, n1, false)
	g.MarkLive(n1)

	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 0, 0, 1}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, PENDING, n1.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 1, 0, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 0, 0, 1}, NodeCounts{})

	g.markError(n0)
	assert.Equal(t, ERROR, n0.state)
	assert.Equal(t, WAITING, n1.state)

	checkCounts(t, g, NodeCounts{1, 0, 0, 0, 1, 1}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 0, 0, 1}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, PENDING, n1.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 1, 0, 0}, NodeCounts{})
}

func TestNodeReadyOneOrderOnly(t *testing.T) {
	m := &FakeWorkManager{}
	g := &WorkGraph{}
	n0 := g.CreateNode(m.Create(true))
	n1 := g.CreateNode(m.Create(true))
	g.CreateEdge(n0, n1, true)
	g.MarkLive(n1)

	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 0, 0, 1}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, PENDING, n1.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 1, 0, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 0, 0, 1}, NodeCounts{})

	g.markError(n0)
	assert.Equal(t, ERROR, n0.state)
	assert.Equal(t, PENDING, n1.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 0, 1, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 0, 0, 1}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, PENDING, n1.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 1, 0, 0}, NodeCounts{})

	g.markSuccess(n1)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, SUCCESS, n1.state)

	checkCounts(t, g, NodeCounts{0, 0, 0, 2, 0, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, SUCCESS, n1.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 1, 0, 1}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, SUCCESS, n1.state)

	checkCounts(t, g, NodeCounts{0, 0, 0, 2, 0, 0}, NodeCounts{})
}

func TestNodeReadyTwoToOne(t *testing.T) {
	m := &FakeWorkManager{}
	g := &WorkGraph{}
	n0 := g.CreateNode(m.Create(true))
	n1 := g.CreateNode(m.Create(true))
	n2 := g.CreateNode(m.Create(true))
	g.CreateEdge(n0, n2, false)
	g.CreateEdge(n1, n2, true)
	g.MarkLive(n2)

	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, PENDING, n1.state)
	assert.Equal(t, WAITING, n2.state)

	checkCounts(t, g, NodeCounts{1, 2, 0, 0, 0, 2}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, PENDING, n1.state)
	assert.Equal(t, WAITING, n2.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 1, 0, 1}, NodeCounts{})

	g.markSuccess(n1)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, SUCCESS, n1.state)
	assert.Equal(t, PENDING, n2.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 2, 0, 0}, NodeCounts{})

	g.Invalidate(n1)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, PENDING, n1.state)
	assert.Equal(t, WAITING, n2.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 1, 0, 1}, NodeCounts{})

	g.markError(n1)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, ERROR, n1.state)
	assert.Equal(t, PENDING, n2.state)

	checkCounts(t, g, NodeCounts{0, 1, 0, 1, 1, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, ERROR, n1.state)
	assert.Equal(t, WAITING, n2.state)

	checkCounts(t, g, NodeCounts{1, 1, 0, 0, 1, 1}, NodeCounts{})

	g.markError(n0)
	assert.Equal(t, ERROR, n0.state)
	assert.Equal(t, ERROR, n1.state)
	assert.Equal(t, WAITING, n2.state)

	checkCounts(t, g, NodeCounts{1, 0, 0, 0, 2, 1}, NodeCounts{})
}

func TestNodeReadyThreeInARow(t *testing.T) {
	m := &FakeWorkManager{}
	g := &WorkGraph{}
	n0 := g.CreateNode(m.Create(true))
	n1 := g.CreateNode(m.Create(true))
	n2 := g.CreateNode(m.Create(true))
	g.CreateEdge(n0, n1, false)
	g.CreateEdge(n1, n2, false)
	g.MarkLive(n2)

	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)
	assert.Equal(t, WAITING, n2.state)
	checkCounts(t, g, NodeCounts{2, 1, 0, 0, 0, 2}, NodeCounts{})

	g.markSuccess(n0)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, PENDING, n1.state)
	assert.Equal(t, WAITING, n2.state)
	checkCounts(t, g, NodeCounts{1, 1, 0, 1, 0, 1}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)
	assert.Equal(t, WAITING, n2.state)
	checkCounts(t, g, NodeCounts{2, 1, 0, 0, 0, 2}, NodeCounts{})

	g.markSuccess(n0)
	g.markSuccess(n1)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, SUCCESS, n1.state)
	assert.Equal(t, PENDING, n2.state)
	checkCounts(t, g, NodeCounts{0, 1, 0, 2, 0, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)
	assert.Equal(t, WAITING, n2.state)
	checkCounts(t, g, NodeCounts{2, 1, 0, 0, 0, 2}, NodeCounts{})

	g.markSuccess(n0)
	g.markSuccess(n1)
	g.markSuccess(n2)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, SUCCESS, n1.state)
	assert.Equal(t, SUCCESS, n2.state)
	checkCounts(t, g, NodeCounts{0, 0, 0, 3, 0, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)
	assert.Equal(t, WAITING, n2.state)
	checkCounts(t, g, NodeCounts{2, 1, 0, 0, 0, 2}, NodeCounts{})

	g.markSuccess(n0)
	g.markSuccess(n1)
	g.markError(n2)
	assert.Equal(t, SUCCESS, n0.state)
	assert.Equal(t, SUCCESS, n1.state)
	assert.Equal(t, ERROR, n2.state)
	checkCounts(t, g, NodeCounts{0, 0, 0, 2, 1, 0}, NodeCounts{})

	g.Invalidate(n0)
	assert.Equal(t, PENDING, n0.state)
	assert.Equal(t, WAITING, n1.state)
	assert.Equal(t, WAITING, n2.state)
	checkCounts(t, g, NodeCounts{2, 1, 0, 0, 0, 2}, NodeCounts{})

	g.Run()
	assert.Equal(t, []int{0, 1, 2}, m.Trace)
}
