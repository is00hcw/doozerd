package paxos

import (
	"log"
)

type Result struct {
	seqn uint64
	v string
}

type instReq struct {
	seqn uint64
	ch chan *Instance
}

type Manager struct{
	self string
	nodes []string
	learned chan Result
	reqs chan instReq
	seqns chan uint64
	start uint64
	logger *log.Logger
}

func NewManager(start uint64, self string, nodes []string, logger *log.Logger) *Manager {
	m := &Manager{
		self: self,
		nodes: nodes,
		learned: make(chan Result),
		reqs: make(chan instReq),
		seqns: make(chan uint64),
		start: start,
		logger: logger,
	}
	return m
}

func (m *Manager) Init(outs Putter) {
	go func() {
		instances := make(map[uint64]*Instance)
		for req := range m.reqs {
			inst, ok := instances[req.seqn]
			if !ok {
				xx := NewCluster(m.self, m.nodes, nil)
				selfIndex := xx.SelfIndex()

				// TODO this ugly cast will go away when we make a proper
				// cluster type
				cluster := fakeCluster{PutWrapper{req.seqn, 1, outs}, uint64(len(m.nodes)), selfIndex}
				inst = NewInstance(cluster, m.logger)
				instances[req.seqn] = inst
				go func() {
					m.learned <- Result{req.seqn, inst.Value()}
				}()
			}
			req.ch <- inst
		}
	}()

	// Generate an infinite stream of sequence numbers (seqns).
	go func() {
		for n := m.start; ; n++ {
			m.seqns <- n
		}
	}()
}

func (m *Manager) getInstance(seqn uint64) *Instance {
	ch := make(chan *Instance)
	m.reqs <- instReq{seqn, ch}
	return <-ch
}

func (m *Manager) Put(msg Msg) {
	m.getInstance(msg.Seqn).Put(msg)
}

func (m *Manager) Propose(v string) string {
	seqn := <-m.seqns
	inst := m.getInstance(seqn)
	m.logger.Logf("paxos %d propose -> %q", seqn, v)
	inst.Propose(v)
	return inst.Value()
}

func (m *Manager) Recv() (uint64, string) {
	result := <-m.learned
	m.logger.Logf("paxos %d learned <- %q", result.seqn, result.v)
	return result.seqn, result.v
}

