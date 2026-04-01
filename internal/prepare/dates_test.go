package prepare

import (
	"testing"
	"time"
)

func TestAddMonthOverflow(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		months   int
		expected time.Time
	}{
		// Normal cases without overflow
		{name: "Jan 15 + 1 month = Feb 15", input: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)},
		{name: "Jan 15 - 1 month = Dec 15", input: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), months: -1, expected: time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)},

		// Overflow: 31st day in 30-day months
		{name: "Mar 31 + 1 month = Apr 30", input: time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC)},
		{name: "May 31 + 1 month = Jun 30", input: time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)},
		{name: "Oct 31 + 1 month = Nov 30", input: time.Date(2024, 10, 31, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2024, 11, 30, 0, 0, 0, 0, time.UTC)},
		{name: "Apr 30 - 1 month = Mar 30", input: time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC), months: -1, expected: time.Date(2024, 3, 30, 0, 0, 0, 0, time.UTC)},

		// February in leap year (2024)
		{name: "Jan 31 2024 + 1 month = Feb 29", input: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)},
		{name: "Mar 31 2024 - 1 month = Feb 29", input: time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC), months: -1, expected: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)},
		{name: "Feb 29 2024 + 12 months = Feb 28 2025", input: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), months: 12, expected: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)},

		// February in non-leap year (2023)
		{name: "Jan 31 2023 + 1 month = Feb 28", input: time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2023, 2, 28, 0, 0, 0, 0, time.UTC)},
		{name: "Mar 31 2023 - 1 month = Feb 28", input: time.Date(2023, 3, 31, 0, 0, 0, 0, time.UTC), months: -1, expected: time.Date(2023, 2, 28, 0, 0, 0, 0, time.UTC)},

		// Year boundary
		{name: "Dec 31 + 1 month = Jan 31", input: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)},
		{name: "Jan 31 - 1 month = Dec 31", input: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), months: -1, expected: time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)},

		// Multiple months with overflow
		{name: "Jan 31 + 2 months = Mar 31", input: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), months: 2, expected: time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)},
		{name: "Jan 31 + 3 months = Apr 30", input: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), months: 3, expected: time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC)},

		// 30th day (no overflow expected)
		{name: "Apr 30 + 1 month = May 30", input: time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC), months: 1, expected: time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC)},
		{name: "May 30 - 1 month = Apr 30", input: time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC), months: -1, expected: time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addMonthOverflow(tt.input, tt.months)
			if !result.Equal(tt.expected) {
				t.Errorf("got %v, expected %v", result.Format("2006-01-02"), tt.expected.Format("2006-01-02"))
			}
		})
	}
}
