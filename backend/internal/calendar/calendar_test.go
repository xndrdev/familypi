package calendar

import (
	"strings"
	"testing"
	"time"

	"familydisplay/internal/store"
)

const sampleICS = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
	"BEGIN:VEVENT\r\nUID:a1\r\nDTSTART;TZID=Europe/Berlin:20261110T090000\r\nDTEND;TZID=Europe/Berlin:20261110T100000\r\nSUMMARY:Zahnarzt\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:a2\r\nDTSTART;VALUE=DATE:20261112\r\nDTEND;VALUE=DATE:20261113\r\nSUMMARY:Ausflug\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:a3\r\nDTSTART:20261109T170000Z\r\nDTEND:20261109T180000Z\r\nRRULE:FREQ=WEEKLY;COUNT=3\r\nSUMMARY:Training\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:a4\r\nDTSTART:20261111T170000Z\r\nDTEND:20261111T180000Z\r\nSTATUS:CANCELLED\r\nSUMMARY:Abgesagt\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func TestParse(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, loc)
	evs, err := Parse(strings.NewReader(sampleICS), 7, loc, now)
	if err != nil {
		t.Fatal(err)
	}
	SortEvents(evs)

	var titles []string
	for _, e := range evs {
		titles = append(titles, e.Title)
		if e.MemberID == nil || *e.MemberID != 7 {
			t.Errorf("%s: member not set", e.Title)
		}
	}
	want := "Training,Zahnarzt,Ausflug,Training,Training"
	if got := strings.Join(titles, ","); got != want {
		t.Fatalf("titles = %s, want %s", got, want)
	}

	ausflug := evs[2]
	if !ausflug.AllDay || ausflug.Start.Format("2006-01-02 15:04") != "2026-11-12 00:00" {
		t.Errorf("all-day event parsed wrong: %+v", ausflug)
	}
	if evs[1].Start.In(loc).Hour() != 9 {
		t.Errorf("TZID not respected: %v", evs[1].Start)
	}
}

func TestWriteICSFoldsAndEscapes(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	mid := int64(1)
	start := time.Date(2026, 11, 10, 9, 0, 0, 0, loc)
	evs := []store.Event{{
		UID: "local-1", Title: strings.Repeat("Größe; ", 20), Start: start, End: start.Add(time.Hour), MemberID: &mid,
	}}
	var sb strings.Builder
	WriteICS(&sb, evs, map[int64]store.Member{1: {ID: 1, Name: "Mia"}}, loc)
	out := sb.String()

	for _, line := range strings.Split(out, "\r\n") {
		if len(line) > 75 {
			t.Errorf("line longer than 75 octets: %q", line)
		}
	}
	unfolded := strings.ReplaceAll(out, "\r\n ", "")
	if !strings.Contains(unfolded, `SUMMARY:Mia: Größe\; Größe\;`) {
		t.Errorf("summary not escaped/prefixed: %s", unfolded)
	}
	if !strings.Contains(out, "DTSTART:20261110T080000Z") {
		t.Errorf("start not in UTC: %s", out)
	}
}
