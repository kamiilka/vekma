<template>
  <Teleport to="body">
    <div
      v-if="visible"
      ref="dropEl"
      :style="style"
      class="anchored-dropdown"
    >
      <slot />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onBeforeUnmount } from 'vue';

const props = defineProps<{
  visible: boolean;
  anchorEl: HTMLElement | null;
  maxHeight?: number;
}>();

const dropEl = ref<HTMLElement | null>(null);
const rect = ref({ top: 0, left: 0, width: 0, bottom: 0 });

function updateRect() {
  if (!props.anchorEl) return;
  const r = props.anchorEl.getBoundingClientRect();
  rect.value = { top: r.bottom, left: r.left, width: r.width, bottom: r.bottom };
}

const style = computed(() => {
  const maxH = props.maxHeight ?? 220;
  const spaceBelow = window.innerHeight - rect.value.top;
  const showAbove = spaceBelow < 120 && rect.value.top > maxH;
  const top = showAbove
    ? `${rect.value.top - rect.value.width /* approx height */ - 8}px`
    : `${rect.value.top + 4}px`;
  return {
    position: 'fixed' as const,
    top,
    left: `${rect.value.left}px`,
    width: `${rect.value.width}px`,
    maxHeight: `${maxH}px`,
    zIndex: 9999,
    overflowY: 'auto' as const,
  };
});

watch(() => props.visible, async (val) => {
  if (val) {
    updateRect();
    await nextTick();
    // also re-measure after render
    updateRect();
  }
});

// Update on scroll/resize
function onScroll() { if (props.visible) updateRect(); }
window.addEventListener('scroll', onScroll, true);
window.addEventListener('resize', onScroll);
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll, true);
  window.removeEventListener('resize', onScroll);
});
</script>

<style scoped>
.anchored-dropdown {
  background: var(--bg-card, #1e2a23);
  border: 1px solid var(--border, rgba(122,185,154,0.25));
  border-radius: var(--radius-sm, 6px);
  box-shadow: 0 8px 32px rgba(0,0,0,0.35), 0 2px 8px rgba(0,0,0,0.2);
}
</style>
