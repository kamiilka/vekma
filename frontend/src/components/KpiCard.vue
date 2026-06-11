<script setup lang="ts">
defineProps<{
  title: string;
  value: string | number;
  hint?: string;
}>();
</script>

<template>
  <article class="kpi-card">
    <p class="kpi-card__title">{{ title }}</p>
    <p class="kpi-card__value">{{ value }}</p>
    <p v-if="hint" class="kpi-card__hint">{{ hint }}</p>
  </article>
</template>
