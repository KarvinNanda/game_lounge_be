package utils

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func respond(status int, msg string) string {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ResponseError(c, status, msg)
	var body struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body.Message
}

func TestResponseError_TidakMembocorkanErrorInternal(t *testing.T) {
	cases := []struct {
		name   string
		status int
		msg    string
		leak   bool // true = pesan asli TIDAK boleh sampai ke client
	}{
		{"500 apa pun disembunyikan", 500, "dial tcp 10.0.0.5:3306: connection refused", true},
		{"400 berisi error MySQL", 400, "Error 1062 (23000): Duplicate entry 'x' for key 'staffs.idx_staffs_email'", true},
		{"400 berisi error database/sql", 400, "sql: no rows in result set", true},
		{"400 pesan validasi service tetap tampil", 400, "slot jam harus berurutan tanpa jeda", false},
		{"404 pesan biasa tetap tampil", 404, "Booking tidak ditemukan", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := respond(tc.status, tc.msg)
			if tc.leak && got == tc.msg {
				t.Errorf("pesan internal bocor ke client: %q", got)
			}
			if !tc.leak && got != tc.msg {
				t.Errorf("pesan aman harus tetap tampil, got %q", got)
			}
		})
	}
}
