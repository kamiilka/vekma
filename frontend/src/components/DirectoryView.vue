<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { api } from "../api";
import type {
  UserSession, Customer, Supplier, CustomerOrder, Sale,
  DebtSummary, Payment, Purchase, SupplierOrder, Cashbox,
  CurrencyCode, DebtPaymentHistoryEntry
} from "../types";

const props = defineProps<{ session: UserSession; initialSubTab?: string }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);

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

// ── UI ────────────────────────────────────────────────────────────────────
const activeSubTab = ref<"customers" | "suppliers">(
  props.initialSubTab === "suppliers" ? "suppliers" : "customers"
);
const searchCustomers = ref("");
const searchSuppliers = ref("");

// Modal
const modalOpen    = ref(false);
const modalLoading = ref(false);

// Payment modal
const showPayment = ref(false);
const paymentForm = ref({
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

// Forms
const showNewCustomer = ref(false);
const custForm = ref({ name: "", phone: "", email: "", comment: "" });
const showNewSupplier = ref(false);
const supForm  = ref({ name: "", contact: "", phone: "", email: "", comments: "" });

// ── Types ─────────────────────────────────────────────────────────────────
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
  contact?: string;
}

const selected = ref<Counterparty | null>(null);
const cardPayments    = ref<Payment[]>([]);
const cardDebtHistory = ref<Map<number, DebtPaymentHistoryEntry[]>>(new Map());

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

// ── Filtered lists ────────────────────────────────────────────────────────
const filteredCustomers = computed(() => {
  const s = searchCustomers.value.toLowerCase();
  return customers.value
    .filter(c => !s || c.name.toLowerCase().includes(s) || c.phone.includes(s) || c.email.toLowerCase().includes(s))
    .sort((a, b) => a.name.localeCompare(b.name, "uk"));
});

const filteredSuppliers = computed(() => {
  const s = searchSuppliers.value.toLowerCase();
  return suppliers.value
    .filter(c => !s || c.name.toLowerCase().includes(s) || c.phone.includes(s) || c.email.toLowerCase().includes(s) || c.contact.toLowerCase().includes(s))
    .sort((a, b) => a.name.localeCompare(b.name, "uk"));
});

// ── Per-counterparty helpers ──────────────────────────────────────────────
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

function cpPurchases(cp: Counterparty): Purchase[] {
  if (!cp.supplierId) return [];
  return purchases.value.filter(p => p.supplierId === cp.supplierId);
}

function cpSupplierOrders(cp: Counterparty): SupplierOrder[] {
  if (!cp.supplierId) return [];
  return supplierOrders.value.filter(o => o.supplierId === cp.supplierId);
}

function cpDebtOwedByCustomer(cp: Counterparty): number {
  const orderIds = new Set(cpOrders(cp).map(o => o.id));
  return debts.value
    .filter(d => d.entityType === "order" && orderIds.has(d.entityId) && d.debt > 0)
    .reduce((a, d) => a + d.debtUah, 0);
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
    .map(d => ({ debt: d, order: orders.value.find(o => o.id === d.entityId)! }))
    .filter(x => x.order);
}

// ── Open modal ────────────────────────────────────────────────────────────
async function openCustomer(c: Customer) {
  selected.value = {
    key: `c-${c.id}`, id: c.id, name: c.name, phone: c.phone,
    email: c.email, comment: c.comment, roles: "customer", customerId: c.id,
  };
  modalOpen.value = true;
  await loadCard();
}

async function openSupplier(s: Supplier) {
  selected.value = {
    key: `s-${s.id}`, id: s.id, name: s.name, phone: s.phone,
    email: s.email, comment: s.comments, roles: "supplier", supplierId: s.id,
    contact: s.contact,
  };
  modalOpen.value = true;
  await loadCard();
}

async function loadCard() {
  if (!selected.value) return;
  modalLoading.value = true;
  cardPayments.value = [];
  cardDebtHistory.value = new Map();
  try {
    const cp = selected.value;
    const orderIds = cpOrders(cp).map(o => o.id).slice(0, 15);
    const allPayments: Payment[] = [];
    for (const id of orderIds) {
      try { allPayments.push(...await api.paymentsForOrder(token.value, id)); } catch {}
    }
    cardPayments.value = allPayments;
    for (const { debt } of cpOrderDebts(cp).slice(0, 5)) {
      try {
        const history = await api.debtPaymentHistory(token.value, "order", debt.entityId);
        cardDebtHistory.value.set(debt.entityId, history);
      } catch {}
    }
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  } finally { modalLoading.value = false; }
}

function closeModal() { modalOpen.value = false; selected.value = null; }

// ── Payment ───────────────────────────────────────────────────────────────
function openPaymentModal(entityType: "order"|"sale", entityId: number, amount: number, currency: CurrencyCode, description: string) {
  paymentForm.value = { entityType, entityId, cashboxId: cashboxes.value[0]?.id || 0, amount, currency, method: "cash", note: "", description };
  showPayment.value = true;
}

async function doPayment() {
  saving.value = true; error.value = "";
  try {
    const payload: any = { cashboxId: paymentForm.value.cashboxId || undefined, amount: paymentForm.value.amount, currency: paymentForm.value.currency, method: paymentForm.value.method, note: paymentForm.value.note };
    if (paymentForm.value.entityType === "order") payload.orderId = paymentForm.value.entityId;
    else payload.saleId = paymentForm.value.entityId;
    await api.createPayment(token.value, payload);
    showPayment.value = false;
    await load();
    if (selected.value) await loadCard();
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
  new: "Нове", in_work: "В роботі", ordered: "Замовлено", expected: "Очікується",
  arrived: "Надійшло", issued: "Видано", closed: "Закрито", cancelled: "Скасовано",
  draft: "Створено", sent: "Відправлено", confirmed: "Підтверджено",
  in_transit: "В дорозі", received: "Отримано",
  processing: "В обробці", paid: "Оплачено", completed: "Завершено",
};

function fmt(n: number) { return n.toLocaleString("uk-UA", { maximumFractionDigits: 2 }); }
function fmtDate(s: string) { return new Date(s).toLocaleDateString("uk"); }

function customerDebt(c: Customer): number {
  const cp: Counterparty = { key: `c-${c.id}`, id: c.id, name: c.name, roles: "customer", customerId: c.id };
  return cpDebtOwedByCustomer(cp);
}
function customerOrderCount(c: Customer): number { return orders.value.filter(o => o.customerName === c.name).length; }
function supplierPurchaseCount(s: Supplier): number { return purchases.value.filter(p => p.supplierId === s.id).length; }
function supplierTotal(s: Supplier): number { return purchases.value.filter(p => p.supplierId === s.id).reduce((a, p) => a + p.totalUah, 0); }

watch(() => props.initialSubTab, (v) => {
  if (v === "suppliers") activeSubTab.value = "suppliers";
  else if (v === "customers") activeSubTab.value = "customers";
});

onMounted(load);
</script>

<template>
  <div class="page-content">
    <!-- ── Header ── -->
    <div class="page-header" style="margin-bottom:1rem">
      <div style="display:flex;align-items:center;gap:1rem">
        <h2 style="margin:0">{{ activeSubTab === 'customers' ? 'Клієнти' : 'Постачальники' }}</h2>
        <div class="tab-row" style="margin:0">
          <button :class="['tab-button', activeSubTab==='customers'&&'tab-button--active']" @click="activeSubTab='customers'">
            👤 Клієнти <span style="font-size:0.75em;opacity:0.7">({{ customers.length }})</span>
          </button>
          <button :class="['tab-button', activeSubTab==='suppliers'&&'tab-button--active']" @click="activeSubTab='suppliers'">
            🏭 Постачальники <span style="font-size:0.75em;opacity:0.7">({{ suppliers.length }})</span>
          </button>
        </div>
      </div>
      <div>
        <button v-if="activeSubTab==='customers'" class="ghost-button" @click="showNewCustomer=true">+ Клієнт</button>
        <button v-else class="ghost-button" @click="showNewSupplier=true">+ Постачальник</button>
      </div>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <!-- ══════════════════════════════════════ -->
    <!-- CUSTOMERS TAB -->
    <!-- ══════════════════════════════════════ -->
    <div v-if="activeSubTab === 'customers' && !loading">
      <input v-model="searchCustomers" placeholder="🔍 Пошук за ім'ям, телефоном, email..." style="width:100%;margin-bottom:1rem;box-sizing:border-box">

      <div v-if="!filteredCustomers.length" class="subtle" style="padding:2rem;text-align:center">
        <div style="font-size:2rem;margin-bottom:0.5rem">👥</div>
        Клієнтів не знайдено
      </div>

      <table v-else style="width:100%;font-size:0.88rem">
        <thead>
          <tr>
            <th style="text-align:left">Ім'я / Назва</th>
            <th style="text-align:left">Телефон</th>
            <th style="text-align:left">Email</th>
            <th style="text-align:center">Замовлень</th>
            <th style="text-align:right">Борг, ₴</th>
            <th style="text-align:right">Додано</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in filteredCustomers" :key="c.id"
            style="cursor:pointer;transition:background 0.12s"
            @click="openCustomer(c)"
            @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(122,185,154,0.07)'"
            @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
            <td style="font-weight:600;padding:0.6rem 0.5rem">
              <span style="margin-right:0.4rem">👤</span>{{ c.name }}
            </td>
            <td style="padding:0.6rem 0.5rem;color:var(--text-subtle)">{{ c.phone || '—' }}</td>
            <td style="padding:0.6rem 0.5rem;color:var(--text-subtle)">{{ c.email || '—' }}</td>
            <td style="text-align:center;padding:0.6rem 0.5rem">
              <span class="chip">{{ customerOrderCount(c) }}</span>
            </td>
            <td style="text-align:right;padding:0.6rem 0.5rem;font-weight:600"
              :style="customerDebt(c) > 0 ? 'color:#e07070' : 'color:var(--text-subtle)'">
              {{ customerDebt(c) > 0 ? fmt(customerDebt(c)) : '—' }}
            </td>
            <td style="text-align:right;padding:0.6rem 0.5rem;color:var(--text-subtle);font-size:0.8rem">
              {{ fmtDate(c.createdAt) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- ══════════════════════════════════════ -->
    <!-- SUPPLIERS TAB -->
    <!-- ══════════════════════════════════════ -->
    <div v-if="activeSubTab === 'suppliers' && !loading">
      <input v-model="searchSuppliers" placeholder="🔍 Пошук за назвою, контактом, телефоном, email..." style="width:100%;margin-bottom:1rem;box-sizing:border-box">

      <div v-if="!filteredSuppliers.length" class="subtle" style="padding:2rem;text-align:center">
        <div style="font-size:2rem;margin-bottom:0.5rem">🏭</div>
        Постачальників не знайдено
      </div>

      <table v-else style="width:100%;font-size:0.88rem">
        <thead>
          <tr>
            <th style="text-align:left">Назва</th>
            <th style="text-align:left">Контактна особа</th>
            <th style="text-align:left">Телефон</th>
            <th style="text-align:left">Email</th>
            <th style="text-align:center">Надходжень</th>
            <th style="text-align:right">Сума закупівель, ₴</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in filteredSuppliers" :key="s.id"
            style="cursor:pointer;transition:background 0.12s"
            @click="openSupplier(s)"
            @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(90,180,220,0.07)'"
            @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
            <td style="font-weight:600;padding:0.6rem 0.5rem">
              <span style="margin-right:0.4rem">🏭</span>{{ s.name }}
            </td>
            <td style="padding:0.6rem 0.5rem;color:var(--text-subtle)">{{ s.contact || '—' }}</td>
            <td style="padding:0.6rem 0.5rem;color:var(--text-subtle)">{{ s.phone || '—' }}</td>
            <td style="padding:0.6rem 0.5rem;color:var(--text-subtle)">{{ s.email || '—' }}</td>
            <td style="text-align:center;padding:0.6rem 0.5rem">
              <span class="chip" style="background:rgba(90,180,220,0.12);color:#60c0e0">{{ supplierPurchaseCount(s) }}</span>
            </td>
            <td style="text-align:right;padding:0.6rem 0.5rem;font-weight:600;color:#60c0e0">
              {{ supplierTotal(s) > 0 ? fmt(supplierTotal(s)) : '—' }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- ══════════════════════════════════════════════════════════════════ -->
    <!-- MODAL: Detail card -->
    <!-- ══════════════════════════════════════════════════════════════════ -->
    <div v-if="modalOpen && selected" class="modal-backdrop" @click.self="closeModal">
      <div class="panel modal-box" style="max-width:560px;width:100%;max-height:85vh;overflow-y:auto">

        <!-- Header -->
        <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:1.25rem">
          <div>
            <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.3rem">
              <span style="font-size:1.5rem">{{ selected.roles === 'supplier' ? '🏭' : '👤' }}</span>
              <h3 style="margin:0;font-size:1.1rem">{{ selected.name }}</h3>
            </div>
            <span style="font-size:0.72rem;padding:0.15rem 0.55rem;border-radius:20px"
              :style="selected.roles==='supplier' ? 'background:rgba(90,180,220,0.13);color:#60c0e0' : 'background:rgba(72,187,120,0.13);color:#48bb78'">
              {{ selected.roles === 'supplier' ? 'Постачальник' : 'Клієнт' }}
            </span>
          </div>
          <button class="ghost-button" style="padding:0.3rem 0.6rem;font-size:1rem" @click="closeModal">✕</button>
        </div>

        <!-- Contacts -->
        <div style="font-size:0.7rem;font-weight:600;letter-spacing:0.06em;color:var(--text-subtle);margin-bottom:0.5rem">КОНТАКТИ</div>
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem 1.5rem;font-size:0.85rem;margin-bottom:1.25rem;padding:0.75rem;background:rgba(122,185,154,0.04);border-radius:8px">
          <div v-if="selected.contact">
            <div style="font-size:0.67rem;color:var(--text-subtle);margin-bottom:0.1rem">КОНТАКТНА ОСОБА</div>
            <div style="font-weight:500">{{ selected.contact }}</div>
          </div>
          <div>
            <div style="font-size:0.67rem;color:var(--text-subtle);margin-bottom:0.1rem">ТЕЛЕФОН</div>
            <div style="font-weight:500">{{ selected.phone || '—' }}</div>
          </div>
          <div>
            <div style="font-size:0.67rem;color:var(--text-subtle);margin-bottom:0.1rem">EMAIL</div>
            <div style="font-weight:500">{{ selected.email || '—' }}</div>
          </div>
        </div>

        <div v-if="modalLoading" class="subtle" style="text-align:center;padding:1rem">Завантаження...</div>

        <template v-if="!modalLoading">
          <!-- KPI row -->
          <div style="display:grid;gap:0.5rem;margin-bottom:1.25rem"
            :style="selected.roles==='customer' ? 'grid-template-columns:repeat(3,1fr)' : 'grid-template-columns:repeat(3,1fr)'">

            <template v-if="selected.customerId">
              <div style="border-radius:8px;padding:0.75rem;text-align:center;background:rgba(72,187,120,0.07)">
                <div style="font-size:1.2rem;font-weight:700;color:var(--accent)">{{ cpOrders(selected).length }}</div>
                <div style="font-size:0.65rem;color:var(--text-subtle)">Замовлень</div>
              </div>
              <div style="border-radius:8px;padding:0.75rem;text-align:center;background:rgba(72,187,120,0.07)">
                <div style="font-size:1rem;font-weight:700;color:var(--accent)">{{ fmt(cpTotalSalesUah(selected)) }}</div>
                <div style="font-size:0.65rem;color:var(--text-subtle)">Продажів ₴</div>
              </div>
              <div style="border-radius:8px;padding:0.75rem;text-align:center"
                :style="cpDebtOwedByCustomer(selected)>0 ? 'background:rgba(224,112,112,0.08)' : 'background:rgba(72,187,120,0.07)'">
                <div style="font-size:1rem;font-weight:700"
                  :style="cpDebtOwedByCustomer(selected)>0 ? 'color:#e07070' : 'color:var(--accent)'">
                  {{ fmt(cpDebtOwedByCustomer(selected)) }}
                </div>
                <div style="font-size:0.65rem;color:var(--text-subtle)">Борг ₴</div>
              </div>
            </template>

            <template v-if="selected.supplierId">
              <div style="border-radius:8px;padding:0.75rem;text-align:center;background:rgba(90,180,220,0.07)">
                <div style="font-size:1.2rem;font-weight:700;color:#60c0e0">{{ cpPurchases(selected).length }}</div>
                <div style="font-size:0.65rem;color:var(--text-subtle)">Закупівель</div>
              </div>
              <div style="border-radius:8px;padding:0.75rem;text-align:center;background:rgba(90,180,220,0.07)">
                <div style="font-size:1rem;font-weight:700;color:#60c0e0">{{ fmt(cpTotalPurchasesUah(selected)) }}</div>
                <div style="font-size:0.65rem;color:var(--text-subtle)">Сума ₴</div>
              </div>
              <div style="border-radius:8px;padding:0.75rem;text-align:center;background:rgba(90,180,220,0.07)">
                <div style="font-size:1.2rem;font-weight:700;color:#60c0e0">{{ cpSupplierOrders(selected).length }}</div>
                <div style="font-size:0.65rem;color:var(--text-subtle)">Замовлень</div>
              </div>
            </template>
          </div>

          <!-- Debts (customers only) -->
          <div v-if="selected.customerId && cpOrderDebts(selected).length" style="margin-bottom:1.25rem">
            <div style="font-size:0.7rem;font-weight:600;letter-spacing:0.06em;color:var(--text-subtle);margin-bottom:0.5rem">АКТИВНІ БОРГИ</div>
            <div v-for="{debt, order} in cpOrderDebts(selected)" :key="debt.entityId"
              style="border-radius:8px;padding:0.65rem 0.85rem;background:rgba(224,112,112,0.07);border:1px solid rgba(224,112,112,0.18);margin-bottom:0.4rem">
              <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:0.25rem">
                <div>
                  <div style="font-size:0.85rem;font-weight:600">📋 Замовлення #{{ order.id }}</div>
                  <div style="font-size:0.72rem;color:var(--text-subtle)">{{ statusLabel[order.status] }} · {{ fmtDate(order.createdAt) }}</div>
                </div>
                <div style="text-align:right">
                  <div style="color:#e07070;font-weight:700">{{ fmt(debt.debtUah) }} ₴</div>
                </div>
              </div>
              <div v-if="cardDebtHistory.get(debt.entityId)?.length" style="padding:0.3rem 0.5rem;background:rgba(0,0,0,0.12);border-radius:5px;margin-bottom:0.3rem">
                <div v-for="h in cardDebtHistory.get(debt.entityId)" :key="h.paymentId"
                  style="display:flex;justify-content:space-between;font-size:0.75rem">
                  <span style="color:#6dd4a0">✓ +{{ fmt(h.amount) }} {{ h.currency }} · {{ h.method }}</span>
                  <span style="color:var(--text-subtle)">{{ fmtDate(h.createdAt) }}</span>
                </div>
              </div>
              <button class="ghost-button" style="font-size:0.76rem;padding:0.2rem 0.6rem;border-color:rgba(224,112,112,0.4);color:#e07070"
                @click="openPaymentModal('order', order.id, debt.debt, order.currency, `Замовлення #${order.id}`)">
                💳 Оплатити
              </button>
            </div>
          </div>

          <!-- Orders (customer) -->
          <div v-if="selected.customerId && cpOrders(selected).length" style="margin-bottom:1.25rem">
            <div style="font-size:0.7rem;font-weight:600;letter-spacing:0.06em;color:var(--text-subtle);margin-bottom:0.5rem">
              ЗАМОВЛЕННЯ ({{ cpOrders(selected).length }})
            </div>
            <div v-for="o in cpOrders(selected).slice(0,10)" :key="o.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(122,185,154,0.1);font-size:0.84rem">
              <div>
                <span style="font-weight:500">#{{ o.id }}</span>
                <span style="margin-left:0.4rem;font-size:0.72rem;padding:0.1rem 0.4rem;border-radius:10px;background:rgba(122,185,154,0.1);color:var(--text-subtle)">
                  {{ statusLabel[o.status] }}
                </span>
              </div>
              <div style="display:flex;align-items:center;gap:0.5rem">
                <span style="font-weight:500">{{ fmt(o.total) }} {{ o.currency }}</span>
                <span style="color:var(--text-subtle);font-size:0.72rem">{{ fmtDate(o.createdAt) }}</span>
              </div>
            </div>
            <div v-if="cpOrders(selected).length > 10" style="font-size:0.75rem;color:var(--text-subtle);margin-top:0.3rem">
              + ще {{ cpOrders(selected).length - 10 }} замовлень
            </div>
          </div>

          <!-- Payments history (customer) -->
          <div v-if="selected.customerId && cardPayments.length" style="margin-bottom:1.25rem">
            <div style="font-size:0.7rem;font-weight:600;letter-spacing:0.06em;color:var(--text-subtle);margin-bottom:0.5rem">
              ОПЛАТИ ({{ cardPayments.length }})
            </div>
            <div v-for="p in cardPayments.slice(0,8)" :key="p.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(122,185,154,0.1);font-size:0.84rem">
              <div>
                <span>Оплата #{{ p.id }}</span>
                <span style="color:var(--text-subtle);margin-left:0.35rem;font-size:0.72rem">{{ p.method }}</span>
                <span v-if="p.orderId" style="color:var(--text-subtle);margin-left:0.35rem;font-size:0.72rem">· Зам. #{{ p.orderId }}</span>
              </div>
              <span style="color:#6dd4a0;font-weight:600">+{{ fmt(p.amount) }} {{ p.currency }}</span>
            </div>
          </div>

          <!-- Purchases (supplier) -->
          <div v-if="selected.supplierId && cpPurchases(selected).length" style="margin-bottom:1.25rem">
            <div style="font-size:0.7rem;font-weight:600;letter-spacing:0.06em;color:var(--text-subtle);margin-bottom:0.5rem">
              НАДХОДЖЕННЯ ({{ cpPurchases(selected).length }})
            </div>
            <div v-for="p in cpPurchases(selected).slice(0,10)" :key="p.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(90,180,220,0.1);font-size:0.84rem">
              <span style="font-weight:500">Надходження #{{ p.id }}</span>
              <div style="display:flex;align-items:center;gap:0.5rem">
                <span style="color:#60c0e0;font-weight:500">{{ fmt(p.total) }} {{ p.currency }}</span>
                <span style="color:var(--text-subtle);font-size:0.72rem">{{ fmtDate(p.createdAt) }}</span>
              </div>
            </div>
          </div>

          <!-- Supplier orders -->
          <div v-if="selected.supplierId && cpSupplierOrders(selected).length" style="margin-bottom:1.25rem">
            <div style="font-size:0.7rem;font-weight:600;letter-spacing:0.06em;color:var(--text-subtle);margin-bottom:0.5rem">
              ЗАМОВЛЕННЯ ПОСТАЧАЛЬНИКУ ({{ cpSupplierOrders(selected).length }})
            </div>
            <div v-for="o in cpSupplierOrders(selected).slice(0,8)" :key="o.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(90,180,220,0.1);font-size:0.84rem">
              <div>
                <span style="font-weight:500">#{{ o.id }}</span>
                <span style="margin-left:0.4rem;font-size:0.72rem;padding:0.1rem 0.4rem;border-radius:10px;background:rgba(90,180,220,0.1);color:#60c0e0">
                  {{ statusLabel[o.status] || o.status }}
                </span>
              </div>
              <div style="display:flex;align-items:center;gap:0.5rem">
                <span style="color:#60c0e0;font-weight:500">{{ fmt(o.total) }} {{ o.currency }}</span>
                <span style="color:var(--text-subtle);font-size:0.72rem">{{ fmtDate(o.createdAt) }}</span>
              </div>
            </div>
          </div>

          <!-- Comment -->
          <div v-if="selected.comment"
            style="font-size:0.83rem;color:var(--text-subtle);font-style:italic;padding:0.6rem;background:rgba(122,185,154,0.05);border-radius:6px">
            💬 {{ selected.comment }}
          </div>
        </template>

      </div>
    </div>

    <!-- ═══ MODAL: Новий клієнт ═══ -->
    <div v-if="showNewCustomer" class="modal-backdrop" @click.self="showNewCustomer=false">
      <div class="panel modal-box">
        <h3>Новий клієнт</h3>
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
          <label>Метод
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
