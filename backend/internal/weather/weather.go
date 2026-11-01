// Package weather holt die Vorhersage von Open-Meteo (kostenlos, ohne API-Key) und leitet Anzieh-Tipps ab.
package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Ab dieser Uhrzeit zeigt das Display schon die Vorhersage fuer morgen.
const tomorrowFromHour = 18

type Config struct {
	Place    string  // Ortsname, wird per Geocoding aufgeloest
	Lat, Lon float64 // alternativ direkte Koordinaten
}

func (c Config) Enabled() bool { return c.Place != "" || c.Lat != 0 || c.Lon != 0 }

type Current struct {
	Temp  float64 `json:"temp"`
	Feels float64 `json:"feels"`
	Wind  float64 `json:"wind"`
	Icon  string  `json:"icon"`
	Text  string  `json:"text"`
}

type Slot struct {
	Hour       int     `json:"hour"`
	Temp       float64 `json:"temp"`
	PrecipProb int     `json:"precipProb"`
	Icon       string  `json:"icon"`
	Text       string  `json:"text"`
}

type Day struct {
	Label      string   `json:"label"` // "Heute" oder "Morgen"
	Date       string   `json:"date"`  // YYYY-MM-DD
	Min        float64  `json:"min"`
	Max        float64  `json:"max"`
	PrecipProb int      `json:"precipProb"`
	UV         float64  `json:"uv"`
	Icon       string   `json:"icon"`
	Text       string   `json:"text"`
	Slots      []Slot   `json:"slots"`
	Advice     []string `json:"advice"`
}

type Report struct {
	Enabled   bool      `json:"enabled"`
	Place     string    `json:"place"`
	UpdatedAt time.Time `json:"updatedAt"`
	Error     string    `json:"error,omitempty"`
	Current   *Current  `json:"current"`
	Day       *Day      `json:"day"`
}

type Service struct {
	cfg    Config
	loc    *time.Location
	client *http.Client

	mu       sync.RWMutex
	place    string
	lat, lon float64
	forecast *forecast
	report   Report
}

func NewService(cfg Config, loc *time.Location) *Service {
	return &Service{
		cfg: cfg, loc: loc,
		client: &http.Client{Timeout: 15 * time.Second},
		place:  cfg.Place, lat: cfg.Lat, lon: cfg.Lon,
		report: Report{Enabled: cfg.Enabled()},
	}
}

// Report liefert den aktuellen Stand. Heute/Morgen wird bei jedem Aufruf neu bestimmt,
// damit der Wechsel um 18 Uhr auch ohne neuen Abruf greift.
func (s *Service) Report() Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.report
	if s.forecast != nil {
		r.Day = s.forecast.day(time.Now().In(s.loc))
	}
	return r
}

// Refresh laedt die Vorhersage neu. Bei Fehlern bleibt die letzte Vorhersage erhalten.
func (s *Service) Refresh(ctx context.Context) error {
	if !s.cfg.Enabled() {
		return nil
	}
	err := s.refresh(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.report.Error = ""
	if err != nil {
		slog.Warn("weather refresh failed", "err", err)
		s.report.Error = err.Error()
	}
	return err
}

func (s *Service) refresh(ctx context.Context) error {
	s.mu.RLock()
	needGeo := s.lat == 0 && s.lon == 0
	s.mu.RUnlock()
	if needGeo {
		if err := s.geocode(ctx); err != nil {
			return err
		}
	}
	s.mu.RLock()
	lat, lon := s.lat, s.lon
	s.mu.RUnlock()

	q := url.Values{
		"latitude":      {strconv.FormatFloat(lat, 'f', 4, 64)},
		"longitude":     {strconv.FormatFloat(lon, 'f', 4, 64)},
		"current":       {"temperature_2m,apparent_temperature,weather_code,wind_speed_10m,is_day"},
		"hourly":        {"temperature_2m,apparent_temperature,precipitation_probability,weather_code,wind_speed_10m,uv_index"},
		"timezone":      {s.loc.String()},
		"forecast_days": {"2"},
	}
	var f forecast
	if err := s.getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), &f); err != nil {
		return err
	}
	if len(f.Hourly.Time) == 0 {
		return errors.New("Vorhersage ohne Stundenwerte")
	}
	f.loc = s.loc
	icon, text := describe(f.Current.WeatherCode, f.Current.IsDay == 1)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.forecast = &f
	s.report.UpdatedAt = time.Now()
	s.report.Place = s.place
	s.report.Current = &Current{
		Temp: f.Current.Temperature, Feels: f.Current.ApparentTemperature,
		Wind: f.Current.WindSpeed, Icon: icon, Text: text,
	}
	return nil
}

func (s *Service) geocode(ctx context.Context) error {
	q := url.Values{"name": {s.cfg.Place}, "count": {"1"}, "language": {"de"}, "format": {"json"}}
	var res struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := s.getJSON(ctx, "https://geocoding-api.open-meteo.com/v1/search?"+q.Encode(), &res); err != nil {
		return fmt.Errorf("Ort suchen: %w", err)
	}
	if len(res.Results) == 0 {
		return fmt.Errorf("Ort %q nicht gefunden", s.cfg.Place)
	}
	s.mu.Lock()
	s.place, s.lat, s.lon = res.Results[0].Name, res.Results[0].Latitude, res.Results[0].Longitude
	s.mu.Unlock()
	return nil
}

func (s *Service) getJSON(ctx context.Context, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// ---- Open-Meteo-Antwort ----

type forecast struct {
	loc     *time.Location
	Current struct {
		Temperature         float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WeatherCode         int     `json:"weather_code"`
		WindSpeed           float64 `json:"wind_speed_10m"`
		IsDay               int     `json:"is_day"`
	} `json:"current"`
	Hourly struct {
		Time        []string   `json:"time"`
		Temperature []float64  `json:"temperature_2m"`
		Apparent    []float64  `json:"apparent_temperature"`
		PrecipProb  []*float64 `json:"precipitation_probability"`
		WeatherCode []int      `json:"weather_code"`
		WindSpeed   []float64  `json:"wind_speed_10m"`
		UV          []*float64 `json:"uv_index"`
	} `json:"hourly"`
}

// hour ist ein Stundenwert der Vorhersage.
type hour struct {
	at                time.Time
	temp, feels, wind float64
	precipProb, uv    float64
	code              int
}

func (f *forecast) hoursOn(date string) []hour {
	var out []hour
	h := f.Hourly
	for i, ts := range h.Time {
		if len(ts) < 10 || ts[:10] != date || i >= len(h.Temperature) {
			continue
		}
		at, err := time.ParseInLocation("2006-01-02T15:04", ts, f.loc)
		if err != nil {
			continue
		}
		out = append(out, hour{
			at: at, temp: h.Temperature[i], feels: at0(h.Apparent, i), wind: at0(h.WindSpeed, i),
			precipProb: deref(h.PrecipProb, i), uv: deref(h.UV, i), code: atInt(h.WeatherCode, i),
		})
	}
	return out
}

// Tagsueber zaehlt, was man draussen erlebt: Schulweg bis Heimweg.
const dayStart, dayEnd = 7, 20

var slotHours = []int{8, 12, 16, 20}

func (f *forecast) day(now time.Time) *Day {
	target, label := now, "Heute"
	if now.Hour() >= tomorrowFromHour {
		target, label = now.AddDate(0, 0, 1), "Morgen"
	}
	date := target.Format("2006-01-02")
	hours := f.hoursOn(date)
	if len(hours) == 0 {
		return nil
	}

	d := &Day{Label: label, Date: date, Min: 1e9, Max: -1e9}
	var daytime []hour
	for _, h := range hours {
		if h.at.Hour() >= dayStart && h.at.Hour() <= dayEnd {
			daytime = append(daytime, h)
		}
	}
	codeCount := map[int]int{}
	for _, h := range daytime {
		d.Min, d.Max = min(d.Min, h.temp), max(d.Max, h.temp)
		d.PrecipProb = max(d.PrecipProb, int(h.precipProb))
		d.UV = max(d.UV, h.uv)
		codeCount[h.code]++
	}
	d.Icon, d.Text = describe(dominantCode(codeCount), true)
	for _, h := range hours {
		for _, sh := range slotHours {
			if h.at.Hour() == sh {
				icon, text := describe(h.code, sh < 19)
				d.Slots = append(d.Slots, Slot{Hour: sh, Temp: h.temp, PrecipProb: int(h.precipProb), Icon: icon, Text: text})
			}
		}
	}
	d.Advice = Advice(daytime)
	return d
}

// dominantCode waehlt das praegende Wetter: Niederschlag schlaegt Wolken, sonst das haeufigste.
func dominantCode(counts map[int]int) int {
	best, bestScore := 0, -1
	for code, n := range counts {
		score := n
		if isWet(code) && n >= 2 {
			score += 100 + code // ab zwei Stunden Regen/Schnee ist das die Hauptinfo
		}
		if score > bestScore || (score == bestScore && code > best) {
			best, bestScore = code, score
		}
	}
	return best
}

func at0(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func atInt(s []int, i int) int {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func deref(s []*float64, i int) float64 {
	if i < len(s) && s[i] != nil {
		return *s[i]
	}
	return 0
}

// RunSync aktualisiert periodisch und ruft onUpdate nach jedem Abruf auf.
func (s *Service) RunSync(ctx context.Context, every time.Duration, onUpdate func()) {
	if !s.cfg.Enabled() {
		return
	}
	for {
		s.Refresh(ctx)
		onUpdate()
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}
