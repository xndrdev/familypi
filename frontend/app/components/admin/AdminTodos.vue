<script setup lang="ts">
import type { Member, Todo } from '~/types'

const props = defineProps<{ members: Member[], tick: number }>()
const { request } = useApi()

const todos = ref<Todo[]>([])
const showDone = ref(false)
const error = ref('')
const empty = () => ({ id: 0, title: '', memberId: null as number | null, due: '' })
const form = ref(empty())

async function load() {
  todos.value = await request<Todo[]>(`/api/todos${showDone.value ? '?all=1' : ''}`)
}

async function save() {
  error.value = ''
  const body = { title: form.value.title, memberId: form.value.memberId, due: form.value.due || null }
  try {
    if (form.value.id) {
      await request(`/api/todos/${form.value.id}`, { method: 'PUT', body })
    } else {
      await request('/api/todos', { method: 'POST', body })
    }
    form.value = empty()
    await load()
  } catch (e) {
    error.value = errorMessage(e)
  }
}

function edit(t: Todo) {
  form.value = { id: t.id, title: t.title, memberId: t.memberId, due: t.due ?? '' }
}

async function toggle(t: Todo) {
  await request(`/api/todos/${t.id}`, { method: 'PUT', body: { done: !t.done } })
  await load()
}

async function remove(t: Todo) {
  await request(`/api/todos/${t.id}`, { method: 'DELETE' })
  if (form.value.id === t.id) form.value = empty()
  await load()
}

const memberById = computed(() => new Map(props.members.map(m => [m.id, m])))

watch(() => props.tick, load)
watch(showDone, load)
onMounted(load)
</script>

<template>
  <section class="adm-card">
    <h2>{{ form.id ? 'Todo bearbeiten' : 'Neues Todo' }}</h2>
    <form class="adm-form" @submit.prevent="save">
      <label class="adm-field adm-field--wide">
        <span>Was ist zu tun?</span>
        <input v-model="form.title" required maxlength="200">
      </label>
      <MemberSelect v-model="form.memberId" :members="members" />
      <label class="adm-field">
        <span>Fällig am (optional)</span>
        <input v-model="form.due" type="date">
      </label>
      <div class="adm-actions">
        <button class="adm-btn adm-btn--primary" type="submit">{{ form.id ? 'Speichern' : 'Hinzufügen' }}</button>
        <button v-if="form.id" class="adm-btn" type="button" @click="form = empty()">Abbrechen</button>
      </div>
      <p v-if="error" class="adm-error" role="alert">{{ error }}</p>
    </form>
  </section>

  <section class="adm-card">
    <h2>Todos</h2>
    <label class="adm-field adm-field--check">
      <input v-model="showDone" type="checkbox"> Auch ältere erledigte anzeigen
    </label>
    <ul class="adm-list">
      <li v-for="t in todos" :key="t.id" class="adm-row">
        <input type="checkbox" :checked="t.done" :aria-label="`${t.title} erledigt`" @change="toggle(t)">
        <span class="adm-row__main">
          <span :class="{ 'adm-done': t.done }">{{ t.title }}</span>
          <span class="adm-row__sub">
            {{ t.memberId ? `${memberById.get(t.memberId)?.emoji ?? ''} ${memberById.get(t.memberId)?.name ?? '?'}` : '👪 Familie' }}
            <template v-if="t.due"> · fällig {{ formatShortDate(t.due) }}</template>
            <template v-if="t.createdBy"> · von {{ t.createdBy }}</template>
          </span>
        </span>
        <button class="adm-btn adm-btn--small" type="button" @click="edit(t)">Bearbeiten</button>
        <button class="adm-btn adm-btn--small adm-btn--danger" type="button" @click="remove(t)">Löschen</button>
      </li>
      <li v-if="!todos.length" class="adm-row adm-muted">Keine Todos.</li>
    </ul>
  </section>
</template>
