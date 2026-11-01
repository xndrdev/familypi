package weather

// Advice leitet aus den Tagesstunden ab, was man anziehen und mitnehmen sollte.
// Massgeblich ist die gefuehlte Temperatur, weil Wind und Feuchte den Unterschied machen.
func Advice(hours []hour) []string {
	if len(hours) == 0 {
		return nil
	}
	minFeels, maxFeels := hours[0].feels, hours[0].feels
	var maxRain, maxWind, maxUV float64
	snow, thunder, rainNow := false, false, false
	for _, h := range hours {
		minFeels, maxFeels = min(minFeels, h.feels), max(maxFeels, h.feels)
		maxRain, maxWind, maxUV = max(maxRain, h.precipProb), max(maxWind, h.wind), max(maxUV, h.uv)
		snow = snow || isSnow(h.code)
		thunder = thunder || h.code >= 95
		rainNow = rainNow || (isWet(h.code) && !isSnow(h.code))
	}

	var out []string
	switch {
	case minFeels < 0:
		out = append(out, "🧤 Winterjacke, Mütze und Handschuhe")
	case minFeels < 7:
		out = append(out, "🧥 Warme Jacke, gern mit Mütze")
	case minFeels < 13:
		out = append(out, "🧥 Jacke und Pulli")
	case minFeels < 18:
		out = append(out, "👕 Pulli oder leichte Jacke")
	case maxFeels >= 24:
		out = append(out, "🩳 T-Shirt und kurze Hose")
	default:
		out = append(out, "👕 T-Shirt, Pulli für später")
	}
	if maxFeels-minFeels >= 8 && minFeels < 18 {
		out = append(out, "🧅 Zwiebellook – später wird es deutlich wärmer")
	}

	switch {
	case snow:
		out = append(out, "🥾 Winterstiefel – es kann schneien")
	case maxRain >= 60 || rainNow:
		out = append(out, "☔ Regenjacke und Gummistiefel")
	case maxRain >= 30:
		out = append(out, "🌂 Schirm einpacken")
	}
	if thunder {
		out = append(out, "⛈️ Gewitter möglich")
	}
	if maxWind >= 40 {
		out = append(out, "🌬️ Sehr windig – Kapuze statt Schirm")
	}
	switch {
	case maxUV >= 6:
		out = append(out, "🧴 Sonnencreme und Kappe")
	case maxUV >= 4 && maxFeels >= 18:
		out = append(out, "🧢 Kappe auf")
	}
	return out
}

func isSnow(code int) bool { return (code >= 71 && code <= 77) || code == 85 || code == 86 }

func isWet(code int) bool { return code >= 51 }

// describe uebersetzt WMO-Wettercodes in Emoji und deutschen Text.
func describe(code int, day bool) (icon, text string) {
	switch {
	case code == 0:
		if day {
			return "☀️", "Sonnig"
		}
		return "🌙", "Klar"
	case code == 1:
		if day {
			return "🌤️", "Überwiegend sonnig"
		}
		return "🌙", "Überwiegend klar"
	case code == 2:
		return "⛅", "Teils bewölkt"
	case code == 3:
		return "☁️", "Bewölkt"
	case code == 45 || code == 48:
		return "🌫️", "Nebel"
	case code >= 51 && code <= 57:
		return "🌦️", "Nieselregen"
	case code == 61 || code == 80:
		return "🌦️", "Leichter Regen"
	case code == 63 || code == 81:
		return "🌧️", "Regen"
	case code == 65 || code == 82:
		return "🌧️", "Starker Regen"
	case code == 66 || code == 67:
		return "🧊", "Gefrierender Regen"
	case isSnow(code):
		return "🌨️", "Schnee"
	case code >= 95:
		return "⛈️", "Gewitter"
	}
	return "🌡️", "Wetter"
}
