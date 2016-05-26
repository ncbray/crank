package workgraph

type NodeState int

const (
	WAITING NodeState = iota
	PENDING
	RUNNING
	SUCCESS
	ERROR
)

type Work interface {
	Invalidated()
	Run() bool
}

type Edge struct {
	Src       *Node
	Dst       *Node
	OrderOnly bool
}

func (e *Edge) Satisfied() bool {
	return e.Src.state == SUCCESS || e.Src.state == ERROR && e.OrderOnly
}

type Node struct {
	Work      Work
	Srcs      []Edge
	Dsts      []Edge
	waitCount int
	state     NodeState
	live      bool
	Prev      *Node
	Next      *Node
}

func (n *Node) Ready() bool {
	if !n.live || n.state != WAITING {
		return false
	}
	for _, e := range n.Srcs {
		if !e.Satisfied() {
			return false
		}
	}
	return true
}

type NodeCounts struct {
	Waiting   int
	Pending   int
	Running   int
	Success   int
	Error     int
	WaitCount int
}

type WorkGraph struct {
	Head      *Node
	Tail      *Node
	LiveNodes NodeCounts
	DeadNodes NodeCounts
}

func (g *WorkGraph) adjustCount(state NodeState, live bool, amt int, waitCount int) {
	var counts *NodeCounts
	if live {
		counts = &g.LiveNodes
	} else {
		counts = &g.DeadNodes
	}
	switch state {
	case WAITING:
		counts.Waiting += amt
	case PENDING:
		counts.Pending += amt
	case RUNNING:
		counts.Pending += amt
	case SUCCESS:
		counts.Success += amt
	case ERROR:
		counts.Error += amt
	default:
		panic(state)
	}
	counts.WaitCount += waitCount
}

func (g *WorkGraph) countNode(n *Node) {
	g.adjustCount(n.state, n.live, 1, n.waitCount)
}

func (g *WorkGraph) uncountNode(n *Node) {
	g.adjustCount(n.state, n.live, -1, -n.waitCount)
}

func (g *WorkGraph) setState(n *Node, state NodeState) {
	if n.state != state {
		g.uncountNode(n)
		n.state = state
		g.countNode(n)
	}
}

func (g *WorkGraph) setLive(n *Node, live bool) {
	if n.live != live {
		g.uncountNode(n)
		n.live = live
		g.countNode(n)
	}
}

func (g *WorkGraph) CreateNode(work Work) *Node {
	n := &Node{
		Work:  work,
		state: WAITING,
		live:  false,
	}
	g.countNode(n)
	return n
}

func (g *WorkGraph) CreateEdge(src *Node, dst *Node, ignore_error bool) {
	e := Edge{
		Src:       src,
		Dst:       dst,
		OrderOnly: ignore_error,
	}
	src.Dsts = append(src.Dsts, e)
	dst.Srcs = append(dst.Srcs, e)
	if !e.Satisfied() {
		g.adjustWaitCount(dst, 1)
	}
}

func (g *WorkGraph) appendPending(n *Node) {
	if n.Prev != nil || n.Next != nil {
		panic(n)
	}
	if g.Tail != nil {
		if g.Tail.Next != nil {
			panic(g.Tail)
		}
		g.Tail.Next = n
		n.Prev = g.Tail
		g.Tail = n
	} else {
		g.Head = n
		g.Tail = n
	}
}

func (g *WorkGraph) dequeuePending(n *Node) {
	if n.Prev != nil {
		if n.Prev.Next != n {
			panic(n.Prev)
		}
		n.Prev.Next = n.Next
	} else {
		if g.Head != n {
			panic(g.Head)
		}
		g.Head = n.Next
	}

	if n.Next != nil {
		if n.Next.Prev != n {
			panic(n.Next)
		}
		n.Next.Prev = n.Prev
	} else {
		if g.Tail != n {
			panic(g.Tail)
		}
		g.Tail = n.Prev
	}
	n.Prev = nil
	n.Next = nil
}

func (g *WorkGraph) adjustPending(n *Node) {
	switch n.state {
	case WAITING:
		if n.waitCount == 0 {
			g.setState(n, PENDING)
			g.appendPending(n)
		}
	case PENDING:
		if n.waitCount != 0 || !n.live {
			g.dequeuePending(n)
			g.setState(n, WAITING)
		}
	}
}

func (g *WorkGraph) adjustWaitCount(n *Node, amt int) {
	if n.live {
		g.LiveNodes.WaitCount += amt
	} else {
		g.DeadNodes.WaitCount += amt
	}
	n.waitCount += amt
	g.adjustPending(n)
}

func (g *WorkGraph) Invalidate(n *Node) {
	if n.state != SUCCESS && n.state != ERROR {
		return
	}
	for _, e := range n.Dsts {
		if e.Satisfied() {
			g.adjustWaitCount(e.Dst, 1)
		}
		if !e.OrderOnly {
			g.Invalidate(e.Dst)
		}
	}
	n.Work.Invalidated()
	g.setState(n, WAITING)
	g.adjustPending(n)
}

func (g *WorkGraph) markComplete(n *Node, state NodeState) {
	// HACK for testibility
	if n.state == PENDING {
		g.beginRunning(n)
	}

	if n.state != RUNNING {
		panic(n.state)
	}

	g.setState(n, state)
	for _, e := range n.Dsts {
		if e.Satisfied() {
			g.adjustWaitCount(e.Dst, -1)
		}
	}
}

func (g *WorkGraph) markSuccess(n *Node) {
	g.markComplete(n, SUCCESS)
}

func (g *WorkGraph) markError(n *Node) {
	g.markComplete(n, ERROR)
}

func (g *WorkGraph) MarkLive(n *Node) {
	if !n.live {
		g.setLive(n, true)
		g.adjustPending(n)
		for _, e := range n.Srcs {
			g.MarkLive(e.Src)
		}
	}
}

func (g *WorkGraph) beginRunning(n *Node) {
	g.dequeuePending(n)
	g.setState(n, RUNNING)
}

func (g *WorkGraph) Run() {
	for g.Head != nil {
		current := g.Head
		g.beginRunning(current)
		if current.Work.Run() {
			g.markSuccess(current)
		} else {
			g.markError(current)
		}
	}
}
