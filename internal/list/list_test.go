package list

import (
	"testing"
)

func TestListPushFront(t *testing.T) {
	t.Parallel()

	l := New[string]()

	l.PushFront("a")
	assertList(t, []string{"a"}, l)

	l.PushFront("b")
	assertList(t, []string{"b", "a"}, l)

	l.PushFront("c")
	assertList(t, []string{"c", "b", "a"}, l)
}

func TestListPushBack(t *testing.T) {
	t.Parallel()

	l := New[string]()

	l.PushBack("a")
	assertList(t, []string{"a"}, l)

	l.PushBack("b")
	assertList(t, []string{"a", "b"}, l)

	l.PushBack("c")
	assertList(t, []string{"a", "b", "c"}, l)
}

func TestListRemove(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")
	c := l.PushBack("c")
	d := l.PushBack("d")

	// remove el from the middle
	l.Remove(b)
	assertList(t, []string{"a", "c", "d"}, l)

	// remove the first el
	l.Remove(a)
	assertList(t, []string{"c", "d"}, l)

	// remove the last el
	l.Remove(d)
	assertList(t, []string{"c"}, l)

	// remove the last remaining el
	l.Remove(c)
	assertList(t, nil, l)
}

func TestListInsertAfter(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")

	_ = l.InsertAfter("c", a)
	assertList(t, []string{"a", "c"}, l)

	_ = l.InsertAfter("b", a)
	assertList(t, []string{"a", "b", "c"}, l)
}

func TestListInsertBefore(t *testing.T) {
	t.Parallel()

	l := New[string]()

	c := l.PushBack("c")

	// insert before first element
	_ = l.InsertBefore("a", c)
	assertList(t, []string{"a", "c"}, l)

	// insert before last element
	_ = l.InsertBefore("b", c)
	assertList(t, []string{"a", "b", "c"}, l)
}

func TestListMoveAfter(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveAfter(b, a)
	assertList(t, []string{"a", "b"}, l)

	l.MoveAfter(a, b)
	assertList(t, []string{"b", "a"}, l)
}

func TestListMoveToFront(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveToFront(a)
	assertList(t, []string{"a", "b"}, l)

	l.MoveToFront(b)
	assertList(t, []string{"b", "a"}, l)
}

func TestListMoveToBack(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveToBack(b)
	assertList(t, []string{"a", "b"}, l)

	l.MoveToBack(a)
	assertList(t, []string{"b", "a"}, l)
}

func TestListMoveBefore(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveBefore(a, b)
	assertList(t, []string{"a", "b"}, l)

	l.MoveBefore(b, a)
	assertList(t, []string{"b", "a"}, l)
}

func assertList[V comparable](t *testing.T, expected []V, l *List[V]) {
	t.Helper()

	if len(expected) == 0 {
		if l.Front() != nil {
			t.Errorf("want nil front, got %v", l.Front())
		}

		if l.Back() != nil {
			t.Errorf("want nil back, got %v", l.Back())
		}

		return
	}

	if expected[0] != l.Front().Value {
		t.Errorf("want front %v, got %v", expected[0], l.Front().Value)
	}

	if expected[len(expected)-1] != l.Back().Value {
		t.Errorf("want back %v, got %v", expected[len(expected)-1], l.Back().Value)
	}

	el := l.Front()

	for i, v := range expected {
		if v != el.Value {
			t.Errorf("want %v at %d, got %v", v, i, el.Value)
		}

		el = el.Next()
	}

	el = l.Back()

	for i := len(expected) - 1; i >= 0; i-- {
		if expected[i] != el.Value {
			t.Errorf("want %v at %d, got %v", expected[i], i, el.Value)
		}

		el = el.Prev()
	}
}
