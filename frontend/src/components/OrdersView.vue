<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import type {
  UserSession,
  CustomerOrder,
  CustomerOrderStatus,
  SupplierOrder,
  SupplierOrderReceiveLine,
  ServiceOrder,
  Customer,
  Supplier,
  Product,
  CurrencyCode,
  Cashbox,
  PaymentMethod,
} from "../types";

const props = defineProps<{
  session: UserSession;
  initialSubTab?: string;
}>();

const emit = defineEmits<{
  (e: "session-expired"): void;
}>();

type SubTab = "customer" | "supplier" | "repair";

const activeSubTab = ref<SubTab>((props.initialSubTab as SubTab) ?? "customer");

const isLoading = ref(false);
const errorText = ref("");

// ── Customer Orders ──────────────────────────────────────────
const customerOrders = ref<CustomerOrder[]>([]);
const customerFilter = ref("");

const filteredCustomerOrders = computed(() => {
  const q = customerFilter.value.trim().toLowerCase();
  if (!q) return customerOrders.value;
  return customerOrders.value.filter((o) =>
    `#${o.id} ${o.customerName} ${o.status} ${o.currency}`.toLowerCase().includes(q)
  );
});

const customerStatusLabel: Record<CustomerOrderStatus, string> = {
  new: "Новий",
  in_work: "В роботі",
  ordered: "Замовлено",
  expected: "Очікується",
  arrived: "Надійшло",
  issued: "Видано",
  closed: "Закрито",
  cancelled: "Скасовано",
};

const customerStatusClass: Record<CustomerOrderStatus, string> = {
  new: "badge--info",
  in_work: "badge--warn",
  ordered: "badge--warn",
  expected: "badge--warn",
  arrived: "badge--ok",
  issued: "badge--ok",
  closed: "badge--neutral",
  cancelled: "badge--error",
};

async function updateCustomerOrderStatus(id: number, status: CustomerOrderStatus) {
  try {
    await api.updateOrderStatus(props.session.token, id, status);
    await loadCustomerOrders();
    if (detailOrder.value && (detailOrder.value as any).id === id) {
      const updated = customerOrders.value.find(o => o.id === id);
      if (updated) detailOrder.value = updated;
    }
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    errorText.value = e?.message ?? "Помилка оновлення статусу";
  }
}

async function updateSupplierOrderStatus(id: number, status: string) {
  supplierStatusError.value = "";
  try {
    await api.updateSupplierOrderStatus(props.session.token, id, status);
    await loadSupplierOrders();
    if (detailOrder.value && (detailOrder.value as any).id === id) {
      const updated = supplierOrders.value.find(o => o.id === id);
      if (updated) detailOrder.value = updated;
    }
  } catch (e: any) {
    if (e?.message === 'SESSION_EXPIRED') emit('session-expired');
    const msg = e?.message ?? 'Помилка оновлення статусу';
    supplierStatusError.value = msg;
    errorText.value = msg;
  }
}

// ── Supplier Orders ──────────────────────────────────────────
const supplierOrders = ref<SupplierOrder[]>([]);
const supplierFilter = ref("");
const supplierStatusError = ref("");

// Receive form
const showReceiveForm = ref(false);
const receiveLines = ref<{ productId: number; productName: string; ordered: number; quantity: number; price: number }[]>([]);
const receiveNote = ref("");
const receiveSaving = ref(false);
const receiveError = ref("");

function openReceiveForm(order: any) {
  receiveLines.value = (order.items ?? []).map((item: any) => ({
    productId: item.productId,
    productName: item.productName || getProductName(item.productId),
    ordered: item.quantity,
    quantity: item.quantity,
    price: item.price,
  }));
  receiveNote.value = "";
  receiveError.value = "";
  showReceiveForm.value = true;
  // Load pending quantities to avoid exceeding ordered amounts
  api.supplierOrdersPending(props.session.token).then(pendingList => {
    const summary = pendingList.find((s: any) => s.orderId === order.id);
    if (!summary) return;
    receiveLines.value = receiveLines.value.map(line => {
      const pending = summary.pendingItems.find((p: any) => p.productId === line.productId);
      return { ...line, quantity: pending ? pending.pending : 0, ordered: pending ? pending.ordered : line.ordered };
    });
  }).catch(() => {});
}

async function submitReceive(orderId: number) {
  receiveError.value = "";
  const linesToSend = receiveLines.value.filter(l => l.quantity > 0);
  if (linesToSend.length === 0) {
    receiveError.value = "Вкажіть кількість хоча б для одного товару";
    return;
  }
  receiveSaving.value = true;
  try {
    const lines: SupplierOrderReceiveLine[] = linesToSend.map(l => ({
      productId: l.productId,
      quantity: l.quantity,
      price: l.price,
    }));
    await api.receiveSupplierOrderByLines(props.session.token, orderId, {
      lines,
      note: receiveNote.value,
    });
    showReceiveForm.value = false;
    await loadSupplierOrders();
    if (detailOrder.value && (detailOrder.value as any).id === orderId) {
      const updated = supplierOrders.value.find(o => o.id === orderId);
      if (updated) detailOrder.value = updated;
    }
  } catch (e: any) {
    if (e?.message === 'SESSION_EXPIRED') emit('session-expired');
    receiveError.value = e?.message ?? 'Помилка при отриманні товарів';
  } finally {
    receiveSaving.value = false;
  }
}

const filteredSupplierOrders = computed(() => {
  const q = supplierFilter.value.trim().toLowerCase();
  if (!q) return supplierOrders.value;
  return supplierOrders.value.filter((o) =>
    `#${o.id} ${o.status} ${o.currency}`.toLowerCase().includes(q)
  );
});

const supplierStatusLabels: Record<string, string> = {
  draft: "Чернетка",
  sent: "Надіслано",
  confirmed: "Підтверджено",
  in_transit: "В дорозі",
  received: "Отримано",
  closed: "Закрито",
  cancelled: "Скасовано",
};

const supplierStatusClass: Record<string, string> = {
  draft: "badge--neutral",
  sent: "badge--info",
  confirmed: "badge--ok",
  in_transit: "badge--warn",
  received: "badge--ok",
  closed: "badge--neutral",
  cancelled: "badge--error",
};

// ── Repair Orders ─────────────────────────────────────────────
const repairOrders = ref<ServiceOrder[]>([]);
const repairFilter = ref("");

const filteredRepairOrders = computed(() => {
  const q = repairFilter.value.trim().toLowerCase();
  if (!q) return repairOrders.value;
  return repairOrders.value.filter((o) =>
    `#${o.id} ${o.title} ${o.technician} ${o.status}`.toLowerCase().includes(q)
  );
});

const repairStatusLabel: Record<string, string> = {
  new: "Новий",
  in_progress: "В роботі",
  done: "Виконано",
  cancelled: "Скасовано",
};

const repairStatusClass: Record<string, string> = {
  new: "badge--info",
  in_progress: "badge--warn",
  done: "badge--ok",
  cancelled: "badge--error",
};

// ── Detail modal ──────────────────────────────────────────────
const showDetailModal = ref(false);
const detailOrder = ref<CustomerOrder | SupplierOrder | ServiceOrder | null>(null);
const detailType = ref<SubTab>("customer");

// ── Inline edit: customer name (customer orders) ────────────────
const editingCustomerName = ref(false);
const editCustomerNameValue = ref("");
const editCustomerNameSaving = ref(false);

async function startEditCustomerName() {
  editCustomerNameValue.value = (detailOrder.value as any).customerName ?? "";
  editingCustomerName.value = true;
}

async function saveCustomerName() {
  if (!editCustomerNameValue.value.trim()) return;
  editCustomerNameSaving.value = true;
  try {
    const order = detailOrder.value as any;
    const updated = await api.updateOrder(props.session.token, order.id, {
      customerName: editCustomerNameValue.value.trim(),
      currency: order.currency,
      dueDate: order.dueDate,
      items: order.items ?? [],
    });
    detailOrder.value = updated;
    const idx = customerOrders.value.findIndex(o => o.id === order.id);
    if (idx !== -1) customerOrders.value[idx] = updated as any;
    editingCustomerName.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    editCustomerNameSaving.value = false;
  }
}

// ── Inline edit: supplier (supplier orders) ─────────────────────
const editingSupplier = ref(false);
const editSupplierIdValue = ref(0);
const editSupplierSaving = ref(false);
const editSupplierSearch = ref("");

async function startEditSupplier() {
  const order = detailOrder.value as any;
  editSupplierIdValue.value = order.supplierId ?? 0;
  editSupplierSearch.value = order.supplierName ?? "";
  editingSupplier.value = true;
  if (allSuppliers.value.length === 0) {
    try { allSuppliers.value = await api.suppliers(props.session.token); } catch {}
  }
}

async function saveSupplier() {
  if (!editSupplierIdValue.value) return;
  editSupplierSaving.value = true;
  try {
    const order = detailOrder.value as any;
    await api.updateSupplierOrder(props.session.token, order.id, { customerOrderId: order.customerOrderId ?? null, supplierId: editSupplierIdValue.value });
    const supplier = allSuppliers.value.find(s => s.id === editSupplierIdValue.value);
    (detailOrder.value as any).supplierId = editSupplierIdValue.value;
    (detailOrder.value as any).supplierName = supplier?.name ?? "";
    const idx = supplierOrders.value.findIndex(o => (o as any).id === order.id);
    if (idx !== -1) {
      (supplierOrders.value[idx] as any).supplierId = editSupplierIdValue.value;
      (supplierOrders.value[idx] as any).supplierName = supplier?.name ?? "";
    }
    editingSupplier.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    editSupplierSaving.value = false;
  }
}

// ── Inline edit: dueDate ──────────────────────────────────────
const editingDueDate = ref(false);
const editDueDateValue = ref("");
const editDueDateSaving = ref(false);

function startEditDueDate() {
  const order = detailOrder.value as any;
  editDueDateValue.value = order.dueDate ? order.dueDate.slice(0, 10) : "";
  editingDueDate.value = true;
}

async function saveDueDate() {
  editDueDateSaving.value = true;
  try {
    const order = detailOrder.value as any;
    const updated = await api.updateOrder(props.session.token, order.id, {
      customerName: order.customerName,
      currency: order.currency,
      dueDate: editDueDateValue.value || undefined,
      items: order.items ?? [],
    });
    detailOrder.value = updated;
    const idx = customerOrders.value.findIndex(o => o.id === order.id);
    if (idx !== -1) customerOrders.value[idx] = updated as any;
    editingDueDate.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    editDueDateSaving.value = false;
  }
}

// ── Inline edit: currency ─────────────────────────────────────
const editingCurrency = ref(false);
const editCurrencyValue = ref<CurrencyCode>("UAH");
const editCurrencySaving = ref(false);

function startEditCurrency() {
  editCurrencyValue.value = (detailOrder.value as any).currency ?? "UAH";
  editingCurrency.value = true;
}

async function saveCurrency() {
  editCurrencySaving.value = true;
  try {
    const order = detailOrder.value as any;
    const updated = await api.updateOrder(props.session.token, order.id, {
      customerName: order.customerName,
      currency: editCurrencyValue.value,
      dueDate: order.dueDate,
      items: order.items ?? [],
    });
    detailOrder.value = updated;
    const idx = customerOrders.value.findIndex(o => o.id === order.id);
    if (idx !== -1) customerOrders.value[idx] = updated as any;
    editingCurrency.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    editCurrencySaving.value = false;
  }
}

// ── Add payment ───────────────────────────────────────────────
const showPaymentForm = ref(false);
const paymentAmount = ref(0);
const paymentMethod = ref<PaymentMethod>("cash");
const paymentCashboxId = ref<number | null>(null);
const paymentNote = ref("");
const paymentSaving = ref(false);
const paymentError = ref("");
const allCashboxes = ref<Cashbox[]>([]);

async function openPaymentForm() {
  paymentAmount.value = 0;
  paymentMethod.value = "cash";
  paymentNote.value = "";
  paymentError.value = "";
  paymentCashboxId.value = null;
  showPaymentForm.value = true;
  if (allCashboxes.value.length === 0) {
    try { allCashboxes.value = await api.cashboxes(props.session.token); } catch {}
  }
}

async function savePayment() {
  const order = detailOrder.value as any;
  if (!paymentAmount.value || paymentAmount.value <= 0) {
    paymentError.value = "Введіть суму оплати";
    return;
  }
  paymentSaving.value = true;
  paymentError.value = "";
  try {
    await api.createPayment(props.session.token, {
      orderId: order.id,
      amount: paymentAmount.value,
      currency: order.currency,
      method: paymentMethod.value,
      cashboxId: paymentCashboxId.value ?? undefined,
      note: paymentNote.value,
    });
    await loadCustomerOrders();
    const updated = customerOrders.value.find(o => o.id === order.id);
    if (updated) detailOrder.value = updated;
    showPaymentForm.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else paymentError.value = e?.message ?? "Помилка збереження";
  } finally {
    paymentSaving.value = false;
  }
}

// ── Inline edit: supplier currency ───────────────────────────
const editingSupplierCurrency = ref(false);
const editSupplierCurrencyValue = ref<CurrencyCode>("UAH");
const editSupplierCurrencySaving = ref(false);

function startEditSupplierCurrency() {
  editSupplierCurrencyValue.value = (detailOrder.value as any).currency ?? "UAH";
  editingSupplierCurrency.value = true;
}

async function saveSupplierCurrency() {
  editSupplierCurrencySaving.value = true;
  try {
    const order = detailOrder.value as any;
    // updateSupplierOrder only supports supplierId/customerOrderId; reload after
    await api.updateSupplierOrder(props.session.token, order.id, {
      supplierId: order.supplierId,
      customerOrderId: order.customerOrderId ?? null,
    });
    // Patch locally since backend doesn't expose currency update here
    (detailOrder.value as any).currency = editSupplierCurrencyValue.value;
    const idx = supplierOrders.value.findIndex((o: any) => o.id === order.id);
    if (idx !== -1) (supplierOrders.value[idx] as any).currency = editSupplierCurrencyValue.value;
    editingSupplierCurrency.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    editSupplierCurrencySaving.value = false;
  }
}

// ── Inline edit: repair fields ────────────────────────────────
const editingRepair = ref(false);
const editRepairForm = ref({ title: "", description: "", technician: "", laborMin: 60, price: 0, currency: "UAH" as CurrencyCode });
const editRepairSaving = ref(false);
const editRepairError = ref("");

function startEditRepair() {
  const order = detailOrder.value as any;
  editRepairForm.value = {
    title: order.title ?? "",
    description: order.description ?? "",
    technician: order.technician ?? "",
    laborMin: order.laborMin ?? 60,
    price: order.price ?? 0,
    currency: order.currency ?? "UAH",
  };
  editRepairError.value = "";
  editingRepair.value = true;
}

async function saveRepair() {
  if (!editRepairForm.value.title.trim()) { editRepairError.value = "Назва обов'язкова"; return; }
  editRepairSaving.value = true;
  editRepairError.value = "";
  try {
    const order = detailOrder.value as any;
    const updated = await api.updateServiceOrder(props.session.token, order.id, {
      title: editRepairForm.value.title.trim(),
      description: editRepairForm.value.description,
      technician: editRepairForm.value.technician,
      laborMin: editRepairForm.value.laborMin,
      price: editRepairForm.value.price,
      currency: editRepairForm.value.currency,
    });
    detailOrder.value = updated;
    const idx = repairOrders.value.findIndex((o: any) => o.id === order.id);
    if (idx !== -1) repairOrders.value[idx] = updated as any;
    editingRepair.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else editRepairError.value = e?.message ?? "Помилка збереження";
  } finally {
    editRepairSaving.value = false;
  }
}

// ── Repair order status update ────────────────────────────────
async function updateRepairOrderStatus(id: number, status: string) {
  try {
    const updated = await api.updateServiceOrderStatus(props.session.token, id, status as any);
    detailOrder.value = updated;
    const idx = repairOrders.value.findIndex((o: any) => o.id === id);
    if (idx !== -1) repairOrders.value[idx] = updated as any;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

// ── Add payment for repair order ──────────────────────────────
const showRepairPaymentForm = ref(false);
const repairPaymentAmount = ref(0);
const repairPaymentMethod = ref<PaymentMethod>("cash");
const repairPaymentCashboxId = ref<number | null>(null);
const repairPaymentNote = ref("");
const repairPaymentSaving = ref(false);
const repairPaymentError = ref("");

async function openRepairPaymentForm() {
  repairPaymentAmount.value = 0;
  repairPaymentMethod.value = "cash";
  repairPaymentNote.value = "";
  repairPaymentError.value = "";
  repairPaymentCashboxId.value = null;
  showRepairPaymentForm.value = true;
  if (allCashboxes.value.length === 0) {
    try { allCashboxes.value = await api.cashboxes(props.session.token); } catch {}
  }
}

async function saveRepairPayment() {
  const order = detailOrder.value as any;
  if (!repairPaymentAmount.value || repairPaymentAmount.value <= 0) {
    repairPaymentError.value = "Введіть суму оплати";
    return;
  }
  repairPaymentSaving.value = true;
  repairPaymentError.value = "";
  try {
    await api.createPayment(props.session.token, {
      serviceOrderId: order.id,
      amount: repairPaymentAmount.value,
      currency: order.currency,
      method: repairPaymentMethod.value,
      cashboxId: repairPaymentCashboxId.value ?? undefined,
      note: repairPaymentNote.value,
    });
    await loadRepairOrders();
    const updated = repairOrders.value.find((o: any) => o.id === order.id);
    if (updated) detailOrder.value = updated;
    showRepairPaymentForm.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else repairPaymentError.value = e?.message ?? "Помилка збереження";
  } finally {
    repairPaymentSaving.value = false;
  }
}

// ── Edit items in detail modal ────────────────────────────────
const editingItems = ref(false);
const detailItems = ref<Array<{ productId: number; quantity: number; price: number }>>([]);
const detailProductSearch = ref("");
const showDetailProductDropdown = ref(false);
const activeDetailItemIdx = ref<number | null>(null);
const detailItemsSaving = ref(false);
const detailItemsError = ref("");
const showNewDetailProductForm = ref(false);
const newDetailProductForm = ref({ name: "", sku: "", category: "", retailPrice: 0, purchasePrice: 0 });
const newDetailProductSaving = ref(false);

const filteredDetailProducts = computed(() => {
  const q = detailProductSearch.value.trim().toLowerCase();
  if (!q) return allProducts.value.filter(p => !p.archived).slice(0, 8);
  return allProducts.value.filter(p =>
    !p.archived && (
      p.name.toLowerCase().includes(q) ||
      p.sku.toLowerCase().includes(q) ||
      (p.barcode || "").includes(q)
    )
  ).slice(0, 10);
});

function startEditItems() {
  const order = detailOrder.value as any;
  detailItems.value = (order.items || []).map((i: any) => ({
    productId: i.productId,
    quantity: i.quantity,
    price: i.price,
  }));
  detailItemsError.value = "";
  editingItems.value = true;
}

function cancelEditItems() {
  editingItems.value = false;
  detailItems.value = [];
  detailProductSearch.value = "";
  showDetailProductDropdown.value = false;
  activeDetailItemIdx.value = null;
  showNewDetailProductForm.value = false;
  newDetailProductForm.value = { name: "", sku: "", category: "", retailPrice: 0, purchasePrice: 0 };
}

function addDetailItem() {
  detailItems.value.push({ productId: 0, quantity: 1, price: 0 });
}

function removeDetailItem(idx: number) {
  detailItems.value.splice(idx, 1);
}

function openDetailProductSearch(idx: number) {
  activeDetailItemIdx.value = idx;
  detailProductSearch.value = "";
  showDetailProductDropdown.value = true;
}

function selectDetailProduct(p: Product, idx: number) {
  detailItems.value[idx].productId = p.id;
  detailItems.value[idx].price = p.retailPrice;
  detailProductSearch.value = "";
  showDetailProductDropdown.value = false;
  activeDetailItemIdx.value = null;
}

async function saveNewDetailProduct() {
  if (!newDetailProductForm.value.name.trim()) return;
  newDetailProductSaving.value = true;
  try {
    const created = await api.createProduct(props.session.token, {
      ...newDetailProductForm.value,
      currency: "UAH",
      stock: 0,
      minStock: 1,
    }) as any;
    allProducts.value = await api.products(props.session.token);
    if (activeDetailItemIdx.value !== null) {
      detailItems.value[activeDetailItemIdx.value].productId = created.id;
      detailItems.value[activeDetailItemIdx.value].price = created.retailPrice;
    }
    showNewDetailProductForm.value = false;
    showDetailProductDropdown.value = false;
    newDetailProductForm.value = { name: "", sku: "", category: "", retailPrice: 0, purchasePrice: 0 };
    activeDetailItemIdx.value = null;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    detailItemsError.value = e?.message ?? "Помилка створення товару";
  } finally {
    newDetailProductSaving.value = false;
  }
}

async function saveDetailItems() {
  const order = detailOrder.value as any;
  detailItemsSaving.value = true;
  detailItemsError.value = "";
  try {
    const updated = await api.updateOrder(props.session.token, order.id, {
      customerName: order.customerName,
      currency: order.currency,
      dueDate: order.dueDate || undefined,
      items: detailItems.value.filter(i => i.productId > 0),
    });
    detailOrder.value = updated;
    await loadCustomerOrders();
    cancelEditItems();
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else detailItemsError.value = e?.message ?? "Помилка збереження";
  } finally {
    detailItemsSaving.value = false;
  }
}

async function openDetail(order: any, type: SubTab) {
  detailOrder.value = order;
  detailType.value = type;
  editingCustomerName.value = false;
  editingSupplier.value = false;
  editingItems.value = false;
  editingDueDate.value = false;
  editingCurrency.value = false;
  showPaymentForm.value = false;
  editingSupplierCurrency.value = false;
  editingRepair.value = false;
  showRepairPaymentForm.value = false;
  showReceiveForm.value = false;
  supplierStatusError.value = "";
  showDetailModal.value = true;
  // Load products for customer orders and repair details
  if (allProducts.value.length === 0 || (type === "repair" && allCustomers.value.length === 0)) {
    try {
      const [customers, products] = await Promise.all([
        api.customers(props.session.token),
        api.products(props.session.token),
      ]);
      allCustomers.value = customers;
      allProducts.value = products;
    } catch (e: any) {
      if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    }
  }
}

// ── Data loading ──────────────────────────────────────────────
async function loadCustomerOrders() {
  try {
    customerOrders.value = await api.orders(props.session.token);
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else errorText.value = e?.message ?? "Помилка завантаження замовлень клієнтів";
  }
}

async function loadSupplierOrders() {
  try {
    supplierOrders.value = await api.supplierOrders(props.session.token);
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else errorText.value = e?.message ?? "Помилка завантаження замовлень постачальників";
  }
}

async function loadRepairOrders() {
  try {
    repairOrders.value = await api.serviceOrders(props.session.token);
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else errorText.value = e?.message ?? "Помилка завантаження ремонтних замовлень";
  }
}

async function loadAll() {
  isLoading.value = true;
  errorText.value = "";
  await Promise.all([loadCustomerOrders(), loadSupplierOrders(), loadRepairOrders()]);
  isLoading.value = false;
}

function formatDate(dt: string) {
  return dt ? new Date(dt).toLocaleDateString("uk-UA") : "—";
}

function formatMoney(amount: number, currency: string) {
  return `${amount.toFixed(2)} ${currency}`;
}

onMounted(loadAll);

// ═══════════════════════════════════════════════════════════════
// CREATE ORDER MODAL
// ═══════════════════════════════════════════════════════════════
const showCreateModal = ref(false);
const createError = ref("");
const createLoading = ref(false);

// Shared data
const allCustomers = ref<Customer[]>([]);
const allSuppliers = ref<Supplier[]>([]);
const allProducts = ref<Product[]>([]);

// ── Customer search & quick-create ────────────────────────────
const customerSearch = ref("");
const showCustomerDropdown = ref(false);
const showNewCustomerForm = ref(false);
const newCustomerForm = ref({ name: "", phone: "", email: "", comment: "" });
const newCustomerSaving = ref(false);

const filteredCustomers = computed(() => {
  const q = customerSearch.value.trim().toLowerCase();
  if (!q) return allCustomers.value.slice(0, 8);
  return allCustomers.value.filter(c =>
    c.name.toLowerCase().includes(q) ||
    (c.phone || "").includes(q) ||
    (c.email || "").toLowerCase().includes(q)
  ).slice(0, 8);
});

function selectCustomer(c: Customer) {
  customerForm.value.customerName = c.name;
  customerSearch.value = c.name;
  showCustomerDropdown.value = false;
}

function selectRepairCustomer(c: Customer) {
  repairForm.value.customerId = c.id;
  repairCustomerSearch.value = c.name;
  showRepairCustomerDropdown.value = false;
}

async function saveNewCustomer(forRepair = false) {
  if (!newCustomerForm.value.name.trim()) return;
  newCustomerSaving.value = true;
  try {
    const created = await api.createCustomer(props.session.token, {
      name: newCustomerForm.value.name.trim(),
      phone: newCustomerForm.value.phone,
      email: newCustomerForm.value.email,
      comment: newCustomerForm.value.comment,
    });
    allCustomers.value = await api.customers(props.session.token);
    if (forRepair) {
      repairForm.value.customerId = (created as any).id;
      repairCustomerSearch.value = (created as any).name;
      showRepairCustomerDropdown.value = false;
    } else {
      customerForm.value.customerName = (created as any).name;
      customerSearch.value = (created as any).name;
      showCustomerDropdown.value = false;
    }
    showNewCustomerForm.value = false;
    newCustomerForm.value = { name: "", phone: "", email: "", comment: "" };
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    createError.value = e?.message ?? "Помилка створення клієнта";
  } finally {
    newCustomerSaving.value = false;
  }
}

// ── Supplier search & quick-create ───────────────────────────
const supplierSearch = ref("");
const showSupplierDropdown = ref(false);
const showNewSupplierForm = ref(false);
const newSupplierForm = ref({ name: "", contact: "", phone: "", email: "", comments: "" });
const newSupplierSaving = ref(false);

const filteredSuppliers = computed(() => {
  const q = supplierSearch.value.trim().toLowerCase();
  if (!q) return allSuppliers.value.slice(0, 8);
  return allSuppliers.value.filter(s =>
    s.name.toLowerCase().includes(q) ||
    (s.contact || "").toLowerCase().includes(q) ||
    (s.phone || "").includes(q)
  ).slice(0, 8);
});

function selectSupplier(s: Supplier) {
  supplierForm.value.supplierId = s.id;
  supplierSearch.value = s.name;
  showSupplierDropdown.value = false;
}

async function saveNewSupplier() {
  if (!newSupplierForm.value.name.trim()) return;
  newSupplierSaving.value = true;
  try {
    const created = await api.createSupplier(props.session.token, {
      name: newSupplierForm.value.name.trim(),
      contact: newSupplierForm.value.contact,
      phone: newSupplierForm.value.phone,
      email: newSupplierForm.value.email,
      comments: newSupplierForm.value.comments,
    });
    allSuppliers.value = await api.suppliers(props.session.token);
    supplierForm.value.supplierId = (created as any).id;
    supplierSearch.value = (created as any).name;
    showSupplierDropdown.value = false;
    showNewSupplierForm.value = false;
    newSupplierForm.value = { name: "", contact: "", phone: "", email: "", comments: "" };
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    createError.value = e?.message ?? "Помилка створення постачальника";
  } finally {
    newSupplierSaving.value = false;
  }
}

// ── Product search & quick-create ────────────────────────────
const productSearch = ref("");
const showProductDropdown = ref(false);
const showNewProductForm = ref(false);
const newProductForm = ref({ name: "", sku: "", category: "", retailPrice: 0, purchasePrice: 0 });
const newProductSaving = ref(false);
const activeProductItemIdx = ref<number | null>(null);

// Repair product search (for "пристрій" field in repair order form)
const repairProductSearch = ref("");
const showRepairProductDropdown = ref(false);
const showNewRepairProductForm = ref(false);
const newRepairProductForm = ref({ name: "", category: "", brand: "", retailPrice: 0, currency: "UAH" });
const newRepairProductSaving = ref(false);

const filteredProducts = computed(() => {
  const q = productSearch.value.trim().toLowerCase();
  if (!q) return allProducts.value.filter(p => !p.archived).slice(0, 8);
  return allProducts.value.filter(p =>
    !p.archived && (
      p.name.toLowerCase().includes(q) ||
      p.sku.toLowerCase().includes(q) ||
      (p.barcode || "").includes(q)
    )
  ).slice(0, 10);
});

const filteredRepairProducts = computed(() => {
  const q = repairProductSearch.value.trim().toLowerCase();
  if (!q) return allProducts.value.filter(p => !p.archived).slice(0, 8);
  return allProducts.value.filter(p =>
    !p.archived && (
      p.name.toLowerCase().includes(q) ||
      p.sku.toLowerCase().includes(q) ||
      (p.barcode || "").includes(q)
    )
  ).slice(0, 10);
});

function selectProduct(p: Product, idx: number) {
  orderItems.value[idx].productId = p.id;
  orderItems.value[idx].price = p.retailPrice;
  productSearch.value = "";
  showProductDropdown.value = false;
  activeProductItemIdx.value = null;
}

function selectRepairProduct(p: Product) {
  repairForm.value.productId = p.id;
  repairProductSearch.value = p.name;
  showRepairProductDropdown.value = false;
}

async function saveNewProduct() {
  if (!newProductForm.value.name.trim()) return;
  newProductSaving.value = true;
  try {
    const created = await api.createProduct(props.session.token, {
      ...newProductForm.value,
      currency: "UAH",
      stock: 0,
      minStock: 1,
    }) as any;
    allProducts.value = await api.products(props.session.token);
    if (activeProductItemIdx.value !== null) {
      orderItems.value[activeProductItemIdx.value].productId = created.id;
      orderItems.value[activeProductItemIdx.value].price = created.retailPrice;
    }
    showNewProductForm.value = false;
    showProductDropdown.value = false;
    newProductForm.value = { name: "", sku: "", category: "", retailPrice: 0, purchasePrice: 0 };
    activeProductItemIdx.value = null;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    createError.value = e?.message ?? "Помилка створення товару";
  } finally {
    newProductSaving.value = false;
  }
}

async function saveNewRepairProduct() {
  if (!newRepairProductForm.value.name.trim()) { createError.value = "Введіть назву товару"; return; }
  newRepairProductSaving.value = true;
  try {
    const created = await api.createProduct(props.session.token, {
      name: newRepairProductForm.value.name,
      category: newRepairProductForm.value.category,
      brand: newRepairProductForm.value.brand,
      retailPrice: newRepairProductForm.value.retailPrice,
      currency: newRepairProductForm.value.currency || "UAH",
      stock: 0, minStock: 0,
    } as any) as any;
    allProducts.value = await api.products(props.session.token);
    repairForm.value.productId = created.id;
    repairProductSearch.value = created.name;
    showNewRepairProductForm.value = false;
    showRepairProductDropdown.value = false;
    newRepairProductForm.value = { name: "", category: "", brand: "", retailPrice: 0, currency: "UAH" };
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    createError.value = e?.message ?? "Помилка створення товару";
  } finally {
    newRepairProductSaving.value = false;
  }
}

function getProductName(id: number) {
  return allProducts.value.find(p => p.id === id)?.name || `Товар #${id}`;
}

// ── Order items ───────────────────────────────────────────────
const orderItems = ref<Array<{ productId: number; quantity: number; price: number }>>([]);

function addOrderItem() {
  orderItems.value.push({ productId: 0, quantity: 1, price: 0 });
}
function removeOrderItem(idx: number) {
  orderItems.value.splice(idx, 1);
}
function openProductSearch(idx: number) {
  activeProductItemIdx.value = idx;
  productSearch.value = "";
  showProductDropdown.value = true;
}

// ── Repair customer search ────────────────────────────────────
const repairCustomerSearch = ref("");
const showRepairCustomerDropdown = ref(false);


function hideDropdown(setter: (v: boolean) => void) {
  setTimeout(() => setter(false), 150);
}

const filteredRepairCustomers = computed(() => {
  const q = repairCustomerSearch.value.trim().toLowerCase();
  if (!q) return allCustomers.value.slice(0, 8);
  return allCustomers.value.filter(c =>
    c.name.toLowerCase().includes(q) || (c.phone || "").includes(q)
  ).slice(0, 8);
});

// ── Forms ─────────────────────────────────────────────────────
const customerForm = ref({
  customerName: "",
  currency: "UAH" as CurrencyCode,
  dueDate: "",
  reserve: false,
});

const supplierForm = ref({
  supplierId: 0,
  currency: "UAH" as CurrencyCode,
});

const repairForm = ref({
  customerId: 0,
  productId: undefined as number | undefined,
  title: "",
  description: "",
  technician: "",
  laborMin: 60,
  price: 0,
  currency: "UAH" as CurrencyCode,
});

async function openCreateModal() {
  createError.value = "";
  showCreateModal.value = true;
  showNewCustomerForm.value = false;
  showNewSupplierForm.value = false;
  showNewProductForm.value = false;
  showNewRepairProductForm.value = false;
  customerSearch.value = "";
  supplierSearch.value = "";
  repairCustomerSearch.value = "";
  repairProductSearch.value = "";
  productSearch.value = "";
  orderItems.value = [];
  customerForm.value = { customerName: "", currency: "UAH", dueDate: "", reserve: false };
  supplierForm.value = { supplierId: 0, currency: "UAH" };
  repairForm.value = { customerId: 0, productId: undefined, title: "", description: "", technician: "", laborMin: 60, price: 0, currency: "UAH" };

  try {
    const [customers, suppliers, products] = await Promise.all([
      api.customers(props.session.token),
      api.suppliers(props.session.token),
      api.products(props.session.token),
    ]);
    allCustomers.value = customers;
    allSuppliers.value = suppliers;
    allProducts.value = products;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

async function submitCreateOrder() {
  createError.value = "";

  // Validate before locking the button
  if (activeSubTab.value === "customer") {
    if (!customerForm.value.customerName.trim()) {
      createError.value = "Вкажіть ім'я клієнта";
      return;
    }
  } else if (activeSubTab.value === "supplier") {
    if (!supplierForm.value.supplierId) {
      createError.value = "Оберіть постачальника";
      return;
    }
  } else if (activeSubTab.value === "repair") {
    if (!repairForm.value.title.trim()) {
      createError.value = "Вкажіть назву замовлення";
      return;
    }
    if (!repairForm.value.customerId) {
      createError.value = "Оберіть клієнта";
      return;
    }
  }

  createLoading.value = true;
  try {
    if (activeSubTab.value === "customer") {
      await api.createOrder(props.session.token, {
        customerName: customerForm.value.customerName.trim(),
        items: orderItems.value.filter(i => i.productId > 0),
        reserve: customerForm.value.reserve,
        dueDate: customerForm.value.dueDate || undefined,
        currency: customerForm.value.currency,
      });
      await loadCustomerOrders();
    } else if (activeSubTab.value === "supplier") {
      await api.createSupplierOrder(props.session.token, {
        supplierId: supplierForm.value.supplierId,
        currency: supplierForm.value.currency,
        items: orderItems.value.filter(i => i.productId > 0),
      });
      await loadSupplierOrders();
    } else if (activeSubTab.value === "repair") {
      await api.createServiceOrder(props.session.token, {
        customerId: repairForm.value.customerId,
        productId: repairForm.value.productId,
        title: repairForm.value.title.trim(),
        description: repairForm.value.description,
        technician: repairForm.value.technician || undefined,
        laborMin: repairForm.value.laborMin,
        price: repairForm.value.price,
        currency: repairForm.value.currency,
      });
      await loadRepairOrders();
    }
    showCreateModal.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else createError.value = e?.message ?? "Помилка створення замовлення";
  } finally {
    createLoading.value = false;
  }
}
</script>

<template>
  <main class="page-content">
    <div class="page-header">
      <h2>Замовлення</h2>
      <div style="display:flex;gap:0.5rem;align-items:center">
        <button class="primary-button" style="font-size:0.85rem" @click="openCreateModal">
          + Створити замовлення
        </button>
        <button class="ghost-button" style="font-size:0.85rem" @click="loadAll" :disabled="isLoading">
          {{ isLoading ? "Оновлення..." : "⟳ Оновити" }}
        </button>
      </div>
    </div>

    <p v-if="errorText" class="error-text" style="margin-bottom:0.75rem">{{ errorText }}</p>

    <div class="tab-row" style="margin-bottom:1.5rem">
      <button :class="['tab-button', activeSubTab === 'customer' && 'tab-button--active']"
        @click="activeSubTab = 'customer'">
        🛍 Замовлення клієнтів
        <span v-if="customerOrders.length" class="badge badge--neutral" style="margin-left:0.4rem">{{ customerOrders.length }}</span>
      </button>
      <button :class="['tab-button', activeSubTab === 'supplier' && 'tab-button--active']"
        @click="activeSubTab = 'supplier'">
        📋 Замовлення постачальника
        <span v-if="supplierOrders.length" class="badge badge--neutral" style="margin-left:0.4rem">{{ supplierOrders.length }}</span>
      </button>
      <button :class="['tab-button', activeSubTab === 'repair' && 'tab-button--active']"
        @click="activeSubTab = 'repair'">
        ⚙ Замовлення на ремонт
        <span v-if="repairOrders.length" class="badge badge--neutral" style="margin-left:0.4rem">{{ repairOrders.length }}</span>
      </button>
    </div>

    <!-- ═══ ЗАМОВЛЕННЯ КЛІЄНТІВ ═══════════════════════════════ -->
    <template v-if="activeSubTab === 'customer'">
      <div class="panel" style="margin-bottom:1rem">
        <input v-model="customerFilter" class="input" placeholder="Пошук за ім'ям, номером, статусом..." style="width:100%" />
      </div>
      <section class="panel">
        <p v-if="isLoading" class="subtle">Завантаження...</p>
        <table v-else-if="filteredCustomerOrders.length > 0">
          <thead>
            <tr><th>№</th><th>Клієнт</th><th>Статус</th><th>Сума</th><th>До видачі</th><th>Дата</th><th>Дії</th></tr>
          </thead>
          <tbody>
            <tr v-for="order in filteredCustomerOrders" :key="order.id"
              style="cursor:pointer"
              @click.stop="openDetail(order, 'customer')">
              <td>#{{ order.id }}</td>
              <td>{{ order.customerName }}</td>
              <td><span :class="['badge', customerStatusClass[order.status]]">{{ customerStatusLabel[order.status] ?? order.status }}</span></td>
              <td>{{ formatMoney(order.total, order.currency) }}</td>
              <td>{{ order.dueDate ? formatDate(order.dueDate) : "—" }}</td>
              <td>{{ formatDate(order.createdAt) }}</td>
              <td @click.stop>
                <select :value="order.status" class="input" style="font-size:0.78rem;padding:0.2rem 0.4rem"
                  @change="updateCustomerOrderStatus(order.id, ($event.target as HTMLSelectElement).value as CustomerOrderStatus)">
                  <option v-for="(label, val) in customerStatusLabel" :key="val" :value="val">{{ label }}</option>
                </select>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="subtle">Замовлень клієнтів поки немає.</p>
      </section>
    </template>

    <!-- ═══ ЗАМОВЛЕННЯ ПОСТАЧАЛЬНИКА ══════════════════════════ -->
    <template v-if="activeSubTab === 'supplier'">
      <div class="panel" style="margin-bottom:1rem">
        <input v-model="supplierFilter" class="input" placeholder="Пошук за номером, статусом, валютою..." style="width:100%" />
      </div>
      <section class="panel">
        <p v-if="isLoading" class="subtle">Завантаження...</p>
        <table v-else-if="filteredSupplierOrders.length > 0">
          <thead>
            <tr><th>№</th><th>Постачальник</th><th>Статус</th><th>Сума</th><th>Сума (UAH)</th><th>Позицій</th><th>Дата</th></tr>
          </thead>
          <tbody>
            <tr v-for="order in filteredSupplierOrders" :key="order.id"
              style="cursor:pointer"
              @click.stop="openDetail(order, 'supplier')">
              <td>#{{ order.id }}</td>
              <td>{{ (order as any).supplierName || `#${order.supplierId}` }}</td>
              <td><span :class="['badge', supplierStatusClass[order.status] ?? 'badge--neutral']">{{ supplierStatusLabels[order.status] ?? order.status }}</span></td>
              <td>{{ formatMoney(order.total, order.currency) }}</td>
              <td>{{ formatMoney(order.totalUah, 'UAH') }}</td>
              <td>{{ order.items?.length ?? 0 }}</td>
              <td>{{ formatDate(order.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="subtle">Замовлень постачальника поки немає.</p>
      </section>
    </template>

    <!-- ═══ ЗАМОВЛЕННЯ НА РЕМОНТ ══════════════════════════════ -->
    <template v-if="activeSubTab === 'repair'">
      <div class="panel" style="margin-bottom:1rem">
        <input v-model="repairFilter" class="input" placeholder="Пошук за назвою, техніком, статусом..." style="width:100%" />
      </div>
      <section class="panel">
        <p v-if="isLoading" class="subtle">Завантаження...</p>
        <table v-else-if="filteredRepairOrders.length > 0">
          <thead>
            <tr><th>№</th><th>Назва</th><th>Технік</th><th>Статус</th><th>Сума</th><th>Борг</th><th>Дата</th></tr>
          </thead>
          <tbody>
            <tr v-for="order in filteredRepairOrders" :key="order.id"
              style="cursor:pointer"
              @click.stop="openDetail(order, 'repair')">
              <td>#{{ order.id }}</td>
              <td>
                <div>{{ order.title }}</div>
                <div class="subtle" style="font-size:0.78rem">{{ order.description }}</div>
              </td>
              <td>{{ order.technician || "—" }}</td>
              <td><span :class="['badge', repairStatusClass[order.status] ?? 'badge--neutral']">{{ repairStatusLabel[order.status] ?? order.status }}</span></td>
              <td>{{ formatMoney(order.total, order.currency) }}</td>
              <td :style="order.debt > 0 ? 'color:var(--danger)' : ''">{{ formatMoney(order.debt, order.currency) }}</td>
              <td>{{ formatDate(order.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="subtle">Замовлень на ремонт поки немає.</p>
      </section>
    </template>

    <!-- ═══ МОДАЛЬНЕ ВІКНО ДЕТАЛЕЙ ЗАМОВЛЕННЯ ════════════════ -->
    <teleport to="body">
      <div v-if="showDetailModal && detailOrder" class="modal-backdrop" @click.self="showDetailModal = false">
        <div class="modal-box" style="max-width:600px;width:100%;">
          <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:1rem;border-bottom:1px solid var(--border);padding-bottom:0.75rem">
            <h3 style="margin:0;font-size:1rem">
              <template v-if="detailType === 'customer'">🛍 Замовлення клієнта #{{ (detailOrder as any).id }}</template>
              <template v-else-if="detailType === 'supplier'">📋 Замовлення постачальника #{{ (detailOrder as any).id }}</template>
              <template v-else>⚙ Замовлення на ремонт #{{ (detailOrder as any).id }}</template>
            </h3>
            <button class="ghost-button" style="font-size:1.2rem;padding:0 0.4rem;line-height:1" @click="showDetailModal = false">✕</button>
          </div>

          <!-- Customer Order Detail -->
          <template v-if="detailType === 'customer'">
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.6rem;font-size:0.88rem;margin-bottom:1rem">
              <div style="display:flex;align-items:center;gap:0.4rem">
                <span class="subtle">Клієнт:</span>
                <template v-if="!editingCustomerName">
                  <strong>{{ (detailOrder as any).customerName }}</strong>
                  <button class="ghost-button" style="font-size:0.72rem;padding:0.1rem 0.4rem;margin-left:0.2rem" @click="startEditCustomerName">✏</button>
                </template>
                <template v-else>
                  <input v-model="editCustomerNameValue" class="input" style="flex:1;font-size:0.85rem;padding:0.25rem 0.5rem" @keydown.enter="saveCustomerName" @keydown.escape="editingCustomerName=false" />
                  <button style="font-size:0.78rem;padding:0.2rem 0.6rem;background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);cursor:pointer" :disabled="editCustomerNameSaving" @click="saveCustomerName">{{ editCustomerNameSaving ? '...' : '✓' }}</button>
                  <button class="ghost-button" style="font-size:0.78rem;padding:0.2rem 0.5rem" @click="editingCustomerName=false">✕</button>
                </template>
              </div>
              <div><span class="subtle">Статус:</span>
                <span :class="['badge', customerStatusClass[(detailOrder as any).status]]" style="margin-left:0.3rem">
                  {{ customerStatusLabel[(detailOrder as any).status] ?? (detailOrder as any).status }}
                </span>
              </div>
              <div><span class="subtle">Сума:</span> <strong>{{ formatMoney((detailOrder as any).total, (detailOrder as any).currency) }}</strong></div>
              <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap">
                <span class="subtle">Оплачено:</span>
                <span>{{ formatMoney((detailOrder as any).paid ?? 0, (detailOrder as any).currency) }}</span>
                <button v-if="!showPaymentForm" class="ghost-button" style="font-size:0.72rem;padding:0.1rem 0.4rem" @click="openPaymentForm">+ Оплата</button>
              </div>
              <div><span class="subtle">Борг:</span>
                <span :style="(detailOrder as any).debt > 0 ? 'color:var(--danger);font-weight:700' : ''">
                  {{ formatMoney((detailOrder as any).debt ?? 0, (detailOrder as any).currency) }}
                </span>
              </div>
              <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap">
                <span class="subtle">До видачі:</span>
                <template v-if="!editingDueDate">
                  <span>{{ (detailOrder as any).dueDate ? formatDate((detailOrder as any).dueDate) : '—' }}</span>
                  <button class="ghost-button" style="font-size:0.72rem;padding:0.1rem 0.4rem" @click="startEditDueDate">✏</button>
                </template>
                <template v-else>
                  <input type="date" v-model="editDueDateValue" class="input" style="font-size:0.82rem;padding:0.2rem 0.4rem" @keydown.enter="saveDueDate" @keydown.escape="editingDueDate=false" />
                  <button style="font-size:0.75rem;padding:0.2rem 0.5rem;background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);cursor:pointer" :disabled="editDueDateSaving" @click="saveDueDate">{{ editDueDateSaving ? '...' : '✓' }}</button>
                  <button class="ghost-button" style="font-size:0.75rem;padding:0.2rem 0.4rem" @click="editingDueDate=false">✕</button>
                </template>
              </div>
              <div><span class="subtle">Дата створення:</span> {{ formatDate((detailOrder as any).createdAt) }}</div>
              <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap">
                <span class="subtle">Валюта:</span>
                <template v-if="!editingCurrency">
                  <span>{{ (detailOrder as any).currency }}</span>
                  <button class="ghost-button" style="font-size:0.72rem;padding:0.1rem 0.4rem" @click="startEditCurrency">✏</button>
                </template>
                <template v-else>
                  <select v-model="editCurrencyValue" class="input" style="font-size:0.82rem;padding:0.2rem 0.4rem">
                    <option value="UAH">UAH</option>
                    <option value="USD">USD</option>
                    <option value="EUR">EUR</option>
                  </select>
                  <button style="font-size:0.75rem;padding:0.2rem 0.5rem;background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);cursor:pointer" :disabled="editCurrencySaving" @click="saveCurrency">{{ editCurrencySaving ? '...' : '✓' }}</button>
                  <button class="ghost-button" style="font-size:0.75rem;padding:0.2rem 0.4rem" @click="editingCurrency=false">✕</button>
                </template>
              </div>
              <div v-if="(detailOrder as any).reserve"><span class="subtle">Резерв:</span> ✓ Так</div>
            </div>

            <!-- Payment form -->
            <div v-if="showPaymentForm" style="background:rgba(122,185,154,0.06);border:1px solid rgba(122,185,154,0.2);border-radius:8px;padding:0.75rem;margin-bottom:1rem">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.6rem">
                <span style="font-size:0.82rem;font-weight:600;color:var(--accent)">💳 Додати оплату</span>
                <button class="ghost-button" style="font-size:0.8rem;padding:0.1rem 0.4rem" @click="showPaymentForm=false">✕</button>
              </div>
              <p v-if="paymentError" style="color:var(--danger);font-size:0.8rem;margin:0 0 0.4rem">{{ paymentError }}</p>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Сума *
                  <input type="number" v-model.number="paymentAmount" min="0" step="0.01" class="input" style="margin-top:0.2rem" placeholder="0.00">
                </label>
                <label style="font-size:0.78rem">Метод
                  <select v-model="paymentMethod" class="input" style="margin-top:0.2rem">
                    <option value="cash">Готівка</option>
                    <option value="card">Картка</option>
                    <option value="bank">Банк</option>
                    <option value="virtual">Інше</option>
                  </select>
                </label>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.6rem">
                <label style="font-size:0.78rem">Каса
                  <select v-model="paymentCashboxId" class="input" style="margin-top:0.2rem">
                    <option :value="null">— без каси —</option>
                    <option v-for="cb in allCashboxes" :key="cb.id" :value="cb.id">{{ cb.name }}</option>
                  </select>
                </label>
                <label style="font-size:0.78rem">Примітка
                  <input v-model="paymentNote" class="input" style="margin-top:0.2rem" placeholder="Необов'язково">
                </label>
              </div>
              <div style="display:flex;gap:0.5rem">
                <button class="ghost-button" style="font-size:0.8rem" @click="showPaymentForm=false" :disabled="paymentSaving">Скасувати</button>
                <button style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.35rem 0.9rem;font-size:0.82rem;cursor:pointer;font-weight:600" @click="savePayment" :disabled="paymentSaving">
                  {{ paymentSaving ? 'Збереження...' : '✓ Зберегти' }}
                </button>
              </div>
            </div>
            
            <div style="border-top:1px solid var(--border);padding-top:0.75rem;margin-bottom:1rem">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
                <div style="font-size:0.82rem;font-weight:600;color:var(--accent)">📦 Товари</div>
                <button v-if="!editingItems" class="ghost-button" style="font-size:0.78rem;padding:0.2rem 0.6rem"
                  @click="startEditItems">✏ Редагувати</button>
              </div>
              <!-- View mode -->
              <template v-if="!editingItems">
                <table v-if="(detailOrder as any).items?.length" style="font-size:0.84rem">
                  <thead><tr><th>Товар</th><th>К-сть</th><th>Ціна</th><th>Сума</th></tr></thead>
                  <tbody>
                    <tr v-for="item in (detailOrder as any).items" :key="item.id">
                      <td>{{ item.productName || getProductName(item.productId) }}</td>
                      <td>{{ item.quantity }}</td>
                      <td>{{ item.price }} {{ (detailOrder as any).currency }}</td>
                      <td>{{ (item.quantity * item.price).toFixed(2) }}</td>
                    </tr>
                  </tbody>
                </table>
                <p v-else class="subtle" style="font-size:0.82rem;margin:0">Товари не додані</p>
              </template>
              <!-- Edit mode -->
              <template v-else>
                <p v-if="detailItemsError" class="error-text" style="margin-bottom:0.5rem;font-size:0.82rem">{{ detailItemsError }}</p>
                <div v-for="(item, idx) in detailItems" :key="idx"
                  style="background:rgba(122,185,154,0.05);border:1px solid rgba(122,185,154,0.15);border-radius:8px;padding:0.6rem;margin-bottom:0.5rem">
                  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.4rem">
                    <span style="font-size:0.8rem;color:var(--text-muted)">Позиція {{ idx + 1 }}</span>
                    <button class="ghost-button" style="padding:0.15rem 0.45rem;font-size:0.8rem;color:var(--danger)" @click="removeDetailItem(idx)">✕</button>
                  </div>
                  <div style="position:relative;margin-bottom:0.4rem">
                    <div v-if="item.productId > 0" style="display:flex;align-items:center;gap:0.4rem;padding:0.4rem 0.6rem;background:rgba(122,185,154,0.1);border-radius:6px;font-size:0.85rem">
                      <span style="flex:1;font-weight:500">{{ getProductName(item.productId) }}</span>
                      <button class="ghost-button" style="padding:0.1rem 0.4rem;font-size:0.75rem"
                        @click="item.productId = 0; openDetailProductSearch(idx)">змінити</button>
                    </div>
                    <div v-else>
                      <input
                        :value="activeDetailItemIdx === idx ? detailProductSearch : ''"
                        class="input" placeholder="Пошук товару за назвою, SKU..." style="width:100%"
                        @focus="openDetailProductSearch(idx)"
                        @input="detailProductSearch = ($event.target as HTMLInputElement).value; showDetailProductDropdown = true"
                        @blur="hideDropdown(v => showDetailProductDropdown = v)"
                      />
                      <div v-if="showDetailProductDropdown && activeDetailItemIdx === idx && (filteredDetailProducts.length > 0 || detailProductSearch.trim())"
                        style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                        <div v-if="filteredDetailProducts.length === 0 && detailProductSearch.trim()"
                          style="padding:0.5rem 0.75rem;font-size:0.84rem;color:var(--text-muted)">
                          Товар "{{ detailProductSearch }}" не знайдено
                        </div>
                        <div v-for="p in filteredDetailProducts" :key="p.id"
                          style="padding:0.45rem 0.75rem;cursor:pointer;border-bottom:1px solid rgba(122,185,154,0.08);font-size:0.84rem"
                          @mousedown.prevent="selectDetailProduct(p, idx)"
                          @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(122,185,154,0.1)'"
                          @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                          <span style="font-weight:600">{{ p.name }}</span>
                          <span class="subtle" style="font-size:0.75rem;margin-left:0.4rem">{{ p.sku }}</span>
                          <span style="float:right;color:var(--accent);font-weight:600;font-size:0.82rem">{{ p.retailPrice }} {{ p.currency }}</span>
                        </div>
                        <div style="padding:0.4rem 0.75rem;border-top:1px solid var(--border)">
                          <button class="ghost-button" style="font-size:0.78rem;width:100%"
                            @mousedown.prevent="showNewDetailProductForm = true; showDetailProductDropdown = false; newDetailProductForm.name = detailProductSearch">
                            + Створити новий товар{{ detailProductSearch.trim() ? ` "${detailProductSearch}"` : '' }}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div style="display:grid;grid-template-columns:80px 100px;gap:0.5rem">
                    <label style="font-size:0.75rem;color:var(--text-muted)">К-сть
                      <input type="number" v-model.number="item.quantity" min="1" class="input" style="margin-top:0.15rem">
                    </label>
                    <label style="font-size:0.75rem;color:var(--text-muted)">Ціна
                      <input type="number" v-model.number="item.price" min="0" class="input" style="margin-top:0.15rem">
                    </label>
                  </div>
                </div>
                <button class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.65rem;margin-bottom:0.75rem" @click="addDetailItem">+ Додати позицію</button>

                <!-- Quick create product form (detail modal) -->
                <div v-if="showNewDetailProductForm"
                  style="background:rgba(122,185,154,0.06);border:1px solid rgba(122,185,154,0.2);border-radius:8px;padding:0.75rem;margin-bottom:0.85rem">
                  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
                    <span style="font-size:0.8rem;font-weight:600;color:var(--accent)">Новий товар</span>
                    <button class="ghost-button" style="font-size:0.8rem" @click="showNewDetailProductForm = false">✕</button>
                  </div>
                  <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                    <label style="font-size:0.78rem">Назва *<input v-model="newDetailProductForm.name" class="input" placeholder="Назва товару" style="margin-top:0.2rem"></label>
                    <label style="font-size:0.78rem">SKU<input v-model="newDetailProductForm.sku" class="input" placeholder="Артикул" style="margin-top:0.2rem"></label>
                  </div>
                  <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                    <label style="font-size:0.78rem">Категорія<input v-model="newDetailProductForm.category" class="input" style="margin-top:0.2rem"></label>
                    <label style="font-size:0.78rem">Ціна закупівлі<input type="number" v-model.number="newDetailProductForm.purchasePrice" min="0" class="input" style="margin-top:0.2rem"></label>
                    <label style="font-size:0.78rem">Ціна продажу *<input type="number" v-model.number="newDetailProductForm.retailPrice" min="0" class="input" style="margin-top:0.2rem"></label>
                  </div>
                  <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newDetailProductSaving || !newDetailProductForm.name.trim()"
                    @click="saveNewDetailProduct">
                    {{ newDetailProductSaving ? '...' : '✓ Зберегти товар' }}
                  </button>
                </div>

                <div style="display:flex;gap:0.5rem">
                  <button class="ghost-button" style="font-size:0.82rem" @click="cancelEditItems" :disabled="detailItemsSaving">Скасувати</button>
                  <button style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.4rem 1rem;font-size:0.82rem;cursor:pointer;font-weight:600"
                    @click="saveDetailItems" :disabled="detailItemsSaving">
                    {{ detailItemsSaving ? 'Збереження...' : '✓ Зберегти товари' }}
                  </button>
                </div>
              </template>
            </div>
            <!-- Linked Supplier Orders -->
            <div style="border-top:1px solid var(--border);padding-top:0.75rem;margin-bottom:1rem">
              <div style="font-size:0.82rem;font-weight:600;color:#60c0e0;margin-bottom:0.5rem">🔗 Пов'язані замовлення постачальнику</div>
              <template v-if="supplierOrders.filter(o => o.customerOrderId === (detailOrder as any).id).length > 0">
                <div v-for="so in supplierOrders.filter(o => o.customerOrderId === (detailOrder as any).id)" :key="so.id"
                  style="display:flex;align-items:center;gap:0.5rem;padding:0.4rem 0.6rem;background:rgba(96,192,224,0.07);border:1px solid rgba(96,192,224,0.2);border-radius:6px;margin-bottom:0.35rem;font-size:0.85rem;cursor:pointer"
                  @click="showDetailModal=false; $nextTick(() => { detailOrder = so; detailType = 'supplier'; showDetailModal = true })">
                  <span style="color:#60c0e0;font-weight:600">📋 #{{ so.id }}</span>
                  <span class="subtle">{{ (so as any).supplierName || allSuppliers.find(s => s.id === so.supplierId)?.name || `Постачальник #${so.supplierId}` }}</span>
                  <span :class="['badge', supplierStatusClass[so.status] ?? 'badge--neutral']" style="margin-left:auto">{{ supplierStatusLabels[so.status] ?? so.status }}</span>
                  <span style="color:var(--text-muted);font-size:0.78rem">{{ formatMoney(so.total, so.currency) }}</span>
                </div>
              </template>
              <p v-else class="subtle" style="font-size:0.82rem;margin:0">Немає пов'язаних замовлень постачальнику</p>
            </div>

            <div style="margin-top:1rem;padding-top:0.75rem;border-top:1px solid var(--border)">
              <div style="font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Оновити статус:</div>
              <select class="input" style="font-size:0.85rem" :value="(detailOrder as any).status"
                @change="updateCustomerOrderStatus((detailOrder as any).id, ($event.target as HTMLSelectElement).value as CustomerOrderStatus)">
                <option v-for="(label, val) in customerStatusLabel" :key="val" :value="val">{{ label }}</option>
              </select>
            </div>
          </template>

          <!-- Supplier Order Detail -->
          <template v-else-if="detailType === 'supplier'">
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.6rem;font-size:0.88rem;margin-bottom:1rem">
              <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap">
                <span class="subtle">Постачальник:</span>
                <template v-if="!editingSupplier">
                  <strong>{{ (detailOrder as any).supplierName || `#${(detailOrder as any).supplierId}` }}</strong>
                  <button class="ghost-button" style="font-size:0.72rem;padding:0.1rem 0.4rem;margin-left:0.2rem" @click="startEditSupplier">✏</button>
                </template>
                <template v-else>
                  <select v-model="editSupplierIdValue" class="input" style="flex:1;font-size:0.85rem;padding:0.25rem 0.5rem">
                    <option :value="0" disabled>— оберіть постачальника —</option>
                    <option v-for="s in allSuppliers" :key="s.id" :value="s.id">{{ s.name }}</option>
                  </select>
                  <button style="font-size:0.78rem;padding:0.2rem 0.6rem;background:#60c0e0;color:#fff;border:none;border-radius:var(--radius-sm);cursor:pointer" :disabled="editSupplierSaving || !editSupplierIdValue" @click="saveSupplier">{{ editSupplierSaving ? '...' : '✓' }}</button>
                  <button class="ghost-button" style="font-size:0.78rem;padding:0.2rem 0.5rem" @click="editingSupplier=false">✕</button>
                </template>
              </div>
              <div><span class="subtle">Статус:</span>
                <span :class="['badge', supplierStatusClass[(detailOrder as any).status] ?? 'badge--neutral']" style="margin-left:0.3rem">
                  {{ supplierStatusLabels[(detailOrder as any).status] ?? (detailOrder as any).status }}
                </span>
              </div>
              <div><span class="subtle">Сума:</span> <strong>{{ formatMoney((detailOrder as any).total, (detailOrder as any).currency) }}</strong></div>
              <div><span class="subtle">Сума (UAH):</span> {{ formatMoney((detailOrder as any).totalUah, 'UAH') }}</div>
              <div><span class="subtle">Кількість позицій:</span> {{ (detailOrder as any).items?.length ?? 0 }}</div>
              <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap">
                <span class="subtle">Валюта:</span>
                <template v-if="!editingSupplierCurrency">
                  <span>{{ (detailOrder as any).currency }}</span>
                  <button class="ghost-button" style="font-size:0.72rem;padding:0.1rem 0.4rem" @click="startEditSupplierCurrency">✏</button>
                </template>
                <template v-else>
                  <select v-model="editSupplierCurrencyValue" class="input" style="font-size:0.82rem;padding:0.2rem 0.4rem">
                    <option value="UAH">UAH</option>
                    <option value="USD">USD</option>
                    <option value="EUR">EUR</option>
                  </select>
                  <button style="font-size:0.75rem;padding:0.2rem 0.5rem;background:#60c0e0;color:#fff;border:none;border-radius:var(--radius-sm);cursor:pointer" :disabled="editSupplierCurrencySaving" @click="saveSupplierCurrency">{{ editSupplierCurrencySaving ? '...' : '✓' }}</button>
                  <button class="ghost-button" style="font-size:0.75rem;padding:0.2rem 0.4rem" @click="editingSupplierCurrency=false">✕</button>
                </template>
              </div>
              <div><span class="subtle">Дата створення:</span> {{ formatDate((detailOrder as any).createdAt) }}</div>
              <div v-if="(detailOrder as any).customerOrderId" style="grid-column:1/-1;display:flex;align-items:center;gap:0.5rem">
                <span class="subtle">Замовлення клієнта:</span>
                <span
                  style="display:inline-flex;align-items:center;gap:0.35rem;padding:0.2rem 0.55rem;background:rgba(122,185,154,0.1);border:1px solid rgba(122,185,154,0.25);border-radius:6px;font-size:0.84rem;cursor:pointer"
                  @click="showDetailModal=false; $nextTick(() => { const co = customerOrders.find(o => o.id === (detailOrder as any).customerOrderId); if(co){ detailOrder = co; detailType = 'customer'; showDetailModal = true; } })">
                  🛍 <strong>#{{ (detailOrder as any).customerOrderId }}</strong>
                  <span class="subtle" style="font-size:0.78rem">{{ customerOrders.find(o => o.id === (detailOrder as any).customerOrderId)?.customerName }}</span>
                </span>
              </div>
            </div>
            <div v-if="(detailOrder as any).items?.length" style="border-top:1px solid var(--border);padding-top:0.75rem;margin-bottom:1rem">
              <div style="font-size:0.82rem;font-weight:600;color:#60c0e0;margin-bottom:0.5rem">📦 Товари</div>
              <table style="font-size:0.84rem">
                <thead><tr><th>Товар</th><th>К-сть</th><th>Ціна закуп.</th><th>Сума</th></tr></thead>
                <tbody>
                  <tr v-for="item in (detailOrder as any).items" :key="item.id">
                    <td>{{ item.productName || getProductName(item.productId) }}</td>
                    <td>{{ item.quantity }}</td>
                    <td>{{ item.price }} {{ (detailOrder as any).currency }}</td>
                    <td>{{ (item.quantity * item.price).toFixed(2) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <!-- Receive form -->
            <div v-if="showReceiveForm && ['sent','confirmed','in_transit'].includes((detailOrder as any).status)"
              style="border-top:1px solid var(--border);padding-top:0.75rem;margin-bottom:0.75rem">
              <div style="font-size:0.85rem;font-weight:600;color:#60c0e0;margin-bottom:0.6rem">📥 Отримання товарів</div>
              <table style="font-size:0.84rem;width:100%;margin-bottom:0.5rem">
                <thead><tr><th style="text-align:left">Товар</th><th>Замовлено</th><th>До отримання</th><th>Ціна</th></tr></thead>
                <tbody>
                  <tr v-for="(line, i) in receiveLines" :key="line.productId">
                    <td style="padding:0.2rem 0.4rem 0.2rem 0">{{ line.productName }}</td>
                    <td style="text-align:center;color:var(--text-muted)">{{ line.ordered }}</td>
                    <td style="text-align:center">
                      <input type="number" :min="0" :max="line.quantity" v-model.number="receiveLines[i].quantity"
                        style="width:60px;padding:0.15rem 0.3rem;background:var(--bg-input,#1e2230);border:1px solid var(--border);border-radius:4px;color:inherit;text-align:center" />
                    </td>
                    <td style="text-align:center">
                      <input type="number" :min="0" v-model.number="receiveLines[i].price"
                        style="width:70px;padding:0.15rem 0.3rem;background:var(--bg-input,#1e2230);border:1px solid var(--border);border-radius:4px;color:inherit;text-align:center" />
                    </td>
                  </tr>
                </tbody>
              </table>
              <input v-model="receiveNote" placeholder="Примітка (необов'язково)"
                style="width:100%;padding:0.3rem 0.5rem;margin-bottom:0.5rem;background:var(--bg-input,#1e2230);border:1px solid var(--border);border-radius:4px;color:inherit;font-size:0.84rem;box-sizing:border-box" />
              <p v-if="receiveError" class="error-text" style="font-size:0.82rem;margin-bottom:0.4rem">{{ receiveError }}</p>
              <div style="display:flex;gap:0.5rem">
                <button :disabled="receiveSaving"
                  style="font-size:0.82rem;padding:0.3rem 0.9rem;background:rgba(109,212,160,0.15);border:1px solid rgba(109,212,160,0.4);color:#6dd4a0;border-radius:6px;cursor:pointer"
                  @click="submitReceive((detailOrder as any).id)">{{ receiveSaving ? 'Збереження...' : '✓ Зберегти отримання' }}</button>
                <button class="ghost-button" style="font-size:0.82rem;padding:0.3rem 0.7rem" @click="showReceiveForm=false">Скасувати</button>
              </div>
            </div>

            <div style="border-top:1px solid var(--border);padding-top:0.75rem">
              <div style="font-size:0.8rem;color:var(--text-muted);margin-bottom:0.5rem">Змінити статус:</div>
              <div style="display:flex;gap:0.4rem;flex-wrap:wrap">
                <button v-if="(detailOrder as any).status==='draft'" class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem"
                  @click="updateSupplierOrderStatus((detailOrder as any).id,'sent')">📤 Надіслано</button>
                <button v-if="(detailOrder as any).status==='sent'" class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem"
                  @click="updateSupplierOrderStatus((detailOrder as any).id,'confirmed')">✅ Підтверджено</button>
                <button v-if="['sent','confirmed'].includes((detailOrder as any).status)" class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem"
                  @click="updateSupplierOrderStatus((detailOrder as any).id,'in_transit')">🚛 В дорозі</button>
                <button v-if="['sent','confirmed','in_transit'].includes((detailOrder as any).status)"
                  style="font-size:0.8rem;padding:0.3rem 0.9rem;background:rgba(96,192,224,0.12);border:1px solid rgba(96,192,224,0.35);color:#60c0e0;border-radius:6px;cursor:pointer"
                  @click="openReceiveForm(detailOrder)">📥 Прийняти товари</button>
                <button v-if="(detailOrder as any).status === 'in_transit'"
                  style="font-size:0.8rem;padding:0.3rem 0.9rem;background:rgba(109,212,160,0.15);border:1px solid rgba(109,212,160,0.4);color:#6dd4a0;border-radius:6px;cursor:pointer"
                  @click="updateSupplierOrderStatus((detailOrder as any).id,'received')">📦 Отримано</button>
                <button v-if="(detailOrder as any).status==='received'" class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem"
                  @click="updateSupplierOrderStatus((detailOrder as any).id,'closed')">🔒 Закрити</button>
                <button v-if="!['closed','cancelled','received'].includes((detailOrder as any).status)"
                  class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem;color:#ff9ca0"
                  @click="updateSupplierOrderStatus((detailOrder as any).id,'cancelled')">✕ Скасувати</button>
                <span v-if="['closed','cancelled'].includes((detailOrder as any).status)" class="subtle" style="font-size:0.82rem;align-self:center">
                  Статус не можна змінити
                </span>
              </div>
              <p v-if="supplierStatusError" class="error-text" style="margin-top:0.5rem;font-size:0.82rem">{{ supplierStatusError }}</p>
            </div>
          </template>

          <!-- Repair Order Detail -->
          <template v-else>
            <!-- View mode -->
            <template v-if="!editingRepair">
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.6rem;font-size:0.88rem;margin-bottom:1rem">
                <div style="display:flex;align-items:center;gap:0.4rem">
                  <span class="subtle">Назва:</span>
                  <strong>{{ (detailOrder as any).title }}</strong>
                </div>
                <div><span class="subtle">Статус:</span>
                  <span :class="['badge', repairStatusClass[(detailOrder as any).status] ?? 'badge--neutral']" style="margin-left:0.3rem">
                    {{ repairStatusLabel[(detailOrder as any).status] ?? (detailOrder as any).status }}
                  </span>
                </div>
                <div><span class="subtle">Клієнт:</span> <strong>{{ allCustomers.find(c => c.id === (detailOrder as any).customerId)?.name || `#${(detailOrder as any).customerId}` }}</strong></div>
                <div><span class="subtle">Пристрій:</span> {{ allProducts.find(p => p.id === (detailOrder as any).productId)?.name || ((detailOrder as any).productId ? `#${(detailOrder as any).productId}` : '—') }}</div>
                <div><span class="subtle">Технік:</span> {{ (detailOrder as any).technician || '—' }}</div>
                <div><span class="subtle">Тривалість:</span> {{ (detailOrder as any).laborMin }} хв</div>
                <div><span class="subtle">Вартість роботи:</span> {{ (detailOrder as any).price }} {{ (detailOrder as any).currency }}</div>
                <div><span class="subtle">Запчастини:</span> {{ (detailOrder as any).partsTotal }} {{ (detailOrder as any).currency }}</div>
                <div><span class="subtle">Разом:</span> <strong>{{ formatMoney((detailOrder as any).total, (detailOrder as any).currency) }}</strong></div>
                <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap">
                  <span class="subtle">Оплачено:</span>
                  <span style="color:#6dd4a0">{{ formatMoney((detailOrder as any).paid ?? 0, (detailOrder as any).currency) }}</span>
                  <button v-if="!showRepairPaymentForm" class="ghost-button" style="font-size:0.72rem;padding:0.1rem 0.4rem" @click="openRepairPaymentForm">+ Оплата</button>
                </div>
                <div><span class="subtle">Борг:</span>
                  <span :style="(detailOrder as any).debt > 0 ? 'color:var(--danger);font-weight:700' : ''">{{ formatMoney((detailOrder as any).debt ?? 0, (detailOrder as any).currency) }}</span>
                </div>
                <div><span class="subtle">Дата:</span> {{ formatDate((detailOrder as any).createdAt) }}</div>
              </div>
              <div v-if="(detailOrder as any).description" style="margin-bottom:0.75rem;font-size:0.88rem">
                <span class="subtle">Опис:</span> {{ (detailOrder as any).description }}
              </div>

              <!-- Repair payment form -->
              <div v-if="showRepairPaymentForm" style="background:rgba(122,185,154,0.06);border:1px solid rgba(122,185,154,0.2);border-radius:8px;padding:0.75rem;margin-bottom:1rem">
                <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.6rem">
                  <span style="font-size:0.82rem;font-weight:600;color:var(--accent)">💳 Додати оплату</span>
                  <button class="ghost-button" style="font-size:0.8rem;padding:0.1rem 0.4rem" @click="showRepairPaymentForm=false">✕</button>
                </div>
                <p v-if="repairPaymentError" style="color:var(--danger);font-size:0.8rem;margin:0 0 0.4rem">{{ repairPaymentError }}</p>
                <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                  <label style="font-size:0.78rem">Сума *
                    <input type="number" v-model.number="repairPaymentAmount" min="0" step="0.01" class="input" style="margin-top:0.2rem" placeholder="0.00">
                  </label>
                  <label style="font-size:0.78rem">Метод
                    <select v-model="repairPaymentMethod" class="input" style="margin-top:0.2rem">
                      <option value="cash">Готівка</option>
                      <option value="card">Картка</option>
                      <option value="bank">Банк</option>
                      <option value="virtual">Інше</option>
                    </select>
                  </label>
                </div>
                <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.6rem">
                  <label style="font-size:0.78rem">Каса
                    <select v-model="repairPaymentCashboxId" class="input" style="margin-top:0.2rem">
                      <option :value="null">— без каси —</option>
                      <option v-for="cb in allCashboxes" :key="cb.id" :value="cb.id">{{ cb.name }}</option>
                    </select>
                  </label>
                  <label style="font-size:0.78rem">Примітка
                    <input v-model="repairPaymentNote" class="input" style="margin-top:0.2rem" placeholder="Необов'язково">
                  </label>
                </div>
                <div style="display:flex;gap:0.5rem">
                  <button class="ghost-button" style="font-size:0.8rem" @click="showRepairPaymentForm=false" :disabled="repairPaymentSaving">Скасувати</button>
                  <button style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.35rem 0.9rem;font-size:0.82rem;cursor:pointer;font-weight:600" @click="saveRepairPayment" :disabled="repairPaymentSaving">
                    {{ repairPaymentSaving ? 'Збереження...' : '✓ Зберегти' }}
                  </button>
                </div>
              </div>

              <div v-if="(detailOrder as any).parts?.length" style="border-top:1px solid var(--border);padding-top:0.75rem;margin-bottom:1rem">
                <div style="font-size:0.82rem;font-weight:600;color:#e0a060;margin-bottom:0.5rem">🔧 Запчастини</div>
                <table style="font-size:0.84rem">
                  <thead><tr><th>Товар</th><th>К-сть</th><th>Ціна</th><th>Сума</th></tr></thead>
                  <tbody>
                    <tr v-for="part in (detailOrder as any).parts" :key="part.id">
                      <td>{{ getProductName(part.productId) }}</td>
                      <td>{{ part.quantity }}</td>
                      <td>{{ part.price }}</td>
                      <td>{{ part.total }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <div style="border-top:1px solid var(--border);padding-top:0.75rem">
                <div style="font-size:0.8rem;color:var(--text-muted);margin-bottom:0.35rem">Змінити статус:</div>
                <div style="display:flex;gap:0.4rem;flex-wrap:wrap;margin-bottom:0.75rem">
                  <button v-if="(detailOrder as any).status==='new'" class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem"
                    @click="updateRepairOrderStatus((detailOrder as any).id,'in_progress')">🔧 В роботі</button>
                  <button v-if="(detailOrder as any).status==='in_progress'"
                    style="font-size:0.8rem;padding:0.3rem 0.9rem;background:rgba(109,212,160,0.15);border:1px solid rgba(109,212,160,0.4);color:#6dd4a0;border-radius:6px;cursor:pointer"
                    @click="updateRepairOrderStatus((detailOrder as any).id,'done')">✅ Виконано</button>
                  <button v-if="!['done','cancelled'].includes((detailOrder as any).status)"
                    class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem;color:#ff9ca0"
                    @click="updateRepairOrderStatus((detailOrder as any).id,'cancelled')">✕ Скасувати</button>
                  <span v-if="['done','cancelled'].includes((detailOrder as any).status)" class="subtle" style="font-size:0.82rem;align-self:center">
                    Статус не можна змінити
                  </span>
                </div>
                <button class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.8rem" @click="startEditRepair">✏ Редагувати дані</button>
              </div>
            </template>

            <!-- Edit mode -->
            <template v-else>
              <p v-if="editRepairError" style="color:var(--danger);font-size:0.82rem;margin-bottom:0.5rem">{{ editRepairError }}</p>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.75rem;font-size:0.88rem">
                <label style="font-size:0.78rem;grid-column:1/-1">Назва *
                  <input v-model="editRepairForm.title" class="input" style="margin-top:0.2rem;width:100%" placeholder="Назва замовлення">
                </label>
                <label style="font-size:0.78rem;grid-column:1/-1">Опис
                  <textarea v-model="editRepairForm.description" class="input" rows="2" style="margin-top:0.2rem;width:100%;resize:vertical" placeholder="Опис проблеми"></textarea>
                </label>
                <label style="font-size:0.78rem">Технік
                  <input v-model="editRepairForm.technician" class="input" style="margin-top:0.2rem" placeholder="Ім'я техніка">
                </label>
                <label style="font-size:0.78rem">Тривалість (хв)
                  <input type="number" v-model.number="editRepairForm.laborMin" min="0" class="input" style="margin-top:0.2rem">
                </label>
                <label style="font-size:0.78rem">Вартість роботи
                  <input type="number" v-model.number="editRepairForm.price" min="0" step="0.01" class="input" style="margin-top:0.2rem">
                </label>
                <label style="font-size:0.78rem">Валюта
                  <select v-model="editRepairForm.currency" class="input" style="margin-top:0.2rem">
                    <option value="UAH">UAH</option>
                    <option value="USD">USD</option>
                    <option value="EUR">EUR</option>
                  </select>
                </label>
              </div>
              <div style="display:flex;gap:0.5rem">
                <button class="ghost-button" style="font-size:0.82rem" @click="editingRepair=false" :disabled="editRepairSaving">Скасувати</button>
                <button style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.4rem 1rem;font-size:0.82rem;cursor:pointer;font-weight:600"
                  @click="saveRepair" :disabled="editRepairSaving">
                  {{ editRepairSaving ? 'Збереження...' : '✓ Зберегти' }}
                </button>
              </div>
            </template>
          </template>

          <div style="display:flex;justify-content:flex-end;margin-top:1rem;padding-top:0.75rem;border-top:1px solid var(--border)">
            <button class="ghost-button" @click="showDetailModal = false">Закрити</button>
          </div>
        </div>
      </div>
    </teleport>

    <!-- ═══ МОДАЛЬНЕ ВІКНО СТВОРЕННЯ ЗАМОВЛЕННЯ ══════════════ -->
    <teleport to="body">
      <div v-if="showCreateModal" class="modal-backdrop" @click.self="showCreateModal = false">
        <div class="modal-box" style="max-width:560px;width:100%;">

          <!-- Header -->
          <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:1rem;border-bottom:1px solid var(--border);padding-bottom:0.75rem">
            <h3 style="margin:0;font-size:1rem">
              {{ activeSubTab === 'customer' ? '🛍 Нове замовлення клієнта'
                 : activeSubTab === 'supplier' ? '📋 Нове замовлення постачальника'
                 : '⚙ Нове замовлення на ремонт' }}
            </h3>
            <button class="ghost-button" style="font-size:1.2rem;padding:0 0.4rem;line-height:1" @click="showCreateModal = false">✕</button>
          </div>

          <p v-if="createError" class="error-text" style="margin-bottom:0.75rem">{{ createError }}</p>

          <!-- ══ CUSTOMER ORDER FORM ══════════════════════════ -->
          <template v-if="activeSubTab === 'customer'">

            <!-- Client search -->
            <div style="margin-bottom:0.85rem;position:relative">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Клієнт *</label>
              <div style="display:flex;gap:0.4rem">
                <div style="flex:1;position:relative">
                  <input
                    v-model="customerSearch"
                    class="input"
                    placeholder="Пошук за ім'ям, телефоном..."
                    @focus="showCustomerDropdown = true"
                    @input="showCustomerDropdown = true; customerForm.customerName = customerSearch"
                    @blur="hideDropdown(v => showCustomerDropdown = v)"
                    style="width:100%"
                  />
                  <div v-if="showCustomerDropdown && filteredCustomers.length > 0"
                    style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                    <div v-for="c in filteredCustomers" :key="c.id"
                      style="padding:0.5rem 0.75rem;cursor:pointer;border-bottom:1px solid rgba(122,185,154,0.08);font-size:0.87rem"
                      @mousedown.prevent="selectCustomer(c)"
                      @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(122,185,154,0.1)'"
                      @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                      <span style="font-weight:600">{{ c.name }}</span>
                      <span v-if="c.phone" class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ c.phone }}</span>
                    </div>
                  </div>
                </div>
                <button class="ghost-button" style="white-space:nowrap;font-size:0.8rem;padding:0.4rem 0.7rem"
                  @click="showNewCustomerForm = !showNewCustomerForm; showCustomerDropdown = false">
                  {{ showNewCustomerForm ? '✕' : '+ Новий' }}
                </button>
              </div>
              <div v-if="customerForm.customerName && !showCustomerDropdown" style="font-size:0.78rem;color:#6dd4a0;margin-top:0.2rem">
                ✓ Обрано: {{ customerForm.customerName }}
              </div>
            </div>

            <!-- Quick create customer -->
            <div v-if="showNewCustomerForm"
              style="background:rgba(122,185,154,0.06);border:1px solid rgba(122,185,154,0.2);border-radius:8px;padding:0.75rem;margin-bottom:0.85rem">
              <div style="font-size:0.8rem;font-weight:600;color:#7ab99a;margin-bottom:0.5rem">Новий клієнт</div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Ім'я *<input v-model="newCustomerForm.name" class="input" placeholder="Ім'я клієнта" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Телефон<input v-model="newCustomerForm.phone" class="input" placeholder="+380..." style="margin-top:0.2rem"></label>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Email<input v-model="newCustomerForm.email" class="input" placeholder="email@..." style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Коментар<input v-model="newCustomerForm.comment" class="input" style="margin-top:0.2rem"></label>
              </div>
              <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newCustomerSaving || !newCustomerForm.name.trim()"
                @click="saveNewCustomer(false)">
                {{ newCustomerSaving ? '...' : '✓ Зберегти клієнта' }}
              </button>
            </div>

            <!-- Currency & date -->
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.75rem;margin-bottom:0.85rem">
              <div>
                <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Валюта</label>
                <select v-model="customerForm.currency" class="input">
                  <option value="UAH">UAH</option><option value="USD">USD</option><option value="EUR">EUR</option>
                </select>
              </div>
              <div>
                <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Дата видачі</label>
                <input v-model="customerForm.dueDate" type="date" class="input">
              </div>
            </div>
            <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:1rem">
              <input id="reserve-check" v-model="customerForm.reserve" type="checkbox">
              <label for="reserve-check" style="font-size:0.85rem;cursor:pointer">Зарезервувати товар</label>
            </div>

            <!-- Order items -->
            <div style="border-top:1px solid var(--border);padding-top:0.85rem;margin-bottom:0.85rem">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.6rem">
                <span style="font-size:0.82rem;font-weight:600;color:#7ab99a">📦 Товари замовлення</span>
                <button class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.65rem" @click="addOrderItem">+ Додати позицію</button>
              </div>

              <div v-for="(item, idx) in orderItems" :key="idx"
                style="background:rgba(122,185,154,0.05);border:1px solid rgba(122,185,154,0.15);border-radius:8px;padding:0.6rem;margin-bottom:0.5rem">
                <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.4rem">
                  <span style="font-size:0.8rem;color:var(--text-muted)">Позиція {{ idx + 1 }}</span>
                  <button class="ghost-button" style="padding:0.15rem 0.45rem;font-size:0.8rem;color:var(--danger)" @click="removeOrderItem(idx)">✕</button>
                </div>

                <!-- Product picker -->
                <div style="position:relative;margin-bottom:0.4rem">
                  <div v-if="item.productId > 0" style="display:flex;align-items:center;gap:0.4rem;padding:0.4rem 0.6rem;background:rgba(122,185,154,0.1);border-radius:6px;font-size:0.85rem">
                    <span style="flex:1;font-weight:500">{{ getProductName(item.productId) }}</span>
                    <button class="ghost-button" style="padding:0.1rem 0.4rem;font-size:0.75rem"
                      @click="item.productId = 0; openProductSearch(idx)">змінити</button>
                  </div>
                  <div v-else>
                    <input
                      :value="activeProductItemIdx === idx ? productSearch : ''"
                      class="input"
                      placeholder="Пошук товару за назвою, SKU..."
                      style="width:100%"
                      @focus="openProductSearch(idx)"
                      @input="productSearch = ($event.target as HTMLInputElement).value; showProductDropdown = true"
                    />
                    <div v-if="showProductDropdown && activeProductItemIdx === idx"
                      style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                      <div v-if="filteredProducts.length === 0 && productSearch.trim()"
                        style="padding:0.5rem 0.75rem;font-size:0.84rem;color:var(--text-muted)">
                        Товар "{{ productSearch }}" не знайдено
                      </div>
                      <div v-for="p in filteredProducts" :key="p.id"
                        style="padding:0.45rem 0.75rem;cursor:pointer;border-bottom:1px solid rgba(122,185,154,0.08);font-size:0.84rem"
                        @mousedown.prevent="selectProduct(p, idx)"
                        @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(122,185,154,0.1)'"
                        @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                        <span style="font-weight:600">{{ p.name }}</span>
                        <span class="subtle" style="font-size:0.75rem;margin-left:0.4rem">{{ p.sku }}</span>
                        <span style="float:right;color:var(--accent);font-weight:600;font-size:0.82rem">{{ p.retailPrice }} {{ p.currency }}</span>
                      </div>
                      <div style="padding:0.4rem 0.75rem;border-top:1px solid var(--border)">
                        <button class="ghost-button" style="font-size:0.78rem;width:100%"
                          @mousedown.prevent="showNewProductForm = true; showProductDropdown = false; newProductForm.name = productSearch">
                          + Створити новий товар{{ productSearch.trim() ? ` "${productSearch}"` : '' }}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                <div style="display:grid;grid-template-columns:80px 100px;gap:0.5rem">
                  <label style="font-size:0.75rem;color:var(--text-muted)">К-сть
                    <input type="number" v-model.number="item.quantity" min="1" class="input" style="margin-top:0.15rem">
                  </label>
                  <label style="font-size:0.75rem;color:var(--text-muted)">Ціна
                    <input type="number" v-model.number="item.price" min="0" class="input" style="margin-top:0.15rem">
                  </label>
                </div>
              </div>

              <p v-if="orderItems.length === 0" class="subtle" style="font-size:0.82rem;margin:0">
                Товари не обов'язкові — можна додати пізніше
              </p>
            </div>

            <!-- Quick create product form -->
            <div v-if="showNewProductForm"
              style="background:rgba(96,192,224,0.06);border:1px solid rgba(96,192,224,0.2);border-radius:8px;padding:0.75rem;margin-bottom:0.85rem">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
                <span style="font-size:0.8rem;font-weight:600;color:#60c0e0">Новий товар</span>
                <button class="ghost-button" style="font-size:0.8rem" @click="showNewProductForm = false">✕</button>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Назва *<input v-model="newProductForm.name" class="input" placeholder="Назва товару" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">SKU<input v-model="newProductForm.sku" class="input" placeholder="Артикул" style="margin-top:0.2rem"></label>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Категорія<input v-model="newProductForm.category" class="input" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Ціна закупівлі<input type="number" v-model.number="newProductForm.purchasePrice" min="0" class="input" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Ціна продажу *<input type="number" v-model.number="newProductForm.retailPrice" min="0" class="input" style="margin-top:0.2rem"></label>
              </div>
              <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newProductSaving || !newProductForm.name.trim()"
                @click="saveNewProduct">
                {{ newProductSaving ? '...' : '✓ Зберегти товар' }}
              </button>
            </div>
          </template>

          <!-- ══ SUPPLIER ORDER FORM ══════════════════════════ -->
          <template v-else-if="activeSubTab === 'supplier'">

            <!-- Supplier search -->
            <div style="margin-bottom:0.85rem">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Постачальник *</label>
              <div style="display:flex;gap:0.4rem">
                <div style="flex:1;position:relative">
                  <input
                    v-model="supplierSearch"
                    class="input"
                    placeholder="Пошук за назвою, контактом..."
                    @focus="showSupplierDropdown = true"
                    @input="showSupplierDropdown = true"
                    @blur="hideDropdown(v => showSupplierDropdown = v)"
                    style="width:100%"
                  />
                  <div v-if="showSupplierDropdown && filteredSuppliers.length > 0"
                    style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                    <div v-for="s in filteredSuppliers" :key="s.id"
                      style="padding:0.5rem 0.75rem;cursor:pointer;border-bottom:1px solid rgba(96,192,224,0.08);font-size:0.87rem"
                      @mousedown.prevent="selectSupplier(s)"
                      @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(96,192,224,0.1)'"
                      @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                      <span style="font-weight:600">{{ s.name }}</span>
                      <span v-if="s.phone" class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ s.phone }}</span>
                    </div>
                  </div>
                </div>
                <button class="ghost-button" style="white-space:nowrap;font-size:0.8rem;padding:0.4rem 0.7rem"
                  @click="showNewSupplierForm = !showNewSupplierForm; showSupplierDropdown = false">
                  {{ showNewSupplierForm ? '✕' : '+ Новий' }}
                </button>
              </div>
              <div v-if="supplierForm.supplierId" style="font-size:0.78rem;color:#60c0e0;margin-top:0.2rem">
                ✓ Обрано: {{ allSuppliers.find(s => s.id === supplierForm.supplierId)?.name }}
              </div>
            </div>

            <!-- Quick create supplier -->
            <div v-if="showNewSupplierForm"
              style="background:rgba(96,192,224,0.06);border:1px solid rgba(96,192,224,0.2);border-radius:8px;padding:0.75rem;margin-bottom:0.85rem">
              <div style="font-size:0.8rem;font-weight:600;color:#60c0e0;margin-bottom:0.5rem">Новий постачальник</div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Назва *<input v-model="newSupplierForm.name" class="input" placeholder="Назва компанії" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Контактна особа<input v-model="newSupplierForm.contact" class="input" style="margin-top:0.2rem"></label>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Телефон<input v-model="newSupplierForm.phone" class="input" placeholder="+380..." style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Email<input v-model="newSupplierForm.email" class="input" style="margin-top:0.2rem"></label>
              </div>
              <label style="font-size:0.78rem;display:block;margin-bottom:0.5rem">Коментар<input v-model="newSupplierForm.comments" class="input" style="margin-top:0.2rem;width:100%"></label>
              <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newSupplierSaving || !newSupplierForm.name.trim()"
                @click="saveNewSupplier">
                {{ newSupplierSaving ? '...' : '✓ Зберегти постачальника' }}
              </button>
            </div>

            <div style="margin-bottom:0.85rem">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Валюта</label>
              <select v-model="supplierForm.currency" class="input">
                <option value="UAH">UAH</option><option value="USD">USD</option><option value="EUR">EUR</option>
              </select>
            </div>

            <!-- Supplier order items -->
            <div style="border-top:1px solid var(--border);padding-top:0.85rem;margin-bottom:0.85rem">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.6rem">
                <span style="font-size:0.82rem;font-weight:600;color:#60c0e0">📦 Товари замовлення</span>
                <button class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.65rem" @click="addOrderItem">+ Додати позицію</button>
              </div>
              <div v-for="(item, idx) in orderItems" :key="idx"
                style="background:rgba(96,192,224,0.05);border:1px solid rgba(96,192,224,0.15);border-radius:8px;padding:0.6rem;margin-bottom:0.5rem">
                <div style="display:flex;justify-content:space-between;margin-bottom:0.4rem">
                  <span style="font-size:0.8rem;color:var(--text-muted)">Позиція {{ idx + 1 }}</span>
                  <button class="ghost-button" style="padding:0.15rem 0.45rem;font-size:0.8rem;color:var(--danger)" @click="removeOrderItem(idx)">✕</button>
                </div>
                <div style="position:relative;margin-bottom:0.4rem">
                  <div v-if="item.productId > 0" style="display:flex;align-items:center;gap:0.4rem;padding:0.4rem 0.6rem;background:rgba(96,192,224,0.1);border-radius:6px;font-size:0.85rem">
                    <span style="flex:1;font-weight:500">{{ getProductName(item.productId) }}</span>
                    <button class="ghost-button" style="padding:0.1rem 0.4rem;font-size:0.75rem" @click="item.productId = 0; openProductSearch(idx)">змінити</button>
                  </div>
                  <div v-else>
                    <input
                      :value="activeProductItemIdx === idx ? productSearch : ''"
                      class="input" placeholder="Пошук товару..." style="width:100%"
                      @focus="openProductSearch(idx)"
                      @input="productSearch = ($event.target as HTMLInputElement).value; showProductDropdown = true"
                    />
                    <div v-if="showProductDropdown && activeProductItemIdx === idx"
                      style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                      <div v-if="filteredProducts.length === 0 && productSearch.trim()"
                        style="padding:0.5rem 0.75rem;font-size:0.84rem;color:var(--text-muted)">
                        Товар "{{ productSearch }}" не знайдено
                      </div>
                      <div v-for="p in filteredProducts" :key="p.id"
                        style="padding:0.45rem 0.75rem;cursor:pointer;border-bottom:1px solid rgba(96,192,224,0.08);font-size:0.84rem"
                        @mousedown.prevent="selectProduct(p, idx)"
                        @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(96,192,224,0.1)'"
                        @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                        <span style="font-weight:600">{{ p.name }}</span>
                        <span class="subtle" style="font-size:0.75rem;margin-left:0.4rem">{{ p.sku }}</span>
                        <span style="float:right;color:#60c0e0;font-weight:600;font-size:0.82rem">{{ p.purchasePrice }} {{ p.currency }}</span>
                      </div>
                      <div style="padding:0.4rem 0.75rem;border-top:1px solid var(--border)">
                        <button class="ghost-button" style="font-size:0.78rem;width:100%"
                          @mousedown.prevent="showNewProductForm = true; showProductDropdown = false; newProductForm.name = productSearch">
                          + Створити новий товар{{ productSearch.trim() ? ` "${productSearch}"` : '' }}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
                <div style="display:grid;grid-template-columns:80px 110px;gap:0.5rem">
                  <label style="font-size:0.75rem;color:var(--text-muted)">К-сть<input type="number" v-model.number="item.quantity" min="1" class="input" style="margin-top:0.15rem"></label>
                  <label style="font-size:0.75rem;color:var(--text-muted)">Ціна закуп.<input type="number" v-model.number="item.price" min="0" class="input" style="margin-top:0.15rem"></label>
                </div>
              </div>
              <p v-if="orderItems.length === 0" class="subtle" style="font-size:0.82rem;margin:0">Товари не обов'язкові — можна додати пізніше</p>
            </div>

            <!-- Quick create product form (supplier order) -->
            <div v-if="showNewProductForm"
              style="background:rgba(96,192,224,0.06);border:1px solid rgba(96,192,224,0.2);border-radius:8px;padding:0.75rem;margin-bottom:0.85rem">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
                <span style="font-size:0.8rem;font-weight:600;color:#60c0e0">Новий товар</span>
                <button class="ghost-button" style="font-size:0.8rem" @click="showNewProductForm = false">✕</button>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Назва *<input v-model="newProductForm.name" class="input" placeholder="Назва товару" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">SKU<input v-model="newProductForm.sku" class="input" placeholder="Артикул" style="margin-top:0.2rem"></label>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Категорія<input v-model="newProductForm.category" class="input" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Ціна закупівлі<input type="number" v-model.number="newProductForm.purchasePrice" min="0" class="input" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Ціна продажу *<input type="number" v-model.number="newProductForm.retailPrice" min="0" class="input" style="margin-top:0.2rem"></label>
              </div>
              <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newProductSaving || !newProductForm.name.trim()"
                @click="saveNewProduct">
                {{ newProductSaving ? '...' : '✓ Зберегти товар' }}
              </button>
            </div>
          </template>

          <!-- ══ REPAIR ORDER FORM ════════════════════════════ -->
          <template v-else-if="activeSubTab === 'repair'">

            <!-- Client search for repair -->
            <div style="margin-bottom:0.85rem">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Клієнт *</label>
              <div style="display:flex;gap:0.4rem">
                <div style="flex:1;position:relative">
                  <input
                    v-model="repairCustomerSearch"
                    class="input"
                    placeholder="Пошук за ім'ям, телефоном..."
                    @focus="showRepairCustomerDropdown = true"
                    @input="showRepairCustomerDropdown = true"
                    @blur="hideDropdown(v => showRepairCustomerDropdown = v)"
                    style="width:100%"
                  />
                  <div v-if="showRepairCustomerDropdown && filteredRepairCustomers.length > 0"
                    style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                    <div v-for="c in filteredRepairCustomers" :key="c.id"
                      style="padding:0.5rem 0.75rem;cursor:pointer;border-bottom:1px solid rgba(224,160,96,0.08);font-size:0.87rem"
                      @mousedown.prevent="selectRepairCustomer(c)"
                      @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(224,160,96,0.1)'"
                      @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                      <span style="font-weight:600">{{ c.name }}</span>
                      <span v-if="c.phone" class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ c.phone }}</span>
                    </div>
                  </div>
                </div>
                <button class="ghost-button" style="white-space:nowrap;font-size:0.8rem;padding:0.4rem 0.7rem"
                  @click="showNewCustomerForm = !showNewCustomerForm; showRepairCustomerDropdown = false">
                  {{ showNewCustomerForm ? '✕' : '+ Новий' }}
                </button>
              </div>
              <div v-if="repairForm.customerId" style="font-size:0.78rem;color:#e0a060;margin-top:0.2rem">
                ✓ Обрано: {{ allCustomers.find(c => c.id === repairForm.customerId)?.name }}
              </div>
            </div>

            <!-- Quick create customer for repair -->
            <div v-if="showNewCustomerForm"
              style="background:rgba(224,160,96,0.06);border:1px solid rgba(224,160,96,0.2);border-radius:8px;padding:0.75rem;margin-bottom:0.85rem">
              <div style="font-size:0.8rem;font-weight:600;color:#e0a060;margin-bottom:0.5rem">Новий клієнт</div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Ім'я *<input v-model="newCustomerForm.name" class="input" placeholder="Ім'я клієнта" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Телефон<input v-model="newCustomerForm.phone" class="input" placeholder="+380..." style="margin-top:0.2rem"></label>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                <label style="font-size:0.78rem">Email<input v-model="newCustomerForm.email" class="input" style="margin-top:0.2rem"></label>
                <label style="font-size:0.78rem">Коментар<input v-model="newCustomerForm.comment" class="input" style="margin-top:0.2rem"></label>
              </div>
              <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newCustomerSaving || !newCustomerForm.name.trim()"
                @click="saveNewCustomer(true)">
                {{ newCustomerSaving ? '...' : '✓ Зберегти клієнта' }}
              </button>
            </div>

            <!-- Repair fields -->
            <div style="margin-bottom:0.75rem">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Назва замовлення *</label>
              <input v-model="repairForm.title" class="input" placeholder="Наприклад: Ремонт телефону" style="width:100%">
            </div>
            <div style="margin-bottom:0.75rem">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Опис</label>
              <textarea v-model="repairForm.description" class="input" rows="2" placeholder="Опис несправності або завдання" style="resize:vertical;height:auto;width:100%" />
            </div>
            <div style="margin-bottom:0.75rem">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Технік</label>
              <input v-model="repairForm.technician" class="input" placeholder="Ім'я техніка (необов'язково)" style="width:100%">
            </div>

            <!-- Пристрій (товар) з пошуком -->
            <div style="margin-bottom:0.85rem">
              <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Пристрій (товар)</label>
              <div v-if="repairForm.productId"
                style="display:flex;align-items:center;gap:0.4rem;padding:0.4rem 0.6rem;background:rgba(224,160,96,0.1);border-radius:6px;font-size:0.85rem;margin-bottom:0.3rem">
                <span style="flex:1;font-weight:500">✓ {{ allProducts.find(p => p.id === repairForm.productId)?.name }}</span>
                <button class="ghost-button" style="padding:0.1rem 0.4rem;font-size:0.75rem"
                  @click="repairForm.productId = undefined; repairProductSearch = ''">✕ змінити</button>
              </div>
              <div v-else style="position:relative">
                <input
                  v-model="repairProductSearch"
                  class="input"
                  placeholder="Пошук товару за назвою, SKU..."
                  style="width:100%"
                  @focus="showRepairProductDropdown = true"
                  @input="showRepairProductDropdown = true"
                />
                <div v-if="showRepairProductDropdown && filteredRepairProducts.length > 0"
                  style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                  <div v-for="p in filteredRepairProducts" :key="p.id"
                    style="padding:0.45rem 0.75rem;cursor:pointer;border-bottom:1px solid rgba(224,160,96,0.08);font-size:0.84rem"
                    @mousedown.prevent="selectRepairProduct(p)"
                    @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(224,160,96,0.1)'"
                    @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                    <span style="font-weight:600">{{ p.name }}</span>
                    <span class="subtle" style="font-size:0.75rem;margin-left:0.4rem">{{ p.sku }}</span>
                  </div>
                  <div style="padding:0.4rem 0.75rem;border-top:1px solid var(--border)">
                    <button class="ghost-button" style="font-size:0.78rem;width:100%"
                      @mousedown.prevent="showNewRepairProductForm = true; showRepairProductDropdown = false">
                      + Створити новий товар
                    </button>
                  </div>
                </div>
                <div v-if="repairProductSearch && !showRepairProductDropdown && filteredRepairProducts.length === 0"
                  style="font-size:0.78rem;color:var(--text-muted);margin-top:0.3rem">
                  Товар не знайдено —
                  <button class="ghost-button" style="font-size:0.78rem;padding:0.1rem 0.4rem;display:inline"
                    @click="showNewRepairProductForm = true">+ створити новий</button>
                </div>
              </div>

              <!-- Quick create product for repair -->
              <div v-if="showNewRepairProductForm"
                style="background:rgba(224,160,96,0.06);border:1px solid rgba(224,160,96,0.2);border-radius:8px;padding:0.75rem;margin-top:0.5rem">
                <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
                  <span style="font-size:0.8rem;font-weight:600;color:#e0a060">Новий товар</span>
                  <button class="ghost-button" style="font-size:0.8rem" @click="showNewRepairProductForm = false">✕</button>
                </div>
                <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                  <label style="font-size:0.78rem">Назва *<input v-model="newRepairProductForm.name" class="input" placeholder="Назва товару" style="margin-top:0.2rem"></label>
                  <label style="font-size:0.78rem">Категорія<input v-model="newRepairProductForm.category" class="input" style="margin-top:0.2rem"></label>
                </div>
                <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
                  <label style="font-size:0.78rem">Бренд<input v-model="newRepairProductForm.brand" class="input" style="margin-top:0.2rem"></label>
                  <label style="font-size:0.78rem">Роздрібна ціна<input type="number" v-model.number="newRepairProductForm.retailPrice" min="0" class="input" style="margin-top:0.2rem"></label>
                  <label style="font-size:0.78rem">Валюта
                    <select v-model="newRepairProductForm.currency" class="input" style="margin-top:0.2rem">
                      <option value="UAH">UAH</option><option value="USD">USD</option><option value="EUR">EUR</option>
                    </select>
                  </label>
                </div>
                <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newRepairProductSaving || !newRepairProductForm.name.trim()"
                  @click="saveNewRepairProduct">
                  {{ newRepairProductSaving ? '...' : '✓ Зберегти товар' }}
                </button>
              </div>
            </div>

            <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:0.75rem;margin-bottom:0.75rem">
              <div>
                <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Час роботи (хв)</label>
                <input v-model.number="repairForm.laborMin" type="number" min="0" class="input">
              </div>
              <div>
                <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Вартість роботи</label>
                <input v-model.number="repairForm.price" type="number" min="0" step="0.01" class="input">
              </div>
              <div>
                <label style="display:block;font-size:0.8rem;color:var(--text-muted);margin-bottom:0.3rem">Валюта</label>
                <select v-model="repairForm.currency" class="input">
                  <option value="UAH">UAH</option><option value="USD">USD</option><option value="EUR">EUR</option>
                </select>
              </div>
            </div>
          </template>

          <!-- Footer -->
          <div style="display:flex;justify-content:flex-end;gap:0.5rem;margin-top:1.25rem;padding-top:0.75rem;border-top:1px solid var(--border)">
            <button class="ghost-button" @click="showCreateModal = false">Скасувати</button>
            <button
              style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.45rem 1.1rem;font-size:0.85rem;cursor:pointer;font-weight:600"
              @click="submitCreateOrder"
              :disabled="createLoading">
              {{ createLoading ? "Збереження..." : "Створити" }}
            </button>
          </div>
        </div>
      </div>
    </teleport>
  </main>
</template>
