package quartz

import "container/heap"

// PriorityQueue Item
type Item struct {
	Job      Job
	Trigger  Trigger
	priority int64 // item priority by next run time
	index    int   // maintained by the heap.Interface methods
}

// implements heap.Interface
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Head() *Item {
	return (*pq)[0]
}

func (pq *PriorityQueue) Update(item *Item, sleepTime int64) {
	item.priority = sleepTime
	heap.Fix(pq, item.index)
}
