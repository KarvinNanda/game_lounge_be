package controller

import (
	"reflect"
	"testing"
)

func TestParseStatusFilter(t *testing.T) {
	cases := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{"", nil, false},
		{"upcoming", []string{"upcoming"}, false},
		{"upcoming,ongoing", []string{"upcoming", "ongoing"}, false},
		{" completed , cancelled ", []string{"completed", "cancelled"}, false},
		{"upcoming,", []string{"upcoming"}, false},
		{"pending", nil, true},
		{"upcoming,'; DROP", nil, true},
	}
	for _, tc := range cases {
		got, err := parseStatusFilter(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("%q: wantErr=%v, err=%v", tc.in, tc.wantErr, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: want %v, got %v", tc.in, tc.want, got)
		}
	}
}
