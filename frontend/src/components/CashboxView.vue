<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import type { UserSession, Cashbox, CashOperation, CashShift, ExchangeRate, DebtSummary, CurrencyCode, PaymentMethod, SupplierOrder, Supplier, ServiceOrder, CustomerOrder } from "../types";

const props = defineProps<{ session: UserSession; initialSubTab?: string }>();
const emit = defineEmits<{ (e: "session-expired"): void; (e: "navigate", payload: { type: string; id: number } | string): void }>();

const token = computed(() => props.session.token);

const cashboxes = ref<Cashbox[]>([]);
const operations = ref<CashOperation[]>([]);
const shifts = ref<CashShift[]>([]);
const rates = ref<ExchangeRate[]>([]);
const debts = ref<DebtSummary[]>([]);
const loading = ref(false);
const error = ref("");
const saving = ref(false);
const subTab = ref<"cashboxes" | "operations" | "shifts" | "debts" | "rates">((props.initialSubTab as any) ?? "cashboxes");
const selectedCashboxId = ref<number | undefined>(undefined);

const showNewCashbox = ref(false);
const cbForm = ref({ name: "", type: "cash" as PaymentMethod, currency: "UAH" as CurrencyCode });

const showNewOperation = ref(false);
const opForm = ref({ cashboxId: 0, type: "incoming" as "incoming" | "outgoing", amount: 0, method: "cash" as PaymentMethod, description: "", linkedType: "" as "" | "supplier_order" | "service_order", linkedId: 0 });

const serviceOrders = ref<ServiceOrder[]>([]);
const customerOrders = ref<CustomerOrder[]>([]);

const showOpenShift = ref(false);
const shiftForm = ref({ cashboxId: 0, note: "" });

const showRateEdit = ref(false);
const rateForm = ref({ currency: "USD" as CurrencyCode, rateToUah: 0 });

// ── Supplier order payment (from debts tab) ───────────────────
const supplierOrders = ref<SupplierOrder[]>([]);
const suppliers = ref<Supplier[]>([]);

const showPaySupplierDebt = ref(false);
const payDebt = ref<DebtSummary | null>(null);
const payForm = ref({ cashboxId: 0, amount: 0, method: "bank" as PaymentMethod, note: "" });
const paying = ref(false);

async function exportVAT(format: "csv" | "xlsx" = "csv") {
  try {
    const blob = format === "xlsx"
      ? await api.exportVATXlsx(token.value)
      : await api.exportVATCsv(token.value);
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url; a.download = `vat-export.${format}`; a.click();
    URL.revokeObjectURL(url);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  }
}

async function load() {
  loading.value = true; error.value = "";
  try {
    [cashboxes.value, rates.value, debts.value, supplierOrders.value, suppliers.value, serviceOrders.value, customerOrders.value] = await Promise.all([
      api.cashboxes(token.value),
      api.exchangeRates(token.value),
      api.debts(token.value),
      api.supplierOrders(token.value),
      api.suppliers(token.value),
      api.serviceOrders(token.value),
      api.orders(token.value),
    ]);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

async function loadOperations() {
  try {
    operations.value = await api.cashOperations(token.value, selectedCashboxId.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

async function loadShifts() {
  try {
    shifts.value = await api.cashShifts(token.value, { cashboxId: selectedCashboxId.value });
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

async function addCashbox() {
  saving.value = true; error.value = "";
  try {
    await api.createCashbox(token.value, cbForm.value);
    showNewCashbox.value = false;
    cbForm.value = { name: "", type: "cash", currency: "UAH" };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function addOperation() {
  saving.value = true; error.value = "";
  try {
    if (opForm.value.linkedType && opForm.value.linkedId) {
      await api.createPayment(token.value, {
        ...(opForm.value.linkedType === "supplier_order" ? { supplierOrderId: opForm.value.linkedId } : { serviceOrderId: opForm.value.linkedId }),
        cashboxId: opForm.value.cashboxId,
        amount: opForm.value.amount,
        currency: "UAH",
        method: opForm.value.method,
        note: opForm.value.description || (opForm.value.linkedType === "supplier_order" ? `Оплата замовлення постачальника #${opForm.value.linkedId}` : `Оплата ремонту #${opForm.value.linkedId}`),
      });
    } else {
      await api.createCashOperation(token.value, {
        cashboxId: opForm.value.cashboxId,
        type: opForm.value.type,
        amount: opForm.value.amount,
        method: opForm.value.method,
        description: opForm.value.description,
      });
    }
    showNewOperation.value = false;
    opForm.value = { cashboxId: 0, type: "incoming", amount: 0, method: "cash", description: "", linkedType: "", linkedId: 0 };
    await loadOperations();
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function openShift() {
  saving.value = true; error.value = "";
  try {
    await api.openCashShift(token.value, shiftForm.value);
    showOpenShift.value = false;
    shiftForm.value = { cashboxId: 0, note: "" };
    await loadShifts();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function closeShift(shiftId: number) {
  try {
    await api.closeCashShift(token.value, shiftId, { note: "" });
    await loadShifts();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
    else error.value = e.message;
  }
}

async function saveRate() {
  saving.value = true; error.value = "";
  try {
    await api.upsertExchangeRate(token.value, rateForm.value);
    showRateEdit.value = false;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

function editRate(r: ExchangeRate) {
  rateForm.value = { currency: r.currency, rateToUah: r.rateToUah };
  showRateEdit.value = true;
}

function supplierNameById(id: number) {
  return suppliers.value.find(s => s.id === id)?.name ?? `#${id}`;
}
function supplierOrderById(id: number) {
  return supplierOrders.value.find(o => o.id === id);
}

function openPayDebt(d: DebtSummary) {
  payDebt.value = d;
  payForm.value = { cashboxId: cashboxes.value[0]?.id ?? 0, amount: d.debtUah, method: "bank", note: "" };
  showPaySupplierDebt.value = true;
}

async function submitPayDebt() {
  if (!payDebt.value) return;
  paying.value = true; error.value = "";
  try {
    await api.createPayment(token.value, {
      supplierOrderId: payDebt.value.entityId,
      cashboxId: payForm.value.cashboxId,
      amount: payForm.value.amount,
      currency: "UAH",
      method: payForm.value.method,
      note: payForm.value.note || `Оплата замовлення #${payDebt.value.entityId}`,
    });
    showPaySupplierDebt.value = false;
    payDebt.value = null;
    await load();
    await loadOperations();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { paying.value = false; }
}

const totalByCurrency = computed(() => {
  const map: Record<string, number> = {};
  for (const cb of cashboxes.value) map[cb.currency] = (map[cb.currency] || 0) + cb.balance;
  return map;
});

const overdueDebts = computed(() => debts.value.filter(d => d.isOverdue));

const methodLabel: Record<string, string> = { cash: "Готівка", card: "Картка", bank: "Банк", virtual: "Віртуальна" };
const typeLabel: Record<string, string> = { cash: "Готівкова", card: "Карткова", bank: "Банківська", virtual: "Віртуальна" };

onMounted(async () => {
  await load();
  await loadOperations();
  await loadShifts();
});
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Каса та фінанси</h2>
      <div style="display:flex;gap:0.5rem">
        <button class="ghost-button" @click="exportVAT('csv')">ПДВ CSV</button>
        <button class="ghost-button" @click="exportVAT('xlsx')">ПДВ XLSX</button>
      </div>
    </div>

    <!-- Summary KPIs -->
    <div class="kpi-grid" style="margin-bottom:1.2rem">
      <div v-for="(amount, cur) in totalByCurrency" :key="cur" class="kpi-card">
        <p class="kpi-card__title">Баланс {{ cur }}</p>
        <p class="kpi-card__value">{{ amount.toFixed(2) }}</p>
      </div>
      <div class="kpi-card">
        <p class="kpi-card__title">Прострочені борги</p>
        <p class="kpi-card__value" :style="overdueDebts.length?'color:#ff9ca0':''">{{ overdueDebts.length }}</p>
      </div>
    </div>

    <div class="tab-row" style="margin-bottom:1rem">
      <button :class="['tab-button', subTab==='cashboxes'&&'tab-button--active']" @click="subTab='cashboxes'">Каси</button>
      <button :class="['tab-button', subTab==='operations'&&'tab-button--active']" @click="subTab='operations';loadOperations()">Операції</button>
      <button :class="['tab-button', subTab==='shifts'&&'tab-button--active']" @click="subTab='shifts';loadShifts()">Зміни</button>
      <button :class="['tab-button', subTab==='debts'&&'tab-button--active']" @click="subTab='debts'">Борги</button>
      <button :class="['tab-button', subTab==='rates'&&'tab-button--active']" @click="subTab='rates'">Курси валют</button>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <!-- CASHBOXES -->
    <div v-if="subTab==='cashboxes'">
      <!-- Cashbox list -->
      <div style="display:flex;gap:0.5rem;align-items:center;margin-bottom:1rem">
        <button class="ghost-button" @click="showNewCashbox=true">+ Каса</button>
      </div>
      <table style="margin-bottom:1.5rem">
        <thead><tr><th>Назва</th><th>Тип</th><th>Валюта</th><th>Баланс</th></tr></thead>
        <tbody>
          <tr v-for="cb in cashboxes" :key="cb.id">
            <td>{{ cb.name }}</td>
            <td>{{ typeLabel[cb.type] || cb.type }}</td>
            <td>{{ cb.currency }}</td>
            <td style="font-weight:700">{{ cb.balance.toFixed(2) }}</td>
          </tr>
          <tr v-if="!cashboxes.length"><td colspan="4" class="subtle">Немає кас</td></tr>
        </tbody>
      </table>
    </div>

    <!-- OPERATIONS -->
    <div v-if="subTab==='operations'">
      <div style="display:flex;gap:0.5rem;margin-bottom:1rem">
        <select v-model="selectedCashboxId" style="min-width:160px" @change="loadOperations">
          <option :value="undefined">Всі каси</option>
          <option v-for="cb in cashboxes" :key="cb.id" :value="cb.id">{{ cb.name }}</option>
        </select>
        <button class="ghost-button" @click="showNewOperation=true">+ Операція</button>
      </div>
      <table>
        <thead><tr><th>Тип</th><th>Сума</th><th>Валюта</th><th>Метод</th><th>Опис</th><th>Дата</th></tr></thead>
        <tbody>
          <tr v-for="op in operations" :key="op.id">
            <td :style="op.type==='incoming'?'color:#6dd4a0':'color:#ff9ca0'">
              {{ op.type === 'incoming' ? '↓ Надходження' : '↑ Видача' }}
            </td>
            <td style="font-weight:600">{{ op.amount }}</td>
            <td>{{ op.currency }}</td>
            <td>{{ methodLabel[op.method] || op.method }}</td>
            <td>{{ op.description }}</td>
            <td>{{ new Date(op.createdAt).toLocaleString('uk') }}</td>
          </tr>
          <tr v-if="!operations.length"><td colspan="6" class="subtle">Немає операцій</td></tr>
        </tbody>
      </table>
    </div>

    <!-- SHIFTS -->
    <div v-if="subTab==='shifts'">
      <div style="display:flex;gap:0.5rem;margin-bottom:1rem">
        <select v-model="selectedCashboxId" style="min-width:160px" @change="loadShifts">
          <option :value="undefined">Всі каси</option>
          <option v-for="cb in cashboxes" :key="cb.id" :value="cb.id">{{ cb.name }}</option>
        </select>
        <button class="ghost-button" @click="showOpenShift=true">Відкрити зміну</button>
      </div>
      <table>
        <thead><tr><th>ID</th><th>Статус</th><th>Відкрив</th><th>Відкрито</th><th>Закрив</th><th>Закрито</th><th>Початк. баланс</th><th>Кінц. баланс</th><th>Дії</th></tr></thead>
        <tbody>
          <tr v-for="s in shifts" :key="s.id">
            <td>{{ s.id }}</td>
            <td :style="s.status==='open'?'color:#6dd4a0':'color:#7ab99a'">{{ s.status === 'open' ? 'Відкрита' : 'Закрита' }}</td>
            <td>{{ s.openedBy }}</td>
            <td>{{ new Date(s.openedAt).toLocaleString('uk') }}</td>
            <td>{{ s.closedBy || '—' }}</td>
            <td>{{ s.closedAt ? new Date(s.closedAt).toLocaleString('uk') : '—' }}</td>
            <td>{{ s.openingBalance }}</td>
            <td>{{ s.closingBalance ?? '—' }}</td>
            <td>
              <button v-if="s.status==='open'" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem" @click="closeShift(s.id)">Закрити</button>
            </td>
          </tr>
          <tr v-if="!shifts.length"><td colspan="9" class="subtle">Немає змін</td></tr>
        </tbody>
      </table>
    </div>

    <!-- DEBTS -->
    <div v-if="subTab==='debts'">
      <div v-if="debts.some(d => d.entityType === 'supplier_order')" style="margin-bottom:1.5rem">
        <div style="font-size:0.8rem;font-weight:600;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.6rem">
          Борги постачальникам
        </div>
        <table>
          <thead>
            <tr>
              <th>Постачальник</th>
              <th>Замовлення</th>
              <th style="text-align:right">Сума</th>
              <th style="text-align:right">Оплачено</th>
              <th style="text-align:right">Залишок</th>
              <th>Статус</th>
              <th>Дії</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in debts.filter(d => d.entityType === 'supplier_order')"
              style="cursor:pointer"
              @click="emit('navigate', { type: 'supplier_order', id: d.entityId })" :key="`so-${d.entityId}`" :class="d.isOverdue ? 'row-overdue' : ''">
              <td style="font-weight:500">{{ supplierOrderById(d.entityId)?.supplierId ? supplierNameById(supplierOrderById(d.entityId)!.supplierId) : '—' }}</td>
              <td>
                <span style="font-size:0.8rem;opacity:0.7">#{{ d.entityId }}</span>
                <span v-if="supplierOrderById(d.entityId)" style="margin-left:0.4rem;font-size:0.75rem;padding:0.1rem 0.4rem;border-radius:8px;background:rgba(255,255,255,0.07)">
                  {{ supplierOrderById(d.entityId)!.status }}
                </span>
              </td>
              <td style="text-align:right">{{ d.total.toFixed(2) }} {{ d.currency }}</td>
              <td style="text-align:right;color:#6dd4a0">{{ d.paid.toFixed(2) }}</td>
              <td style="text-align:right" :style="d.debt>0?'color:#ff9ca0;font-weight:700':'color:#6dd4a0'">{{ d.debt.toFixed(2) }} {{ d.currency }}</td>
              <td>
                <span v-if="d.isOverdue" style="color:#ff9ca0;font-size:0.8rem">⚠ {{ d.overdueDays }} дн</span>
                <span v-else style="color:#6dd4a0;font-size:0.8rem">В нормі</span>
              </td>
              <td>
                <button v-if="d.debt > 0" class="ghost-button" style="padding:0.3rem 0.7rem;font-size:0.82rem" @click="openPayDebt(d)">💳 Сплатити</button>
                <span v-else style="color:#6dd4a0;font-size:0.8rem">✓ Оплачено</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="debts.some(d => d.entityType !== 'supplier_order')">
        <div style="font-size:0.8rem;font-weight:600;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.6rem">
          Інші борги
        </div>
        <table>
          <thead><tr><th>Тип</th><th>ID</th><th>Сума</th><th>Оплачено</th><th>Борг</th><th>Термін</th><th>Прострочено</th></tr></thead>
          <tbody>
            <tr v-for="d in debts.filter(d => d.entityType !== 'supplier_order')" :key="`${d.entityType}-${d.entityId}`"
              :class="d.isOverdue?'row-overdue':''"
              style="cursor:pointer"
              @click="emit('navigate', { type: d.entityType, id: d.entityId })">
              <td style="font-weight:500">{{ { sale: 'Продаж', order: 'Замовлення', customer_order: 'Замовлення', service_order: 'Ремонт' }[d.entityType] ?? d.entityType }}</td>
              <td>#{{ d.entityId }}</td>
              <td>{{ d.total }} {{ d.currency }}</td>
              <td style="color:#6dd4a0">{{ d.paid }}</td>
              <td :style="d.debt>0?'color:#ff9ca0;font-weight:700':''">{{ d.debt }}</td>
              <td>{{ d.dueDate ? new Date(d.dueDate).toLocaleDateString('uk') : '—' }}</td>
              <td>
                <span v-if="d.isOverdue" style="color:#ff9ca0">{{ d.overdueDays }} дн</span>
                <span v-else style="color:#6dd4a0">Ні</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <p v-if="!debts.length" class="subtle" style="margin-top:1rem">Немає боргів</p>
    </div>

    <!-- RATES -->
    <div v-if="subTab==='rates'">
      <table>
        <thead><tr><th>Валюта</th><th>Курс до UAH</th><th>Оновлено</th><th>Дії</th></tr></thead>
        <tbody>
          <tr v-for="r in rates" :key="r.currency">
            <td>{{ r.currency }}</td>
            <td style="font-weight:700">{{ r.rateToUah }}</td>
            <td>{{ new Date(r.updatedAt).toLocaleString('uk') }}</td>
            <td><button class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem" @click="editRate(r)">✏ Змінити</button></td>
          </tr>
          <tr v-if="!rates.length"><td colspan="4" class="subtle">Немає курсів</td></tr>
        </tbody>
      </table>
      <button class="ghost-button" style="margin-top:0.8rem" @click="showRateEdit=true;rateForm={currency:'USD',rateToUah:0}">+ Додати курс</button>
    </div>

    <!-- MODAL: New Cashbox -->
    <div v-if="showNewCashbox" class="modal-backdrop" @click.self="showNewCashbox=false">
      <div class="panel modal-box">
        <h3>Нова каса</h3>
        <div class="grid">
          <label>Назва <input v-model="cbForm.name" placeholder="Назва каси"></label>
          <label>Тип
            <select v-model="cbForm.type">
              <option value="cash">Готівкова</option>
              <option value="card">Карткова</option>
              <option value="bank">Банківська</option>
              <option value="virtual">Віртуальна</option>
            </select>
          </label>
          <label>Валюта
            <select v-model="cbForm.currency"><option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option></select>
          </label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="addCashbox" :disabled="saving||!cbForm.name">{{ saving?'..':'Зберегти' }}</button>
          <button class="ghost-button" @click="showNewCashbox=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: New Operation -->
    <div v-if="showNewOperation" class="modal-backdrop" @click.self="showNewOperation=false">
      <div class="panel modal-box">
        <h3>Касова операція</h3>
        <div class="grid">
          <label>Каса
            <select v-model.number="opForm.cashboxId">
              <option value="0" disabled>Оберіть касу</option>
              <option v-for="cb in cashboxes" :key="cb.id" :value="cb.id">{{ cb.name }}</option>
            </select>
          </label>
          <label>Тип
            <select v-model="opForm.type" :disabled="!!opForm.linkedType">
              <option value="incoming">Надходження (ПКО)</option>
              <option value="outgoing">Видача (ВКО)</option>
            </select>
          </label>
          <label>Сума <input type="number" min="0" v-model.number="opForm.amount"></label>
          <label>Метод
            <select v-model="opForm.method">
              <option value="cash">Готівка</option>
              <option value="card">Картка</option>
              <option value="bank">Банк</option>
            </select>
          </label>
          <label>Опис <input v-model="opForm.description"></label>
          <label style="grid-column:1/-1">
            <span style="font-size:0.82rem;color:var(--text-muted);font-weight:600;text-transform:uppercase;letter-spacing:0.04em">Прив'язати до замовлення (необов'язково)</span>
            <select v-model="opForm.linkedType" style="margin-top:0.3rem" @change="opForm.linkedId=0">
              <option value="">— Без прив'язки —</option>
              <option value="supplier_order">Замовлення постачальника</option>
              <option value="service_order">Замовлення на ремонт</option>
            </select>
          </label>
          <label v-if="opForm.linkedType === 'supplier_order'" style="grid-column:1/-1">
            Замовлення постачальника
            <select v-model.number="opForm.linkedId">
              <option value="0" disabled>Оберіть замовлення</option>
              <option v-for="so in supplierOrders" :key="so.id" :value="so.id">
                #{{ so.id }} — {{ supplierNameById(so.supplierId) }} ({{ so.status }}, {{ so.total.toFixed(2) }} {{ so.currency }})
              </option>
            </select>
          </label>
          <label v-if="opForm.linkedType === 'service_order'" style="grid-column:1/-1">
            Замовлення на ремонт
            <select v-model.number="opForm.linkedId">
              <option value="0" disabled>Оберіть замовлення</option>
              <option v-for="svo in serviceOrders" :key="svo.id" :value="svo.id">
                #{{ svo.id }} — {{ svo.customerName }} ({{ svo.status }}, {{ svo.total.toFixed(2) }} {{ svo.currency }})
              </option>
            </select>
          </label>
          <div v-if="opForm.linkedType && opForm.linkedId" style="grid-column:1/-1;padding:0.5rem 0.75rem;background:rgba(109,212,160,0.08);border:1px solid rgba(109,212,160,0.2);border-radius:6px;font-size:0.82rem;color:var(--text-muted)">
            ✓ Операція буде зарахована як оплата <strong>{{ opForm.linkedType === 'supplier_order' ? 'замовлення постачальника' : 'ремонту' }} #{{ opForm.linkedId }}</strong>
          </div>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="addOperation" :disabled="saving||!opForm.cashboxId||!opForm.amount||(!!opForm.linkedType&&!opForm.linkedId)">{{ saving?'..':'Провести' }}</button>
          <button class="ghost-button" @click="showNewOperation=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Open Shift -->
    <div v-if="showOpenShift" class="modal-backdrop" @click.self="showOpenShift=false">
      <div class="panel modal-box">
        <h3>Відкрити касову зміну</h3>
        <div class="grid">
          <label>Каса
            <select v-model.number="shiftForm.cashboxId">
              <option value="0" disabled>Оберіть касу</option>
              <option v-for="cb in cashboxes" :key="cb.id" :value="cb.id">{{ cb.name }}</option>
            </select>
          </label>
          <label>Примітка <input v-model="shiftForm.note"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="openShift" :disabled="saving||!shiftForm.cashboxId">{{ saving?'..':'Відкрити' }}</button>
          <button class="ghost-button" @click="showOpenShift=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Edit Rate -->
    <div v-if="showRateEdit" class="modal-backdrop" @click.self="showRateEdit=false">
      <div class="panel modal-box">
        <h3>Курс валюти</h3>
        <div class="grid">
          <label>Валюта
            <select v-model="rateForm.currency"><option>USD</option><option>EUR</option></select>
          </label>
          <label>Курс до UAH <input type="number" min="0" step="0.01" v-model.number="rateForm.rateToUah"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="saveRate" :disabled="saving||!rateForm.rateToUah">{{ saving?'..':'Зберегти' }}</button>
          <button class="ghost-button" @click="showRateEdit=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Pay Supplier Debt -->
    <div v-if="showPaySupplierDebt && payDebt" class="modal-backdrop" @click.self="showPaySupplierDebt=false">
      <div class="panel modal-box">
        <h3>Оплата постачальнику</h3>
        <div style="margin-bottom:1rem;padding:0.8rem;background:rgba(255,255,255,0.04);border-radius:8px">
          <div style="font-size:0.75rem;color:var(--text-subtle);margin-bottom:0.3rem">Замовлення #{{ payDebt.entityId }}</div>
          <div style="display:flex;justify-content:space-between;font-size:0.9rem">
            <span>Загальна сума: <strong>{{ payDebt.total.toFixed(2) }} {{ payDebt.currency }}</strong></span>
            <span>Оплачено: <span style="color:#6dd4a0">{{ payDebt.paid.toFixed(2) }}</span></span>
          </div>
          <div style="margin-top:0.4rem;font-size:1rem;font-weight:700" :style="payDebt.debt>0?'color:#ff9ca0':''">
            Залишок боргу: {{ payDebt.debt.toFixed(2) }} {{ payDebt.currency }}
          </div>
        </div>
        <div class="grid">
          <label>Каса
            <select v-model.number="payForm.cashboxId">
              <option value="0" disabled>Оберіть касу</option>
              <option v-for="cb in cashboxes" :key="cb.id" :value="cb.id">{{ cb.name }} ({{ cb.currency }})</option>
            </select>
          </label>
          <label>Сума оплати (UAH)
            <input type="number" min="0" :max="payDebt.debtUah" step="0.01" v-model.number="payForm.amount">
            <span style="font-size:0.75rem;color:var(--text-subtle)">Макс: {{ payDebt.debtUah.toFixed(2) }} грн</span>
          </label>
          <label>Метод оплати
            <select v-model="payForm.method">
              <option value="cash">Готівка</option>
              <option value="card">Картка</option>
              <option value="bank">Банківський переказ</option>
            </select>
          </label>
          <label>Примітка
            <input v-model="payForm.note" :placeholder="`Оплата замовлення #${payDebt.entityId}`">
          </label>
        </div>
        <div v-if="payForm.amount > 0" style="margin-top:0.8rem;padding:0.6rem;background:rgba(109,212,160,0.08);border-radius:6px;border:1px solid rgba(109,212,160,0.2);font-size:0.85rem">
          Після оплати борг зменшиться з
          <strong style="color:#ff9ca0">{{ payDebt.debtUah.toFixed(2) }} грн</strong> до
          <strong style="color:#6dd4a0">{{ Math.max(0, payDebt.debtUah - payForm.amount).toFixed(2) }} грн</strong>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="submitPayDebt" :disabled="paying||!payForm.cashboxId||!payForm.amount||payForm.amount<=0">
            {{ paying ? 'Проведення...' : '✓ Провести оплату' }}
          </button>
          <button class="ghost-button" @click="showPaySupplierDebt=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

  </div>
</template>
