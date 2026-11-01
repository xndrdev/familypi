package calendar

import (
	"fmt"
	"strings"
	"time"

	"familydisplay/internal/store"
)

// WriteICS erzeugt einen ICS-Feed, den Familienmitglieder auf dem Handy abonnieren koennen.
func WriteICS(sb *strings.Builder, events []store.Event, members map[int64]store.Member, loc *time.Location) {
	now := time.Now().UTC().Format("20060102T150405Z")
	sb.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//familydisplay//DE\r\nCALSCALE:GREGORIAN\r\n")
	sb.WriteString("X-WR-CALNAME:Familie\r\n")
	for _, e := range events {
		title := e.Title
		if e.MemberID != nil {
			if m, ok := members[*e.MemberID]; ok {
				title = m.Name + ": " + title
			}
		}
		sb.WriteString("BEGIN:VEVENT\r\n")
		writeLine(sb, "UID", e.UID+"@familydisplay")
		writeLine(sb, "DTSTAMP", now)
		if e.AllDay {
			writeLine(sb, "DTSTART;VALUE=DATE", e.Start.In(loc).Format("20060102"))
			writeLine(sb, "DTEND;VALUE=DATE", e.End.In(loc).Format("20060102"))
		} else {
			writeLine(sb, "DTSTART", e.Start.UTC().Format("20060102T150405Z"))
			writeLine(sb, "DTEND", e.End.UTC().Format("20060102T150405Z"))
		}
		writeLine(sb, "SUMMARY", escapeText(title))
		if e.Location != "" {
			writeLine(sb, "LOCATION", escapeText(e.Location))
		}
		if e.Notes != "" {
			writeLine(sb, "DESCRIPTION", escapeText(e.Notes))
		}
		sb.WriteString("END:VEVENT\r\n")
	}
	sb.WriteString("END:VCALENDAR\r\n")
}

func escapeText(s string) string {
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`)
	return r.Replace(s)
}

// writeLine faltet Zeilen nach RFC 5545 bei 75 Oktetts, ohne UTF-8-Zeichen zu zerschneiden.
func writeLine(sb *strings.Builder, key, value string) {
	line := fmt.Sprintf("%s:%s", key, value)
	limit := 75
	for len(line) > limit {
		cut := limit
		for cut > 0 && (line[cut]&0xC0) == 0x80 {
			cut--
		}
		sb.WriteString(line[:cut])
		sb.WriteString("\r\n ")
		line = line[cut:]
		limit = 74 // Folgezeilen beginnen mit einem Leerzeichen
	}
	sb.WriteString(line)
	sb.WriteString("\r\n")
}
