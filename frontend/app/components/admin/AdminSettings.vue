<script setup lang="ts">
import type { FeedStatus, Member, WeatherReport } from '~/types'

const props = defineProps<{ members: Member[], authRequired: boolean }>()
defineEmits<{ logout: [] }>()
const { request } = useApi()

const feeds = ref<FeedStatus[]>([])
const weather = ref<WeatherReport | null>(null)
const refreshing = ref(false)
const icsUrl = ref('')
const copied = ref(false)

async function load() {
  ;[feeds.value, weather.value] = await Promise.all([
    request<FeedStatus[]>('/api/calendars'),
    request<WeatherReport>('/api/weather'),
  ])
}

async function refresh() {
  refreshing.value = true
  try {
    feeds.value = await request<FeedStatus[]>('/api/calendars/refresh', { method: 'POST' })
  } finally {
    refreshing.value = false
  }
}

async function copy() {
  try {
    await navigator.clipboard.writeText(icsUrl.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // Zwischenablage nicht verfuegbar (z.B. ohne HTTPS) – Link bleibt markierbar.
  }
}

const memberName = (id: number) => props.members.find(m => m.id === id)?.name ?? '?'

onMounted(() => {
  icsUrl.value = `${window.location.origin.replace(/^http/, 'webcal')}/calendar.ics`
  load()
})
</script>

<template>
  <section class="adm-card">
    <h2>Familienkalender abonnieren</h2>
    <p class="adm-muted">
      Diesen Link auf dem Handy als Kalender-Abo hinzufügen (iPhone: Einstellungen → Kalender → Accounts → Kalenderabo;
      Google Kalender: „Per URL hinzufügen“ mit https:// statt webcal://). Enthält alle hier eingetragenen Termine.
      Ist <code>FD_ICS_TOKEN</code> gesetzt, muss <code>?token=…</code> angehängt werden.
    </p>
    <div class="adm-form">
      <label class="adm-field adm-field--wide">
        <span>Abo-Link</span>
        <input :value="icsUrl" readonly @focus="($event.target as HTMLInputElement).select()">
      </label>
      <div class="adm-actions">
        <button class="adm-btn" type="button" @click="copy">{{ copied ? 'Kopiert ✓' : 'Kopieren' }}</button>
      </div>
    </div>
  </section>

  <section class="adm-card">
    <h2>Verknüpfte Kalender</h2>
    <p class="adm-muted">Externe Kalender werden alle 15 Minuten neu geladen. Den Link pro Person unter „Familie“ hinterlegen.</p>
    <ul class="adm-list">
      <li v-for="f in feeds" :key="f.memberId" class="adm-row">
        <span class="adm-row__main">
          {{ memberName(f.memberId) }}
          <span class="adm-row__sub">
            {{ f.events }} Termine · zuletzt {{ formatTime(f.fetchedAt) }}
          </span>
        </span>
        <span v-if="f.error" class="adm-error">⚠ {{ f.error }}</span>
        <span v-else class="adm-badge">ok</span>
      </li>
      <li v-if="!feeds.length" class="adm-row adm-muted">Kein externer Kalender verknüpft.</li>
    </ul>
    <button class="adm-btn" type="button" :disabled="refreshing" @click="refresh">
      {{ refreshing ? 'Lädt …' : 'Jetzt aktualisieren' }}
    </button>
  </section>

  <section class="adm-card">
    <h2>Wetter</h2>
    <p v-if="!weather?.enabled" class="adm-muted">
      Ausgeschaltet. Setze <code>FD_WEATHER_PLACE</code> (z.B. <code>Leipzig</code>) oder
      <code>FD_WEATHER_LAT</code>/<code>FD_WEATHER_LON</code> und starte den Dienst neu.
    </p>
    <template v-else>
      <p>
        {{ weather.place || 'Ort wird gesucht …' }}
        <template v-if="weather.current"> · {{ weather.current.icon }} {{ Math.round(weather.current.temp) }} °C</template>
        <span v-if="weather.updatedAt && !weather.updatedAt.startsWith('0001')" class="adm-muted"> · aktualisiert {{ formatTime(weather.updatedAt) }}</span>
      </p>
      <p v-if="weather.error" class="adm-error">⚠ {{ weather.error }}</p>
    </template>
  </section>

  <section v-if="authRequired" class="adm-card">
    <h2>Zugang</h2>
    <button class="adm-btn" type="button" @click="$emit('logout')">Abmelden</button>
  </section>
</template>
