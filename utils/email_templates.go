package utils

import "fmt"

// BuildBookingEmailHTML membuat HTML email konfirmasi booking yang menarik.
// Dipakai sebagai fallback ketika template DB belum diisi admin.
func BuildBookingEmailHTML(bookingCode, customerName, roomName, date, startTime, endTime, durasi, totalHarga string) string {
	roomRow := ""
	if roomName != "" {
		roomRow = fmt.Sprintf(`
          <tr>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#6b7280;font-size:13px;width:42%%">🎮&nbsp; Ruangan</td>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#111827;font-size:13px;font-weight:600">%s</td>
          </tr>`, roomName)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background-color:#f5f3ff;font-family:Arial,Helvetica,sans-serif">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f3ff;padding:32px 16px">
  <tr><td align="center">
  <table width="520" cellpadding="0" cellspacing="0" style="max-width:520px;background:white;border-radius:16px;overflow:hidden;box-shadow:0 6px 28px rgba(124,58,237,0.18)">

    <!-- ═══ HEADER ═══ -->
    <tr>
      <td style="background:#7c3aed;padding:30px 20px;text-align:center">
        <p style="margin:0 0 6px;font-size:36px">🎮</p>
        <h1 style="margin:0 0 4px;color:white;font-size:22px;font-weight:bold;letter-spacing:2px">QUANTUM GAMING CENTER</h1>
        <p style="margin:0;color:#c4b5fd;font-size:13px">PlayStation Rental</p>
      </td>
    </tr>

    <!-- ═══ SUCCESS BADGE ═══ -->
    <tr>
      <td style="background:#f0fdf4;padding:14px 20px;text-align:center;border-bottom:2px solid #bbf7d0">
        <span style="background:#16a34a;color:white;border-radius:20px;padding:6px 20px;font-size:12px;font-weight:bold;letter-spacing:1px">✓&nbsp;&nbsp;BOOKING BERHASIL DIKONFIRMASI</span>
      </td>
    </tr>

    <!-- ═══ BOOKING CODE ═══ -->
    <tr>
      <td style="padding:32px 28px;text-align:center">
        <p style="margin:0 0 12px;color:#7c3aed;font-size:11px;font-weight:bold;letter-spacing:3px;text-transform:uppercase">Kode Booking Kamu</p>
        <div style="background:#faf5ff;border:2px dashed #7c3aed;border-radius:12px;padding:20px 28px;display:inline-block">
          <span style="color:#5b21b6;font-size:36px;font-weight:bold;letter-spacing:5px;font-family:Courier New,monospace">%s</span>
        </div>
        <p style="margin:14px 0 0;color:#9ca3af;font-size:12px">Tunjukkan kode ini kepada staff saat tiba di lokasi</p>
      </td>
    </tr>

    <!-- ═══ DIVIDER ═══ -->
    <tr><td style="padding:0 28px"><div style="height:1px;background:#ede9fe"></div></td></tr>

    <!-- ═══ DETAIL TABLE ═══ -->
    <tr>
      <td style="padding:24px 28px">
        <p style="margin:0 0 14px;color:#7c3aed;font-size:11px;font-weight:bold;letter-spacing:2px;text-transform:uppercase">Detail Sesi</p>
        <table width="100%%" cellpadding="0" cellspacing="0">
          <tr>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#6b7280;font-size:13px;width:42%%">👤&nbsp; Nama</td>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#111827;font-size:13px;font-weight:600">%s</td>
          </tr>
          %s
          <tr>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#6b7280;font-size:13px">📅&nbsp; Tanggal</td>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#111827;font-size:13px">%s</td>
          </tr>
          <tr>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#6b7280;font-size:13px">🕐&nbsp; Waktu</td>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#111827;font-size:13px">%s – %s</td>
          </tr>
          <tr>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#6b7280;font-size:13px">⏱&nbsp; Durasi</td>
            <td style="padding:9px 0;border-bottom:1px solid #f3f4f6;color:#111827;font-size:13px">%s Jam</td>
          </tr>
          <tr>
            <td style="padding:14px 0 0;color:#6b7280;font-size:13px;vertical-align:middle">💰&nbsp; Total Bayar</td>
            <td style="padding:14px 0 0;vertical-align:middle">
              <span style="color:#7c3aed;font-size:20px;font-weight:bold">Rp %s</span>
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <!-- ═══ CTA ═══ -->
    <tr>
      <td style="background:#faf5ff;padding:18px 28px;border-top:1px solid #ede9fe;text-align:center">
        <p style="margin:0;color:#6d28d9;font-size:13px;font-weight:600">🕹️&nbsp;&nbsp;Sampai jumpa di Quantum Gaming Center — siap main bareng!</p>
      </td>
    </tr>

    <!-- ═══ FOOTER ═══ -->
    <tr>
      <td style="padding:14px;text-align:center;border-top:1px solid #f3f4f6">
        <p style="margin:0;color:#d1d5db;font-size:11px">© Quantum Gaming Center &nbsp;·&nbsp; PlayStation Rental</p>
      </td>
    </tr>

  </table>
  </td></tr>
</table>
</body>
</html>`,
		bookingCode,
		customerName, roomRow,
		date,
		startTime, endTime,
		durasi,
		totalHarga,
	)
}

// BuildVoucherEmailHTML membuat HTML email notifikasi voucher yang menarik.
// Dipakai sebagai fallback ketika template DB belum diisi admin.
func BuildVoucherEmailHTML(customerName, voucherName, code, expiry, description string) string {
	descBlock := ""
	if description != "" {
		descBlock = fmt.Sprintf(`
    <tr>
      <td style="padding:0 28px 20px;text-align:center">
        <p style="margin:0;color:#6b7280;font-size:13px;font-style:italic">"%s"</p>
      </td>
    </tr>`, description)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background-color:#f5f3ff;font-family:Arial,Helvetica,sans-serif">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f3ff;padding:32px 16px">
  <tr><td align="center">
  <table width="520" cellpadding="0" cellspacing="0" style="max-width:520px;background:white;border-radius:16px;overflow:hidden;box-shadow:0 6px 28px rgba(124,58,237,0.18)">

    <!-- ═══ HEADER ═══ -->
    <tr>
      <td style="background:#7c3aed;padding:30px 20px;text-align:center">
        <p style="margin:0 0 6px;font-size:38px">🎁</p>
        <h1 style="margin:0 0 4px;color:white;font-size:20px;font-weight:bold;letter-spacing:1px">Voucher Spesial Untukmu!</h1>
        <p style="margin:0;color:#c4b5fd;font-size:13px">Quantum Gaming Center &nbsp;·&nbsp; PlayStation Rental</p>
      </td>
    </tr>

    <!-- ═══ GREETING ═══ -->
    <tr>
      <td style="padding:28px 28px 12px">
        <p style="margin:0;color:#374151;font-size:15px">Halo, <strong style="color:#5b21b6">%s</strong>! 🎮</p>
        <p style="margin:10px 0 0;color:#6b7280;font-size:13px;line-height:1.6">
          Kamu mendapatkan voucher eksklusif dari kami. Yuk, gunakan segera sebelum kedaluwarsa!
        </p>
      </td>
    </tr>

    <!-- ═══ VOUCHER CARD ═══ -->
    <tr>
      <td style="padding:8px 28px 24px">
        <div style="border:2px dashed #7c3aed;border-radius:14px;background:#faf5ff;padding:26px 16px;text-align:center">
          <p style="margin:0 0 10px;color:#7c3aed;font-size:11px;font-weight:bold;letter-spacing:3px;text-transform:uppercase">%s</p>
          <p style="margin:0;color:#5b21b6;font-size:38px;font-weight:bold;letter-spacing:7px;font-family:Courier New,monospace">%s</p>
          <div style="display:inline-block;background:#ede9fe;border-radius:20px;padding:5px 16px;margin-top:14px">
            <span style="color:#6d28d9;font-size:12px;font-weight:600">🗓&nbsp; Berlaku sampai: %s</span>
          </div>
        </div>
      </td>
    </tr>

    %s

    <!-- ═══ DIVIDER ═══ -->
    <tr><td style="padding:0 28px"><div style="height:1px;background:#ede9fe"></div></td></tr>

    <!-- ═══ CARA PAKAI ═══ -->
    <tr>
      <td style="padding:20px 28px">
        <p style="margin:0 0 12px;color:#374151;font-size:13px;font-weight:bold">📋&nbsp; Cara Pakai Voucher:</p>
        <table cellpadding="0" cellspacing="0">
          <tr>
            <td style="padding:5px 0;color:#6b7280;font-size:13px">
              &nbsp;&nbsp;1.&nbsp; Hubungi staff atau buka aplikasi saat booking
            </td>
          </tr>
          <tr>
            <td style="padding:5px 0;color:#6b7280;font-size:13px">
              &nbsp;&nbsp;2.&nbsp; Masukkan kode voucher di kolom yang tersedia
            </td>
          </tr>
          <tr>
            <td style="padding:5px 0;color:#6b7280;font-size:13px">
              &nbsp;&nbsp;3.&nbsp; Diskon otomatis diterapkan ke total tagihan 🎉
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <!-- ═══ WARNING NOTE ═══ -->
    <tr>
      <td style="background:#fffbeb;padding:13px 28px;border-top:1px solid #fde68a;border-bottom:1px solid #fde68a">
        <p style="margin:0;color:#92400e;font-size:12px;text-align:center">
          ⚠️&nbsp; Voucher berlaku <strong>1x per akun</strong>. Tidak dapat digabung dengan promo lain.
        </p>
      </td>
    </tr>

    <!-- ═══ FOOTER ═══ -->
    <tr>
      <td style="padding:16px;text-align:center">
        <p style="margin:0;color:#d1d5db;font-size:11px">© Quantum Gaming Center &nbsp;·&nbsp; PlayStation Rental</p>
      </td>
    </tr>

  </table>
  </td></tr>
</table>
</body>
</html>`,
		customerName,
		voucherName, code, expiry,
		descBlock,
	)
}
