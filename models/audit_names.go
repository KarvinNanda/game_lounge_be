package models

// AuditNames are display names for created_by and updated_by. They are not
// stored: audit.Fill resolves them (staff id -> username, customer id -> name,
// usernames as is) for admin list and detail responses. Omitted when not filled.
type AuditNames struct {
	CreatedByName *string `json:"created_by_name,omitempty" gorm:"-"`
	UpdatedByName *string `json:"updated_by_name,omitempty" gorm:"-"`
}

// AuditRefs give audit.Fill access to the audit fields of each model.
func (m *Booking) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
func (m *EventBooking) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
func (m *Voucher) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
func (m *Role) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
func (m *Staff) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
func (m *Customer) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
func (m *StoreFlashSale) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
func (m *PlayCreditsPackage) AuditRefs() (**string, **string, *AuditNames) {
	return &m.CreatedBy, &m.UpdatedBy, &m.AuditNames
}
