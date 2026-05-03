package seasonservice_test

import (
	"github.com/uptrace/bun/driver/pgdriver"
	"io"
	"log/slog"
	"unsafe"
)

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// pgErrLayout mirrors the unexported layout of pgdriver.Error (single map field).
// Used to construct FK violation errors in unit tests.
type pgErrLayout struct{ m map[byte]string }

func pgFKErr(code string) error {
	src := pgErrLayout{m: map[byte]string{'C': code}}
	return *(*pgdriver.Error)(unsafe.Pointer(&src))
}
