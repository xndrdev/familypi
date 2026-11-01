<script setup lang="ts">
import type { Member } from '~/types'

defineProps<{ members: Member[] }>()
const emit = defineEmits<{ changed: [] }>()
const { request } = useApi()

const error = ref('')
const empty = () => ({ id: 0, name: '', color: '#4f8cff', emoji: '', discordUserId: '', icalUrl: '', sort: 0 })
const form = ref<Member>(empty())

async function save() {
  error.value = ''
  try {
    if (form.value.id) {
      await request(`/api/members/${form.value.id}`, { method: 'PUT', body: form.value })
    } else {
      await request('/api/members', { method: 'POST', body: form.value })
    }
    form.value = empty()
    emit('changed')
  } catch (e) {
    error.value = errorMessage(e)
  }
}

async function remove(m: Member) {
  if (!window.confirm(`${m.name} wirklich entfernen? Ziele von ${m.name} werden mitgelöscht, Todos und Termine gehen an die Familie.`)) return
  await request(`/api/members/${m.id}`, { method: 'DELETE' })
  if (form.value.id === m.id) form.value = empty()
  emit('changed')
}
</script>

<template>
  <section class="adm-card">
    <h2>{{ form.id ? `${form.name} bearbeiten` : 'Familienmitglied hinzufügen' }}</h2>
    <form class="adm-form" @submit.prevent="save">
      <label class="adm-field">
        <span>Name</span>
        <input v-model="form.name" required maxlength="40">
      </label>
      <label class="adm-field">
        <span>Emoji</span>
        <input v-model="form.emoji" maxlength="8" placeholder="🦄">
      </label>
      <label class="adm-field">
        <span>Farbe</span>
        <input v-model="form.color" type="color">
      </label>
      <label class="adm-field">
        <span>Reihenfolge</span>
        <input v-model.number="form.sort" type="number">
      </label>
      <label class="adm-field adm-field--wide">
        <span>Discord-User-ID (für „ich“ bei /ziel)</span>
        <input v-model="form.discordUserId" inputmode="numeric" placeholder="z.B. 123456789012345678">
      </label>
      <label class="adm-field adm-field--wide">
        <span>Privater iCal-Link (Google, iCloud, Outlook) – optional</span>
        <input v-model="form.icalUrl" type="url" placeholder="https://calendar.google.com/calendar/ical/…/basic.ics">
      </label>
      <div class="adm-actions">
        <button class="adm-btn adm-btn--primary" type="submit">{{ form.id ? 'Speichern' : 'Hinzufügen' }}</button>
        <button v-if="form.id" class="adm-btn" type="button" @click="form = empty()">Abbrechen</button>
      </div>
      <p v-if="error" class="adm-error" role="alert">{{ error }}</p>
    </form>
  </section>

  <section class="adm-card">
    <h2>Familie</h2>
    <ul class="adm-list">
      <li v-for="m in members" :key="m.id" class="adm-row">
        <span class="adm-dot" :style="{ '--c': m.color }" aria-hidden="true" />
        <span class="adm-row__main">
          {{ m.emoji }} {{ m.name }}
          <span class="adm-row__sub">
            {{ m.icalUrl ? 'Kalender verknüpft' : 'kein externer Kalender' }}
            · {{ m.discordUserId ? 'Discord verknüpft' : 'Discord nicht verknüpft' }}
          </span>
        </span>
        <button class="adm-btn adm-btn--small" type="button" @click="form = { ...m }">Bearbeiten</button>
        <button class="adm-btn adm-btn--small adm-btn--danger" type="button" @click="remove(m)">Entfernen</button>
      </li>
      <li v-if="!members.length" class="adm-row adm-muted">Noch niemand angelegt.</li>
    </ul>
  </section>
</template>
