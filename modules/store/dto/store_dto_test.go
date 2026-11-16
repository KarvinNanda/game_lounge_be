package dto

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
)

// binding:"dive" makes the validator check each element; without it the tags
// on OperatingHourInput and HolidayInput would never run.
func TestStoreRequest_ValidatesNestedHoursAndHolidays(t *testing.T) {
	base := func() CreateStoreRequest {
		return CreateStoreRequest{Name: "Store", Address: "x"}
	}
	ok := base()
	ok.OperatingHours = []OperatingHourInput{{DayType: "weekday"}}
	ok.Holidays = []HolidayInput{{Date: "2026-12-25"}}
	if err := binding.Validator.ValidateStruct(&ok); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	badDay := base()
	badDay.OperatingHours = []OperatingHourInput{{DayType: "not-a-day"}}
	if binding.Validator.ValidateStruct(&badDay) == nil {
		t.Error("invalid day_type must be rejected")
	}

	badDate := base()
	badDate.Holidays = []HolidayInput{{Date: "25-12-2026"}}
	if binding.Validator.ValidateStruct(&badDate) == nil {
		t.Error("invalid holiday date must be rejected (it was saved as a zero date)")
	}

	upd := UpdateStoreRequest{Name: "Store", Address: "x", OperatingHours: []OperatingHourInput{{DayType: "x"}}}
	if binding.Validator.ValidateStruct(&upd) == nil {
		t.Error("update request must validate nested hours too")
	}
}
