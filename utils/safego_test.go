package utils

import (
	"testing"
	"time"
)

func TestSafeGo_PanicTidakMematikanProses(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		panic("boom")
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine tidak selesai")
	}
}
