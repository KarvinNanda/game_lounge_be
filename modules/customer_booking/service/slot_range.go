package service

import (
	"errors"
	"game_lounge_be/utils"
	"sort"
)

// resolveSlotRange mengubah slot pilihan customer menjadi 1 rentang waktu.
//
// dayOrder = slot resmi hari itu, urut dari jam buka (slot setelah tengah malam
// ada di akhir). Slot pilihan wajib unik, ada di dayOrder, dan berurutan.
// Dulu start/end diambil dari slot terkecil/terbesar secara string, sehingga
// ["10:00","20:00"] dibayar 2 jam tapi memblokir room 10:00–21:00.
func resolveSlotRange(selected, dayOrder []string) (start, end string, hours int, err error) {
	if len(selected) == 0 {
		return "", "", 0, errors.New("pilih minimal 1 jam bermain")
	}
	pos := make(map[string]int, len(dayOrder))
	for i, s := range dayOrder {
		pos[s] = i
	}

	idx := make([]int, 0, len(selected))
	seen := make(map[string]bool, len(selected))
	for _, s := range selected {
		if seen[s] {
			return "", "", 0, errors.New("slot jam tidak boleh duplikat")
		}
		seen[s] = true
		i, ok := pos[s]
		if !ok {
			return "", "", 0, errors.New("slot " + s + " di luar jam operasional")
		}
		idx = append(idx, i)
	}
	sort.Ints(idx)
	for k := 1; k < len(idx); k++ {
		if idx[k] != idx[k-1]+1 {
			return "", "", 0, errors.New("slot jam harus berurutan tanpa jeda")
		}
	}

	last := dayOrder[idx[len(idx)-1]]
	return dayOrder[idx[0]], utils.MinsToClock(utils.MinsOf(last) + 60), len(idx), nil
}
