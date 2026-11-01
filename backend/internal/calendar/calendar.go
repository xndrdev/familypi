// Package calendar importiert externe iCal-Feeds (Google, iCloud, Outlook) und exportiert den Familienkalender als ICS.
package calendar

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/apognu/gocal"

	"familydisplay/internal/store"
)

// Window legt fest, wie weit externe Feeds eingelesen werden.
const (
	lookBack  = 7 * 24 * time.Hour
	lookAhead = 120 * 24 * time.Hour
)

type FeedStatus struct {
	MemberID  int64     `json:"memberId"`
	Events    int       `json:"events"`
	Error     string    `json:"error"`
	FetchedAt time.Time `json:"fetchedAt"`
}

type Importer struct {
	loc    *time.Location
	client *http.Client

	mu     sync.RWMutex
	events map[int64][]store.Event
	status map[int64]FeedStatus
}

func NewImporter(loc *time.Location) *Importer {
	return &Importer{
		loc:    loc,
		client: &http.Client{Timeout: 30 * time.Second},
		events: map[int64][]store.Event{},
		status: map[int64]FeedStatus{},
	}
}

// Refresh laedt alle Feeds der uebergebenen Mitglieder neu. Mitglieder ohne URL werden aus dem Cache entfernt.
func (im *Importer) Refresh(ctx context.Context, members []store.Member) {
	events := map[int64][]store.Event{}
	status := map[int64]FeedStatus{}
	for _, m := range members {
		if strings.TrimSpace(m.ICalURL) == "" {
			continue
		}
		evs, err := im.fetch(ctx, m)
		st := FeedStatus{MemberID: m.ID, Events: len(evs), FetchedAt: time.Now()}
		if err != nil {
			slog.Warn("ical import failed", "member", m.Name, "err", err)
			st.Error = err.Error()
			// Bei Fehlern den alten Stand behalten, damit das Display nicht leer wird.
			im.mu.RLock()
			evs = im.events[m.ID]
			im.mu.RUnlock()
		}
		events[m.ID] = evs
		status[m.ID] = st
	}
	im.mu.Lock()
	im.events, im.status = events, status
	im.mu.Unlock()
}

func (im *Importer) fetch(ctx context.Context, m store.Member) ([]store.Event, error) {
	url := strings.TrimSpace(m.ICalURL)
	if rest, ok := strings.CutPrefix(url, "webcal://"); ok {
		url = "https://" + rest
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := im.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return Parse(io.LimitReader(resp.Body, 20<<20), m.ID, im.loc, time.Now())
}

// Parse liest einen ICS-Feed und liefert die Termine im Import-Fenster rund um now.
func Parse(r io.Reader, memberID int64, loc *time.Location, now time.Time) ([]store.Event, error) {
	start, end := now.Add(-lookBack), now.Add(lookAhead)
	p := gocal.NewParser(r)
	p.Start, p.End = &start, &end
	p.AllDayEventsTZ = loc
	// Tolerant parsen: Feeds aus der Praxis lassen gern Pflichtfelder wie DTSTAMP weg.
	p.Strict.Mode = gocal.StrictModeFailAttribute
	p.Duplicate.Mode = gocal.DuplicateModeKeepFirst
	if err := p.Parse(); err != nil {
		return nil, err
	}
	out := make([]store.Event, 0, len(p.Events))
	for _, e := range p.Events {
		if e.Start == nil || strings.EqualFold(e.Status, "CANCELLED") {
			continue
		}
		allDay := e.RawStart.Params["VALUE"] == "DATE" || len(e.RawStart.Value) == 8
		ev := store.Event{
			UID:      fmt.Sprintf("ical-%d-%s-%d", memberID, e.Uid, e.Start.Unix()),
			Title:    e.Summary,
			Start:    e.Start.In(loc),
			AllDay:   allDay,
			MemberID: &memberID,
			Location: e.Location,
			Source:   "ical",
		}
		switch {
		case e.End != nil:
			ev.End = e.End.In(loc)
		case allDay:
			ev.End = ev.Start.AddDate(0, 0, 1)
		default:
			ev.End = ev.Start.Add(time.Hour)
		}
		out = append(out, ev)
	}
	return out, nil
}

// Events liefert alle importierten Termine, die das Intervall [from, to) beruehren.
func (im *Importer) Events(from, to time.Time) []store.Event {
	im.mu.RLock()
	defer im.mu.RUnlock()
	out := []store.Event{}
	for _, evs := range im.events {
		for _, e := range evs {
			if e.End.After(from) && e.Start.Before(to) {
				out = append(out, e)
			}
		}
	}
	return out
}

func (im *Importer) Status() []FeedStatus {
	im.mu.RLock()
	defer im.mu.RUnlock()
	out := make([]FeedStatus, 0, len(im.status))
	for _, s := range im.status {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MemberID < out[j].MemberID })
	return out
}

// SortEvents sortiert nach Start; Ganztagestermine stehen vor Terminen mit Uhrzeit am selben Tag.
func SortEvents(evs []store.Event) {
	sort.SliceStable(evs, func(i, j int) bool {
		a, b := evs[i], evs[j]
		if a.Start.Equal(b.Start) {
			return a.AllDay && !b.AllDay
		}
		return a.Start.Before(b.Start)
	})
}
