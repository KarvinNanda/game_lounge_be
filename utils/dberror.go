package utils

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicateKey bernilai true jika err adalah pelanggaran UNIQUE di MySQL (1062).
func IsDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
