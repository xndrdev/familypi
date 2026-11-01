package discord

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"familydisplay/internal/app"
	"familydisplay/internal/calendar"
	"familydisplay/internal/hub"
	"familydisplay/internal/store"
)

func newTestBot(t *testing.T) (*Bot, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	loc, _ := time.LoadLocation("Europe/Berlin")
	a := &app.App{Store: st, Hub: hub.New(), Importer: calendar.NewImporter(loc), Loc: loc}
	ctx := context.Background()
	for _, m := range []store.Member{{Name: "Alex", Emoji: "🧔", DiscordUserID: "42"}, {Name: "Mia", Emoji: "🦄"}} {
		if err := st.CreateMember(ctx, &m); err != nil {
			t.Fatal(err)
		}
	}
	return &Bot{app: a}, st
}

func opt(name string, value any) *discordgo.ApplicationCommandInteractionDataOption {
	typ := discordgo.ApplicationCommandOptionString
	if _, ok := value.(float64); ok {
		typ = discordgo.ApplicationCommandOptionInteger
	}
	return &discordgo.ApplicationCommandInteractionDataOption{Name: name, Type: typ, Value: value}
}

// interaction baut einen Slash-Command wie "/todo neu text:..." von Discord-User 42.
func interaction(cmd, sub string, opts ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	data := discordgo.ApplicationCommandInteractionData{Name: cmd, Options: opts}
	if sub != "" {
		data.Options = []*discordgo.ApplicationCommandInteractionDataOption{{Name: sub, Type: discordgo.ApplicationCommandOptionSubCommand, Options: opts}}
	}
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:   discordgo.InteractionApplicationCommand,
		Data:   data,
		Member: &discordgo.Member{User: &discordgo.User{ID: "42", Username: "alex"}},
	}}
}

func run(t *testing.T, b *Bot, i *discordgo.InteractionCreate) string {
	t.Helper()
	msg, err := b.handleCommand(context.Background(), i)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	return msg
}

func TestTodoFlow(t *testing.T) {
	b, st := newTestBot(t)
	ctx := context.Background()

	msg := run(t, b, interaction("todo", "neu", opt("text", "Milch kaufen"), opt("fällig", "morgen")))
	if !strings.Contains(msg, "Familie") || !strings.Contains(msg, "Milch kaufen") {
		t.Errorf("unexpected reply: %s", msg)
	}
	run(t, b, interaction("todo", "neu", opt("text", "Zimmer aufräumen"), opt("für", "mia")))

	todos, _ := st.ListTodos(ctx, false)
	if len(todos) != 2 || todos[0].Due == nil || todos[0].CreatedBy != "alex" {
		t.Fatalf("todos not stored as expected: %+v", todos)
	}
	if todos[1].MemberID == nil || *todos[1].MemberID != 2 {
		t.Errorf("member not resolved by name: %+v", todos[1])
	}

	list := run(t, b, interaction("todo", "liste", opt("für", "2")))
	if !strings.Contains(list, "Zimmer") || strings.Contains(list, "Milch") {
		t.Errorf("filter by member broken: %s", list)
	}

	run(t, b, interaction("todo", "erledigt", opt("todo", "1")))
	done, _ := st.GetTodo(ctx, 1)
	if !done.Done || done.DoneAt == nil {
		t.Errorf("todo not marked done: %+v", done)
	}

	if _, err := b.handleCommand(ctx, interaction("todo", "neu", opt("text", "x"), opt("für", "Oma"))); err == nil {
		t.Error("unknown member should fail")
	}
}

func TestEventAndGoalFlow(t *testing.T) {
	b, st := newTestBot(t)
	ctx := context.Background()

	msg := run(t, b, interaction("termin", "neu", opt("titel", "Turnen"), opt("datum", "morgen"), opt("uhrzeit", "16:30"), opt("dauer", 90.0), opt("für", "Mia")))
	if !strings.Contains(msg, "16:30") {
		t.Errorf("unexpected reply: %s", msg)
	}
	run(t, b, interaction("termin", "neu", opt("titel", "Ausflug"), opt("datum", "übermorgen")))
	from := b.app.Today()
	events, _ := st.ListEvents(ctx, from, from.AddDate(0, 0, 7))
	if len(events) != 2 || events[0].End.Sub(events[0].Start) != 90*time.Minute || !events[1].AllDay {
		t.Fatalf("events not stored as expected: %+v", events)
	}

	// Ohne "für" gilt der Absender (Discord-ID 42 = Alex).
	run(t, b, interaction("ziel", "neu", opt("titel", "Halbmarathon")))
	run(t, b, interaction("ziel", "neu", opt("titel", "Klimmzüge")))
	run(t, b, interaction("ziel", "fortschritt", opt("prozent", 40.0)))
	next, _ := st.NextGoals(ctx)
	if g := next[1]; g.Title != "Halbmarathon" || g.Progress != 40 {
		t.Fatalf("current goal wrong: %+v", g)
	}
	run(t, b, interaction("ziel", "erreicht"))
	next, _ = st.NextGoals(ctx)
	if g := next[1]; g.Title != "Klimmzüge" {
		t.Errorf("after reaching a goal the next one should move up, got %+v", g)
	}

	if _, err := b.handleCommand(ctx, interaction("ziel", "fortschritt", opt("prozent", 10.0), opt("für", "Mia"))); err == nil {
		t.Error("member without goal should fail")
	}
	if out := run(t, b, interaction("heute", "")); out == "" {
		t.Error("/heute returned nothing")
	}
}
