package utils

import (
	"strings"
	"testing"
)

func TestBuildBookingEmailHTML_NamaDiEscape(t *testing.T) {
	out := BuildBookingEmailHTML("BK-1", `<a href="https://evil">klik</a>`, "VIP 1", "Senin", "10:00", "12:00", "2 jam", "Rp 100")
	if strings.Contains(out, `<a href="https://evil">`) {
		t.Error("nama customer harus di-escape, bukan HTML mentah")
	}
	if !strings.Contains(out, "&lt;a href=") {
		t.Error("nama customer harus muncul dalam bentuk escaped")
	}
}

func TestBuildVoucherEmailHTML_SemuaFieldDiEscape(t *testing.T) {
	out := BuildVoucherEmailHTML("<b>x</b>", "<i>v</i>", "<s>c</s>", "<u>e</u>", "<script>d</script>")
	for _, raw := range []string{"<b>x</b>", "<i>v</i>", "<s>c</s>", "<u>e</u>", "<script>"} {
		if strings.Contains(out, raw) {
			t.Errorf("%q harus di-escape", raw)
		}
	}
}

func TestBuildMessage_HeaderInjection(t *testing.T) {
	msg := buildMessage("Game Lounge", "noreply@example.com",
		"Budi\r\nBcc: victim@example.com", "budi@example.com",
		"Halo\r\nX-Evil: 1", "<p>isi</p>")
	head := msg[:strings.Index(msg, "\r\n\r\n")]
	if strings.Contains(head, "\r\nBcc:") || strings.Contains(head, "\r\nX-Evil:") {
		t.Errorf("CR/LF dari nama/subject tidak boleh membuat header baru:\n%s", head)
	}
}

func TestBuildMessage_NamaNonASCII_Encoded(t *testing.T) {
	msg := buildMessage("Game Lounge", "noreply@example.com", "Dévi", "d@example.com", "Halo Dévi", "x")
	head := msg[:strings.Index(msg, "\r\n\r\n")]
	if strings.Contains(head, "Dévi") {
		t.Errorf("header non-ASCII harus RFC 2047 encoded:\n%s", head)
	}
}
