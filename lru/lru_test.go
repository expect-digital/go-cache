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

func Test_Cache_WithSize(t *testing.T) {
	t.Parallel()

	c := New(WithSize[int, int](10))

	assert.Equal(t, 10, c.Size())
	assert.Zero(t, c.Len())
}

func Test_Cache_WithTTL(t *testing.T) {
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

	require.ErrorContains(t, err, "getter is nil")
	assert.Empty(t, v)
}

func Test_Cache_WithGetter(t *testing.T) {
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

func Test_Cache_WithGetter_Parallel(t *testing.T) {
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

	for i := 0; i < 10_000; i++ {
		eg.Go(func() error {
			v, err := c.Get(context.Background(), 1)
			if err != nil {
				return err
			}

			if v != value {
				return fmt.Errorf("value is %s, expected %s", v, value) //nolint:goerr113
			}

			return nil
		})
	}

	assert.NoError(t, eg.Wait())
	assert.EqualValues(t, 1, count)
}

func Test_Cache_Getter_Panics(t *testing.T) {
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

func Test_Cache_WithAfterEvict(t *testing.T) {
	t.Parallel()

	c := New(
		WithTTL[int, string](time.Nanosecond),
		WithAfterEvict[int, string](func(ctx context.Context, v string) error {
			panic("panic")
		}),
	)

	time.Sleep(time.Millisecond)

	ctx := context.Background()

	err := c.Set(ctx, 1, "one")
	assert.NoError(t, err)

	v, err := c.Get(ctx, 1)

	assert.EqualError(t, err, "evict expired value for key: evict value for key: 1: panic")
	assert.Empty(t, v)
}

func Test_Cache_Get_One_Key_Multiple_Times(t *testing.T) {
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
