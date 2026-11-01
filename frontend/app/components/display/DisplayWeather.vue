<script setup lang="ts">
import type { WeatherReport } from '~/types'

defineProps<{ weather: WeatherReport }>()

const deg = (t: number) => `${Math.round(t)}°`
</script>

<template>
  <section v-if="weather.current || weather.day" class="d-weather" aria-label="Wetter">
    <div v-if="weather.current" class="d-weather__now">
      <span class="d-weather__icon" aria-hidden="true">{{ weather.current.icon }}</span>
      <div>
        <div class="d-weather__temp">{{ deg(weather.current.temp) }}</div>
        <div class="d-weather__sub">{{ weather.current.text }} · gefühlt {{ deg(weather.current.feels) }}</div>
      </div>
    </div>

    <div v-if="weather.day" class="d-weather__day">
      <div class="d-weather__summary">
        <strong>{{ weather.day.label }}:</strong>
        {{ weather.day.icon }} {{ deg(weather.day.min) }} bis {{ deg(weather.day.max) }}
        <template v-if="weather.day.precipProb >= 20"> · 💧 {{ weather.day.precipProb }} %</template>
      </div>
      <ol class="d-weather__slots">
        <li v-for="s in weather.day.slots" :key="s.hour" class="d-weather__slot" :title="s.text">
          <span class="d-weather__hour">{{ s.hour }} Uhr</span>
          <span aria-hidden="true">{{ s.icon }}</span>
          <span class="d-weather__slot-temp">{{ deg(s.temp) }}</span>
          <span v-if="s.precipProb >= 30" class="d-weather__rain">{{ s.precipProb }} %</span>
        </li>
      </ol>
    </div>

    <ul v-if="weather.day?.advice.length" class="d-weather__advice" aria-label="Anzieh-Tipps">
      <li v-for="a in weather.day.advice" :key="a">{{ a }}</li>
    </ul>
  </section>
</template>

<style>
.d-weather {
  display: flex;
  align-items: center;
  gap: 1.4vw;
  min-width: 0;
  white-space: nowrap;
}

.d-weather__now {
  display: flex;
  align-items: center;
  gap: 0.6vw;
}

.d-weather__icon {
  font-size: 2.6em;
  line-height: 1;
}

.d-weather__temp {
  font-size: 2.2em;
  font-weight: 700;
  line-height: 1;
}

.d-weather__sub {
  font-size: 0.8em;
  color: var(--d-muted);
}

.d-weather__day {
  display: flex;
  flex-direction: column;
  gap: 0.3vw;
}

.d-weather__summary {
  font-size: 0.95em;
}

.d-weather__slots {
  display: flex;
  gap: 0.9vw;
  margin: 0;
  padding: 0;
  list-style: none;
}

.d-weather__slot {
  display: flex;
  align-items: baseline;
  gap: 0.25vw;
  font-size: 0.85em;
}

.d-weather__hour {
  color: var(--d-muted);
}

.d-weather__slot-temp {
  font-weight: 600;
}

.d-weather__rain {
  font-size: 0.85em;
  color: #7cc4ff;
}

.d-weather__advice {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.4vw;
  margin: 0;
  padding: 0;
  list-style: none;
}

.d-weather__advice li {
  padding: 0.25vw 0.7vw;
  font-size: 0.85em;
  font-weight: 600;
  background: color-mix(in srgb, var(--d-accent) 16%, transparent);
  border: 1px solid color-mix(in srgb, var(--d-accent) 45%, transparent);
  border-radius: 999px;
}

@media (orientation: portrait) {
  .d-weather {
    flex-wrap: wrap;
  }
}
</style>
