package service

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"game_lounge_be/modules/admin_recovery/repository"
	"game_lounge_be/utils"
)

// RequestReset memproses permintaan reset password.
// SELALU return response yang sama agar penyerang tidak tahu apakah email valid.
// Logic internal: hanya proses jika email adalah super admin.
func RequestReset(email, ip string) {
	// 1. Rate limiting: max 3 request per jam per IP
	count, _ := repository.CountRequestsFromIP(ip)
	if count >= 3 {
		// Diam-diam reject, response tetap sama di controller
		return
	}

	// 2. Cari super admin dengan email ini (diam-diam, tidak return error)
	staff, err := repository.FindSuperAdminByEmail(email)
	if err != nil {
		// Email tidak ditemukan atau bukan super admin → diam-diam skip
		return
	}

	// 3. Generate token acak 32-byte
	token, err := repository.GenerateToken()
	if err != nil {
		log.Printf("Gagal generate token: %v", err)
		return
	}

	// 4. Simpan token ke DB
	if err := repository.CreateToken(staff.ID, token, ip); err != nil {
		log.Printf("Gagal simpan token: %v", err)
		return
	}

	// 5. Kirim email dengan link reset (async, tidak blocking)
	utils.SafeGo(func() { sendResetEmail(staff.Email, staff.Username, token) })
}

// ValidateToken memvalidasi token sebelum menampilkan form reset password.
// Return error jika token tidak valid, expired, atau sudah dipakai.
func ValidateToken(token string) error {
	_, err := repository.FindValidToken(token)
	if err != nil {
		return errors.New("link reset password tidak valid atau sudah kadaluwarsa")
	}
	return nil
}

// ResetPassword memperbarui password dengan token yang valid.
func ResetPassword(token, newPassword string) error {
	// 1. Validasi token
	resetToken, err := repository.FindValidToken(token)
	if err != nil {
		return errors.New("link reset password tidak valid atau sudah kadaluwarsa")
	}

	// 2. Forgot-password tokens are for super admins only. A link that a super
	// admin created for a staff (StaffResetMarker) works for any staff.
	if !resetToken.Staff.Role.IsSystem && resetToken.IPAddress != repository.StaffResetMarker {
		return errors.New("akses ditolak")
	}

	// 3. Hash password baru
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.New("gagal memproses password")
	}

	// 4. Tandai token terpakai + update password secara atomik (one-time use)
	if err := repository.ConsumeTokenAndSetPassword(token, resetToken.StaffID, hashedPassword); err != nil {
		if errors.Is(err, repository.ErrTokenUsed) {
			return errors.New("link reset password tidak valid atau sudah kadaluwarsa")
		}
		return errors.New("gagal menyimpan password baru")
	}

	return nil
}

// sendResetEmail mengirim email berisi link reset password ke super admin.
func sendResetEmail(email, name, token string) {
	// ADMIN_APP_URL, not APP_URL: APP_URL is the customer app, which has no
	// /admin-recovery page.
	resetLink := fmt.Sprintf("%s/admin-recovery/%s", os.Getenv("ADMIN_APP_URL"), token)

	subject := "Reset Password — Quantum Playstation Admin"
	body := fmt.Sprintf(`Halo %s,

Kami menerima permintaan reset password untuk akun super admin kamu.

Klik link berikut untuk membuat password baru:
%s

Link ini hanya berlaku selama 15 menit dan hanya bisa digunakan 1 kali.

Jika kamu tidak meminta reset password, abaikan email ini.
Password kamu tidak akan berubah.

Salam,
Tim Quantum Playstation Rental`, name, resetLink)

	if err := sendEmail(email, name, subject, body); err != nil {
		log.Printf("Gagal kirim email reset password ke %s: %v", email, err)
	}
}

// sendEmail is a variable so tests can replace it: the real one sends email.
var sendEmail = utils.SendEmail

// staffResetTTL: how long the link from a super admin reset works.
const staffResetTTL = 24 * time.Hour

// staffInviteTTL: how long the link for a new staff works. Longer than a reset,
// like the customer welcome link, because the person did not ask for it.
const staffInviteTTL = 72 * time.Hour

// StaffLinksConfigured reports whether staff links can be built. Callers check it
// before changing anything, so a missing ADMIN_APP_URL never leaves a staff
// without a way to log in.
func StaffLinksConfigured() error {
	if os.Getenv("ADMIN_APP_URL") == "" {
		return errors.New("ADMIN_APP_URL belum diset, link atur password tidak bisa dibuat")
	}
	return nil
}

// IssueStaffPasswordReset is the super admin "reset password" for a staff.
// The old password stops working and all sessions end right away (a reset by
// an admin usually means the credential must not be used again). The staff gets
// a one-time link to {ADMIN_APP_URL}/admin-recovery/{token}, valid 24 hours.
// The email is sent synchronously: if it fails, the super admin must know,
// because the staff cannot log in until a new link arrives.
func IssueStaffPasswordReset(staffID string) error {
	return issueStaffLink(staffID, staffResetTTL,
		"Password Akun Quantum Anda Telah Direset",
		`Halo %s,

Password akun Quantum Playstation Rental kamu telah direset oleh Super Admin.
Password lama sudah tidak berlaku dan semua sesi login sudah diakhiri.

Klik link berikut untuk membuat password baru:
%s

Link ini berlaku 24 jam dan hanya bisa digunakan 1 kali.
Jika link sudah tidak berlaku, minta Super Admin untuk reset ulang.

Salam,
Tim Quantum Gaming Center`,
		"password lama sudah dinonaktifkan, tapi email gagal terkirim (%v); coba reset ulang")
}

// IssueStaffInvite sends a new staff a one-time link (72 hours) to set their
// first password. Same mechanism as IssueStaffPasswordReset.
func IssueStaffInvite(staffID string) error {
	return issueStaffLink(staffID, staffInviteTTL,
		"Akun Quantum Anda Sudah Dibuat",
		`Halo %s,

Akun staff Quantum Playstation Rental kamu sudah dibuat oleh Super Admin.

Klik link berikut untuk membuat password:
%s

Link ini berlaku 72 jam dan hanya bisa digunakan 1 kali.
Jika link sudah tidak berlaku, minta Super Admin untuk reset password.

Salam,
Tim Quantum Gaming Center`,
		"email undangan gagal terkirim (%v); gunakan Reset Password untuk mengirim link baru")
}

// issueStaffLink revokes older links, makes the current password unusable, ends
// sessions, stores a new link token and emails the link. bodyFmt takes the
// username and the link; sendErrFmt takes the send error.
func issueStaffLink(staffID string, ttl time.Duration, subject, bodyFmt, sendErrFmt string) error {
	if err := StaffLinksConfigured(); err != nil {
		return err
	}
	staff, err := repository.FindActiveStaff(staffID)
	if err != nil {
		return errors.New("staff tidak ditemukan")
	}
	token, err := repository.GenerateToken()
	if err != nil {
		return errors.New("gagal membuat link")
	}
	unusable, err := UnusablePasswordHash()
	if err != nil {
		return errors.New("gagal membuat link")
	}
	if err := repository.IssueStaffResetToken(staff.ID, token, unusable, ttl); err != nil {
		return errors.New("gagal menyimpan link")
	}

	link := fmt.Sprintf("%s/admin-recovery/%s", os.Getenv("ADMIN_APP_URL"), token)
	if err := sendEmail(staff.Email, staff.Username, subject, fmt.Sprintf(bodyFmt, staff.Username, link)); err != nil {
		return fmt.Errorf(sendErrFmt, err)
	}
	return nil
}

// UnusablePasswordHash hashes a random secret that nobody knows, so no password
// matches it. Used for staff who must set their password through a link.
func UnusablePasswordHash() (string, error) {
	secret, err := repository.GenerateToken()
	if err != nil {
		return "", err
	}
	return utils.HashPassword(secret)
}
