<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import type {
  UserSession, Customer, Supplier, CustomerOrder, Sale,
  DebtSummary, Payment, Purchase, SupplierOrder, Cashbox,
  CurrencyCode, DebtPaymentHistoryEntry
} from "../types";

const props = defineProps<{ session: UserSession }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);
const can = (p: string) => props.session.permissions?.includes(p) ?? false;

// ── Data ──────────────────────────────────────────────────────────────────
const customers      = ref<Customer[]>([]);
const suppliers      = ref<Supplier[]>([]);
const orders         = ref<CustomerOrder[]>([]);
const sales          = ref<Sale[]>([]);
const purchases      = ref<Purchase[]>([]);
const supplierOrders = ref<SupplierOrder[]>([]);
const debts          = ref<DebtSummary[]>([]);
const cashboxes      = ref<Cashbox[]>([]);

const loading = ref(false);
const error   = ref("");

// ── UI state ──────────────────────────────────────────────────────────────
// Tabs: "all" | "customers" | "suppliers"
const activeTab = ref<"all" | "customers" | "suppliers">("all");
const searchAll       = ref("");
const searchCustomers = ref("");
const searchSuppliers = ref("");

// Selected unified counterparty card
type CounterpartyType = "customer" | "supplier" | "both";
interface Counterparty {
  key: string;
  id: number;
  name: string;
  phone?: string;
  email?: string;
  comment?: string;
  roles: CounterpartyType;
  customerId?: number;
  supplierId?: number;
}

const selected = ref<Counterparty | null>(null);
const selectedCustomer = ref<Customer | null>(null);
const selectedSupplier = ref<Supplier | null>(null);

// Card detail data
const cardLoading        = ref(false);
const cardPayments       = ref<Payment[]>([]);
const cardDebtHistory    = ref<Map<number, DebtPaymentHistoryEntry[]>>(new Map());

// Forms
const showNewCustomer = ref(false);
const custForm = ref({ name: "", phone: "", email: "", comment: "" });
const showNewSupplier = ref(false);
const supForm  = ref({ name: "", contact: "", phone: "", email: "", comments: "" });

// Payment modal
const showPayment  = ref(false);
const paymentForm  = ref({
  entityType: "order" as "order" | "sale",
  entityId: 0,
  cashboxId: 0,
  amount: 0,
  currency: "UAH" as CurrencyCode,
  method: "cash" as any,
  note: "",
  description: "",
});
const saving = ref(false);

// ── Load ──────────────────────────────────────────────────────────────────
async function load() {
  loading.value = true; error.value = "";
  try {
    [customers.value, suppliers.value, orders.value, sales.value, debts.value, cashboxes.value] =
      await Promise.all([
        api.customers(token.value),
        api.suppliers(token.value),
        api.orders(token.value),
        api.sales(token.value),
        api.debts(token.value),
        api.cashboxes(token.value),
      ]);
    try { supplierOrders.value = await api.supplierOrders(token.value); } catch {}
    try { purchases.value = (await api.purchases(token.value)) as Purchase[]; } catch {}
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

// ── Unified counterparty list ─────────────────────────────────────────────
const counterparties = computed<Counterparty[]>(() => {
  const map = new Map<string, Counterparty>();

  customers.value.forEach(c => {
    const key = `c-${c.id}`;
    map.set(key, {
      key, id: c.id, name: c.name, phone: c.phone, email: c.email,
      comment: c.comment, roles: "customer", customerId: c.id,
    });
  });

  suppliers.value.forEach(s => {
    const nameNorm = s.name.trim().toLowerCase();
    const existing = [...map.values()].find(
      cp => cp.customerId && cp.name.trim().toLowerCase() === nameNorm
    );
    if (existing) {
      existing.roles = "both";
      existing.supplierId = s.id;
      existing.phone = existing.phone || s.phone;
      existing.email = existing.email || s.email;
    } else {
      const key = `s-${s.id}`;
      map.set(key, {
        key, id: s.id, name: s.name, phone: s.phone, email: s.email,
        comment: s.comments, roles: "supplier", supplierId: s.id,
      });
    }
  });

  return [...map.values()].sort((a, b) => a.name.localeCompare(b.name, "uk"));
});

const filteredAll = computed(() => {
  const s = searchAll.value.toLowerCase();
  if (!s) return counterparties.value;
  return counterparties.value.filter(c =>
    c.name.toLowerCase().includes(s) ||
    (c.phone || "").includes(s) ||
    (c.email || "").toLowerCase().includes(s)
  );
});

// ── Filtered customer list ────────────────────────────────────────────────
const filteredCustomers = computed(() => {
  const s = searchCustomers.value.toLowerCase();
  if (!s) return customers.value.slice().sort((a, b) => a.name.localeCompare(b.name, "uk"));
  return customers.value
    .filter(c =>
      c.name.toLowerCase().includes(s) ||
      c.phone.includes(s) ||
      c.email.toLowerCase().includes(s)
    )
    .sort((a, b) => a.name.localeCompare(b.name, "uk"));
});

// ── Filtered supplier list ────────────────────────────────────────────────
const filteredSuppliers = computed(() => {
  const s = searchSuppliers.value.toLowerCase();
  if (!s) return suppliers.value.slice().sort((a, b) => a.name.localeCompare(b.name, "uk"));
  return suppliers.value
    .filter(c =>
      c.name.toLowerCase().includes(s) ||
      c.phone.includes(s) ||
      c.email.toLowerCase().includes(s) ||
      c.contact.toLowerCase().includes(s)
    )
    .sort((a, b) => a.name.localeCompare(b.name, "uk"));
});

// ── Per-counterparty computeds ────────────────────────────────────────────
function cpOrders(cp: Counterparty): CustomerOrder[] {
  if (!cp.customerId) return [];
  const cust = customers.value.find(c => c.id === cp.customerId);
  if (!cust) return [];
  return orders.value.filter(o => o.customerName === cust.name);
}

function cpSales(cp: Counterparty): Sale[] {
  const orderIds = new Set(cpOrders(cp).map(o => o.id));
  return sales.value.filter(s => s.orderId && orderIds.has(s.orderId));
}

function cpSupplierOrders(cp: Counterparty): SupplierOrder[] {
  if (!cp.supplierId) return [];
  return supplierOrders.value.filter(o => o.supplierId === cp.supplierId);
}

function cpPurchases(cp: Counterparty): Purchase[] {
  if (!cp.supplierId) return [];
  return purchases.value.filter(p => p.supplierId === cp.supplierId);
}

function cpDebtOwedByCustomer(cp: Counterparty): number {
  const orderIds = new Set(cpOrders(cp).map(o => o.id));
  return debts.value
    .filter(d => d.entityType === "order" && orderIds.has(d.entityId) && d.debt > 0)
    .reduce((a, d) => a + d.debtUah, 0);
}

function cpDebtOwedToSupplier(cp: Counterparty): number {
  return cpPurchases(cp).reduce((a, p) => a + p.totalUah, 0);
}

function cpTotalSalesUah(cp: Counterparty): number {
  return cpSales(cp).reduce((a, s) => a + s.totalUah, 0);
}

function cpTotalPurchasesUah(cp: Counterparty): number {
  return cpPurchases(cp).reduce((a, p) => a + p.totalUah, 0);
}

function cpOrderDebts(cp: Counterparty) {
  const orderIds = new Set(cpOrders(cp).map(o => o.id));
  return debts.value
    .filter(d => d.entityType === "order" && orderIds.has(d.entityId) && d.debt > 0)
    .map(d => ({
      debt: d,
      order: orders.value.find(o => o.id === d.entityId)!,
    }))
    .filter(x => x.order);
}

// ── Customer-specific helpers ─────────────────────────────────────────────
function customerToCounterparty(c: Customer): Counterparty {
  return {
    key: `c-${c.id}`, id: c.id, name: c.name, phone: c.phone,
    email: c.email, comment: c.comment, roles: "customer", customerId: c.id,
  };
}

function supplierToCounterparty(s: Supplier): Counterparty {
  return {
    key: `s-${s.id}`, id: s.id, name: s.name, phone: s.phone,
    email: s.email, comment: s.comments, roles: "supplier", supplierId: s.id,
  };
}

function customerOrderCount(c: Customer): number {
  return orders.value.filter(o => o.customerName === c.name).length;
}

function customerDebt(c: Customer): number {
  const cp = customerToCounterparty(c);
  return cpDebtOwedByCustomer(cp);
}

function supplierPurchaseCount(s: Supplier): number {
  return purchases.value.filter(p => p.supplierId === s.id).length;
}

// ── Open counterparty card ────────────────────────────────────────────────
async function openCard(cp: Counterparty) {
  selected.value = cp;
  selectedCustomer.value = null;
  selectedSupplier.value = null;
  await loadCardData(cp);
}

async function openCustomerCard(c: Customer) {
  selectedCustomer.value = c;
  selectedSupplier.value = null;
  selected.value = customerToCounterparty(c);
  await loadCardData(selected.value);
}

async function openSupplierCard(s: Supplier) {
  selectedSupplier.value = s;
  selectedCustomer.value = null;
  selected.value = supplierToCounterparty(s);
  await loadCardData(selected.value);
}

async function loadCardData(cp: Counterparty) {
  cardLoading.value = true;
  cardPayments.value = [];
  cardDebtHistory.value = new Map();

  try {
    const orderIds = cpOrders(cp).map(o => o.id).slice(0, 15);
    const allPayments: Payment[] = [];
    for (const id of orderIds) {
      try { allPayments.push(...await api.paymentsForOrder(token.value, id)); } catch {}
    }
    cardPayments.value = allPayments;

    const orderDebts = cpOrderDebts(cp);
    for (const { debt } of orderDebts.slice(0, 5)) {
      try {
        const history = await api.debtPaymentHistory(token.value, "order", debt.entityId);
        cardDebtHistory.value.set(debt.entityId, history);
      } catch {}
    }
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    cardLoading.value = false;
  }
}

function closeCard() {
  selected.value = null;
  selectedCustomer.value = null;
  selectedSupplier.value = null;
}

function switchTab(tab: "all" | "customers" | "suppliers") {
  activeTab.value = tab;
  closeCard();
}

// ── Payment ───────────────────────────────────────────────────────────────
function openPaymentModal(
  entityType: "order" | "sale",
  entityId: number,
  amount: number,
  currency: CurrencyCode,
  description: string
) {
  paymentForm.value = {
    entityType, entityId,
    cashboxId: cashboxes.value[0]?.id || 0,
    amount, currency, method: "cash", note: "",
    description,
  };
  showPayment.value = true;
}

async function doPayment() {
  saving.value = true; error.value = "";
  try {
    const payload: any = {
      cashboxId: paymentForm.value.cashboxId || undefined,
      amount: paymentForm.value.amount,
      currency: paymentForm.value.currency,
      method: paymentForm.value.method,
      note: paymentForm.value.note,
    };
    if (paymentForm.value.entityType === "order") payload.orderId = paymentForm.value.entityId;
    else payload.saleId = paymentForm.value.entityId;
    await api.createPayment(token.value, payload);
    showPayment.value = false;
    await load();
    if (selected.value) await loadCardData(selected.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

// ── Create forms ──────────────────────────────────────────────────────────
async function createCustomer() {
  saving.value = true; error.value = "";
  try {
    await api.createCustomer(token.value, custForm.value);
    showNewCustomer.value = false;
    custForm.value = { name: "", phone: "", email: "", comment: "" };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function createSupplier() {
  saving.value = true; error.value = "";
  try {
    await api.createSupplier(token.value, supForm.value);
    showNewSupplier.value = false;
    supForm.value = { name: "", contact: "", phone: "", email: "", comments: "" };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

// ── Helpers ───────────────────────────────────────────────────────────────
const statusLabel: Record<string, string> = {
  new: "Нове", in_work: "В роботі", ordered: "Замовлено у пост.",
  expected: "Очікується", arrived: "Надійшло", issued: "Видано",
  closed: "Закрито", cancelled: "Скасовано",
  draft: "Створено", sent: "Відправлено", confirmed: "Підтверджено",
  in_transit: "В дорозі", received: "Отримано",
  processing: "В обробці", paid: "Оплачено", completed: "Завершено",
  partially_received: "В дорозі",
};

function roleLabel(r: CounterpartyType) {
  if (r === "both") return "Покупець + Постачальник";
  if (r === "customer") return "Покупець";
  return "Постачальник";
}

function roleBadgeStyle(r: CounterpartyType) {
  if (r === "both") return "background:rgba(120,100,220,0.13);color:#a080f0;";
  if (r === "customer") return "background:rgba(72,187,120,0.13);color:#48bb78;";
  return "background:rgba(90,180,220,0.13);color:#60c0e0;";
}

function fmt(n: number) { return n.toLocaleString("uk-UA", { maximumFractionDigits: 2 }); }
function fmtDate(s: string) { return new Date(s).toLocaleDateString("uk"); }

onMounted(load);
</script>

<template>
  <div class="page-content">
    <!-- ── Header ── -->
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Контрагенти</h2>
      <div style="display:flex;gap:0.5rem">
        <button class="ghost-button" @click="showNewCustomer=true">+ Покупець</button>
        <button class="ghost-button" @click="showNewSupplier=true">+ Постачальник</button>
      </div>
    </div>

    <p v-if="error"   class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <!-- ── Tabs ── -->
    <div class="tab-row" style="margin-bottom:1rem">
      <button :class="['tab-button', activeTab==='all'&&'tab-button--active']"       @click="switchTab('all')">
        Всі <span style="font-size:0.75em;opacity:0.7">({{ counterparties.length }})</span>
      </button>
      <button :class="['tab-button', activeTab==='customers'&&'tab-button--active']" @click="switchTab('customers')">
        👥 Покупці <span style="font-size:0.75em;opacity:0.7">({{ customers.length }})</span>
      </button>
      <button :class="['tab-button', activeTab==='suppliers'&&'tab-button--active']" @click="switchTab('suppliers')">
        🏭 Постачальники <span style="font-size:0.75em;opacity:0.7">({{ suppliers.length }})</span>
      </button>
    </div>

    <!-- ══════════════════════════════════════════════════════════════════ -->
    <!-- TAB: Всі контрагенти -->
    <!-- ══════════════════════════════════════════════════════════════════ -->
    <div v-if="activeTab==='all' && !loading">
      <input v-model="searchAll" placeholder="🔍 Пошук за ім'ям, телефоном, email..." style="width:100%;margin-bottom:1rem;box-sizing:border-box">

      <div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;align-items:start">
        <!-- LIST -->
        <div>
          <div v-if="!filteredAll.length" class="subtle">Контрагентів не знайдено</div>
          <div v-for="cp in filteredAll" :key="cp.key"
            class="panel" style="padding:0.85rem;margin-bottom:0.55rem;cursor:pointer;transition:border-color 0.15s"
            :style="selected?.key === cp.key ? 'border-color:var(--accent)' : ''"
            @click="openCard(cp)">
            <div style="display:flex;justify-content:space-between;align-items:flex-start">
              <div style="flex:1;min-width:0">
                <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap;margin-bottom:0.2rem">
                  <div style="font-weight:600;font-size:0.95rem">{{ cp.name }}</div>
                  <span style="font-size:0.68rem;padding:0.1rem 0.45rem;border-radius:20px;white-space:nowrap"
                    :style="roleBadgeStyle(cp.roles)">
                    {{ roleLabel(cp.roles) }}
                  </span>
                </div>
                <div class="subtle" style="font-size:0.78rem">
                  {{ cp.phone || '—' }}
                  <span v-if="cp.email" style="margin-left:0.3rem">· {{ cp.email }}</span>
                </div>
              </div>
              <div style="text-align:right;flex-shrink:0;margin-left:0.5rem">
                <div v-if="cpDebtOwedByCustomer(cp) > 0"
                  style="color:#e07070;font-weight:700;font-size:0.85rem;white-space:nowrap">
                  ↑ {{ fmt(cpDebtOwedByCustomer(cp)) }} ₴
                </div>
                <div class="subtle" style="font-size:0.72rem">
                  <span v-if="cp.customerId">{{ cpOrders(cp).length }} замов.</span>
                  <span v-if="cp.supplierId" style="margin-left:0.3rem">{{ cpPurchases(cp).length }} надх.</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- CARD -->
        <div v-if="selected" style="position:sticky;top:1rem">
          <template v-if="selected">
            <div class="panel" style="padding:1.1rem">
              <!-- include shared card template -->
              <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:1rem">
                <div>
                  <h3 style="margin:0 0 0.25rem">{{ selected.name }}</h3>
                  <span style="font-size:0.72rem;padding:0.1rem 0.5rem;border-radius:20px"
                    :style="roleBadgeStyle(selected.roles)">
                    {{ roleLabel(selected.roles) }}
                  </span>
                </div>
                <button class="ghost-button" style="padding:0.3rem 0.6rem" @click="closeCard">✕</button>
              </div>
              <!-- Contact info -->
              <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.5rem">КОНТАКТНА ІНФОРМАЦІЯ</div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.4rem 1rem;font-size:0.84rem;margin-bottom:1rem">
                <div>
                  <div class="subtle" style="font-size:0.68rem;margin-bottom:0.1rem">ТЕЛЕФОН</div>
                  <div>{{ selected.phone || '—' }}</div>
                </div>
                <div>
                  <div class="subtle" style="font-size:0.68rem;margin-bottom:0.1rem">EMAIL</div>
                  <div>{{ selected.email || '—' }}</div>
                </div>
              </div>
              <!-- Roles badges -->
              <div style="display:flex;gap:0.4rem;margin-bottom:1rem;flex-wrap:wrap">
                <span v-if="selected.customerId" style="display:inline-flex;align-items:center;gap:0.3rem;font-size:0.8rem;padding:0.25rem 0.65rem;border-radius:20px;background:rgba(72,187,120,0.1);color:#48bb78">✓ Покупець</span>
                <span v-if="selected.supplierId" style="display:inline-flex;align-items:center;gap:0.3rem;font-size:0.8rem;padding:0.25rem 0.65rem;border-radius:20px;background:rgba(90,180,220,0.1);color:#60c0e0">✓ Постачальник</span>
              </div>
              <!-- KPI -->
              <div style="display:grid;gap:0.5rem;margin-bottom:1rem"
                :style="selected.roles==='both' ? 'grid-template-columns:repeat(2,1fr)' : 'grid-template-columns:repeat(3,1fr)'">
                <template v-if="selected.customerId">
                  <div style="background:rgba(72,187,120,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                    <div style="font-size:1.05rem;font-weight:700;color:var(--accent)">{{ cpOrders(selected).length }}</div>
                    <div class="subtle" style="font-size:0.68rem">Замовлень</div>
                  </div>
                  <div style="background:rgba(72,187,120,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                    <div style="font-size:1.05rem;font-weight:700;color:var(--accent)">{{ fmt(cpTotalSalesUah(selected)) }}</div>
                    <div class="subtle" style="font-size:0.68rem">Продажів ₴</div>
                  </div>
                </template>
                <template v-if="selected.supplierId">
                  <div style="background:rgba(90,180,220,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                    <div style="font-size:1.05rem;font-weight:700;color:#60c0e0">{{ cpPurchases(selected).length }}</div>
                    <div class="subtle" style="font-size:0.68rem">Закупівель</div>
                  </div>
                  <div style="background:rgba(90,180,220,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                    <div style="font-size:1.05rem;font-weight:700;color:#60c0e0">{{ fmt(cpTotalPurchasesUah(selected)) }}</div>
                    <div class="subtle" style="font-size:0.68rem">Закупівель ₴</div>
                  </div>
                </template>
              </div>
              <!-- Debts -->
              <div style="margin-bottom:1rem">
                <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.5rem">БОРГИ</div>
                <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.4rem;margin-bottom:0.75rem">
                  <div v-if="selected.customerId"
                    :style="`border-radius:8px;padding:0.65rem;text-align:center;border:1px solid;${cpDebtOwedByCustomer(selected)>0 ? 'background:rgba(224,112,112,0.08);border-color:rgba(224,112,112,0.3)' : 'background:rgba(72,187,120,0.06);border-color:rgba(72,187,120,0.2)'}`">
                    <div :style="`font-size:1rem;font-weight:700;${cpDebtOwedByCustomer(selected)>0?'color:#e07070':'color:var(--accent)'}`">
                      {{ fmt(cpDebtOwedByCustomer(selected)) }} ₴
                    </div>
                    <div class="subtle" style="font-size:0.68rem">Нам винні</div>
                  </div>
                  <div v-if="selected.supplierId"
                    style="border-radius:8px;padding:0.65rem;text-align:center;border:1px solid rgba(90,180,220,0.25);background:rgba(90,180,220,0.06)">
                    <div style="font-size:1rem;font-weight:700;color:#60c0e0">{{ fmt(cpDebtOwedToSupplier(selected)) }} ₴</div>
                    <div class="subtle" style="font-size:0.68rem">Ми винні</div>
                  </div>
                </div>
                <div v-for="{debt, order} in cpOrderDebts(selected)" :key="debt.entityId"
                  style="border-radius:8px;padding:0.6rem 0.8rem;background:rgba(224,112,112,0.07);border:1px solid rgba(224,112,112,0.18);margin-bottom:0.4rem">
                  <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:0.3rem">
                    <div>
                      <div style="font-size:0.84rem;font-weight:600">📋 Замовлення #{{ order.id }}</div>
                      <div class="subtle" style="font-size:0.72rem">{{ statusLabel[order.status] }} · {{ fmtDate(order.createdAt) }} · Сума: {{ fmt(order.totalUah) }} ₴</div>
                    </div>
                    <div style="text-align:right">
                      <div style="color:#e07070;font-weight:700;font-size:0.9rem">{{ fmt(debt.debtUah) }} ₴</div>
                      <div class="subtle" style="font-size:0.68rem">борг</div>
                    </div>
                  </div>
                  <div v-if="cardDebtHistory.get(debt.entityId)?.length" style="margin:0.35rem 0;padding:0.4rem 0.6rem;background:rgba(0,0,0,0.12);border-radius:5px">
                    <div class="subtle" style="font-size:0.68rem;margin-bottom:0.2rem">ОПЛАТИ ЗА ЦИМ ЗАМОВЛЕННЯМ:</div>
                    <div v-for="h in cardDebtHistory.get(debt.entityId)" :key="h.paymentId"
                      style="display:flex;justify-content:space-between;font-size:0.78rem;padding:0.15rem 0">
                      <span style="color:#6dd4a0">✓ +{{ fmt(h.amount) }} {{ h.currency }} · {{ h.method }}</span>
                      <span class="subtle">{{ fmtDate(h.createdAt) }}</span>
                    </div>
                  </div>
                  <div style="display:flex;justify-content:flex-end;margin-top:0.3rem">
                    <button class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.7rem;border-color:rgba(224,112,112,0.4);color:#e07070"
                      @click="openPaymentModal('order', order.id, debt.debt, order.currency, `Замовлення #${order.id} від ${fmtDate(order.createdAt)}`)">
                      💳 Оплатити борг
                    </button>
                  </div>
                </div>
                <div v-if="!cpOrderDebts(selected).length && !cpDebtOwedByCustomer(selected) && !cpDebtOwedToSupplier(selected)"
                  class="subtle" style="font-size:0.82rem;padding:0.4rem 0">Заборгованостей немає ✓</div>
              </div>
              <!-- Orders (customer) -->
              <div v-if="selected.customerId" style="margin-bottom:1rem">
                <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.4rem">ЗАМОВЛЕННЯ</div>
                <div v-if="cardLoading" class="subtle" style="font-size:0.82rem">Завантаження...</div>
                <div v-for="o in cpOrders(selected).slice(0,10)" :key="o.id"
                  style="display:flex;justify-content:space-between;align-items:center;padding:0.35rem 0;border-bottom:1px solid rgba(122,185,154,0.1);font-size:0.83rem">
                  <div>
                    <span style="font-weight:500">#{{ o.id }}</span>
                    <span class="subtle" style="margin-left:0.4rem;font-size:0.75rem">{{ statusLabel[o.status] }}</span>
                  </div>
                  <div style="display:flex;align-items:center;gap:0.5rem">
                    <span>{{ fmt(o.total) }} {{ o.currency }}</span>
                    <span class="subtle" style="font-size:0.72rem">{{ fmtDate(o.createdAt) }}</span>
                  </div>
                </div>
                <div v-if="!cpOrders(selected).length" class="subtle" style="font-size:0.82rem">Замовлень немає</div>
              </div>
              <!-- Purchases (supplier) -->
              <div v-if="selected.supplierId" style="margin-bottom:1rem">
                <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.4rem">ЗАКУПІВЛІ / НАДХОДЖЕННЯ</div>
                <div v-for="p in cpPurchases(selected).slice(0,8)" :key="p.id"
                  style="display:flex;justify-content:space-between;align-items:center;padding:0.35rem 0;border-bottom:1px solid rgba(90,180,220,0.1);font-size:0.83rem">
                  <span>Надходження #{{ p.id }}</span>
                  <div style="display:flex;align-items:center;gap:0.5rem">
                    <span style="color:#60c0e0">{{ fmt(p.total) }} {{ p.currency }}</span>
                    <span class="subtle" style="font-size:0.72rem">{{ fmtDate(p.createdAt) }}</span>
                  </div>
                </div>
                <div v-if="!cpPurchases(selected).length" class="subtle" style="font-size:0.82rem">Закупівель немає</div>
              </div>
              <!-- Payments history -->
              <div v-if="cardPayments.length" style="margin-bottom:1rem">
                <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.4rem">ІСТОРІЯ ОПЛАТ</div>
                <div v-for="p in cardPayments.slice(0,10)" :key="p.id"
                  style="display:flex;justify-content:space-between;align-items:center;padding:0.35rem 0;border-bottom:1px solid rgba(122,185,154,0.1);font-size:0.83rem">
                  <div>
                    <span>Оплата #{{ p.id }}</span>
                    <span class="subtle" style="margin-left:0.35rem;font-size:0.72rem">{{ p.method }}</span>
                    <span v-if="p.orderId" class="subtle" style="margin-left:0.35rem;font-size:0.72rem">· Зам. #{{ p.orderId }}</span>
                  </div>
                  <div style="display:flex;align-items:center;gap:0.5rem">
                    <span style="color:#6dd4a0;font-weight:600">+{{ fmt(p.amount) }} {{ p.currency }}</span>
                    <span class="subtle" style="font-size:0.72rem">{{ fmtDate(p.createdAt) }}</span>
                  </div>
                </div>
              </div>
              <!-- Comment -->
              <div v-if="selected.comment"
                style="font-size:0.82rem;color:var(--text-subtle);font-style:italic;padding:0.5rem;background:rgba(122,185,154,0.05);border-radius:6px">
                {{ selected.comment }}
              </div>
            </div>
          </template>
        </div>

        <div v-else class="panel" style="padding:1.5rem;text-align:center;color:var(--text-subtle)">
          <div style="font-size:2rem;margin-bottom:0.5rem">🤝</div>
          Оберіть контрагента для перегляду повної картки
        </div>
      </div>
    </div>

    <!-- ══════════════════════════════════════════════════════════════════ -->
    <!-- TAB: Покупці -->
    <!-- ══════════════════════════════════════════════════════════════════ -->
    <div v-if="activeTab==='customers' && !loading">
      <!-- Summary row -->
      <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:0.75rem;margin-bottom:1rem">
        <div class="panel" style="padding:0.75rem;text-align:center">
          <div style="font-size:1.4rem;font-weight:700;color:var(--accent)">{{ customers.length }}</div>
          <div class="subtle" style="font-size:0.75rem">Всього покупців</div>
        </div>
        <div class="panel" style="padding:0.75rem;text-align:center">
          <div style="font-size:1.4rem;font-weight:700;color:var(--accent)">{{ orders.length }}</div>
          <div class="subtle" style="font-size:0.75rem">Замовлень</div>
        </div>
        <div class="panel" style="padding:0.75rem;text-align:center">
          <div style="font-size:1.1rem;font-weight:700;color:#e07070">
            {{ fmt(debts.filter(d=>d.entityType==='order'&&d.debt>0).reduce((a,d)=>a+d.debtUah,0)) }} ₴
          </div>
          <div class="subtle" style="font-size:0.75rem">Загальний борг</div>
        </div>
      </div>

      <input v-model="searchCustomers" placeholder="🔍 Пошук покупця за ім'ям, телефоном, email..." style="width:100%;margin-bottom:1rem;box-sizing:border-box">

      <div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;align-items:start">
        <!-- Customers list -->
        <div>
          <div v-if="!filteredCustomers.length" class="subtle">Покупців не знайдено</div>
          <div v-for="c in filteredCustomers" :key="c.id"
            class="panel" style="padding:0.85rem;margin-bottom:0.55rem;cursor:pointer;transition:border-color 0.15s"
            :style="selected?.customerId === c.id ? 'border-color:var(--accent)' : ''"
            @click="openCustomerCard(c)">
            <div style="display:flex;justify-content:space-between;align-items:flex-start">
              <div style="flex:1;min-width:0">
                <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.25rem">
                  <span style="font-size:1rem">👤</span>
                  <div style="font-weight:600;font-size:0.95rem">{{ c.name }}</div>
                </div>
                <div class="subtle" style="font-size:0.78rem;display:flex;flex-wrap:wrap;gap:0.5rem">
                  <span v-if="c.phone">📞 {{ c.phone }}</span>
                  <span v-if="c.email">✉️ {{ c.email }}</span>
                  <span v-if="!c.phone && !c.email">Контакти не вказані</span>
                </div>
              </div>
              <div style="text-align:right;flex-shrink:0;margin-left:0.5rem">
                <div v-if="customerDebt(c) > 0"
                  style="color:#e07070;font-weight:700;font-size:0.82rem;white-space:nowrap">
                  Борг: {{ fmt(customerDebt(c)) }} ₴
                </div>
                <div class="subtle" style="font-size:0.72rem;margin-top:0.1rem">
                  {{ customerOrderCount(c) }} замов.
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Customer detail card -->
        <div v-if="selected && activeTab==='customers'" style="position:sticky;top:1rem">
          <div class="panel" style="padding:1.1rem">
            <!-- Header -->
            <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:1rem">
              <div>
                <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.25rem">
                  <span style="font-size:1.4rem">👤</span>
                  <h3 style="margin:0">{{ selected.name }}</h3>
                </div>
                <span style="font-size:0.72rem;padding:0.15rem 0.55rem;border-radius:20px;background:rgba(72,187,120,0.13);color:#48bb78">
                  Покупець
                </span>
                <div class="subtle" style="font-size:0.72rem;margin-top:0.3rem">з {{ fmtDate(customers.find(c=>c.id===selected!.customerId)?.createdAt||'') }}</div>
              </div>
              <button class="ghost-button" style="padding:0.3rem 0.6rem" @click="closeCard">✕</button>
            </div>

            <!-- Contact info -->
            <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.5rem">КОНТАКТНА ІНФОРМАЦІЯ</div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem 1rem;font-size:0.84rem;margin-bottom:1rem">
              <div>
                <div class="subtle" style="font-size:0.68rem;margin-bottom:0.1rem">ТЕЛЕФОН</div>
                <div style="font-weight:500">{{ selected.phone || '—' }}</div>
              </div>
              <div>
                <div class="subtle" style="font-size:0.68rem;margin-bottom:0.1rem">EMAIL</div>
                <div style="font-weight:500">{{ selected.email || '—' }}</div>
              </div>
            </div>

            <!-- KPI -->
            <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:0.5rem;margin-bottom:1rem">
              <div style="background:rgba(72,187,120,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                <div style="font-size:1.1rem;font-weight:700;color:var(--accent)">{{ cpOrders(selected).length }}</div>
                <div class="subtle" style="font-size:0.65rem">Замовлень</div>
              </div>
              <div style="background:rgba(72,187,120,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                <div style="font-size:0.95rem;font-weight:700;color:var(--accent)">{{ fmt(cpTotalSalesUah(selected)) }}</div>
                <div class="subtle" style="font-size:0.65rem">Продажів ₴</div>
              </div>
              <div :style="`border-radius:8px;padding:0.65rem;text-align:center;${cpDebtOwedByCustomer(selected)>0?'background:rgba(224,112,112,0.08)':'background:rgba(72,187,120,0.07)'}`">
                <div :style="`font-size:0.95rem;font-weight:700;${cpDebtOwedByCustomer(selected)>0?'color:#e07070':'color:var(--accent)'}`">
                  {{ fmt(cpDebtOwedByCustomer(selected)) }}
                </div>
                <div class="subtle" style="font-size:0.65rem">Борг ₴</div>
              </div>
            </div>

            <!-- Debts detail -->
            <div v-if="cpOrderDebts(selected).length" style="margin-bottom:1rem">
              <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.5rem">АКТИВНІ БОРГИ</div>
              <div v-for="{debt, order} in cpOrderDebts(selected)" :key="debt.entityId"
                style="border-radius:8px;padding:0.6rem 0.8rem;background:rgba(224,112,112,0.07);border:1px solid rgba(224,112,112,0.18);margin-bottom:0.4rem">
                <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:0.3rem">
                  <div>
                    <div style="font-size:0.84rem;font-weight:600">📋 Замовлення #{{ order.id }}</div>
                    <div class="subtle" style="font-size:0.72rem">{{ statusLabel[order.status] }} · {{ fmtDate(order.createdAt) }}</div>
                  </div>
                  <div style="text-align:right">
                    <div style="color:#e07070;font-weight:700;font-size:0.9rem">{{ fmt(debt.debtUah) }} ₴</div>
                  </div>
                </div>
                <div v-if="cardDebtHistory.get(debt.entityId)?.length" style="margin:0.25rem 0;padding:0.35rem 0.5rem;background:rgba(0,0,0,0.12);border-radius:5px">
                  <div v-for="h in cardDebtHistory.get(debt.entityId)" :key="h.paymentId"
                    style="display:flex;justify-content:space-between;font-size:0.75rem">
                    <span style="color:#6dd4a0">✓ +{{ fmt(h.amount) }} {{ h.currency }}</span>
                    <span class="subtle">{{ fmtDate(h.createdAt) }}</span>
                  </div>
                </div>
                <button class="ghost-button" style="font-size:0.76rem;padding:0.2rem 0.6rem;border-color:rgba(224,112,112,0.4);color:#e07070;margin-top:0.3rem"
                  @click="openPaymentModal('order', order.id, debt.debt, order.currency, `Замовлення #${order.id}`)">
                  💳 Оплатити
                </button>
              </div>
            </div>

            <!-- Orders list -->
            <div style="margin-bottom:1rem">
              <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.4rem">
                ЗАМОВЛЕННЯ ({{ cpOrders(selected).length }})
              </div>
              <div v-if="cardLoading" class="subtle" style="font-size:0.82rem">Завантаження...</div>
              <div v-for="o in cpOrders(selected).slice(0,10)" :key="o.id"
                style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(122,185,154,0.1);font-size:0.83rem">
                <div>
                  <span style="font-weight:500">#{{ o.id }}</span>
                  <span style="margin-left:0.4rem;font-size:0.72rem;padding:0.1rem 0.4rem;border-radius:10px;background:rgba(122,185,154,0.1);color:var(--text-subtle)">
                    {{ statusLabel[o.status] }}
                  </span>
                </div>
                <div style="display:flex;align-items:center;gap:0.5rem">
                  <span style="font-weight:500">{{ fmt(o.total) }} {{ o.currency }}</span>
                  <span class="subtle" style="font-size:0.72rem">{{ fmtDate(o.createdAt) }}</span>
                </div>
              </div>
              <div v-if="!cpOrders(selected).length" class="subtle" style="font-size:0.82rem;padding:0.3rem 0">Замовлень немає</div>
              <div v-if="cpOrders(selected).length > 10" class="subtle" style="font-size:0.75rem;margin-top:0.3rem">
                + ще {{ cpOrders(selected).length - 10 }} замовлень
              </div>
            </div>

            <!-- Payment history -->
            <div v-if="cardPayments.length" style="margin-bottom:1rem">
              <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.4rem">
                ОПЛАТИ ({{ cardPayments.length }})
              </div>
              <div v-for="p in cardPayments.slice(0,8)" :key="p.id"
                style="display:flex;justify-content:space-between;align-items:center;padding:0.35rem 0;border-bottom:1px solid rgba(122,185,154,0.1);font-size:0.83rem">
                <div>
                  <span>Оплата #{{ p.id }}</span>
                  <span class="subtle" style="margin-left:0.35rem;font-size:0.72rem">{{ p.method }}</span>
                </div>
                <span style="color:#6dd4a0;font-weight:600">+{{ fmt(p.amount) }} {{ p.currency }}</span>
              </div>
            </div>

            <!-- Comment -->
            <div v-if="selected.comment"
              style="font-size:0.82rem;color:var(--text-subtle);font-style:italic;padding:0.5rem;background:rgba(122,185,154,0.05);border-radius:6px">
              💬 {{ selected.comment }}
            </div>
          </div>
        </div>

        <div v-else-if="activeTab==='customers'" class="panel" style="padding:1.5rem;text-align:center;color:var(--text-subtle)">
          <div style="font-size:2rem;margin-bottom:0.5rem">👥</div>
          Оберіть покупця для перегляду картки
        </div>
      </div>
    </div>

    <!-- ══════════════════════════════════════════════════════════════════ -->
    <!-- TAB: Постачальники -->
    <!-- ══════════════════════════════════════════════════════════════════ -->
    <div v-if="activeTab==='suppliers' && !loading">
      <!-- Summary row -->
      <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:0.75rem;margin-bottom:1rem">
        <div class="panel" style="padding:0.75rem;text-align:center">
          <div style="font-size:1.4rem;font-weight:700;color:#60c0e0">{{ suppliers.length }}</div>
          <div class="subtle" style="font-size:0.75rem">Всього постачальників</div>
        </div>
        <div class="panel" style="padding:0.75rem;text-align:center">
          <div style="font-size:1.4rem;font-weight:700;color:#60c0e0">{{ purchases.length }}</div>
          <div class="subtle" style="font-size:0.75rem">Надходжень</div>
        </div>
        <div class="panel" style="padding:0.75rem;text-align:center">
          <div style="font-size:1.1rem;font-weight:700;color:#60c0e0">
            {{ fmt(purchases.reduce((a,p)=>a+p.totalUah,0)) }} ₴
          </div>
          <div class="subtle" style="font-size:0.75rem">Закупівель загалом</div>
        </div>
      </div>

      <input v-model="searchSuppliers" placeholder="🔍 Пошук постачальника за назвою, контактом, телефоном..." style="width:100%;margin-bottom:1rem;box-sizing:border-box">

      <div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;align-items:start">
        <!-- Suppliers list -->
        <div>
          <div v-if="!filteredSuppliers.length" class="subtle">Постачальників не знайдено</div>
          <div v-for="s in filteredSuppliers" :key="s.id"
            class="panel" style="padding:0.85rem;margin-bottom:0.55rem;cursor:pointer;transition:border-color 0.15s"
            :style="selected?.supplierId === s.id ? 'border-color:#60c0e0' : ''"
            @click="openSupplierCard(s)">
            <div style="display:flex;justify-content:space-between;align-items:flex-start">
              <div style="flex:1;min-width:0">
                <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.25rem">
                  <span style="font-size:1rem">🏭</span>
                  <div style="font-weight:600;font-size:0.95rem">{{ s.name }}</div>
                </div>
                <div class="subtle" style="font-size:0.78rem;display:flex;flex-wrap:wrap;gap:0.5rem">
                  <span v-if="s.contact">👤 {{ s.contact }}</span>
                  <span v-if="s.phone">📞 {{ s.phone }}</span>
                  <span v-if="s.email">✉️ {{ s.email }}</span>
                  <span v-if="!s.contact && !s.phone && !s.email">Контакти не вказані</span>
                </div>
              </div>
              <div style="text-align:right;flex-shrink:0;margin-left:0.5rem">
                <div style="color:#60c0e0;font-weight:700;font-size:0.82rem;white-space:nowrap">
                  {{ fmt(purchases.filter(p=>p.supplierId===s.id).reduce((a,p)=>a+p.totalUah,0)) }} ₴
                </div>
                <div class="subtle" style="font-size:0.72rem;margin-top:0.1rem">
                  {{ supplierPurchaseCount(s) }} надх.
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Supplier detail card -->
        <div v-if="selected && activeTab==='suppliers'" style="position:sticky;top:1rem">
          <div class="panel" style="padding:1.1rem">
            <!-- Header -->
            <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:1rem">
              <div>
                <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.25rem">
                  <span style="font-size:1.4rem">🏭</span>
                  <h3 style="margin:0">{{ selected.name }}</h3>
                </div>
                <span style="font-size:0.72rem;padding:0.15rem 0.55rem;border-radius:20px;background:rgba(90,180,220,0.13);color:#60c0e0">
                  Постачальник
                </span>
              </div>
              <button class="ghost-button" style="padding:0.3rem 0.6rem" @click="closeCard">✕</button>
            </div>

            <!-- Contact info -->
            <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.5rem">КОНТАКТНА ІНФОРМАЦІЯ</div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem 1rem;font-size:0.84rem;margin-bottom:1rem">
              <div v-if="suppliers.find(s=>s.id===selected!.supplierId)?.contact">
                <div class="subtle" style="font-size:0.68rem;margin-bottom:0.1rem">КОНТАКТНА ОСОБА</div>
                <div style="font-weight:500">{{ suppliers.find(s=>s.id===selected!.supplierId)?.contact }}</div>
              </div>
              <div>
                <div class="subtle" style="font-size:0.68rem;margin-bottom:0.1rem">ТЕЛЕФОН</div>
                <div style="font-weight:500">{{ selected.phone || '—' }}</div>
              </div>
              <div>
                <div class="subtle" style="font-size:0.68rem;margin-bottom:0.1rem">EMAIL</div>
                <div style="font-weight:500">{{ selected.email || '—' }}</div>
              </div>
            </div>

            <!-- KPI -->
            <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:0.5rem;margin-bottom:1rem">
              <div style="background:rgba(90,180,220,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                <div style="font-size:1.1rem;font-weight:700;color:#60c0e0">{{ cpPurchases(selected).length }}</div>
                <div class="subtle" style="font-size:0.65rem">Закупівель</div>
              </div>
              <div style="background:rgba(90,180,220,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                <div style="font-size:0.95rem;font-weight:700;color:#60c0e0">{{ fmt(cpTotalPurchasesUah(selected)) }}</div>
                <div class="subtle" style="font-size:0.65rem">Сума ₴</div>
              </div>
              <div style="background:rgba(90,180,220,0.07);border-radius:8px;padding:0.65rem;text-align:center">
                <div style="font-size:1.1rem;font-weight:700;color:#60c0e0">{{ cpSupplierOrders(selected).length }}</div>
                <div class="subtle" style="font-size:0.65rem">Замовлень</div>
              </div>
            </div>

            <!-- Purchases list -->
            <div style="margin-bottom:1rem">
              <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.4rem">
                НАДХОДЖЕННЯ ({{ cpPurchases(selected).length }})
              </div>
              <div v-for="p in cpPurchases(selected).slice(0,10)" :key="p.id"
                style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(90,180,220,0.1);font-size:0.83rem">
                <span style="font-weight:500">Надходження #{{ p.id }}</span>
                <div style="display:flex;align-items:center;gap:0.5rem">
                  <span style="color:#60c0e0;font-weight:500">{{ fmt(p.total) }} {{ p.currency }}</span>
                  <span class="subtle" style="font-size:0.72rem">{{ fmtDate(p.createdAt) }}</span>
                </div>
              </div>
              <div v-if="!cpPurchases(selected).length" class="subtle" style="font-size:0.82rem;padding:0.3rem 0">Надходжень немає</div>
              <div v-if="cpPurchases(selected).length > 10" class="subtle" style="font-size:0.75rem;margin-top:0.3rem">
                + ще {{ cpPurchases(selected).length - 10 }} надходжень
              </div>
            </div>

            <!-- Supplier orders -->
            <div v-if="cpSupplierOrders(selected).length" style="margin-bottom:1rem">
              <div style="font-size:0.72rem;font-weight:600;letter-spacing:0.05em;color:var(--text-subtle);margin-bottom:0.4rem">
                ЗАМОВЛЕННЯ ПОСТАЧАЛЬНИКУ ({{ cpSupplierOrders(selected).length }})
              </div>
              <div v-for="o in cpSupplierOrders(selected).slice(0,8)" :key="o.id"
                style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(90,180,220,0.1);font-size:0.83rem">
                <div>
                  <span style="font-weight:500">#{{ o.id }}</span>
                  <span style="margin-left:0.4rem;font-size:0.72rem;padding:0.1rem 0.4rem;border-radius:10px;background:rgba(90,180,220,0.1);color:#60c0e0">
                    {{ statusLabel[o.status] || o.status }}
                  </span>
                </div>
                <div style="display:flex;align-items:center;gap:0.5rem">
                  <span style="color:#60c0e0;font-weight:500">{{ fmt(o.total) }} {{ o.currency }}</span>
                  <span class="subtle" style="font-size:0.72rem">{{ fmtDate(o.createdAt) }}</span>
                </div>
              </div>
            </div>

            <!-- Comment -->
            <div v-if="selected.comment"
              style="font-size:0.82rem;color:var(--text-subtle);font-style:italic;padding:0.5rem;background:rgba(90,180,220,0.05);border-radius:6px">
              💬 {{ selected.comment }}
            </div>
          </div>
        </div>

        <div v-else-if="activeTab==='suppliers'" class="panel" style="padding:1.5rem;text-align:center;color:var(--text-subtle)">
          <div style="font-size:2rem;margin-bottom:0.5rem">🏭</div>
          Оберіть постачальника для перегляду картки
        </div>
      </div>
    </div>

    <!-- ═══ MODAL: Новий покупець ═══ -->
    <div v-if="showNewCustomer" class="modal-backdrop" @click.self="showNewCustomer=false">
      <div class="panel modal-box">
        <h3>Новий покупець</h3>
        <div class="grid">
          <label>Ім'я / назва * <input v-model="custForm.name" autofocus></label>
          <label>Телефон <input v-model="custForm.phone"></label>
          <label>Email <input type="email" v-model="custForm.email"></label>
          <label>Коментар <textarea v-model="custForm.comment" rows="2"></textarea></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="createCustomer" :disabled="saving||!custForm.name">{{ saving?'..':'Зберегти' }}</button>
          <button class="ghost-button" @click="showNewCustomer=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- ═══ MODAL: Новий постачальник ═══ -->
    <div v-if="showNewSupplier" class="modal-backdrop" @click.self="showNewSupplier=false">
      <div class="panel modal-box">
        <h3>Новий постачальник</h3>
        <div class="grid">
          <label>Назва * <input v-model="supForm.name" autofocus></label>
          <label>Контактна особа <input v-model="supForm.contact"></label>
          <label>Телефон <input v-model="supForm.phone"></label>
          <label>Email <input type="email" v-model="supForm.email"></label>
          <label>Коментар <textarea v-model="supForm.comments" rows="2"></textarea></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="createSupplier" :disabled="saving||!supForm.name">{{ saving?'..':'Зберегти' }}</button>
          <button class="ghost-button" @click="showNewSupplier=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- ═══ MODAL: Оплата боргу ═══ -->
    <div v-if="showPayment" class="modal-backdrop" @click.self="showPayment=false">
      <div class="panel modal-box">
        <h3>💳 Оплата боргу</h3>
        <div v-if="paymentForm.description" style="font-size:0.82rem;color:var(--text-subtle);margin-bottom:0.75rem;padding:0.5rem;background:rgba(122,185,154,0.06);border-radius:6px">
          📋 {{ paymentForm.description }}
        </div>
        <div class="grid">
          <label>Каса
            <select v-model.number="paymentForm.cashboxId">
              <option v-for="c in cashboxes" :key="c.id" :value="c.id">{{ c.name }} ({{ c.currency }})</option>
            </select>
          </label>
          <label>Сума <input type="number" min="0" v-model.number="paymentForm.amount"></label>
          <label>Валюта
            <select v-model="paymentForm.currency">
              <option value="UAH">UAH (₴)</option>
              <option value="USD">USD ($)</option>
              <option value="EUR">EUR (€)</option>
            </select>
          </label>
          <label>Метод оплати
            <select v-model="paymentForm.method">
              <option value="cash">Готівка</option>
              <option value="card">Картка</option>
              <option value="bank">Банківський переказ</option>
            </select>
          </label>
          <label>Примітка <input v-model="paymentForm.note" placeholder="Необов'язково"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="doPayment" :disabled="saving||!paymentForm.amount">{{ saving?'Обробка...':'✓ Провести оплату' }}</button>
          <button class="ghost-button" @click="showPayment=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

  </div>
</template>
