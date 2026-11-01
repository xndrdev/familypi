<script setup lang="ts">
import type { Dashboard, Member, WeatherReport } from '~/types'

useHead({ title: 'Familien-Display' })

const { request } = useApi()
const data = ref<Dashboard | null>(null)
const weather = ref<WeatherReport | null>(null)
const error = ref('')
const now = ref(new Date())

async function load() {
  try {
    const [dashboard, report] = await Promise.all([
      request<Dashboard>('/api/dashboard?days=7'),
      // Wetter ist Zusatz: ein Fehler dort soll das restliche Display nicht blockieren.
      request<WeatherReport>('/api/weather').catch(() => null),
    ])
    data.value = dashboard
    weather.value = report
    error.value = ''
  } catch (e) {
    error.value = errorMessage(e)
  }
}

const { connected } = useLiveUpdates(load)

let clockTimer: ReturnType<typeof setInterval> | undefined
let reloadTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  load()
  clockTimer = setInterval(() => {
    const prev = now.value
    now.value = new Date()
    // Tageswechsel und 18 Uhr (Wetter springt auf morgen): neu laden
    if (prev.getDate() !== now.value.getDate() || prev.getHours() !== now.value.getHours()) load()
  }, 1000)
  // Sicherheitsnetz, falls der Live-Stream unbemerkt abreisst
  reloadTimer = setInterval(load, 5 * 60 * 1000)
})
onBeforeUnmount(() => {
  clearInterval(clockTimer)
  clearInterval(reloadTimer)
})

const members = computed(() => {
  const map = new Map<number, Member>()
  for (const m of data.value?.members ?? []) map.set(m.id, m)
  return map
})

// Nachts das Display abdunkeln, um Blendung und Einbrennen zu vermeiden.
const night = computed(() => now.value.getHours() >= 22 || now.value.getHours() < 6)
</script>

<template>
  <div class="display" :class="{ 'display--night': night }">
    <DisplayHeader :now="now" :connected="connected" :error="error">
      <DisplayWeather v-if="weather?.enabled" :weather="weather" />
    </DisplayHeader>
    <main v-if="data" class="display__grid">
      <DisplayWeek class="display__week" :events="data.events" :members="members" :now="now" />
      <DisplayGoals class="display__goals" :goals="data.goals" />
      <DisplayTodos class="display__todos" :todos="data.todos" :members="members" />
    </main>
    <p v-else-if="!error" class="display__loading">Lade …</p>
  </div>
</template>

<style>
.display {
  --d-bg: #0e1117;
  --d-surface: #171b24;
  --d-surface-2: #1f2430;
  --d-border: #2a3140;
  --d-text: #eef1f6;
  --d-muted: #95a0b3;
  --d-accent: #ffcf5c;

  position: fixed;
  inset: 0;
  display: flex;
  flex-direction: column;
  gap: 1.2vh;
  padding: 2vh 1.6vw;
  background: var(--d-bg);
  color: var(--d-text);
  font-size: clamp(0.9rem, 1.15vw, 2.2rem);
  overflow: hidden;
  cursor: none;
  transition: filter 2s;
}

.display--night {
  filter: brightness(0.45);
}

.display__grid {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 2.2fr 1fr;
  grid-template-rows: 1.7fr 1fr;
  gap: 1.2vw;
}

.display__week {
  grid-column: 1;
  grid-row: 1;
}

.display__goals {
  grid-column: 1;
  grid-row: 2;
}

.display__todos {
  grid-column: 2;
  grid-row: 1 / span 2;
}

.display__loading {
  margin: auto;
  color: var(--d-muted);
}

.panel {
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 1.2vw;
  background: var(--d-surface);
  border: 1px solid var(--d-border);
  border-radius: 1vw;
}

.panel__title {
  margin: 0 0 0.8vw;
  font-size: 0.8em;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--d-muted);
}

/* Hochformat-Monitore: alles untereinander */
@media (orientation: portrait) {
  .display {
    font-size: clamp(0.9rem, 2.1vw, 2.2rem);
  }

  .display__grid {
    grid-template-columns: 1fr;
    grid-template-rows: 2fr auto 1.2fr;
  }

  .display__week,
  .display__goals,
  .display__todos {
    grid-column: 1;
    grid-row: auto;
  }
}
</style>
