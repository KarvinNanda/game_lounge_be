package service

import "testing"

func TestCanTransitionOrder(t *testing.T) {
	allowed := map[[2]string]bool{
		{"pending", "preparing"}: true, {"pending", "cancelled"}: true,
		{"preparing", "delivered"}: true, {"preparing", "cancelled"}: true,
	}
	all := []string{"pending", "preparing", "delivered", "cancelled"}
	for _, from := range all {
		for _, to := range all {
			want := allowed[[2]string{from, to}]
			if got := canTransitionOrder(from, to); got != want {
				t.Errorf("%s → %s: want %v, got %v", from, to, want, got)
			}
		}
	}
}
