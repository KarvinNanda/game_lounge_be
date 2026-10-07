package service

import "errors"

// ErrStaffForbidden dikembalikan saat aksi pada staff akan menaikkan hak akses.
var ErrStaffForbidden = errors.New("tidak diizinkan: aksi ini hanya untuk Super Admin")

// Actor adalah staff yang sedang melakukan aksi (dari context AuthMiddleware).
type Actor struct {
	ID       string
	Username string
	IsSystem bool
}

// staffWrite merangkum fakta yang dibutuhkan untuk memutuskan create/update/delete staff.
type staffWrite struct {
	actor           Actor
	targetIsSystem  bool // staff yang diubah/dihapus adalah Super Admin
	newRoleIsSystem bool // role yang akan diberikan adalah role is_system
	isSelf          bool
	roleChanged     bool
}

// checkStaffWrite mencegah privilege escalation lewat endpoint staff:
//   - hanya Super Admin yang boleh memberi role is_system atau menyentuh akun Super Admin
//   - tidak ada yang boleh mengubah role dirinya sendiri (termasuk Super Admin,
//     supaya Super Admin terakhir tidak mengunci dirinya sendiri)
func checkStaffWrite(w staffWrite) error {
	if w.isSelf && w.roleChanged {
		return ErrStaffForbidden
	}
	if !w.actor.IsSystem && (w.targetIsSystem || w.newRoleIsSystem) {
		return ErrStaffForbidden
	}
	return nil
}
