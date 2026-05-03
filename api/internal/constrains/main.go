package constrains

import (
	"github.com/uptrace/bun"
)

type Database interface {
	DB() *bun.DB
}

type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
	GetOrSet(key K, fn func() V) V
	Delete(key K)
}
