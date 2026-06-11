<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { api } from "../api";
import type {
  UserSession,
  SupplierReport,
  SupplierReportRow,
  CounterpartyReport,
  CounterpartyReportRow,
} from "../types";

const props = defineProps<{ session: UserSession; initialSubTab?: string }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);

type SubTab = "suppliers" | "counterparties";
const subTab = ref<SubTab>((props.initialSubTab as SubTab) ?? "suppliers");

// ── Data ──────────────────────────────────────────────────────────────────────
const supplierReport = ref<SupplierReport | null>(null);
const counterpartyReport = ref<CounterpartyReport | null>(null);
const loading = ref(false);
const error = ref("");

// ── Sorting ───────────────────────────────────────────────────────────────────
type SortKey = string;
const sortKey = ref<SortKey>("supplierName");
const sortDir = ref<1 | -1>(1);

function setSort(key: SortKey) {
  if (sortKey.value === key) { sortDir.value = sortDir.value === 1 ? -1 : 1; }
  else { sortKey.value = key; sortDir.value = 1; }
}

function sortArrow(key: SortKey) {
  if (sortKey.value !== key) return "↕";
  return sortDir.value === 1 ? "↑" : "↓";
}

// ── Filters ───────────────────────────────────────────────────────────────────
const searchQ = ref("");

const sortedSuppliers = computed(() => {
  if (!supplierReport.value) return [];
  let rows = [...supplierReport.value.rows];
  const q = searchQ.value.toLowerCase().trim();
  if (q) rows = rows.filter(r => r.supplierName.toLowerCase().includes(q));
  rows.sort((a, b) => {
    const av = (a as any)[sortKey.value] ?? "";
    const bv = (b as any)[sortKey.value] ?? "";
    return typeof av === "number"
      ? (av - bv) * sortDir.value
      : String(av).localeCompare(String(bv)) * sortDir.value;
  });
  return rows;
});

const sortedCounterparties = computed(() => {
  if (!counterpartyReport.value) return [];
  let rows = [...counterpartyReport.value.rows];
  const q = searchQ.value.toLowerCase().trim();
  if (q) rows = rows.filter(r => r.counterpartyName.toLowerCase().includes(q));
  rows.sort((a, b) => {
    const av = (a as any)[sortKey.value] ?? "";
    const bv = (b as any)[sortKey.value] ?? "";
    return typeof av === "number"
      ? (av - bv) * sortDir.value
      : String(av).localeCompare(String(bv)) * sortDir.value;
  });
  return rows;
});

// ── Load ──────────────────────────────────────────────────────────────────────
async function loadSuppliers() {
  if (supplierReport.value) return;
  loading.value = true; error.value = "";
  try {
    supplierReport.value = await api.supplierReport(token.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

async function loadCounterparties() {
  if (counterpartyReport.value) return;
  loading.value = true; error.value = "";
  try {
    counterpartyReport.value = await api.counterpartyReport(token.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

async function refresh() {
  if (subTab.value === "suppliers") {
    supplierReport.value = null;
    await loadSuppliers();
  } else {
    counterpartyReport.value = null;
    await loadCounterparties();
  }
}

watch(subTab, (val) => {
  searchQ.value = "";
  sortKey.value = val === "suppliers" ? "supplierName" : "counterpartyName";
  sortDir.value = 1;
  if (val === "suppliers") loadSuppliers();
  else loadCounterparties();
});

onMounted(() => {
  if (subTab.value === "suppliers") loadSuppliers();
  else loadCounterparties();
});

// ── Formatters ────────────────────────────────────────────────────────────────
function fmt(n: number) {
  return n.toLocaleString("uk", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function debtColor(debt: number) {
  if (debt <= 0) return "";
  if (debt > 10000) return "color:#e08080;font-weight:700";
  return "color:#e0c060;font-weight:600";
}
</script>

<template>
  <div class="page-content">
    <!-- Header -->
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Звіти</h2>
      <button class="ghost-button" @click="refresh" :disabled="loading">
        {{ loading ? "Завантаження…" : "↻ Оновити" }}
      </button>
    </div>

    <!-- Tabs -->
    <div class="tab-row" style="margin-bottom:1rem">
      <button
        :class="['tab-button', subTab === 'suppliers' && 'tab-button--active']"
        @click="subTab = 'suppliers'"
      >📦 По постачальниках</button>
      <button
        :class="['tab-button', subTab === 'counterparties' && 'tab-button--active']"
        @click="subTab = 'counterparties'"
      >◎ По контрагентах</button>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження…</p>

    <!-- ══════════════════════════════════════════════════════════
         SUPPLIERS REPORT
    ══════════════════════════════════════════════════════════ -->
    <div v-if="subTab === 'suppliers' && supplierReport && !loading">
      <!-- Summary cards -->
      <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:0.75rem;margin-bottom:1.2rem">
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Постачальників</div>
          <div style="font-size:1.4rem;font-weight:700">{{ supplierReport.rows.length }}</div>
        </div>
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Закуплено (UAH)</div>
          <div style="font-size:1.2rem;font-weight:700;color:#9fe8c4">{{ fmt(supplierReport.totalPurchasedUah) }}</div>
        </div>
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Оплачено (UAH)</div>
          <div style="font-size:1.2rem;font-weight:700;color:#7ab4d4">{{ fmt(supplierReport.totalPaidUah) }}</div>
        </div>
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Борг (UAH)</div>
          <div style="font-size:1.2rem;font-weight:700" :style="debtColor(supplierReport.totalDebtUah)">
            {{ fmt(supplierReport.totalDebtUah) }}
          </div>
        </div>
      </div>

      <!-- Search -->
      <input
        v-model="searchQ"
        placeholder="Пошук постачальника…"
        style="margin-bottom:0.8rem;max-width:300px"
      />

      <!-- Table -->
      <table>
        <thead>
          <tr>
            <th style="cursor:pointer" @click="setSort('supplierName')">
              Постачальник {{ sortArrow('supplierName') }}
            </th>
            <th style="cursor:pointer;text-align:right" @click="setSort('ordersCount')">
              Надходжень {{ sortArrow('ordersCount') }}
            </th>
            <th style="cursor:pointer;text-align:right" @click="setSort('purchasedUah')">
              Закуплено (UAH) {{ sortArrow('purchasedUah') }}
            </th>
            <th style="cursor:pointer;text-align:right" @click="setSort('paidUah')">
              Оплачено (UAH) {{ sortArrow('paidUah') }}
            </th>
            <th style="cursor:pointer;text-align:right" @click="setSort('debtUah')">
              Борг (UAH) {{ sortArrow('debtUah') }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in sortedSuppliers" :key="row.supplierId">
            <td style="font-weight:500">{{ row.supplierName }}</td>
            <td style="text-align:right;color:var(--text-subtle)">{{ row.ordersCount }}</td>
            <td style="text-align:right;font-weight:600;color:#9fe8c4">{{ fmt(row.purchasedUah) }}</td>
            <td style="text-align:right;color:#7ab4d4">{{ fmt(row.paidUah) }}</td>
            <td style="text-align:right" :style="debtColor(row.debtUah)">{{ fmt(row.debtUah) }}</td>
          </tr>
          <tr v-if="!sortedSuppliers.length">
            <td colspan="5" class="subtle" style="text-align:center;padding:1.5rem">
              {{ searchQ ? 'Нічого не знайдено' : 'Немає даних' }}
            </td>
          </tr>
          <!-- Totals row -->
          <tr v-if="sortedSuppliers.length" style="background:rgba(255,255,255,0.04);font-weight:700;border-top:2px solid rgba(255,255,255,0.12)">
            <td>Разом</td>
            <td style="text-align:right">{{ sortedSuppliers.reduce((s,r)=>s+r.ordersCount,0) }}</td>
            <td style="text-align:right;color:#9fe8c4">{{ fmt(sortedSuppliers.reduce((s,r)=>s+r.purchasedUah,0)) }}</td>
            <td style="text-align:right;color:#7ab4d4">{{ fmt(sortedSuppliers.reduce((s,r)=>s+r.paidUah,0)) }}</td>
            <td style="text-align:right" :style="debtColor(sortedSuppliers.reduce((s,r)=>s+r.debtUah,0))">
              {{ fmt(sortedSuppliers.reduce((s,r)=>s+r.debtUah,0)) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- ══════════════════════════════════════════════════════════
         COUNTERPARTIES REPORT
    ══════════════════════════════════════════════════════════ -->
    <div v-if="subTab === 'counterparties' && counterpartyReport && !loading">
      <!-- Summary cards -->
      <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:0.75rem;margin-bottom:1.2rem">
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Контрагентів</div>
          <div style="font-size:1.4rem;font-weight:700">{{ counterpartyReport.rows.length }}</div>
        </div>
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Продажі (UAH)</div>
          <div style="font-size:1.2rem;font-weight:700;color:#9fe8c4">{{ fmt(counterpartyReport.totalSalesUah) }}</div>
        </div>
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Закупівлі (UAH)</div>
          <div style="font-size:1.2rem;font-weight:700;color:#b09adc">{{ fmt(counterpartyReport.totalPurchasedUah) }}</div>
        </div>
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Оплачено (UAH)</div>
          <div style="font-size:1.2rem;font-weight:700;color:#7ab4d4">{{ fmt(counterpartyReport.totalPaidUah) }}</div>
        </div>
        <div class="panel" style="padding:0.9rem 1rem">
          <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Борг (UAH)</div>
          <div style="font-size:1.2rem;font-weight:700" :style="debtColor(counterpartyReport.totalDebtUah)">
            {{ fmt(counterpartyReport.totalDebtUah) }}
          </div>
        </div>
      </div>

      <!-- Search + filter -->
      <div style="display:flex;gap:0.6rem;align-items:center;margin-bottom:0.8rem;flex-wrap:wrap">
        <input
          v-model="searchQ"
          placeholder="Пошук контрагента…"
          style="max-width:260px"
        />
        <span class="subtle" style="font-size:0.82rem;margin-left:auto">
          {{ sortedCounterparties.length }} / {{ counterpartyReport.rows.length }}
        </span>
      </div>

      <!-- Table -->
      <table>
        <thead>
          <tr>
            <th style="cursor:pointer" @click="setSort('counterpartyName')">
              Контрагент {{ sortArrow('counterpartyName') }}
            </th>
            <th>Тип</th>
            <th style="cursor:pointer;text-align:right" @click="setSort('salesUah')">
              Продажі (UAH) {{ sortArrow('salesUah') }}
            </th>
            <th style="cursor:pointer;text-align:right" @click="setSort('purchasedUah')">
              Закупівлі (UAH) {{ sortArrow('purchasedUah') }}
            </th>
            <th style="cursor:pointer;text-align:right" @click="setSort('paidUah')">
              Оплачено (UAH) {{ sortArrow('paidUah') }}
            </th>
            <th style="cursor:pointer;text-align:right" @click="setSort('debtUah')">
              Борг (UAH) {{ sortArrow('debtUah') }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in sortedCounterparties" :key="row.counterpartyId">
            <td style="font-weight:500">{{ row.counterpartyName }}</td>
            <td>
              <span v-if="row.isCustomer" style="font-size:0.75rem;padding:0.1rem 0.4rem;border-radius:12px;background:rgba(122,180,212,0.15);color:#7ab4d4;margin-right:0.2rem">Покупець</span>
              <span v-if="row.isSupplier" style="font-size:0.75rem;padding:0.1rem 0.4rem;border-radius:12px;background:rgba(176,154,220,0.15);color:#b09adc">Постачальник</span>
            </td>
            <td style="text-align:right;font-weight:600;color:#9fe8c4">{{ fmt(row.salesUah) }}</td>
            <td style="text-align:right;color:#b09adc">{{ fmt(row.purchasedUah) }}</td>
            <td style="text-align:right;color:#7ab4d4">{{ fmt(row.paidUah) }}</td>
            <td style="text-align:right" :style="debtColor(row.debtUah)">{{ fmt(row.debtUah) }}</td>
          </tr>
          <tr v-if="!sortedCounterparties.length">
            <td colspan="6" class="subtle" style="text-align:center;padding:1.5rem">
              {{ searchQ ? 'Нічого не знайдено' : 'Немає даних' }}
            </td>
          </tr>
          <!-- Totals -->
          <tr v-if="sortedCounterparties.length" style="background:rgba(255,255,255,0.04);font-weight:700;border-top:2px solid rgba(255,255,255,0.12)">
            <td colspan="2">Разом</td>
            <td style="text-align:right;color:#9fe8c4">{{ fmt(sortedCounterparties.reduce((s,r)=>s+r.salesUah,0)) }}</td>
            <td style="text-align:right;color:#b09adc">{{ fmt(sortedCounterparties.reduce((s,r)=>s+r.purchasedUah,0)) }}</td>
            <td style="text-align:right;color:#7ab4d4">{{ fmt(sortedCounterparties.reduce((s,r)=>s+r.paidUah,0)) }}</td>
            <td style="text-align:right" :style="debtColor(sortedCounterparties.reduce((s,r)=>s+r.debtUah,0))">
              {{ fmt(sortedCounterparties.reduce((s,r)=>s+r.debtUah,0)) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
