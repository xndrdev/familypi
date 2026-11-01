<script setup lang="ts">
import type { CalendarEvent, Member } from '~/types'

const props = defineProps<{ members: Member[], tick: number }>()
const { request } = useApi()

const RANGE_DAYS = 60
const events = ref<CalendarEvent[]>([])
const error = ref('')

function empty() {
  const today = isoDate(new Date())
  return {
    id: 0, title: '', memberId: null as number | null, allDay: false,
    startDate: today, startTime: '09:00', endDate: today, endTime: '10:00',
    location: '', notes: '',
  }
}
const form = ref(empty())

// Beim Verschieben des Starts das Ende mitziehen, wenn es sonst davor laege.
watch(() => form.value.startDate, (d) => {
  if (form.value.endDate < d) form.value.endDate = d
})

async function load() {
  const from = startOfDay(new Date())
  const to = addDays(from, RANGE_DAYS)
  events.value = await request<CalendarEvent[]>(`/api/events?from=${encodeURIComponent(from.toISOString())}&to=${encodeURIComponent(to.toISOString())}`)
}

function toBody() {
  const f = form.value
  const start = f.allDay ? parseIsoDate(f.startDate) : new Date(`${f.startDate}T${f.startTime}`)
  // Ganztaegig: Ende ist exklusiv, also Tag nach dem letzten Tag.
  const end = f.allDay ? addDays(parseIsoDate(f.endDate), 1) : new Date(`${f.endDate}T${f.endTime}`)
  return {
    title: f.title, memberId: f.memberId, allDay: f.allDay,
    start: start.toISOString(), end: end.toISOString(),
    location: f.location, notes: f.notes,
  }
}

async function save() {
  error.value = ''
  try {
    if (form.value.id) {
      await request(`/api/events/${form.value.id}`, { method: 'PUT', body: toBody() })
    } else {
      await request('/api/events', { method: 'POST', body: toBody() })
    }
    form.value = empty()
    await load()
  } catch (e) {
    error.value = errorMessage(e)
  }
}

function edit(e: CalendarEvent) {
  const start = new Date(e.start)
  const end = new Date(e.end)
  const lastDay = e.allDay ? addDays(end, -1) : end
  form.value = {
    id: e.id, title: e.title, memberId: e.memberId, allDay: e.allDay,
    startDate: isoDate(start), startTime: formatTime(start),
    endDate: isoDate(lastDay), endTime: formatTime(end),
    location: e.location, notes: e.notes,
  }
  window.scrollTo({ top: 0 })
}

async function remove(e: CalendarEvent) {
  await request(`/api/events/${e.id}`, { method: 'DELETE' })
  if (form.value.id === e.id) form.value = empty()
  await load()
}

const memberById = computed(() => new Map(props.members.map(m => [m.id, m])))

const byDay = computed(() => {
  const groups = new Map<string, CalendarEvent[]>()
  for (const e of events.value) {
    const key = isoDate(new Date(e.start) < startOfDay(new Date()) ? new Date() : new Date(e.start))
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(e)
  }
  return [...groups.entries()]
})

function timeLabel(e: CalendarEvent): string {
  if (e.allDay) {
    const last = addDays(new Date(e.end), -1)
    return isoDate(last) === isoDate(new Date(e.start)) ? 'ganztägig' : `ganztägig bis ${formatShortDate(last)}`
  }
  return `${formatTime(e.start)} – ${formatTime(e.end)}`
}

watch(() => props.tick, load)
onMounted(load)
</script>

<template>
  <section class="adm-card">
    <h2>{{ form.id ? 'Termin bearbeiten' : 'Neuer Termin' }}</h2>
    <form class="adm-form" @submit.prevent="save">
      <label class="adm-field adm-field--wide">
        <span>Titel</span>
        <input v-model="form.title" required maxlength="200">
      </label>
      <MemberSelect v-model="form.memberId" :members="members" />
      <label class="adm-field adm-field--check">
        <input v-model="form.allDay" type="checkbox"> Ganztägig
      </label>
      <label class="adm-field">
        <span>{{ form.allDay ? 'Von' : 'Beginn' }}</span>
        <input v-model="form.startDate" type="date" required>
      </label>
      <label v-if="!form.allDay" class="adm-field">
        <span>Uhrzeit</span>
        <input v-model="form.startTime" type="time" required>
      </label>
      <label class="adm-field">
        <span>{{ form.allDay ? 'Bis' : 'Ende' }}</span>
        <input v-model="form.endDate" type="date" required :min="form.startDate">
      </label>
      <label v-if="!form.allDay" class="adm-field">
        <span>Uhrzeit</span>
        <input v-model="form.endTime" type="time" required>
      </label>
      <label class="adm-field">
        <span>Ort</span>
        <input v-model="form.location" maxlength="200">
      </label>
      <label class="adm-field adm-field--wide">
        <span>Notiz</span>
        <textarea v-model="form.notes" rows="2" maxlength="2000" />
      </label>
      <div class="adm-actions">
        <button class="adm-btn adm-btn--primary" type="submit">{{ form.id ? 'Speichern' : 'Eintragen' }}</button>
        <button v-if="form.id" class="adm-btn" type="button" @click="form = empty()">Abbrechen</button>
      </div>
      <p v-if="error" class="adm-error" role="alert">{{ error }}</p>
    </form>
  </section>

  <section class="adm-card">
    <h2>Die nächsten {{ RANGE_DAYS }} Tage</h2>
    <p v-if="!byDay.length" class="adm-muted">Keine Termine.</p>
    <div v-for="[day, list] in byDay" :key="day">
      <h3 class="adm-section-title">{{ formatShortDate(day) }}</h3>
      <ul class="adm-list">
        <li v-for="e in list" :key="e.uid" class="adm-row">
          <span class="adm-dot" :style="{ '--c': e.memberId ? memberById.get(e.memberId)?.color : '#ffcf5c' }" aria-hidden="true" />
          <span class="adm-row__main">
            {{ e.title }}
            <span class="adm-row__sub">
              {{ timeLabel(e) }}
              · {{ e.memberId ? memberById.get(e.memberId)?.name ?? '?' : 'Familie' }}
              <template v-if="e.location"> · {{ e.location }}</template>
            </span>
          </span>
          <template v-if="e.source === 'local'">
            <button class="adm-btn adm-btn--small" type="button" @click="edit(e)">Bearbeiten</button>
            <button class="adm-btn adm-btn--small adm-btn--danger" type="button" @click="remove(e)">Löschen</button>
          </template>
          <span v-else class="adm-badge" title="Aus einem verknüpften Kalender – dort bearbeiten">extern</span>
        </li>
      </ul>
    </div>
  </section>
</template>
