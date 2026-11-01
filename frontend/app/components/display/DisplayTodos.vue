<script setup lang="ts">
import type { Member, Todo } from '~/types'

const props = defineProps<{ todos: Todo[], members: Map<number, Member> }>()

// Gruppiert nach Familie zuerst, dann in der Reihenfolge der Mitglieder.
const groups = computed(() => {
  const result: { key: string, name: string, emoji: string, color: string, todos: Todo[] }[] = []
  const family = props.todos.filter(t => t.memberId === null)
  if (family.length) result.push({ key: 'family', name: 'Familie', emoji: '👪', color: '#ffcf5c', todos: family })
  for (const m of props.members.values()) {
    const own = props.todos.filter(t => t.memberId === m.id)
    if (own.length) result.push({ key: String(m.id), name: m.name, emoji: m.emoji, color: m.color, todos: own })
  }
  return result
})

function dueClass(t: Todo): string {
  if (!t.due || t.done) return ''
  const n = daysUntil(t.due)
  if (n < 0) return 'd-todos__due--over'
  if (n === 0) return 'd-todos__due--today'
  return ''
}
</script>

<template>
  <section class="panel d-todos" aria-label="Offene Todos">
    <h2 class="panel__title">Todos</h2>
    <div class="d-todos__scroll">
      <div v-for="g in groups" :key="g.key" class="d-todos__group" :style="{ '--c': g.color }">
        <h3 class="d-todos__who">
          <span aria-hidden="true">{{ g.emoji }}</span> {{ g.name }}
        </h3>
        <ul class="d-todos__list">
          <li v-for="t in g.todos" :key="t.id" class="d-todos__item" :class="{ 'd-todos__item--done': t.done }">
            <span class="d-todos__box" aria-hidden="true">{{ t.done ? '✓' : '' }}</span>
            <span class="d-todos__title">{{ t.title }}</span>
            <span v-if="t.due && !t.done" class="d-todos__due" :class="dueClass(t)">{{ relativeDays(t.due) }}</span>
          </li>
        </ul>
      </div>
      <p v-if="!groups.length" class="d-todos__empty">🎉 Alles erledigt!</p>
    </div>
  </section>
</template>

<style>
.d-todos__scroll {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 1vw;
}

.d-todos__who {
  margin: 0 0 0.4vw;
  font-size: 0.95em;
  color: var(--c);
}

.d-todos__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.4vw;
}

.d-todos__item {
  display: flex;
  align-items: baseline;
  gap: 0.6vw;
}

.d-todos__box {
  flex: none;
  display: inline-grid;
  place-items: center;
  width: 1.1em;
  height: 1.1em;
  font-size: 0.85em;
  border: 2px solid var(--c);
  border-radius: 0.3em;
  transform: translateY(0.15em);
}

.d-todos__item--done .d-todos__box {
  background: var(--c);
  color: var(--d-bg);
}

.d-todos__item--done .d-todos__title {
  text-decoration: line-through;
  color: var(--d-muted);
}

.d-todos__title {
  min-width: 0;
  overflow-wrap: break-word;
  hyphens: auto;
}

.d-todos__due {
  margin-left: auto;
  flex: none;
  font-size: 0.75em;
  color: var(--d-muted);
}

.d-todos__due--today {
  color: var(--d-accent);
  font-weight: 700;
}

.d-todos__due--over {
  color: #ff8a8a;
  font-weight: 700;
}

.d-todos__empty {
  margin: auto;
  font-size: 1.3em;
  color: var(--d-muted);
}
</style>
