// Package audit resolves created_by / updated_by into display names for admin
// responses. Those columns hold a staff username for staff actions, "seed" for
// seed data, and a customer id for bookings a customer made online.
package audit

import (
	"log/slog"
	"regexp"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// Audited is implemented by every model with created_by and updated_by.
type Audited interface {
	AuditRefs() (createdBy, updatedBy **string, names *models.AuditNames)
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// FillSlice fills names for a slice of values (e.g. []models.Booking).
func FillSlice[T any, PT interface {
	*T
	Audited
}](items []T) error {
	refs := make([]Audited, len(items))
	for i := range items {
		refs[i] = PT(&items[i])
	}
	return Fill(refs...)
}

// Fill resolves names for all items in at most 2 queries:
//   - staff id -> staff username, customer id -> customer name (soft-deleted
//     rows included, so old records keep a name);
//   - a value that is not an id (username, "seed") is used as is;
//   - an id that matches nothing is shown as is;
//   - nil stays nil.
func Fill(items ...Audited) error {
	ids := map[string]bool{}
	for _, it := range items {
		c, u, _ := it.AuditRefs()
		for _, v := range []*string{*c, *u} {
			if v != nil && uuidRE.MatchString(*v) {
				ids[*v] = true
			}
		}
	}
	names := map[string]string{}
	if len(ids) > 0 {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		var staff []struct{ ID, Username string }
		if err := config.DB.Model(&models.Staff{}).Select("id, username").Where("id IN ?", list).Scan(&staff).Error; err != nil {
			return err
		}
		for _, s := range staff {
			names[s.ID] = s.Username
		}
		var customers []struct{ ID, Name string }
		if err := config.DB.Model(&models.Customer{}).Select("id, name").Where("id IN ?", list).Scan(&customers).Error; err != nil {
			return err
		}
		for _, c := range customers {
			if _, taken := names[c.ID]; !taken {
				names[c.ID] = c.Name
			}
		}
	}
	resolve := func(v *string) *string {
		if v == nil {
			return nil
		}
		if n, ok := names[*v]; ok {
			return &n
		}
		s := *v
		return &s
	}
	for _, it := range items {
		c, u, out := it.AuditRefs()
		out.CreatedByName, out.UpdatedByName = resolve(*c), resolve(*u)
	}
	return nil
}

// ApplySlice is FillSlice for response handlers: names are extra information,
// so a lookup error is logged and the response goes out without them.
func ApplySlice[T any, PT interface {
	*T
	Audited
}](items []T) {
	if err := FillSlice[T, PT](items); err != nil {
		slog.Warn("audit names not resolved", "error", err.Error())
	}
}

// Apply is Fill for response handlers (see ApplySlice).
func Apply(items ...Audited) {
	if err := Fill(items...); err != nil {
		slog.Warn("audit names not resolved", "error", err.Error())
	}
}
