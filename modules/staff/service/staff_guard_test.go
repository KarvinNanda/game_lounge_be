package service

import (
	"errors"
	"testing"
)

func TestCheckStaffWrite(t *testing.T) {
	admin := Actor{ID: "a-1", IsSystem: true}
	manager := Actor{ID: "m-1", IsSystem: false}

	cases := []struct {
		name string
		in   staffWrite
		want error
	}{
		{"super admin boleh assign role is_system", staffWrite{actor: admin, newRoleIsSystem: true}, nil},
		{"super admin boleh edit super admin lain", staffWrite{actor: admin, targetIsSystem: true}, nil},
		{"non-super admin buat staff biasa", staffWrite{actor: manager}, nil},
		{"non-super admin assign role is_system", staffWrite{actor: manager, newRoleIsSystem: true}, ErrStaffForbidden},
		{"non-super admin edit/hapus super admin", staffWrite{actor: manager, targetIsSystem: true}, ErrStaffForbidden},
		{"ubah role diri sendiri", staffWrite{actor: manager, isSelf: true, roleChanged: true}, ErrStaffForbidden},
		{"super admin ubah role diri sendiri", staffWrite{actor: admin, isSelf: true, roleChanged: true}, ErrStaffForbidden},
		{"edit profil sendiri tanpa ubah role", staffWrite{actor: manager, isSelf: true}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkStaffWrite(tc.in); !errors.Is(got, tc.want) {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
