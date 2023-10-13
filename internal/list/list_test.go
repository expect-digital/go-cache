package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList_PushFront(t *testing.T) {
	t.Parallel()

	l := New[string]()

	l.PushFront("a")
	assertList(t, []string{"a"}, l)

	l.PushFront("b")
	assertList(t, []string{"b", "a"}, l)

	l.PushFront("c")
	assertList(t, []string{"c", "b", "a"}, l)
}

func TestList_PushBack(t *testing.T) {
	t.Parallel()

	l := New[string]()

	l.PushBack("a")
	assertList(t, []string{"a"}, l)

	l.PushBack("b")
	assertList(t, []string{"a", "b"}, l)

	l.PushBack("c")
	assertList(t, []string{"a", "b", "c"}, l)
}

func TestList_Remove(t *testing.T) {
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

func TestList_InsertAfter(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")

	_ = l.InsertAfter("c", a)
	assertList(t, []string{"a", "c"}, l)

	_ = l.InsertAfter("b", a)
	assertList(t, []string{"a", "b", "c"}, l)
}

func TestList_InsertBefore(t *testing.T) {
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

func TestList_MoveAfter(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveAfter(b, a)
	assertList(t, []string{"a", "b"}, l)

	l.MoveAfter(a, b)
	assertList(t, []string{"b", "a"}, l)
}

func TestList_MoveToFront(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveToFront(a)
	assertList(t, []string{"a", "b"}, l)

	l.MoveToFront(b)
	assertList(t, []string{"b", "a"}, l)
}

func TestList_MoveToBack(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveToBack(b)
	assertList(t, []string{"a", "b"}, l)

	l.MoveToBack(a)
	assertList(t, []string{"b", "a"}, l)
}

func TestList_MoveBefore(t *testing.T) {
	t.Parallel()

	l := New[string]()

	a := l.PushBack("a")
	b := l.PushBack("b")

	l.MoveBefore(a, b)
	assertList(t, []string{"a", "b"}, l)

	l.MoveBefore(b, a)
	assertList(t, []string{"b", "a"}, l)
}

func assertList[V any](t *testing.T, expected []V, l *List[V]) {
	t.Helper()

	assert.Equal(t, len(expected), l.Len(), expected)

	if len(expected) == 0 {
		assert.Nil(t, l.Front(), expected)
		assert.Nil(t, l.Back(), expected)

		return
	}

	assert.Equal(t, l.Front().Value, expected[0], expected)
	assert.Equal(t, l.Back().Value, expected[len(expected)-1], expected)

	el := l.Front()

	for _, v := range expected {
		assert.Equal(t, v, el.Value, expected)
		el = el.Next()
	}

	el = l.Back()

	for i := len(expected) - 1; i >= 0; i-- {
		assert.Equal(t, expected[i], el.Value, expected)
		el = el.Prev()
	}
}
