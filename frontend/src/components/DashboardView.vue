<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { SESSION_EXPIRED, api } from "../api";
import type { Product, ProfitabilityReport, Summary, UserSession, SupplierReport, CounterpartyReport, Supplier } from "../types";
import KpiCard from "./KpiCard.vue";

const props = defineProps<{
  session: UserSession;
}>();

const emit = defineEmits<{
  sessionExpired: [];
  navigate: [payload: string | { type: string; id: number }];
}>();

// ── State ─────────────────────────────────────────────────────
const summary = ref<Summary>({
  productCount: 0,
  lowStock: 0,
  totalStock: 0,
  salesCount: 0,
  revenue: 0
});
const profitability = ref<ProfitabilityReport>({
  items: [],
  totalRevenue: 0,
  totalCost: 0,
  totalProfit: 0,
  marginPct: 0
});
const salesGrouped = ref<Array<{ period: string; salesQty: number; revenue: number; profit: number }>>([]);
const supplierReport = ref<SupplierReport | null>(null);
const counterpartyReport = ref<CounterpartyReport | null>(null);
const allProducts = ref<Product[]>([]);
const suppliers = ref<Supplier[]>([]);

const token = computed(() => props.session.token);

// ── Quick-order modal ─────────────────────────────────────────
const showOrderModal = ref(false);
const orderProduct = ref<Product | null>(null);
const orderQty = ref(1);
const orderPrice = ref(0);
const orderSaving = ref(false);
const orderError = ref("");
const orderSuccess = ref("");
const selectedSupplierId = ref<number | null>(null); // explicit supplier chosen in modal

// ── Supplier inline edit/create ───────────────────────────────
const showSupplierForm = ref(false);
const supplierFormMode = ref<"create" | "edit">("create");
const supplierFormName = ref("");
const supplierFormContact = ref("");
const supplierFormPhone = ref("");
const supplierFormEmail = ref("");
const supplierFormComments = ref("");
const supplierFormSaving = ref(false);
const supplierFormError = ref("");
const supplierFormSuccess = ref("");

function openSupplierCreate() {
  supplierFormMode.value = "create";
  supplierFormName.value = orderProduct.value?.supplier || "";
  supplierFormContact.value = "";
  supplierFormPhone.value = "";
  supplierFormEmail.value = "";
  supplierFormComments.value = "";
  supplierFormError.value = "";
  supplierFormSuccess.value = "";
  showSupplierForm.value = true;
}

function openSupplierEdit() {
  if (!orderProduct.value) return;
  const sup = matchedSupplier(orderProduct.value);
  if (!sup) return;
  supplierFormMode.value = "edit";
  supplierFormName.value = sup.name;
  supplierFormContact.value = sup.contact || "";
  supplierFormPhone.value = sup.phone || "";
  supplierFormEmail.value = sup.email || "";
  supplierFormComments.value = sup.comments || "";
  supplierFormError.value = "";
  supplierFormSuccess.value = "";
  showSupplierForm.value = true;
}

async function saveSupplierForm() {
  if (!supplierFormName.value.trim()) { supplierFormError.value = "Назва постачальника обов'язкова"; return; }
  supplierFormSaving.value = true;
  supplierFormError.value = "";
  supplierFormSuccess.value = "";
  try {
    if (supplierFormMode.value === "create") {
      const newSup = await api.createSupplier(token.value, {
        name: supplierFormName.value.trim(),
        contact: supplierFormContact.value.trim() || undefined,
        phone: supplierFormPhone.value.trim() || undefined,
        email: supplierFormEmail.value.trim() || undefined,
        comments: supplierFormComments.value.trim() || undefined,
      });
      suppliers.value = [...suppliers.value, newSup];
      selectedSupplierId.value = newSup.id;
      supplierFormSuccess.value = `✓ Постачальника "${newSup.name}" додано`;
    } else {
      const sup = matchedSupplier(orderProduct.value!);
      if (!sup) { supplierFormError.value = "Постачальника не знайдено"; return; }
      // Find the counterparty for this supplier and update it
      const counterparties = await api.counterparties(token.value);
      const cp = counterparties.find(c => c.supplierId === sup.id || (c.isSupplier && c.name.toLowerCase().trim() === sup.name.toLowerCase().trim()));
      if (!cp) { supplierFormError.value = "Не вдалося знайти контрагента для оновлення"; return; }
      await api.updateCounterparty(token.value, cp.id, {
        name: supplierFormName.value.trim(),
        phone: supplierFormPhone.value.trim(),
        email: supplierFormEmail.value.trim(),
        comment: supplierFormComments.value.trim(),
        isCustomer: cp.isCustomer,
        isSupplier: true,
      });
      // Refresh suppliers list
      suppliers.value = await api.suppliers(token.value);
      supplierFormSuccess.value = `✓ Постачальника оновлено`;
    }
    showSupplierForm.value = false;
  } catch (e: any) {
    if (e.message === SESSION_EXPIRED) { emit("sessionExpired"); return; }
    supplierFormError.value = e.message;
  } finally { supplierFormSaving.value = false; }
}

function openOrderModal(p: Product) {
  orderProduct.value = p;
  orderQty.value = Math.max(1, p.minStock - p.stock);
  orderPrice.value = p.purchasePrice ?? 0;
  orderError.value = "";
  orderSuccess.value = "";
  selectedSupplierId.value = null;
  showSupplierForm.value = false;
  showOrderModal.value = true;
}

function matchedSupplier(p: Product): Supplier | undefined {
  if (!p.supplier) return undefined;
  const name = p.supplier.toLowerCase().trim();
  return suppliers.value.find(s => s.name.toLowerCase().trim() === name);
}

async function submitOrder() {
  if (!orderProduct.value) return;
  const sup = matchedSupplier(orderProduct.value)
    ?? (selectedSupplierId.value ? suppliers.value.find(s => s.id === selectedSupplierId.value) : undefined);
  if (!sup) { orderError.value = "Постачальника не знайдено в базі. Уточніть назву постачальника у картці товару."; return; }
  orderSaving.value = true; orderError.value = ""; orderSuccess.value = "";
  try {
    await api.createSupplierOrder(token.value, {
      supplierId: sup.id,
      items: [{ productId: orderProduct.value.id, quantity: orderQty.value, price: orderPrice.value }],
    });
    orderSuccess.value = `✓ Замовлення до "${sup.name}" створено на ${orderQty.value} шт.`;
  } catch (e: any) {
    if (e.message === SESSION_EXPIRED) { emit("sessionExpired"); return; }
    orderError.value = e.message;
  } finally { orderSaving.value = false; }
}

// ── Bulk auto-order ───────────────────────────────────────────
const bulkSaving = ref(false);
const showBulkResult = ref(false);
type BulkResultItem = { productName: string; supplierName: string; qty: number; status: 'ok' | 'no_supplier' | 'error'; error?: string };
const bulkResults = ref<BulkResultItem[]>([]);
const bulkOrdersCreated = ref(0);

async function getLastPurchasedQty(productId: number): Promise<number | null> {
  try {
    const lifecycle = await api.productLifecycle(token.value, productId);
    const purchaseEvents = lifecycle.events.filter(e => e.eventType === 'purchased' || e.eventType === 'received');
    if (purchaseEvents.length === 0) return null;
    // Sort descending by date, take the last purchase quantity
    purchaseEvents.sort((a, b) => new Date(b.eventDate).getTime() - new Date(a.eventDate).getTime());
    return purchaseEvents[0].quantity ?? null;
  } catch { return null; }
}

async function runBulkAutoOrder() {
  bulkSaving.value = true;
  bulkResults.value = [];
  bulkOrdersCreated.value = 0;
  showBulkResult.value = false;

  const products = lowStockProducts.value;

  // Resolve qty for each product
  const resolved: Array<{ product: Product; qty: number; supplier: Supplier | undefined }> = [];
  await Promise.all(products.map(async (p) => {
    const lastQty = await getLastPurchasedQty(p.id);
    const qty = lastQty ?? Math.max(1, p.minStock - p.stock);
    resolved.push({ product: p, qty, supplier: matchedSupplier(p) });
  }));

  // Group by supplier
  const bySupplier = new Map<number, { supplier: Supplier; items: Array<{ productId: number; quantity: number; price: number }> }>();
  const noSupplier: BulkResultItem[] = [];

  for (const { product, qty, supplier } of resolved) {
    if (!supplier) {
      noSupplier.push({ productName: product.name, supplierName: product.supplier || '—', qty, status: 'no_supplier' });
      continue;
    }
    if (!bySupplier.has(supplier.id)) {
      bySupplier.set(supplier.id, { supplier, items: [] });
    }
    bySupplier.get(supplier.id)!.items.push({ productId: product.id, quantity: qty, price: product.purchasePrice ?? 0 });
  }

  // Create one order per supplier
  for (const [, { supplier, items }] of bySupplier) {
    try {
      await api.createSupplierOrder(token.value, { supplierId: supplier.id, items });
      bulkOrdersCreated.value++;
      for (const item of items) {
        const p = products.find(x => x.id === item.productId)!;
        bulkResults.value.push({ productName: p.name, supplierName: supplier.name, qty: item.quantity, status: 'ok' });
      }
    } catch (e: any) {
      if (e.message === SESSION_EXPIRED) { emit("sessionExpired"); return; }
      for (const item of items) {
        const p = products.find(x => x.id === item.productId)!;
        bulkResults.value.push({ productName: p.name, supplierName: supplier.name, qty: item.quantity, status: 'error', error: e.message });
      }
    }
  }

  bulkResults.value.push(...noSupplier);
  bulkSaving.value = false;
  showBulkResult.value = true;
}

const errorText = ref("");
const isLoading = ref(false);

// ── Dashboard sub-tabs ────────────────────────────────────────
type DashTab = "overview" | "analytics";
const dashTab = ref<DashTab>("overview");

// Analytics sub-tabs
const analyticsSubTab = ref<"profitability" | "sales">("sales");
const groupBy = ref<"day" | "month">("month");

// Report builder state
const visibleColumns = ref<Set<string>>(new Set(["productName", "quantitySold", "revenueUah", "profitUah", "marginPct"]));
const sortField = ref<string>("revenueUah");
const sortDir = ref<"asc" | "desc">("desc");
const minMargin = ref(0);
const analyticsSearch = ref("");

const allColumns = [
  { key: "productName", label: "Назва" },
  { key: "sku", label: "SKU" },
  { key: "quantitySold", label: "Продано" },
  { key: "revenueUah", label: "Виторг, UAH" },
  { key: "costUah", label: "Собівартість, UAH" },
  { key: "profitUah", label: "Прибуток, UAH" },
  { key: "marginPct", label: "Маржа, %" },
];

// Reports sub-tabs
type ReportsSubTab = "suppliers" | "counterparties";
const reportsSubTab = ref<ReportsSubTab>("suppliers");
const reportsSortKey = ref<string>("supplierName");
const reportsSortDir = ref<1 | -1>(1);
const reportsSearch = ref("");

// ── Computed ──────────────────────────────────────────────────
const lowStockProducts = computed(() =>
  allProducts.value.filter(p => !p.archived && p.minStock > 0 && p.stock <= p.minStock)
    .sort((a, b) => (a.stock - a.minStock) - (b.stock - b.minStock))
);

const filteredAnalyticsItems = computed(() => {
  let items = [...profitability.value.items];
  if (analyticsSearch.value) {
    const s = analyticsSearch.value.toLowerCase();
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

const sortedSuppliers = computed(() => {
  if (!supplierReport.value) return [];
  let rows = [...supplierReport.value.rows];
  const q = reportsSearch.value.toLowerCase().trim();
  if (q) rows = rows.filter(r => r.supplierName.toLowerCase().includes(q));
  rows.sort((a, b) => {
    const av = (a as any)[reportsSortKey.value] ?? "";
    const bv = (b as any)[reportsSortKey.value] ?? "";
    return typeof av === "number"
      ? (av - bv) * reportsSortDir.value
      : String(av).localeCompare(String(bv)) * reportsSortDir.value;
  });
  return rows;
});

const sortedCounterparties = computed(() => {
  if (!counterpartyReport.value) return [];
  let rows = [...counterpartyReport.value.rows];
  const q = reportsSearch.value.toLowerCase().trim();
  if (q) rows = rows.filter(r => r.counterpartyName.toLowerCase().includes(q));
  rows.sort((a, b) => {
    const av = (a as any)[reportsSortKey.value] ?? "";
    const bv = (b as any)[reportsSortKey.value] ?? "";
    return typeof av === "number"
      ? (av - bv) * reportsSortDir.value
      : String(av).localeCompare(String(bv)) * reportsSortDir.value;
  });
  return rows;
});

// ── Helpers ───────────────────────────────────────────────────
function handleError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message === SESSION_EXPIRED) {
    emit("sessionExpired");
    return;
  }
  errorText.value = error instanceof Error ? error.message : fallback;
}

function fmt(n: number) { return n.toLocaleString("uk-UA", { minimumFractionDigits: 0, maximumFractionDigits: 0 }); }
function fmtPct(n: number) { return n.toFixed(1) + "%"; }
function fmtFull(n: number) { return n.toLocaleString("uk", { minimumFractionDigits: 2, maximumFractionDigits: 2 }); }
function debtColor(debt: number) {
  if (debt <= 0) return "";
  if (debt > 10000) return "color:#e08080;font-weight:700";
  return "color:#e0c060;font-weight:600";
}

function toggleColumn(key: string) {
  if (visibleColumns.value.has(key)) { visibleColumns.value.delete(key); }
  else { visibleColumns.value.add(key); }
}

function setAnalyticsSort(key: string) {
  if (sortField.value === key) { sortDir.value = sortDir.value === "asc" ? "desc" : "asc"; }
  else { sortField.value = key; sortDir.value = "desc"; }
}

function setReportsSort(key: string) {
  if (reportsSortKey.value === key) { reportsSortDir.value = reportsSortDir.value === 1 ? -1 : 1; }
  else { reportsSortKey.value = key; reportsSortDir.value = 1; }
}

function sortArrow(key: string) {
  if (reportsSortKey.value !== key) return "↕";
  return reportsSortDir.value === 1 ? "↑" : "↓";
}

// ── Data loading ──────────────────────────────────────────────
async function loadData() {
  try {
    isLoading.value = true;
    errorText.value = "";
    const [loadedSummary, loadedProfitability, loadedProducts, loadedSuppliers] = await Promise.all([
      api.summary(props.session.token),
      api.profitability(props.session.token),
      api.products(props.session.token),
      api.suppliers(props.session.token),
    ]);
    summary.value = loadedSummary;
    profitability.value = loadedProfitability;
    allProducts.value = loadedProducts;
    suppliers.value = loadedSuppliers;
  } catch (error) {
    handleError(error, "Не вдалося завантажити дані");
  } finally {
    isLoading.value = false;
  }
}

async function loadSalesGrouped() {
  try {
    salesGrouped.value = await api.salesGrouped(props.session.token, groupBy.value);
  } catch (error) {
    handleError(error, "Помилка завантаження продажів");
  }
}

async function loadSuppliers(force = false) {
  if (supplierReport.value && !force) return;
  try {
    supplierReport.value = await api.supplierReport(props.session.token);
  } catch (error) {
    handleError(error, "Помилка завантаження постачальників");
  }
}

async function loadCounterparties() {
  if (counterpartyReport.value) return;
  try {
    counterpartyReport.value = await api.counterpartyReport(props.session.token);
  } catch (error) {
    handleError(error, "Помилка завантаження контрагентів");
  }
}

async function refreshReports() {
  if (reportsSubTab.value === "suppliers") {
    await loadSuppliers(true);
  } else {
    counterpartyReport.value = null;
    await loadCounterparties();
  }
}

watch(dashTab, (val) => {
  if (val === "analytics") { loadSalesGrouped(); }
});

onMounted(() => {
  loadData();
  loadSuppliers();
});
</script>

<template>
  <main class="layout page-content">

    <!-- KPI cards -->
    <section class="kpi-grid">
      <KpiCard title="Товарів" :value="summary.productCount" hint="Активна номенклатура" />
      <KpiCard title="Склад (шт)" :value="summary.totalStock" hint="Загальний залишок" />
      <KpiCard title="Продажі" :value="summary.salesCount" hint="Закриті чеки" />
      <KpiCard title="Виторг" :value="`${summary.revenue.toFixed(2)} грн`" hint="Сума продажів" />
      <KpiCard title="Валовий прибуток" :value="`${profitability.totalProfit.toFixed(2)} грн`" hint="Revenue - собівартість" />
      <KpiCard
        v-if="summary.lowStock > 0"
        title="⚠ Мало на складі"
        :value="summary.lowStock"
        hint="Нижче мінімуму"
        style="border-color:rgba(255,156,160,0.5)"
      />
      <KpiCard
        v-if="supplierReport !== null"
        :title="supplierReport.totalDebtUah > 0 ? '⚠ Борг постачальникам' : 'Борг постачальникам'"
        :value="fmtFull(supplierReport.totalDebtUah) + ' грн'"
        hint="Сума незакритих замовлень"
        :style="supplierReport.totalDebtUah > 0 ? 'border-color:rgba(224,128,128,0.6)' : ''"
      />
    </section>

    <p v-if="errorText" class="error-text">{{ errorText }}</p>
    <p v-if="isLoading" class="subtle">Оновлення даних...</p>

    <!-- Dashboard tab switcher -->
    <div class="tab-row" style="margin-top:1.4rem;margin-bottom:1rem">
      <button :class="['tab-button', dashTab==='overview' && 'tab-button--active']" @click="dashTab='overview'">Огляд</button>
      <button :class="['tab-button', dashTab==='analytics' && 'tab-button--active']" @click="dashTab='analytics'">Аналітика</button>

    </div>

    <!-- ══════════════════════════════ OVERVIEW ══════════════════════════════ -->
    <template v-if="dashTab==='overview'">
      <section class="panel">
        <h2>Прибутковість (маржа)</h2>
        <div class="chip-row">
          <span class="chip chip--ok">Revenue: {{ profitability.totalRevenue.toFixed(2) }} грн</span>
          <span class="chip chip--required">Cost: {{ profitability.totalCost.toFixed(2) }} грн</span>
          <span class="chip" :class="profitability.totalProfit >= 0 ? 'chip--ok' : 'chip--warn'">
            Profit: {{ profitability.totalProfit.toFixed(2) }} грн
          </span>
          <span class="chip">Margin: {{ profitability.marginPct.toFixed(2) }}%</span>
        </div>
        <table>
          <thead>
            <tr>
              <th>Товар</th><th>SKU</th><th>Продано</th>
              <th>Виторг, UAH</th><th>Собівартість, UAH</th>
              <th>Прибуток, UAH</th><th>Margin %</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in profitability.items.slice(0, 10)" :key="`profit-${item.productId}`">
              <td>{{ item.productName }}</td>
              <td>{{ item.sku }}</td>
              <td>{{ item.quantitySold }}</td>
              <td>{{ item.revenueUah.toFixed(2) }}</td>
              <td>{{ item.costUah.toFixed(2) }}</td>
              <td>{{ item.profitUah.toFixed(2) }}</td>
              <td>{{ item.marginPct.toFixed(2) }}%</td>
            </tr>
          </tbody>
        </table>
        <p v-if="profitability.items.length === 0" class="subtle">Немає продажів для розрахунку прибутковості.</p>
      </section>

      <section class="panel">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:0.6rem">
          <h2 style="margin:0">Низький залишок</h2>
          <button
            v-if="lowStockProducts.length > 0"
            class="ghost-button"
            style="font-size:0.85rem;padding:0.4rem 0.9rem;border-color:rgba(109,212,160,0.4);color:#6dd4a0"
            :disabled="bulkSaving"
            @click="runBulkAutoOrder"
          >
            <span v-if="bulkSaving">⏳ Формуємо замовлення...</span>
            <span v-else>⚡ Авто-замовлення всіх дефіцитів</span>
          </button>
        </div>
        <div v-if="lowStockProducts.length === 0" class="chip-row">
          <span class="subtle">Критичних позицій немає</span>
        </div>
        <template v-else>
          <div class="chip-row" style="margin-bottom:0.8rem">
            <span class="chip chip--warn">{{ lowStockProducts.length }} позицій нижче мінімуму</span>
          </div>
          <table>
            <thead>
              <tr>
                <th>Товар</th>
                <th>SKU</th>
                <th style="text-align:right">Залишок</th>
                <th style="text-align:right">Мінімум</th>
                <th style="text-align:right">Дефіцит</th>
                <th>Постачальник</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="p in lowStockProducts"
                :key="p.id"
              >
                <td style="font-weight:500">{{ p.name }}</td>
                <td class="subtle">{{ p.sku }}</td>
                <td style="text-align:right;font-weight:700" :style="p.stock === 0 ? 'color:#ff9ca0' : 'color:#fad07a'">
                  {{ p.stock }}
                </td>
                <td style="text-align:right;color:var(--text-muted)">{{ p.minStock }}</td>
                <td style="text-align:right;color:#ff9ca0;font-weight:600">
                  −{{ Math.max(0, p.minStock - p.stock) }}
                </td>
                <td class="subtle">{{ p.supplier || '—' }}</td>
                <td @click.stop>
                  <button
                    class="ghost-button"
                    style="padding:0.25rem 0.65rem;font-size:0.8rem;white-space:nowrap"
                    :title="matchedSupplier(p) ? `Замовити у ${matchedSupplier(p)!.name}` : 'Замовити'"
                    @click="openOrderModal(p)"
                  >📦 Замовити</button>
                </td>
              </tr>
            </tbody>
          </table>

          <!-- Quick-order modal -->
          <div v-if="showOrderModal && orderProduct" class="modal-backdrop" @click.self="showOrderModal=false">
            <div class="panel modal-box" style="max-width:480px">
              <h3 style="margin:0 0 1rem">Замовлення товару</h3>

              <div style="padding:0.8rem;background:rgba(255,255,255,0.04);border-radius:8px;margin-bottom:1rem">
                <div style="font-weight:600;margin-bottom:0.3rem">{{ orderProduct.name }}</div>
                <div class="subtle" style="font-size:0.82rem">SKU: {{ orderProduct.sku }}</div>
                <div style="display:flex;gap:1.2rem;margin-top:0.5rem;font-size:0.85rem">
                  <span>Залишок: <strong :style="orderProduct.stock===0?'color:#ff9ca0':'color:#fad07a'">{{ orderProduct.stock }} шт</strong></span>
                  <span>Мінімум: <strong>{{ orderProduct.minStock }} шт</strong></span>
                  <span>Дефіцит: <strong style="color:#ff9ca0">{{ Math.max(0, orderProduct.minStock - orderProduct.stock) }} шт</strong></span>
                </div>
              </div>

              <!-- Supplier found -->
              <div v-if="(matchedSupplier(orderProduct) || selectedSupplierId) && !showSupplierForm" style="display:flex;align-items:center;gap:0.5rem;margin-bottom:1rem;padding:0.55rem 0.8rem;background:rgba(109,212,160,0.07);border:1px solid rgba(109,212,160,0.2);border-radius:6px;font-size:0.88rem">
                <span style="color:#6dd4a0">✓</span>
                <span>Постачальник: <strong>{{ matchedSupplier(orderProduct)?.name ?? suppliers.find(s => s.id === selectedSupplierId)?.name }}</strong></span>
                <button class="ghost-button" style="margin-left:auto;font-size:0.78rem;padding:0.2rem 0.6rem" @click="openSupplierEdit">✏ Редагувати</button>
              </div>

              <!-- Supplier not found -->
              <div v-if="!matchedSupplier(orderProduct) && !selectedSupplierId && !showSupplierForm" style="margin-bottom:1rem;padding:0.55rem 0.8rem;background:rgba(255,156,160,0.07);border:1px solid rgba(255,156,160,0.25);border-radius:6px;font-size:0.85rem;color:#ff9ca0">
                <div style="display:flex;align-items:center;gap:0.5rem">
                  <span>⚠</span>
                  <span>Постачальника "{{ orderProduct.supplier || '(не вказано)' }}" не знайдено в базі контрагентів.</span>
                </div>
                <div style="display:flex;align-items:center;gap:0.5rem;margin-top:0.5rem">
                  <span class="subtle" style="font-size:0.78rem;flex:1">Додайте постачальника, щоб мати змогу створити замовлення.</span>
                  <button class="ghost-button" style="font-size:0.8rem;padding:0.25rem 0.7rem;color:#6dd4a0;border-color:rgba(109,212,160,0.3)" @click="openSupplierCreate">+ Додати</button>
                </div>
              </div>

              <!-- Inline supplier form -->
              <div v-if="showSupplierForm" style="margin-bottom:1rem;padding:0.8rem;background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.1);border-radius:8px">
                <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:0.75rem">
                  <strong style="font-size:0.9rem">{{ supplierFormMode === 'create' ? '+ Новий постачальник' : '✏ Редагувати постачальника' }}</strong>
                  <button class="ghost-button" style="font-size:0.75rem;padding:0.15rem 0.5rem" @click="showSupplierForm=false">✕</button>
                </div>
                <div class="grid" style="gap:0.6rem;margin-bottom:0.6rem">
                  <label style="grid-column:1/-1">
                    Назва <span style="color:#ff9ca0">*</span>
                    <input v-model="supplierFormName" type="text" placeholder="Назва постачальника" style="margin-top:0.25rem">
                  </label>
                  <label>
                    Контактна особа
                    <input v-model="supplierFormContact" type="text" placeholder="ПІБ" style="margin-top:0.25rem">
                  </label>
                  <label>
                    Телефон
                    <input v-model="supplierFormPhone" type="text" placeholder="+380..." style="margin-top:0.25rem">
                  </label>
                  <label>
                    Email
                    <input v-model="supplierFormEmail" type="email" placeholder="email@example.com" style="margin-top:0.25rem">
                  </label>
                  <label>
                    Коментар
                    <input v-model="supplierFormComments" type="text" placeholder="Примітки" style="margin-top:0.25rem">
                  </label>
                </div>
                <div v-if="supplierFormError" style="font-size:0.82rem;color:#ff9ca0;margin-bottom:0.5rem">{{ supplierFormError }}</div>
                <div style="display:flex;gap:0.5rem">
                  <button @click="saveSupplierForm" :disabled="supplierFormSaving || !supplierFormName.trim()" style="font-size:0.85rem;padding:0.35rem 0.9rem">
                    {{ supplierFormSaving ? 'Збереження...' : (supplierFormMode === 'create' ? '+ Додати постачальника' : '✓ Зберегти зміни') }}
                  </button>
                  <button class="ghost-button" style="font-size:0.85rem;padding:0.35rem 0.9rem" @click="showSupplierForm=false">Скасувати</button>
                </div>
              </div>

              <div v-if="!showSupplierForm" class="grid" style="margin-bottom:1rem">
                <label>
                  Кількість (шт)
                  <input type="number" min="1" v-model.number="orderQty" style="margin-top:0.3rem">
                </label>
                <label>
                  Ціна закупівлі (UAH)
                  <input type="number" min="0" step="0.01" v-model.number="orderPrice" style="margin-top:0.3rem">
                </label>
              </div>

              <div v-if="!showSupplierForm && orderQty > 0 && orderPrice > 0" style="margin-bottom:1rem;padding:0.5rem 0.75rem;background:rgba(255,255,255,0.03);border-radius:6px;font-size:0.85rem">
                Сума замовлення: <strong>{{ (orderQty * orderPrice).toFixed(2) }} UAH</strong>
              </div>

              <div v-if="orderSuccess" style="margin-bottom:0.8rem;padding:0.55rem 0.8rem;background:rgba(109,212,160,0.1);border:1px solid rgba(109,212,160,0.3);border-radius:6px;color:#6dd4a0;font-size:0.88rem">
                {{ orderSuccess }}
              </div>
              <p v-if="orderError" class="error-text">{{ orderError }}</p>

              <div v-if="!showSupplierForm" style="display:flex;gap:0.5rem">
                <button
                  @click="submitOrder"
                  :disabled="orderSaving || !orderQty || (!matchedSupplier(orderProduct) && !selectedSupplierId) || !!orderSuccess"
                >{{ orderSaving ? 'Створення...' : '✓ Створити замовлення' }}</button>
                <button class="ghost-button" @click="showOrderModal=false">{{ orderSuccess ? 'Закрити' : 'Скасувати' }}</button>
              </div>
            </div>
          </div>

          <!-- Bulk auto-order result modal -->
          <div v-if="showBulkResult" class="modal-backdrop" @click.self="showBulkResult=false">
            <div class="panel modal-box" style="max-width:560px">
              <h3 style="margin:0 0 1rem">Результат авто-замовлення</h3>

              <div style="display:flex;gap:1rem;margin-bottom:1rem;flex-wrap:wrap">
                <div style="padding:0.55rem 1rem;background:rgba(109,212,160,0.1);border:1px solid rgba(109,212,160,0.25);border-radius:8px;font-size:0.88rem">
                  ✓ Замовлень створено: <strong style="color:#6dd4a0">{{ bulkOrdersCreated }}</strong>
                </div>
                <div v-if="bulkResults.some(r => r.status === 'no_supplier')" style="padding:0.55rem 1rem;background:rgba(250,208,122,0.1);border:1px solid rgba(250,208,122,0.25);border-radius:8px;font-size:0.88rem">
                  ⚠ Без постачальника: <strong style="color:#fad07a">{{ bulkResults.filter(r => r.status === 'no_supplier').length }}</strong>
                </div>
                <div v-if="bulkResults.some(r => r.status === 'error')" style="padding:0.55rem 1rem;background:rgba(255,156,160,0.1);border:1px solid rgba(255,156,160,0.25);border-radius:8px;font-size:0.88rem">
                  ✗ Помилок: <strong style="color:#ff9ca0">{{ bulkResults.filter(r => r.status === 'error').length }}</strong>
                </div>
              </div>

              <div style="max-height:340px;overflow-y:auto">
                <table>
                  <thead>
                    <tr>
                      <th>Товар</th>
                      <th>Постачальник</th>
                      <th style="text-align:right">К-сть</th>
                      <th>Статус</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(r, i) in bulkResults" :key="i">
                      <td style="font-weight:500">{{ r.productName }}</td>
                      <td class="subtle">{{ r.supplierName }}</td>
                      <td style="text-align:right">{{ r.qty }}</td>
                      <td>
                        <span v-if="r.status === 'ok'" style="color:#6dd4a0;font-size:0.82rem">✓ Замовлено</span>
                        <span v-else-if="r.status === 'no_supplier'" style="color:#fad07a;font-size:0.82rem">⚠ Немає постачальника</span>
                        <span v-else style="color:#ff9ca0;font-size:0.82rem" :title="r.error">✗ Помилка</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <div style="margin-top:1rem">
                <button class="ghost-button" @click="showBulkResult=false">Закрити</button>
              </div>
            </div>
          </div>
        </template>
      </section>
    </template>

    <!-- ══════════════════════════════ ANALYTICS ══════════════════════════════ -->
    <template v-if="dashTab==='analytics'">
      <div class="page-header" style="margin-bottom:1rem">
        <h2 style="margin:0">Аналітика</h2>
        <button class="ghost-button" @click="loadData();loadSalesGrouped()">↻ Оновити</button>
      </div>

      <div class="tab-row" style="margin-bottom:1rem">
        <button :class="['tab-button', analyticsSubTab==='sales'&&'tab-button--active']" @click="analyticsSubTab='sales';loadSalesGrouped()">Продажі по періодах</button>
        <button :class="['tab-button', analyticsSubTab==='profitability'&&'tab-button--active']" @click="analyticsSubTab='profitability'">Рентабельність</button>
      </div>

      <!-- SALES      <!-- SALES GROUPED -->
      <div v-if="analyticsSubTab==='sales'">
        <div style="display:flex;gap:0.5rem;align-items:center;margin-bottom:1rem">
          <button :class="['ghost-button', groupBy==='month'&&'tab-button--active']"
            style="padding:0.4rem 0.9rem" @click="groupBy='month';loadSalesGrouped()">По місяцях</button>
          <button :class="['ghost-button', groupBy==='day'&&'tab-button--active']"
            style="padding:0.4rem 0.9rem" @click="groupBy='day';loadSalesGrouped()">По днях</button>
        </div>

        <div v-if="salesGrouped.length" class="panel" style="padding:1rem;margin-bottom:1rem;overflow-x:auto">
          <div style="display:flex;align-items:flex-end;gap:4px;height:140px;min-width:400px">
            <template v-for="row in [...salesGrouped].reverse()" :key="row.period">
              <div style="display:flex;flex-direction:column;align-items:center;flex:1;min-width:24px">
                <div style="font-size:0.68rem;color:#6a9e84;margin-bottom:2px">{{ fmt(row.revenue) }}</div>
                <div :style="`width:100%;background:linear-gradient(180deg,#3aad72,#2d8a58);border-radius:4px 4px 0 0;height:${Math.max(4,(row.revenue/Math.max(...salesGrouped.map(r=>r.revenue),1))*110)}px;transition:height 0.3s;`"></div>
                <div style="font-size:0.65rem;color:#6a9e84;margin-top:3px;writing-mode:vertical-lr;transform:rotate(180deg);max-height:48px;overflow:hidden">{{ row.period }}</div>
              </div>
            </template>
          </div>
        </div>
        <div v-else class="subtle">Немає даних</div>

        <table>
          <thead>
            <tr>
              <th>Період</th><th>Продажів</th><th>Виторг, UAH</th><th>Прибуток, UAH</th>
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
      <div v-if="analyticsSubTab==='profitability'">
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
            <input v-model="analyticsSearch" placeholder="Пошук товару..." style="flex:1;min-width:160px">
            <label style="display:flex;align-items:center;gap:0.4rem;font-size:0.85rem">
              Мін. маржа %
              <input type="number" v-model.number="minMargin" min="0" max="100" style="width:70px">
            </label>
          </div>
        </div>

        <div class="kpi-grid" style="margin-bottom:1rem">
          <div class="kpi-card">
            <p class="kpi-card__title">Рядків у звіті</p>
            <p class="kpi-card__value">{{ filteredAnalyticsItems.length }}</p>
          </div>
          <div class="kpi-card">
            <p class="kpi-card__title">Виторг (фільтр)</p>
            <p class="kpi-card__value" style="color:#6dd4a0">{{ fmt(filteredAnalyticsItems.reduce((a,i)=>a+i.revenueUah,0)) }}</p>
          </div>
          <div class="kpi-card">
            <p class="kpi-card__title">Прибуток (фільтр)</p>
            <p class="kpi-card__value">{{ fmt(filteredAnalyticsItems.reduce((a,i)=>a+i.profitUah,0)) }}</p>
          </div>
        </div>

        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th v-for="col in allColumns.filter(c => visibleColumns.has(c.key))" :key="col.key"
                  style="cursor:pointer;white-space:nowrap"
                  @click="setAnalyticsSort(col.key)">
                  {{ col.label }}
                  <span v-if="sortField===col.key">{{ sortDir==='asc'?'↑':'↓' }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in filteredAnalyticsItems" :key="item.productId">
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
              <tr v-if="!filteredAnalyticsItems.length"><td :colspan="visibleColumns.size" class="subtle">Немає даних</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- REPORTS TAB REMOVED -->
    <template v-if="false">
      <div class="page-header" style="margin-bottom:1rem">
        <h2 style="margin:0">Звіти</h2>
        <button class="ghost-button" @click="refreshReports" :disabled="isLoading">
          {{ isLoading ? "Завантаження…" : "↻ Оновити" }}
        </button>
      </div>

      <div class="tab-row" style="margin-bottom:1rem">
        <button :class="['tab-button', reportsSubTab==='suppliers'&&'tab-button--active']" @click="reportsSubTab='suppliers'">📦 По постачальниках</button>
        <button :class="['tab-button', reportsSubTab==='counterparties'&&'tab-button--active']" @click="reportsSubTab='counterparties'">◎ По контрагентах</button>
      </div>

      <!-- SUPPLIERS REPORT -->
      <div v-if="reportsSubTab==='suppliers' && supplierReport && !isLoading">
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:0.75rem;margin-bottom:1.2rem">
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Постачальників</div>
            <div style="font-size:1.4rem;font-weight:700">{{ supplierReport.rows.length }}</div>
          </div>
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Закуплено (UAH)</div>
            <div style="font-size:1.2rem;font-weight:700;color:#9fe8c4">{{ fmtFull(supplierReport.totalPurchasedUah) }}</div>
          </div>
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Оплачено (UAH)</div>
            <div style="font-size:1.2rem;font-weight:700;color:#7ab4d4">{{ fmtFull(supplierReport.totalPaidUah) }}</div>
          </div>
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Борг (UAH)</div>
            <div style="font-size:1.2rem;font-weight:700" :style="debtColor(supplierReport.totalDebtUah)">{{ fmtFull(supplierReport.totalDebtUah) }}</div>
          </div>
        </div>

        <input v-model="reportsSearch" placeholder="Пошук постачальника…" style="margin-bottom:0.8rem;max-width:300px" />

        <table>
          <thead>
            <tr>
              <th style="cursor:pointer" @click="setReportsSort('supplierName')">Постачальник {{ sortArrow('supplierName') }}</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('ordersCount')">Надходжень {{ sortArrow('ordersCount') }}</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('purchasedUah')">Закуплено (UAH) {{ sortArrow('purchasedUah') }}</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('paidUah')">Оплачено (UAH) {{ sortArrow('paidUah') }}</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('debtUah')">Борг (UAH) {{ sortArrow('debtUah') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in sortedSuppliers" :key="row.supplierId">
              <td style="font-weight:500">{{ row.supplierName }}</td>
              <td style="text-align:right;color:var(--text-subtle)">{{ row.ordersCount }}</td>
              <td style="text-align:right;font-weight:600;color:#9fe8c4">{{ fmtFull(row.purchasedUah) }}</td>
              <td style="text-align:right;color:#7ab4d4">{{ fmtFull(row.paidUah) }}</td>
              <td style="text-align:right" :style="debtColor(row.debtUah)">{{ fmtFull(row.debtUah) }}</td>
            </tr>
            <tr v-if="!sortedSuppliers.length">
              <td colspan="5" class="subtle" style="text-align:center;padding:1.5rem">{{ reportsSearch ? 'Нічого не знайдено' : 'Немає даних' }}</td>
            </tr>
            <tr v-if="sortedSuppliers.length" style="background:rgba(255,255,255,0.04);font-weight:700;border-top:2px solid rgba(255,255,255,0.12)">
              <td>Разом</td>
              <td style="text-align:right">{{ sortedSuppliers.reduce((s,r)=>s+r.ordersCount,0) }}</td>
              <td style="text-align:right;color:#9fe8c4">{{ fmtFull(sortedSuppliers.reduce((s,r)=>s+r.purchasedUah,0)) }}</td>
              <td style="text-align:right;color:#7ab4d4">{{ fmtFull(sortedSuppliers.reduce((s,r)=>s+r.paidUah,0)) }}</td>
              <td style="text-align:right" :style="debtColor(sortedSuppliers.reduce((s,r)=>s+r.debtUah,0))">{{ fmtFull(sortedSuppliers.reduce((s,r)=>s+r.debtUah,0)) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- COUNTERPARTIES REPORT -->
      <div v-if="reportsSubTab==='counterparties' && counterpartyReport && !isLoading">
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:0.75rem;margin-bottom:1.2rem">
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Контрагентів</div>
            <div style="font-size:1.4rem;font-weight:700">{{ counterpartyReport.rows.length }}</div>
          </div>
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Продажі (UAH)</div>
            <div style="font-size:1.2rem;font-weight:700;color:#9fe8c4">{{ fmtFull(counterpartyReport.totalSalesUah) }}</div>
          </div>
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Закупівлі (UAH)</div>
            <div style="font-size:1.2rem;font-weight:700;color:#b09adc">{{ fmtFull(counterpartyReport.totalPurchasedUah) }}</div>
          </div>
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Оплачено (UAH)</div>
            <div style="font-size:1.2rem;font-weight:700;color:#7ab4d4">{{ fmtFull(counterpartyReport.totalPaidUah) }}</div>
          </div>
          <div class="panel" style="padding:0.9rem 1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.3rem">Борг (UAH)</div>
            <div style="font-size:1.2rem;font-weight:700" :style="debtColor(counterpartyReport.totalDebtUah)">{{ fmtFull(counterpartyReport.totalDebtUah) }}</div>
          </div>
        </div>

        <div style="display:flex;gap:0.6rem;align-items:center;margin-bottom:0.8rem;flex-wrap:wrap">
          <input v-model="reportsSearch" placeholder="Пошук контрагента…" style="max-width:260px" />
          <span class="subtle" style="font-size:0.82rem;margin-left:auto">{{ sortedCounterparties.length }} / {{ counterpartyReport.rows.length }}</span>
        </div>

        <table>
          <thead>
            <tr>
              <th style="cursor:pointer" @click="setReportsSort('counterpartyName')">Контрагент {{ sortArrow('counterpartyName') }}</th>
              <th>Тип</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('salesUah')">Продажі (UAH) {{ sortArrow('salesUah') }}</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('purchasedUah')">Закупівлі (UAH) {{ sortArrow('purchasedUah') }}</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('paidUah')">Оплачено (UAH) {{ sortArrow('paidUah') }}</th>
              <th style="cursor:pointer;text-align:right" @click="setReportsSort('debtUah')">Борг (UAH) {{ sortArrow('debtUah') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in sortedCounterparties" :key="row.counterpartyId">
              <td style="font-weight:500">{{ row.counterpartyName }}</td>
              <td>
                <span v-if="row.isCustomer" style="font-size:0.75rem;padding:0.1rem 0.4rem;border-radius:12px;background:rgba(122,180,212,0.15);color:#7ab4d4;margin-right:0.2rem">Покупець</span>
                <span v-if="row.isSupplier" style="font-size:0.75rem;padding:0.1rem 0.4rem;border-radius:12px;background:rgba(176,154,220,0.15);color:#b09adc">Постачальник</span>
              </td>
              <td style="text-align:right;font-weight:600;color:#9fe8c4">{{ fmtFull(row.salesUah) }}</td>
              <td style="text-align:right;color:#b09adc">{{ fmtFull(row.purchasedUah) }}</td>
              <td style="text-align:right;color:#7ab4d4">{{ fmtFull(row.paidUah) }}</td>
              <td style="text-align:right" :style="debtColor(row.debtUah)">{{ fmtFull(row.debtUah) }}</td>
            </tr>
            <tr v-if="!sortedCounterparties.length">
              <td colspan="6" class="subtle" style="text-align:center;padding:1.5rem">{{ reportsSearch ? 'Нічого не знайдено' : 'Немає даних' }}</td>
            </tr>
            <tr v-if="sortedCounterparties.length" style="background:rgba(255,255,255,0.04);font-weight:700;border-top:2px solid rgba(255,255,255,0.12)">
              <td colspan="2">Разом</td>
              <td style="text-align:right;color:#9fe8c4">{{ fmtFull(sortedCounterparties.reduce((s,r)=>s+r.salesUah,0)) }}</td>
              <td style="text-align:right;color:#b09adc">{{ fmtFull(sortedCounterparties.reduce((s,r)=>s+r.purchasedUah,0)) }}</td>
              <td style="text-align:right;color:#7ab4d4">{{ fmtFull(sortedCounterparties.reduce((s,r)=>s+r.paidUah,0)) }}</td>
              <td style="text-align:right" :style="debtColor(sortedCounterparties.reduce((s,r)=>s+r.debtUah,0))">{{ fmtFull(sortedCounterparties.reduce((s,r)=>s+r.debtUah,0)) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <p v-if="isLoading" class="subtle" style="margin-top:1rem">Завантаження...</p>
    </template>

  </main>
</template>