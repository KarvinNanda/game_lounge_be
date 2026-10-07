package models

import (
	"reflect"
	"strings"
	"testing"
)

// GORM melewati nilai zero (false) saat INSERT jika field punya tag default,
// sehingga MySQL mengisi DEFAULT true: data yang dibuat "nonaktif" tersimpan aktif.
// Field bool tidak boleh punya default:true di tag GORM.
func TestBoolFieldsTanpaDefaultTrue(t *testing.T) {
	all := []any{Banner{}, FnbCategory{}, FnbItem{}, CustomerPlayCredit{}, NotificationTemplate{}, Facility{},
		PlayCreditsPackage{}, RoomTemplate{}, StoreFlashSale{}, StoreOperatingHour{}, StorePricing{}, StoreRoom{},
		Voucher{}, Staff{}, Role{}, Customer{}, EventBooking{}}
	for _, m := range all {
		typ := reflect.TypeOf(m)
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.Type.Kind() == reflect.Bool && strings.Contains(f.Tag.Get("gorm"), "default:true") {
				t.Errorf("%s.%s: hapus default:true dari tag gorm", typ.Name(), f.Name)
			}
		}
	}
}
