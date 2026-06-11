<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { api } from "../api";
import type { UserSession, DocumentRegistryItem } from "../types";

const props = defineProps<{ session: UserSession }>();
const emit  = defineEmits<{ (e: "session-expired"): void }>();
const token = computed(() => props.session.token);

// ── State ──────────────────────────────────────────────────────────────────
const query        = ref("");
const loading      = ref(false);
const error        = ref("");
const results      = ref<DocumentRegistryItem[]>([]);
const searched     = ref(false);

// Type filter chips
type DocTypeKey = "customer_order"|"supplier_order"|"purchase"|"sale"|"service_order"|"payment_in"|"payment_out"|"transfer"|"inventory"|"document";

const ALL_TYPES: { key: DocTypeKey; label: string; icon: string; color: string }[] = [
  { key: "customer_order",  label: "Замовлення покупця",       icon: "📋", color: "rgba(72,187,120,0.15)"  },
  { key: "supplier_order",  label: "Замовлення постачальнику", icon: "📦", color: "rgba(100,160,220,0.15)" },
  { key: "purchase",        label: "Надходження",              icon: "🚚", color: "rgba(100,160,220,0.12)" },
  { key: "sale",            label: "Продаж",                   icon: "🧾", color: "rgba(72,187,120,0.12)"  },
  { key: "service_order",   label: "Ремонт",                   icon: "⚙️",  color: "rgba(180,140,80,0.15)"  },
  { key: "payment_in",      label: "ПКО (прибутковий)",        icon: "💰", color: "rgba(72,187,120,0.18)"  },
  { key: "payment_out",     label: "ВКО (видатковий)",         icon: "💸", color: "rgba(224,112,112,0.15)" },
  { key: "transfer",        label: "Переміщення",              icon: "🔄", color: "rgba(160,120,220,0.15)" },
  { key: "inventory",       label: "Інвентаризація",           icon: "📊", color: "rgba(120,180,200,0.15)" },
  { key: "document",        label: "Повернення / Накладна",    icon: "📄", color: "rgba(180,180,180,0.15)" },
];

const activeTypes = ref<Set<DocTypeKey>>(new Set());

function toggleType(key: DocTypeKey) {
  if (activeTypes.value.has(key)) activeTypes.value.delete(key);
  else activeTypes.value.add(key);
  activeTypes.value = new Set(activeTypes.value); // trigger reactivity
}
function clearTypes() { activeTypes.value = new Set(); }
function allActive()  { return activeTypes.value.size === 0; }

// ── Search ─────────────────────────────────────────────────────────────────
let debounceTimer: ReturnType<typeof setTimeout> | null = null;

async function doSearch() {
  loading.value = true; error.value = "";
  try {
    const types = activeTypes.value.size > 0 ? [...activeTypes.value] : undefined;
    results.value = await api.searchDocuments(token.value, query.value, types, 150);
    searched.value = true;
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

watch(query, () => {
  if (debounceTimer) clearTimeout(debounceTimer);
  debounceTimer = setTimeout(doSearch, 300);
});
watch(activeTypes, doSearch, { deep: false });

onMounted(doSearch); // load recent docs on open

// ── Grouping ───────────────────────────────────────────────────────────────
const grouped = computed(() => {
  const map = new Map<string, DocumentRegistryItem[]>();
  for (const item of results.value) {
    const list = map.get(item.docType) || [];
    list.push(item);
    map.set(item.docType, list);
  }
  return map;
});

function typeInfo(key: string) {
  return ALL_TYPES.find(t => t.key === key) ?? { key, label: key, icon: "📄", color: "rgba(180,180,180,0.1)" };
}

// ── Helpers ────────────────────────────────────────────────────────────────
const statusLabel: Record<string, string> = {
    new: "Нове", in_work: "В роботі", ordered: "Замовлено у пост.",
  expected: "Очікується", arrived: "Надійшло", issued: "Видано",
  closed: "Закрито", cancelled: "Скасовано",
  draft: "Створено", sent: "Відправлено", confirmed: "Підтверджено",
  in_transit: "В дорозі", received: "Отримано",
  new_repair: "Нове", in_progress: "В роботі", done: "Готово", issued_repair: "Видано",
  pending: "Очікує", applied: "Проведено", posted: "Проведено",
  customer_return: "Повернення від покупця", supplier_return: "Повернення постачальнику",
};

const statusColor: Record<string, string> = {
    new: "#94a3b8", in_work: "#60a5fa", ordered: "#a78bfa",
  expected: "#f59e0b", arrived: "#34d399", issued: "#10b981",
  closed: "#6b7280", cancelled: "#f87171",
  draft: "#94a3b8", sent: "#60a5fa", confirmed: "#818cf8",
  in_transit: "#f59e0b", received: "#34d399",
};

function fmt(n?: number) { return n ? n.toLocaleString("uk-UA", { maximumFractionDigits: 2 }) : ""; }
function fmtDate(s: string) { return new Date(s).toLocaleDateString("uk-UA"); }
</script>

<template>
  <div class="page-content">

    <!-- ── Header ── -->
    <div class="page-header" style="margin-bottom:1.25rem">
      <div>
        <h2 style="margin:0 0 0.2rem">Реєстр документів</h2>
        <div class="subtle" style="font-size:0.8rem">Єдиний пошук по всіх документах системи</div>
      </div>
    </div>

    <!-- ── Search bar ── -->
    <div style="margin-bottom:1rem">
      <div style="position:relative">
        <span style="position:absolute;left:0.85rem;top:50%;transform:translateY(-50%);font-size:1rem;opacity:0.5">🔍</span>
        <input
          v-model="query"
          placeholder="Пошук по назві товару, контрагенту, примітці, номеру документа..."
          style="width:100%;padding:0.65rem 1rem 0.65rem 2.5rem;font-size:0.95rem;box-sizing:border-box"
          autofocus
        >
      </div>
    </div>

    <!-- ── Type filter chips ── -->
    <div style="display:flex;flex-wrap:wrap;gap:0.4rem;margin-bottom:1.25rem;align-items:center">
      <span class="subtle" style="font-size:0.72rem;margin-right:0.25rem">Тип:</span>
      <button
        :class="['chip', allActive() && 'chip--active']"
        :style="allActive() ? 'border-color:var(--accent);color:var(--accent);background:rgba(72,187,120,0.12)' : ''"
        @click="clearTypes()"
        style="cursor:pointer;font-size:0.75rem;padding:0.2rem 0.65rem"
      >Всі</button>
      <button
        v-for="t in ALL_TYPES" :key="t.key"
        :class="['chip']"
        :style="activeTypes.has(t.key)
          ? `border-color:var(--accent);color:var(--accent);background:${t.color};cursor:pointer;font-size:0.75rem;padding:0.2rem 0.65rem`
          : `cursor:pointer;font-size:0.75rem;padding:0.2rem 0.65rem;opacity:0.65`"
        @click="toggleType(t.key)"
      >{{ t.icon }} {{ t.label }}</button>
    </div>

    <!-- ── Loading / error ── -->
    <p v-if="loading" class="subtle" style="text-align:center;padding:2rem">Пошук...</p>
    <p v-else-if="error" class="error-text">{{ error }}</p>

    <!-- ── Results ── -->
    <div v-else-if="searched">

      <!-- Summary bar -->
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.75rem">
        <div class="subtle" style="font-size:0.8rem">
          <span v-if="query">Результати для «<strong>{{ query }}</strong>»: </span>
          <span v-else>Останні документи: </span>
          <strong>{{ results.length }}</strong> знайдено
        </div>
        <div class="subtle" style="font-size:0.75rem">{{ grouped.size }} типів документів</div>
      </div>

      <!-- Empty state -->
      <div v-if="!results.length"
        style="text-align:center;padding:3rem;color:var(--text-subtle)">
        <div style="font-size:2.5rem;margin-bottom:0.75rem">🔍</div>
        <div style="font-size:0.95rem;margin-bottom:0.4rem">Документів не знайдено</div>
        <div style="font-size:0.8rem">Спробуйте інший запит або очистіть фільтр типів</div>
      </div>

      <!-- Grouped by type -->
      <template v-for="[docType, items] in grouped" :key="docType">
        <div style="margin-bottom:1.5rem">
          <!-- Group header -->
          <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.5rem;padding:0.4rem 0.6rem;border-radius:7px"
            :style="`background:${typeInfo(docType).color}`">
            <span style="font-size:1.1rem">{{ typeInfo(docType).icon }}</span>
            <span style="font-weight:600;font-size:0.88rem">{{ typeInfo(docType).label }}</span>
            <span class="chip" style="font-size:0.7rem;margin-left:auto">{{ items.length }}</span>
          </div>

          <!-- Items grid -->
          <div style="display:grid;gap:0.4rem">
            <div
              v-for="item in items" :key="item.docType+item.id"
              class="panel"
              style="padding:0.7rem 0.9rem;display:grid;grid-template-columns:1fr auto;gap:0.3rem 1rem;align-items:start"
            >
              <!-- Left: number + meta -->
              <div>
                <div style="display:flex;align-items:center;gap:0.5rem;flex-wrap:wrap;margin-bottom:0.15rem">
                  <span style="font-weight:600;font-size:0.9rem">{{ item.number }}</span>
                  <span v-if="item.status"
                    class="chip"
                    :style="`font-size:0.68rem;background:${(statusColor[item.status]||'#888')}22;color:${statusColor[item.status]||'#888'};border-color:${statusColor[item.status]||'#888'}44`">
                    {{ statusLabel[item.status] || item.status }}
                  </span>
                </div>

                <!-- Counter name -->
                <div v-if="item.counterName" class="subtle" style="font-size:0.78rem;margin-bottom:0.1rem">
                  👤 {{ item.counterName }}
                </div>

                <!-- Product hits — what matched -->
                <div v-if="item.productHits?.length"
                  style="font-size:0.75rem;color:#6db8f0;margin-top:0.15rem;display:flex;flex-wrap:wrap;gap:0.3rem">
                  <span style="color:var(--text-subtle)">Товари:</span>
                  <span v-for="h in item.productHits.slice(0,5)" :key="h"
                    style="background:rgba(100,160,220,0.12);border-radius:4px;padding:0.05rem 0.35rem;border:1px solid rgba(100,160,220,0.25)">
                    {{ h }}
                  </span>
                  <span v-if="item.productHits.length > 5" class="subtle">+{{ item.productHits.length - 5 }}</span>
                </div>

                <!-- Note -->
                <div v-if="item.note && item.note !== item.number"
                  class="subtle" style="font-size:0.75rem;margin-top:0.15rem;font-style:italic">
                  {{ item.note.slice(0, 80) }}{{ item.note.length > 80 ? '…' : '' }}
                </div>
              </div>

              <!-- Right: amount + date -->
              <div style="text-align:right;flex-shrink:0">
                <div v-if="item.total" style="font-weight:700;font-size:0.9rem;white-space:nowrap"
                  :style="item.docType==='payment_out' ? 'color:#e07070' : 'color:var(--accent)'">
                  {{ item.docType==='payment_out' ? '−' : '' }}{{ fmt(item.total) }} {{ item.currency }}
                </div>
                <div class="subtle" style="font-size:0.72rem;margin-top:0.1rem">{{ fmtDate(item.date) }}</div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>

  </div>
</template>
