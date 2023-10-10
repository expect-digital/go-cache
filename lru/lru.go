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

type getterResult[V any] struct {
	err   error
	value V
}

// Cache is a least recently used cache.
type Cache[K comparable, V any] struct {
	n          int
	ttl        time.Duration
	getter     Getter[K, V]
	afterEvict AfterEvict[V]
	cache      *list.List[listValue[K, V]]
	pending    map[K][]chan getterResult[V]
	sync.RWMutex
}

// Size returns the max size of the cache.
func (c *Cache[K, V]) Size() int {
	return c.n
}

// Len returns the length of the cache.
func (c *Cache[K, V]) Len() int {
	return c.cache.Len()
}

// Get returns the value associated with the key from the cache. If the value is not found,
// the value is populated by the getter.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) { //nolint:ireturn
	el := c.getCached(key)
	if el == nil {
		return c.populateByGetter(ctx, key)
	}

	if el.Value.exp.IsZero() || el.Value.exp.After(time.Now()) {
		return el.Value.val, nil
	}

	c.Lock()
	err := c.evict(ctx, el)
	c.Unlock()

	if err != nil {
		var v V

		return v, fmt.Errorf("evict expired value for key: %w", err)
	}

	return c.populateByGetter(ctx, key)
}

func (c *Cache[K, V]) populateByGetter(ctx context.Context, key K) (V, error) { //nolint:ireturn
	c.Lock()

	if c.getter == nil {
		var v V

		c.Unlock()

		return v, fmt.Errorf("get value by getter for key: %v: %w", key, ErrNotFound)
	}

	ch := make(chan getterResult[V], 1)
	defer close(ch)

	c.pending[key] = append(c.pending[key], ch)

	n := len(c.pending[key])

	c.Unlock()

	if n == 1 {
		go c.execGetter(ctx, key)
	}

	msg := <-ch

	return msg.value, msg.err
}

func (c *Cache[K, V]) execGetter(ctx context.Context, key K) {
	var (
		v   V
		err error
	)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("exec getter for key: %v: %v", key, r) //nolint:goerr113
		}

		c.Lock()

		for _, ch := range c.pending[key] {
			ch <- getterResult[V]{value: v, err: err}
		}

		delete(c.pending, key)
		c.Unlock()
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

// getCached returns the value associated with the key from the cache.
func (c *Cache[K, V]) getCached(key K) *list.Element[listValue[K, V]] {
	c.RLock()
	defer c.RUnlock()

	el := c.cache.Front()
	if el == nil {
		return nil
	}

	for el != nil {
		v := el.Value
		if v.key == key {
			return el
		}

		el = el.Next()
	}

	return nil
}

func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) error {
	c.Lock()
	defer c.Unlock()

	var exp time.Time

	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}

	c.cache.PushFront(listValue[K, V]{key: key, val: value, exp: exp})

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

	if c.afterEvict == nil {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("evict value for key: %v: %v", el.Value.key, r) //nolint:goerr113
		}
	}()

	err = c.afterEvict(ctx, el.Value.val)
	if err != nil {
		return fmt.Errorf("evict value for key: %v: %w", el.Value.key, err)
	}

	return nil
}

// evictExpired removes expired values from the cache.
// If ttl is 0, evictExpired is a no-op.
// If ttl is > 0, expired values are removed from the cache.
func (c *Cache[K, V]) evictExpired(ctx context.Context) error {
	if c.ttl == 0 {
		return nil
	}

	now := time.Now()
	el := c.cache.Front()

	for el != nil {
		if el.Value.exp.Before(now) {
			c.cache.Remove(el)

			if err := c.evict(ctx, el); err != nil {
				return err
			}
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

type AfterEvict[V any] func(ctx context.Context, v V) error

// WithAfterEvict sets a function to be called after evicting a value from the cache.
func WithAfterEvict[K comparable, V any](afterEvict AfterEvict[V]) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.afterEvict = afterEvict
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
	c.pending = make(map[K][]chan getterResult[V])

	return c
}
