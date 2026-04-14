package domain

import "github.com/uptrace/bun"

type Database interface {
	DB() *bun.DB
}
