// Package store kapselt die SQLite-Persistenz fuer Mitglieder, Todos, Ziele und Termine.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type Member struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Color         string `json:"color"`
	Emoji         string `json:"emoji"`
	DiscordUserID string `json:"discordUserId"`
	ICalURL       string `json:"icalUrl"`
	Sort          int    `json:"sort"`
}

type Todo struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	MemberID  *int64     `json:"memberId"`
	Due       *string    `json:"due"` // YYYY-MM-DD
	Done      bool       `json:"done"`
	DoneAt    *time.Time `json:"doneAt"`
	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
}

type Goal struct {
	ID          int64     `json:"id"`
	MemberID    int64     `json:"memberId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TargetDate  *string   `json:"targetDate"` // YYYY-MM-DD
	Progress    int       `json:"progress"`   // 0-100
	Done        bool      `json:"done"`
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Event struct {
	ID       int64     `json:"id"`
	UID      string    `json:"uid"`
	Title    string    `json:"title"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"allDay"`
	MemberID *int64    `json:"memberId"`
	Location string    `json:"location"`
	Notes    string    `json:"notes"`
	Source   string    `json:"source"` // "local" oder "ical"
}

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS members (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	color TEXT NOT NULL DEFAULT '#4f8cff',
	emoji TEXT NOT NULL DEFAULT '',
	discord_user_id TEXT NOT NULL DEFAULT '',
	ical_url TEXT NOT NULL DEFAULT '',
	sort INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS todos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	member_id INTEGER REFERENCES members(id) ON DELETE SET NULL,
	due TEXT,
	done INTEGER NOT NULL DEFAULT 0,
	done_at TEXT,
	created_at TEXT NOT NULL,
	created_by TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_todos_done ON todos(done);
CREATE TABLE IF NOT EXISTS goals (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	target_date TEXT,
	progress INTEGER NOT NULL DEFAULT 0,
	done INTEGER NOT NULL DEFAULT 0,
	sort INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_goals_member ON goals(member_id, done);
CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	start TEXT NOT NULL,
	end TEXT NOT NULL,
	all_day INTEGER NOT NULL DEFAULT 0,
	member_id INTEGER REFERENCES members(id) ON DELETE SET NULL,
	location TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_events_start ON events(start);
`

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// ---- Members ----

func (s *Store) ListMembers(ctx context.Context) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, color, emoji, discord_user_id, ical_url, sort FROM members ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Name, &m.Color, &m.Emoji, &m.DiscordUserID, &m.ICalURL, &m.Sort); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CreateMember(ctx context.Context, m *Member) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO members (name, color, emoji, discord_user_id, ical_url, sort) VALUES (?, ?, ?, ?, ?, ?)`,
		m.Name, m.Color, m.Emoji, m.DiscordUserID, m.ICalURL, m.Sort)
	if err != nil {
		return err
	}
	m.ID, err = res.LastInsertId()
	return err
}

func (s *Store) UpdateMember(ctx context.Context, m *Member) error {
	return expectOne(s.db.ExecContext(ctx, `UPDATE members SET name=?, color=?, emoji=?, discord_user_id=?, ical_url=?, sort=? WHERE id=?`,
		m.Name, m.Color, m.Emoji, m.DiscordUserID, m.ICalURL, m.Sort, m.ID))
}

func (s *Store) DeleteMember(ctx context.Context, id int64) error {
	return expectOne(s.db.ExecContext(ctx, `DELETE FROM members WHERE id=?`, id))
}

// ---- Todos ----

const todoCols = `id, title, member_id, due, done, done_at, created_at, created_by`

func (s *Store) ListTodos(ctx context.Context, includeDone bool) ([]Todo, error) {
	q := `SELECT ` + todoCols + ` FROM todos`
	if !includeDone {
		// Erledigte Todos bleiben noch 24h sichtbar, damit das Abhaken auf dem Display sichtbar ist.
		q += ` WHERE done = 0 OR done_at >= ?`
	}
	q += ` ORDER BY done, due IS NULL, due, id`
	var args []any
	if !includeDone {
		args = append(args, fmtTime(time.Now().Add(-24*time.Hour)))
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Todo{}
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTodo(ctx context.Context, id int64) (Todo, error) {
	t, err := scanTodo(s.db.QueryRowContext(ctx, `SELECT `+todoCols+` FROM todos WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) CreateTodo(ctx context.Context, t *Todo) error {
	t.CreatedAt = time.Now()
	res, err := s.db.ExecContext(ctx, `INSERT INTO todos (title, member_id, due, done, created_at, created_by) VALUES (?, ?, ?, 0, ?, ?)`,
		t.Title, t.MemberID, t.Due, fmtTime(t.CreatedAt), t.CreatedBy)
	if err != nil {
		return err
	}
	t.ID, err = res.LastInsertId()
	return err
}

func (s *Store) UpdateTodo(ctx context.Context, t *Todo) error {
	if t.Done && t.DoneAt == nil {
		now := time.Now()
		t.DoneAt = &now
	}
	if !t.Done {
		t.DoneAt = nil
	}
	var doneAt *string
	if t.DoneAt != nil {
		v := fmtTime(*t.DoneAt)
		doneAt = &v
	}
	return expectOne(s.db.ExecContext(ctx, `UPDATE todos SET title=?, member_id=?, due=?, done=?, done_at=? WHERE id=?`,
		t.Title, t.MemberID, t.Due, t.Done, doneAt, t.ID))
}

func (s *Store) DeleteTodo(ctx context.Context, id int64) error {
	return expectOne(s.db.ExecContext(ctx, `DELETE FROM todos WHERE id=?`, id))
}

// ---- Goals ----

const goalCols = `id, member_id, title, description, target_date, progress, done, sort, created_at`

func (s *Store) ListGoals(ctx context.Context) ([]Goal, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+goalCols+` FROM goals ORDER BY member_id, done, sort, target_date IS NULL, target_date, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Goal{}
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) GetGoal(ctx context.Context, id int64) (Goal, error) {
	g, err := scanGoal(s.db.QueryRowContext(ctx, `SELECT `+goalCols+` FROM goals WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

// NextGoals liefert pro Mitglied das naechste offene Ziel (niedrigste Sortierung, dann fruehestes Zieldatum).
func (s *Store) NextGoals(ctx context.Context) (map[int64]Goal, error) {
	goals, err := s.ListGoals(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]Goal{}
	for _, g := range goals {
		if g.Done {
			continue
		}
		if _, ok := out[g.MemberID]; !ok {
			out[g.MemberID] = g
		}
	}
	return out, nil
}

func (s *Store) CreateGoal(ctx context.Context, g *Goal) error {
	g.CreatedAt = time.Now()
	res, err := s.db.ExecContext(ctx, `INSERT INTO goals (member_id, title, description, target_date, progress, done, sort, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		g.MemberID, g.Title, g.Description, g.TargetDate, clampProgress(g.Progress), g.Done, g.Sort, fmtTime(g.CreatedAt))
	if err != nil {
		return err
	}
	g.ID, err = res.LastInsertId()
	return err
}

func (s *Store) UpdateGoal(ctx context.Context, g *Goal) error {
	g.Progress = clampProgress(g.Progress)
	return expectOne(s.db.ExecContext(ctx, `UPDATE goals SET member_id=?, title=?, description=?, target_date=?, progress=?, done=?, sort=? WHERE id=?`,
		g.MemberID, g.Title, g.Description, g.TargetDate, g.Progress, g.Done, g.Sort, g.ID))
}

func (s *Store) DeleteGoal(ctx context.Context, id int64) error {
	return expectOne(s.db.ExecContext(ctx, `DELETE FROM goals WHERE id=?`, id))
}

// ---- Events ----

const eventCols = `id, title, start, end, all_day, member_id, location, notes`

// ListEvents liefert lokale Termine, die das Intervall [from, to) beruehren.
func (s *Store) ListEvents(ctx context.Context, from, to time.Time) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+eventCols+` FROM events WHERE end > ? AND start < ? ORDER BY start`,
		fmtTime(from), fmtTime(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateEvent(ctx context.Context, e *Event) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO events (title, start, end, all_day, member_id, location, notes) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.Title, fmtTime(e.Start), fmtTime(e.End), e.AllDay, e.MemberID, e.Location, e.Notes)
	if err != nil {
		return err
	}
	e.ID, err = res.LastInsertId()
	e.Source = "local"
	e.UID = fmt.Sprintf("local-%d", e.ID)
	return err
}

func (s *Store) UpdateEvent(ctx context.Context, e *Event) error {
	return expectOne(s.db.ExecContext(ctx, `UPDATE events SET title=?, start=?, end=?, all_day=?, member_id=?, location=?, notes=? WHERE id=?`,
		e.Title, fmtTime(e.Start), fmtTime(e.End), e.AllDay, e.MemberID, e.Location, e.Notes, e.ID))
}

func (s *Store) DeleteEvent(ctx context.Context, id int64) error {
	return expectOne(s.db.ExecContext(ctx, `DELETE FROM events WHERE id=?`, id))
}

// ---- Helpers ----

type scanner interface{ Scan(dest ...any) error }

func scanTodo(r scanner) (Todo, error) {
	var t Todo
	var doneAt sql.NullString
	var createdAt string
	if err := r.Scan(&t.ID, &t.Title, &t.MemberID, &t.Due, &t.Done, &doneAt, &createdAt, &t.CreatedBy); err != nil {
		return t, err
	}
	t.CreatedAt = parseTime(createdAt)
	if doneAt.Valid {
		v := parseTime(doneAt.String)
		t.DoneAt = &v
	}
	return t, nil
}

func scanGoal(r scanner) (Goal, error) {
	var g Goal
	var createdAt string
	if err := r.Scan(&g.ID, &g.MemberID, &g.Title, &g.Description, &g.TargetDate, &g.Progress, &g.Done, &g.Sort, &createdAt); err != nil {
		return g, err
	}
	g.CreatedAt = parseTime(createdAt)
	return g, nil
}

func scanEvent(r scanner) (Event, error) {
	var e Event
	var start, end string
	if err := r.Scan(&e.ID, &e.Title, &start, &end, &e.AllDay, &e.MemberID, &e.Location, &e.Notes); err != nil {
		return e, err
	}
	e.Start, e.End = parseTime(start), parseTime(end)
	e.Source = "local"
	e.UID = fmt.Sprintf("local-%d", e.ID)
	return e, nil
}

// Zeiten werden in UTC gespeichert, damit String-Vergleiche in SQL korrekt sortieren.
func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func clampProgress(p int) int { return max(0, min(100, p)) }

func expectOne(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
