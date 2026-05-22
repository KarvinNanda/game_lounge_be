package service

import (
	"errors"
	"fmt"
	"log"
	"os"

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
	go sendResetEmail(staff.Email, staff.Username, token)
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

	// 2. Validasi staff masih super admin
	if !resetToken.Staff.Role.IsSystem {
		return errors.New("akses ditolak")
	}

	// 3. Hash password baru
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.New("gagal memproses password")
	}

	// 4. Update password di DB
	if err := repository.UpdateStaffPassword(resetToken.StaffID, hashedPassword); err != nil {
		return errors.New("gagal menyimpan password baru")
	}

	// 5. Tandai token sudah dipakai (one-time use)
	_ = repository.MarkTokenUsed(token)

	return nil
}

// sendResetEmail mengirim email berisi link reset password ke super admin.
func sendResetEmail(email, name, token string) {
	appURL := os.Getenv("APP_URL")
	resetLink := fmt.Sprintf("%s/admin-recovery/%s", appURL, token)

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

	if err := utils.SendEmail(email, name, subject, body); err != nil {
		log.Printf("Gagal kirim email reset password ke %s: %v", email, err)
	}
}
