package quartz

import "container/heap"

// Item is the PriorityQueue item.
type Item struct {
	Job      Job
	Trigger  Trigger
	priority int64 // item priority, backed by the next run time.
	index    int   // maintained by the heap.Interface methods.
}

// PriorityQueue implements the heap.Interface.
type PriorityQueue []*Item

// Len returns the PriorityQueue length.
func (pq PriorityQueue) Len() int { return len(pq) }

// Less is the items less comparator.
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

// Swap exchanges the indexes of the items.
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push implements the heap.Interface.Push.
// Adds x as element Len().
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

// Pop implements the heap.Interface.Pop.
// Removes and returns element Len() - 1.
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// Head returns the first item of a PriorityQueue without removing it.
func (pq *PriorityQueue) Head() *Item {
	return (*pq)[0]
}

// Remove removes and returns the element at index i from the PriorityQueue.
func (pq *PriorityQueue) Remove(i int) interface{} {
	return heap.Remove(pq, i)
}
