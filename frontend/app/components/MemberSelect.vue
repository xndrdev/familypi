<script setup lang="ts">
import type { Member } from '~/types'

withDefaults(defineProps<{ members: Member[], label?: string, allowFamily?: boolean }>(), {
  label: 'Für',
  allowFamily: true,
})
const model = defineModel<number | null>({ required: true })
</script>

<template>
  <label class="adm-field">
    <span>{{ label }}</span>
    <select v-model="model">
      <option v-if="allowFamily" :value="null">👪 Familie</option>
      <option v-for="m in members" :key="m.id" :value="m.id">{{ m.emoji }} {{ m.name }}</option>
    </select>
  </label>
</template>
