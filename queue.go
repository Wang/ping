package ping

import "sync"

type Queue struct {
	mu   sync.Mutex
	list map[string]*PingCmd
}

func NewQueue() *Queue {
	return &Queue{
		list: make(map[string]*PingCmd),
	}
}

func (q *Queue) Add(k string, p *PingCmd) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.list[k]; ok {
		return false
	}
	q.list[k] = p

	return true
}

func (q *Queue) Del(k string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.list, k)
}

func (q *Queue) Get(k string) (p *PingCmd, ok bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	p, ok = q.list[k]

	return
}
