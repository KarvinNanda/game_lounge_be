package service

import (
	"sync"
	"testing"

	"game_lounge_be/modules/store/dto"
)

// Two admins save the same form at the same time. Without a lock both read
// "0 units" and both add 2, leaving 4.
func TestUpdateStore_ConcurrentSaveDoesNotDuplicateUnits(t *testing.T) {
	skipIfNoDB(t)
	for round := 0; round < 5; round++ {
		storeID, tmplID := newRoomsFixture(t)
		req := updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2})

		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, _ = UpdateStore(storeID, req, "test")
			}()
		}
		close(start)
		wg.Wait()

		if n := len(unitsOf(t, storeID)); n != 2 {
			t.Fatalf("round %d: want 2 units, got %d", round, n)
		}
	}
}
