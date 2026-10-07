package utils

import (
	"log"
	"runtime/debug"
)

// SafeGo menjalankan fn di goroutine baru dengan recover. gin.Recovery hanya
// melindungi goroutine request; panic di goroutine lain (email, sync, job)
// akan mematikan seluruh proses API.
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] goroutine: %v\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}
