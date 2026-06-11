<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import BarcodeScanner from "./BarcodeScanner.vue";
import OrderChainTree from "./OrderChainTree.vue";
import type { UserSession, Product, Sale, CustomerOrder, Customer, Cashbox, CurrencyCode, SaleItem, Payment, SupplierOrder, Purchase, OrderChain, OrderChainNode, ServiceOrder, DebtSummary, Supplier } from "../types";

const props = defineProps<{ session: UserSession; initialSubTab?: string }>();
const emit = defineEmits<{
  (e: "session-expired"): void;
  (e: "navigate", payload: { type: string; id: number } | string): void;
}>();

const token = computed(() => props.session.token);
const can = (p: string) => props.session.permissions?.includes(p) ?? false;

const products = ref<Product[]>([]);
const sales = ref<Sale[]>([]);
const orders = ref<CustomerOrder[]>([]);
const customers = ref<Customer[]>([]);
const cashboxes = ref<Cashbox[]>([]);
const loading = ref(false);
const error = ref("");
const subTab = ref<"pos" | "sales">((props.initialSubTab as any) ?? "pos");

// POS / cart
const cartItems = ref<Array<{ product: Product; quantity: number; price: number; discountPct: number }>>([]);
const posSearch = ref("");
const posCurrency = ref<CurrencyCode>("UAH");
const posOrderId = ref<number | undefined>(undefined);
const saleSaving = ref(false);
const saleSuccess = ref("");

// Quick-create product inline
const showQuickProduct = ref(false);
const quickProductForm = ref({ name: "", sku: "", category: "", retailPrice: 0, purchasePrice: 0 });
const quickProductSaving = ref(false);

// Order detail drawer
const selectedOrder = ref<CustomerOrder | null>(null);
const orderPayments = ref<Payment[]>([]);
const orderLinkedSales = ref<Sale[]>([]);
const orderDetailLoading = ref(false);
const orderChain = ref<OrderChain | null>(null);

// Supplier orders & purchases for chain
const supplierOrders = ref<SupplierOrder[]>([]);
const purchases = ref<Purchase[]>([]);
const serviceOrders = ref<ServiceOrder[]>([]);
const debts = ref<DebtSummary[]>([]);
const suppliers = ref<Supplier[]>([]);

// Quick pay (order search)
type QuickSearchResult = {
  type: "supplier_order" | "service_order" | "customer_order";
  id: number;
  label: string;
  sub: string;
  total: number;
  paid: number;
  currency: string;
  status: string;
};
const orderSearch = ref("");
const showQuickPay = ref(false);
const quickPayTarget = ref<QuickSearchResult | null>(null);
const quickPayForm = ref({ cashboxId: 0, amount: 0, method: "cash" as "cash" | "card" | "bank", note: "" });
const quickPaying = ref(false);

const quickStatusLabel: Record<string, string> = {
  draft: "Створено", sent: "Відправлено", confirmed: "Підтверджено",
  in_transit: "В дорозі", received: "Отримано", closed: "Закрито", cancelled: "Скасовано",
};
const quickTypeIcon: Record<string, string> = {
  supplier_order: "🏭", service_order: "🔧", customer_order: "🛒",
};

const orderSearchResults = computed((): QuickSearchResult[] => {
  const q = orderSearch.value.trim().toLowerCase();
  if (!q) return [];
  const results: QuickSearchResult[] = [];
  for (const so of supplierOrders.value) {
    const supplier = suppliers.value.find(s => s.id === so.supplierId);
    const name = supplier?.name ?? `Постачальник #${so.supplierId}`;
    const debt = debts.value.find(d => d.entityType === "supplier_order" && d.entityId === so.id);
    if (`#${so.id}`.includes(q) || name.toLowerCase().includes(q) || so.status.toLowerCase().includes(q)) {
      const soTotal = debt?.total ?? so.total;
      const soPaid = debt?.paid ?? 0;
      results.push({ type: "supplier_order", id: so.id, label: `Замовлення постачальника #${so.id} — ${name}`,
        sub: `${quickStatusLabel[so.status] ?? so.status} · ${soTotal.toFixed(2)} ${so.currency}`,
        total: soTotal, paid: soPaid, currency: so.currency, status: so.status });
    }
  }
  for (const svo of serviceOrders.value) {
    if (`#${svo.id}`.includes(q) || (svo.customerName || "").toLowerCase().includes(q) || svo.status.toLowerCase().includes(q)) {
      const debt = debts.value.find(d => d.entityType === "service_order" && d.entityId === svo.id);
      results.push({ type: "service_order", id: svo.id, label: `Ремонт #${svo.id} — ${svo.customerName || "Клієнт"}`,
        sub: `${svo.status} · ${svo.total.toFixed(2)} ${svo.currency}`,
        total: (debt as any)?.total ?? (svo as any).totalUah ?? svo.total, paid: (debt as any)?.paid ?? 0, currency: "UAH", status: svo.status });
    }
  }
  for (const co of orders.value) {
    if (`#${co.id}`.includes(q) || (co.customerName || "").toLowerCase().includes(q) || co.status.toLowerCase().includes(q)) {
      const debt = debts.value.find(d => d.entityType === "customer_order" && d.entityId === co.id);
      results.push({ type: "customer_order", id: co.id, label: `Замовлення клієнта #${co.id} — ${co.customerName || "Клієнт"}`,
        sub: `${co.status} · ${co.total.toFixed(2)} ${co.currency}`,
        total: (debt as any)?.total ?? (co as any).totalUah ?? co.total, paid: (debt as any)?.paid ?? 0, currency: "UAH", status: co.status });
    }
  }
  return results.slice(0, 12);
});

function openQuickPay(r: QuickSearchResult) {
  quickPayTarget.value = r;
  quickPayForm.value = {
    cashboxId: cashboxes.value[0]?.id ?? 0,
    amount: Math.max(0, r.total - r.paid),
    method: r.type === "supplier_order" ? "bank" : "cash",
    note: "",
  };
  showQuickPay.value = true;
}

async function submitQuickPay() {
  if (!quickPayTarget.value) return;
  quickPaying.value = true; error.value = "";
  try {
    const t = quickPayTarget.value;
    await api.createPayment(token.value, {
      ...(t.type === "supplier_order" ? { supplierOrderId: t.id }
        : t.type === "service_order" ? { serviceOrderId: t.id }
        : { orderId: t.id }),
      cashboxId: quickPayForm.value.cashboxId,
      amount: quickPayForm.value.amount,
      currency: "UAH",
      method: quickPayForm.value.method,
      note: quickPayForm.value.note || `Оплата ${t.label}`,
    });
    showQuickPay.value = false;
    quickPayTarget.value = null;
    orderSearch.value = "";
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { quickPaying.value = false; }
}


const showNewOrder = ref(false);
const orderForm = ref({ customerName: "", currency: "UAH" as CurrencyCode, items: [] as SaleItem[], dueDate: "" });
const orderFormItems = ref<Array<{ productId: number; quantity: number; price: number }>>([]);

// Customer search for order form
const orderCustomerSearch = ref("");
const showOrderCustomerDropdown = ref(false);
const showNewOrderCustomerForm = ref(false);
const newOrderCustomerForm = ref({ name: "", phone: "", email: "", comment: "" });
const newOrderCustomerSaving = ref(false);

const filteredOrderCustomers = computed(() => {
  const q = orderCustomerSearch.value.trim().toLowerCase();
  if (!q) return customers.value.slice(0, 8);
  return customers.value.filter(c =>
    c.name.toLowerCase().includes(q) || (c.phone || "").includes(q)
  ).slice(0, 10);
});

function selectOrderCustomer(c: Customer) {
  orderForm.value.customerName = c.name;
  orderCustomerSearch.value = c.name;
  showOrderCustomerDropdown.value = false;
}

async function saveNewOrderCustomer() {
  if (!newOrderCustomerForm.value.name.trim()) return;
  newOrderCustomerSaving.value = true;
  try {
    const created = await api.createCustomer(token.value, {
      name: newOrderCustomerForm.value.name.trim(),
      phone: newOrderCustomerForm.value.phone,
      email: newOrderCustomerForm.value.email,
      comment: newOrderCustomerForm.value.comment,
    }) as any;
    customers.value = await api.customers(token.value);
    orderForm.value.customerName = created.name;
    orderCustomerSearch.value = created.name;
    showOrderCustomerDropdown.value = false;
    showNewOrderCustomerForm.value = false;
    newOrderCustomerForm.value = { name: "", phone: "", email: "", comment: "" };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { newOrderCustomerSaving.value = false; }
}

// Product search per item in order form
const orderItemProductSearch = ref<string[]>([]);
const showOrderItemProductDropdown = ref<boolean[]>([]);
const showNewOrderItemProductForm = ref<boolean[]>([]);
const newOrderItemProductForm = ref({ name: "", sku: "", category: "", retailPrice: 0 });
const newOrderItemProductSaving = ref(false);
const activeOrderItemIdx = ref<number | null>(null);

function filteredOrderItemProducts(idx: number) {
  const q = (orderItemProductSearch.value[idx] || "").trim().toLowerCase();
  if (!q) return products.value.filter(p => !p.archived).slice(0, 8);
  return products.value.filter(p => !p.archived && (
    p.name.toLowerCase().includes(q) || p.sku.toLowerCase().includes(q) || (p.barcode || "").includes(q)
  )).slice(0, 10);
}

function selectOrderItemProduct(p: Product, idx: number) {
  orderFormItems.value[idx].productId = p.id;
  orderFormItems.value[idx].price = p.retailPrice;
  orderItemProductSearch.value[idx] = p.name;
  showOrderItemProductDropdown.value[idx] = false;
}

async function saveNewOrderItemProduct(idx: number) {
  if (!newOrderItemProductForm.value.name.trim()) return;
  newOrderItemProductSaving.value = true;
  try {
    const created = await api.createProduct(token.value, {
      name: newOrderItemProductForm.value.name,
      sku: newOrderItemProductForm.value.sku,
      category: newOrderItemProductForm.value.category,
      retailPrice: newOrderItemProductForm.value.retailPrice,
      currency: "UAH", stock: 0, minStock: 0,
    } as any) as any;
    products.value = [...products.value, created];
    selectOrderItemProduct(created, idx);
    showNewOrderItemProductForm.value[idx] = false;
    newOrderItemProductForm.value = { name: "", sku: "", category: "", retailPrice: 0 };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { newOrderItemProductSaving.value = false; }
}

// Edit existing order
const showEditOrder = ref(false);
const editOrderId = ref<number>(0);
const editOrderForm = ref({ customerName: "", currency: "UAH" as CurrencyCode, dueDate: "" });
const editOrderItems = ref<Array<{ productId: number; quantity: number; price: number }>>([]);
const editOrderSaving = ref(false);

// Link supplier order to customer order
const showLinkSupplierOrder = ref(false);
const linkTargetCustomerOrderId = ref<number>(0);
const linkSupplierOrderId = ref<number>(0);
const linkSaving = ref(false);

function addOrderItem() {
  orderFormItems.value.push({ productId: 0, quantity: 1, price: 0 });
  orderItemProductSearch.value.push("");
  showOrderItemProductDropdown.value.push(false);
  showNewOrderItemProductForm.value.push(false);
}
function removeOrderItem(idx: number) {
  orderFormItems.value.splice(idx, 1);
  orderItemProductSearch.value.splice(idx, 1);
  showOrderItemProductDropdown.value.splice(idx, 1);
  showNewOrderItemProductForm.value.splice(idx, 1);
}
function onOrderItemProduct(idx: number) {
  const item = orderFormItems.value[idx];
  const p = products.value.find(p => p.id === item.productId);
  if (p) item.price = p.retailPrice;
}


// Payment modal
const showPayment = ref(false);
const paymentForm = ref({ entityType: "sale" as "sale" | "order", entityId: 0, cashboxId: 0, amount: 0, currency: "UAH" as CurrencyCode, method: "cash" as any, note: "" });

const saving = ref(false);

async function load() {
  loading.value = true; error.value = "";
  try {
    [products.value, sales.value, orders.value, customers.value, cashboxes.value] = await Promise.all([
      api.products(token.value),
      api.sales(token.value),
      api.orders(token.value),
      api.customers(token.value),
      api.cashboxes(token.value),
    ]);
    try { supplierOrders.value = await api.supplierOrders(token.value); } catch {}
    try { purchases.value = (await api.purchases(token.value)) as Purchase[]; } catch {}
    try { serviceOrders.value = await api.serviceOrders(token.value); } catch {}
    try { debts.value = await api.debts(token.value); } catch {}
    try { suppliers.value = await api.suppliers(token.value); } catch {}
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

const posProducts = computed(() => {
  if (!posSearch.value) return products.value.filter(p => !p.archived);
  const s = posSearch.value.toLowerCase();
  return products.value.filter(p => !p.archived && (p.name.toLowerCase().includes(s) || p.sku.toLowerCase().includes(s) || (p.barcode || "").includes(s)));
});

function addToCart(p: Product) {
  const existing = cartItems.value.find(c => c.product.id === p.id);
  if (existing) { existing.quantity++; }
  else { cartItems.value.push({ product: p, quantity: 1, price: p.retailPrice, discountPct: 0 }); }
}

function scanToCart(barcode: string) {
  const product = products.value.find(p =>
    !p.archived && (p.barcode === barcode || p.sku === barcode || String(p.id) === barcode)
  );
  if (product) {
    addToCart(product);
    posSearch.value = "";
  } else {
    posSearch.value = barcode;
  }
}

function removeFromCart(idx: number) { cartItems.value.splice(idx, 1); }

function itemEffectivePrice(item: { price: number; discountPct: number }) {
  return item.price * (1 - (item.discountPct || 0) / 100);
}

const cartTotal = computed(() =>
  cartItems.value.reduce((a, c) => a + itemEffectivePrice(c) * c.quantity, 0)
);

const cartDiscount = computed(() =>
  cartItems.value.reduce((a, c) => a + (c.price * (c.discountPct / 100)) * c.quantity, 0)
);

const fiscalResult = ref<Record<number, string>>({});

// ── Sale detail modal ─────────────────────────────────────────────────────
const selectedSale = ref<Sale | null>(null);
const saleDetailPayments = ref<Payment[]>([]);
const saleDetailReceipts = ref<any[]>([]);
const saleDetailLoading = ref(false);

async function openSaleDetail(sale: Sale) {
  selectedSale.value = sale;
  saleDetailLoading.value = true;
  saleDetailPayments.value = [];
  saleDetailReceipts.value = [];
  try {
    const [pays, recs] = await Promise.all([
      api.payments(token.value, { saleId: sale.id }).catch(() => [] as Payment[]),
      api.receipts(token.value, { saleId: sale.id }).catch(() => [] as any[]),
    ]);
    saleDetailPayments.value = pays;
    saleDetailReceipts.value = recs;
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    saleDetailLoading.value = false;
  }
}

function closeSaleDetail() {
  selectedSale.value = null;
  saleDetailPayments.value = [];
  saleDetailReceipts.value = [];
}

const saleDetailOrder = computed(() =>
  selectedSale.value?.orderId
    ? orders.value.find(o => o.id === selectedSale.value!.orderId) ?? null
    : null
);

const saleDetailPaid = computed(() =>
  saleDetailPayments.value.reduce((a, p) => a + p.amount, 0)
);
// ─────────────────────────────────────────────────────────────────────────

// pagination
const SALES_PAGE_SIZE = 30;
const salesPage = ref(1);
const salesSearch = ref("");
const filteredSales = computed(() => {
  if (!salesSearch.value) return sales.value;
  const s = salesSearch.value.toLowerCase();
  return sales.value.filter(sale =>
    String(sale.id).includes(s) ||
    sale.status?.toLowerCase().includes(s) ||
    String(sale.total).includes(s)
  );
});
const pagedSales = computed(() => {
  const start = (salesPage.value - 1) * SALES_PAGE_SIZE;
  return filteredSales.value.slice(start, start + SALES_PAGE_SIZE);
});
const salesTotalPages = computed(() => Math.max(1, Math.ceil(filteredSales.value.length / SALES_PAGE_SIZE)));
const salesPageNums = computed(() => {
  const pages: number[] = []; const t = salesTotalPages.value; const c = salesPage.value;
  for (let i = Math.max(1, c - 2); i <= Math.min(t, c + 2); i++) pages.push(i);
  return pages;
});

// Orders search/filter
const ordersSearch = ref("");
const ordersStatusFilter = ref("");
const filteredOrders = computed(() => {
  return orders.value.filter(o => {
    const s = ordersSearch.value.toLowerCase();
    const matchSearch = !s || o.customerName.toLowerCase().includes(s) || String(o.id).includes(s);
    const matchStatus = !ordersStatusFilter.value || o.status === ordersStatusFilter.value;
    return matchSearch && matchStatus;
  });
});

async function sendFiscal(saleId: number) {
  try {
    const receipt = await api.sendReceiptForSale(token.value, saleId);
    fiscalResult.value[saleId] = receipt.status === "sent"
      ? `✓ Фіскальний #${receipt.fiscalNumber}`
      : `⏳ ${receipt.status}`;
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    fiscalResult.value[saleId] = `✗ ${e.message}`;
  }
}

async function completeSale() {
  if (!cartItems.value.length) return;
  saleSaving.value = true; error.value = ""; saleSuccess.value = "";
  try {
    const sale = await api.createSale(token.value, {
      items: cartItems.value.map(c => ({
        productId: c.product.id,
        quantity: c.quantity,
        price: itemEffectivePrice(c)
      })),
      orderId: posOrderId.value,
      currency: posCurrency.value
    });
    saleSuccess.value = `Продаж #${sale.id} на ${sale.total.toFixed(2)} ${sale.currency} — проведено!`;
    cartItems.value = [];
    posOrderId.value = undefined;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saleSaving.value = false; }
}

async function createOrder() {
  saving.value = true; error.value = "";
  try {
    const name = orderForm.value.customerName.trim();
    const exists = customers.value.some(c => c.name.toLowerCase() === name.toLowerCase());
    if (!exists) {
      await api.createCustomer(token.value, { name, phone: "", email: "", comment: "" });
      await load();
    }
    await api.createOrder(token.value, { ...orderForm.value, reserve: true, items: orderFormItems.value.filter(i => i.productId > 0) });
    showNewOrder.value = false;
    orderForm.value = { customerName: "", currency: "UAH", items: [], dueDate: "" };
    orderFormItems.value = [];
    orderCustomerSearch.value = "";
    orderItemProductSearch.value = [];
    showOrderItemProductDropdown.value = [];
    showNewOrderItemProductForm.value = [];
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function updateOrderStatus(orderId: number, status: any) {
  try {
    await api.updateOrderStatus(token.value, orderId, status);
    await load();
    if (selectedOrder.value?.id === orderId) {
      selectedOrder.value = orders.value.find(o => o.id === orderId) || null;
    }
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
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
    if (paymentForm.value.entityType === "sale") payload.saleId = paymentForm.value.entityId;
    else if (paymentForm.value.entityType === "serviceOrder") payload.serviceOrderId = paymentForm.value.entityId;
    else payload.orderId = paymentForm.value.entityId;
    await api.createPayment(token.value, payload);
    showPayment.value = false;
    await load();
    // refresh order detail if open
    if (selectedOrder.value) await openOrderDetail(selectedOrder.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

function openPaymentFor(type: "sale" | "order" | "serviceOrder", id: number, total: number, currency: CurrencyCode) {
  paymentForm.value = { entityType: type as any, entityId: id, cashboxId: cashboxes.value[0]?.id || 0, amount: total, currency, method: "cash", note: "" };
  showPayment.value = true;
}

async function openOrderDetail(order: CustomerOrder) {
  selectedOrder.value = order;
  orderDetailLoading.value = true;
  orderPayments.value = [];
  orderLinkedSales.value = [];
  orderChain.value = null;
  try {
    const [payments, allSales, chain] = await Promise.all([
      api.paymentsForOrder(token.value, order.id),
      api.sales(token.value),
      api.getOrderChain(token.value, order.id).catch(() => null),
    ]);
    orderPayments.value = payments;
    orderLinkedSales.value = allSales.filter((s: Sale) => s.orderId === order.id);
    orderChain.value = chain;
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    orderDetailLoading.value = false;
  }
}

function closeOrderDetail() {
  selectedOrder.value = null;
  orderPayments.value = [];
  orderLinkedSales.value = [];
  orderChain.value = null;
}

const orderPaid = computed(() =>
  orderPayments.value.reduce((a, p) => a + p.amount, 0)
);
const orderDebt = computed(() =>
  (selectedOrder.value?.total || 0) - orderPaid.value
);

// ── Full chain helpers ────────────────────────────────────────
// Supplier orders linked to a customer order (via customerOrderId field)
function chainSupplierOrders(customerOrderId: number): SupplierOrder[] {
  return supplierOrders.value.filter((so: any) => so.customerOrderId === customerOrderId);
}
// Purchases (receipts) linked to a supplier order
function chainPurchases(supplierOrderId: number): Purchase[] {
  return purchases.value.filter((p: any) => p.supplierOrderId === supplierOrderId);
}

// ── Edit existing order ───────────────────────────────────────
function openEditOrder(order: CustomerOrder) {
  editOrderId.value = order.id;
  editOrderForm.value = { customerName: order.customerName, currency: order.currency, dueDate: order.dueDate || "" };
  editOrderItems.value = (order.items || []).map((i: SaleItem) => ({ productId: i.productId, quantity: i.quantity, price: i.price || 0 }));
  showEditOrder.value = true;
}

function addEditOrderItem() {
  editOrderItems.value.push({ productId: 0, quantity: 1, price: 0 });
}
function removeEditOrderItem(idx: number) {
  editOrderItems.value.splice(idx, 1);
}
function onEditOrderItemProduct(idx: number) {
  const item = editOrderItems.value[idx];
  const p = products.value.find(p => p.id === item.productId);
  if (p) item.price = p.retailPrice;
}

async function saveEditOrder() {
  editOrderSaving.value = true; error.value = "";
  try {
    await api.updateOrder(token.value, editOrderId.value, {
      customerName: editOrderForm.value.customerName,
      currency: editOrderForm.value.currency,
      dueDate: editOrderForm.value.dueDate,
      items: editOrderItems.value.filter(i => i.productId > 0),
    });
    showEditOrder.value = false;
    await load();
    // Refresh detail panel if this order is open
    const updated = orders.value.find(o => o.id === editOrderId.value);
    if (updated && selectedOrder.value?.id === editOrderId.value) {
      await openOrderDetail(updated);
    }
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { editOrderSaving.value = false; }
}

// ── Link supplier order to customer order ─────────────────────
function openLinkSupplierOrder(customerOrderId: number) {
  linkTargetCustomerOrderId.value = customerOrderId;
  linkSupplierOrderId.value = 0;
  showLinkSupplierOrder.value = true;
}

async function saveLinkSupplierOrder() {
  if (!linkSupplierOrderId.value) return;
  linkSaving.value = true; error.value = "";
  try {
    await api.updateSupplierOrder(token.value, linkSupplierOrderId.value, {
      customerOrderId: linkTargetCustomerOrderId.value,
    });
    showLinkSupplierOrder.value = false;
    try { supplierOrders.value = await api.supplierOrders(token.value); } catch {}
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { linkSaving.value = false; }
}

async function createQuickProduct() {
  quickProductSaving.value = true;
  try {
    const p = await api.createProduct(token.value, {
      ...quickProductForm.value,
      currency: "UAH",
      stock: 0,
      minStock: 1,
    });
    await load();
    addToCart(p as any);
    showQuickProduct.value = false;
    quickProductForm.value = { name: "", sku: "", category: "", retailPrice: 0, purchasePrice: 0 };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { quickProductSaving.value = false; }
}

const statusLabel: Record<string, string> = {
  new: "Нове", in_work: "В роботі", ordered: "Замовлено у пост.",
  expected: "Очікується", arrived: "Надійшло", issued: "Видано",
  closed: "Закрито", cancelled: "Скасовано",
  // supplier order statuses
  draft: "Створено", sent: "Відправлено", confirmed: "Підтверджено",
  in_transit: "В дорозі", received: "Отримано",
  // legacy
  processing: "В обробці", paid: "Оплачено", completed: "Завершено",
};

const statusColor: Record<string, string> = {
  new: "#94a3b8", in_work: "#60a5fa", ordered: "#a78bfa",
  expected: "#f59e0b", arrived: "#34d399", issued: "#10b981",
  closed: "#6b7280", cancelled: "#f87171",
  // supplier
  draft: "#94a3b8", sent: "#60a5fa", confirmed: "#818cf8",
  in_transit: "#f59e0b", received: "#34d399",
  // legacy
  processing: "#e0c060", paid: "#4ecb8d", completed: "#3cb87a",
};

onMounted(load);
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Продажі</h2>
    </div>

    <div class="tab-row" style="margin-bottom:1rem">
      <button :class="['tab-button', subTab==='pos'&&'tab-button--active']" @click="subTab='pos'">Каса / POS</button>
      <button :class="['tab-button', subTab==='sales'&&'tab-button--active']" @click="subTab='sales'">Продажі</button>

    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <!-- POS -->
    <div v-if="subTab==='pos'" style="display:grid;grid-template-columns:1fr 360px;gap:1rem">
      <div>
        <div style="display:flex;gap:0.5rem;margin-bottom:0.8rem;align-items:center">
          <div style="flex:1">
            <BarcodeScanner
              placeholder="Штрихкод або назва товару..."
              :autofocus="true"
              @scanned="scanToCart"
            />
          </div>
          <button v-if="can('products:write')" class="ghost-button" style="white-space:nowrap;padding:0.5rem 0.8rem;font-size:0.85rem"
            @click="showQuickProduct=true" title="Товару немає — створити прямо зараз">
            + Новий товар
          </button>
        </div>
        <input v-if="posSearch" v-model="posSearch" placeholder="Уточнити пошук..." style="width:100%;margin-bottom:0.8rem">
        <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:0.5rem;max-height:60vh;overflow-y:auto">
          <div v-for="p in posProducts" :key="p.id"
            class="panel" style="padding:0.8rem;cursor:pointer;transition:border-color 0.15s"
            @click="addToCart(p)">
            <div style="font-weight:600;font-size:0.9rem">{{ p.name }}</div>
            <div class="subtle" style="font-size:0.8rem">{{ p.sku }}</div>
            <div style="margin-top:0.4rem;color:#6dd4a0;font-weight:700">{{ p.retailPrice }} {{ p.currency||'UAH' }}</div>
            <div class="subtle" style="font-size:0.78rem">Залишок: {{ p.stock }}</div>
          </div>
        </div>
      </div>

      <!-- Cart -->
      <div>
        <div class="panel" style="padding:1rem">
          <h3 style="margin-top:0">Кошик</h3>
          <div v-if="!cartItems.length" class="subtle" style="margin-bottom:0.8rem">Додайте товари або скануйте штрихкод</div>

          <!-- Cart items with discount -->
          <div v-for="(c, i) in cartItems" :key="i" style="margin-bottom:0.7rem;padding-bottom:0.7rem;border-bottom:1px solid rgba(122,185,154,0.12)">
            <div style="display:flex;align-items:center;gap:0.3rem;margin-bottom:0.3rem">
              <div style="flex:1;font-size:0.88rem;font-weight:500">{{ c.product.name }}</div>
              <button class="ghost-button" style="padding:0.2rem 0.45rem;font-size:0.8rem" @click="removeFromCart(i)">✕</button>
            </div>
            <div style="display:grid;grid-template-columns:60px 80px 72px;gap:0.3rem;align-items:center">
              <label style="font-size:0.75rem;color:var(--text-subtle)">К-сть
                <input type="number" min="1" v-model.number="c.quantity" style="width:100%;padding:0.25rem;font-size:0.85rem">
              </label>
              <label style="font-size:0.75rem;color:var(--text-subtle)">Ціна
                <input type="number" min="0" v-model.number="c.price" style="width:100%;padding:0.25rem;font-size:0.85rem">
              </label>
              <label style="font-size:0.75rem;color:var(--text-subtle)">Знижка %
                <input type="number" min="0" max="100" v-model.number="c.discountPct" style="width:100%;padding:0.25rem;font-size:0.85rem;border-color:rgba(224,192,96,0.4);color:#e0c060">
              </label>
            </div>
            <div style="display:flex;justify-content:space-between;margin-top:0.25rem;font-size:0.82rem">
              <span v-if="c.discountPct > 0" style="color:#e0c060">
                Знижка: -{{ (c.price * (c.discountPct/100) * c.quantity).toFixed(2) }}
              </span>
              <span v-else></span>
              <span style="font-weight:700;color:var(--accent)">{{ (itemEffectivePrice(c) * c.quantity).toFixed(2) }}</span>
            </div>
          </div>

          <hr style="border-color:rgba(100,200,140,0.22);margin:0.6rem 0">

          <div v-if="cartDiscount > 0" style="display:flex;justify-content:space-between;font-size:0.85rem;margin-bottom:0.3rem;color:#e0c060">
            <span>Загальна знижка:</span>
            <span>-{{ cartDiscount.toFixed(2) }}</span>
          </div>
          <div style="display:flex;justify-content:space-between;font-weight:700;margin-bottom:0.8rem;font-size:1.05rem">
            <span>Разом:</span>
            <span style="color:var(--accent)">{{ cartTotal.toFixed(2) }}</span>
          </div>

          <div style="display:grid;gap:0.4rem;margin-bottom:0.8rem">
            <label style="font-size:0.85rem">Валюта
              <select v-model="posCurrency" style="width:100%">
                <option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option>
              </select>
            </label>
            <label style="font-size:0.85rem">Прив'язати до замовлення
              <select v-model="posOrderId" style="width:100%">
                <option :value="undefined">— Без замовлення —</option>
                <option v-for="o in orders.filter(o=>o.status!=='completed'&&o.status!=='cancelled')" :key="o.id" :value="o.id">
                  #{{ o.id }} {{ o.customerName }}
                </option>
              </select>
            </label>
          </div>
          <button style="width:100%" :disabled="saleSaving||!cartItems.length" @click="completeSale">
            {{ saleSaving ? 'Проводимо...' : '✓ Провести продаж' }}
          </button>
          <p v-if="saleSuccess" class="ok-text" style="margin-top:0.5rem;font-size:0.88rem">{{ saleSuccess }}</p>
        </div>

        <!-- ── Pending orders panel ─────────────────────────────── -->
        <div class="panel" style="padding:1rem;margin-top:1rem">
          <div style="font-size:0.75rem;font-weight:600;color:#7ab99a;text-transform:uppercase;letter-spacing:0.06em;margin-bottom:0.75rem">
            🗂 Незакриті замовлення — оплата на касі
          </div>

          <!-- Customer (товарні) orders -->
          <div style="margin-bottom:1rem">
            <div style="font-size:0.7rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.4rem">📦 ТОВАРНІ ЗАМОВЛЕННЯ</div>
            <div v-if="orders.filter(o=>!['completed','cancelled','paid'].includes(o.status)&&(o.total||0)>0&&(debts.find(d=>d.entityType==='customer_order'&&d.entityId===o.id)?.debt??o.total)>0).length === 0"
              class="subtle" style="font-size:0.82rem;padding:0.4rem 0">Немає активних замовлень</div>
            <div v-for="o in orders.filter(o=>!['completed','cancelled','paid'].includes(o.status)&&(o.total||0)>0&&(debts.find(d=>d.entityType==='customer_order'&&d.entityId===o.id)?.debt??o.total)>0)"
              :key="'co-'+o.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.5rem 0.6rem;border-radius:8px;margin-bottom:0.35rem;background:rgba(122,185,154,0.06);border:1px solid rgba(122,185,154,0.15);cursor:pointer;transition:border-color 0.15s"
              @mouseenter="($event.currentTarget as HTMLElement).style.borderColor='rgba(122,185,154,0.4)'"
              @mouseleave="($event.currentTarget as HTMLElement).style.borderColor='rgba(122,185,154,0.15)'"
              @click="openPaymentFor('order', o.id, o.total, o.currency)">
              <div>
                <div style="font-size:0.85rem;font-weight:600">#{{ o.id }} — {{ o.customerName }}</div>
                <div style="font-size:0.75rem;color:var(--text-subtle)">
                  {{ o.items?.length ?? 0 }} поз.
                  <span v-if="o.dueDate" style="margin-left:0.4rem">· до {{ new Date(o.dueDate).toLocaleDateString('uk') }}</span>
                </div>
              </div>
              <div style="text-align:right">
                <div style="font-weight:700;color:var(--accent);font-size:0.95rem">{{ (o.total||0).toFixed(2) }} {{ o.currency }}</div>
                <div style="font-size:0.72rem;padding:0.1rem 0.45rem;border-radius:10px;margin-top:0.15rem;display:inline-block"
                  :style="'background:rgba(90,180,220,0.12);color:#60c0e0'">
                  💳 Оплатити
                </div>
              </div>
            </div>
          </div>

          <!-- Service (ремонт) orders -->
          <div>
            <div style="font-size:0.7rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.4rem">🔧 ЗАМОВЛЕННЯ НА РЕМОНТ</div>
            <div v-if="serviceOrders.filter(o=>o.status!=='cancelled'&&o.debt>0).length === 0"
              class="subtle" style="font-size:0.82rem;padding:0.4rem 0">Немає замовлень з боргом</div>
            <div v-for="so in serviceOrders.filter(o=>o.status!=='cancelled'&&o.debt>0)"
              :key="'so-'+so.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.5rem 0.6rem;border-radius:8px;margin-bottom:0.35rem;background:rgba(224,160,96,0.06);border:1px solid rgba(224,160,96,0.15);cursor:pointer;transition:border-color 0.15s"
              @mouseenter="($event.currentTarget as HTMLElement).style.borderColor='rgba(224,160,96,0.4)'"
              @mouseleave="($event.currentTarget as HTMLElement).style.borderColor='rgba(224,160,96,0.15)'"
              @click="openPaymentFor('serviceOrder', so.id, so.debt, so.currency)">
              <div>
                <div style="font-size:0.85rem;font-weight:600">#{{ so.id }} — {{ so.title }}</div>
                <div style="font-size:0.75rem;color:var(--text-subtle)">
                  <span :style="'padding:0.1rem 0.4rem;border-radius:8px;font-size:0.7rem;background:' + (so.status==='done'?'rgba(72,187,120,0.12)':'rgba(90,180,220,0.12)') + ';color:' + (so.status==='done'?'#6dd4a0':'#60c0e0')">
                    {{ so.status === 'done' ? 'Готово' : so.status === 'in_progress' ? 'В роботі' : 'Нове' }}
                  </span>
                  <span style="margin-left:0.4rem">Всього: {{ so.total.toFixed(2) }}</span>
                  <span v-if="so.paid > 0" style="margin-left:0.4rem;color:#6dd4a0">· Сплачено: {{ so.paid.toFixed(2) }}</span>
                </div>
              </div>
              <div style="text-align:right">
                <div style="font-weight:700;color:#e0a060;font-size:0.95rem">{{ so.debt.toFixed(2) }} {{ so.currency }}</div>
                <div style="font-size:0.72rem;padding:0.1rem 0.45rem;border-radius:10px;margin-top:0.15rem;display:inline-block;background:rgba(224,160,96,0.12);color:#e0a060">
                  💳 Оплатити
                </div>
              </div>
            </div>
          </div>
        </div>
        <!-- ─────────────────────────────────────────────────────── -->

      </div>

      <!-- ── Знайти замовлення та оплатити (права колонка POS) ── -->
      <div style="display:flex;flex-direction:column;gap:0.8rem">
        <div style="padding:1rem;background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.07);border-radius:10px">
          <div style="font-size:0.8rem;font-weight:600;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.7rem">
            💳 Знайти замовлення та оплатити
          </div>
          <input
            v-model="orderSearch"
            placeholder="Пошук по номеру, імені клієнта..."
            style="width:100%;box-sizing:border-box;margin-bottom:0.6rem"
          >
          <div v-if="orderSearch && !orderSearchResults.length" class="subtle" style="font-size:0.85rem;padding:0.5rem 0">
            Нічого не знайдено
          </div>
          <div v-if="orderSearchResults.length" style="display:flex;flex-direction:column;gap:0.4rem">
            <div
              v-for="r in orderSearchResults"
              :key="`${r.type}-${r.id}`"
              style="display:flex;align-items:center;gap:0.6rem;padding:0.5rem 0.7rem;background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.07);border-radius:8px;flex-wrap:wrap"
            >
              <span style="font-size:1rem">{{ quickTypeIcon[r.type] }}</span>
              <div style="flex:1;min-width:0">
                <div style="font-weight:600;font-size:0.83rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{{ r.label }}</div>
                <div class="subtle" style="font-size:0.75rem">{{ r.sub }}</div>
              </div>
              <div style="text-align:right">
                <div v-if="r.total > 0 && r.total - r.paid > 0" style="font-size:0.8rem;color:#ff9ca0;font-weight:700">
                  {{ (r.total - r.paid).toFixed(2) }} UAH
                </div>
                <div v-else-if="r.total > 0" style="font-size:0.8rem;color:#6dd4a0;font-weight:600">✓</div>
                <div v-else-if="r.type === 'supplier_order'" style="font-size:0.8rem;color:var(--text-muted)">сума не вказана</div>
                <div v-else style="font-size:0.8rem;color:var(--text-muted)">—</div>
              </div>
              <button
                v-if="r.total - r.paid > 0 && !['closed','cancelled','paid','completed'].includes(r.status)"
                style="padding:0.3rem 0.7rem;font-size:0.8rem;background:rgba(109,212,160,0.12);border:1px solid rgba(109,212,160,0.35);color:#6dd4a0;border-radius:6px;cursor:pointer;white-space:nowrap"
                @click="openQuickPay(r)"
              >
                💳 Оплатити
              </button>
              <span v-else-if="r.total > 0 || ['closed','paid','completed'].includes(r.status)" style="font-size:0.9rem">✅</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- SALES LIST -->
    <div v-if="subTab==='sales'">
      <div style="display:flex;gap:0.5rem;margin-bottom:1rem;flex-wrap:wrap;align-items:center">
        <input v-model="salesSearch" placeholder="🔍 Пошук (ID, статус, сума)..." style="flex:1;min-width:200px">
        <span class="subtle" style="font-size:0.85rem">{{ filteredSales.length }} записів</span>
      </div>
      <table>
        <thead>
          <tr><th>#</th><th>Дата</th><th>Замовлення</th><th>Сума</th><th>Валюта</th><th>Статус</th><th>Дії</th></tr>
        </thead>
        <tbody>
          <tr v-for="s in pagedSales" :key="s.id" style="cursor:pointer" @click="openSaleDetail(s)">
            <td>{{ s.id }}</td>
            <td>{{ new Date(s.createdAt).toLocaleString('uk') }}</td>
            <td>
              <span v-if="s.orderId" class="chip" style="font-size:0.78rem;cursor:pointer"
                @click="emit('navigate', {type:'customer_order', id:s.orderId!})">
                Замов. #{{ s.orderId }}
              </span>
              <span v-else class="subtle" style="font-size:0.78rem">—</span>
            </td>
            <td style="font-weight:600">{{ s.total.toFixed(2) }}</td>
            <td>{{ s.currency }}</td>
            <td>{{ s.status }}</td>
            <td @click.stop>
              <div style="display:flex;gap:0.4rem;flex-wrap:wrap;align-items:center">
                <button class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem"
                  @click="openSaleDetail(s)">
                  Детально
                </button>
                <button v-if="s.status !== 'paid' && s.status !== 'completed'" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem"
                  @click="openPaymentFor('sale', s.id, s.total, s.currency)">
                  Оплата
                </button>
                <span v-else style="font-size:0.78rem;color:#6dd4a0;font-weight:600">✓ Оплачено</span>
                <button v-if="session.features?.checkboxEnabled" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem"
                  @click="sendFiscal(s.id)" title="Надіслати фіскальний чек до Checkbox">Чек
                </button>
                <span v-if="fiscalResult[s.id]" :style="'font-size:0.78rem;color:' + (fiscalResult[s.id].startsWith('✓') ? '#6dd4a0' : '#e07070')">
                  {{ fiscalResult[s.id] }}
                </span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="pagination" v-if="salesTotalPages > 1">
        <button class="pagination__btn" :disabled="salesPage===1" @click="salesPage=1">«</button>
        <button class="pagination__btn" :disabled="salesPage===1" @click="salesPage--">‹</button>
        <button v-for="n in salesPageNums" :key="n"
          :class="['pagination__btn', n===salesPage && 'pagination__btn--active']"
          @click="salesPage=n">{{ n }}</button>
        <button class="pagination__btn" :disabled="salesPage===salesTotalPages" @click="salesPage++">›</button>
        <button class="pagination__btn" :disabled="salesPage===salesTotalPages" @click="salesPage=salesTotalPages">»</button>
        <span class="subtle" style="font-size:0.82rem;margin-left:0.3rem">
          {{ (salesPage-1)*SALES_PAGE_SIZE+1 }}–{{ Math.min(salesPage*SALES_PAGE_SIZE, filteredSales.length) }} з {{ filteredSales.length }}
        </span>
      </div>
    </div>

    </div>

    <!-- MODAL: Quick Pay (order search) -->
    <div v-if="showQuickPay && quickPayTarget" class="modal-backdrop" @click.self="showQuickPay=false">
      <div class="panel modal-box" style="max-width:480px">
        <h3>💳 Оплата замовлення</h3>
        <div style="margin-bottom:1rem;padding:0.8rem;background:rgba(255,255,255,0.04);border-radius:8px">
          <div style="font-size:0.75rem;color:var(--text-muted);margin-bottom:0.25rem">{{ quickPayTarget.label }}</div>
          <div style="display:flex;justify-content:space-between;font-size:0.9rem;flex-wrap:wrap;gap:0.3rem">
            <span>Всього: <strong>{{ quickPayTarget.total.toFixed(2) }} UAH</strong></span>
            <span>Оплачено: <span style="color:#6dd4a0">{{ quickPayTarget.paid.toFixed(2) }}</span></span>
            <span>Борг: <strong style="color:#ff9ca0">{{ (quickPayTarget.total - quickPayTarget.paid).toFixed(2) }} UAH</strong></span>
          </div>
        </div>
        <div class="grid">
          <label>Каса
            <select v-model.number="quickPayForm.cashboxId">
              <option value="0" disabled>Оберіть касу</option>
              <option v-for="cb in cashboxes" :key="cb.id" :value="cb.id">{{ cb.name }} ({{ cb.currency }})</option>
            </select>
          </label>
          <label>Сума (UAH)
            <input type="number" min="0" step="0.01" v-model.number="quickPayForm.amount">
          </label>
          <label>Метод оплати
            <select v-model="quickPayForm.method">
              <option value="cash">Готівка</option>
              <option value="card">Картка</option>
              <option value="bank">Банківський переказ</option>
            </select>
          </label>
          <label style="grid-column:1/-1">Примітка
            <input v-model="quickPayForm.note" :placeholder="`Оплата: ${quickPayTarget.label}`">
          </label>
        </div>
        <div v-if="quickPayForm.amount > 0" style="margin-top:0.8rem;padding:0.6rem;background:rgba(109,212,160,0.08);border-radius:6px;border:1px solid rgba(109,212,160,0.2);font-size:0.85rem">
          Борг після оплати:
          <strong :style="(quickPayTarget.total - quickPayTarget.paid - quickPayForm.amount) <= 0 ? 'color:#6dd4a0' : 'color:#ff9ca0'">
            {{ Math.max(0, quickPayTarget.total - quickPayTarget.paid - quickPayForm.amount).toFixed(2) }} UAH
          </strong>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="submitQuickPay" :disabled="quickPaying||!quickPayForm.cashboxId||!quickPayForm.amount||quickPayForm.amount<=0">
            {{ quickPaying ? 'Проведення...' : '✓ Провести оплату' }}
          </button>
          <button class="ghost-button" @click="showQuickPay=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: New Order -->
    <div v-if="showNewOrder" class="modal-backdrop" @click.self="showNewOrder=false; orderCustomerSearch=''; orderForm.customerName=''">
      <div class="panel modal-box">
        <h3>Нове замовлення покупця</h3>
        <div class="grid">
          <label>Клієнт *
            <div style="display:flex;gap:0.4rem;align-items:flex-start">
              <div style="flex:1;position:relative">
                <input v-model="orderCustomerSearch" placeholder="Пошук за іменем, телефоном..."
                  style="width:100%"
                  @focus="showOrderCustomerDropdown=true"
                  @input="showOrderCustomerDropdown=true; orderForm.customerName=orderCustomerSearch">
                <div v-if="showOrderCustomerDropdown && filteredOrderCustomers.length > 0"
                  style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                  <div v-for="c in filteredOrderCustomers" :key="c.id"
                    style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.87rem;border-bottom:1px solid rgba(72,187,120,0.08)"
                    @mousedown.prevent="selectOrderCustomer(c)"
                    @mouseover="($event.currentTarget as HTMLElement).style.background='rgba(72,187,120,0.12)'"
                    @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                    <span style="font-weight:600">{{ c.name }}</span>
                    <span v-if="c.phone" class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ c.phone }}</span>
                  </div>
                </div>
              </div>
              <button class="ghost-button" style="white-space:nowrap;font-size:0.8rem;padding:0.4rem 0.7rem"
                @click="showNewOrderCustomerForm=!showNewOrderCustomerForm; showOrderCustomerDropdown=false">
                {{ showNewOrderCustomerForm ? '✕' : '+ Новий' }}
              </button>
            </div>
            <div v-if="orderForm.customerName && !showOrderCustomerDropdown" style="font-size:0.78rem;color:#6dd4a0;margin-top:0.2rem">
              ✓ Обрано: {{ orderForm.customerName }}
            </div>
          </label>

          <!-- Quick create customer -->
          <div v-if="showNewOrderCustomerForm"
            style="background:rgba(72,187,120,0.06);border:1px solid rgba(72,187,120,0.2);border-radius:8px;padding:0.75rem">
            <div style="font-size:0.8rem;font-weight:600;color:#7ab99a;margin-bottom:0.5rem">Новий клієнт</div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
              <label style="font-size:0.78rem">Ім'я *<input v-model="newOrderCustomerForm.name" class="input" placeholder="Ім'я" style="margin-top:0.2rem"></label>
              <label style="font-size:0.78rem">Телефон<input v-model="newOrderCustomerForm.phone" class="input" placeholder="+380..." style="margin-top:0.2rem"></label>
            </div>
            <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="newOrderCustomerSaving||!newOrderCustomerForm.name.trim()"
              @click="saveNewOrderCustomer">
              {{ newOrderCustomerSaving ? '..' : '✓ Зберегти клієнта' }}
            </button>
          </div>

          <label>Валюта
            <select v-model="orderForm.currency"><option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option></select>
          </label>
          <label>Термін виконання <input type="date" v-model="orderForm.dueDate"></label>
        </div>

        <div style="margin-top:1rem">
          <div style="font-size:0.85rem;font-weight:600;margin-bottom:0.5rem;color:#7ab99a">Товари</div>
          <div v-for="(item, idx) in orderFormItems" :key="idx"
            style="background:rgba(72,187,120,0.05);border:1px solid rgba(72,187,120,0.15);border-radius:8px;padding:0.6rem;margin-bottom:0.5rem">
            <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.4rem">
              <span style="font-size:0.78rem;color:var(--text-muted)">Позиція {{ idx+1 }}</span>
              <button class="ghost-button" style="padding:0.15rem 0.45rem;font-size:0.8rem;color:var(--danger)" @click="removeOrderItem(idx)">✕</button>
            </div>
            <!-- Product search per item -->
            <div style="position:relative;margin-bottom:0.4rem">
              <div v-if="item.productId > 0"
                style="display:flex;align-items:center;gap:0.4rem;padding:0.4rem 0.6rem;background:rgba(72,187,120,0.1);border-radius:6px;font-size:0.85rem">
                <span style="flex:1;font-weight:500">{{ products.find(p=>p.id===item.productId)?.name }}</span>
                <button class="ghost-button" style="padding:0.1rem 0.4rem;font-size:0.75rem"
                  @click="item.productId=0; orderItemProductSearch[idx]=''; showOrderItemProductDropdown[idx]=true">змінити</button>
              </div>
              <div v-else>
                <input
                  :value="orderItemProductSearch[idx] || ''"
                  class="input" placeholder="Пошук товару за назвою, SKU..." style="width:100%"
                  @focus="showOrderItemProductDropdown[idx]=true; activeOrderItemIdx=idx"
                  @input="orderItemProductSearch[idx]=($event.target as HTMLInputElement).value; showOrderItemProductDropdown[idx]=true; activeOrderItemIdx=idx"
                />
                <div v-if="showOrderItemProductDropdown[idx]"
                  style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                  <div v-for="p in filteredOrderItemProducts(idx)" :key="p.id"
                    style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.84rem;border-bottom:1px solid rgba(72,187,120,0.08)"
                    @mousedown.prevent="selectOrderItemProduct(p, idx)"
                    @mouseover="($event.currentTarget as HTMLElement).style.background='rgba(72,187,120,0.12)'"
                    @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                    <span style="font-weight:600">{{ p.name }}</span>
                    <span class="subtle" style="font-size:0.75rem;margin-left:0.4rem">{{ p.sku }}</span>
                    <span style="float:right;color:var(--accent);font-size:0.8rem">{{ p.retailPrice }} {{ p.currency }}</span>
                  </div>
                  <div style="padding:0.4rem 0.7rem;border-top:1px solid rgba(72,187,120,0.15)">
                    <button class="ghost-button" style="font-size:0.78rem;width:100%"
                      @mousedown.prevent="showNewOrderItemProductForm[idx]=true; showOrderItemProductDropdown[idx]=false; activeOrderItemIdx=idx">
                      + Створити новий товар
                    </button>
                  </div>
                </div>
              </div>
            </div>
            <!-- Quick create product per item -->
            <div v-if="showNewOrderItemProductForm[idx]"
              style="background:rgba(96,192,224,0.06);border:1px solid rgba(96,192,224,0.2);border-radius:8px;padding:0.6rem;margin-bottom:0.4rem">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.4rem">
                <span style="font-size:0.78rem;font-weight:600;color:#60c0e0">Новий товар</span>
                <button class="ghost-button" style="font-size:0.78rem" @click="showNewOrderItemProductForm[idx]=false">✕</button>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.4rem;margin-bottom:0.4rem">
                <label style="font-size:0.75rem">Назва *<input v-model="newOrderItemProductForm.name" class="input" placeholder="Назва" style="margin-top:0.15rem"></label>
                <label style="font-size:0.75rem">SKU<input v-model="newOrderItemProductForm.sku" class="input" style="margin-top:0.15rem"></label>
              </div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.4rem;margin-bottom:0.4rem">
                <label style="font-size:0.75rem">Категорія<input v-model="newOrderItemProductForm.category" class="input" style="margin-top:0.15rem"></label>
                <label style="font-size:0.75rem">Ціна продажу<input type="number" v-model.number="newOrderItemProductForm.retailPrice" min="0" class="input" style="margin-top:0.15rem"></label>
              </div>
              <button style="font-size:0.78rem;padding:0.3rem 0.7rem" :disabled="newOrderItemProductSaving||!newOrderItemProductForm.name.trim()"
                @click="saveNewOrderItemProduct(idx)">
                {{ newOrderItemProductSaving ? '..' : '✓ Зберегти товар' }}
              </button>
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
          <button class="ghost-button" @click="addOrderItem" style="margin-top:0.3rem;font-size:0.85rem">+ Додати товар</button>
        </div>

        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="createOrder" :disabled="saving||!orderForm.customerName||orderFormItems.filter(i=>i.productId>0).length===0">{{ saving?'..':'Створити' }}</button>
          <button class="ghost-button" @click="showNewOrder=false; orderCustomerSearch=''; orderForm.customerName=''">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Quick Create Product -->
    <div v-if="showQuickProduct" class="modal-backdrop" @click.self="showQuickProduct=false">
      <div class="panel modal-box" style="max-width:440px">
        <h3>Новий товар — швидке створення</h3>
        <p class="subtle" style="font-size:0.85rem;margin-top:-0.3rem">Товар буде створено і одразу додано до кошика</p>
        <div class="grid">
          <label>Назва * <input v-model="quickProductForm.name" placeholder="Назва товару" autofocus></label>
          <label>SKU <input v-model="quickProductForm.sku" placeholder="Артикул (можна залишити порожнім)"></label>
          <label>Категорія <input v-model="quickProductForm.category" placeholder="Категорія"></label>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem">
            <label>Ціна закупівлі <input type="number" v-model.number="quickProductForm.purchasePrice" min="0"></label>
            <label>Ціна продажу * <input type="number" v-model.number="quickProductForm.retailPrice" min="0"></label>
          </div>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="createQuickProduct" :disabled="quickProductSaving||!quickProductForm.name||!quickProductForm.retailPrice">
            {{ quickProductSaving ? 'Створення...' : '+ Створити і додати до кошика' }}
          </button>
          <button class="ghost-button" @click="showQuickProduct=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Payment -->
    <div v-if="showPayment" class="modal-backdrop" @click.self="showPayment=false">
      <div class="panel modal-box">
        <h3>Провести оплату</h3>
        <div class="grid">
          <label>Каса
            <select v-model.number="paymentForm.cashboxId">
              <option v-for="c in cashboxes" :key="c.id" :value="c.id">{{ c.name }} ({{ c.currency }})</option>
            </select>
          </label>
          <label>Сума <input type="number" min="0" v-model.number="paymentForm.amount"></label>
          <label>Валюта
            <select v-model="paymentForm.currency"><option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option></select>
          </label>
          <label>Метод
            <select v-model="paymentForm.method">
              <option value="cash">Готівка</option>
              <option value="card">Картка</option>
              <option value="bank">Банк</option>
            </select>
          </label>
          <label>Примітка <input v-model="paymentForm.note"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="doPayment" :disabled="saving">{{ saving?'..':'Провести' }}</button>
          <button class="ghost-button" @click="showPayment=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>
    <!-- MODAL: Edit Order -->
    <div v-if="showEditOrder" class="modal-backdrop" @click.self="showEditOrder=false">
      <div class="panel modal-box" style="max-width:580px">
        <h3>Редагувати замовлення #{{ editOrderId }}</h3>
        <div class="grid">
          <label>Ім'я клієнта
            <div style="position:relative">
              <input v-model="editOrderForm.customerName" placeholder="Клієнт" style="width:100%"
                list="edit-customers-list">
              <datalist id="edit-customers-list">
                <option v-for="c in customers" :key="c.id" :value="c.name" />
              </datalist>
            </div>
          </label>
          <label>Валюта
            <select v-model="editOrderForm.currency">
              <option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option>
            </select>
          </label>
          <label>Термін виконання <input type="date" v-model="editOrderForm.dueDate"></label>
        </div>

        <div style="margin-top:1rem">
          <div style="font-size:0.85rem;font-weight:600;margin-bottom:0.5rem;color:#7ab99a">Товари</div>
          <div v-for="(item, idx) in editOrderItems" :key="idx"
            style="display:grid;grid-template-columns:2fr 80px 100px 32px;gap:0.4rem;margin-bottom:0.4rem;align-items:center">
            <select v-model.number="item.productId" @change="onEditOrderItemProduct(idx)">
              <option :value="0" disabled>— Оберіть товар —</option>
              <option v-for="p in products.filter(p=>!p.archived)" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
            <input type="number" v-model.number="item.quantity" min="1" placeholder="К-сть">
            <input type="number" v-model.number="item.price" min="0" placeholder="Ціна">
            <button class="ghost-button" @click="removeEditOrderItem(idx)" style="padding:0.3rem 0.5rem">✕</button>
          </div>
          <button class="ghost-button" @click="addEditOrderItem" style="margin-top:0.3rem;font-size:0.85rem">+ Додати товар</button>
        </div>

        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="saveEditOrder" :disabled="editOrderSaving||!editOrderForm.customerName">
            {{ editOrderSaving ? 'Збереження...' : '💾 Зберегти зміни' }}
          </button>
          <button class="ghost-button" @click="showEditOrder=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Link supplier order to customer order -->
    <div v-if="showLinkSupplierOrder" class="modal-backdrop" @click.self="showLinkSupplierOrder=false">
      <div class="panel modal-box" style="max-width:440px">
        <h3>Прив'язати замовлення постачальнику</h3>
        <p class="subtle" style="font-size:0.85rem;margin-top:-0.3rem">
          До замовлення покупця #{{ linkTargetCustomerOrderId }}
        </p>
        <label style="display:block;margin-top:1rem">Замовлення постачальнику
          <select v-model.number="linkSupplierOrderId" style="width:100%;margin-top:0.3rem">
            <option :value="0" disabled>— Оберіть замовлення —</option>
            <option v-for="so in supplierOrders" :key="so.id" :value="so.id">
              #{{ so.id }} — {{ (so as any).supplierName || `Постач. #${so.supplierId}` }} — {{ (so.total||0).toFixed(2) }} {{ so.currency }}
            </option>
          </select>
        </label>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="saveLinkSupplierOrder" :disabled="linkSaving||!linkSupplierOrderId">
            {{ linkSaving ? '...' : '🔗 Прив\'язати' }}
          </button>
          <button class="ghost-button" @click="showLinkSupplierOrder=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Sale Detail ──────────────────────────────────────────────────── -->
    <div v-if="selectedSale"
      style="position:fixed;inset:0;background:rgba(0,0,0,0.6);z-index:200;display:flex;align-items:flex-start;justify-content:center;padding:2rem 1rem;overflow-y:auto"
      @click.self="closeSaleDetail">
      <div style="background:var(--bg-card);border-radius:12px;border:1px solid var(--border);width:100%;max-width:680px;padding:1.5rem;position:relative">

        <!-- Header -->
        <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:1.2rem">
          <div>
            <h2 style="margin:0 0 0.25rem">🧾 Продаж #{{ selectedSale.id }}</h2>
            <div style="display:flex;gap:0.5rem;flex-wrap:wrap">
              <span class="chip" style="font-size:0.75rem">
                {{ new Date(selectedSale.createdAt).toLocaleString('uk', {day:'2-digit',month:'long',year:'numeric',hour:'2-digit',minute:'2-digit'}) }}
              </span>
              <span class="chip" :style="'font-size:0.75rem;background:rgba(72,187,120,0.12);color:' + (selectedSale.status==='completed'||selectedSale.status==='paid'?'#6dd4a0':'#e0c060')">
                {{ selectedSale.status }}
              </span>
              <span v-if="selectedSale.currency" class="chip" style="font-size:0.75rem">{{ selectedSale.currency }}</span>
            </div>
          </div>
          <button class="ghost-button" style="padding:0.3rem 0.7rem;flex-shrink:0" @click="closeSaleDetail">✕</button>
        </div>

        <div v-if="saleDetailLoading" class="subtle" style="text-align:center;padding:2rem">Завантаження...</div>
        <div v-else>

          <!-- Info grid -->
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-bottom:1.2rem">

            <!-- Client -->
            <div class="panel" style="padding:1rem">
              <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.6rem">👤 КЛІЄНТ</div>
              <div v-if="saleDetailOrder">
                <div style="font-weight:700;font-size:1rem;margin-bottom:0.2rem">{{ saleDetailOrder.customerName }}</div>
                <div class="subtle" style="font-size:0.8rem">Замовлення #{{ saleDetailOrder.id }}</div>
                <div v-if="saleDetailOrder.dueDate" class="subtle" style="font-size:0.78rem;margin-top:0.2rem">
                  Термін: {{ new Date(saleDetailOrder.dueDate).toLocaleDateString('uk') }}
                </div>
              </div>
              <div v-else class="subtle" style="font-size:0.88rem">Роздрібний продаж (без замовлення)</div>
            </div>

            <!-- Summary -->
            <div class="panel" style="padding:1rem">
              <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.6rem">💰 СУМА</div>
              <div style="font-size:1.5rem;font-weight:800;color:var(--accent);margin-bottom:0.3rem">
                {{ selectedSale.total.toFixed(2) }} {{ selectedSale.currency }}
              </div>
              <div v-if="saleDetailPayments.length > 0">
                <div style="font-size:0.82rem;color:#6dd4a0">Оплачено: {{ saleDetailPaid.toFixed(2) }} {{ selectedSale.currency }}</div>
                <div v-if="saleDetailPaid < selectedSale.total" style="font-size:0.82rem;color:#e07070">
                  Борг: {{ (selectedSale.total - saleDetailPaid).toFixed(2) }} {{ selectedSale.currency }}
                </div>
              </div>
            </div>
          </div>

          <!-- Items -->
          <div class="panel" style="padding:1rem;margin-bottom:1rem">
            <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.75rem">📦 ТОВАРИ</div>
            <table style="font-size:0.87rem">
              <thead>
                <tr>
                  <th style="text-align:left">Товар</th>
                  <th style="text-align:right">Ціна</th>
                  <th style="text-align:right">К-сть</th>
                  <th style="text-align:right">Сума</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in selectedSale.items" :key="item.productId"
                  style="border-bottom:1px solid rgba(122,185,154,0.08)">
                  <td style="padding:0.4rem 0">
                    <div style="font-weight:600">
                      {{ products.find(p => p.id === item.productId)?.name || `Товар #${item.productId}` }}
                    </div>
                    <div class="subtle" style="font-size:0.75rem;font-family:monospace">
                      {{ products.find(p => p.id === item.productId)?.sku || '' }}
                    </div>
                  </td>
                  <td style="text-align:right;padding:0.4rem 0.5rem">{{ item.price.toFixed(2) }}</td>
                  <td style="text-align:right;padding:0.4rem 0.5rem">{{ item.quantity }}</td>
                  <td style="text-align:right;padding:0.4rem 0;font-weight:700;color:var(--accent)">
                    {{ (item.price * item.quantity).toFixed(2) }}
                  </td>
                </tr>
              </tbody>
              <tfoot>
                <tr>
                  <td colspan="3" style="text-align:right;padding:0.5rem 0.5rem 0;font-size:0.85rem;color:var(--text-subtle)">Разом:</td>
                  <td style="text-align:right;padding:0.5rem 0 0;font-weight:800;font-size:1rem;color:var(--accent)">
                    {{ selectedSale.total.toFixed(2) }} {{ selectedSale.currency }}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>

          <!-- Payments -->
          <div v-if="saleDetailPayments.length > 0" class="panel" style="padding:1rem;margin-bottom:1rem">
            <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.75rem">💳 ОПЛАТИ</div>
            <div v-for="pay in saleDetailPayments" :key="pay.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:1px solid rgba(122,185,154,0.08);font-size:0.87rem">
              <div>
                <span style="font-weight:600">{{ pay.method === 'cash' ? '💵 Готівка' : pay.method === 'card' ? '💳 Картка' : '🏦 Банк' }}</span>
                <span v-if="pay.note" class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ pay.note }}</span>
              </div>
              <div>
                <span style="font-weight:700;color:#6dd4a0">{{ pay.amount.toFixed(2) }} {{ pay.currency }}</span>
                <span class="subtle" style="font-size:0.75rem;margin-left:0.5rem">
                  {{ new Date(pay.createdAt).toLocaleString('uk', {day:'2-digit',month:'short',hour:'2-digit',minute:'2-digit'}) }}
                </span>
              </div>
            </div>
          </div>

          <!-- Receipt / Fiscal -->
          <div v-if="saleDetailReceipts.length > 0" class="panel" style="padding:1rem;margin-bottom:1rem">
            <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.75rem">🧾 ФІСКАЛЬНІ ЧЕКИ</div>
            <div v-for="rec in saleDetailReceipts" :key="rec.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;font-size:0.87rem">
              <div>
                <span style="font-weight:600">Чек #{{ rec.id }}</span>
                <span class="subtle" style="font-size:0.78rem;margin-left:0.5rem">{{ rec.provider }}</span>
                <span v-if="rec.fiscalNumber" style="font-size:0.78rem;font-family:monospace;color:#7ab99a;margin-left:0.5rem">
                  {{ rec.fiscalNumber }}
                </span>
              </div>
              <span :style="'font-size:0.8rem;padding:0.15rem 0.5rem;border-radius:10px;background:' + (rec.status==='sent'?'rgba(72,187,120,0.15)':'rgba(224,160,96,0.15)') + ';color:' + (rec.status==='sent'?'#6dd4a0':'#e0a060')">
                {{ rec.status === 'sent' ? '✓ Надіслано' : rec.status === 'failed' ? '✗ Помилка' : '⏳ Очікує' }}
              </span>
            </div>
          </div>

          <!-- Fiscal result banner -->
          <div v-if="fiscalResult[selectedSale.id]"
            :style="'font-size:0.85rem;padding:0.5rem 0.75rem;border-radius:8px;margin-bottom:1rem;background:' + (fiscalResult[selectedSale.id].startsWith('✓')?'rgba(72,187,120,0.1)':'rgba(224,112,112,0.1)')">
            {{ fiscalResult[selectedSale.id] }}
          </div>

          <!-- Actions -->
          <div style="display:flex;gap:0.5rem;justify-content:flex-end;flex-wrap:wrap">
            <button v-if="selectedSale!.status !== 'paid' && selectedSale!.status !== 'completed'"
              class="ghost-button"
              @click="openPaymentFor('sale', selectedSale!.id, selectedSale!.total, selectedSale!.currency); closeSaleDetail()">
              💳 Оплата
            </button>
            <button v-if="session.features?.checkboxEnabled" class="ghost-button"
              @click="sendFiscal(selectedSale!.id)">
              🧾 Надіслати чек
            </button>
            <button v-if="saleDetailOrder" class="ghost-button"
              @click="emit('navigate', {type:'customer_order', id:saleDetailOrder!.id})">
              📋 Замовлення #{{ saleDetailOrder.id }}
            </button>
            <button class="ghost-button" @click="closeSaleDetail">Закрити</button>
          </div>
        </div>
      </div>

      <!-- Right: order detail + document chain -->
      <div v-if="selectedOrder" style="position:sticky;top:1rem">
        <div style="background:var(--panel-bg);border:1px solid var(--border);border-radius:12px;padding:1rem">

          <!-- Header -->
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.75rem;border-bottom:1px solid var(--border);padding-bottom:0.6rem">
            <div>
              <div style="font-size:0.95rem;font-weight:700">🛍 Замовлення #{{ selectedOrder.id }}</div>
              <div style="font-size:0.82rem;color:var(--text-muted);margin-top:0.1rem">{{ selectedOrder.customerName }}</div>
            </div>
            <button class="ghost-button" style="padding:0.2rem 0.5rem;font-size:1rem" @click="closeOrderDetail">✕</button>
          </div>

          <!-- Summary row -->
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.4rem 1rem;font-size:0.83rem;margin-bottom:0.75rem">
            <div><span class="subtle">Статус:</span>
              <span style="margin-left:0.3rem;font-size:0.75rem;padding:0.15rem 0.5rem;border-radius:10px;font-weight:600"
                :style="`background:${statusColor[selectedOrder.status]||'#64748b'}22;color:${statusColor[selectedOrder.status]||'#94a3b8'}`">
                {{ statusLabel[selectedOrder.status] ?? selectedOrder.status }}
              </span>
            </div>
            <div><span class="subtle">Дата:</span> {{ new Date(selectedOrder.createdAt).toLocaleDateString('uk') }}</div>
            <div><span class="subtle">Сума:</span> <strong>{{ selectedOrder.total.toFixed(2) }} {{ selectedOrder.currency }}</strong></div>
            <div><span class="subtle">Оплачено:</span> <span style="color:#6dd4a0">{{ orderPaid.toFixed(2) }}</span></div>
            <div><span class="subtle">Борг:</span>
              <span :style="orderDebt > 0 ? 'color:#ff9ca0;font-weight:700' : 'color:var(--text-muted)'">{{ orderDebt.toFixed(2) }} {{ selectedOrder.currency }}</span>
            </div>
            <div v-if="selectedOrder.dueDate"><span class="subtle">До видачі:</span> {{ new Date(selectedOrder.dueDate).toLocaleDateString('uk') }}</div>
          </div>

          <!-- Items -->
          <div v-if="selectedOrder.items?.length" style="border-top:1px solid var(--border);padding-top:0.6rem;margin-bottom:0.75rem">
            <div style="font-size:0.78rem;font-weight:600;color:var(--accent);margin-bottom:0.35rem">📦 Товари</div>
            <table style="font-size:0.8rem">
              <thead><tr><th>Товар</th><th>К-сть</th><th>Ціна</th><th>Сума</th></tr></thead>
              <tbody>
                <tr v-for="item in selectedOrder.items" :key="item.id ?? item.productId">
                  <td>{{ products.find(p=>p.id===item.productId)?.name ?? `#${item.productId}` }}</td>
                  <td>{{ item.quantity }}</td>
                  <td>{{ item.price }}</td>
                  <td>{{ (item.quantity * item.price).toFixed(2) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Loading -->
          <div v-if="orderDetailLoading" style="text-align:center;padding:1rem;color:var(--text-muted);font-size:0.85rem">
            ⏳ Завантаження дерева документів...
          </div>

          <!-- Document chain tree -->
          <div v-else-if="orderChain" style="border-top:1px solid var(--border);padding-top:0.75rem;margin-bottom:0.75rem">
            <div style="font-size:0.82rem;font-weight:700;color:var(--accent);margin-bottom:0.6rem">🌳 Дерево документів</div>
            <OrderChainTree :node="orderChain.root" :emit-navigate="(p:any) => emit('navigate', p)" />
          </div>

          <!-- Payments list -->
          <div v-if="orderPayments.length" style="border-top:1px solid var(--border);padding-top:0.6rem;margin-bottom:0.75rem">
            <div style="font-size:0.78rem;font-weight:600;color:#6dd4a0;margin-bottom:0.35rem">💳 Оплати</div>
            <div v-for="p in orderPayments" :key="p.id"
              style="display:flex;justify-content:space-between;align-items:center;font-size:0.8rem;padding:0.25rem 0;border-bottom:1px solid rgba(255,255,255,0.04)">
              <span class="subtle">{{ new Date(p.createdAt).toLocaleDateString('uk') }} · {{ p.method }}</span>
              <span style="font-weight:600;color:#6dd4a0">+{{ p.amount.toFixed(2) }} {{ p.currency }}</span>
            </div>
          </div>

          <!-- Linked sales -->
          <div v-if="orderLinkedSales.length" style="border-top:1px solid var(--border);padding-top:0.6rem;margin-bottom:0.75rem">
            <div style="font-size:0.78rem;font-weight:600;color:#a78bfa;margin-bottom:0.35rem">🧾 Продажі</div>
            <div v-for="s in orderLinkedSales" :key="s.id"
              style="display:flex;justify-content:space-between;align-items:center;font-size:0.8rem;padding:0.25rem 0;border-bottom:1px solid rgba(255,255,255,0.04)">
              <span class="subtle">#{{ s.id }} · {{ new Date(s.createdAt).toLocaleDateString('uk') }}</span>
              <span style="font-weight:600">{{ s.total.toFixed(2) }} {{ s.currency }}</span>
            </div>
          </div>

          <!-- Actions -->
          <div style="border-top:1px solid var(--border);padding-top:0.6rem;display:flex;gap:0.4rem;flex-wrap:wrap">
            <button v-if="orderDebt > 0" class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem;color:#6dd4a0"
              @click="openPaymentFor('order', selectedOrder.id, orderDebt, selectedOrder.currency)">
              💳 Оплата
            </button>
            <button class="ghost-button" style="font-size:0.8rem;padding:0.3rem 0.7rem"
              @click="openEditOrder(selectedOrder)">
              ✏ Редагувати
            </button>
            <button class="ghost-button" @click="closeOrderDetail" style="font-size:0.8rem;padding:0.3rem 0.7rem">Закрити</button>
          </div>
        </div>
      </div>

      <!-- Placeholder when nothing selected -->
      <div v-else style="background:var(--panel-bg);border:1px dashed var(--border);border-radius:12px;padding:2rem;text-align:center;color:var(--text-muted);font-size:0.85rem">
        Оберіть замовлення зі списку щоб переглянути деталі та дерево документів
      </div>
    </div>
</template>