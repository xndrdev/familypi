package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"familydisplay/internal/app"
	"familydisplay/internal/store"
)

var (
	reDate  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// ---- Members ----

func (api *API) listMembers(w http.ResponseWriter, r *http.Request) {
	members, err := api.app.Store.ListMembers(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	if !api.authorized(r) {
		for i := range members {
			members[i] = publicMember(members[i])
		}
	}
	writeJSON(w, members)
}

func (api *API) createMember(w http.ResponseWriter, r *http.Request) {
	var m store.Member
	if !decode(w, r, &m) || !validMember(w, &m) {
		return
	}
	if err := api.app.Store.CreateMember(r.Context(), &m); err != nil {
		serverError(w, err)
		return
	}
	api.memberChanged()
	writeJSON(w, m)
}

func (api *API) updateMember(w http.ResponseWriter, r *http.Request) {
	var m store.Member
	id, err := pathID(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "ungültige ID")
		return
	}
	if !decode(w, r, &m) || !validMember(w, &m) {
		return
	}
	m.ID = id
	if err := api.app.Store.UpdateMember(r.Context(), &m); err != nil {
		storeError(w, err)
		return
	}
	api.memberChanged()
	writeJSON(w, m)
}

func (api *API) deleteMember(w http.ResponseWriter, r *http.Request) {
	api.deleteByID(w, r, api.app.Store.DeleteMember, app.TopicMembers, app.TopicGoals, app.TopicTodos, app.TopicEvents)
	go api.app.RefreshCalendars(context.Background())
}

// memberChanged benachrichtigt das Display und laedt den iCal-Feed neu, falls einer hinterlegt ist.
func (api *API) memberChanged() {
	api.app.Hub.Publish(app.TopicMembers)
	go api.app.RefreshCalendars(context.Background())
}

func validMember(w http.ResponseWriter, m *store.Member) bool {
	m.Name = strings.TrimSpace(m.Name)
	m.ICalURL = strings.TrimSpace(m.ICalURL)
	m.DiscordUserID = strings.TrimSpace(m.DiscordUserID)
	if m.Color == "" {
		m.Color = "#4f8cff"
	}
	switch {
	case !validText(m.Name, 40):
		httpError(w, http.StatusBadRequest, "Name fehlt oder ist zu lang")
	case !reColor.MatchString(m.Color):
		httpError(w, http.StatusBadRequest, "Farbe muss im Format #rrggbb sein")
	case utf8.RuneCountInString(m.Emoji) > 8:
		httpError(w, http.StatusBadRequest, "Emoji ist zu lang")
	case m.ICalURL != "" && !validICalURL(m.ICalURL):
		httpError(w, http.StatusBadRequest, "iCal-URL muss mit https:// oder webcal:// beginnen")
	default:
		return true
	}
	return false
}

func validICalURL(u string) bool {
	return strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "webcal://") || strings.HasPrefix(u, "http://")
}

// ---- Todos ----

func (api *API) listTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := api.app.Store.ListTodos(r.Context(), r.URL.Query().Get("all") == "1")
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, todos)
}

func (api *API) createTodo(w http.ResponseWriter, r *http.Request) {
	var t store.Todo
	if !decode(w, r, &t) || !api.validTodo(w, r, &t) {
		return
	}
	if t.CreatedBy == "" {
		t.CreatedBy = "Web"
	}
	if err := api.app.Store.CreateTodo(r.Context(), &t); err != nil {
		serverError(w, err)
		return
	}
	api.app.Hub.Publish(app.TopicTodos)
	writeJSON(w, t)
}

func (api *API) updateTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "ungültige ID")
		return
	}
	existing, err := api.app.Store.GetTodo(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	t := existing
	if !decode(w, r, &t) || !api.validTodo(w, r, &t) {
		return
	}
	t.ID, t.DoneAt = id, existing.DoneAt
	if err := api.app.Store.UpdateTodo(r.Context(), &t); err != nil {
		storeError(w, err)
		return
	}
	api.app.Hub.Publish(app.TopicTodos)
	writeJSON(w, t)
}

func (api *API) deleteTodo(w http.ResponseWriter, r *http.Request) {
	api.deleteByID(w, r, api.app.Store.DeleteTodo, app.TopicTodos)
}

func (api *API) validTodo(w http.ResponseWriter, r *http.Request, t *store.Todo) bool {
	t.Title = strings.TrimSpace(t.Title)
	if t.Due != nil && *t.Due == "" {
		t.Due = nil
	}
	switch {
	case !validText(t.Title, 200):
		httpError(w, http.StatusBadRequest, "Titel fehlt oder ist zu lang")
	case t.Due != nil && !reDate.MatchString(*t.Due):
		httpError(w, http.StatusBadRequest, "Fälligkeit muss im Format YYYY-MM-DD sein")
	case !api.memberExists(w, r, t.MemberID):
	default:
		return true
	}
	return false
}

// ---- Goals ----

func (api *API) listGoals(w http.ResponseWriter, r *http.Request) {
	goals, err := api.app.Store.ListGoals(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, goals)
}

func (api *API) createGoal(w http.ResponseWriter, r *http.Request) {
	var g store.Goal
	if !decode(w, r, &g) || !api.validGoal(w, r, &g) {
		return
	}
	if err := api.app.Store.CreateGoal(r.Context(), &g); err != nil {
		serverError(w, err)
		return
	}
	api.app.Hub.Publish(app.TopicGoals)
	writeJSON(w, g)
}

func (api *API) updateGoal(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "ungültige ID")
		return
	}
	g, err := api.app.Store.GetGoal(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	if !decode(w, r, &g) || !api.validGoal(w, r, &g) {
		return
	}
	g.ID = id
	if err := api.app.Store.UpdateGoal(r.Context(), &g); err != nil {
		storeError(w, err)
		return
	}
	api.app.Hub.Publish(app.TopicGoals)
	writeJSON(w, g)
}

func (api *API) deleteGoal(w http.ResponseWriter, r *http.Request) {
	api.deleteByID(w, r, api.app.Store.DeleteGoal, app.TopicGoals)
}

func (api *API) validGoal(w http.ResponseWriter, r *http.Request, g *store.Goal) bool {
	g.Title = strings.TrimSpace(g.Title)
	if g.TargetDate != nil && *g.TargetDate == "" {
		g.TargetDate = nil
	}
	switch {
	case !validText(g.Title, 120):
		httpError(w, http.StatusBadRequest, "Titel fehlt oder ist zu lang")
	case utf8.RuneCountInString(g.Description) > 1000:
		httpError(w, http.StatusBadRequest, "Beschreibung ist zu lang")
	case g.TargetDate != nil && !reDate.MatchString(*g.TargetDate):
		httpError(w, http.StatusBadRequest, "Zieldatum muss im Format YYYY-MM-DD sein")
	case g.MemberID == 0:
		httpError(w, http.StatusBadRequest, "Ein Ziel gehört immer zu einem Familienmitglied")
	case !api.memberExists(w, r, &g.MemberID):
	default:
		return true
	}
	return false
}

// ---- Events ----

func (api *API) listEvents(w http.ResponseWriter, r *http.Request) {
	from, to := api.app.Today(), api.app.Today().AddDate(0, 0, 31)
	if v, err := time.Parse(time.RFC3339, r.URL.Query().Get("from")); err == nil {
		from = v
	}
	if v, err := time.Parse(time.RFC3339, r.URL.Query().Get("to")); err == nil && v.After(from) && v.Sub(from) <= 400*24*time.Hour {
		to = v
	}
	events, err := api.app.Events(r.Context(), from, to)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, events)
}

func (api *API) createEvent(w http.ResponseWriter, r *http.Request) {
	var e store.Event
	if !decode(w, r, &e) || !api.validEvent(w, r, &e) {
		return
	}
	if err := api.app.Store.CreateEvent(r.Context(), &e); err != nil {
		serverError(w, err)
		return
	}
	api.app.Hub.Publish(app.TopicEvents)
	writeJSON(w, e)
}

func (api *API) updateEvent(w http.ResponseWriter, r *http.Request) {
	var e store.Event
	id, err := pathID(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "ungültige ID")
		return
	}
	if !decode(w, r, &e) || !api.validEvent(w, r, &e) {
		return
	}
	e.ID = id
	if err := api.app.Store.UpdateEvent(r.Context(), &e); err != nil {
		storeError(w, err)
		return
	}
	api.app.Hub.Publish(app.TopicEvents)
	writeJSON(w, e)
}

func (api *API) deleteEvent(w http.ResponseWriter, r *http.Request) {
	api.deleteByID(w, r, api.app.Store.DeleteEvent, app.TopicEvents)
}

func (api *API) validEvent(w http.ResponseWriter, r *http.Request, e *store.Event) bool {
	e.Title = strings.TrimSpace(e.Title)
	if e.AllDay {
		// Ganztagig: auf Mitternacht der Familien-Zeitzone normalisieren, Ende exklusiv.
		s, en := e.Start.In(api.app.Loc), e.End.In(api.app.Loc)
		e.Start = time.Date(s.Year(), s.Month(), s.Day(), 0, 0, 0, 0, api.app.Loc)
		e.End = time.Date(en.Year(), en.Month(), en.Day(), 0, 0, 0, 0, api.app.Loc)
		if !e.End.After(e.Start) {
			e.End = e.Start.AddDate(0, 0, 1)
		}
	}
	switch {
	case !validText(e.Title, 200):
		httpError(w, http.StatusBadRequest, "Titel fehlt oder ist zu lang")
	case e.Start.IsZero() || e.End.IsZero():
		httpError(w, http.StatusBadRequest, "Start und Ende sind Pflicht")
	case e.End.Before(e.Start):
		httpError(w, http.StatusBadRequest, "Ende liegt vor dem Start")
	case utf8.RuneCountInString(e.Location) > 200 || utf8.RuneCountInString(e.Notes) > 2000:
		httpError(w, http.StatusBadRequest, "Ort oder Notiz ist zu lang")
	case !api.memberExists(w, r, e.MemberID):
	default:
		return true
	}
	return false
}

// ---- Kalender-Feeds ----

func (api *API) calendarStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, api.app.Importer.Status())
}

func (api *API) refreshCalendars(w http.ResponseWriter, r *http.Request) {
	api.app.RefreshCalendars(r.Context())
	writeJSON(w, api.app.Importer.Status())
}

// ---- Gemeinsame Helfer ----

func (api *API) deleteByID(w http.ResponseWriter, r *http.Request, del func(context.Context, int64) error, topics ...string) {
	id, err := pathID(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "ungültige ID")
		return
	}
	if err := del(r.Context(), id); err != nil {
		storeError(w, err)
		return
	}
	for _, t := range topics {
		api.app.Hub.Publish(t)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) memberExists(w http.ResponseWriter, r *http.Request, id *int64) bool {
	if id == nil {
		return true
	}
	members, err := api.app.Store.ListMembers(r.Context())
	if err != nil {
		serverError(w, err)
		return false
	}
	for _, m := range members {
		if m.ID == *id {
			return true
		}
	}
	httpError(w, http.StatusBadRequest, "Familienmitglied existiert nicht")
	return false
}

func validText(s string, maxRunes int) bool {
	return s != "" && utf8.RuneCountInString(s) <= maxRunes
}
