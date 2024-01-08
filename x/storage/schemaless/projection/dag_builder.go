package projection

import (
	"container/list"
	"crypto/md5"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

var _ Builder = &DAGBuilder{}

func NewDAGBuilder() *DAGBuilder {
	return &DAGBuilder{
		nodesFromTo: make(map[Node]*list.List),
		nodesToFrom: make(map[Node]*list.List),
		dag:         nil,
		ctx: &DefaultContext{
			name: "root",
		},
	}
}

type DAGBuilder struct {
	nodesFromTo map[Node]*list.List
	nodesToFrom map[Node]*list.List
	dag         Node
	ctx         *DefaultContext
}

func (d *DAGBuilder) nextNumber() int {
	return len(d.nodesFromTo)
}

func (d *DAGBuilder) addNode(node Node) {
	// check if node name is already in use, yes - fail
	for n := range d.nodesFromTo {
		if n == nil || node == nil {
			panic("node is nil")
		}

		if GetCtx(n).Name() == GetCtx(node).Name() {
			panic(fmt.Sprintf("node name %s is already in use", GetCtx(node).Name()))
		}
	}

	if _, ok := d.nodesFromTo[node]; !ok {
		d.nodesFromTo[node] = list.New()
	}
	if _, ok := d.nodesToFrom[node]; !ok {
		d.nodesToFrom[node] = list.New()
	}
}

func (d *DAGBuilder) addDependency(from, to Node) {
	if _, ok := d.nodesFromTo[from]; !ok {
		d.addNode(from)
	}
	if _, ok := d.nodesFromTo[to]; !ok {
		d.addNode(to)
	}

	d.nodesFromTo[from].PushBack(to)
	d.nodesToFrom[to].PushBack(from)
}

// DoLoad loads data from a source. This node is a root of the DAG. DAG can have many DoLoad nodesFromTo.
func (d *DAGBuilder) Load(f Handler, opts ...ContextOptionFunc) Builder {
	ctx := d.ctx.Scope(fmt.Sprintf("DoLoad%d", d.nextNumber()))
	for _, opt := range opts {
		opt(ctx)
	}

	if d.dag != nil {
		panic(fmt.Errorf("load must be the first node in the DAG,"+
			"but it's being connected to %s node", GetCtx(d.dag).Name()))
	}

	node := &DoLoad{
		Ctx:    ctx,
		OnLoad: f,
	}

	d.addNode(node)

	return &DAGBuilder{
		nodesFromTo: d.nodesFromTo,
		nodesToFrom: d.nodesToFrom,
		dag:         node,
		ctx:         ctx,
	}
}

func (d *DAGBuilder) Window(opts ...ContextOptionFunc) Builder {
	ctx := d.ctx.Scope(fmt.Sprintf("Window%d", d.nextNumber()))
	for _, opt := range opts {
		opt(ctx)
	}

	node := &DoWindow{
		Ctx:   ctx,
		Input: d.dag,
	}

	d.addDependency(d.dag, node)

	return &DAGBuilder{
		nodesFromTo: d.nodesFromTo,
		nodesToFrom: d.nodesToFrom,
		dag:         node,
		ctx:         ctx,
	}
}

func (d *DAGBuilder) Map(f Handler, opts ...ContextOptionFunc) Builder {
	ctx := d.ctx.Scope(fmt.Sprintf("DoWindow%d", d.nextNumber()))
	for _, opt := range opts {
		opt(ctx)
	}

	node := &DoMap{
		Ctx:   ctx,
		OnMap: f,
		Input: d.dag,
	}

	d.addDependency(d.dag, node)

	return &DAGBuilder{
		nodesFromTo: d.nodesFromTo,
		nodesToFrom: d.nodesToFrom,
		dag:         node,
		ctx:         ctx,
	}
}

func (d *DAGBuilder) Join(a, b Builder, opts ...ContextOptionFunc) Builder {
	ctx := d.ctx.Scope(fmt.Sprintf("DoJoin%d", d.nextNumber()))
	for _, opt := range opts {
		opt(ctx)
	}

	node := &DoJoin{
		Ctx: ctx,
		Input: []Node{
			a.(*DAGBuilder).dag,
			b.(*DAGBuilder).dag,
		},
	}

	d.addDependency(a.(*DAGBuilder).dag, node)
	d.addDependency(b.(*DAGBuilder).dag, node)

	return &DAGBuilder{
		nodesFromTo: d.nodesFromTo,
		nodesToFrom: d.nodesToFrom,
		dag:         node,
		ctx:         ctx,
	}
}
func (d *DAGBuilder) Build() []Node {
	result := ReverseSort(Sort(d))
	log.Debugf("Build graph:\n%s\n", ToMermaidGraphWithOrder(d, result))

	return result
}

func (d *DAGBuilder) GetByName(name string) (*DAGBuilder, error) {
	//TODO fix me!

	for node := range d.nodesFromTo {
		if node == nil {
			//continue
			panic("node is nil")
		}

		if GetCtx(node).Name() == name {
			return &DAGBuilder{
				nodesFromTo: d.nodesFromTo,
				dag:         node,
				ctx:         GetCtx(node),
			}, nil
		}
	}
	return nil, ErrNotFound
}

func HashNode(n Node) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(GetCtx(n).Name())))
}

func ToMermaidGraph(dag *DAGBuilder) string {
	return ToMermaidGraphWithOrder(dag, dag.Build())
}

func ToMermaidGraphWithOrder(dag *DAGBuilder, order []Node) string {
	var sb strings.Builder
	sb.WriteString("graph TD\n")
	for _, from := range order {
		sb.WriteString(fmt.Sprintf("\t"+`%s["%s: %s"]`+"\n",
			HashNode(from),
			NodeToString(from),
			GetCtx(from).Name()))
	}

	for _, from := range order {
		tos := dag.nodesFromTo[from]
		for to := tos.Front(); to != nil; to = to.Next() {
			sb.WriteString(fmt.Sprintf("\t%s --> %s\n",
				HashNode(from),
				HashNode(to.Value.(Node))))
		}
	}
	return sb.String()
}

// Sort sorts nodesFromTo in topological order
// https://en.wikipedia.org/wiki/Topological_sorting
// using Kahn's algorithm
func Sort(dag *DAGBuilder) []Node {
	// copy nodesFromTo
	nodesFromTo := make(map[Node]*list.List, len(dag.nodesFromTo))
	for node, froms := range dag.nodesFromTo {
		nodesFromTo[node] = list.New()
		for e := froms.Front(); e != nil; e = e.Next() {
			nodesFromTo[node].PushBack(e.Value)
		}
	}

	// copy nodesToFrom
	nodesToFrom := make(map[Node]*list.List, len(dag.nodesToFrom))
	for node, tos := range dag.nodesToFrom {
		nodesToFrom[node] = list.New()
		for e := tos.Front(); e != nil; e = e.Next() {
			nodesToFrom[node].PushBack(e.Value)
		}
	}

	// L <- Empty list that will contain the sorted elements
	L := make([]Node, 0, len(nodesFromTo))
	// S <- Set of all nodesFromTo with no incoming edges
	// in our case, those should be only DoLoad nodes
	S := make([]Node, 0, len(nodesFromTo))
	Sm := make(map[Node]struct{}, len(nodesFromTo))
	for node, froms := range nodesToFrom {
		if froms.Len() == 0 {
			// act like a set
			if _, ok := Sm[node]; !ok {
				S = append(S, node)
				Sm[node] = struct{}{}
			}
		}
	}

	// while S is non-empty do
	for len(S) > 0 {
		// remove a node n from S
		n := S[0]
		S = S[1:]
		delete(Sm, n)

		// add n to tail of L
		L = append(L, n)

		// termination nodes, may not have any outgoing edges
		if nodesFromTo[n] == nil {
			continue
		}

		// for each node m with an edge e from n to m do
		for mEl := nodesFromTo[n].Front(); mEl != nil; {
			m := mEl.Value.(Node)
			// remove edge e from the graph
			mCopy := mEl
			mEl = mEl.Next()
			nodesFromTo[n].Remove(mCopy)

			// remove edge e from the graph
			for nEl := nodesToFrom[m].Front(); nEl != nil; nEl = nEl.Next() {
				if nEl.Value.(Node) == n {
					nodesToFrom[m].Remove(nEl)
					break
				}
			}

			// if m has no other incoming edges then insert m into S
			if nodesToFrom[m].Len() == 0 {
				// act like a set
				if _, ok := Sm[m]; !ok {
					S = append(S, m)
					Sm[m] = struct{}{}
				}
			}
		}
	}
	// if graph has edges then
	for node, tos := range nodesFromTo {
		// return error (graph has at least one cycle)
		if tos.Len() > 0 {
			//for to := tos.Front(); to != nil; to = to.Next() {
			//	log.Debugf("node %s has edge to %s \n", GetCtx(node).Name(), GetCtx(to.Value.(Node)).Name())
			//}
			panic(fmt.Errorf("graph has at least one cycle; node %s has %d edges \n", GetCtx(node).Name(), tos.Len()))
		}
	}

	// return L (a topologically sorted order)
	return L
}

func ReverseSort(nodes []Node) []Node {
	reversed := make([]Node, len(nodes))
	for i, node := range nodes {
		reversed[len(nodes)-1-i] = node
	}
	return reversed
}
