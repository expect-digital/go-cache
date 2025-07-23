package lru

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func Test_WithSize(t *testing.T) {
	t.Parallel()

	c := New(WithSize[int, int](10))

	assert.Equal(t, 10, c.Size())
	assert.Zero(t, c.Len())
}

func Test_WithTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ttl := 100 * time.Millisecond
	c := New(WithTTL[int, string](ttl))

	err := c.Set(ctx, 1, "one")
	assert.NoError(t, err)

	v, err := c.Get(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, "one", v)

	time.Sleep(ttl)

	v, err = c.Get(ctx, 1)

	require.ErrorIs(t, err, ErrNotFound)
	assert.Empty(t, v)
}

func Test_WithGetter(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(key int, value string) bool {
		ctx := context.Background()

		c := New(WithGetter(func(ctx context.Context, k int) (string, error) {
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

	assert.NoError(t, err)
}

func Test_WithGetterParallel(t *testing.T) {
	t.Parallel()

	key, value := 1, "OK"

	var count int32

	c := New(
		WithGetter(func(ctx context.Context, k int) (string, error) {
			atomic.AddInt32(&count, 1)
			time.Sleep(50 * time.Millisecond) // arbitrary sleep to simulate network latency
			if k == key {
				return value, nil
			}

			return "", ErrNotFound
		}),
	)

	var eg errgroup.Group

	for i := 0; i < 1000; i++ {
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

	assert.NoError(t, eg.Wait())
	assert.EqualValues(t, 1, count)
}

func Test_GetterPanics(t *testing.T) {
	t.Parallel()

	c := New(
		WithGetter(func(ctx context.Context, k int) (string, error) {
			time.Sleep(50 * time.Millisecond) // arbitrary sleep to simulate network latency
			panic("panic")
		}),
	)

	var wg sync.WaitGroup

	n := 10

	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			v, err := c.Get(context.Background(), 1)

			require.ErrorContains(t, err, "panic")
			assert.Empty(t, v)
			wg.Done()
		}()
	}

	wg.Wait()
}

func Test_GetterReturnsError(t *testing.T) {
	t.Parallel()

	c := New(
		WithGetter(func(ctx context.Context, k int) (string, error) {
			time.Sleep(50 * time.Millisecond) // arbitrary sleep to simulate network latency

			return "", errors.New("fail test")
		}),
	)

	_, err := c.Get(context.Background(), 1)

	require.ErrorContains(t, err, "fail test")
}

func Test_OnEvictPanics(t *testing.T) {
	t.Parallel()

	c := New(
		WithTTL[int, string](time.Nanosecond),
		WithOnEvict[int, string](func(ctx context.Context, v string) error {
			panic("panic")
		}),
	)

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	assert.NoError(t, err)

	time.Sleep(time.Millisecond)

	v, err := c.Get(ctx, 1)

	assert.EqualError(t, err, "evict expired value: evict value for key: 1: panic")
	assert.Empty(t, v)
}

func Test_OnEvictReturnsError(t *testing.T) {
	t.Parallel()

	c := New(
		WithTTL[int, string](time.Nanosecond),
		WithOnEvict[int, string](func(ctx context.Context, v string) error {
			return errors.New("oops")
		}),
	)

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	assert.NoError(t, err)

	time.Sleep(time.Millisecond)

	v, err := c.Get(ctx, 1)

	assert.EqualError(t, err, "evict expired value: evict value for key: 1: oops")
	assert.Empty(t, v)
}

func Test_OnEvictOK(t *testing.T) {
	t.Parallel()

	c := New(
		WithTTL[int, string](time.Nanosecond),
		WithOnEvict[int, string](func(ctx context.Context, v string) error {
			return nil
		}),
	)

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	assert.NoError(t, err)

	time.Sleep(time.Millisecond)

	v, err := c.Get(ctx, 1)

	assert.ErrorIs(t, err, ErrNotFound)
	assert.Empty(t, v)
}

func Test_GetOneKeyMultipleTimes(t *testing.T) {
	t.Parallel()

	var getterExecCount int

	getter := func(ctx context.Context, k int) (string, error) {
		getterExecCount++

		return "", nil
	}

	c := New(WithGetter(getter))

	ctx := context.Background()

	_, err := c.Get(ctx, 1)
	require.NoError(t, err)

	_, err = c.Get(ctx, 1)
	require.NoError(t, err)

	// Getter should be called only once, second time the value is from the cache.
	assert.Equal(t, 1, getterExecCount)
}

func Test_UpdateKey(t *testing.T) {
	t.Parallel()

	c := New[int, string]()

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	require.NoError(t, err)

	err = c.Set(ctx, 1, "two")
	require.NoError(t, err)

	v, err := c.Get(ctx, 1)
	require.NoError(t, err)

	// The value should be updated.
	require.Equal(t, "two", v)

	// The length should be 1, as the key is updated, not added.
	require.Equal(t, 1, c.Len())
	require.Equal(t, 1, len(c.lookup))
}

func Test_EvictLeastRecent(t *testing.T) {
	t.Parallel()

	c := New[int, string](WithSize[int, string](2))

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	require.NoError(t, err)

	err = c.Set(ctx, 2, "two")
	require.NoError(t, err)

	_, err = c.Get(ctx, 1)
	require.NoError(t, err)

	err = c.Set(ctx, 3, "three")
	require.NoError(t, err)

	v, err := c.Get(ctx, 2)
	require.ErrorIs(t, err, ErrNotFound)
	require.Empty(t, v)
}

func Test_CacheWithLargeCacheSize(t *testing.T) {
	t.Parallel()

	cacheSize := 10000
	dataSize := 20000

	c := New(WithSize[int, int](cacheSize))

	for i := 0; i < dataSize; i++ {
		require.NoError(t, c.Set(context.Background(), i, i))
	}

	for i := 0; i < dataSize; i++ {
		v, err := c.Get(context.Background(), i)

		// The cache should contain only the last cSize items.
		if i < dataSize-cacheSize {
			require.ErrorIs(t, err, ErrNotFound)
			require.Empty(t, v)
		} else {
			require.NoError(t, err)
			require.Equal(t, i, v)
		}
	}
}

func Test_EvictExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ttl := 5 * time.Millisecond

	c := New(
		WithTTL[int, int](ttl),
		WithSize[int, int](2),
	)

	// Set a value with a TTL
	require.NoError(t, c.Set(ctx, 1, 1))
	require.NoError(t, c.Set(ctx, 2, 2))

	time.Sleep(ttl * 2)

	require.NoError(t, c.Set(ctx, 3, 3))

	require.Equal(t, 1, c.Len())
	require.Equal(t, 2, c.Size())

	v, err := c.Get(ctx, 3)
	require.NoError(t, err)

	require.Equal(t, 3, v)
}

//nolint:funlen
func Test_ConcurrentGetAndSet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataCount := 2000

	testConcurrent := func(c *Cache[int, int], expectedLen int) {
		var wg sync.WaitGroup

		wg.Add(2)

		go func() {
			defer wg.Done()

			for i := 0; i < dataCount; i++ {
				require.NoError(t, c.Set(ctx, i, i))
			}
		}()

		go func() {
			defer wg.Done()

			for i := 0; i < dataCount; i++ {
				val, err := c.Get(ctx, i)

				// As getting and setting are executed concurrently, the value
				// may or may not be in the cache.
				switch err {
				case nil:
					require.Equal(t, i, val)
				default:
					if c.getter == nil {
						require.ErrorIs(t, err, ErrNotFound)
					} else {
						require.FailNow(t, "getter is not nil and could not get the value: %w", err)
					}
				}
			}
		}()

		wg.Wait()

		require.Equal(t, expectedLen, c.Len())
		require.Equal(t, expectedLen, c.Size())
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
				WithGetter[int, int](func(ctx context.Context, key int) (int, error) { return key, nil }),
			),
			expectedLen: dataCount,
		},
		{
			name: "Not enough cache size with getter",
			cache: New(
				WithSize[int, int](dataCount/5),
				WithGetter[int, int](func(ctx context.Context, key int) (int, error) { return key, nil }),
			),
			expectedLen: dataCount / 5,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testConcurrent(tt.cache, tt.expectedLen)
		})
	}
}
