package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// LocalCache — потокобезопасный кэш в памяти.
// Get — lock-free через atomic.Pointer.
// Set/Delete — копируют map под mutex (copy-on-write).
// Весь кэш сбрасывается целиком каждые cleanupInterval.
// Горутина очистки завершается вместе с ctx.
type LocalCache[K comparable, V any] struct {
	mu    sync.Mutex
	items atomic.Pointer[map[K]V]
}

func NewLocalCache[K comparable, V any](ctx context.Context, cleanupInterval time.Duration) *LocalCache[K, V] {
	c := &LocalCache[K, V]{}
	m := make(map[K]V)
	c.items.Store(&m)
	go c.cleanupLoop(ctx, cleanupInterval)
	return c
}

// Get — lock-free, не блокирует других читателей и писателей.
func (c *LocalCache[K, V]) Get(key K) (V, bool) {
	v, ok := (*c.items.Load())[key]
	return v, ok
}

// Set — copy-on-write: копирует текущую map, добавляет запись, атомарно подменяет указатель.
func (c *LocalCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	next := c.copyWithSet(key, value)
	c.items.Store(&next)
	c.mu.Unlock()
}

// GetOrSet — возвращает значение если есть, иначе вызывает fn(), сохраняет и возвращает результат.
// Атомарно: fn() гарантированно вызывается не более одного раза для одного ключа.
func (c *LocalCache[K, V]) GetOrSet(key K, fn func() V) V {
	// fast path — без блокировки
	if v, ok := c.Get(key); ok {
		return v
	}
	// slow path — под mutex, повторная проверка чтобы не вычислять дважды
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := (*c.items.Load())[key]; ok {
		return v
	}
	v := fn()
	next := c.copyWithSet(key, v)
	c.items.Store(&next)
	return v
}

// Delete — copy-on-write: копирует текущую map без ключа, атомарно подменяет указатель.
func (c *LocalCache[K, V]) Delete(key K) {
	c.mu.Lock()
	next := c.copyWithDelete(key)
	c.items.Store(&next)
	c.mu.Unlock()
}

func (c *LocalCache[K, V]) copyWithSet(key K, value V) map[K]V {
	old := *c.items.Load()
	next := make(map[K]V, len(old)+1)
	for k, v := range old {
		next[k] = v
	}
	next[key] = value
	return next
}

func (c *LocalCache[K, V]) copyWithDelete(key K) map[K]V {
	old := *c.items.Load()
	next := make(map[K]V, len(old))
	for k, v := range old {
		if k != key {
			next[k] = v
		}
	}
	return next
}

func (c *LocalCache[K, V]) cleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			m := make(map[K]V)
			c.items.Store(&m)
			c.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
