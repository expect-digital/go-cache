package lru

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestWithSize(t *testing.T) {
	t.Parallel()

	c := New(WithSize[int, int](10))

	if c.Size() != 10 {
		t.Errorf("want size 10, got %d", c.Size())
	}

	if c.Len() != 0 {
		t.Errorf("want empty, got %d", c.Len())
	}
}

func TestWithTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ttl := 100 * time.Millisecond
	c := New(WithTTL[int, string](ttl))

	err := c.Set(ctx, 1, "one")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	v, err := c.Get(ctx, 1)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if v != "one" {
		t.Errorf("want value 'one', got %q", v)
	}

	time.Sleep(ttl)

	v, err = c.Get(ctx, 1)
	if err == nil {
		t.Errorf("want % error, got nil", ErrNotFound)
	}

	if v != "" {
		t.Errorf("want empty value, got %q", v)
	}
}

func TestWithGetter(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(key int, value string) bool {
		ctx := context.Background()

		c := New(WithGetter(func(_ context.Context, k int) (string, error) {
			if k == key {
				return value, nil
			}

			return "", ErrNotFound
		}))

		v, err := c.Get(ctx, key+1)
		if v != "" || !errors.Is(err, ErrNotFound) {
			return false
		}

		v, err = c.Get(ctx, key)

		return value == v && err == nil
	}, nil)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
}

func TestWithGetterParallel(t *testing.T) {
	t.Parallel()

	key, value := 1, "OK"

	var count int32

	c := New(
		WithGetter(func(_ context.Context, k int) (string, error) {
			atomic.AddInt32(&count, 1)
			time.Sleep(50 * time.Millisecond) // arbitrary sleep to simulate network latency

			if k == key {
				return value, nil
			}

			return "", ErrNotFound
		}),
	)

	var eg errgroup.Group

	for range 1000 {
		eg.Go(func() error {
			v, err := c.Get(context.Background(), 1)
			if err != nil {
				return err
			}

			if v != value {
				return fmt.Errorf("value is %s, expected %s", v, value)
			}

			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if count != 1 {
		t.Errorf("want count 1, got %d", count)
	}
}

func TestGetterPanics(t *testing.T) {
	t.Parallel()

	c := New(
		WithGetter(func(_ context.Context, _ int) (string, error) {
			time.Sleep(50 * time.Millisecond) // arbitrary sleep to simulate network latency
			panic("panic")
		}),
	)

	var wg sync.WaitGroup

	n := 10

	wg.Add(n)

	for range n {
		go func() {
			v, err := c.Get(context.Background(), 1)
			if err == nil || !strings.Contains(err.Error(), "panic") {
				t.Errorf("want panic error, got %v", err)
			}

			if v != "" {
				t.Errorf("want empty value, got %s", v)
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func TestGetterReturnsError(t *testing.T) {
	t.Parallel()

	c := New(
		WithGetter(func(_ context.Context, _ int) (string, error) {
			time.Sleep(50 * time.Millisecond) // arbitrary sleep to simulate network latency

			return "", errors.New("fail test")
		}),
	)

	_, err := c.Get(context.Background(), 1)

	if !strings.Contains(err.Error(), "fail test") {
		t.Errorf("want fail test error, got %v", err)
	}
}

func TestOnEvictPanics(t *testing.T) {
	t.Parallel()

	c := New(
		WithTTL[int, string](time.Nanosecond),
		WithOnEvict[int](func(_ context.Context, _ string) error {
			panic("panic")
		}),
	)

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	time.Sleep(time.Millisecond)

	v, err := c.Get(ctx, 1)
	if !strings.Contains(err.Error(), "evict expired value: evict value for key: 1: panic") {
		t.Errorf("want evict expired value: evict value for key: 1: panic, got %v", err)
	}

	if v != "" {
		t.Errorf("want empty value, got %s", v)
	}
}

func TestOnEvictReturnsError(t *testing.T) {
	t.Parallel()

	c := New(
		WithTTL[int, string](time.Nanosecond),
		WithOnEvict[int](func(_ context.Context, _ string) error {
			return errors.New("oops")
		}),
	)

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	time.Sleep(time.Millisecond)

	v, err := c.Get(ctx, 1)
	if !strings.Contains(err.Error(), "evict expired value: evict value for key: 1: oops") {
		t.Errorf("want evict expired value: evict value for key: 1: oops, got %v", err)
	}

	if v != "" {
		t.Errorf("want empty, got %s", v)
	}
}

func TestOnEvictOK(t *testing.T) {
	t.Parallel()

	c := New(
		WithTTL[int, string](time.Nanosecond),
		WithOnEvict[int](func(_ context.Context, _ string) error {
			return nil
		}),
	)

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	time.Sleep(time.Millisecond)

	v, err := c.Get(ctx, 1)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}

	if v != "" {
		t.Errorf("want empty, got %s", v)
	}
}

func TestGetOneKeyMultipleTimes(t *testing.T) {
	t.Parallel()

	var getterExecCount int

	getter := func(_ context.Context, _ int) (string, error) {
		getterExecCount++

		return "", nil
	}

	c := New(WithGetter(getter))

	ctx := context.Background()

	_, err := c.Get(ctx, 1)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	_, err = c.Get(ctx, 1)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	// Getter should be called only once, second time the value is from the cache.
	if getterExecCount != 1 {
		t.Errorf("want getter called once, got %d times", getterExecCount)
	}
}

func TestUpdateKey(t *testing.T) {
	t.Parallel()

	c := New[int, string]()

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	err = c.Set(ctx, 1, "two")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	v, err := c.Get(ctx, 1)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	// The value should be updated.
	if v != "two" {
		t.Errorf("want value 'two', got %q", v)
	}

	// The length should be 1, as the key is updated, not added.
	if c.Len() != 1 {
		t.Errorf("want length 1, got %d", c.Len())
	}

	if len(c.lookup) != 1 {
		t.Errorf("want lookup length 1, got %d", len(c.lookup))
	}
}

func TestEvictLeastRecent(t *testing.T) {
	t.Parallel()

	c := New(WithSize[int, string](2))

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	err = c.Set(ctx, 2, "two")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	_, err = c.Get(ctx, 1)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	err = c.Set(ctx, 3, "three")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	v, err := c.Get(ctx, 2)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	if v != "" {
		t.Errorf("want empty, got %s", v)
	}
}

func TestCacheWithLargeCacheSize(t *testing.T) {
	t.Parallel()

	cacheSize := 10000
	dataSize := 20000

	c := New(WithSize[int, int](cacheSize))

	for i := range dataSize {
		err := c.Set(context.Background(), i, i)
		if err != nil {
			t.Fatalf("want no error, got %v", err)
		}
	}

	for i := range dataSize {
		v, err := c.Get(context.Background(), i)

		// The cache should contain only the last cSize items.
		if i < dataSize-cacheSize {
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("want ErrNotFound, got %v", err)
			}

			if v != 0 {
				t.Errorf("want zero, got %d", v)
			}
		} else {
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}

			if v != i {
				t.Errorf("want %d, got %d", i, v)
			}
		}
	}
}

func TestEvictExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ttl := 5 * time.Millisecond

	c := New(
		WithTTL[int, int](ttl),
		WithSize[int, int](2),
	)

	// Set a value with a TTL
	err := c.Set(ctx, 1, 1)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	err = c.Set(ctx, 2, 2)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	time.Sleep(ttl * 2)

	err = c.Set(ctx, 3, 3)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if c.Len() != 1 {
		t.Errorf("want length 1, got %d", c.Len())
	}

	if c.Size() != 2 {
		t.Errorf("want size 2, got %d", c.Size())
	}

	v, err := c.Get(ctx, 3)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if v != 3 {
		t.Errorf("want 3, got %d", v)
	}
}

func TestConcurrentGetAndSet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataCount := 2000

	testConcurrent := func(c *Cache[int, int], expectedLen int) {
		var wg sync.WaitGroup

		wg.Add(2)

		go func() {
			defer wg.Done()

			for i := range dataCount {
				err := c.Set(ctx, i, i)
				if err != nil {
					t.Errorf("want no error, got %v", err)
				}
			}
		}()

		go func() {
			defer wg.Done()

			for i := range dataCount {
				val, err := c.Get(ctx, i)

				// As getting and setting are executed concurrently, the value
				// may or may not be in the cache.
				switch err {
				case nil:
					if i != val {
						t.Errorf("want %d, got %d", i, val)
					}
				default:
					if c.getter == nil {
						if !errors.Is(err, ErrNotFound) {
							t.Errorf("want ErrNotFound, got %v", err)
						}
					} else {
						if !strings.Contains(err.Error(), "getter is not nil and could not get the value") {
							t.Errorf("want getter is not nil and could not get the value, got %v", err)
						}
					}
				}
			}
		}()

		wg.Wait()

		if c.Len() != expectedLen {
			t.Errorf("want length %d, got %d", expectedLen, c.Len())
		}

		if c.Size() != expectedLen {
			t.Errorf("want size %d, got %d", expectedLen, c.Size())
		}
	}

	tests := []struct {
		name        string
		cache       *Cache[int, int]
		expectedLen int
	}{
		{
			name:        "Enough cache size without getter",
			cache:       New(WithSize[int, int](dataCount)),
			expectedLen: dataCount,
		},
		{
			name:        "Not enough cache size without getter",
			cache:       New(WithSize[int, int](dataCount / 5)),
			expectedLen: dataCount / 5,
		},
		{
			name: "Enough cache size with getter",
			cache: New(
				WithSize[int, int](dataCount),
				WithGetter(func(_ context.Context, key int) (int, error) { return key, nil }),
			),
			expectedLen: dataCount,
		},
		{
			name: "Not enough cache size with getter",
			cache: New(
				WithSize[int, int](dataCount/5),
				WithGetter(func(_ context.Context, key int) (int, error) { return key, nil }),
			),
			expectedLen: dataCount / 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testConcurrent(tt.cache, tt.expectedLen)
		})
	}
}
