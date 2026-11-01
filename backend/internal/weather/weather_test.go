package weather

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func h(feels, rain, wind, uv float64, code int) hour {
	return hour{feels: feels, precipProb: rain, wind: wind, uv: uv, code: code}
}

func TestAdvice(t *testing.T) {
	cases := []struct {
		name  string
		hours []hour
		want  []string
	}{
		{"frostig und Schnee", []hour{h(-4, 20, 10, 0, 3), h(-1, 70, 15, 1, 73)}, []string{"🧤", "🥾"}},
		{"kuehler Regentag", []hour{h(4, 80, 20, 1, 63), h(6, 60, 25, 1, 61)}, []string{"🧥 Warme", "☔"}},
		{"Herbst mit Schauerrisiko", []hour{h(9, 35, 10, 2, 3), h(12, 20, 10, 2, 2)}, []string{"🧥 Jacke und Pulli", "🌂"}},
		{"Fruehling kalt-warm", []hour{h(8, 0, 5, 3, 1), h(19, 0, 5, 4, 0)}, []string{"🧥 Jacke und Pulli", "🧅"}},
		{"Sommer", []hour{h(22, 0, 5, 7, 0), h(29, 10, 5, 8, 0)}, []string{"🩳", "🧴"}},
		{"Sturm", []hour{h(10, 10, 55, 1, 3)}, []string{"🌬️"}},
	}
	for _, c := range cases {
		got := Advice(c.hours)
		for _, prefix := range c.want {
			if !slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, prefix) }) {
				t.Errorf("%s: want advice starting with %q, got %v", c.name, prefix, got)
			}
		}
	}
	if got := Advice([]hour{h(22, 0, 5, 7, 0)}); slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, "☔") }) {
		t.Errorf("dry day should not suggest rain gear: %v", got)
	}
}

// fakeForecast baut zwei Tage Stundenwerte: morgens kalt, nachmittags Regen.
func fakeForecast(loc *time.Location) *forecast {
	f := &forecast{loc: loc}
	for _, day := range []string{"2026-11-02", "2026-11-03"} {
		for hr := 0; hr < 24; hr++ {
			f.Hourly.Time = append(f.Hourly.Time, fmt.Sprintf("%sT%02d:00", day, hr))
			f.Hourly.Temperature = append(f.Hourly.Temperature, float64(hr/3))
			f.Hourly.Apparent = append(f.Hourly.Apparent, float64(hr/3)-2)
			f.Hourly.WindSpeed = append(f.Hourly.WindSpeed, 10)
			code, rain := 3, 10.0
			if hr >= 14 && hr <= 18 {
				code, rain = 63, 80
			}
			f.Hourly.WeatherCode = append(f.Hourly.WeatherCode, code)
			f.Hourly.PrecipProb = append(f.Hourly.PrecipProb, &rain)
			f.Hourly.UV = append(f.Hourly.UV, nil) // Open-Meteo liefert nachts teils null
		}
	}
	return f
}

func TestDaySwitchesToTomorrowInTheEvening(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	f := fakeForecast(loc)

	morning := f.day(time.Date(2026, 11, 2, 7, 0, 0, 0, loc))
	if morning.Label != "Heute" || morning.Date != "2026-11-02" {
		t.Fatalf("morning: %+v", morning)
	}
	if morning.Min != 2 || morning.Max != 6 || morning.PrecipProb != 80 {
		t.Errorf("daytime range wrong: min %v max %v rain %v", morning.Min, morning.Max, morning.PrecipProb)
	}
	if morning.Text != "Regen" {
		t.Errorf("rainy afternoon should dominate, got %q", morning.Text)
	}
	if len(morning.Slots) != 4 || morning.Slots[2].Hour != 16 || morning.Slots[2].PrecipProb != 80 {
		t.Errorf("slots wrong: %+v", morning.Slots)
	}

	evening := f.day(time.Date(2026, 11, 2, 19, 0, 0, 0, loc))
	if evening.Label != "Morgen" || evening.Date != "2026-11-03" {
		t.Errorf("evening should show tomorrow: %+v", evening)
	}
	if f.day(time.Date(2026, 11, 3, 19, 0, 0, 0, loc)) != nil {
		t.Error("no data for the day after should yield nil")
	}
}

func TestDecodeOpenMeteoNulls(t *testing.T) {
	raw := `{"current":{"temperature_2m":7.8,"apparent_temperature":5.9,"weather_code":2,"wind_speed_10m":4.4,"is_day":0},
	"hourly":{"time":["2026-11-02T08:00"],"temperature_2m":[5],"apparent_temperature":[3],"precipitation_probability":[null],"weather_code":[61],"wind_speed_10m":[12],"uv_index":[null]}}`
	var f forecast
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Berlin")
	f.loc = loc
	hours := f.hoursOn("2026-11-02")
	if len(hours) != 1 || hours[0].precipProb != 0 || hours[0].code != 61 {
		t.Errorf("decode wrong: %+v", hours)
	}
}
