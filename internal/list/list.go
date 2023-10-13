package list

// Element represents a list element.
type Element[V any] struct {
	Value  V
	next   *Element[V]
	prev   *Element[V]
	isRoot bool
}

// Next returns the next list element or nil if it is the last element.
func (e *Element[V]) Next() *Element[V] {
	if e.isRoot {
		return nil
	}

	return e.next
}

// Prev returns the previous list element or nil if it is the first element.
func (e *Element[V]) Prev() *Element[V] {
	if e.isRoot {
		return nil
	}

	return e.prev
}

// List represents a doubly linked list.
type List[V any] struct {
	n    int
	root Element[V]
}

// New returns a new doubly linked list.
func New[V any]() *List[V] {
	l := new(List[V])

	l.root.isRoot = true
	l.root.next = &l.root
	l.root.prev = &l.root

	return l
}

func (l *List[V]) insert(e, at *Element[V]) *Element[V] {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	l.n++

	return e
}

func (l *List[V]) move(element, after *Element[V]) {
	if element == after || element.prev == after {
		return
	}

	element.prev.next = element.next
	element.next.prev = element.prev

	element.prev = after
	element.next = after.next
	element.prev.next = element
	element.next.prev = element
}

// Back returns the last element of list or nil if the list is empty.
func (l *List[V]) Back() *Element[V] {
	if l.root.prev == &l.root {
		return nil
	}

	return l.root.prev
}

// Front returns the first element of list or nil if the list is empty.
func (l *List[V]) Front() *Element[V] {
	if l.root.next == &l.root {
		return nil
	}

	return l.root.next
}

// InsertAfter inserts e after at.
func (l *List[V]) InsertAfter(v V, at *Element[V]) *Element[V] {
	return l.insert(&Element[V]{Value: v}, at)
}

// InsertBefore inserts e before at.
func (l *List[V]) InsertBefore(v V, at *Element[V]) *Element[V] {
	return l.insert(&Element[V]{Value: v}, at.prev)
}

// Len returns the number of elements of list.
func (l *List[V]) Len() int { return l.n }

// MoveAfter moves e after after.
func (l *List[V]) MoveAfter(e, after *Element[V]) {
	l.move(e, after)
}

// MoveBefore moves e before before.
func (l *List[V]) MoveBefore(e, before *Element[V]) {
	l.move(e, before.prev)
}

// MoveToBack moves e to the back.
func (l *List[V]) MoveToBack(e *Element[V]) {
	l.move(e, l.root.prev)
}

// MoveToFront moves e to the front.
func (l *List[V]) MoveToFront(e *Element[V]) {
	l.move(e, &l.root)
}

// PushBack inserts e at the back.
func (l *List[V]) PushBack(v V) *Element[V] {
	return l.InsertAfter(v, l.root.prev)
}

// PushFront inserts e at the front.
func (l *List[V]) PushFront(v V) *Element[V] {
	return l.InsertAfter(v, &l.root)
}

// Remove removes e from list.
func (l *List[V]) Remove(e *Element[V]) V { //nolint:ireturn
	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = nil
	e.next = nil

	l.n--

	return e.Value
}
