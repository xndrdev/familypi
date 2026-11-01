// Package api stellt die REST-API, den Live-Stream (SSE) und den ICS-Feed bereit.
package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"familydisplay/internal/app"
	"familydisplay/internal/calendar"
	"familydisplay/internal/store"
)

type Config struct {
	AdminToken string // leer = Schreibzugriff ohne Login (nur im vertrauenswuerdigen Heimnetz)
	ICSToken   string // leer = ICS-Feed ohne Token abrufbar
}

type API struct {
	app *app.App
	cfg Config
}

func New(a *app.App, cfg Config) *API { return &API{app: a, cfg: cfg} }

func (api *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/auth", api.handleAuth)
	mux.HandleFunc("GET /api/dashboard", api.handleDashboard)
	mux.HandleFunc("GET /api/stream", api.handleStream)
	mux.HandleFunc("GET /api/weather", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, api.app.Weather.Report()) })
	mux.HandleFunc("GET /calendar.ics", api.handleICS)

	mux.HandleFunc("GET /api/members", api.listMembers)
	mux.HandleFunc("POST /api/members", api.write(api.createMember))
	mux.HandleFunc("PUT /api/members/{id}", api.write(api.updateMember))
	mux.HandleFunc("DELETE /api/members/{id}", api.write(api.deleteMember))

	mux.HandleFunc("GET /api/todos", api.listTodos)
	mux.HandleFunc("POST /api/todos", api.write(api.createTodo))
	mux.HandleFunc("PUT /api/todos/{id}", api.write(api.updateTodo))
	mux.HandleFunc("DELETE /api/todos/{id}", api.write(api.deleteTodo))

	mux.HandleFunc("GET /api/goals", api.listGoals)
	mux.HandleFunc("POST /api/goals", api.write(api.createGoal))
	mux.HandleFunc("PUT /api/goals/{id}", api.write(api.updateGoal))
	mux.HandleFunc("DELETE /api/goals/{id}", api.write(api.deleteGoal))

	mux.HandleFunc("GET /api/events", api.listEvents)
	mux.HandleFunc("POST /api/events", api.write(api.createEvent))
	mux.HandleFunc("PUT /api/events/{id}", api.write(api.updateEvent))
	mux.HandleFunc("DELETE /api/events/{id}", api.write(api.deleteEvent))

	mux.HandleFunc("GET /api/calendars", api.write(api.calendarStatus))
	mux.HandleFunc("POST /api/calendars/refresh", api.write(api.refreshCalendars))
}

// ---- Auth ----

func (api *API) authorized(r *http.Request) bool {
	if api.cfg.AdminToken == "" {
		return true
	}
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	ok := subtle.ConstantTimeCompare([]byte(token), []byte(api.cfg.AdminToken)) == 1
	if !ok && token != "" {
		// Falsche Passwoerter bremsen, damit Durchprobieren im WLAN unattraktiv wird.
		time.Sleep(time.Second)
	}
	return ok
}

func (api *API) write(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !api.authorized(r) {
			httpError(w, http.StatusUnauthorized, "Anmeldung erforderlich")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		h(w, r)
	}
}

func (api *API) handleAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]bool{"required": api.cfg.AdminToken != "", "valid": api.authorized(r)})
}

// ---- Dashboard ----

type MemberGoal struct {
	Member store.Member `json:"member"`
	Goal   *store.Goal  `json:"goal"`
}

type Dashboard struct {
	Now     time.Time      `json:"now"`
	Members []store.Member `json:"members"`
	Events  []store.Event  `json:"events"`
	Todos   []store.Todo   `json:"todos"`
	Goals   []MemberGoal   `json:"goals"`
}

func (api *API) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	days := 14
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 && d <= 60 {
		days = d
	}
	members, err := api.app.Store.ListMembers(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	from := api.app.Today()
	events, err := api.app.Events(ctx, from, from.AddDate(0, 0, days))
	if err != nil {
		serverError(w, err)
		return
	}
	todos, err := api.app.Store.ListTodos(ctx, false)
	if err != nil {
		serverError(w, err)
		return
	}
	next, err := api.app.Store.NextGoals(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	goals := make([]MemberGoal, 0, len(members))
	for _, m := range members {
		mg := MemberGoal{Member: publicMember(m)}
		if g, ok := next[m.ID]; ok {
			mg.Goal = &g
		}
		goals = append(goals, mg)
	}
	for i := range members {
		members[i] = publicMember(members[i])
	}
	writeJSON(w, Dashboard{Now: time.Now().In(api.app.Loc), Members: members, Events: events, Todos: todos, Goals: goals})
}

// publicMember blendet Felder aus, die das Display nicht braucht und die nicht im Netz herumliegen sollen.
func publicMember(m store.Member) store.Member {
	m.ICalURL, m.DiscordUserID = "", ""
	return m
}

// ---- Live-Stream ----

func (api *API) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := api.app.Hub.Subscribe()
	defer api.app.Hub.Unsubscribe(ch)

	fmt.Fprint(w, "retry: 3000\nevent: hello\ndata: {}\n\n")
	flusher.Flush()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case topic := <-ch:
			fmt.Fprintf(w, "event: change\ndata: %q\n\n", topic)
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
		}
		flusher.Flush()
	}
}

// ---- ICS-Feed ----

func (api *API) handleICS(w http.ResponseWriter, r *http.Request) {
	if api.cfg.ICSToken != "" && subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("token")), []byte(api.cfg.ICSToken)) != 1 {
		httpError(w, http.StatusUnauthorized, "token fehlt oder ist falsch")
		return
	}
	ctx := r.Context()
	members, err := api.app.Store.ListMembers(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	byID := make(map[int64]store.Member, len(members))
	for _, m := range members {
		byID[m.ID] = m
	}
	from := api.app.Today().AddDate(0, -1, 0)
	// Nur lokale Termine exportieren: importierte Kalender haben die Mitglieder bereits auf ihren Geraeten.
	events, err := api.app.Store.ListEvents(ctx, from, from.AddDate(1, 0, 0))
	if err != nil {
		serverError(w, err)
		return
	}
	var sb strings.Builder
	calendar.WriteICS(&sb, events, byID, api.app.Loc)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Write([]byte(sb.String()))
}

// ---- Helpers ----

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		httpError(w, http.StatusBadRequest, "ungültiges JSON: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write json", "err", err)
	}
}

func httpError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func serverError(w http.ResponseWriter, err error) {
	slog.Error("request failed", "err", err)
	httpError(w, http.StatusInternalServerError, "interner Fehler")
}

// storeError uebersetzt Store-Fehler in HTTP-Antworten.
func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		httpError(w, http.StatusNotFound, "nicht gefunden")
		return
	}
	serverError(w, err)
}
