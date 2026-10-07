package utils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestIsDuplicateKey(t *testing.T) {
	dup := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
	if !IsDuplicateKey(dup) || !IsDuplicateKey(fmt.Errorf("wrapped: %w", dup)) {
		t.Error("1062 (juga yang di-wrap) harus terdeteksi")
	}
	if IsDuplicateKey(&mysql.MySQLError{Number: 1452}) || IsDuplicateKey(errors.New("x")) || IsDuplicateKey(nil) {
		t.Error("error lain tidak boleh dianggap duplicate key")
	}
}
