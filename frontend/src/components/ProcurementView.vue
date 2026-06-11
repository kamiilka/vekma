<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import type { UserSession, Supplier, SupplierOrder, Purchase, PurchaseRecommendationGroup, CurrencyCode, PurchaseItem, Product, Payment, PaymentMethod, Cashbox, Warehouse, WarehouseStock } from "../types";

const props = defineProps<{ session: UserSession; initialSubTab?: string; initialOrderId?: number }>();
const emit = defineEmits<{
  (e: "session-expired"): void;
  (e: "navigate", payload: { type: string; id: number } | string): void;
}>();

const token = computed(() => props.session.token);

const suppliers = ref<Supplier[]>([]);
const supplierOrders = ref<SupplierOrder[]>([]);
const purchases = ref<Purchase[]>([]);
const recommendations = ref<PurchaseRecommendationGroup[]>([]);
const products = ref<Product[]>([]);
const loading = ref(false);
const error = ref("");
const saving = ref(false);
const subTab = ref<"orders" | "recommendations" | "suppliers">((props.initialSubTab as any) ?? "recommendations");

// Forms
const showNewSupplier = ref(false);
const suppForm = ref({ name: "", contact: "", phone: "", email: "", comments: "" });

const showNewOrder = ref(false);
const orderForm = ref({ supplierId: 0, currency: "UAH" as CurrencyCode, items: [] as (PurchaseItem & { _name?: string })[] });
const supplierNameInput = ref("");

// Edit order
const showEditOrder = ref(false);
const editOrderRef = ref<SupplierOrder | null>(null);
const editOrderForm = ref({ supplierId: 0, currency: "UAH" as CurrencyCode, items: [] as (PurchaseItem & { _name?: string })[] });
const editSupplierNameInput = ref("");
const editOrderLineSearch = ref("");
const editOrderLineSelectedProduct = ref<Product | null>(null);
const editOrderLineQty = ref(1);
const editOrderLinePrice = ref(0);
const editOrderLineShowDropdown = ref(false);
const showEditSupplierDropdown = ref(false);

// Supplier dropdown search
const showSupplierDropdown = ref(false);
const showNewSupplierInlineForm = ref(false);
const newInlineSupplierForm = ref({ name: "", phone: "", contact: "" });
const newInlineSupplierSaving = ref(false);

const filteredSupplierSearch = computed(() => {
  const q = supplierNameInput.value.trim().toLowerCase();
  if (!q) return suppliers.value.slice(0, 8);
  return suppliers.value.filter(s =>
    s.name.toLowerCase().includes(q) || (s.phone || "").includes(q)
  ).slice(0, 10);
});

function selectSupplier(s: Supplier) {
  orderForm.value.supplierId = s.id;
  supplierNameInput.value = s.name;
  showSupplierDropdown.value = false;
}

async function saveNewInlineSupplier() {
  if (!newInlineSupplierForm.value.name.trim()) return;
  newInlineSupplierSaving.value = true;
  try {
    const created = await api.createSupplier(token.value, {
      name: newInlineSupplierForm.value.name.trim(),
      contact: newInlineSupplierForm.value.contact,
      phone: newInlineSupplierForm.value.phone,
    }) as any;
    suppliers.value = await api.suppliers(token.value);
    orderForm.value.supplierId = created.id;
    supplierNameInput.value = created.name;
    showSupplierDropdown.value = false;
    showNewSupplierInlineForm.value = false;
    newInlineSupplierForm.value = { name: "", phone: "", contact: "" };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { newInlineSupplierSaving.value = false; }
}

// Order line — вибір або створення номенклатури
const orderLineSearch = ref("");
const orderLineSelectedProduct = ref<Product | null>(null);
const orderLineQty = ref(1);
const orderLinePrice = ref(0);
const orderLineShowDropdown = ref(false);
const showNewNomenclature = ref(false);
const newNomForm = ref({ name: "", sku: "", barcode: "", category: "", brand: "", purchasePrice: 0, retailPrice: 0 });
const savingNom = ref(false);
const nomError = ref("");

const filteredProducts = computed(() => {
  const q = orderLineSearch.value.toLowerCase().trim();
  if (!q) return products.value.slice(0, 20);
  return products.value.filter(p =>
    p.name.toLowerCase().includes(q) ||
    p.sku.toLowerCase().includes(q) ||
    (p.barcode && p.barcode.toLowerCase().includes(q)) ||
    (p.article && p.article.toLowerCase().includes(q))
  ).slice(0, 20);
});

const showReceive = ref<SupplierOrder | null>(null);
const receiveLines = ref<Array<{ productId: number; quantity: number; price: number }>>([]);
const receiveNote = ref("");

const showNewPurchase = ref(false);
const purchaseForm = ref({ supplierId: 0, supplierOrderId: undefined as number | undefined, currency: "UAH" as CurrencyCode, note: "", items: [] as PurchaseItem[] });

// Bulk recommendation ordering
const selectedGroups = ref<Set<number>>(new Set());

async function load() {
  loading.value = true; error.value = "";
  try {
    [suppliers.value, supplierOrders.value, purchases.value, products.value, warehouses.value, warehouseStocks.value] = await Promise.all([
      api.suppliers(token.value),
      api.supplierOrders(token.value),
      api.purchases(token.value),
      api.products(token.value),
      api.warehouses(token.value),
      api.warehouseStocks(token.value),
    ]);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

async function loadRecommendations() {
  try {
    recommendations.value = await api.supplierOrderRecommendationsGrouped(token.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

async function addSupplier() {
  saving.value = true; error.value = "";
  try {
    await api.createSupplier(token.value, suppForm.value);
    showNewSupplier.value = false;
    suppForm.value = { name: "", contact: "", phone: "", email: "", comments: "" };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

function hideDropdownDelayed() {
  setTimeout(() => { orderLineShowDropdown.value = false; }, 150);
}

// ── Edit order helpers ────────────────────────────────────────
const filteredEditSupplierSearch = computed(() => {
  const q = editSupplierNameInput.value.trim().toLowerCase();
  if (!q) return suppliers.value.slice(0, 8);
  return suppliers.value.filter(s =>
    s.name.toLowerCase().includes(q) || (s.phone || "").includes(q)
  ).slice(0, 10);
});

const filteredEditProducts = computed(() => {
  const q = editOrderLineSearch.value.toLowerCase().trim();
  if (!q) return products.value.slice(0, 20);
  return products.value.filter(p =>
    p.name.toLowerCase().includes(q) ||
    p.sku.toLowerCase().includes(q) ||
    (p.barcode && p.barcode.toLowerCase().includes(q)) ||
    (p.article && p.article.toLowerCase().includes(q))
  ).slice(0, 20);
});

function hideEditDropdownDelayed() {
  setTimeout(() => { editOrderLineShowDropdown.value = false; }, 150);
}

function selectEditSupplier(s: Supplier) {
  editOrderForm.value.supplierId = s.id;
  editSupplierNameInput.value = s.name;
  showEditSupplierDropdown.value = false;
}

function selectEditOrderProduct(p: Product) {
  editOrderLineSelectedProduct.value = p;
  editOrderLineSearch.value = p.name;
  editOrderLinePrice.value = p.purchasePrice || 0;
  editOrderLineShowDropdown.value = false;
}

function addEditOrderLine() {
  if (!editOrderLineSelectedProduct.value) return;
  const p = editOrderLineSelectedProduct.value;
  editOrderForm.value.items.push({ productId: p.id, quantity: editOrderLineQty.value, price: editOrderLinePrice.value, _name: p.name });
  editOrderLineSelectedProduct.value = null;
  editOrderLineSearch.value = "";
  editOrderLineQty.value = 1;
  editOrderLinePrice.value = 0;
}

function openEditOrder(order: SupplierOrder) {
  editOrderRef.value = order;
  const supplier = supplierById(order.supplierId);
  editSupplierNameInput.value = supplier?.name ?? "";
  editOrderForm.value = {
    supplierId: order.supplierId,
    currency: order.currency as CurrencyCode,
    items: order.items.map(i => ({
      productId: i.productId,
      quantity: i.quantity,
      price: i.price,
      _name: productById(i.productId)?.name ?? `Товар #${i.productId}`,
    })),
  };
  editOrderLineSelectedProduct.value = null;
  editOrderLineSearch.value = "";
  showEditOrder.value = true;
}

async function saveEditOrder() {
  if (!editOrderRef.value) return;
  if (!editOrderForm.value.supplierId) { error.value = "Оберіть постачальника"; return; }
  if (!editOrderForm.value.items.length) { error.value = "Додайте хоча б одну позицію"; return; }
  saving.value = true; error.value = "";
  try {
    const orderId = editOrderRef.value.id;
    // Update supplier if changed
    if (editOrderForm.value.supplierId !== editOrderRef.value.supplierId) {
      await api.updateSupplierOrder(token.value, orderId, { supplierId: editOrderForm.value.supplierId });
    }
    // Always update items (covers currency + items + totals)
    await api.updateSupplierOrderItems(token.value, orderId, {
      items: editOrderForm.value.items.map(({ productId, quantity, price }) => ({ productId, quantity, price })),
      currency: editOrderForm.value.currency,
    });
    showEditOrder.value = false;
    await load();
    // Refresh detail panel if this order is open
    if (selectedOrder.value?.id === orderId) {
      const updated = supplierOrders.value.find(o => o.id === orderId);
      if (updated) await openOrderDetail(updated);
    }
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

function selectOrderProduct(p: Product) {
  orderLineSelectedProduct.value = p;
  orderLineSearch.value = p.name;
  orderLinePrice.value = p.purchasePrice || 0;
  orderLineShowDropdown.value = false;
}

function addOrderLine() {
  if (!orderLineSelectedProduct.value) return;
  const p = orderLineSelectedProduct.value;
  orderForm.value.items.push({ productId: p.id, quantity: orderLineQty.value, price: orderLinePrice.value, _name: p.name });
  orderLineSelectedProduct.value = null;
  orderLineSearch.value = "";
  orderLineQty.value = 1;
  orderLinePrice.value = 0;
}

async function saveNewNomenclature() {
  nomError.value = "";
  savingNom.value = true;
  try {
    const created = await api.createProduct(token.value, {
      name: newNomForm.value.name,
      sku: newNomForm.value.sku,
      barcode: newNomForm.value.barcode,
      category: newNomForm.value.category,
      brand: newNomForm.value.brand,
      purchasePrice: newNomForm.value.purchasePrice,
      retailPrice: newNomForm.value.retailPrice,
    });
    products.value.push(created);
    selectOrderProduct(created);
    showNewNomenclature.value = false;
    nomError.value = "";
    newNomForm.value = { name: "", sku: "", barcode: "", category: "", brand: "", purchasePrice: 0, retailPrice: 0 };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    nomError.value = e.message;
  } finally { savingNom.value = false; }
}

async function createOrder() {
  saving.value = true; error.value = "";
  try {
    if (!orderForm.value.supplierId) { error.value = "Оберіть постачальника зі списку або створіть нового"; saving.value = false; return; }
    await api.createSupplierOrder(token.value, { ...orderForm.value });
    showNewOrder.value = false;
    supplierNameInput.value = "";
    orderForm.value = { supplierId: 0, currency: "UAH", items: [] };
    orderLineSelectedProduct.value = null;
    orderLineSearch.value = "";
    showNewSupplierInlineForm.value = false;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function updateOrderStatus(id: number, status: any) {
  try {
    await api.updateSupplierOrderStatus(token.value, id, status);
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
    else error.value = e.message;
  }
}

function openReceive(order: SupplierOrder) {
  showReceive.value = order;
  receiveLines.value = order.items.map(i => ({ productId: i.productId, quantity: i.quantity, price: i.price }));
  receiveNote.value = "";
}

async function doReceive() {
  if (!showReceive.value) return;
  saving.value = true; error.value = "";
  try {
    await api.receiveSupplierOrderByLines(token.value, showReceive.value.id, { lines: receiveLines.value, note: receiveNote.value });
    showReceive.value = null;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function orderFromGroup(group: PurchaseRecommendationGroup) {
  if (!group.supplierId) { alert("У цієї групи немає постачальника"); return; }
  saving.value = true; error.value = "";
  try {
    await api.createSupplierOrderFromRecommendations(token.value, {
      supplierId: group.supplierId,
      items: group.items.map(i => ({ productId: i.productId, quantity: i.recommendedQty }))
    });
    alert(`Замовлення для ${group.supplierName} створено`);
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function bulkOrderFromRecommendations() {
  const groups = recommendations.value.filter(g => g.supplierId && selectedGroups.value.has(g.supplierId!));
  if (!groups.length) return;
  saving.value = true; error.value = "";
  try {
    await api.createSupplierOrdersBulkFromRecommendations(token.value, {
      orders: groups.map(g => ({
        supplierId: g.supplierId!,
        items: g.items.map(i => ({ productId: i.productId, quantity: i.recommendedQty }))
      }))
    });
    selectedGroups.value.clear();
    alert("Замовлення створено для всіх обраних постачальників");
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

function buildTelegramMsg(group: PurchaseRecommendationGroup): string {
  const lines = [`Замовлення від магазину:\n`];
  for (const item of group.items) {
    lines.push(`• ${item.productName} (${item.sku}) — ${item.recommendedQty} шт.`);
  }
  return lines.join('\n');
}

function supplierName(id: number) {
  return suppliers.value.find(s => s.id === id)?.name ?? `#${id}`;
}

const statusLabel: Record<string, string> = {
  draft: "Створено", sent: "Відправлено", confirmed: "Підтверджено",
  in_transit: "В дорозі", received: "Отримано", closed: "Закрито", cancelled: "Скасовано",
  // legacy
  partially_received: "В дорозі",
};

const statusColor: Record<string, string> = {
  draft: "#a0b4c8", sent: "#fad07a", confirmed: "#9fe8c4",
  in_transit: "#b09adc", received: "#6dd4a0", closed: "#6a9e84", cancelled: "#ff9ca0",
  partially_received: "#b09adc",
};

// ── Order detail panel ────────────────────────────────────────
const selectedOrder = ref<SupplierOrder | null>(null);
const detailPurchases = ref<Purchase[]>([]);
const detailPayments = ref<Payment[]>([]);

// Payment form state
const showPaymentForm = ref(false);
const paymentAmount = ref(0);
const paymentMethod = ref<PaymentMethod>("cash");
const paymentNote = ref("");
const paymentError = ref("");
const paymentSaving = ref(false);
const paymentCashboxId = ref<number | null>(null);
const allCashboxes = ref<Cashbox[]>([]);

async function openPaymentForm() {
  paymentAmount.value = 0;
  paymentMethod.value = "cash";
  paymentNote.value = "";
  paymentError.value = "";
  paymentCashboxId.value = null;
  showPaymentForm.value = true;
  if (allCashboxes.value.length === 0) {
    try { allCashboxes.value = await api.cashboxes(token.value); } catch {}
  }
}

async function saveSupplierPayment() {
  const order = selectedOrder.value;
  if (!order) return;
  if (!paymentAmount.value || paymentAmount.value <= 0) {
    paymentError.value = "Введіть суму оплати";
    return;
  }
  paymentSaving.value = true;
  paymentError.value = "";
  try {
    await api.createPayment(token.value, {
      supplierOrderId: order.id,
      amount: paymentAmount.value,
      currency: order.currency as CurrencyCode,
      method: paymentMethod.value,
      cashboxId: paymentCashboxId.value ?? undefined,
      note: paymentNote.value,
    });
    detailPayments.value = await api.payments(token.value, { supplierOrderId: order.id });
    supplierOrders.value = await api.supplierOrders(token.value);
    const updated = supplierOrders.value.find(o => o.id === order.id);
    if (updated) selectedOrder.value = updated;
    showPaymentForm.value = false;
  } catch (e: any) {
    if (e?.message === "SESSION_EXPIRED") emit("session-expired");
    else paymentError.value = e?.message ?? "Помилка збереження";
  } finally {
    paymentSaving.value = false;
  }
}
const detailLoading = ref(false);

function supplierById(id: number): Supplier | undefined {
  return suppliers.value.find(s => s.id === id);
}

function productById(id: number): Product | undefined {
  return products.value.find(p => p.id === id);
}

const detailPaid = computed(() =>
  detailPayments.value.reduce((sum, p) => sum + p.amountUah, 0)
);
const detailDebt = computed(() =>
  Math.max(0, (selectedOrder.value?.totalUah ?? 0) - detailPaid.value)
);
const detailReceived = computed(() =>
  detailPurchases.value.length > 0
);

async function openOrderDetail(order: SupplierOrder) {
  selectedOrder.value = order;
  detailPurchases.value = [];
  detailPayments.value = [];
  showPaymentForm.value = false;
  detailLoading.value = true;
  try {
    [detailPurchases.value, detailPayments.value] = await Promise.all([
      api.purchases(token.value, order.id),
      api.payments(token.value, { supplierOrderId: order.id }),
    ]);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    detailLoading.value = false;
  }
}

const methodLabel: Record<string, string> = { cash: "Готівка", card: "Картка", bank: "Банк", virtual: "Віртуальна" };

// ── Warehouses + stock for movement modal ─────────────────────
const warehouses = ref<Warehouse[]>([]);
const warehouseStocks = ref<WarehouseStock[]>([]);

function stockForProduct(productId: number): number {
  return warehouseStocks.value
    .filter(s => s.productId === productId)
    .reduce((sum, s) => sum + s.quantity, 0);
}

// ── Вбудована модалка "Рух товару" ───────────────────────────
const showMovement = ref(false);
const movLines = ref<Array<{ productId: number; productName: string; warehouseId: number; quantity: number; note: string }>>([]);
const movOrderRef = ref<SupplierOrder | null>(null);

function openMovementFromOrder(order: SupplierOrder) {
  movOrderRef.value = order;
  movLines.value = order.items.map(item => ({
    productId: item.productId,
    productName: productById(item.productId)?.name ?? `Товар #${item.productId}`,
    warehouseId: 0,
    quantity: item.quantity,
    note: `Надходження по замовленню #${order.id}`,
  }));
  showMovement.value = true;
}

async function doMovementFromOrder() {
  if (!movLines.value.length) return;
  saving.value = true; error.value = "";
  try {
    for (const line of movLines.value) {
      if (line.quantity > 0) {
        await api.createMovement(token.value, {
          productId: line.productId,
          warehouseId: line.warehouseId,
          type: "receipt",
          quantity: line.quantity,
          note: line.note,
        });
      }
    }
    showMovement.value = false;
    // Reload stocks
    warehouseStocks.value = await api.warehouseStocks(token.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

// ── Модалка "Повернення постачальнику" ────────────────────────
const showReturn = ref(false);
const returnOrderRef = ref<SupplierOrder | null>(null);
const returnLines = ref<Array<{ productId: number; productName: string; warehouseId: number; quantity: number; note: string; include: boolean }>>([]);
const returnGlobalNote = ref("");

function openReturnFromOrder(order: SupplierOrder) {
  returnOrderRef.value = order;
  returnGlobalNote.value = `Повернення по замовленню #${order.id}`;
  returnLines.value = order.items.map(item => ({
    productId: item.productId,
    productName: productById(item.productId)?.name ?? `Товар #${item.productId}`,
    warehouseId: 0,
    quantity: 1,
    note: "",
    include: true,
  }));
  showReturn.value = true;
}

async function doReturnToSupplier() {
  const lines = returnLines.value.filter(l => l.include && l.quantity > 0);
  if (!lines.length) return;
  saving.value = true; error.value = "";
  try {
    for (const line of lines) {
      await api.createMovement(token.value, {
        productId: line.productId,
        warehouseId: line.warehouseId,
        type: "return_to_supplier",
        quantity: line.quantity,
        note: line.note || returnGlobalNote.value,
      });
    }
    showReturn.value = false;
    warehouseStocks.value = await api.warehouseStocks(token.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

// ── Status change with auto-open movement modal ───────────────
async function changeStatus(id: number, status: string) {
  try {
    await api.updateSupplierOrderStatus(token.value, id, status);
    await load();
    // Refresh detail
    if (selectedOrder.value?.id === id) {
      const updated = supplierOrders.value.find(o => o.id === id);
      if (updated) {
        selectedOrder.value = updated;
        [detailPurchases.value, detailPayments.value] = await Promise.all([
          api.purchases(token.value, id),
          api.payments(token.value, { supplierOrderId: id }),
        ]);
        // Auto-open movement modal when status becomes "received"
        if (status === "received") {
          openMovementFromOrder(updated);
        }
      }
    }
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
    else error.value = e.message;
  }
}

onMounted(async () => {
  await load();
  await loadRecommendations();
  if (props.initialOrderId) {
    const order = supplierOrders.value.find(o => o.id === props.initialOrderId);
    if (order) {
      subTab.value = "orders";
      await openOrderDetail(order);
    }
  }
});
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Закупівлі та постачальники</h2>
      <div style="display:flex;gap:0.5rem">
        <button class="ghost-button" @click="showNewOrder=true">+ Замовлення постачальнику</button>
        <button class="ghost-button" @click="showNewSupplier=true">+ Постачальник</button>
      </div>
    </div>

    <div class="tab-row" style="margin-bottom:1rem">
      <button :class="['tab-button', subTab==='orders'&&'tab-button--active']" @click="subTab='orders'">Замовлення</button>
      <button :class="['tab-button', subTab==='recommendations'&&'tab-button--active']" @click="subTab='recommendations'">Рекомендації</button>
      <button :class="['tab-button', subTab==='suppliers'&&'tab-button--active']" @click="subTab='suppliers'">Постачальники</button>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <!-- ORDERS LIST -->
    <div v-if="subTab==='orders'" style="display:flex;gap:1rem;align-items:flex-start;flex-wrap:wrap">

      <!-- List -->
      <div :style="selectedOrder ? 'flex:0 0 360px;min-width:260px' : 'flex:1'">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th>Постачальник</th>
              <th>Статус</th>
              <th style="text-align:right">Сума</th>
              <th>Дата</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="o in [...supplierOrders].sort((a,b)=>b.id-a.id)"
              :key="o.id"
              style="cursor:pointer"
              :style="selectedOrder?.id===o.id ? 'background:rgba(109,212,160,0.08)' : ''"
              @click="openOrderDetail(o)"
            >
              <td class="subtle">#{{ o.id }}</td>
              <td style="font-weight:500">{{ supplierById(o.supplierId)?.name ?? `#${o.supplierId}` }}</td>
              <td>
                <span :style="`font-size:0.78rem;padding:0.15rem 0.5rem;border-radius:10px;background:${statusColor[o.status]}22;color:${statusColor[o.status]};font-weight:600`">
                  {{ statusLabel[o.status] ?? o.status }}
                </span>
              </td>
              <td style="text-align:right;font-weight:600">{{ o.total.toFixed(2) }} {{ o.currency }}</td>
              <td class="subtle">{{ new Date(o.createdAt).toLocaleDateString('uk') }}</td>
            </tr>
            <tr v-if="!supplierOrders.length"><td colspan="5" class="subtle">Немає замовлень</td></tr>
          </tbody>
        </table>
      </div>

      <!-- Detail panel -->
      <div v-if="selectedOrder" style="flex:1;min-width:300px">
        <div class="panel" style="padding:1.2rem">

          <!-- Header -->
          <div style="display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:1rem">
            <div>
              <div style="font-size:0.75rem;color:var(--text-muted);margin-bottom:0.25rem">Замовлення постачальнику</div>
              <h3 style="margin:0;font-size:1.1rem">#{{ selectedOrder.id }} — {{ supplierById(selectedOrder.supplierId)?.name ?? `Постачальник #${selectedOrder.supplierId}` }}</h3>
            </div>
            <div style="display:flex;gap:0.4rem;align-items:center">
              <button v-if="selectedOrder.status==='draft'" class="ghost-button" style="padding:0.2rem 0.6rem;font-size:0.82rem" @click="openEditOrder(selectedOrder)">✏️ Редагувати</button>
              <button class="ghost-button" style="padding:0.2rem 0.5rem;font-size:0.9rem" @click="selectedOrder=null">✕</button>
            </div>
          </div>

          <!-- Status + dates -->
          <div style="display:flex;gap:0.6rem;flex-wrap:wrap;margin-bottom:0.6rem">
            <span :style="`padding:0.3rem 0.8rem;border-radius:12px;font-size:0.82rem;font-weight:700;background:${statusColor[selectedOrder.status]}22;color:${statusColor[selectedOrder.status]}`">
              {{ statusLabel[selectedOrder.status] ?? selectedOrder.status }}
            </span>
            <span class="subtle" style="font-size:0.82rem;align-self:center">Створено: {{ new Date(selectedOrder.createdAt).toLocaleString('uk') }}</span>
            <span class="subtle" style="font-size:0.82rem;align-self:center">Оновлено: {{ new Date(selectedOrder.updatedAt).toLocaleString('uk') }}</span>
          </div>

          <!-- Status action buttons -->
          <div style="display:flex;gap:0.4rem;flex-wrap:wrap;margin-bottom:1rem;padding:0.6rem 0.8rem;background:rgba(255,255,255,0.03);border-radius:8px;border:1px solid rgba(255,255,255,0.06)">
            <span style="font-size:0.72rem;color:var(--text-muted);align-self:center;margin-right:0.3rem;text-transform:uppercase;letter-spacing:0.04em">Змінити статус:</span>
            <button v-if="selectedOrder.status==='draft'" class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.6rem"
              @click="changeStatus(selectedOrder.id,'sent')">📤 Відправлено</button>
            <button v-if="selectedOrder.status==='sent'" class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.6rem"
              @click="changeStatus(selectedOrder.id,'confirmed')">✅ Підтверджено</button>
            <button v-if="['sent','confirmed'].includes(selectedOrder.status)" class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.6rem"
              @click="changeStatus(selectedOrder.id,'in_transit')">🚛 В дорозі</button>
            <button v-if="['confirmed','in_transit'].includes(selectedOrder.status)"
              style="font-size:0.78rem;padding:0.25rem 0.8rem;background:rgba(109,212,160,0.15);border:1px solid rgba(109,212,160,0.4);color:#6dd4a0;border-radius:6px;cursor:pointer"
              @click="changeStatus(selectedOrder.id,'received')">📦 Надійшло →</button>
            <button v-if="selectedOrder.status==='received'" class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.6rem"
              @click="changeStatus(selectedOrder.id,'closed')">🔒 Закрити</button>
            <button v-if="!['closed','cancelled','received'].includes(selectedOrder.status)" class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.6rem;color:#ff9ca0"
              @click="changeStatus(selectedOrder.id,'cancelled')">✕ Скасувати</button>
            <button v-if="selectedOrder.status==='received'" class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.6rem;margin-left:auto;border-color:rgba(109,212,160,0.4);color:#6dd4a0"
              @click="openMovementFromOrder(selectedOrder)">📦 Рух товару</button>
            <button v-if="['received','closed'].includes(selectedOrder.status)" class="ghost-button" style="font-size:0.78rem;padding:0.25rem 0.6rem;border-color:rgba(224,160,96,0.4);color:#e0a060"
              @click="openReturnFromOrder(selectedOrder)">↪️ Повернення</button>
          </div>

          <!-- Supplier info -->
          <div v-if="supplierById(selectedOrder.supplierId)" style="margin-bottom:1rem;padding:0.7rem 0.9rem;background:rgba(255,255,255,0.04);border-radius:8px">
            <div style="font-size:0.72rem;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.4rem">Постачальник</div>
            <div style="font-weight:600;margin-bottom:0.2rem">{{ supplierById(selectedOrder.supplierId)!.name }}</div>
            <div v-if="supplierById(selectedOrder.supplierId)!.phone" class="subtle" style="font-size:0.82rem">📞 {{ supplierById(selectedOrder.supplierId)!.phone }}</div>
            <div v-if="supplierById(selectedOrder.supplierId)!.email" class="subtle" style="font-size:0.82rem">✉ {{ supplierById(selectedOrder.supplierId)!.email }}</div>
          </div>

          <!-- Financials summary -->
          <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:0.5rem;margin-bottom:1rem">
            <div style="padding:0.6rem;background:rgba(255,255,255,0.04);border-radius:8px;text-align:center">
              <div style="font-size:0.68rem;color:var(--text-muted);text-transform:uppercase;margin-bottom:0.2rem">Сума</div>
              <div style="font-weight:700;font-size:1rem">{{ selectedOrder.total.toFixed(2) }}<br><span style="font-size:0.75rem;font-weight:400">{{ selectedOrder.currency }}</span></div>
            </div>
            <div style="padding:0.6rem;background:rgba(109,212,160,0.07);border-radius:8px;text-align:center;border:1px solid rgba(109,212,160,0.15)">
              <div style="font-size:0.68rem;color:var(--text-muted);text-transform:uppercase;margin-bottom:0.2rem">Оплачено</div>
              <div style="font-weight:700;font-size:1rem;color:#6dd4a0">{{ detailPaid.toFixed(2) }}<br><span style="font-size:0.75rem;font-weight:400">UAH</span></div>
            </div>
            <div :style="`padding:0.6rem;background:${detailDebt>0?'rgba(255,156,160,0.07)':'rgba(109,212,160,0.05)'};border-radius:8px;text-align:center;border:1px solid ${detailDebt>0?'rgba(255,156,160,0.2)':'rgba(109,212,160,0.1)'}`">
              <div style="font-size:0.68rem;color:var(--text-muted);text-transform:uppercase;margin-bottom:0.2rem">Залишок</div>
              <div :style="`font-weight:700;font-size:1rem;color:${detailDebt>0?'#ff9ca0':'#6dd4a0'}`">{{ detailDebt.toFixed(2) }}<br><span style="font-size:0.75rem;font-weight:400">UAH</span></div>
            </div>
          </div>

          <div v-if="detailLoading" class="subtle" style="text-align:center;padding:1rem">Завантаження...</div>
          <template v-else>

            <!-- Order items -->
            <div style="font-size:0.72rem;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.4rem">Позиції замовлення</div>
            <table style="font-size:0.85rem;margin-bottom:1rem">
              <thead><tr><th>Товар</th><th style="text-align:right">К-сть</th><th style="text-align:right">Ціна</th><th style="text-align:right">Сума</th></tr></thead>
              <tbody>
                <tr v-for="(item, i) in selectedOrder.items" :key="i">
                  <td>{{ productById(item.productId)?.name ?? `Товар #${item.productId}` }}</td>
                  <td style="text-align:right">{{ item.quantity }}</td>
                  <td style="text-align:right">{{ item.price.toFixed(2) }}</td>
                  <td style="text-align:right;font-weight:600">{{ (item.quantity * item.price).toFixed(2) }}</td>
                </tr>
              </tbody>
            </table>

            <!-- Receipts / Purchases -->
            <div style="font-size:0.72rem;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.4rem">
              Надходження
              <span :style="`margin-left:0.4rem;padding:0.1rem 0.45rem;border-radius:8px;font-size:0.7rem;background:${detailReceived?'rgba(109,212,160,0.15)':'rgba(255,156,160,0.12)'};color:${detailReceived?'#6dd4a0':'#ff9ca0'}`">
                {{ detailReceived ? '✓ Отримано' : '⏳ Очікується' }}
              </span>
            </div>
            <div v-if="!detailPurchases.length" class="subtle" style="font-size:0.85rem;margin-bottom:1rem">Товар ще не надходив</div>
            <table v-else style="font-size:0.85rem;margin-bottom:1rem">
              <thead><tr><th>#</th><th>Дата</th><th style="text-align:right">Сума</th><th>Примітка</th></tr></thead>
              <tbody>
                <tr v-for="p in detailPurchases" :key="p.id">
                  <td class="subtle">#{{ p.id }}</td>
                  <td>{{ new Date(p.createdAt).toLocaleDateString('uk') }}</td>
                  <td style="text-align:right;font-weight:600;color:#6dd4a0">{{ p.total.toFixed(2) }} {{ p.currency }}</td>
                  <td class="subtle" style="font-size:0.8rem">{{ p.note || '—' }}</td>
                </tr>
              </tbody>
            </table>

            <!-- Payments -->
            <div style="font-size:0.72rem;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.4rem">
              Оплати
              <span :style="`margin-left:0.4rem;padding:0.1rem 0.45rem;border-radius:8px;font-size:0.7rem;background:${detailDebt<=0?'rgba(109,212,160,0.15)':'rgba(255,156,160,0.12)'};color:${detailDebt<=0?'#6dd4a0':'#ff9ca0'}`">
                {{ detailDebt <= 0 ? '✓ Повністю оплачено' : `Борг: ${detailDebt.toFixed(2)} UAH` }}
              </span>
            </div>
            <div v-if="!detailPayments.length" class="subtle" style="font-size:0.85rem;margin-bottom:0.5rem">Оплат не зафіксовано</div>
            <table v-else style="font-size:0.85rem;margin-bottom:0.5rem">
              <thead><tr><th>Дата</th><th>Метод</th><th style="text-align:right">Сума</th><th>Примітка</th></tr></thead>
              <tbody>
                <tr v-for="p in detailPayments" :key="p.id">
                  <td>{{ new Date(p.createdAt).toLocaleDateString('uk') }}</td>
                  <td>{{ methodLabel[p.method] ?? p.method }}</td>
                  <td style="text-align:right;font-weight:600;color:#6dd4a0">{{ p.amountUah.toFixed(2) }} UAH</td>
                  <td class="subtle" style="font-size:0.8rem">{{ p.note || '—' }}</td>
                </tr>
              </tbody>
            </table>

            <!-- Add Payment Button -->
            <button v-if="!showPaymentForm && !['closed','cancelled'].includes(selectedOrder!.status)"
              class="ghost-button" style="font-size:0.8rem;padding:0.25rem 0.7rem;border-color:rgba(109,212,160,0.4);color:#6dd4a0"
              @click="openPaymentForm">+ Оплата</button>

            <!-- Add Payment Form -->
            <div v-if="showPaymentForm" style="background:rgba(122,185,154,0.06);border:1px solid rgba(122,185,154,0.2);border-radius:8px;padding:0.75rem;margin-top:0.5rem">
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
                <button style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.35rem 0.9rem;font-size:0.82rem;cursor:pointer;font-weight:600" @click="saveSupplierPayment" :disabled="paymentSaving">
                  {{ paymentSaving ? 'Збереження...' : '✓ Зберегти' }}
                </button>
              </div>
            </div>

          </template>
        </div>
      </div>
    </div>

    <!-- RECOMMENDATIONS -->
    <div v-if="subTab==='recommendations'">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.8rem">
        <span class="subtle" style="font-size:0.9rem">Рекомендації по дефіциту (мін. залишок + продажі за 30 днів)</span>
        <button v-if="selectedGroups.size" class="ghost-button" @click="bulkOrderFromRecommendations" :disabled="saving">
          Замовити обраних ({{ selectedGroups.size }})
        </button>
      </div>
      <div v-if="!recommendations.length" class="subtle">Дефіциту не виявлено</div>
      <div v-for="group in recommendations" :key="group.supplierName" class="panel" style="padding:1rem;margin-bottom:0.8rem">
        <div style="display:flex;align-items:center;gap:0.8rem;margin-bottom:0.7rem">
          <input v-if="group.supplierId" type="checkbox"
            :checked="selectedGroups.has(group.supplierId!)"
            @change="group.supplierId && (selectedGroups.has(group.supplierId!) ? selectedGroups.delete(group.supplierId!) : selectedGroups.add(group.supplierId!))">
          <strong>{{ group.supplierName }}</strong>
          <span class="subtle" style="font-size:0.85rem">{{ group.productsCount }} позицій · {{ group.totalRecommended }} шт</span>
          <button class="ghost-button" style="margin-left:auto;padding:0.3rem 0.8rem;font-size:0.85rem"
            @click="orderFromGroup(group)" :disabled="saving||!group.supplierId">
            Замовити
          </button>
          <a v-if="group.supplierPhone"
            :href="`https://t.me/${group.supplierPhone.replace(/[^0-9]/,'')}?text=${encodeURIComponent(buildTelegramMsg(group))}`"
            target="_blank" rel="noopener"
            class="ghost-button" style="padding:0.3rem 0.8rem;font-size:0.85rem;text-decoration:none">Telegram
          </a>
        </div>
        <table style="font-size:0.85rem">
          <thead><tr><th>Товар</th><th>Залишок</th><th>Мін</th><th>Продано 30д</th><th>Рекомендовано</th></tr></thead>
          <tbody>
            <tr v-for="item in group.items" :key="item.productId">
              <td>{{ item.productName }} <span class="subtle">({{ item.sku }})</span></td>
              <td :style="item.currentStock<=item.minStock?'color:#ff9ca0;font-weight:700':''">{{ item.currentStock }}</td>
              <td>{{ item.minStock }}</td>
              <td>{{ item.soldLast30Days }}</td>
              <td style="font-weight:700;color:#9fe8c4">{{ item.recommendedQty }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <!-- SUPPLIERS -->
    <div v-if="subTab==='suppliers'">
      <table>
        <thead><tr><th>Назва</th><th>Контакт</th><th>Телефон</th><th>Email</th></tr></thead>
        <tbody>
          <tr v-for="s in suppliers" :key="s.id">
            <td>{{ s.name }}</td><td>{{ s.contact }}</td><td>{{ s.phone }}</td><td>{{ s.email }}</td>
          </tr>
          <tr v-if="!suppliers.length"><td colspan="4" class="subtle">Немає постачальників</td></tr>
        </tbody>
      </table>
    </div>

    <!-- SUPPLIER B2B PORTAL -->
    <!-- MODAL: New Supplier -->
    <div v-if="showNewSupplier" class="modal-backdrop" @click.self="showNewSupplier=false">
      <div class="panel modal-box">
        <h3>Новий постачальник</h3>
        <div class="grid">
          <label>Назва <input v-model="suppForm.name"></label>
          <label>Контактна особа <input v-model="suppForm.contact"></label>
          <label>Телефон <input v-model="suppForm.phone"></label>
          <label>Email <input type="email" v-model="suppForm.email"></label>
          <label>Коментарі <textarea v-model="suppForm.comments" rows="2"></textarea></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="addSupplier" :disabled="saving||!suppForm.name">{{ saving?'..':'Зберегти' }}</button>
          <button class="ghost-button" @click="showNewSupplier=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: New Supplier Order -->
    <div v-if="showNewOrder" class="modal-backdrop" @click.self="showNewOrder=false; supplierNameInput=''">
      <div class="panel modal-box" style="max-width:560px">
        <h3>Нове замовлення постачальнику</h3>
        <div class="grid">
          <label>Постачальник *
            <div style="display:flex;gap:0.4rem;align-items:flex-start">
              <div style="flex:1;position:relative">
                <input v-model="supplierNameInput" placeholder="Пошук за назвою, телефоном..."
                  style="width:100%"
                  @focus="showSupplierDropdown=true"
                  @input="showSupplierDropdown=true; orderForm.supplierId=0">
                <div v-if="showSupplierDropdown && filteredSupplierSearch.length > 0"
                  style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                  <div v-for="s in filteredSupplierSearch" :key="s.id"
                    style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.87rem;border-bottom:1px solid rgba(72,187,120,0.08)"
                    @mousedown.prevent="selectSupplier(s)"
                    @mouseover="($event.currentTarget as HTMLElement).style.background='rgba(72,187,120,0.12)'"
                    @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                    <span style="font-weight:600">{{ s.name }}</span>
                    <span v-if="s.phone" class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ s.phone }}</span>
                  </div>
                </div>
              </div>
              <button class="ghost-button" style="white-space:nowrap;font-size:0.8rem;padding:0.4rem 0.7rem"
                @click="showNewSupplierInlineForm=!showNewSupplierInlineForm; showSupplierDropdown=false">
                {{ showNewSupplierInlineForm ? '✕' : '+ Новий' }}
              </button>
            </div>
            <div v-if="orderForm.supplierId" style="font-size:0.78rem;color:#6dd4a0;margin-top:0.2rem">
              ✓ Обрано: {{ supplierNameInput }}
            </div>
          </label>

          <!-- Quick create supplier inline -->
          <div v-if="showNewSupplierInlineForm"
            style="background:rgba(72,187,120,0.06);border:1px solid rgba(72,187,120,0.2);border-radius:8px;padding:0.75rem">
            <div style="font-size:0.8rem;font-weight:600;color:#7ab99a;margin-bottom:0.5rem">Новий постачальник</div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
              <label style="font-size:0.78rem">Назва *<input v-model="newInlineSupplierForm.name" class="input" placeholder="Назва" style="margin-top:0.2rem"></label>
              <label style="font-size:0.78rem">Телефон<input v-model="newInlineSupplierForm.phone" class="input" placeholder="+380..." style="margin-top:0.2rem"></label>
            </div>
            <label style="font-size:0.78rem;display:block;margin-bottom:0.5rem">Контакт<input v-model="newInlineSupplierForm.contact" class="input" style="margin-top:0.2rem;width:100%"></label>
            <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newInlineSupplierSaving||!newInlineSupplierForm.name.trim()"
              @click="saveNewInlineSupplier">
              {{ newInlineSupplierSaving ? '..' : '✓ Зберегти постачальника' }}
            </button>
          </div>

          <label>Валюта
            <select v-model="orderForm.currency"><option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option></select>
          </label>
        </div>
        <div style="margin-top:0.8rem">
          <strong style="font-size:0.9rem">Позиції:</strong>
          <table v-if="orderForm.items.length" style="font-size:0.85rem;margin-top:0.4rem">
            <thead><tr><th>Номенклатура</th><th>К-сть</th><th>Ціна</th><th></th></tr></thead>
            <tbody>
              <tr v-for="(item, i) in orderForm.items" :key="i">
                <td>{{ item._name || `#${item.productId}` }}</td><td>{{ item.quantity }}</td><td>{{ item.price }}</td>
                <td><button class="ghost-button" style="padding:0.2rem 0.5rem" @click="orderForm.items.splice(i,1)">✕</button></td>
              </tr>
            </tbody>
          </table>

          <div style="margin-top:0.6rem;border:1px solid rgba(255,255,255,0.08);border-radius:8px;padding:0.7rem;background:rgba(255,255,255,0.03)">
            <div style="font-size:0.82rem;color:var(--text-subtle);margin-bottom:0.5rem">Додати позицію</div>
            <div style="position:relative;margin-bottom:0.5rem">
              <input
                v-model="orderLineSearch"
                @focus="orderLineShowDropdown=true"
                @blur="hideDropdownDelayed()"
                placeholder="Пошук за назвою, SKU, штрихкодом..."
                style="width:100%;box-sizing:border-box"
              >
              <div v-if="orderLineShowDropdown && !orderLineSelectedProduct"
                style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                <div v-for="p in filteredProducts" :key="p.id"
                  @mousedown.prevent="selectOrderProduct(p)"
                  style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.85rem;display:flex;justify-content:space-between;align-items:center"
                  class="dropdown-item">
                  <span>{{ p.name }}</span>
                  <span style="color:var(--text-subtle);font-size:0.78rem">{{ p.sku }}</span>
                </div>
                <div v-if="!filteredProducts.length" style="padding:0.5rem 0.7rem;font-size:0.85rem;color:var(--text-subtle)">Не знайдено</div>
                <div @mousedown.prevent="showNewNomenclature=true;orderLineShowDropdown=false"
                  style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.85rem;border-top:1px solid rgba(255,255,255,0.1);color:#9fe8c4">
                  + Створити нову номенклатуру
                </div>
              </div>
            </div>
            <div v-if="orderLineSelectedProduct" style="display:flex;gap:0.4rem;align-items:center;flex-wrap:wrap">
              <span style="font-size:0.85rem;flex:1;min-width:120px;color:#9fe8c4">✓ {{ orderLineSelectedProduct.name }}</span>
              <input type="number" v-model.number="orderLineQty" placeholder="К-сть" min="1" style="width:70px">
              <input type="number" v-model.number="orderLinePrice" placeholder="Ціна" min="0" style="width:80px">
              <button class="ghost-button" @click="addOrderLine">+</button>
              <button class="ghost-button" style="padding:0.3rem 0.5rem" @click="orderLineSelectedProduct=null;orderLineSearch=''">✕</button>
            </div>
            <div v-else-if="!orderLineShowDropdown" style="margin-top:0.3rem">
              <button class="ghost-button" style="font-size:0.82rem" @click="showNewNomenclature=true">+ Створити нову номенклатуру</button>
            </div>
          </div>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="createOrder" :disabled="saving||!orderForm.supplierId||!orderForm.items.length">{{ saving?'..':'Створити' }}</button>
          <button class="ghost-button" @click="showNewOrder=false; supplierNameInput=''; orderForm.supplierId=0; showNewSupplierInlineForm=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Receive Order -->
    <div v-if="showReceive" class="modal-backdrop" @click.self="showReceive=null">
      <div class="panel modal-box" style="max-width:560px">
        <h3>Прийняти замовлення #{{ showReceive.id }}</h3>
        <table style="font-size:0.85rem;margin-bottom:0.8rem">
          <thead><tr><th>ID товару</th><th>Замовлено</th><th>Приймаємо</th><th>Ціна</th></tr></thead>
          <tbody>
            <tr v-for="(line, i) in receiveLines" :key="i">
              <td>{{ line.productId }}</td>
              <td>{{ showReceive.items[i]?.quantity }}</td>
              <td><input type="number" min="0" v-model.number="line.quantity" style="width:70px;padding:0.25rem"></td>
              <td><input type="number" min="0" v-model.number="line.price" style="width:80px;padding:0.25rem"></td>
            </tr>
          </tbody>
        </table>
        <label>Примітка <input v-model="receiveNote" placeholder="Примітка"></label>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="doReceive" :disabled="saving">{{ saving?'..':'Провести надходження' }}</button>
          <button class="ghost-button" @click="showReceive=null">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>
    <!-- MODAL: New Nomenclature (from order) -->
    <div v-if="showNewNomenclature" class="modal-backdrop modal-backdrop--top" @click.self="showNewNomenclature=false">
      <div class="panel modal-box" style="max-width:480px">
        <h3>Нова номенклатура</h3>
        <div class="grid">
          <label>Назва* <input v-model="newNomForm.name" placeholder="Назва товару"></label>
          <label>SKU <input v-model="newNomForm.sku" placeholder="Артикул/SKU"></label>
          <label>Штрихкод <input v-model="newNomForm.barcode" placeholder="Штрихкод"></label>
          <label>Категорія <input v-model="newNomForm.category" placeholder="Категорія"></label>
          <label>Бренд <input v-model="newNomForm.brand" placeholder="Бренд"></label>
          <label>Закупівельна ціна <input type="number" min="0" v-model.number="newNomForm.purchasePrice"></label>
          <label>Роздрібна ціна <input type="number" min="0" v-model.number="newNomForm.retailPrice"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="saveNewNomenclature" :disabled="savingNom||!newNomForm.name">{{ savingNom?'..':'Створити та додати' }}</button>
          <button class="ghost-button" @click="showNewNomenclature=false; nomError=''">Скасувати</button>
        </div>
        <p v-if="nomError" class="error-text">{{ nomError }}</p>
      </div>
    </div>

    <!-- MODAL: Рух товару (з замовлення постачальника) -->
    <div v-if="showMovement" class="modal-backdrop" @click.self="showMovement=false">
      <div class="panel modal-box" style="max-width:600px">
        <h3>📦 Рух товару — Замовлення #{{ movOrderRef?.id }}</h3>
        <p class="subtle" style="font-size:0.85rem;margin-bottom:0.8rem">
          Вкажіть кількість та склад для кожної позиції. Усі позиції будуть оприбутковані як <strong>Надходження</strong>.
        </p>
        <table style="font-size:0.85rem;margin-bottom:1rem">
          <thead>
            <tr>
              <th>Товар</th>
              <th>Залишок</th>
              <th style="text-align:right">К-сть</th>
              <th>Склад</th>
              <th>Примітка</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(line, i) in movLines" :key="i">
              <td style="font-weight:500">{{ line.productName }}</td>
              <td style="color:#aaa;font-size:0.8rem">{{ stockForProduct(line.productId) }} шт</td>
              <td style="text-align:right">
                <input type="number" min="0" v-model.number="line.quantity" style="width:70px;padding:0.2rem 0.4rem">
              </td>
              <td>
                <select v-model.number="line.warehouseId" style="font-size:0.8rem;padding:0.2rem 0.4rem">
                  <option :value="0">— Основний —</option>
                  <optgroup label="Магазини">
                    <option v-for="w in warehouses.filter(w=>w.locationType==='shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                  </optgroup>
                  <optgroup label="Склади">
                    <option v-for="w in warehouses.filter(w=>w.locationType!=='shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                  </optgroup>
                </select>
              </td>
              <td>
                <input v-model="line.note" style="font-size:0.8rem;padding:0.2rem 0.4rem;width:100%;min-width:120px">
              </td>
            </tr>
          </tbody>
        </table>
        <div style="display:flex;gap:0.5rem;margin-top:0.5rem">
          <button @click="doMovementFromOrder" :disabled="saving">{{ saving ? 'Збереження...' : '✓ Провести надходження' }}</button>
          <button class="ghost-button" @click="showMovement=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Повернення постачальнику -->
    <div v-if="showReturn" class="modal-backdrop" @click.self="showReturn=false">
      <div class="panel modal-box" style="max-width:660px">
        <h3>↪️ Повернення постачальнику — Замовлення #{{ returnOrderRef?.id }}</h3>
        <p class="subtle" style="font-size:0.85rem;margin-bottom:0.8rem">
          Позначте позиції для повернення, вкажіть кількість та причину. Залишок на складі зменшиться на вказану кількість.
        </p>

        <!-- Global note -->
        <label style="display:block;margin-bottom:0.8rem;font-size:0.85rem">
          Загальна примітка (застосовується до всіх позицій без власної)
          <input v-model="returnGlobalNote" placeholder="Причина повернення..." style="margin-top:0.3rem;width:100%;box-sizing:border-box">
        </label>

        <table style="font-size:0.85rem;margin-bottom:1rem">
          <thead>
            <tr>
              <th style="width:28px"></th>
              <th>Товар</th>
              <th style="text-align:center">На складі</th>
              <th style="text-align:right">К-сть повернення</th>
              <th>Склад</th>
              <th>Причина (необов.)</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(line, i) in returnLines" :key="i"
              :style="!line.include ? 'opacity:0.4' : ''">
              <td>
                <input type="checkbox" v-model="line.include" style="cursor:pointer">
              </td>
              <td style="font-weight:500">{{ line.productName }}</td>
              <td style="text-align:center">
                <span :style="`font-weight:700;color:${stockForProduct(line.productId)>0?'#6dd4a0':'#ff9ca0'}`">
                  {{ stockForProduct(line.productId) }} шт
                </span>
              </td>
              <td style="text-align:right">
                <input type="number" min="1" :max="stockForProduct(line.productId)"
                  v-model.number="line.quantity"
                  :disabled="!line.include"
                  style="width:75px;padding:0.2rem 0.4rem">
              </td>
              <td>
                <select v-model.number="line.warehouseId" :disabled="!line.include" style="font-size:0.8rem;padding:0.2rem 0.4rem">
                  <option :value="0">— Основний —</option>
                  <optgroup label="Магазини">
                    <option v-for="w in warehouses.filter(w=>w.locationType==='shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                  </optgroup>
                  <optgroup label="Склади">
                    <option v-for="w in warehouses.filter(w=>w.locationType!=='shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                  </optgroup>
                </select>
              </td>
              <td>
                <input v-model="line.note" :disabled="!line.include"
                  placeholder="Брак, пересорт..."
                  style="font-size:0.8rem;padding:0.2rem 0.4rem;width:100%;min-width:110px">
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Summary -->
        <div style="background:rgba(224,160,96,0.08);border:1px solid rgba(224,160,96,0.2);border-radius:8px;padding:0.6rem 0.9rem;margin-bottom:0.9rem;font-size:0.85rem">
          Повернень: <strong>{{ returnLines.filter(l=>l.include&&l.quantity>0).length }}</strong> позицій,
          <strong>{{ returnLines.filter(l=>l.include).reduce((s,l)=>s+l.quantity,0) }}</strong> шт. загалом
        </div>

        <div style="display:flex;gap:0.5rem">
          <button
            style="background:rgba(224,160,96,0.15);border:1px solid rgba(224,160,96,0.45);color:#e0a060;border-radius:6px;padding:0.4rem 1rem;cursor:pointer"
            @click="doReturnToSupplier"
            :disabled="saving || !returnLines.some(l=>l.include&&l.quantity>0)">
            {{ saving ? 'Збереження...' : '↪️ Провести повернення' }}
          </button>
          <button class="ghost-button" @click="showReturn=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Редагування замовлення постачальника -->
    <div v-if="showEditOrder" class="modal-backdrop" @click.self="showEditOrder=false">
      <div class="panel modal-box" style="max-width:580px">
        <h3>✏️ Редагування замовлення #{{ editOrderRef?.id }}</h3>
        <p class="subtle" style="font-size:0.83rem;margin-bottom:0.8rem">Редагування доступне лише для замовлень у статусі <strong>Створено</strong>.</p>

        <div class="grid">
          <!-- Supplier selector -->
          <label>Постачальник *
            <div style="position:relative">
              <input v-model="editSupplierNameInput" placeholder="Пошук постачальника..."
                style="width:100%;box-sizing:border-box"
                @focus="showEditSupplierDropdown=true"
                @input="showEditSupplierDropdown=true; editOrderForm.supplierId=0">
              <div v-if="showEditSupplierDropdown && filteredEditSupplierSearch.length > 0"
                style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:200px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45)">
                <div v-for="s in filteredEditSupplierSearch" :key="s.id"
                  style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.87rem;border-bottom:1px solid rgba(72,187,120,0.08)"
                  @mousedown.prevent="selectEditSupplier(s)"
                  @mouseover="($event.currentTarget as HTMLElement).style.background='rgba(72,187,120,0.12)'"
                  @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                  <span style="font-weight:600">{{ s.name }}</span>
                  <span v-if="s.phone" class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ s.phone }}</span>
                </div>
              </div>
            </div>
            <div v-if="editOrderForm.supplierId" style="font-size:0.78rem;color:#6dd4a0;margin-top:0.2rem">✓ Обрано: {{ editSupplierNameInput }}</div>
          </label>

          <!-- Currency -->
          <label>Валюта
            <select v-model="editOrderForm.currency">
              <option value="UAH">UAH (грн)</option>
              <option value="USD">USD ($)</option>
              <option value="EUR">EUR (€)</option>
            </select>
          </label>
        </div>

        <!-- Items table -->
        <div style="margin-top:0.9rem">
          <strong style="font-size:0.9rem">Позиції:</strong>
          <table v-if="editOrderForm.items.length" style="font-size:0.85rem;margin-top:0.4rem">
            <thead><tr><th>Номенклатура</th><th style="text-align:right">К-сть</th><th style="text-align:right">Ціна</th><th style="text-align:right">Сума</th><th></th></tr></thead>
            <tbody>
              <tr v-for="(item, i) in editOrderForm.items" :key="i">
                <td>{{ item._name || `#${item.productId}` }}</td>
                <td style="text-align:right">
                  <input type="number" min="1" v-model.number="item.quantity" style="width:65px;padding:0.2rem 0.3rem;text-align:right">
                </td>
                <td style="text-align:right">
                  <input type="number" min="0" step="0.01" v-model.number="item.price" style="width:80px;padding:0.2rem 0.3rem;text-align:right">
                </td>
                <td style="text-align:right;font-weight:600">{{ (item.quantity * item.price).toFixed(2) }}</td>
                <td><button class="ghost-button" style="padding:0.2rem 0.5rem;color:#ff9ca0" @click="editOrderForm.items.splice(i,1)">✕</button></td>
              </tr>
              <tr style="border-top:1px solid rgba(255,255,255,0.1)">
                <td colspan="3" style="text-align:right;font-size:0.8rem;color:var(--text-muted)">Разом:</td>
                <td style="text-align:right;font-weight:700;color:#9fe8c4">
                  {{ editOrderForm.items.reduce((s, i) => s + i.quantity * i.price, 0).toFixed(2) }} {{ editOrderForm.currency }}
                </td>
                <td></td>
              </tr>
            </tbody>
          </table>
          <div v-else class="subtle" style="font-size:0.85rem;margin:0.4rem 0">Немає позицій</div>

          <!-- Add line -->
          <div style="margin-top:0.6rem;border:1px solid rgba(255,255,255,0.08);border-radius:8px;padding:0.7rem;background:rgba(255,255,255,0.03)">
            <div style="font-size:0.82rem;color:var(--text-subtle);margin-bottom:0.5rem">Додати позицію</div>
            <div style="position:relative;margin-bottom:0.5rem">
              <input
                v-model="editOrderLineSearch"
                @focus="editOrderLineShowDropdown=true"
                @blur="hideEditDropdownDelayed()"
                placeholder="Пошук за назвою, SKU, штрихкодом..."
                style="width:100%;box-sizing:border-box"
              >
              <div v-if="editOrderLineShowDropdown && !editOrderLineSelectedProduct"
                style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:200px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45)">
                <div v-for="p in filteredEditProducts" :key="p.id"
                  @mousedown.prevent="selectEditOrderProduct(p)"
                  style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.85rem;display:flex;justify-content:space-between;align-items:center"
                  class="dropdown-item">
                  <span>{{ p.name }}</span>
                  <span style="color:var(--text-subtle);font-size:0.78rem">{{ p.sku }}</span>
                </div>
                <div v-if="!filteredEditProducts.length" style="padding:0.5rem 0.7rem;font-size:0.85rem;color:var(--text-subtle)">Не знайдено</div>
              </div>
            </div>
            <div v-if="editOrderLineSelectedProduct" style="display:flex;gap:0.4rem;align-items:center;flex-wrap:wrap">
              <span style="font-size:0.85rem;flex:1;min-width:120px;color:#9fe8c4">✓ {{ editOrderLineSelectedProduct.name }}</span>
              <input type="number" v-model.number="editOrderLineQty" placeholder="К-сть" min="1" style="width:70px">
              <input type="number" v-model.number="editOrderLinePrice" placeholder="Ціна" min="0" style="width:80px">
              <button class="ghost-button" @click="addEditOrderLine">+</button>
              <button class="ghost-button" style="padding:0.3rem 0.5rem" @click="editOrderLineSelectedProduct=null;editOrderLineSearch=''">✕</button>
            </div>
          </div>
        </div>

        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="saveEditOrder" :disabled="saving || !editOrderForm.supplierId || !editOrderForm.items.length">
            {{ saving ? '...' : '✓ Зберегти зміни' }}
          </button>
          <button class="ghost-button" @click="showEditOrder=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>
  </div>
</template>
