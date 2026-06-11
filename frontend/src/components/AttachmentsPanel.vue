<template>
  <div class="attachments-panel">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.6rem">
      <strong style="font-size:0.9rem">📎 Вкладення</strong>
      <label class="ghost-button" style="padding:0.3rem 0.7rem;font-size:0.8rem;cursor:pointer" :class="{disabled: uploading}">
        {{ uploading ? 'Завантаження...' : '+ Додати' }}
        <input type="file" style="display:none" :accept="acceptTypes" @change="onFileSelected" :disabled="uploading" multiple>
      </label>
    </div>

    <!-- Error -->
    <p v-if="error" style="color:#e06060;font-size:0.8rem;margin:0 0 0.5rem">{{ error }}</p>

    <!-- Empty state -->
    <div v-if="!items.length && !loading" style="font-size:0.82rem;color:#888;padding:0.5rem 0">
      Немає вкладень
    </div>

    <!-- Loading -->
    <div v-if="loading" style="font-size:0.82rem;color:#7ab99a;padding:0.5rem 0">Завантаження...</div>

    <!-- List -->
    <div v-for="item in items" :key="item.id"
      style="display:flex;align-items:center;gap:0.5rem;padding:0.4rem 0;border-bottom:1px solid rgba(255,255,255,0.05)">
      <!-- Icon -->
      <span style="font-size:1.1rem;min-width:1.4rem;text-align:center">{{ fileIcon(item.mimeType, item.fileName) }}</span>
      <!-- Name + meta -->
      <div style="flex:1;min-width:0">
        <div style="font-size:0.85rem;font-weight:500;white-space:nowrap;overflow:hidden;text-overflow:ellipsis" :title="item.fileName">
          {{ item.fileName }}
        </div>
        <div style="font-size:0.72rem;color:#888">
          {{ formatSize(item.sizeBytes) }} · {{ formatDate(item.createdAt) }}
        </div>
      </div>
      <!-- Actions -->
      <button class="ghost-button" style="padding:0.2rem 0.55rem;font-size:0.75rem" @click="download(item)" title="Завантажити">⬇</button>
      <button v-if="canDelete" class="ghost-button" style="padding:0.2rem 0.55rem;font-size:0.75rem;color:#e06060;border-color:rgba(224,96,96,0.35)"
        @click="remove(item)" title="Видалити">✕</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { api } from '../api';
import type { AttachmentItem } from '../types';

const props = defineProps<{
  token: string;
  entityType: string;
  entityId: number;
  canDelete?: boolean;
}>();

const acceptTypes = 'image/*,application/pdf,.pdf,.jpg,.jpeg,.png,.gif,.webp,.doc,.docx,.xls,.xlsx';
const items = ref<AttachmentItem[]>([]);
const loading = ref(false);
const uploading = ref(false);
const error = ref('');

async function load() {
  if (!props.entityId) return;
  loading.value = true;
  error.value = '';
  try {
    items.value = await api.listAttachments(props.token, props.entityType, props.entityId);
  } catch (e: any) {
    error.value = e.message ?? 'Помилка завантаження';
  } finally {
    loading.value = false;
  }
}

async function onFileSelected(e: Event) {
  const input = e.target as HTMLInputElement;
  if (!input.files?.length) return;
  uploading.value = true;
  error.value = '';
  try {
    for (const file of Array.from(input.files)) {
      if (file.size > 10 * 1024 * 1024) {
        error.value = `Файл "${file.name}" перевищує 10 МБ`;
        continue;
      }
      const item = await api.uploadAttachment(props.token, props.entityType, props.entityId, file);
      items.value.push(item);
    }
  } catch (e: any) {
    error.value = e.message ?? 'Помилка завантаження';
  } finally {
    uploading.value = false;
    input.value = '';
  }
}

async function download(item: AttachmentItem) {
  try {
    const blob = await api.downloadAttachment(props.token, item.id);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = item.fileName; a.click();
    setTimeout(() => URL.revokeObjectURL(url), 2000);
  } catch (e: any) {
    error.value = e.message ?? 'Помилка завантаження файлу';
  }
}

async function remove(item: AttachmentItem) {
  if (!confirm(`Видалити "${item.fileName}"?`)) return;
  try {
    await api.deleteAttachment(props.token, item.id);
    items.value = items.value.filter(i => i.id !== item.id);
  } catch (e: any) {
    error.value = e.message ?? 'Помилка видалення';
  }
}

function fileIcon(mimeType: string, fileName: string): string {
  if (mimeType.startsWith('image/')) return '🖼️';
  if (mimeType === 'application/pdf' || fileName.toLowerCase().endsWith('.pdf')) return '📄';
  if (mimeType.includes('word') || fileName.match(/\.docx?$/i)) return '📝';
  if (mimeType.includes('excel') || fileName.match(/\.xlsx?$/i)) return '📊';
  return '📎';
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} Б`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`;
  return `${(bytes / 1024 / 1024).toFixed(1)} МБ`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString('uk', { day: '2-digit', month: 'short', year: 'numeric' });
}

onMounted(load);
watch(() => props.entityId, load);
</script>
