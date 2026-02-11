package lru

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.expect.digital/cache/internal/linked"
)

const defaultSize = 1024

// ErrNotFound is an error that is returned when a requested key is not found in the cache, and no getter is set.
var ErrNotFound = errors.New("not found")

// zeroValue returns the zero value of the type.
func zeroValue[T any]() (zero T) { //nolint:ireturn
	return
}

type getterResult[V any] struct {
	err   error
	value V
}

// Cache is a generic, concurrent-safe, least recently used (LRU) cache structure.
// The zero value for Cache is an ready to use cache with size of 1024.
type Cache[K comparable, V any] struct {
	n       int
	ttl     time.Duration
	getter  Getter[K, V]
	onEvict OnEvict[V]
	cache   *linked.List[listValue[K, V]]
	lookup  map[K]*linked.Element[listValue[K, V]]
	pending map[K][]chan getterResult[V]
	mu      sync.RWMutex
}

// Size returns the maximum size of the cache. The size is the maximum number of entries
// the cache can hold before it starts evicting the least recently used entries.
func (c *Cache[K, V]) Size() int {
	return c.n
}

// Len returns the current number of entries in the cache. This number will always be less than
// or equal to the size returned by Size().
func (c *Cache[K, V]) Len() int {
	return c.cache.Len()
}

/*
Get retrieves the value associated with the provided key from the cache and moves it to the front of the cache.
If the value is not found or has expired, the method attempts to populate it using the getter function, if provided.

	cache.Get(ctx, 1) // cache[int, ...]
	cache.Get(ctx, "key") // cache[string, ...]
*/
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) { //nolint:ireturn
	// TODO: too many locks?
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

// populateByGetter populates the cache with the value returned by the getter.
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
	err := c.Set(ctx, key, msg.value)
	if err != nil {
		return zeroValue[V](), fmt.Errorf("set value for key: %v: %w", key, err)
	}

	return msg.value, nil
}

// execGetter executes the getter function and sends the result to all pending channels.
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

/*
Set adds a value to the cache associated with the provided key and moves it to the front of the cache.

If the key already exists in the cache, the method updates the value and its
expiration time, and moves it to the front of the cache.
If the key does not exist, the method adds the value to the cache
and sets its expiration time based on the cache's TTL.

If adding the value causes the cache to exceed its maximum size, the method evicts
expired entries and if still exceeding the maximum size, the least recently used entry.

	cache.Set(ctx, 1, "value") // cache[int, string]
	cache.Set(ctx, "key", "value") // cache[string, string]

	type user struct {
		Id   int
		Name string
	}
	cache.Set(ctx, 1, user{Id: 1, Name: "John Doe"})  // cache[int, user]
*/
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

	err := c.evictExpired(ctx)
	if err != nil {
		return err
	}

	if c.cache.Len() <= c.n {
		return nil
	}

	return c.evict(ctx, c.cache.Back())
}

// evict removes the element from the cache.
func (c *Cache[K, V]) evict(ctx context.Context, el *linked.Element[listValue[K, V]]) (err error) {
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
func (c *Cache[K, V]) evictExpired(ctx context.Context) error {
	if c.ttl == 0 {
		return nil
	}

	now := time.Now()

	for _, v := range c.lookup {
		if v.Value.exp.After(now) {
			continue
		}

		err := c.evict(ctx, v)
		if err != nil {
			return err
		}
	}

	return nil
}

type Option[K comparable, V any] func(*Cache[K, V])

// WithSize is an Option that sets the max size of the cache.
// If the cache is full, the least recently used value is evicted.
func WithSize[K comparable, V any](n int) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.n = n
	}
}

// WithTTL is an Option that sets the time to live for the cached values.
// If a value is expired, it is evicted from the cache.
func WithTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.ttl = ttl
	}
}

/*
OnEvict is a function type that defines a function to be called after evicting a value from the cache.

	onEvicter := func(ctx context.Context, v user) error {
		// Do something with the evicted value. Keep track of the evicted values, log them, etc.
		return nil
	}
*/
type OnEvict[V any] func(ctx context.Context, v V) error

// WithOnEvict is an Option that sets a function to be called after evicting a value from the cache.
func WithOnEvict[K comparable, V any](onEvict OnEvict[V]) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.onEvict = onEvict
	}
}

/*
Getter is a function type that defines a function to be used to populate the cache.
If the getter is set and no value found in the cache, the cache will populate the cache
with the value returned by the getter.

	type user struct {
		ID   int
		Name string
	}

	// cache[int,user]
	getter := func(ctx context.Context, id int) (user, error) {
		// Query the database or retrieve the value from any another source.
		return user{ID: id, Name: "John Doe"}, nil
	}
*/
type Getter[K comparable, V any] func(ctx context.Context, key K) (V, error)

// WithGetter is an Option that sets a function to be used to populate the cache.
func WithGetter[K comparable, V any](getter Getter[K, V]) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.getter = getter
	}
}

// New creates a new Cache with the provided options.
func New[K comparable, V any](options ...Option[K, V]) *Cache[K, V] {
	c := new(Cache[K, V])

	for _, f := range options {
		f(c)
	}

	if c.n <= 0 {
		c.n = defaultSize
	}

	c.cache = linked.New[listValue[K, V]]()
	c.lookup = make(map[K]*linked.Element[listValue[K, V]])
	c.pending = make(map[K][]chan getterResult[V])

	return c
}
