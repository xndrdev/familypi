// Package discord stellt einen Bot mit Slash-Commands bereit, ueber den die Familie Inhalte pflegt.
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"familydisplay/internal/app"
	"familydisplay/internal/store"
)

const familyChoice = "familie"

type Bot struct {
	app     *app.App
	guildID string
	session *discordgo.Session
}

func New(a *app.App, token, guildID string) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	s.Identify.Intents = discordgo.IntentsGuilds
	b := &Bot{app: a, guildID: guildID, session: s}
	s.AddHandler(b.onInteraction)
	return b, nil
}

// Run verbindet den Bot, registriert die Commands und blockiert bis ctx endet.
func (b *Bot) Run(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}
	defer b.session.Close()
	if _, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, b.guildID, commands()); err != nil {
		return fmt.Errorf("register commands: %w", err)
	}
	slog.Info("discord bot ready", "user", b.session.State.User.Username, "guild", b.guildID)
	<-ctx.Done()
	return nil
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Nur Interaktionen aus dem Familien-Server annehmen.
	if b.guildID != "" && i.GuildID != b.guildID {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch i.Type {
	case discordgo.InteractionApplicationCommandAutocomplete:
		b.respondAutocomplete(ctx, i)
	case discordgo.InteractionApplicationCommand:
		msg, err := b.handleCommand(ctx, i)
		if err != nil {
			b.reply(i, "⚠️ "+err.Error(), true)
			return
		}
		b.reply(i, msg, false)
	}
}

func (b *Bot) reply(i *discordgo.InteractionCreate, content string, ephemeral bool) {
	data := &discordgo.InteractionResponseData{Content: content}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	err := b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
	if err != nil {
		slog.Warn("discord reply failed", "err", err)
	}
}

// ---- Optionen auslesen ----

type options map[string]*discordgo.ApplicationCommandInteractionDataOption

// commandPath liefert z.B. "todo neu" und die Optionen des Subcommands.
func commandPath(data discordgo.ApplicationCommandInteractionData) (string, options) {
	path, opts := data.Name, data.Options
	if len(opts) == 1 && opts[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		path += " " + opts[0].Name
		opts = opts[0].Options
	}
	m := options{}
	for _, o := range opts {
		m[o.Name] = o
	}
	return path, m
}

func (o options) str(name string) string {
	if v, ok := o[name]; ok {
		return strings.TrimSpace(v.StringValue())
	}
	return ""
}

func (o options) int(name string, def int) int {
	if v, ok := o[name]; ok {
		return int(v.IntValue())
	}
	return def
}

func userOf(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

func displayName(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.Nick != "" {
		return i.Member.Nick
	}
	u := userOf(i)
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}

// resolveMember interpretiert die "fuer"-Option. Leer bedeutet: Default verwenden
// (der Absender selbst, falls verknuepft und selfDefault gesetzt ist, sonst die ganze Familie).
func (b *Bot) resolveMember(ctx context.Context, i *discordgo.InteractionCreate, value string, selfDefault bool) (*store.Member, error) {
	if value == "" {
		if !selfDefault {
			return nil, nil
		}
		m, err := b.app.MemberByDiscordID(ctx, userOf(i).ID)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return nil, fmt.Errorf("dein Discord-Account ist keinem Familienmitglied zugeordnet – gib `für` an oder hinterlege deine Discord-ID in der Verwaltung")
		}
		return m, nil
	}
	if strings.EqualFold(value, familyChoice) {
		return nil, nil
	}
	members, err := b.app.Store.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		if fmt.Sprint(m.ID) == value || strings.EqualFold(m.Name, value) {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("Familienmitglied %q kenne ich nicht", value)
}

func memberLabel(m *store.Member) string {
	if m == nil {
		return "Familie"
	}
	return strings.TrimSpace(m.Emoji + " " + m.Name)
}

func memberIDPtr(m *store.Member) *int64 {
	if m == nil {
		return nil
	}
	id := m.ID
	return &id
}

// ---- Autocomplete ----

func (b *Bot) respondAutocomplete(ctx context.Context, i *discordgo.InteractionCreate) {
	path, opts := commandPath(i.ApplicationCommandData())
	var focused *discordgo.ApplicationCommandInteractionDataOption
	for _, o := range opts {
		if o.Focused {
			focused = o
		}
	}
	if focused == nil {
		return
	}
	query := strings.ToLower(focused.StringValue())
	var choices []*discordgo.ApplicationCommandOptionChoice

	switch {
	case focused.Name == "für":
		if !strings.HasPrefix(path, "ziel") {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: "👪 Familie", Value: familyChoice})
		}
		members, _ := b.app.Store.ListMembers(ctx)
		for _, m := range members {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: memberLabel(&m), Value: fmt.Sprint(m.ID)})
		}
	case focused.Name == "todo":
		todos, _ := b.app.Store.ListTodos(ctx, false)
		for _, t := range todos {
			if !t.Done {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: truncate(t.Title, 90), Value: fmt.Sprint(t.ID)})
			}
		}
	}

	filtered := choices[:0]
	for _, c := range choices {
		if query == "" || strings.Contains(strings.ToLower(c.Name), query) {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) > 25 {
		filtered = filtered[:25]
	}
	err := b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: filtered},
	})
	if err != nil {
		slog.Warn("discord autocomplete failed", "err", err)
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
