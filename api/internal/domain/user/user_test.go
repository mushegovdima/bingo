package user_test

import (
	"errors"
	"testing"

	userdomain "go.mod/internal/domain/user"
)

func TestUser_Roles(t *testing.T) {
	t.Run("HasRole and IsManager", func(t *testing.T) {
		u := &userdomain.User{Roles: []userdomain.UserRole{userdomain.Manager, userdomain.Resident}}
		if !u.HasRole(userdomain.Manager) || !u.IsManager() {
			t.Errorf("expected user to be manager")
		}
		if !u.HasRole(userdomain.Resident) {
			t.Errorf("expected user to be resident")
		}

		u2 := &userdomain.User{Roles: []userdomain.UserRole{userdomain.Resident}}
		if u2.IsManager() {
			t.Errorf("non-manager reported as manager")
		}
	})

	t.Run("AddRole is idempotent and rejects unknown roles", func(t *testing.T) {
		u := &userdomain.User{}
		if err := u.AddRole(userdomain.Manager); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := u.AddRole(userdomain.Manager); err != nil {
			t.Fatalf("re-adding existing role must not error: %v", err)
		}
		if len(u.Roles) != 1 {
			t.Errorf("Roles: got %v, want 1 element", u.Roles)
		}
		if err := u.AddRole(""); !errors.Is(err, userdomain.ErrInvalidRole) {
			t.Errorf("got err=%v, want ErrInvalidRole", err)
		}
	})

	t.Run("RemoveRole is idempotent", func(t *testing.T) {
		u := &userdomain.User{Roles: []userdomain.UserRole{userdomain.Manager, userdomain.Resident}}
		u.RemoveRole(userdomain.Manager)
		if u.HasRole(userdomain.Manager) {
			t.Errorf("Manager not removed")
		}
		u.RemoveRole(userdomain.Manager) // again
		if len(u.Roles) != 1 || u.Roles[0] != userdomain.Resident {
			t.Errorf("unexpected roles: %v", u.Roles)
		}
	})
}

func TestUser_Block(t *testing.T) {
	u := &userdomain.User{}
	u.Block()
	if !u.IsBlocked {
		t.Errorf("Block did not set IsBlocked")
	}
	u.Unblock()
	if u.IsBlocked {
		t.Errorf("Unblock did not clear IsBlocked")
	}
}
