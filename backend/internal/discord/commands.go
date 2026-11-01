package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"familydisplay/internal/app"
	"familydisplay/internal/dateparse"
	"familydisplay/internal/store"
)

func commands() []*discordgo.ApplicationCommand {
	str := func(name, desc string, required, autocomplete bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: name, Description: desc, Required: required, Autocomplete: autocomplete}
	}
	sub := func(name, desc string, opts ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: desc, Options: opts}
	}
	minPct, maxPct := 0.0, 100.0
	minDays := 1.0

	return []*discordgo.ApplicationCommand{
		{
			Name: "todo", Description: "Todos verwalten",
			Options: []*discordgo.ApplicationCommandOption{
				sub("neu", "Neues Todo anlegen",
					str("text", "Was ist zu tun?", true, false),
					str("für", "Für wen? (leer = ganze Familie)", false, true),
					str("fällig", "Bis wann? z.B. morgen, freitag, 24.12.", false, false)),
				sub("erledigt", "Todo abhaken", str("todo", "Welches Todo?", true, true)),
				sub("löschen", "Todo löschen", str("todo", "Welches Todo?", true, true)),
				sub("liste", "Offene Todos anzeigen", str("für", "Nur für ...", false, true)),
			},
		},
		{
			Name: "termin", Description: "Termine im Familienkalender",
			Options: []*discordgo.ApplicationCommandOption{
				sub("neu", "Neuen Termin eintragen",
					str("titel", "Was steht an?", true, false),
					str("datum", "Wann? z.B. morgen, freitag, 24.12.", true, false),
					str("uhrzeit", "Uhrzeit, z.B. 14:30 (leer = ganztägig)", false, false),
					&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionInteger, Name: "dauer", Description: "Dauer in Minuten (Standard 60)", MinValue: &minDays},
					str("für", "Für wen? (leer = ganze Familie)", false, true),
					str("ort", "Wo?", false, false)),
				sub("liste", "Kommende Termine anzeigen",
					&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionInteger, Name: "tage", Description: "Wie viele Tage voraus (Standard 7)", MinValue: &minDays}),
			},
		},
		{
			Name: "ziel", Description: "Große Ziele der Familienmitglieder",
			Options: []*discordgo.ApplicationCommandOption{
				sub("neu", "Neues Ziel setzen",
					str("titel", "Was ist das Ziel?", true, false),
					str("für", "Für wen? (leer = du selbst)", false, true),
					str("bis", "Bis wann? z.B. 1.6.", false, false),
					str("beschreibung", "Ein paar Worte dazu", false, false)),
				sub("fortschritt", "Fortschritt des aktuellen Ziels setzen",
					&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionInteger, Name: "prozent", Description: "0 bis 100", Required: true, MinValue: &minPct, MaxValue: maxPct},
					str("für", "Für wen? (leer = du selbst)", false, true)),
				sub("erreicht", "Aktuelles Ziel als erreicht markieren 🎉", str("für", "Für wen? (leer = du selbst)", false, true)),
				sub("liste", "Aktuelle Ziele aller Familienmitglieder"),
			},
		},
		{Name: "heute", Description: "Was steht heute an?"},
	}
}

func (b *Bot) handleCommand(ctx context.Context, i *discordgo.InteractionCreate) (string, error) {
	path, o := commandPath(i.ApplicationCommandData())
	switch path {
	case "todo neu":
		return b.todoNew(ctx, i, o)
	case "todo erledigt":
		return b.todoDone(ctx, o)
	case "todo löschen":
		return b.todoDelete(ctx, o)
	case "todo liste":
		return b.todoList(ctx, i, o)
	case "termin neu":
		return b.eventNew(ctx, i, o)
	case "termin liste":
		return b.eventList(ctx, o.int("tage", 7))
	case "ziel neu":
		return b.goalNew(ctx, i, o)
	case "ziel fortschritt":
		return b.goalUpdate(ctx, i, o, func(g *store.Goal) { g.Progress = o.int("prozent", 0) })
	case "ziel erreicht":
		return b.goalUpdate(ctx, i, o, func(g *store.Goal) { g.Done, g.Progress = true, 100 })
	case "ziel liste":
		return b.goalList(ctx)
	case "heute":
		return b.today(ctx)
	}
	return "", fmt.Errorf("unbekannter Befehl %q", path)
}

// ---- Todos ----

func (b *Bot) todoNew(ctx context.Context, i *discordgo.InteractionCreate, o options) (string, error) {
	member, err := b.resolveMember(ctx, i, o.str("für"), false)
	if err != nil {
		return "", err
	}
	t := store.Todo{Title: truncate(o.str("text"), 200), MemberID: memberIDPtr(member), CreatedBy: displayName(i)}
	if v := o.str("fällig"); v != "" {
		d, err := dateparse.Date(v, time.Now(), b.app.Loc)
		if err != nil {
			return "", err
		}
		s := d.Format("2006-01-02")
		t.Due = &s
	}
	if err := b.app.Store.CreateTodo(ctx, &t); err != nil {
		return "", err
	}
	b.app.Hub.Publish(app.TopicTodos)
	msg := fmt.Sprintf("✅ Todo für **%s**: %s", memberLabel(member), t.Title)
	if t.Due != nil {
		msg += " (fällig " + b.formatDate(*t.Due) + ")"
	}
	return msg, nil
}

func (b *Bot) findTodo(ctx context.Context, o options) (store.Todo, error) {
	id, err := strconv.ParseInt(o.str("todo"), 10, 64)
	if err != nil {
		return store.Todo{}, fmt.Errorf("bitte ein Todo aus der Vorschlagsliste auswählen")
	}
	t, err := b.app.Store.GetTodo(ctx, id)
	if err != nil {
		return t, fmt.Errorf("Todo nicht gefunden")
	}
	return t, nil
}

func (b *Bot) todoDone(ctx context.Context, o options) (string, error) {
	t, err := b.findTodo(ctx, o)
	if err != nil {
		return "", err
	}
	t.Done = true
	if err := b.app.Store.UpdateTodo(ctx, &t); err != nil {
		return "", err
	}
	b.app.Hub.Publish(app.TopicTodos)
	return "☑️ Erledigt: ~~" + t.Title + "~~", nil
}

func (b *Bot) todoDelete(ctx context.Context, o options) (string, error) {
	t, err := b.findTodo(ctx, o)
	if err != nil {
		return "", err
	}
	if err := b.app.Store.DeleteTodo(ctx, t.ID); err != nil {
		return "", err
	}
	b.app.Hub.Publish(app.TopicTodos)
	return "🗑️ Gelöscht: " + t.Title, nil
}

func (b *Bot) todoList(ctx context.Context, i *discordgo.InteractionCreate, o options) (string, error) {
	filter := o.str("für")
	var member *store.Member
	if filter != "" {
		var err error
		if member, err = b.resolveMember(ctx, i, filter, false); err != nil {
			return "", err
		}
	}
	todos, err := b.app.Store.ListTodos(ctx, false)
	if err != nil {
		return "", err
	}
	names, err := b.memberNames(ctx)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, t := range todos {
		if t.Done || (filter != "" && !sameMember(t.MemberID, member)) {
			continue
		}
		fmt.Fprintf(&sb, "• %s — %s", t.Title, names.label(t.MemberID))
		if t.Due != nil {
			sb.WriteString(" (fällig " + b.formatDate(*t.Due) + ")")
		}
		sb.WriteString("\n")
	}
	if sb.Len() == 0 {
		return "🎉 Keine offenen Todos.", nil
	}
	return "**Offene Todos**\n" + sb.String(), nil
}

func sameMember(id *int64, m *store.Member) bool {
	if m == nil {
		return id == nil
	}
	return id != nil && *id == m.ID
}

// ---- Termine ----

func (b *Bot) eventNew(ctx context.Context, i *discordgo.InteractionCreate, o options) (string, error) {
	day, err := dateparse.Date(o.str("datum"), time.Now(), b.app.Loc)
	if err != nil {
		return "", err
	}
	member, err := b.resolveMember(ctx, i, o.str("für"), false)
	if err != nil {
		return "", err
	}
	e := store.Event{Title: truncate(o.str("titel"), 200), MemberID: memberIDPtr(member), Location: truncate(o.str("ort"), 200)}
	if clock := o.str("uhrzeit"); clock != "" {
		h, m, err := dateparse.Clock(clock)
		if err != nil {
			return "", err
		}
		e.Start = time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, b.app.Loc)
		e.End = e.Start.Add(time.Duration(o.int("dauer", 60)) * time.Minute)
	} else {
		e.AllDay, e.Start, e.End = true, day, day.AddDate(0, 0, 1)
	}
	if err := b.app.Store.CreateEvent(ctx, &e); err != nil {
		return "", err
	}
	b.app.Hub.Publish(app.TopicEvents)
	return fmt.Sprintf("📅 Eingetragen für **%s**: %s — %s", memberLabel(member), e.Title, b.formatEventTime(e)), nil
}

func (b *Bot) eventList(ctx context.Context, days int) (string, error) {
	days = min(days, 31)
	from := b.app.Today()
	events, err := b.app.Events(ctx, from, from.AddDate(0, 0, days))
	if err != nil {
		return "", err
	}
	if len(events) == 0 {
		return fmt.Sprintf("Keine Termine in den nächsten %d Tagen.", days), nil
	}
	names, err := b.memberNames(ctx)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "**Termine der nächsten %d Tage**\n", days)
	for _, e := range events {
		line := fmt.Sprintf("• %s — %s (%s)\n", b.formatEventTime(e), e.Title, names.label(e.MemberID))
		if sb.Len()+len(line) > 1900 { // Discord-Limit: 2000 Zeichen
			sb.WriteString("…")
			break
		}
		sb.WriteString(line)
	}
	return sb.String(), nil
}

// ---- Ziele ----

func (b *Bot) goalNew(ctx context.Context, i *discordgo.InteractionCreate, o options) (string, error) {
	member, err := b.resolveMember(ctx, i, o.str("für"), true)
	if err != nil {
		return "", err
	}
	if member == nil {
		return "", fmt.Errorf("ein Ziel gehört immer zu einem Familienmitglied")
	}
	g := store.Goal{MemberID: member.ID, Title: truncate(o.str("titel"), 120), Description: truncate(o.str("beschreibung"), 1000)}
	if v := o.str("bis"); v != "" {
		d, err := dateparse.Date(v, time.Now(), b.app.Loc)
		if err != nil {
			return "", err
		}
		s := d.Format("2006-01-02")
		g.TargetDate = &s
	}
	// Neue Ziele landen hinter den bestehenden offenen Zielen, das aktuelle Ziel bleibt vorne.
	goals, err := b.app.Store.ListGoals(ctx)
	if err != nil {
		return "", err
	}
	for _, existing := range goals {
		if existing.MemberID == member.ID && !existing.Done {
			g.Sort = max(g.Sort, existing.Sort+1)
		}
	}
	if err := b.app.Store.CreateGoal(ctx, &g); err != nil {
		return "", err
	}
	b.app.Hub.Publish(app.TopicGoals)
	return fmt.Sprintf("🎯 Neues Ziel für **%s**: %s", memberLabel(member), g.Title), nil
}

func (b *Bot) goalUpdate(ctx context.Context, i *discordgo.InteractionCreate, o options, change func(*store.Goal)) (string, error) {
	member, err := b.resolveMember(ctx, i, o.str("für"), true)
	if err != nil {
		return "", err
	}
	if member == nil {
		return "", fmt.Errorf("bitte ein Familienmitglied angeben")
	}
	next, err := b.app.Store.NextGoals(ctx)
	if err != nil {
		return "", err
	}
	g, ok := next[member.ID]
	if !ok {
		return "", fmt.Errorf("%s hat gerade kein offenes Ziel", member.Name)
	}
	change(&g)
	if err := b.app.Store.UpdateGoal(ctx, &g); err != nil {
		return "", err
	}
	b.app.Hub.Publish(app.TopicGoals)
	if g.Done {
		return fmt.Sprintf("🎉 **%s** hat das Ziel erreicht: %s", memberLabel(member), g.Title), nil
	}
	return fmt.Sprintf("📈 %s — %s: %d %%", memberLabel(member), g.Title, g.Progress), nil
}

func (b *Bot) goalList(ctx context.Context) (string, error) {
	members, err := b.app.Store.ListMembers(ctx)
	if err != nil {
		return "", err
	}
	next, err := b.app.Store.NextGoals(ctx)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("**Aktuelle Ziele**\n")
	for _, m := range members {
		if g, ok := next[m.ID]; ok {
			fmt.Fprintf(&sb, "• %s: %s (%d %%)\n", memberLabel(&m), g.Title, g.Progress)
		} else {
			fmt.Fprintf(&sb, "• %s: –\n", memberLabel(&m))
		}
	}
	return sb.String(), nil
}

// ---- Heute ----

func (b *Bot) today(ctx context.Context) (string, error) {
	events, err := b.eventList(ctx, 1)
	if err != nil {
		return "", err
	}
	todos, err := b.app.Store.ListTodos(ctx, false)
	if err != nil {
		return "", err
	}
	today := b.app.Today().Format("2006-01-02")
	var due []string
	for _, t := range todos {
		if !t.Done && t.Due != nil && *t.Due <= today {
			due = append(due, "• "+t.Title)
		}
	}
	msg := events
	if b.app.Weather != nil {
		if d := b.app.Weather.Report().Day; d != nil {
			msg = fmt.Sprintf("%s **Wetter %s:** %s, %.0f–%.0f °C, Regen %d %%\n%s\n\n",
				d.Icon, strings.ToLower(d.Label), d.Text, d.Min, d.Max, d.PrecipProb, strings.Join(d.Advice, " · ")) + msg
		}
	}
	if len(due) > 0 {
		msg += "\n**Heute fällig**\n" + strings.Join(due, "\n")
	}
	return msg, nil
}

// ---- Formatierung ----

type memberNames map[int64]string

func (b *Bot) memberNames(ctx context.Context) (memberNames, error) {
	members, err := b.app.Store.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	out := memberNames{}
	for _, m := range members {
		out[m.ID] = memberLabel(&m)
	}
	return out, nil
}

func (n memberNames) label(id *int64) string {
	if id == nil {
		return "Familie"
	}
	if name, ok := n[*id]; ok {
		return name
	}
	return "?"
}

var weekdayShort = [...]string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}

func (b *Bot) formatDay(t time.Time) string {
	t = t.In(b.app.Loc)
	return fmt.Sprintf("%s %02d.%02d.", weekdayShort[t.Weekday()], t.Day(), t.Month())
}

func (b *Bot) formatDate(isoDate string) string {
	d, err := time.ParseInLocation("2006-01-02", isoDate, b.app.Loc)
	if err != nil {
		return isoDate
	}
	return b.formatDay(d)
}

func (b *Bot) formatEventTime(e store.Event) string {
	if e.AllDay {
		return b.formatDay(e.Start) + " ganztägig"
	}
	return b.formatDay(e.Start) + " " + e.Start.In(b.app.Loc).Format("15:04")
}
