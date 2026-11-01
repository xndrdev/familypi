package dateparse

import (
	"testing"
	"time"
)

func TestDate(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	now := time.Date(2026, 11, 6, 15, 0, 0, 0, loc) // Freitag

	cases := map[string]string{
		"heute":          "2026-11-06",
		"Morgen":         "2026-11-07",
		"übermorgen":     "2026-11-08",
		"montag":         "2026-11-09",
		"freitag":        "2026-11-13", // gleicher Wochentag -> naechste Woche
		"am Sa":          "2026-11-07",
		"in 3 tagen":     "2026-11-09",
		"in 2 wochen":    "2026-11-20",
		"24.12.":         "2026-12-24",
		"1.3.":           "2027-03-01", // schon vorbei -> naechstes Jahr
		"6.11.":          "2026-11-06",
		"05.01.27":       "2027-01-05",
		"2026-11-30":     "2026-11-30",
		"  15.08.2027  ": "2027-08-15",
	}
	for in, want := range cases {
		got, err := Date(in, now, loc)
		if err != nil {
			t.Errorf("Date(%q): %v", in, err)
			continue
		}
		if got.Format("2006-01-02") != want {
			t.Errorf("Date(%q) = %s, want %s", in, got.Format("2006-01-02"), want)
		}
	}

	for _, bad := range []string{"", "irgendwann", "31.02.2027", "13.13."} {
		if _, err := Date(bad, now, loc); err == nil {
			t.Errorf("Date(%q) should fail", bad)
		}
	}
}

func TestClock(t *testing.T) {
	cases := map[string][2]int{
		"14:30":  {14, 30},
		"9":      {9, 0},
		"9.15":   {9, 15},
		"18 Uhr": {18, 0},
	}
	for in, want := range cases {
		h, m, err := Clock(in)
		if err != nil || h != want[0] || m != want[1] {
			t.Errorf("Clock(%q) = %d:%d, %v; want %d:%d", in, h, m, err, want[0], want[1])
		}
	}
	for _, bad := range []string{"25:00", "12:75", "mittags"} {
		if _, _, err := Clock(bad); err == nil {
			t.Errorf("Clock(%q) should fail", bad)
		}
	}
}
