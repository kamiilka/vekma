<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import type { UserSession, Summary, ProfitabilityReport } from "../types";

const props = defineProps<{ session: UserSession; initialSubTab?: string }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);

const summary = ref<Summary | null>(null);
const profitability = ref<ProfitabilityReport | null>(null);
const salesGrouped = ref<Array<{ period: string; salesQty: number; revenue: number; profit: number }>>([]);
const loading = ref(false);
const error = ref("");
const subTab = ref<"kpi" | "profitability" | "sales">((props.initialSubTab as any) ?? "kpi");
const groupBy = ref<"day" | "month">("month");

// Report builder state
const visibleColumns = ref<Set<string>>(new Set(["productName", "quantitySold", "revenueUah", "profitUah", "marginPct"]));
const sortField = ref<string>("revenueUah");
const sortDir = ref<"asc" | "desc">("desc");
const minMargin = ref(0);
const search = ref("");

const allColumns = [
  { key: "productName", label: "Назва" },
  { key: "sku", label: "SKU" },
  { key: "quantitySold", label: "Продано" },
  { key: "revenueUah", label: "Виторг, UAH" },
  { key: "costUah", label: "Собівартість, UAH" },
  { key: "profitUah", label: "Прибуток, UAH" },
  { key: "marginPct", label: "Маржа, %" },
];

async function load() {
  loading.value = true; error.value = "";
  try {
    [summary.value, profitability.value] = await Promise.all([
      api.summary(token.value),
      api.profitability(token.value),
    ]);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

async function loadSalesGrouped() {
  try {
    salesGrouped.value = await api.salesGrouped(token.value, groupBy.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

function toggleColumn(key: string) {
  if (visibleColumns.value.has(key)) { visibleColumns.value.delete(key); }
  else { visibleColumns.value.add(key); }
}

function setSort(key: string) {
  if (sortField.value === key) { sortDir.value = sortDir.value === "asc" ? "desc" : "asc"; }
  else { sortField.value = key; sortDir.value = "desc"; }
}

const filteredItems = computed(() => {
  if (!profitability.value) return [];
  let items = [...profitability.value.items];
  if (search.value) {
    const s = search.value.toLowerCase();
    items = items.filter(i => i.productName.toLowerCase().includes(s) || i.sku.toLowerCase().includes(s));
  }
  if (minMargin.value > 0) {
    items = items.filter(i => i.marginPct >= minMargin.value);
  }
  items.sort((a, b) => {
    const av = (a as any)[sortField.value] ?? 0;
    const bv = (b as any)[sortField.value] ?? 0;
    return sortDir.value === "asc" ? av - bv : bv - av;
  });
  return items;
});

function fmt(n: number) { return n.toLocaleString("uk-UA", { minimumFractionDigits: 0, maximumFractionDigits: 0 }); }
function fmtPct(n: number) { return n.toFixed(1) + "%"; }

onMounted(() => { load(); loadSalesGrouped(); });
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Аналітика та звіти</h2>
      <button class="ghost-button" @click="load">↻ Оновити</button>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <div class="tab-row" style="margin-bottom:1rem">
      <button :class="['tab-button', subTab==='kpi'&&'tab-button--active']" @click="subTab='kpi'">KPI</button>
      <button :class="['tab-button', subTab==='sales'&&'tab-button--active']" @click="subTab='sales';loadSalesGrouped()">Продажі по періодах</button>
      <button :class="['tab-button', subTab==='profitability'&&'tab-button--active']" @click="subTab='profitability'">Рентабельність</button>
    </div>

    <!-- KPI -->
    <div v-if="subTab==='kpi' && summary">
      <div class="kpi-grid">
        <div class="kpi-card">
          <p class="kpi-card__title">Товарів у базі</p>
          <p class="kpi-card__value">{{ summary.productCount }}</p>
        </div>
        <div class="kpi-card">
          <p class="kpi-card__title">Низький залишок</p>
          <p class="kpi-card__value" :style="summary.lowStock>0?'color:#ff9ca0':''">{{ summary.lowStock }}</p>
          <p class="kpi-card__hint">нижче мінімуму</p>
        </div>
        <div class="kpi-card">
          <p class="kpi-card__title">Загалом на складі</p>
          <p class="kpi-card__value">{{ summary.totalStock.toLocaleString('uk') }}</p>
          <p class="kpi-card__hint">штук</p>
        </div>
        <div class="kpi-card">
          <p class="kpi-card__title">Продажів</p>
          <p class="kpi-card__value">{{ summary.salesCount }}</p>
        </div>
        <div class="kpi-card">
          <p class="kpi-card__title">Виторг</p>
          <p class="kpi-card__value" style="color:#6dd4a0">{{ fmt(summary.revenue) }}</p>
          <p class="kpi-card__hint">UAH</p>
        </div>
      </div>

      <div v-if="profitability" style="margin-top:2rem">
        <h3>Зведення по прибутку</h3>
        <div class="kpi-grid">
          <div class="kpi-card">
            <p class="kpi-card__title">Виторг</p>
            <p class="kpi-card__value" style="color:#6dd4a0">{{ fmt(profitability.totalRevenue) }} UAH</p>
          </div>
          <div class="kpi-card">
            <p class="kpi-card__title">Собівартість</p>
            <p class="kpi-card__value" style="color:#ff9ca0">{{ fmt(profitability.totalCost) }} UAH</p>
          </div>
          <div class="kpi-card">
            <p class="kpi-card__title">Прибуток</p>
            <p class="kpi-card__value" :style="`color:${profitability.totalProfit>=0?'#6dd4a0':'#ff9ca0'}`">{{ fmt(profitability.totalProfit) }} UAH</p>
          </div>
          <div class="kpi-card">
            <p class="kpi-card__title">Маржа</p>
            <p class="kpi-card__value">{{ fmtPct(profitability.marginPct) }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- SALES GROUPED -->
    <div v-if="subTab==='sales'">
      <div style="display:flex;gap:0.5rem;align-items:center;margin-bottom:1rem">
        <button :class="['ghost-button', groupBy==='month'&&'tab-button--active']"
          style="padding:0.4rem 0.9rem" @click="groupBy='month';loadSalesGrouped()">По місяцях</button>
        <button :class="['ghost-button', groupBy==='day'&&'tab-button--active']"
          style="padding:0.4rem 0.9rem" @click="groupBy='day';loadSalesGrouped()">По днях</button>
      </div>

      <!-- Bar chart -->
      <div v-if="salesGrouped.length" class="panel" style="padding:1rem;margin-bottom:1rem;overflow-x:auto">
        <div style="display:flex;align-items:flex-end;gap:4px;height:140px;min-width:400px">
          <template v-for="row in [...salesGrouped].reverse()" :key="row.period">
            <div style="display:flex;flex-direction:column;align-items:center;flex:1;min-width:24px">
              <div style="font-size:0.68rem;color:#6a9e84;margin-bottom:2px">
                {{ fmt(row.revenue) }}
              </div>
              <div
                :style="`
                  width:100%;
                  background:linear-gradient(180deg,#3aad72,#2d8a58);
                  border-radius:4px 4px 0 0;
                  height:${Math.max(4, (row.revenue / Math.max(...salesGrouped.map(r=>r.revenue),1))*110)}px;
                  transition:height 0.3s;
                `"
              ></div>
              <div style="font-size:0.65rem;color:#6a9e84;margin-top:3px;writing-mode:vertical-lr;transform:rotate(180deg);max-height:48px;overflow:hidden">
                {{ row.period }}
              </div>
            </div>
          </template>
        </div>
      </div>
      <div v-else class="subtle">Немає даних</div>

      <!-- Table -->
      <table>
        <thead>
          <tr>
            <th>Період</th>
            <th>Продажів</th>
            <th>Виторг, UAH</th>
            <th>Прибуток, UAH</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in salesGrouped" :key="row.period">
            <td style="font-weight:600">{{ row.period }}</td>
            <td>{{ row.salesQty }}</td>
            <td style="color:#6dd4a0">{{ fmt(row.revenue) }}</td>
            <td :style="`color:${row.profit>=0?'#6dd4a0':'#ff9ca0'}`">{{ fmt(row.profit) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- PROFITABILITY REPORT BUILDER -->
    <div v-if="subTab==='profitability'">
      <!-- Report builder controls -->
      <div class="panel" style="padding:1rem;margin-bottom:1rem">
        <div style="font-size:0.85rem;color:#7ab99a;margin-bottom:0.5rem;font-weight:600">Конструктор звіту — оберіть колонки:</div>
        <div class="chip-row" style="margin-bottom:0.8rem">
          <label v-for="col in allColumns" :key="col.key"
            class="chip" :class="visibleColumns.has(col.key)?'chip--ok':''"
            style="cursor:pointer">
            <input type="checkbox" :checked="visibleColumns.has(col.key)" @change="toggleColumn(col.key)" style="display:none">
            {{ col.label }}
          </label>
        </div>
        <div style="display:flex;gap:0.5rem;flex-wrap:wrap">
          <input v-model="search" placeholder="Пошук товару..." style="flex:1;min-width:160px">
          <label style="display:flex;align-items:center;gap:0.4rem;font-size:0.85rem">
            Мін. маржа %
            <input type="number" v-model.number="minMargin" min="0" max="100" style="width:70px">
          </label>
        </div>
      </div>

      <!-- Summary row -->
      <div v-if="profitability" class="kpi-grid" style="margin-bottom:1rem">
        <div class="kpi-card">
          <p class="kpi-card__title">Рядків у звіті</p>
          <p class="kpi-card__value">{{ filteredItems.length }}</p>
        </div>
        <div class="kpi-card">
          <p class="kpi-card__title">Виторг (фільтр)</p>
          <p class="kpi-card__value" style="color:#6dd4a0">{{ fmt(filteredItems.reduce((a,i)=>a+i.revenueUah,0)) }}</p>
        </div>
        <div class="kpi-card">
          <p class="kpi-card__title">Прибуток (фільтр)</p>
          <p class="kpi-card__value">{{ fmt(filteredItems.reduce((a,i)=>a+i.profitUah,0)) }}</p>
        </div>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th v-for="col in allColumns.filter(c => visibleColumns.has(c.key))" :key="col.key"
                style="cursor:pointer;white-space:nowrap"
                @click="setSort(col.key)">
                {{ col.label }}
                <span v-if="sortField===col.key">{{ sortDir==='asc'?'↑':'↓' }}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredItems" :key="item.productId">
              <template v-for="col in allColumns.filter(c => visibleColumns.has(c.key))" :key="col.key">
                <td v-if="col.key==='productName'">{{ item.productName }}</td>
                <td v-else-if="col.key==='sku'" class="subtle">{{ item.sku }}</td>
                <td v-else-if="col.key==='quantitySold'">{{ item.quantitySold }}</td>
                <td v-else-if="col.key==='revenueUah'" style="color:#6dd4a0">{{ fmt(item.revenueUah) }}</td>
                <td v-else-if="col.key==='costUah'" style="color:#ff9ca0">{{ fmt(item.costUah) }}</td>
                <td v-else-if="col.key==='profitUah'" :style="`color:${item.profitUah>=0?'#6dd4a0':'#ff9ca0'}`">{{ fmt(item.profitUah) }}</td>
                <td v-else-if="col.key==='marginPct'">
                  <span :style="`color:${item.marginPct>=20?'#6dd4a0':item.marginPct>=5?'#fad07a':'#ff9ca0'}`">
                    {{ fmtPct(item.marginPct) }}
                  </span>
                </td>
              </template>
            </tr>
            <tr v-if="!filteredItems.length"><td :colspan="visibleColumns.size" class="subtle">Немає даних</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
