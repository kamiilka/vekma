<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { SESSION_EXPIRED, api } from "../api";
import type {
  AppUser,
  AuditLog,
  CashOperation,
  CashShift,
  CashShiftStatus,
  Cashbox,
  Customer,
  CustomerReminder,
  CurrencyCode,
  Document,
  DocumentCSVImportResult,
  DocumentItem,
  DocumentTemplate,
  DocumentTemplatePreview,
  DocumentTemplateValidation,
  PaymentMethod,
  Payment,
  Product,
  ProductCSVImportResult,
  ProductPriceBulkUpdateRequest,
  Purchase,
  PurchaseCSVImportResult,
  PurchaseRecommendationCreateOrderRequest,
  PurchaseRecommendationGroup,
  ReceiptBulkRetryResult,
  Receipt,
  ReceiptStatus,
  ReturnAvailability,
  RolePermissions,
  Sale,
  ServiceOrder,
  ServiceOrderStatus,
  StockMovement,
  Supplier,
  TemplatePlaceholder,
  Warehouse,
  UserSession
} from "../types";

const props = defineProps<{
  session: UserSession;
}>();

const emit = defineEmits<{
  sessionExpired: [];
}>();

const auditLogs = ref<AuditLog[]>([]);
const auditLifecycleActions = new Set<string>([
  "service_order.status.update",
  "service_order.reopen",
  "service_order.auto_act.create"
]);
const auditFilter = ref({
  action: "",
  search: "",
  onlyServiceLifecycle: false
});
const adminUsers = ref<AppUser[]>([]);
const rolePermissions = ref<RolePermissions[]>([]);
const allPermissions = ref<string[]>([]);
const roleDrafts = ref<Record<string, string[]>>({});
const documentTemplates = ref<DocumentTemplate[]>([]);
const templatePlaceholders = ref<TemplatePlaceholder[]>([]);
const selectedTemplateCode = ref("");
const templateForm = ref({
  code: "",
  name: "",
  body: "",
  isActive: true
});
const templateStrict = ref(true);
const templateValidation = ref<DocumentTemplateValidation | null>(null);
const templatePreview = ref<DocumentTemplatePreview | null>(null);
const suppliers = ref<Supplier[]>([]);
const recommendationGroups = ref<PurchaseRecommendationGroup[]>([]);
const plannerCurrency = ref<CurrencyCode>("UAH");
const plannerSelections = ref<Record<string, { enabled: boolean; quantity: number; price: string }>>({});
const plannerResultText = ref("");
const cashboxes = ref<Cashbox[]>([]);
const cashOperations = ref<CashOperation[]>([]);
const cashShifts = ref<CashShift[]>([]);
const payments = ref<Payment[]>([]);
const customers = ref<Customer[]>([]);
const customerReminders = ref<CustomerReminder[]>([]);
const serviceOrders = ref<ServiceOrder[]>([]);
const serviceOrderActs = ref<Document[]>([]);
const customerSearch = ref("");
const newCustomer = ref({
  name: "",
  phone: "",
  email: "",
  comment: ""
});
const reminderForm = ref({
  customerId: 0,
  text: "",
  dueAt: ""
});
const serviceOrderForm = ref({
  customerId: 0,
  productId: 0,
  title: "",
  description: "",
  technician: "",
  laborMin: 0,
  price: 0,
  currency: "UAH" as CurrencyCode
});
const serviceOrderFilter = ref<{ customerId: number; status: "" | ServiceOrderStatus }>({
  customerId: 0,
  status: ""
});
const serviceOrderEditForm = ref({
  orderId: 0,
  productId: 0,
  title: "",
  description: "",
  technician: "",
  laborMin: 0,
  price: 0,
  currency: "UAH" as CurrencyCode
});
const serviceOrderPartForm = ref({
  orderId: 0,
  productId: 0,
  quantity: 1,
  price: 0
});
const serviceOrderPaymentForm = ref({
  orderId: 0,
  cashboxId: 0,
  amount: 0,
  method: "cash" as PaymentMethod,
  note: "Оплата ремонту"
});
const serviceOrderActNoteDraft = ref<Record<number, string>>({});
const warehouses = ref<Warehouse[]>([]);
const products = ref<Product[]>([]);
const stockMovements = ref<StockMovement[]>([]);
const stockMovementFilter = ref({
  productId: 0,
  warehouseId: 0
});
const productsCsvForm = ref({
  updateExisting: true,
  csv: "name,sku,stock,purchasePrice,retailPrice,currency\nSample Product,SAMPLE-1,5,100,150,UAH"
});
const productsCsvResult = ref<ProductCSVImportResult | null>(null);
const productsXlsxFile = ref<File | null>(null);
const bulkPriceForm = ref<ProductPriceBulkUpdateRequest>({
  mode: "percent",
  value: 10,
  priceField: "retailPrice",
  roundMode: "none",
  roundTo: 1,
  search: "",
  category: "",
  brand: "",
  supplier: "",
  includeArchived: false
});
const bulkPriceResultText = ref("");
const mergeProductsForm = ref({
  targetProductId: 0,
  sourceProductIdsRaw: ""
});
const mergeProductsResultText = ref("");
const sales = ref<Sale[]>([]);
const receipts = ref<Receipt[]>([]);
const purchases = ref<Purchase[]>([]);
const purchaseCsvForm = ref({
  supplierId: 0,
  supplierOrderId: 0,
  currency: "UAH" as CurrencyCode,
  csv: "productId,quantity,price\n1,1,100",
  note: "Імпорт CSV"
});
const purchaseCsvResult = ref<PurchaseCSVImportResult | null>(null);
const returnDocuments = ref<Document[]>([]);
const documentCsvForm = ref({
  type: "invoice" as Document["type"],
  warehouseId: 0,
  cashboxId: 0,
  currency: "UAH" as CurrencyCode,
  total: 0,
  csv: "productId,quantity,price\n1,1,100",
  note: "Імпорт CSV документа"
});
const documentCsvResult = ref<DocumentCSVImportResult | null>(null);
const receiptFilter = ref<{ saleId: number; status: "" | ReceiptStatus }>({
  saleId: 0,
  status: ""
});
const receiptBulkRetryForm = ref<{ status: "" | "pending" | "failed"; limit: number }>({
  status: "",
  limit: 20
});
const receiptBulkResult = ref<ReceiptBulkRetryResult | null>(null);
const receiptBulkResultText = ref("");
const receiptJobResultText = ref("");
const reservationJobResultText = ref("");
const customerReturnAvailability = ref<ReturnAvailability | null>(null);
const supplierReturnAvailability = ref<ReturnAvailability | null>(null);
const customerReturnForm = ref({
  saleId: 0,
  warehouseId: 0,
  note: ""
});
const supplierReturnForm = ref({
  purchaseId: 0,
  warehouseId: 0,
  note: ""
});
const customerReturnQtyDraft = ref<Record<number, number>>({});
const supplierReturnQtyDraft = ref<Record<number, number>>({});
const returnResultText = ref("");
const saleSearchText = ref("");
const purchaseSearchText = ref("");
const returnDocsFilter = ref<{
  type: "" | "return_from_customer" | "return_to_supplier";
  status: "" | "draft" | "posted" | "cancelled";
}>({
  type: "",
  status: ""
});
const cashShiftFilter = ref<{ cashboxId: number; status: "" | CashShiftStatus }>({
  cashboxId: 0,
  status: ""
});
const openShiftForm = ref({
  cashboxId: 0,
  note: ""
});
const cashOperationForm = ref({
  cashboxId: 0,
  type: "incoming" as "incoming" | "outgoing",
  amount: 0,
  method: "cash" as PaymentMethod,
  description: "Ручна операція"
});
const errorText = ref("");
const isLoading = ref(false);

const newUser = ref({
  username: "",
  password: "",
  role: "seller"
});

function handleError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message === SESSION_EXPIRED) {
    emit("sessionExpired");
    return;
  }
  errorText.value = error instanceof Error ? error.message : fallback;
}

async function loadAdminData() {
  try {
    isLoading.value = true;
    errorText.value = "";
    const [
      loadedLogs,
      loadedUsers,
      loadedRoles,
      loadedTemplates,
      loadedPlaceholders,
      loadedSuppliers,
      loadedGroupedRecommendations,
      loadedCashboxes,
      loadedCashOperations,
      loadedCashShifts,
      loadedPayments,
      loadedCustomers,
      loadedCustomerReminders,
      loadedServiceOrders,
      loadedServiceActs,
      loadedWarehouses,
      loadedProducts,
      loadedSales,
      loadedReceipts,
      loadedPurchases,
      loadedCustomerReturns,
      loadedSupplierReturns,
      loadedStockMovements
    ] = await Promise.all([
      api.auditLogs(props.session.token, 100),
      api.users(props.session.token),
      api.roles(props.session.token),
      api.documentTemplates(props.session.token),
      api.documentTemplatePlaceholders(props.session.token),
      api.suppliers(props.session.token),
      api.supplierOrderRecommendationsGrouped(props.session.token, 200),
      api.cashboxes(props.session.token),
      api.cashOperations(props.session.token),
      api.cashShifts(props.session.token),
      api.payments(props.session.token),
      api.customers(props.session.token),
      api.customerReminders(props.session.token),
      api.serviceOrders(props.session.token),
      api.documents(props.session.token, { type: "act" }),
      api.warehouses(props.session.token),
      api.products(props.session.token),
      api.sales(props.session.token),
      api.receipts(props.session.token),
      api.purchases(props.session.token),
      api.documents(props.session.token, { type: "return_from_customer" }),
      api.documents(props.session.token, { type: "return_to_supplier" }),
      api.stockMovements(props.session.token)
    ]);
    auditLogs.value = loadedLogs;
    adminUsers.value = loadedUsers;
    rolePermissions.value = loadedRoles.roles;
    allPermissions.value = loadedRoles.permissions;
    roleDrafts.value = {};
    for (const role of loadedRoles.roles) {
      roleDrafts.value[role.role] = [...role.permissions];
    }
    documentTemplates.value = loadedTemplates;
    templatePlaceholders.value = loadedPlaceholders;
    suppliers.value = loadedSuppliers;
    recommendationGroups.value = loadedGroupedRecommendations;
    cashboxes.value = loadedCashboxes;
    cashOperations.value = loadedCashOperations;
    cashShifts.value = loadedCashShifts;
    payments.value = loadedPayments;
    customers.value = loadedCustomers;
    customerReminders.value = loadedCustomerReminders;
    serviceOrders.value = loadedServiceOrders;
    serviceOrderActs.value = loadedServiceActs.filter((item) => item.sourceServiceOrderId && item.status !== "cancelled");
    warehouses.value = loadedWarehouses;
    products.value = loadedProducts;
    sales.value = loadedSales;
    receipts.value = loadedReceipts;
    purchases.value = loadedPurchases;
    returnDocuments.value = [...loadedCustomerReturns, ...loadedSupplierReturns].sort((a, b) => b.id - a.id);
    stockMovements.value = loadedStockMovements;
    if (openShiftForm.value.cashboxId === 0 && loadedCashboxes.length > 0) {
      openShiftForm.value.cashboxId = loadedCashboxes[0].id;
    }
    if (cashOperationForm.value.cashboxId === 0 && loadedCashboxes.length > 0) {
      cashOperationForm.value.cashboxId = loadedCashboxes[0].id;
    }
    if (reminderForm.value.customerId === 0 && loadedCustomers.length > 0) {
      reminderForm.value.customerId = loadedCustomers[0].id;
    }
    if (serviceOrderForm.value.customerId === 0 && loadedCustomers.length > 0) {
      serviceOrderForm.value.customerId = loadedCustomers[0].id;
    }
    if (serviceOrderEditForm.value.orderId === 0 && loadedServiceOrders.length > 0) {
      const first = loadedServiceOrders[0];
      serviceOrderEditForm.value.orderId = first.id;
      serviceOrderEditForm.value.productId = first.productId ?? 0;
      serviceOrderEditForm.value.title = first.title;
      serviceOrderEditForm.value.description = first.description;
      serviceOrderEditForm.value.technician = first.technician;
      serviceOrderEditForm.value.laborMin = first.laborMin;
      serviceOrderEditForm.value.price = first.price;
      serviceOrderEditForm.value.currency = first.currency;
    }
    if (serviceOrderPartForm.value.orderId === 0 && loadedServiceOrders.length > 0) {
      serviceOrderPartForm.value.orderId = loadedServiceOrders[0].id;
    }
    if (serviceOrderPaymentForm.value.orderId === 0 && loadedServiceOrders.length > 0) {
      const withDebt = loadedServiceOrders.find((item) => item.debt > 0);
      serviceOrderPaymentForm.value.orderId = withDebt ? withDebt.id : loadedServiceOrders[0].id;
    }
    if (serviceOrderPaymentForm.value.cashboxId === 0 && loadedCashboxes.length > 0) {
      serviceOrderPaymentForm.value.cashboxId = loadedCashboxes[0].id;
    }
    if (customerReturnForm.value.warehouseId === 0 && loadedWarehouses.length > 0) {
      customerReturnForm.value.warehouseId = loadedWarehouses[0].id;
    }
    if (customerReturnForm.value.saleId === 0 && loadedSales.length > 0) {
      customerReturnForm.value.saleId = loadedSales[0].id;
    }
    if (supplierReturnForm.value.warehouseId === 0 && loadedWarehouses.length > 0) {
      supplierReturnForm.value.warehouseId = loadedWarehouses[0].id;
    }
    if (supplierReturnForm.value.purchaseId === 0 && loadedPurchases.length > 0) {
      supplierReturnForm.value.purchaseId = loadedPurchases[0].id;
    }
    if (purchaseCsvForm.value.supplierId === 0 && loadedSuppliers.length > 0) {
      purchaseCsvForm.value.supplierId = loadedSuppliers[0].id;
    }
    if (documentCsvForm.value.warehouseId === 0 && loadedWarehouses.length > 0) {
      documentCsvForm.value.warehouseId = loadedWarehouses[0].id;
    }
    if (documentCsvForm.value.cashboxId === 0 && loadedCashboxes.length > 0) {
      documentCsvForm.value.cashboxId = loadedCashboxes[0].id;
    }
    syncPlannerSelections(loadedGroupedRecommendations);
    if (!selectedTemplateCode.value && loadedTemplates.length > 0) {
      selectedTemplateCode.value = loadedTemplates[0].code;
    }
    syncTemplateFormByCode(selectedTemplateCode.value);
  } catch (error) {
    handleError(error, "Не вдалося завантажити дані адмін-панелі");
  } finally {
    isLoading.value = false;
  }
}

function filteredCashShifts() {
  return cashShifts.value.filter((shift) => {
    if (cashShiftFilter.value.cashboxId > 0 && shift.cashboxId !== cashShiftFilter.value.cashboxId) {
      return false;
    }
    if (cashShiftFilter.value.status !== "" && shift.status !== cashShiftFilter.value.status) {
      return false;
    }
    return true;
  });
}

function isServiceLifecycleAuditAction(action: string): boolean {
  return auditLifecycleActions.has(action);
}

function filteredAuditLogs() {
  const query = auditFilter.value.search.trim().toLowerCase();
  return auditLogs.value.filter((log) => {
    if (auditFilter.value.onlyServiceLifecycle && !isServiceLifecycleAuditAction(log.action)) {
      return false;
    }
    if (auditFilter.value.action !== "" && log.action !== auditFilter.value.action) {
      return false;
    }
    if (query === "") {
      return true;
    }
    const fullText = `${log.user} ${log.action} ${log.entity} ${log.details}`.toLowerCase();
    return fullText.includes(query);
  });
}

function auditActionsForFilter() {
  return Array.from(new Set(auditLogs.value.map((entry) => entry.action)))
    .filter((action) => !auditLifecycleActions.has(action))
    .sort((a, b) => a.localeCompare(b));
}

function auditCountByAction(action: string): number {
  return auditLogs.value.filter((entry) => entry.action === action).length;
}

function serviceLifecycleAuditCount(): number {
  return auditLogs.value.filter((entry) => isServiceLifecycleAuditAction(entry.action)).length;
}

function resetAuditFilter() {
  auditFilter.value.action = "";
  auditFilter.value.search = "";
  auditFilter.value.onlyServiceLifecycle = false;
}

function auditActionLabel(action: string): string {
  switch (action) {
    case "service_order.status.update":
      return "Оновлено статус ремонту";
    case "service_order.reopen":
      return "Ремонт перевідкрито";
    case "service_order.auto_act.create":
      return "Автостворення акта";
    default:
      return action;
  }
}

function auditActionClass(action: string): string {
  switch (action) {
    case "service_order.status.update":
      return "chip chip--required";
    case "service_order.reopen":
      return "chip chip--warn";
    case "service_order.auto_act.create":
      return "chip chip--ok";
    default:
      return "chip";
  }
}

function filteredReturnDocuments() {
  return returnDocuments.value.filter((doc) => {
    if (returnDocsFilter.value.type !== "" && doc.type !== returnDocsFilter.value.type) {
      return false;
    }
    if (returnDocsFilter.value.status !== "" && doc.status !== returnDocsFilter.value.status) {
      return false;
    }
    return true;
  });
}

function returnDocsCountByStatus(status: "draft" | "posted" | "cancelled"): number {
  return returnDocuments.value.filter((doc) => doc.status === status).length;
}

function filteredCustomers() {
  const query = customerSearch.value.trim().toLowerCase();
  if (query === "") {
    return customers.value;
  }
  return customers.value.filter((customer) =>
    `${customer.name} ${customer.phone} ${customer.email}`.toLowerCase().includes(query)
  );
}

function productLabel(productId: number): string {
  const product = products.value.find((item) => item.id === productId);
  if (!product) {
    return `#${productId}`;
  }
  return `${product.name} (${product.sku})`;
}

function filteredSalesOptions() {
  const search = saleSearchText.value.trim().toLowerCase();
  if (search === "") {
    return sales.value;
  }
  return sales.value.filter((sale) => {
    const text = `#${sale.id} ${sale.currency} ${sale.total.toFixed(2)} ${sale.status}`.toLowerCase();
    return text.includes(search);
  });
}

function filteredReceipts() {
  return receipts.value.filter((receipt) => {
    if (receiptFilter.value.saleId > 0 && receipt.saleId !== receiptFilter.value.saleId) {
      return false;
    }
    if (receiptFilter.value.status !== "" && receipt.status !== receiptFilter.value.status) {
      return false;
    }
    return true;
  });
}

function receiptStatusLabel(status: ReceiptStatus): string {
  switch (status) {
    case "pending":
      return "Очікує";
    case "sent":
      return "sent";
    case "failed":
      return "Помилка";
    default:
      return status;
  }
}

function receiptStatusClass(status: ReceiptStatus): string {
  switch (status) {
    case "sent":
      return "chip chip--ok";
    case "failed":
      return "chip chip--warn";
    default:
      return "chip chip--required";
  }
}

function filteredPurchasesOptions() {
  const search = purchaseSearchText.value.trim().toLowerCase();
  if (search === "") {
    return purchases.value;
  }
  return purchases.value.filter((purchase) => {
    const text = `#${purchase.id} ${purchase.currency} ${purchase.total.toFixed(2)} ${purchase.supplierId}`.toLowerCase();
    return text.includes(search);
  });
}

function syncTemplateFormByCode(code: string) {
  const selected = documentTemplates.value.find((tpl) => tpl.code === code);
  if (!selected) {
    return;
  }
  templateForm.value.code = selected.code;
  templateForm.value.name = selected.name;
  templateForm.value.body = selected.body;
  templateForm.value.isActive = selected.isActive;
  templateValidation.value = null;
  templatePreview.value = null;
}

function recommendationKey(supplierId: number | undefined, productId: number): string {
  return `${supplierId ?? 0}:${productId}`;
}

function getPlannerSelection(supplierId: number | undefined, productId: number, recommendedQty: number) {
  const key = recommendationKey(supplierId, productId);
  if (!plannerSelections.value[key]) {
    plannerSelections.value[key] = {
      enabled: true,
      quantity: recommendedQty,
      price: ""
    };
  }
  return plannerSelections.value[key];
}

function syncPlannerSelections(groups: PurchaseRecommendationGroup[]) {
  const next: Record<string, { enabled: boolean; quantity: number; price: string }> = {};
  for (const group of groups) {
    for (const item of group.items) {
      const key = recommendationKey(group.supplierId, item.productId);
      next[key] = plannerSelections.value[key] ?? {
        enabled: true,
        quantity: item.recommendedQty,
        price: ""
      };
    }
  }
  plannerSelections.value = next;
}

function supplierLabel(group: PurchaseRecommendationGroup): string {
  if (group.supplierId) {
    const supplier = suppliers.value.find((item) => item.id === group.supplierId);
    if (supplier) {
      return `${supplier.name} (#${supplier.id})`;
    }
    return `${group.supplierName} (#${group.supplierId})`;
  }
  return `${group.supplierName} (без ID)`;
}

function roleHasPermission(role: string, permission: string): boolean {
  const list = roleDrafts.value[role] ?? [];
  return list.includes(permission);
}

function toggleRolePermission(role: string, permission: string) {
  const current = roleDrafts.value[role] ?? [];
  if (current.includes(permission)) {
    roleDrafts.value[role] = current.filter((item) => item !== permission);
    return;
  }
  roleDrafts.value[role] = [...current, permission];
}

async function createUser() {
  if (!newUser.value.username || !newUser.value.password || !newUser.value.role) {
    errorText.value = "Заповніть username, password і role";
    return;
  }
  try {
    await api.createUser(props.session.token, newUser.value);
    await loadAdminData();
    newUser.value.username = "";
    newUser.value.password = "";
  } catch (error) {
    handleError(error, "Помилка створення користувача");
  }
}

async function createCustomer() {
  if (!newCustomer.value.name.trim()) {
    errorText.value = "Вкажіть ім'я клієнта";
    return;
  }
  try {
    errorText.value = "";
    await api.createCustomer(props.session.token, newCustomer.value);
    newCustomer.value.name = "";
    newCustomer.value.phone = "";
    newCustomer.value.email = "";
    newCustomer.value.comment = "";
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення клієнта");
  }
}

async function createCustomerReminder() {
  if (reminderForm.value.customerId <= 0) {
    errorText.value = "Оберіть клієнта";
    return;
  }
  if (!reminderForm.value.text.trim()) {
    errorText.value = "Вкажіть текст нагадування";
    return;
  }
  try {
    errorText.value = "";
    await api.createCustomerReminder(props.session.token, {
      customerId: reminderForm.value.customerId,
      text: reminderForm.value.text,
      dueAt: reminderForm.value.dueAt ? new Date(reminderForm.value.dueAt).toISOString() : undefined
    });
    reminderForm.value.text = "";
    reminderForm.value.dueAt = "";
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення нагадування");
  }
}

async function completeCustomerReminder(reminderId: number) {
  try {
    errorText.value = "";
    await api.completeCustomerReminder(props.session.token, reminderId);
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка завершення нагадування");
  }
}

function filteredServiceOrders() {
  return serviceOrders.value.filter((item) => {
    if (serviceOrderFilter.value.customerId > 0 && item.customerId !== serviceOrderFilter.value.customerId) {
      return false;
    }
    if (serviceOrderFilter.value.status !== "" && item.status !== serviceOrderFilter.value.status) {
      return false;
    }
    return true;
  });
}

function syncServiceOrderEditFormByOrderId() {
  const selected = serviceOrders.value.find((item) => item.id === serviceOrderEditForm.value.orderId);
  if (!selected) {
    return;
  }
  serviceOrderEditForm.value.productId = selected.productId ?? 0;
  serviceOrderEditForm.value.title = selected.title;
  serviceOrderEditForm.value.description = selected.description;
  serviceOrderEditForm.value.technician = selected.technician;
  serviceOrderEditForm.value.laborMin = selected.laborMin;
  serviceOrderEditForm.value.price = selected.price;
  serviceOrderEditForm.value.currency = selected.currency;
}

function selectedServiceOrderForPayment() {
  return serviceOrders.value.find((item) => item.id === serviceOrderPaymentForm.value.orderId) ?? null;
}

function serviceOrderActByOrderId(orderId: number) {
  return serviceOrderActs.value.find((doc) => doc.sourceServiceOrderId === orderId) ?? null;
}

async function createServiceOrder() {
  if (serviceOrderForm.value.customerId <= 0) {
    errorText.value = "Оберіть клієнта";
    return;
  }
  if (!serviceOrderForm.value.title.trim()) {
    errorText.value = "Вкажіть назву сервісного замовлення";
    return;
  }
  if (serviceOrderForm.value.price < 0) {
    errorText.value = "Ціна не може бути від'ємною";
    return;
  }
  if (serviceOrderForm.value.laborMin < 0) {
    errorText.value = "Тривалість робіт не може бути від'ємною";
    return;
  }
  try {
    errorText.value = "";
    await api.createServiceOrder(props.session.token, {
      customerId: serviceOrderForm.value.customerId,
      productId: serviceOrderForm.value.productId > 0 ? serviceOrderForm.value.productId : undefined,
      title: serviceOrderForm.value.title,
      description: serviceOrderForm.value.description,
      technician: serviceOrderForm.value.technician,
      laborMin: serviceOrderForm.value.laborMin,
      price: serviceOrderForm.value.price,
      currency: serviceOrderForm.value.currency
    });
    serviceOrderForm.value.title = "";
    serviceOrderForm.value.description = "";
    serviceOrderForm.value.technician = "";
    serviceOrderForm.value.laborMin = 0;
    serviceOrderForm.value.productId = 0;
    serviceOrderForm.value.price = 0;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення сервісного замовлення");
  }
}

async function updateServiceOrderStatus(orderId: number, status: ServiceOrderStatus) {
  try {
    errorText.value = "";
    await api.updateServiceOrderStatus(props.session.token, orderId, status);
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка оновлення статусу сервісного замовлення");
  }
}

async function updateServiceOrderDetails() {
  if (serviceOrderEditForm.value.orderId <= 0) {
    errorText.value = "Оберіть сервісне замовлення";
    return;
  }
  if (!serviceOrderEditForm.value.title.trim()) {
    errorText.value = "Вкажіть назву сервісного замовлення";
    return;
  }
  if (serviceOrderEditForm.value.laborMin < 0 || serviceOrderEditForm.value.price < 0) {
    errorText.value = "Тривалість і ціна мають бути невід'ємними";
    return;
  }
  try {
    errorText.value = "";
    await api.updateServiceOrder(props.session.token, serviceOrderEditForm.value.orderId, {
      productId: serviceOrderEditForm.value.productId > 0 ? serviceOrderEditForm.value.productId : undefined,
      title: serviceOrderEditForm.value.title,
      description: serviceOrderEditForm.value.description,
      technician: serviceOrderEditForm.value.technician,
      laborMin: serviceOrderEditForm.value.laborMin,
      price: serviceOrderEditForm.value.price,
      currency: serviceOrderEditForm.value.currency
    });
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка оновлення деталей сервісного замовлення");
  }
}

async function addServiceOrderPart() {
  if (serviceOrderPartForm.value.orderId <= 0) {
    errorText.value = "Оберіть сервісне замовлення";
    return;
  }
  if (serviceOrderPartForm.value.productId <= 0) {
    errorText.value = "Оберіть товар";
    return;
  }
  if (serviceOrderPartForm.value.quantity <= 0 || serviceOrderPartForm.value.price < 0) {
    errorText.value = "Вкажіть коректні quantity і price";
    return;
  }
  try {
    errorText.value = "";
    await api.addServiceOrderPart(props.session.token, serviceOrderPartForm.value.orderId, {
      productId: serviceOrderPartForm.value.productId,
      quantity: serviceOrderPartForm.value.quantity,
      price: serviceOrderPartForm.value.price
    });
    serviceOrderPartForm.value.quantity = 1;
    serviceOrderPartForm.value.price = 0;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка додавання запчастини");
  }
}

async function createServiceOrderPayment() {
  const order = selectedServiceOrderForPayment();
  if (!order) {
    errorText.value = "Оберіть сервісне замовлення";
    return;
  }
  if (serviceOrderPaymentForm.value.cashboxId <= 0) {
    errorText.value = "Оберіть касу";
    return;
  }
  if (serviceOrderPaymentForm.value.amount <= 0) {
    errorText.value = "Сума оплати має бути більшою за 0";
    return;
  }
  try {
    errorText.value = "";
    await api.createPayment(props.session.token, {
      serviceOrderId: order.id,
      cashboxId: serviceOrderPaymentForm.value.cashboxId,
      amount: serviceOrderPaymentForm.value.amount,
      currency: order.currency,
      method: serviceOrderPaymentForm.value.method,
      note: serviceOrderPaymentForm.value.note
    });
    serviceOrderPaymentForm.value.amount = 0;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення оплати ремонту");
  }
}

async function createServiceOrderAct(orderId: number) {
  const note = serviceOrderActNoteDraft.value[orderId] ?? "";
  try {
    errorText.value = "";
    await api.createServiceOrderActDocument(props.session.token, orderId, {
      note,
      autoPost: true
    });
    serviceOrderActNoteDraft.value[orderId] = "";
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення акта виконаних робіт");
  }
}

async function downloadServiceOrderActPdf(orderId: number) {
  try {
    errorText.value = "";
    const blob = await api.serviceOrderActPdf(props.session.token, orderId);
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `service-order-${orderId}-act.pdf`;
    link.click();
    window.URL.revokeObjectURL(url);
  } catch (error) {
    handleError(error, "Помилка завантаження PDF акта");
  }
}

async function cancelServiceOrderAct(orderId: number) {
  try {
    errorText.value = "";
    await api.cancelServiceOrderActDocument(props.session.token, orderId);
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка скасування акта");
  }
}

async function saveRolePermissions(role: string) {
  try {
    await api.updateRolePermissions(props.session.token, role, roleDrafts.value[role] ?? []);
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка збереження прав ролі");
  }
}

async function validateTemplate() {
  if (!templateForm.value.code || !templateForm.value.body) {
    errorText.value = "Вкажіть code та body шаблону";
    return;
  }
  try {
    errorText.value = "";
    templateValidation.value = await api.validateDocumentTemplate(props.session.token, {
      code: templateForm.value.code,
      body: templateForm.value.body,
      strict: templateStrict.value
    });
  } catch (error) {
    handleError(error, "Помилка валідації шаблону");
  }
}

async function previewTemplate() {
  if (!templateForm.value.code || !templateForm.value.body) {
    errorText.value = "Вкажіть code та body шаблону";
    return;
  }
  try {
    errorText.value = "";
    templatePreview.value = await api.previewDocumentTemplate(props.session.token, {
      code: templateForm.value.code,
      body: templateForm.value.body,
      strict: templateStrict.value
    });
    templateValidation.value = templatePreview.value.validation;
  } catch (error) {
    handleError(error, "Помилка preview шаблону");
  }
}

async function saveDocumentTemplate() {
  if (!templateForm.value.code || !templateForm.value.name || !templateForm.value.body) {
    errorText.value = "Заповніть code, name і body";
    return;
  }
  try {
    errorText.value = "";
    await api.upsertDocumentTemplate(props.session.token, {
      code: templateForm.value.code,
      name: templateForm.value.name,
      body: templateForm.value.body,
      isActive: templateForm.value.isActive
    });
    selectedTemplateCode.value = templateForm.value.code;
    await loadAdminData();
    await validateTemplate();
  } catch (error) {
    handleError(error, "Помилка збереження шаблону");
  }
}

async function refreshPlanner() {
  try {
    plannerResultText.value = "";
    recommendationGroups.value = await api.supplierOrderRecommendationsGrouped(props.session.token, 200);
    syncPlannerSelections(recommendationGroups.value);
  } catch (error) {
    handleError(error, "Помилка оновлення рекомендацій закупівлі");
  }
}

async function createOrdersFromPlanner() {
  try {
    const ordersPayload: PurchaseRecommendationCreateOrderRequest[] = [];
    for (const group of recommendationGroups.value) {
      if (!group.supplierId) {
        continue;
      }
      const items: Array<{ productId: number; quantity: number; price?: number }> = [];
      for (const item of group.items) {
        const key = recommendationKey(group.supplierId, item.productId);
        const selection = plannerSelections.value[key];
        if (!selection || !selection.enabled) {
          continue;
        }
        if (selection.quantity <= 0) {
          continue;
        }
        const line: { productId: number; quantity: number; price?: number } = {
          productId: item.productId,
          quantity: selection.quantity
        };
        const price = Number.parseFloat(selection.price);
        if (selection.price.trim() !== "" && Number.isFinite(price) && price >= 0) {
          line.price = price;
        }
        items.push(line);
      }
      if (items.length > 0) {
        ordersPayload.push({
          supplierId: group.supplierId,
          currency: plannerCurrency.value,
          items
        });
      }
    }
    if (ordersPayload.length === 0) {
      errorText.value = "Немає валідних позицій для створення замовлень";
      return;
    }
    const created = await api.createSupplierOrdersBulkFromRecommendations(props.session.token, {
      orders: ordersPayload
    });
    plannerResultText.value = `Створено замовлень: ${created.length}`;
    await refreshPlanner();
  } catch (error) {
    handleError(error, "Помилка bulk-створення замовлень з рекомендацій");
  }
}

async function refreshCashShifts() {
  try {
    cashShifts.value = await api.cashShifts(props.session.token);
  } catch (error) {
    handleError(error, "Помилка оновлення касових змін");
  }
}

async function refreshReceipts() {
  try {
    receipts.value = await api.receipts(props.session.token);
  } catch (error) {
    handleError(error, "Помилка оновлення фіскальних чеків");
  }
}

async function retryReceiptSend(receiptId: number) {
  try {
    errorText.value = "";
    receiptBulkResultText.value = "";
    await api.retryReceipt(props.session.token, receiptId);
    await refreshReceipts();
    receiptBulkResult.value = null;
    receiptBulkResultText.value = `Чек #${receiptId} — повторна відправка виконана успішно`;
  } catch (error) {
    handleError(error, "Помилка повторної відправки чека");
  }
}

async function retryReceiptsBulk() {
  if (receiptBulkRetryForm.value.limit <= 0) {
    errorText.value = "Ліміт має бути більшим за 0";
    return;
  }
  try {
    errorText.value = "";
    const result = await api.retryReceiptsBulk(props.session.token, {
      status: receiptBulkRetryForm.value.status === "" ? undefined : receiptBulkRetryForm.value.status,
      limit: receiptBulkRetryForm.value.limit
    });
    receiptBulkResult.value = result;
    receiptBulkResultText.value = `Масовий повтор: спроб=${result.attempted}, успішно=${result.succeeded}, помилок=${result.failed}`;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка масового retry чеків");
  }
}

async function enqueueReceiptRetryJob() {
  if (receiptBulkRetryForm.value.limit <= 0) {
    errorText.value = "Ліміт має бути більшим за 0";
    return;
  }
  try {
    errorText.value = "";
    const job = await api.enqueueReceiptRetryJob(props.session.token, {
      status: receiptBulkRetryForm.value.status === "" ? undefined : receiptBulkRetryForm.value.status,
      limit: receiptBulkRetryForm.value.limit
    });
    receiptJobResultText.value = `Retry job queued: #${job.id} (${job.jobType})`;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка постановки retry job");
  }
}

async function runReceiptRetryJobsNow() {
  try {
    errorText.value = "";
    const runResult = await api.runBackgroundJobs(props.session.token);
    const receiptJobs = runResult.processed.filter((job) => job.jobType === "receipt_retries");
    receiptJobResultText.value = `Jobs run: processed=${runResult.count}, receipt_jobs=${receiptJobs.length}`;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка запуску фонових job");
  }
}

async function enqueueReservationExpiryJob() {
  try {
    errorText.value = "";
    const job = await api.enqueueReservationExpiryJob(props.session.token);
    reservationJobResultText.value = `Reservation expiry job queued: #${job.id}`;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка постановки reservation expiry job");
  }
}

async function importPurchaseCsv() {
  if (purchaseCsvForm.value.supplierId <= 0) {
    errorText.value = "Оберіть постачальника для імпорту надходження";
    return;
  }
  if (purchaseCsvForm.value.csv.trim() === "") {
    errorText.value = "CSV надходження не може бути порожнім";
    return;
  }
  try {
    errorText.value = "";
    purchaseCsvResult.value = await api.importPurchaseCsv(props.session.token, {
      supplierId: purchaseCsvForm.value.supplierId,
      supplierOrderId: purchaseCsvForm.value.supplierOrderId > 0 ? purchaseCsvForm.value.supplierOrderId : undefined,
      currency: purchaseCsvForm.value.currency,
      csv: purchaseCsvForm.value.csv,
      note: purchaseCsvForm.value.note
    });
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка імпорту надходження CSV");
  }
}

async function importDocumentCsv() {
  if (documentCsvForm.value.csv.trim() === "") {
    errorText.value = "CSV документа не може бути порожнім";
    return;
  }
  try {
    errorText.value = "";
    documentCsvResult.value = await api.importDocumentCsv(props.session.token, {
      type: documentCsvForm.value.type,
      warehouseId: documentCsvForm.value.warehouseId > 0 ? documentCsvForm.value.warehouseId : undefined,
      cashboxId: documentCsvForm.value.cashboxId > 0 ? documentCsvForm.value.cashboxId : undefined,
      currency: documentCsvForm.value.currency,
      total: documentCsvForm.value.total,
      csv: documentCsvForm.value.csv,
      note: documentCsvForm.value.note
    });
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка імпорту документа CSV");
  }
}

async function openCashShift() {
  if (openShiftForm.value.cashboxId <= 0) {
    errorText.value = "Оберіть касу для відкриття зміни";
    return;
  }
  try {
    errorText.value = "";
    await api.openCashShift(props.session.token, {
      cashboxId: openShiftForm.value.cashboxId,
      note: openShiftForm.value.note
    });
    openShiftForm.value.note = "";
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка відкриття зміни");
  }
}

async function createCashOperation() {
  if (cashOperationForm.value.cashboxId <= 0 || cashOperationForm.value.amount <= 0) {
    errorText.value = "Оберіть касу та суму касової операції";
    return;
  }
  try {
    errorText.value = "";
    await api.createCashOperation(props.session.token, cashOperationForm.value);
    cashOperationForm.value.amount = 0;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення касової операції");
  }
}

async function closeCashShift(shiftId: number) {
  try {
    errorText.value = "";
    await api.closeCashShift(props.session.token, shiftId, { note: "closed from admin panel" });
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка закриття зміни");
  }
}

async function loadCustomerReturnAvailability() {
  if (customerReturnForm.value.saleId <= 0) {
    errorText.value = "Вкажіть коректний saleId";
    return;
  }
  try {
    errorText.value = "";
    customerReturnAvailability.value = await api.customerReturnAvailability(props.session.token, customerReturnForm.value.saleId);
    customerReturnQtyDraft.value = {};
    for (const item of customerReturnAvailability.value.items) {
      customerReturnQtyDraft.value[item.productId] = item.availableQty;
    }
  } catch (error) {
    handleError(error, "Помилка завантаження доступних позицій повернення покупця");
  }
}

function fillMaxCustomerReturnQty() {
  if (!customerReturnAvailability.value) {
    return;
  }
  for (const item of customerReturnAvailability.value.items) {
    customerReturnQtyDraft.value[item.productId] = item.availableQty;
  }
}

function clearCustomerReturnQty() {
  if (!customerReturnAvailability.value) {
    return;
  }
  for (const item of customerReturnAvailability.value.items) {
    customerReturnQtyDraft.value[item.productId] = 0;
  }
}

async function loadSupplierReturnAvailability() {
  if (supplierReturnForm.value.purchaseId <= 0) {
    errorText.value = "Вкажіть коректний purchaseId";
    return;
  }
  try {
    errorText.value = "";
    supplierReturnAvailability.value = await api.supplierReturnAvailability(props.session.token, supplierReturnForm.value.purchaseId);
    supplierReturnQtyDraft.value = {};
    for (const item of supplierReturnAvailability.value.items) {
      supplierReturnQtyDraft.value[item.productId] = item.availableQty;
    }
  } catch (error) {
    handleError(error, "Помилка завантаження доступних позицій повернення постачальнику");
  }
}

function fillMaxSupplierReturnQty() {
  if (!supplierReturnAvailability.value) {
    return;
  }
  for (const item of supplierReturnAvailability.value.items) {
    supplierReturnQtyDraft.value[item.productId] = item.availableQty;
  }
}

function clearSupplierReturnQty() {
  if (!supplierReturnAvailability.value) {
    return;
  }
  for (const item of supplierReturnAvailability.value.items) {
    supplierReturnQtyDraft.value[item.productId] = 0;
  }
}

function selectedReturnItems(
  availability: ReturnAvailability | null,
  qtyDraft: Record<number, number>
): DocumentItem[] {
  if (!availability) {
    return [];
  }
  const items: DocumentItem[] = [];
  for (const item of availability.items) {
    const requested = Math.max(0, Math.floor(qtyDraft[item.productId] ?? 0));
    const quantity = Math.min(requested, item.availableQty);
    if (quantity <= 0) {
      continue;
    }
    items.push({
      productId: item.productId,
      quantity,
      price: item.price
    });
  }
  return items;
}

async function createCustomerReturnDocument() {
  if (!customerReturnAvailability.value) {
    errorText.value = "Спочатку завантажте доступні позиції повернення";
    return;
  }
  if (customerReturnForm.value.warehouseId <= 0) {
    errorText.value = "Оберіть склад";
    return;
  }
  const items = selectedReturnItems(customerReturnAvailability.value, customerReturnQtyDraft.value);
  if (items.length === 0) {
    errorText.value = "Оберіть хоча б одну позицію з quantity > 0";
    return;
  }
  try {
    errorText.value = "";
    const doc = await api.createCustomerReturnDocument(props.session.token, {
      saleId: customerReturnForm.value.saleId,
      warehouseId: customerReturnForm.value.warehouseId,
      currency: customerReturnAvailability.value.currency,
      items,
      note: customerReturnForm.value.note
    });
    returnResultText.value = `Створено документ повернення від покупця: ${doc.number}`;
    customerReturnForm.value.note = "";
    await loadCustomerReturnAvailability();
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення документа повернення від покупця");
  }
}

async function createSupplierReturnDocument() {
  if (!supplierReturnAvailability.value) {
    errorText.value = "Спочатку завантажте доступні позиції повернення";
    return;
  }
  if (supplierReturnForm.value.warehouseId <= 0) {
    errorText.value = "Оберіть склад";
    return;
  }
  const items = selectedReturnItems(supplierReturnAvailability.value, supplierReturnQtyDraft.value);
  if (items.length === 0) {
    errorText.value = "Оберіть хоча б одну позицію з quantity > 0";
    return;
  }
  try {
    errorText.value = "";
    const doc = await api.createSupplierReturnDocument(props.session.token, {
      purchaseId: supplierReturnForm.value.purchaseId,
      warehouseId: supplierReturnForm.value.warehouseId,
      currency: supplierReturnAvailability.value.currency,
      items,
      note: supplierReturnForm.value.note
    });
    returnResultText.value = `Створено документ повернення постачальнику: ${doc.number}`;
    supplierReturnForm.value.note = "";
    await loadSupplierReturnAvailability();
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка створення документа повернення постачальнику");
  }
}

async function postReturnDocument(documentId: number) {
  try {
    errorText.value = "";
    const doc = await api.postDocument(props.session.token, documentId);
    returnResultText.value = `Документ ${doc.number} проведено`;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка проведення return документа");
  }
}

async function importProductsCsv() {
  if (productsCsvForm.value.csv.trim() === "") {
    errorText.value = "CSV не може бути порожнім";
    return;
  }
  try {
    errorText.value = "";
    productsCsvResult.value = await api.importProductsCsv(props.session.token, {
      csv: productsCsvForm.value.csv,
      updateExisting: productsCsvForm.value.updateExisting
    });
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка імпорту CSV");
  }
}

async function exportProductsCsv() {
  try {
    errorText.value = "";
    const blob = await api.exportProductsCsv(props.session.token, true);
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "products.csv";
    link.click();
    window.URL.revokeObjectURL(url);
  } catch (error) {
    handleError(error, "Помилка експорту CSV");
  }
}

async function exportProductsXlsx() {
  try {
    errorText.value = "";
    const blob = await api.exportProductsXlsx(props.session.token, true);
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "products.xlsx";
    link.click();
    window.URL.revokeObjectURL(url);
  } catch (error) {
    handleError(error, "Помилка експорту XLSX");
  }
}

async function importProductsXlsx() {
  if (!productsXlsxFile.value) {
    errorText.value = "Оберіть XLSX файл для імпорту";
    return;
  }
  try {
    errorText.value = "";
    productsCsvResult.value = await api.importProductsXlsx(props.session.token, {
      file: productsXlsxFile.value,
      updateExisting: productsCsvForm.value.updateExisting
    });
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка імпорту XLSX");
  }
}

async function applyBulkPriceUpdate() {
  if (!Number.isFinite(bulkPriceForm.value.value) || bulkPriceForm.value.value === 0) {
    errorText.value = "Вкажіть ненульове value для масового оновлення";
    return;
  }
  try {
    errorText.value = "";
    const result = await api.bulkUpdateProductPrices(props.session.token, bulkPriceForm.value);
    bulkPriceResultText.value = `Оновлено товарів: ${result.updated}`;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка масового оновлення цін");
  }
}

async function generateProductBarcode(productId: number) {
  try {
    await api.generateProductBarcode(props.session.token, productId);
    await loadAdminData();
  } catch (error) {
    handleError(error, "Не вдалося згенерувати штрихкод");
  }
}

async function archiveProduct(productId: number, archived: boolean) {
  try {
    await api.archiveProduct(props.session.token, productId, archived);
    await loadAdminData();
  } catch (error) {
    handleError(error, "Не вдалося змінити статус архіву товару");
  }
}

function filteredStockMovements() {
  return stockMovements.value.filter((movement) => {
    if (stockMovementFilter.value.productId > 0 && movement.productId !== stockMovementFilter.value.productId) {
      return false;
    }
    if (
      stockMovementFilter.value.warehouseId > 0 &&
      movement.fromWarehouseId !== stockMovementFilter.value.warehouseId &&
      movement.toWarehouseId !== stockMovementFilter.value.warehouseId
    ) {
      return false;
    }
    return true;
  });
}

async function mergeDuplicateProducts() {
  if (mergeProductsForm.value.targetProductId <= 0) {
    errorText.value = "Вкажіть targetProductId";
    return;
  }
  const sourceIds = mergeProductsForm.value.sourceProductIdsRaw
    .split(",")
    .map((item) => Number.parseInt(item.trim(), 10))
    .filter((value) => Number.isFinite(value) && value > 0);
  if (sourceIds.length === 0) {
    errorText.value = "Вкажіть sourceProductIds через кому";
    return;
  }
  try {
    errorText.value = "";
    const result = await api.mergeDuplicateProducts(props.session.token, {
      targetProductId: mergeProductsForm.value.targetProductId,
      sourceProductIds: sourceIds
    });
    mergeProductsResultText.value = `Об'єднано в target #${result.targetProductId}: ${result.mergedProductIds.join(", ")}`;
    await loadAdminData();
  } catch (error) {
    handleError(error, "Помилка об'єднання дублів");
  }
}

watch(selectedTemplateCode, (nextCode) => {
  syncTemplateFormByCode(nextCode);
});

watch(
  () => serviceOrderEditForm.value.orderId,
  () => {
    syncServiceOrderEditFormByOrderId();
  }
);

watch(
  () => serviceOrderPaymentForm.value.orderId,
  () => {
    const order = selectedServiceOrderForPayment();
    if (!order) {
      return;
    }
    serviceOrderPaymentForm.value.amount = Number(order.debt.toFixed(2));
    const matchedCashbox = cashboxes.value.find((item) => item.currency === order.currency);
    if (matchedCashbox) {
      serviceOrderPaymentForm.value.cashboxId = matchedCashbox.id;
    }
  }
);

const adminTab = ref('audit');

onMounted(loadAdminData);
</script>

<template>
  <main class="page-content">
    <div class="page-header">
      <h2>Адміністратор</h2>
    </div>
    <p v-if="errorText" class="error-text" style="margin-bottom:0.75rem">{{ errorText }}</p>
    <p v-if="isLoading" class="subtle" style="margin-bottom:0.75rem">Оновлення даних...</p>

    <div class="tab-row" style="margin-bottom:1.5rem">
      <button :class="['tab-button', adminTab==='audit'&&'tab-button--active']" @click="adminTab='audit'">Журнал дій</button>
      <button :class="['tab-button', adminTab==='users'&&'tab-button--active']" @click="adminTab='users'">Користувачі</button>
    </div>

    <section v-if="adminTab==='audit'" class="panel">
      <h2>Журнал дій</h2>
      <div class="chip-row">
        <label>
          Фільтр по дії
          <select v-model="auditFilter.action">
            <option value="">Усі</option>
            <option value="service_order.status.update">service_order.status.update</option>
            <option value="service_order.reopen">service_order.reopen</option>
            <option value="service_order.auto_act.create">service_order.auto_act.create</option>
            <option v-for="action in auditActionsForFilter()" :key="`audit-action-${action}`" :value="action">
              {{ action }}
            </option>
          </select>
        </label>
        <label>
          Пошук
          <input v-model="auditFilter.search" placeholder="користувач / дія / деталі" />
        </label>
        <label class="permission-item">
          <input v-model="auditFilter.onlyServiceLifecycle" type="checkbox" />
          <span>Лише lifecycle ремонтів</span>
        </label>
        <button type="button" class="ghost-button" @click="resetAuditFilter">Скинути фільтр</button>
      </div>
      <div class="chip-row">
        <span class="chip chip--required">status updates: {{ auditCountByAction("service_order.status.update") }}</span>
        <span class="chip chip--warn">reopen: {{ auditCountByAction("service_order.reopen") }}</span>
        <span class="chip chip--ok">auto acts: {{ auditCountByAction("service_order.auto_act.create") }}</span>
        <span class="chip">lifecycle total: {{ serviceLifecycleAuditCount() }}</span>
        <span class="chip">shown: {{ filteredAuditLogs().length }} / {{ auditLogs.length }}</span>
      </div>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Користувач</th>
            <th>Дія</th>
            <th>Об'єкт</th>
            <th>Деталі</th>
            <th>Коли</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in filteredAuditLogs()" :key="log.id">
            <td>{{ log.id }}</td>
            <td>{{ log.user }}</td>
            <td>
              <span :class="auditActionClass(log.action)">{{ auditActionLabel(log.action) }}</span>
            </td>
            <td>{{ log.entity }}</td>
            <td>{{ log.details }}</td>
            <td>{{ new Date(log.createdAt).toLocaleString() }}</td>
          </tr>
        </tbody>
      </table>
      <p v-if="filteredAuditLogs().length === 0" class="subtle">За поточним фільтром записів не знайдено.</p>
    </section>

    <section v-if="adminTab==='audit'" class="panel">
      <h2>Фіскальні чеки</h2>
      <div class="chip-row">
        <label>
          Sale
          <select v-model.number="receiptFilter.saleId">
            <option :value="0">Усі sale</option>
            <option v-for="sale in sales" :key="`receipt-sale-filter-${sale.id}`" :value="sale.id">
              #{{ sale.id }} · {{ sale.total.toFixed(2) }} {{ sale.currency }}
            </option>
          </select>
        </label>
        <label>
          Статус
          <select v-model="receiptFilter.status">
            <option value="">Усі</option>
            <option value="pending">Очікує</option>
            <option value="sent">Надіслано</option>
            <option value="failed">Помилка</option>
          </select>
        </label>
        <label>
          Bulk status
          <select v-model="receiptBulkRetryForm.status">
            <option value="">pending + failed</option>
            <option value="pending">Тільки очікуючі</option>
            <option value="failed">Тільки помилкові</option>
          </select>
        </label>
        <label>
          Bulk limit
          <input v-model.number="receiptBulkRetryForm.limit" min="1" max="200" type="number" />
        </label>
        <button type="button" class="ghost-button" @click="refreshReceipts">Оновити чеки</button>
        <button type="button" class="ghost-button" @click="retryReceiptsBulk">Повторити масово</button>
        <button type="button" class="ghost-button" @click="enqueueReceiptRetryJob">Поставити в чергу</button>
        <button type="button" class="ghost-button" @click="runReceiptRetryJobsNow">Запустити зараз</button>
        <button
          type="button"
          class="ghost-button"
          @click="
            receiptFilter.saleId = 0;
            receiptFilter.status = '';
          "
        >
          Скинути фільтри
        </button>
      </div>
      <p v-if="receiptBulkResultText" class="ok-text">{{ receiptBulkResultText }}</p>
      <p v-if="receiptJobResultText" class="ok-text">{{ receiptJobResultText }}</p>
      <div class="chip-row">
        <span class="chip chip--required">pending: {{ receipts.filter((item) => item.status === "pending").length }}</span>
        <span class="chip chip--ok">sent: {{ receipts.filter((item) => item.status === "sent").length }}</span>
        <span class="chip chip--warn">failed: {{ receipts.filter((item) => item.status === "failed").length }}</span>
        <span class="chip">shown: {{ filteredReceipts().length }} / {{ receipts.length }}</span>
      </div>
      <div v-if="receiptBulkResult" class="chip-row">
        <span class="chip">last attempted: {{ receiptBulkResult.attempted }}</span>
        <span class="chip chip--ok">last succeeded: {{ receiptBulkResult.succeeded }}</span>
        <span class="chip chip--warn">last failed: {{ receiptBulkResult.failed }}</span>
      </div>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Продаж</th>
            <th>Статус</th>
            <th>Провайдер</th>
            <th>Зовнішній</th>
            <th>Фіскальний</th>
            <th>QR</th>
            <th>Помилка</th>
            <th>Надіслано</th>
            <th>Дія</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="receipt in filteredReceipts()" :key="receipt.id">
            <td>{{ receipt.id }}</td>
            <td>#{{ receipt.saleId }}</td>
            <td>
              <span :class="receiptStatusClass(receipt.status)">{{ receiptStatusLabel(receipt.status) }}</span>
            </td>
            <td>{{ receipt.provider }}</td>
            <td>{{ receipt.externalId || "-" }}</td>
            <td>{{ receipt.fiscalNumber || "-" }}</td>
            <td>
              <a v-if="receipt.qrUrl" :href="receipt.qrUrl" rel="noreferrer" target="_blank">QR</a>
              <span v-else>-</span>
            </td>
            <td>{{ receipt.errorMessage || "-" }}</td>
            <td>{{ receipt.sentAt ? new Date(receipt.sentAt).toLocaleString() : "-" }}</td>
            <td>
              <button
                v-if="receipt.status !== 'sent'"
                type="button"
                class="ghost-button"
                title="Повторно надіслати цей чек до Checkbox"
                @click="retryReceiptSend(receipt.id)"
              >
                Повторити
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="filteredReceipts().length === 0" class="subtle">Чеки не знайдено за поточним фільтром.</p>
    </section>

    <section v-if="adminTab==='users'" class="panel">
      <article class="panel">
        <h2>Користувачі</h2>
        <form class="grid" @submit.prevent="createUser">
          <label>Username <input v-model="newUser.username" /></label>
          <label>Пароль <input v-model="newUser.password" type="password" /></label>
          <label>
            Роль
            <select v-model="newUser.role">
              <option v-for="role in rolePermissions" :key="role.role" :value="role.role">
                {{ role.role }}
              </option>
            </select>
          </label>
          <button type="submit">Додати користувача</button>
        </form>
        <table>
          <thead>
            <tr>
              <th>Логін</th>
              <th>Роль</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in adminUsers" :key="user.username">
              <td>{{ user.username }}</td>
              <td>{{ user.role }}</td>
            </tr>
          </tbody>
        </table>
      </article>

      <article v-if="adminTab==='users'" class="panel">
        <h2>Права ролей</h2>
        <div v-for="role in rolePermissions" :key="role.role" class="grid">
          <strong>{{ role.role }}</strong>
          <div class="chip-row">
            <label v-for="permission in allPermissions" :key="`${role.role}-${permission}`" class="permission-item">
              <input
                :checked="roleHasPermission(role.role, permission)"
                type="checkbox"
                @change="toggleRolePermission(role.role, permission)"
              />
              <span>{{ permission }}</span>
            </label>
          </div>
          <button type="button" @click="saveRolePermissions(role.role)">Зберегти права ролі</button>
        </div>
      </article>
    </section>

    <section v-if="adminTab==='users'" class="panel">
      <h2>Клієнти та нагадування</h2>
      <div class="panel-grid">
        <article class="panel">
          <h3>Новий клієнт</h3>
          <form class="grid" @submit.prevent="createCustomer">
            <label>Ім'я <input v-model="newCustomer.name" required /></label>
            <label>Телефон <input v-model="newCustomer.phone" /></label>
            <label>Email <input v-model="newCustomer.email" type="email" /></label>
            <label>Коментар <input v-model="newCustomer.comment" /></label>
            <button type="submit">Додати клієнта</button>
          </form>
        </article>

        <article class="panel">
          <h3>Нове нагадування</h3>
          <form class="grid" @submit.prevent="createCustomerReminder">
            <label>
              Клієнт
              <select v-model.number="reminderForm.customerId">
                <option :value="0">Оберіть клієнта</option>
                <option v-for="customer in customers" :key="`reminder-customer-${customer.id}`" :value="customer.id">
                  {{ customer.name }} (#{{ customer.id }})
                </option>
              </select>
            </label>
            <label>Текст <input v-model="reminderForm.text" required /></label>
            <label>Due at <input v-model="reminderForm.dueAt" type="datetime-local" /></label>
            <button type="submit">Створити нагадування</button>
          </form>
        </article>
      </div>

      <div class="chip-row">
        <label>
          Пошук клієнтів
          <input v-model="customerSearch" placeholder="ім'я / телефон / email" />
        </label>
      </div>

      <h3>Клієнти</h3>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Ім'я</th>
            <th>Телефон</th>
            <th>Email</th>
            <th>Коментар</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="customer in filteredCustomers()" :key="`crm-customer-${customer.id}`">
            <td>{{ customer.id }}</td>
            <td>{{ customer.name }}</td>
            <td>{{ customer.phone || "-" }}</td>
            <td>{{ customer.email || "-" }}</td>
            <td>{{ customer.comment || "-" }}</td>
          </tr>
        </tbody>
      </table>
      <p v-if="filteredCustomers().length === 0" class="subtle">Клієнтів не знайдено.</p>

      <h3>Нагадування</h3>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Клієнт ID</th>
            <th>Текст</th>
            <th>Статус</th>
            <th>Термін</th>
            <th>Дія</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in customerReminders" :key="`crm-reminder-${item.id}`">
            <td>{{ item.id }}</td>
            <td>{{ item.customerId }}</td>
            <td>{{ item.text }}</td>
            <td>{{ item.status }}</td>
            <td>{{ item.dueAt ?? "-" }}</td>
            <td>
              <button
                v-if="item.status === 'pending'"
                type="button"
                class="ghost-button"
                @click="completeCustomerReminder(item.id)"
              >
                Виконано
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="customerReminders.length === 0" class="subtle">Нагадувань поки немає.</p>
    </section>
    
  </main>
</template>
