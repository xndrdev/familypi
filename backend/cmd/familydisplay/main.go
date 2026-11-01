// Command familydisplay startet API, Discord-Bot, Kalender-Sync und liefert das Nuxt-Frontend aus.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata" // Zeitzonen auch auf minimalen Pi-Images verfuegbar

	"familydisplay/internal/api"
	"familydisplay/internal/app"
	"familydisplay/internal/calendar"
	"familydisplay/internal/discord"
	"familydisplay/internal/hub"
	"familydisplay/internal/store"
	"familydisplay/internal/weather"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	loc, err := time.LoadLocation(env("FD_TZ", "Europe/Berlin"))
	if err != nil {
		return err
	}
	dbPath := env("FD_DB", "data/family.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	weatherCfg, err := weatherConfig()
	if err != nil {
		return err
	}
	a := &app.App{Store: st, Hub: hub.New(), Importer: calendar.NewImporter(loc), Weather: weather.NewService(weatherCfg, loc), Loc: loc}

	refresh, err := time.ParseDuration(env("FD_CALENDAR_REFRESH", "15m"))
	if err != nil {
		return err
	}
	go a.RunCalendarSync(ctx, refresh)
	if weatherCfg.Enabled() {
		go a.Weather.RunSync(ctx, 15*time.Minute, func() { a.Hub.Publish(app.TopicWeather) })
	} else {
		slog.Info("FD_WEATHER_PLACE not set, weather disabled")
	}

	if token := os.Getenv("FD_DISCORD_TOKEN"); token != "" {
		bot, err := discord.New(a, token, os.Getenv("FD_DISCORD_GUILD_ID"))
		if err != nil {
			return err
		}
		go func() {
			if err := bot.Run(ctx); err != nil {
				slog.Error("discord bot stopped", "err", err)
			}
		}()
	} else {
		slog.Info("FD_DISCORD_TOKEN not set, discord bot disabled")
	}

	adminToken := os.Getenv("FD_ADMIN_TOKEN")
	if adminToken == "" {
		slog.Warn("FD_ADMIN_TOKEN not set: everyone in the network can change data")
	}
	mux := http.NewServeMux()
	api.New(a, api.Config{AdminToken: adminToken, ICSToken: os.Getenv("FD_ICS_TOKEN")}).Register(mux)
	mux.Handle("/", spaHandler(env("FD_STATIC_DIR", "public")))

	srv := &http.Server{Addr: env("FD_ADDR", ":8080"), Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()
	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// spaHandler liefert das statisch generierte Nuxt-Frontend aus und faellt fuer Client-Routen auf die SPA-Shell zurueck.
func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		// Fehlende Assets (z.B. alte Chunks nach einem Update) nicht mit der HTML-Shell beantworten.
		if filepath.Ext(r.URL.Path) != "" {
			http.NotFound(w, r)
			return
		}
		for _, shell := range []string{"200.html", "index.html"} {
			if _, err := os.Stat(filepath.Join(dir, shell)); err == nil {
				http.ServeFile(w, r, filepath.Join(dir, shell))
				return
			}
		}
		http.NotFound(w, r)
	})
}

// weatherConfig liest den Ort fuers Wetter: entweder FD_WEATHER_PLACE oder FD_WEATHER_LAT/FD_WEATHER_LON.
func weatherConfig() (weather.Config, error) {
	cfg := weather.Config{Place: os.Getenv("FD_WEATHER_PLACE")}
	lat, lon := os.Getenv("FD_WEATHER_LAT"), os.Getenv("FD_WEATHER_LON")
	if lat == "" && lon == "" {
		return cfg, nil
	}
	var err1, err2 error
	cfg.Lat, err1 = strconv.ParseFloat(lat, 64)
	cfg.Lon, err2 = strconv.ParseFloat(lon, 64)
	if err := errors.Join(err1, err2); err != nil {
		return cfg, fmt.Errorf("FD_WEATHER_LAT/FD_WEATHER_LON: %w", err)
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
