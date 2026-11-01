<script setup lang="ts">
import type { Member } from '~/types'

useHead({ title: 'Familie · Verwaltung' })

const { request, token, setToken } = useApi()

const tabs = [
  { id: 'todos', label: 'Todos' },
  { id: 'events', label: 'Termine' },
  { id: 'goals', label: 'Ziele' },
  { id: 'members', label: 'Familie' },
  { id: 'settings', label: 'Kalender & Zugang' },
] as const
type TabId = typeof tabs[number]['id']

const route = useRoute()
const router = useRouter()
const tab = computed<TabId>({
  get: () => (tabs.some(t => t.id === route.query.tab) ? route.query.tab as TabId : 'todos'),
  set: v => router.replace({ query: { tab: v } }),
})

const auth = ref<{ required: boolean, valid: boolean } | null>(null)
const tokenInput = ref('')
const members = ref<Member[]>([])
const tick = ref(0) // erhoeht sich bei Live-Updates, Tabs laden dann neu

async function checkAuth() {
  auth.value = await request<{ required: boolean, valid: boolean }>('/api/auth')
}

async function loadMembers() {
  members.value = await request<Member[]>('/api/members')
}

async function login() {
  setToken(tokenInput.value.trim())
  await checkAuth()
  if (auth.value?.valid) await loadMembers()
}

function logout() {
  setToken('')
  checkAuth()
}

useLiveUpdates(() => {
  loadMembers()
  tick.value++
})

onMounted(async () => {
  await checkAuth()
  await loadMembers()
})

const canWrite = computed(() => !!auth.value && (!auth.value.required || auth.value.valid))
</script>

<template>
  <div class="adm">
    <header class="adm__top">
      <h1 class="adm__brand">👪 Familien-Verwaltung</h1>
      <nav class="adm__tabs" aria-label="Bereiche">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          class="adm__tab"
          :aria-current="tab === t.id ? 'page' : undefined"
          @click="tab = t.id"
        >
          {{ t.label }}
        </button>
      </nav>
      <NuxtLink class="adm__link" to="/display" target="_blank">Display öffnen ↗</NuxtLink>
    </header>

    <main class="adm__main">
      <form v-if="auth && !canWrite" class="adm-card adm-login" @submit.prevent="login">
        <h2>Anmelden</h2>
        <p class="adm-muted">Die Verwaltung ist geschützt. Das Passwort steht als <code>FD_ADMIN_TOKEN</code> in der Konfiguration.</p>
        <label class="adm-field">
          <span>Passwort</span>
          <input v-model="tokenInput" type="password" autocomplete="current-password" required>
        </label>
        <p v-if="token && !auth.valid" class="adm-error" role="alert">Das Passwort ist falsch.</p>
        <button class="adm-btn adm-btn--primary" type="submit">Anmelden</button>
      </form>

      <template v-else-if="canWrite">
        <AdminTodos v-if="tab === 'todos'" :members="members" :tick="tick" />
        <AdminEvents v-else-if="tab === 'events'" :members="members" :tick="tick" />
        <AdminGoals v-else-if="tab === 'goals'" :members="members" :tick="tick" />
        <AdminMembers v-else-if="tab === 'members'" :members="members" @changed="loadMembers" />
        <AdminSettings v-else :members="members" :auth-required="auth!.required" @logout="logout" />
      </template>
    </main>
  </div>
</template>

<style>
.adm {
  min-height: 100vh;
}

.adm__top {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem 1.5rem;
  padding: 0.75rem 1rem;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
}

.adm__brand {
  margin: 0;
  font-size: 1.15rem;
}

.adm__tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.adm__tab {
  padding: 0.45rem 0.8rem;
  background: none;
  border: 0;
  border-radius: 0.5rem;
  color: var(--muted);
  cursor: pointer;
}

.adm__tab:hover {
  background: var(--bg);
}

.adm__tab[aria-current='page'] {
  background: var(--accent);
  color: var(--accent-text);
}

.adm__link {
  margin-left: auto;
  color: var(--accent);
}

.adm__main {
  max-width: 64rem;
  margin: 0 auto;
  padding: 1.5rem 1rem 4rem;
}

/* ---- Bausteine, die alle Admin-Tabs teilen ---- */

.adm-card {
  padding: 1.25rem;
  margin-bottom: 1.5rem;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.adm-card h2 {
  margin: 0 0 1rem;
  font-size: 1.1rem;
}

.adm-login {
  max-width: 26rem;
  margin: 3rem auto;
}

.adm-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
  gap: 0.75rem 1rem;
  align-items: end;
}

.adm-field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  font-size: 0.9rem;
}

.adm-field--wide {
  grid-column: 1 / -1;
}

.adm-field--check {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}

.adm-field input:not([type='checkbox'], [type='color'], [type='range']),
.adm-field select,
.adm-field textarea {
  width: 100%;
  padding: 0.5rem 0.65rem;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
}

.adm-field input[type='color'] {
  width: 100%;
  height: 2.4rem;
  padding: 0.15rem;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
}

.adm-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  grid-column: 1 / -1;
}

.adm-btn {
  padding: 0.5rem 1rem;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  cursor: pointer;
}

.adm-btn--primary {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--accent-text);
  font-weight: 600;
}

.adm-btn--danger {
  color: var(--danger);
}

.adm-btn--small {
  padding: 0.25rem 0.6rem;
  font-size: 0.85rem;
}

.adm-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.adm-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem 0.75rem;
  padding: 0.65rem 0;
  border-top: 1px solid var(--border);
}

.adm-row:first-child {
  border-top: 0;
}

.adm-row__main {
  flex: 1;
  min-width: 12rem;
}

.adm-row__sub {
  display: block;
  font-size: 0.85rem;
  color: var(--muted);
}

.adm-dot {
  display: inline-block;
  width: 0.7rem;
  height: 0.7rem;
  border-radius: 50%;
  background: var(--c, var(--muted));
}

.adm-badge {
  padding: 0.1rem 0.5rem;
  font-size: 0.75rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--muted);
}

.adm-muted {
  color: var(--muted);
}

.adm-error {
  color: var(--danger);
}

.adm-done {
  text-decoration: line-through;
  color: var(--muted);
}

.adm-section-title {
  margin: 1.25rem 0 0.25rem;
  font-size: 0.95rem;
  color: var(--muted);
}
</style>
