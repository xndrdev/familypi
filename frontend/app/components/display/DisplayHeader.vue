<script setup lang="ts">
defineProps<{ now: Date, connected: boolean, error: string }>()

function greeting(d: Date): string {
  const h = d.getHours()
  if (h < 5) return 'Gute Nacht'
  if (h < 11) return 'Guten Morgen'
  if (h < 17) return 'Hallo zusammen'
  if (h < 22) return 'Guten Abend'
  return 'Gute Nacht'
}
</script>

<template>
  <header class="d-header">
    <div class="d-header__time">{{ formatTime(now) }}</div>
    <div class="d-header__date">
      <div class="d-header__greeting">{{ greeting(now) }}</div>
      <div>{{ formatLongDate(now) }}</div>
    </div>
    <slot />
    <div class="d-header__status" role="status">
      <span v-if="error" class="d-header__error">⚠ {{ error }}</span>
      <span v-else-if="!connected" class="d-header__offline">Verbindung wird aufgebaut …</span>
    </div>
  </header>
</template>

<style>
.d-header {
  position: relative;
  display: flex;
  align-items: center;
  gap: 1.6vw;
}

.d-header__time {
  font-size: 3.6em;
  font-weight: 700;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.d-header__date {
  margin-right: auto;
  white-space: nowrap;
  font-size: 1.2em;
  color: var(--d-muted);
}

.d-header__greeting {
  font-size: 1.25em;
  font-weight: 600;
  color: var(--d-text);
}

.d-header__status {
  position: absolute;
  top: 0;
  right: 0;
  font-size: 0.8em;
}

.d-header__error {
  color: #ff8a8a;
}

.d-header__offline {
  color: var(--d-muted);
}
</style>
