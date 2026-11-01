<script setup lang="ts">
import type { CalendarEvent, Member } from '~/types'

const props = defineProps<{ events: CalendarEvent[], members: Map<number, Member>, now: Date }>()

const FAMILY_COLOR = '#ffcf5c'

const days = computed(() => {
  const today = startOfDay(props.now)
  return Array.from({ length: 7 }, (_, i) => {
    const date = addDays(today, i)
    return { date, key: isoDate(date), events: eventsOnDay(props.events, date) }
  })
})

function color(e: CalendarEvent): string {
  return (e.memberId && props.members.get(e.memberId)?.color) || FAMILY_COLOR
}

function who(e: CalendarEvent): string {
  const m = e.memberId ? props.members.get(e.memberId) : undefined
  return m ? (m.emoji || m.name.charAt(0)) : '👪'
}

function isPast(e: CalendarEvent): boolean {
  return !e.allDay && new Date(e.end) < props.now
}

function label(i: number, d: Date): string {
  if (i === 0) return 'Heute'
  if (i === 1) return 'Morgen'
  return weekdayName(d)
}
</script>

<template>
  <section class="panel d-week" aria-label="Termine der nächsten sieben Tage">
    <h2 class="panel__title">Kalender</h2>
    <div class="d-week__days">
      <div v-for="(day, i) in days" :key="day.key" class="d-week__day" :class="{ 'd-week__day--today': i === 0 }">
        <div class="d-week__head">
          <span class="d-week__name">{{ label(i, day.date) }}</span>
          <span class="d-week__num">{{ day.date.getDate() }}.{{ day.date.getMonth() + 1 }}.</span>
        </div>
        <ul class="d-week__list">
          <li
            v-for="e in day.events"
            :key="e.uid"
            class="d-week__event"
            :class="{ 'd-week__event--allday': e.allDay, 'd-week__event--past': isPast(e) }"
            :style="{ '--c': color(e) }"
          >
            <span class="d-week__meta">
              <span aria-hidden="true">{{ who(e) }}</span>
              <span v-if="!e.allDay" class="d-week__time">{{ formatTime(e.start) }}</span>
            </span>
            <span class="d-week__text">{{ e.title }}</span>
          </li>
          <li v-if="!day.events.length" class="d-week__empty">–</li>
        </ul>
      </div>
    </div>
  </section>
</template>

<style>
.d-week__days {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 0.6vw;
}

.d-week__day {
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0.6vw;
  background: var(--d-surface-2);
  border-radius: 0.7vw;
  overflow: hidden;
}

.d-week__day--today {
  outline: 2px solid var(--d-accent);
}

.d-week__head {
  display: flex;
  flex-direction: column;
  margin-bottom: 0.5vw;
}

.d-week__name {
  font-weight: 700;
}

.d-week__day--today .d-week__name {
  color: var(--d-accent);
}

.d-week__num {
  font-size: 0.8em;
  color: var(--d-muted);
}

.d-week__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.35vw;
  overflow: hidden;
}

.d-week__event {
  display: flex;
  flex-direction: column;
  gap: 0.1vw;
  padding: 0.3vw 0.4vw;
  font-size: 0.8em;
  line-height: 1.25;
  border-left: 0.25vw solid var(--c);
  background: color-mix(in srgb, var(--c) 12%, transparent);
  border-radius: 0.3vw;
}

.d-week__event--allday {
  background: color-mix(in srgb, var(--c) 32%, transparent);
  font-weight: 600;
}

.d-week__event--past {
  opacity: 0.45;
}

.d-week__text {
  min-width: 0;
  overflow-wrap: break-word;
  hyphens: auto;
}

.d-week__meta {
  display: flex;
  gap: 0.35vw;
}

.d-week__time {
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.d-week__empty {
  color: var(--d-muted);
}

@media (orientation: portrait) {
  .d-week__days {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
