package lru

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.expect.digital/cache/internal/list"
)

const defaultSize = 1024

var ErrNotFound = errors.New("not found")

// zeroValue returns the zero value of the type.
func zeroValue[T any]() (zero T) { //nolint:ireturn
	return
}

type getterResult[V any] struct {
	err   error
	value V
}

// Cache is a least recently used cache.
type Cache[K comparable, V any] struct {
	n       int
	ttl     time.Duration
	getter  Getter[K, V]
	onEvict OnEvict[V]
	cache   *list.List[listValue[K, V]]
	lookup  map[K]*list.Element[listValue[K, V]]
	pending map[K][]chan getterResult[V]
	mu      sync.RWMutex
}

// Size returns the max size of the cache.
func (c *Cache[K, V]) Size() int {
	return c.n
}

// Len returns the length of the values stored in the cache.
func (c *Cache[K, V]) Len() int {
	return c.cache.Len()
}

// Get returns the value associated with the key from the cache. If the value is not found,
// the value is populated by the getter.
// TODO: too many locks?
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) { //nolint:ireturn
	for {
		c.mu.RLock()

		el, ok := c.lookup[key]
		if !ok {
			c.mu.RUnlock()

			return c.populateByGetter(ctx, key)
		}

		if el.Value.exp.IsZero() || el.Value.exp.After(time.Now()) {
			c.mu.RUnlock()

			c.mu.Lock()
			// Check again in case another goroutine removed the element.
			if _, ok := c.lookup[key]; ok {
				c.cache.MoveToFront(el)
				c.mu.Unlock()

				return el.Value.val, nil
			}
			c.mu.Unlock()
		} else {
			c.mu.RUnlock()

			c.mu.Lock()
			err := c.evict(ctx, el)
			c.mu.Unlock()

			if err != nil {
				return zeroValue[V](), fmt.Errorf("evict expired value: %w", err)
			}

			return c.populateByGetter(ctx, key)
		}
	}
}

func (c *Cache[K, V]) populateByGetter(ctx context.Context, key K) (V, error) { //nolint:ireturn
	if c.getter == nil {
		return zeroValue[V](), fmt.Errorf("value not found for key: %v: %w", key, ErrNotFound)
	}

	c.mu.Lock()

	ch := make(chan getterResult[V], 1)
	defer close(ch)

	c.pending[key] = append(c.pending[key], ch)
	n := len(c.pending[key])

	c.mu.Unlock()

	if n == 1 {
		go c.execGetter(ctx, key)
	}

	msg := <-ch

	if msg.err != nil {
		return zeroValue[V](), fmt.Errorf("get value by getter for key: %v: %w", key, msg.err)
	}

	// Add the new value to the cache.
	if err := c.Set(ctx, key, msg.value); err != nil {
		return zeroValue[V](), fmt.Errorf("set value for key: %v: %w", key, err)
	}

	return msg.value, nil
}

func (c *Cache[K, V]) execGetter(ctx context.Context, key K) {
	var (
		v   V
		err error
	)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("exec getter for key: %v: %v", key, r)
		}

		c.mu.Lock()

		for _, ch := range c.pending[key] {
			ch <- getterResult[V]{value: v, err: err}
		}

		delete(c.pending, key)
		c.mu.Unlock()
	}()

	v, err = c.getter(ctx, key)
	if err != nil {
		err = fmt.Errorf("get value by getter for key: %v: %w", key, err)
	}
}

type listValue[K comparable, V any] struct {
	key K
	val V
	exp time.Time
}

func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}

	// If the key already exists, update the value and exp, and move the element to the front of the list.
	if el, ok := c.lookup[key]; ok {
		el.Value.val = value
		el.Value.exp = exp

		c.cache.MoveToFront(el)

		return nil
	}

	// If the key does not exist, add the value to the cache and move the element to the front of the list.
	el := c.cache.PushFront(listValue[K, V]{key: key, val: value, exp: exp})
	c.lookup[key] = el

	// In favor of optimizing the speed of Set, evicting happens only when the cache is full.
	if c.cache.Len() <= c.n {
		return nil
	}

	if err := c.evictExpired(ctx); err != nil {
		return err
	}

	if c.cache.Len() <= c.n {
		return nil
	}

	return c.evict(ctx, c.cache.Back())
}

// evict removes the element from the cache.
func (c *Cache[K, V]) evict(ctx context.Context, el *list.Element[listValue[K, V]]) (err error) {
	c.cache.Remove(el)
	delete(c.lookup, el.Value.key)

	if c.onEvict == nil {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("evict value for key: %v: %v", el.Value.key, r)
		}
	}()

	err = c.onEvict(ctx, el.Value.val)
	if err != nil {
		return fmt.Errorf("evict value for key: %v: %w", el.Value.key, err)
	}

	return nil
}

// evictExpired removes expired values from the cache.
// If ttl is 0, evictExpired is a no-op.
// If ttl is > 0, expired values are removed from the cache.
// TODO: Investigate infinite loop.
func (c *Cache[K, V]) evictExpired(ctx context.Context) error {
	if c.ttl == 0 {
		return nil
	}

	now := time.Now()
	el := c.cache.Front()

	for el != nil {
		if el.Value.exp.After(now) {
			continue
		}

		if err := c.evict(ctx, el); err != nil {
			return err
		}

		el = el.Next()
	}

	return nil
}

type Option[K comparable, V any] func(*Cache[K, V])

// WithSize sets the max size of the cache.
// If the cache is full, the least recently used value is evicted.
func WithSize[K comparable, V any](n int) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.n = n
	}
}

// WithTTL sets the time to live for the cached values.
func WithTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.ttl = ttl
	}
}

type OnEvict[V any] func(ctx context.Context, v V) error

// WithOnEvict sets a function to be called after evicting a value from the cache.
func WithOnEvict[K comparable, V any](onEvict OnEvict[V]) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.onEvict = onEvict
	}
}

type Getter[K comparable, V any] func(ctx context.Context, key K) (V, error)

// WithGetter sets a function to be used to populate the cache.
// If the getter is set and no value found in the cache, the cache will populate the cache
// with the value returned by the getter.
func WithGetter[K comparable, V any](getter Getter[K, V]) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.getter = getter
	}
}

func New[K comparable, V any](options ...Option[K, V]) *Cache[K, V] {
	c := new(Cache[K, V])

	for _, f := range options {
		f(c)
	}

	if c.n <= 0 {
		c.n = defaultSize
	}

	c.cache = list.New[listValue[K, V]]()
	c.lookup = make(map[K]*list.Element[listValue[K, V]])
	c.pending = make(map[K][]chan getterResult[V])

	return c
}
