package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLocalCache_GetSetDelete(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewLocalCache[string, int](ctx, time.Hour)

	if _, ok := c.Get("missing"); ok {
		t.Fatalf("expected miss")
	}

	c.Set("a", 1)
	c.Set("b", 2)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("got %v %v, want 1 true", v, ok)
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("got %v %v, want 2 true", v, ok)
	}

	c.Delete("a")
	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected a to be deleted")
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("b should still be present")
	}
}

func TestLocalCache_GetOrSet_CallsOnce(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewLocalCache[string, int](ctx, time.Hour)

	var calls atomic.Int32
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	results := make(chan int, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			v := c.GetOrSet("key", func() int {
				calls.Add(1)
				return 42
			})
			results <- v
		}()
	}
	wg.Wait()
	close(results)

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected fn to be called exactly once, got %d", got)
	}
	for v := range results {
		if v != 42 {
			t.Fatalf("got %d, want 42", v)
		}
	}
}

func TestLocalCache_ConcurrentSetGetDelete(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewLocalCache[int, int](ctx, time.Hour)

	const workers = 8
	const ops = 1000
	var wg sync.WaitGroup
	wg.Add(workers * 3)

	for w := 0; w < workers; w++ {
		go func(base int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				c.Set(base*ops+i, i)
			}
		}(w)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				_, _ = c.Get(base*ops + i)
			}
		}(w)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				c.Delete(base*ops + i)
			}
		}(w)
	}
	wg.Wait()
}

func TestLocalCache_CleanupClearsAll(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewLocalCache[string, int](ctx, 20*time.Millisecond)

	c.Set("a", 1)
	c.Set("b", 2)

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		_, okA := c.Get("a")
		_, okB := c.Get("b")
		if !okA && !okB {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("cache was not cleared by cleanup loop")
}

func TestLocalCache_CleanupStopsOnContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	c := NewLocalCache[string, int](ctx, time.Millisecond)
	c.Set("a", 1)
	cancel()
	// after cancel, Set/Get should still work without panic
	time.Sleep(20 * time.Millisecond)
	c.Set("b", 2)
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("cache must remain usable after ctx cancel")
	}
}
