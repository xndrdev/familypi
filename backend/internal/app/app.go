// Package app buendelt die Anwendungslogik, die sowohl die HTTP-API als auch der Discord-Bot nutzen.
package app

import (
	"context"
	"log/slog"
	"time"

	"familydisplay/internal/calendar"
	"familydisplay/internal/hub"
	"familydisplay/internal/store"
	"familydisplay/internal/weather"
)

// Topics fuer Live-Updates.
const (
	TopicMembers = "members"
	TopicTodos   = "todos"
	TopicGoals   = "goals"
	TopicEvents  = "events"
	TopicWeather = "weather"
)

type App struct {
	Store    *store.Store
	Hub      *hub.Hub
	Importer *calendar.Importer
	Weather  *weather.Service
	Loc      *time.Location
}

// Events liefert lokale und importierte Termine im Intervall [from, to), sortiert.
func (a *App) Events(ctx context.Context, from, to time.Time) ([]store.Event, error) {
	local, err := a.Store.ListEvents(ctx, from, to)
	if err != nil {
		return nil, err
	}
	for i := range local {
		local[i].Start, local[i].End = local[i].Start.In(a.Loc), local[i].End.In(a.Loc)
	}
	all := append(local, a.Importer.Events(from, to)...)
	calendar.SortEvents(all)
	return all, nil
}

func (a *App) RefreshCalendars(ctx context.Context) {
	members, err := a.Store.ListMembers(ctx)
	if err != nil {
		slog.Error("list members for calendar refresh", "err", err)
		return
	}
	a.Importer.Refresh(ctx, members)
	a.Hub.Publish(TopicEvents)
}

// RunCalendarSync aktualisiert die externen Kalender periodisch, bis ctx endet.
func (a *App) RunCalendarSync(ctx context.Context, every time.Duration) {
	a.RefreshCalendars(ctx)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.RefreshCalendars(ctx)
		}
	}
}

// Today liefert Mitternacht des aktuellen Tages in der Familien-Zeitzone.
func (a *App) Today() time.Time {
	now := time.Now().In(a.Loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, a.Loc)
}

// MemberByDiscordID findet das Mitglied, das zu einem Discord-Account gehoert.
func (a *App) MemberByDiscordID(ctx context.Context, discordID string) (*store.Member, error) {
	members, err := a.Store.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		if m.DiscordUserID != "" && m.DiscordUserID == discordID {
			return &m, nil
		}
	}
	return nil, nil
}
