<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  id: string
  label: string
  placeholder: string
  tone?: 'good_actions' | 'bad_actions'
}>()

const values = defineModel<string[]>({ required: true })
const input = ref('')

function addValue() {
  const value = input.value.trim()
  if (!value) return
  values.value = [...new Set([...values.value, value])]
  input.value = ''
}

function removeValue(value: string) {
  values.value = values.value.filter((item) => item !== value)
}
</script>

<template>
  <div class="chip-editor" :data-chip-list="props.tone">
    <label :for="props.id">
      <slot name="icon" />{{ props.label }}
    </label>
    <div class="chip-list">
      <span v-for="value in values" :key="value" class="chip">
        {{ value }}
        <button type="button" :aria-label="`删除 ${value}`" @click="removeValue(value)">×</button>
      </span>
    </div>
    <input
      :id="props.id"
      v-model="input"
      type="text"
      :placeholder="props.placeholder"
      autocomplete="off"
      @keydown.enter.prevent="addValue"
    />
  </div>
</template>
