package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	userdomain "go.mod/internal/domain/user"
)

type fakeUsers struct {
	user *userdomain.User
	err  error
}

func (f *fakeUsers) GetById(_ context.Context, _ int64) (*userdomain.User, error) {
	return f.user, f.err
}

func newStore(t *testing.T) *sessions.CookieStore {
	t.Helper()
	store := sessions.NewCookieStore([]byte("0123456789abcdef0123456789abcdef"))
	store.Options = &sessions.Options{Path: "/", MaxAge: 3600, HttpOnly: true}
	return store
}

func TestRequireAuth_NoSession_Returns401(t *testing.T) {
	t.Parallel()
	called := false
	h := RequireAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
	if called {
		t.Fatalf("next must not be called")
	}
}

func TestRequireAuth_WithSession_PassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	h := RequireAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithSession(req.Context(), &SessionCtx{SessionID: 1, UserID: 7}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !called {
		t.Fatalf("next must be called")
	}
}

func TestRequireRole_NoSession_Returns401(t *testing.T) {
	t.Parallel()
	mw := RequireRole(&fakeUsers{}, userdomain.Manager)
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("must not pass") }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestRequireRole_UserNotFound_Returns403(t *testing.T) {
	t.Parallel()
	mw := RequireRole(&fakeUsers{user: nil, err: nil}, userdomain.Manager)
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("must not pass") }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithSession(req.Context(), &SessionCtx{UserID: 1}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireRole_GetterError_Returns403(t *testing.T) {
	t.Parallel()
	mw := RequireRole(&fakeUsers{err: errors.New("boom")}, userdomain.Manager)
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("must not pass") }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithSession(req.Context(), &SessionCtx{UserID: 1}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireRole_BlockedUser_Returns403(t *testing.T) {
	t.Parallel()
	u := &userdomain.User{ID: 1, IsBlocked: true, Roles: []userdomain.UserRole{userdomain.Manager}}
	mw := RequireRole(&fakeUsers{user: u}, userdomain.Manager)
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("must not pass") }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithSession(req.Context(), &SessionCtx{UserID: 1}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireRole_RoleMismatch_Returns403(t *testing.T) {
	t.Parallel()
	u := &userdomain.User{ID: 1, Roles: []userdomain.UserRole{userdomain.Resident}}
	mw := RequireRole(&fakeUsers{user: u}, userdomain.Manager)
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("must not pass") }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithSession(req.Context(), &SessionCtx{UserID: 1}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireRole_AllowedRole_PassesThrough(t *testing.T) {
	t.Parallel()
	u := &userdomain.User{ID: 1, Roles: []userdomain.UserRole{userdomain.Manager}}
	mw := RequireRole(&fakeUsers{user: u}, userdomain.Manager, userdomain.Resident)
	called := false
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithSession(req.Context(), &SessionCtx{UserID: 1}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !called {
		t.Fatalf("expected pass-through, got status=%d called=%v", rr.Code, called)
	}
}

func TestSessionRefresh_NoCookie_NoSessionInContext(t *testing.T) {
	t.Parallel()
	store := newStore(t)
	var seen *SessionCtx
	mw := SessionRefresh(store)
	h := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = SessionFromContext(r.Context())
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if seen != nil {
		t.Fatalf("expected nil session, got %+v", seen)
	}
}

func TestSessionRefresh_ValidCookie_PopulatesContext(t *testing.T) {
	t.Parallel()
	store := newStore(t)

	// First, create a request through SaveSession so the cookie is signed correctly.
	saveReq := httptest.NewRequest(http.MethodGet, "/save", nil)
	saveRR := httptest.NewRecorder()
	if err := SaveSession(saveRR, saveReq, store, 11, 22); err != nil {
		t.Fatalf("save: %v", err)
	}
	cookie := saveRR.Result().Cookies()
	if len(cookie) == 0 {
		t.Fatalf("no cookie set")
	}

	// Now use that cookie on a second request through SessionRefresh.
	var seen *SessionCtx
	mw := SessionRefresh(store)
	h := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = SessionFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	for _, c := range cookie {
		req.AddCookie(c)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if seen == nil {
		t.Fatalf("expected session in context")
	}
	if seen.SessionID != 11 || seen.UserID != 22 {
		t.Fatalf("got %+v, want SessionID=11 UserID=22", seen)
	}
}

func TestSessionRefresh_InvalidSignature_NoSessionInContext(t *testing.T) {
	t.Parallel()
	store := newStore(t)

	var seen *SessionCtx
	mw := SessionRefresh(store)
	h := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = SessionFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "this-is-not-a-valid-signed-cookie"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if seen != nil {
		t.Fatalf("expected nil session for bad signature, got %+v", seen)
	}
}
