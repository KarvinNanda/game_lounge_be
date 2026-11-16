package main

import (
	"log"

	"gorm.io/gorm"
)

// softDeleteRoomsOfDeletedStores fixes data written before DeleteStore also
// soft-deleted the store's rooms: those rooms still counted in the dashboard
// total. Idempotent: only rooms that are still live are touched.
func softDeleteRoomsOfDeletedStores(db *gorm.DB) {
	res := db.Exec(`UPDATE store_rooms r JOIN stores s ON s.id = r.store_id
		SET r.is_active = false, r.deleted_at = s.deleted_at, r.deleted_by = 'migrate'
		WHERE s.deleted_at IS NOT NULL AND r.deleted_at IS NULL`)
	if res.Error != nil {
		log.Fatalf("Gagal menonaktifkan room milik store yang sudah dihapus: %v", res.Error)
	}
	if res.RowsAffected > 0 {
		log.Printf("Room milik store yang sudah dihapus dinonaktifkan: %d", res.RowsAffected)
	}
}
