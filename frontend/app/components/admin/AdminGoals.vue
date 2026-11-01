<script setup lang="ts">
import type { Goal, Member } from '~/types'

const props = defineProps<{ members: Member[], tick: number }>()
const { request } = useApi()

const goals = ref<Goal[]>([])
const error = ref('')
const empty = () => ({
  id: 0, memberId: props.members[0]?.id ?? null as number | null, title: '', description: '',
  targetDate: '', progress: 0, done: false, sort: 0,
})
const form = ref(empty())

async function load() {
  goals.value = await request<Goal[]>('/api/goals')
}

async function save() {
  error.value = ''
  const body = { ...form.value, targetDate: form.value.targetDate || null }
  try {
    if (form.value.id) {
      await request(`/api/goals/${form.value.id}`, { method: 'PUT', body })
    } else {
      await request('/api/goals', { method: 'POST', body })
    }
    form.value = empty()
    await load()
  } catch (e) {
    error.value = errorMessage(e)
  }
}

function edit(g: Goal) {
  form.value = { ...g, targetDate: g.targetDate ?? '' }
  window.scrollTo({ top: 0 })
}

async function patch(g: Goal, changes: Partial<Goal>) {
  await request(`/api/goals/${g.id}`, { method: 'PUT', body: changes })
  await load()
}

async function remove(g: Goal) {
  await request(`/api/goals/${g.id}`, { method: 'DELETE' })
  if (form.value.id === g.id) form.value = empty()
  await load()
}

// Pro Mitglied: offene Ziele in Anzeige-Reihenfolge, das erste ist "das naechste grosse Ziel".
const grouped = computed(() => props.members.map(m => ({
  member: m,
  open: goals.value.filter(g => g.memberId === m.id && !g.done),
  done: goals.value.filter(g => g.memberId === m.id && g.done),
})))

// Ziel in der Reihenfolge nach oben/unten schieben, indem die Sortierwerte neu vergeben werden.
async function move(list: Goal[], index: number, dir: -1 | 1) {
  const target = index + dir
  if (target < 0 || target >= list.length) return
  const reordered = [...list]
  ;[reordered[index], reordered[target]] = [reordered[target]!, reordered[index]!]
  await Promise.all(reordered.map((g, i) => g.sort === i ? null : request(`/api/goals/${g.id}`, { method: 'PUT', body: { sort: i } })))
  await load()
}

watch(() => props.tick, load)
watch(() => props.members, () => {
  if (!form.value.memberId) form.value.memberId = props.members[0]?.id ?? null
})
onMounted(load)
</script>

<template>
  <section class="adm-card">
    <h2>{{ form.id ? 'Ziel bearbeiten' : 'Neues Ziel' }}</h2>
    <p v-if="!members.length" class="adm-muted">Lege zuerst unter „Familie“ die Familienmitglieder an.</p>
    <form v-else class="adm-form" @submit.prevent="save">
      <label class="adm-field adm-field--wide">
        <span>Ziel</span>
        <input v-model="form.title" required maxlength="120">
      </label>
      <MemberSelect v-model="form.memberId" :members="members" :allow-family="false" label="Wessen Ziel?" />
      <label class="adm-field">
        <span>Bis (optional)</span>
        <input v-model="form.targetDate" type="date">
      </label>
      <label class="adm-field">
        <span>Fortschritt: {{ form.progress }} %</span>
        <input v-model.number="form.progress" type="range" min="0" max="100" step="5">
      </label>
      <label class="adm-field adm-field--check">
        <input v-model="form.done" type="checkbox"> Erreicht
      </label>
      <label class="adm-field adm-field--wide">
        <span>Beschreibung</span>
        <textarea v-model="form.description" rows="2" maxlength="1000" />
      </label>
      <div class="adm-actions">
        <button class="adm-btn adm-btn--primary" type="submit">{{ form.id ? 'Speichern' : 'Anlegen' }}</button>
        <button v-if="form.id" class="adm-btn" type="button" @click="form = empty()">Abbrechen</button>
      </div>
      <p v-if="error" class="adm-error" role="alert">{{ error }}</p>
    </form>
  </section>

  <section v-for="g in grouped" :key="g.member.id" class="adm-card">
    <h2><span class="adm-dot" :style="{ '--c': g.member.color }" aria-hidden="true" /> {{ g.member.emoji }} {{ g.member.name }}</h2>
    <p class="adm-muted">Das oberste offene Ziel erscheint auf dem Display.</p>
    <ul class="adm-list">
      <li v-for="(goal, i) in g.open" :key="goal.id" class="adm-row">
        <span class="adm-row__main">
          <strong v-if="i === 0">⭐ </strong>{{ goal.title }}
          <span class="adm-row__sub">
            {{ goal.progress }} %<template v-if="goal.targetDate"> · bis {{ formatShortDate(goal.targetDate) }}</template>
          </span>
        </span>
        <button class="adm-btn adm-btn--small" type="button" :disabled="i === 0" aria-label="Nach oben" @click="move(g.open, i, -1)">↑</button>
        <button class="adm-btn adm-btn--small" type="button" :disabled="i === g.open.length - 1" aria-label="Nach unten" @click="move(g.open, i, 1)">↓</button>
        <button class="adm-btn adm-btn--small" type="button" @click="patch(goal, { done: true, progress: 100 })">Erreicht 🎉</button>
        <button class="adm-btn adm-btn--small" type="button" @click="edit(goal)">Bearbeiten</button>
        <button class="adm-btn adm-btn--small adm-btn--danger" type="button" @click="remove(goal)">Löschen</button>
      </li>
      <li v-if="!g.open.length" class="adm-row adm-muted">Kein offenes Ziel.</li>
    </ul>
    <details v-if="g.done.length">
      <summary class="adm-muted">Erreichte Ziele ({{ g.done.length }})</summary>
      <ul class="adm-list">
        <li v-for="goal in g.done" :key="goal.id" class="adm-row">
          <span class="adm-row__main adm-done">{{ goal.title }}</span>
          <button class="adm-btn adm-btn--small" type="button" @click="patch(goal, { done: false })">Wieder öffnen</button>
          <button class="adm-btn adm-btn--small adm-btn--danger" type="button" @click="remove(goal)">Löschen</button>
        </li>
      </ul>
    </details>
  </section>
</template>
