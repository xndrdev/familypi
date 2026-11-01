// Package dateparse versteht die deutschen Datums- und Zeitangaben, die man im Chat tippt.
package dateparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var weekdays = map[string]time.Weekday{
	"mo": time.Monday, "montag": time.Monday,
	"di": time.Tuesday, "dienstag": time.Tuesday,
	"mi": time.Wednesday, "mittwoch": time.Wednesday,
	"do": time.Thursday, "donnerstag": time.Thursday,
	"fr": time.Friday, "freitag": time.Friday,
	"sa": time.Saturday, "samstag": time.Saturday,
	"so": time.Sunday, "sonntag": time.Sunday,
}

var (
	reISO      = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)
	reGerman   = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.(\d{2}|\d{4})?$`)
	reInDays   = regexp.MustCompile(`^in (\d+) (tag|tagen|woche|wochen)$`)
	reTime     = regexp.MustCompile(`^(\d{1,2})(?:[:.](\d{2}))?(?: ?uhr)?$`)
	whitespace = regexp.MustCompile(`\s+`)
)

// Date wandelt Angaben wie "morgen", "freitag", "5.11.", "05.11.2026", "2026-11-05" oder "in 3 tagen"
// in ein Datum (Mitternacht in loc) um. Wochentage meinen immer das naechste Vorkommen nach heute.
// Datumsangaben ohne Jahr, die schon vorbei sind, werden ins naechste Jahr gelegt.
func Date(input string, now time.Time, loc *time.Location) (time.Time, error) {
	s := normalize(input)
	now = now.In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	switch s {
	case "heute":
		return today, nil
	case "morgen":
		return today.AddDate(0, 0, 1), nil
	case "übermorgen", "uebermorgen":
		return today.AddDate(0, 0, 2), nil
	}
	if wd, ok := weekdays[strings.TrimPrefix(s, "nächsten ")]; ok {
		diff := (int(wd) - int(today.Weekday()) + 7) % 7
		if diff == 0 {
			diff = 7
		}
		return today.AddDate(0, 0, diff), nil
	}
	if m := reInDays.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		if strings.HasPrefix(m[2], "woche") {
			n *= 7
		}
		return today.AddDate(0, 0, n), nil
	}
	if m := reISO.FindStringSubmatch(s); m != nil {
		return build(atoi(m[1]), atoi(m[2]), atoi(m[3]), loc)
	}
	if m := reGerman.FindStringSubmatch(s); m != nil {
		day, month := atoi(m[1]), atoi(m[2])
		if m[3] == "" {
			d, err := build(today.Year(), month, day, loc)
			if err == nil && d.Before(today) {
				d, err = build(today.Year()+1, month, day, loc)
			}
			return d, err
		}
		year := atoi(m[3])
		if year < 100 {
			year += 2000
		}
		return build(year, month, day, loc)
	}
	return time.Time{}, fmt.Errorf("Datum %q nicht verstanden (z.B. heute, morgen, freitag, 5.11., 2026-11-05)", input)
}

// Clock liest Uhrzeiten wie "14:30", "9", "9.15" oder "18 uhr" und liefert Stunde und Minute.
func Clock(input string) (hour, minute int, err error) {
	m := reTime.FindStringSubmatch(normalize(input))
	if m == nil {
		return 0, 0, fmt.Errorf("Uhrzeit %q nicht verstanden (z.B. 14:30)", input)
	}
	hour = atoi(m[1])
	if m[2] != "" {
		minute = atoi(m[2])
	}
	if hour > 23 || minute > 59 {
		return 0, 0, fmt.Errorf("Uhrzeit %q ist ungültig", input)
	}
	return hour, minute, nil
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "am ")
	return whitespace.ReplaceAllString(s, " ")
}

func build(year, month, day int, loc *time.Location) (time.Time, error) {
	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
	// time.Date normalisiert z.B. den 31.02. – solche Eingaben lehnen wir ab.
	if d.Day() != day || int(d.Month()) != month {
		return time.Time{}, fmt.Errorf("Datum %02d.%02d.%d gibt es nicht", day, month, year)
	}
	return d, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
