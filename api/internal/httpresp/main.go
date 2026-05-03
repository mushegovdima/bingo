package httpresp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type errorEnvelope struct {
	Error string `json:"error"`
}

// OK пишет 200 с телом { "data": v }.
func OK[T any](w http.ResponseWriter, v T) {
	write(w, http.StatusOK, v)
}

// Created пишет 201 с телом { "data": v }.
func Created[T any](w http.ResponseWriter, v T) {
	write(w, http.StatusCreated, v)
}

// Err пишет указанный статус с телом { "error": "message" }.
func Err(w http.ResponseWriter, status int, message string) {
	write(w, status, errorEnvelope{Error: message})
}

// DecodeJSON декодирует тело запроса и возвращает (T, error).
// Ошибка содержит читаемое сообщение — можно передавать напрямую в Err.
func DecodeJSON[T any](r *http.Request) (T, error) {
	var dst T
	if err := json.NewDecoder(r.Body).Decode(&dst); err != nil {
		return dst, fmt.Errorf("%s", decodeErrMessage(err))
	}
	return dst, nil
}

// decodeErrMessage переводит ошибки json-декодера в читаемый текст.
func decodeErrMessage(err error) string {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset)
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return fmt.Sprintf("field %q must be %s, got %s", typeErr.Field, typeErr.Type, typeErr.Value)
	}

	return "invalid request body"
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// PathInt64 reads a chi URL parameter and parses it as int64. On parse error it
// writes a 400 with a descriptive message and returns ok=false; callers should
// just `return` on !ok.
func PathInt64(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := chi.URLParam(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		Err(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", name))
		return 0, false
	}
	return id, true
}
