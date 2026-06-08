<script setup lang="ts">
/**
 * BarcodeScanner — universal barcode input.
 * Modes:
 *  1. USB/keyboard-wedge: rapid keystrokes ending in Enter (< 50ms gap)
 *  2. Manual text input (real-time, v-model compatible)
 *  3. Camera (via @zxing/library BrowserMultiFormatReader)
 */
import { ref, watch, onMounted, onBeforeUnmount } from "vue";

const emit = defineEmits<{
  (e: "scanned", barcode: string): void;
  (e: "update:modelValue", value: string): void;
}>();
const props = defineProps<{
  placeholder?: string;
  autofocus?: boolean;
  compact?: boolean;
  modelValue?: string;
}>();

const manualCode   = ref(props.modelValue ?? "");
const cameraActive = ref(false);
const cameraError  = ref("");
const videoRef     = ref<HTMLVideoElement | null>(null);
const lastKeyTime  = ref(0);
const keyBuffer    = ref("");

const USB_MS  = 50;
const MIN_LEN = 1;

// Sync external modelValue → local input
watch(() => props.modelValue, (val) => {
  if (val !== undefined && val !== manualCode.value) manualCode.value = val;
});

// Real-time: emit update on every keystroke
function onInput() {
  emit("update:modelValue", manualCode.value);
  emit("scanned", manualCode.value);
}

// ── keyboard-wedge listener ──────────────────────────────────────────────────
function onGlobalKey(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement)?.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
  const now = Date.now();
  const delta = now - lastKeyTime.value;
  lastKeyTime.value = now;
  if (e.key === "Enter") {
    if (keyBuffer.value.length >= MIN_LEN) {
      manualCode.value = keyBuffer.value;
      emit("scanned", keyBuffer.value);
      emit("update:modelValue", keyBuffer.value);
    }
    keyBuffer.value = "";
    return;
  }
  if (e.key.length === 1) {
    keyBuffer.value = (delta < USB_MS || keyBuffer.value.length > 0)
      ? keyBuffer.value + e.key
      : e.key;
  }
}

// ── manual submit (Enter key or button) ──────────────────────────────────────
function submit() {
  const c = manualCode.value.trim();
  if (c) { emit("scanned", c); emit("update:modelValue", c); }
}
function onKey(e: KeyboardEvent) { if (e.key === "Enter") { e.preventDefault(); submit(); } }

// ── camera ───────────────────────────────────────────────────────────────────
let codeReader: any = null;

async function startCamera() {
  cameraError.value = "";
  cameraActive.value = true;
  await new Promise(r => setTimeout(r, 80));
  try {
    const { BrowserMultiFormatReader } = await import("@zxing/library");
    codeReader = new BrowserMultiFormatReader();
    const devices = await codeReader.listVideoInputDevices();
    const deviceId = devices.find((d: any) =>
      d.label.toLowerCase().includes("back") ||
      d.label.toLowerCase().includes("rear") ||
      d.label.toLowerCase().includes("environment")
    )?.deviceId ?? devices[0]?.deviceId;

    if (!deviceId) { cameraError.value = "Камеру не знайдено. Скористайтесь USB-сканером або введіть код вручну."; stopCamera(); return; }

    await codeReader.decodeFromVideoDevice(deviceId, videoRef.value!, (result: any, err: any) => {
      if (result) {
        emit("scanned", result.getText());
        stopCamera();
      }
    });
  } catch (err: any) {
    cameraError.value = err?.message ?? "Помилка камери";
    stopCamera();
  }
}

function stopCamera() {
  codeReader?.reset();
  codeReader = null;
  cameraActive.value = false;
}

// ── lifecycle ────────────────────────────────────────────────────────────────
onMounted(() => { window.addEventListener("keydown", onGlobalKey); });
onBeforeUnmount(() => { window.removeEventListener("keydown", onGlobalKey); stopCamera(); });
</script>

<template>
  <div>
    <!-- Input row -->
    <div style="display:flex;gap:0.4rem;align-items:center">
      <input
        v-model="manualCode"
        :placeholder="placeholder ?? 'Штрихкод або назва товару...'"
        :autofocus="autofocus"
        style="flex:1"
        @input="onInput"
        @keydown="onKey"
      />
      <button
        v-if="manualCode"
        class="ghost-button"
        style="padding:0.45rem 0.6rem;font-size:0.9rem"
        @click="manualCode=''; emit('scanned',''); emit('update:modelValue','')"
        title="Очистити"
      >✕</button>
      <button
        class="ghost-button"
        style="padding:0.45rem 0.9rem;font-size:0.82rem;white-space:nowrap"
        @click="submit"
        title="Знайти товар (або натисніть Enter)"
      >
        Пошук
      </button>
      <button
        class="ghost-button"
        style="padding:0.45rem 0.9rem;font-size:0.82rem;white-space:nowrap"
        :style="cameraActive ? 'border-color:rgba(72,187,120,0.6);color:#6dd4a0;background:rgba(72,187,120,0.08)' : ''"
        @click="cameraActive ? stopCamera() : startCamera()"
        :title="cameraActive ? 'Зупинити камеру' : 'Сканувати штрихкод камерою'"
      >
        {{ cameraActive ? 'Зупинити камеру' : 'Камера' }}
      </button>
    </div>

    <!-- Usage hint -->
    <div v-if="!cameraActive && !cameraError" style="font-size:0.75rem;color:#6a9e84;margin-top:0.35rem;line-height:1.5">
      Введіть назву або штрихкод і натисніть Enter &nbsp;·&nbsp; Або підключіть USB-сканер і скануйте одразу &nbsp;·&nbsp; Або натисніть "Камера"
    </div>

    <!-- Camera view -->
    <div v-if="cameraActive" style="margin-top:0.7rem;position:relative">
      <div style="font-size:0.82rem;font-weight:600;color:#6dd4a0;margin-bottom:0.4rem">
        Наведіть камеру на штрихкод товару
      </div>
      <video
        ref="videoRef"
        style="width:100%;max-width:420px;border-radius:0.8rem;border:2px solid rgba(72,187,120,0.4);display:block"
        autoplay muted playsinline
      />
      <!-- Scanning overlay -->
      <div style="position:absolute;top:28px;left:0;right:0;max-width:420px;bottom:0;display:flex;align-items:center;justify-content:center;pointer-events:none">
        <div style="width:70%;height:28%;border:2px solid rgba(72,187,120,0.85);border-radius:0.4rem;box-shadow:0 0 0 9999px rgba(0,0,0,0.38)"/>
      </div>
      <div style="margin-top:0.5rem;display:flex;align-items:center;gap:0.6rem">
        <span style="font-size:0.8rem;color:#aaa">Сканування…</span>
        <button class="ghost-button" style="padding:0.3rem 0.8rem;font-size:0.82rem" @click="stopCamera">Закрити камеру</button>
      </div>
    </div>

    <p v-if="cameraError" style="font-size:0.82rem;margin-top:0.4rem;color:#e07070">
      {{ cameraError }}
    </p>
  </div>
</template>
