<script setup lang="ts">
import type { Dashboard } from '~/types'

defineProps<{ goals: Dashboard['goals'] }>()
</script>

<template>
  <section class="panel d-goals" aria-label="Nächstes großes Ziel pro Familienmitglied">
    <h2 class="panel__title">Unsere nächsten großen Ziele</h2>
    <div class="d-goals__list">
      <article v-for="{ member, goal } in goals" :key="member.id" class="d-goals__card" :style="{ '--c': member.color }">
        <header class="d-goals__who">
          <span class="d-goals__avatar" aria-hidden="true">{{ member.emoji || member.name.charAt(0) }}</span>
          {{ member.name }}
        </header>
        <template v-if="goal">
          <p class="d-goals__title">{{ goal.title }}</p>
          <div class="d-goals__bar" role="progressbar" :aria-valuenow="goal.progress" aria-valuemin="0" aria-valuemax="100">
            <span :style="{ width: goal.progress + '%' }" />
          </div>
          <p class="d-goals__meta">
            {{ goal.progress }} %
            <template v-if="goal.targetDate"> · bis {{ formatShortDate(goal.targetDate) }} ({{ relativeDays(goal.targetDate) }})</template>
          </p>
        </template>
        <p v-else class="d-goals__none">Noch kein Ziel – Zeit für ein neues!</p>
      </article>
      <p v-if="!goals.length" class="d-goals__none">Lege in der Verwaltung die Familienmitglieder an.</p>
    </div>
  </section>
</template>

<style>
.d-goals__list {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(12vw, 1fr));
  gap: 0.8vw;
}

.d-goals__card {
  display: flex;
  flex-direction: column;
  gap: 0.4vw;
  padding: 0.9vw;
  background: var(--d-surface-2);
  border-top: 0.3vw solid var(--c);
  border-radius: 0.7vw;
  overflow: hidden;
}

.d-goals__who {
  display: flex;
  align-items: center;
  gap: 0.5vw;
  font-weight: 700;
}

.d-goals__avatar {
  display: grid;
  place-items: center;
  width: 2em;
  height: 2em;
  border-radius: 50%;
  background: color-mix(in srgb, var(--c) 30%, transparent);
}

.d-goals__title {
  margin: 0;
  font-size: 1.15em;
  font-weight: 600;
  line-height: 1.25;
}

.d-goals__bar {
  height: 0.6vw;
  min-height: 6px;
  margin-top: auto;
  background: var(--d-border);
  border-radius: 999px;
  overflow: hidden;
}

.d-goals__bar span {
  display: block;
  height: 100%;
  background: var(--c);
  transition: width 1s ease;
}

.d-goals__meta,
.d-goals__none {
  margin: 0;
  font-size: 0.8em;
  color: var(--d-muted);
}
</style>
