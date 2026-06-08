export type UserSession = {
  token: string;
  role: string;
  user: string;
  permissions?: string[];
  scopes?: UserAccessScopes;
  features?: { checkboxEnabled?: boolean };
};

export type UserAccessScopes = {
  warehouseIds: number[];
  cashboxIds: number[];
};

export type UserSessionInfo = {
  role: string;
  user: string;
  permissions: string[];
  scopes: UserAccessScopes;
  features?: { checkboxEnabled?: boolean };
};

export type AppUser = {
  username: string;
  role: string;
};

export type RolePermissions = {
  role: string;
  permissions: string[];
};

export type Product = {
  id: number;
  name: string;
  code?: string;
  sku: string;
  article?: string;
  barcode: string;
  serialNumber?: string;
  category: string;
  brand: string;
  supplier: string;
  purchasePrice: number;
  retailPrice: number;
  wholesalePrice?: number;
  currency?: string;
  vatPercent?: number;
  stock: number;
  minStock: number;
  warehousePosition?: string;
  comments?: string;
  archived?: boolean;
  supplierSku?: string;
  supplierNameExt?: string;
  createdAt: string;
};

export type ProductDuplicate = {
  field: string;
  value: string;
  productIds: number[];
  skus: string[];
  names: string[];
};

export type ProductCSVImportRowError = {
  line: number;
  error: string;
};

export type ProductCSVImportResult = {
  imported: number;
  updated: number;
  skipped: number;
  errors: ProductCSVImportRowError[];
};

export type PurchaseCSVImportResult = {
  purchase: Purchase;
  imported: number;
  skipped: number;
  errors: ProductCSVImportRowError[];
};

export type DocumentCSVImportResult = {
  document: Document;
  imported: number;
  skipped: number;
  errors: ProductCSVImportRowError[];
};

export type ProductPriceBulkUpdateRequest = {
  mode: "percent" | "fixed";
  value: number;
  priceField: "retailPrice" | "purchasePrice" | "wholesalePrice";
  roundMode?: "none" | "nearest" | "up" | "down";
  roundTo?: number;
  search?: string;
  category?: string;
  brand?: string;
  supplier?: string;
  includeArchived: boolean;
};

export type ProductPriceBulkUpdateResult = {
  updated: number;
};

export type ProductMergeResult = {
  targetProductId: number;
  mergedProductIds: number[];
};

export type Warehouse = {
  id: number;
  name: string;
  isVirtual: boolean;
  locationType: "warehouse" | "shop";
  createdAt: string;
};

export type WarehouseStock = {
  warehouseId: number;
  productId: number;
  quantity: number;
  updatedAt: string;
};

export type StockMovement = {
  id: number;
  productId: number;
  fromWarehouseId?: number;
  toWarehouseId?: number;
  fromCellId?: number;
  toCellId?: number;
  type: string;
  quantity: number;
  note: string;
  createdAt: string;
};

export type WarehouseZone = {
  id: number;
  warehouseId: number;
  name: string;
  createdAt: string;
};

export type WarehouseCell = {
  id: number;
  warehouseId: number;
  zoneId: number;
  code: string;
  createdAt: string;
};

export type CellStock = {
  cellId: number;
  productId: number;
  quantity: number;
  updatedAt: string;
};

export type StockTransferItem = {
  productId: number;
  quantity: number;
  fromCellId?: number;
  toCellId?: number;
};

export type CellTransferFIFOItem = {
  productId: number;
  quantity: number;
};

export type StockTransfer = {
  id: number;
  fromWarehouseId: number;
  toWarehouseId: number;
  items: StockTransferItem[];
  note: string;
  createdAt: string;
};

export type InventoryStatus = "draft" | "applied" | "canceled";

export type InventoryItem = {
  productId: number;
  systemQuantity: number;
  actualQuantity: number;
  adjustment: number;
};

export type Inventory = {
  id: number;
  warehouseId: number;
  status: InventoryStatus;
  items: InventoryItem[];
  note: string;
  appliedAt?: string;
  createdAt: string;
};

export type DocumentType =
  | "invoice"
  | "act"
  | "cash_in_order"
  | "cash_out_order"
  | "vat"
  | "return_from_customer"
  | "return_to_supplier";

export type DocumentStatus = "draft" | "posted" | "cancelled";

export type DocumentItem = {
  productId: number;
  quantity: number;
  price: number;
};

export type Document = {
  id: number;
  type: DocumentType;
  number: string;
  status: DocumentStatus;
  sourceSaleId?: number;
  sourcePurchaseId?: number;
  sourceServiceOrderId?: number;
  warehouseId?: number;
  cashboxId?: number;
  currency: CurrencyCode;
  total: number;
  items: DocumentItem[];
  note: string;
  postedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type ReturnAvailabilityItem = {
  productId: number;
  sourceQty: number;
  returnedQty: number;
  availableQty: number;
  price: number;
};

export type ReturnAvailability = {
  sourceId: number;
  currency: CurrencyCode;
  items: ReturnAvailabilityItem[];
};

export type DocumentTemplate = {
  id: number;
  code: string;
  name: string;
  body: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type TemplatePlaceholder = {
  name: string;
  description: string;
};

export type DocumentTemplateValidation = {
  valid: boolean;
  strict: boolean;
  used: string[];
  unknown: string[];
  missingRequired: string[];
  allowed: string[];
  required: string[];
  templateLen: number;
};

export type DocumentTemplatePreview = {
  content: string;
  validation: DocumentTemplateValidation;
};

export type Summary = {
  productCount: number;
  lowStock: number;
  totalStock: number;
  salesCount: number;
  revenue: number;
};

export type ProductProfitability = {
  productId: number;
  productName: string;
  sku: string;
  quantitySold: number;
  revenueUah: number;
  costUah: number;
  profitUah: number;
  marginPct: number;
};

export type ProfitabilityReport = {
  items: ProductProfitability[];
  totalRevenue: number;
  totalCost: number;
  totalProfit: number;
  marginPct: number;
};

export type SaleItem = {
  productId: number;
  quantity: number;
  price: number;
};

export type Sale = {
  id: number;
  orderId?: number;
  items: SaleItem[];
  total: number;
  currency: CurrencyCode;
  totalUah: number;
  status: string;
  createdAt: string;
};

export type ReceiptStatus = "pending" | "sent" | "failed";

export type Receipt = {
  id: number;
  saleId: number;
  provider: string;
  status: ReceiptStatus;
  externalId?: string;
  fiscalNumber?: string;
  qrUrl?: string;
  errorMessage?: string;
  payload?: string;
  sentAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type ReceiptBulkRetryResult = {
  attempted: number;
  succeeded: number;
  failed: number;
  items: Receipt[];
};

export type PurchaseItem = {
  productId: number;
  quantity: number;
  price: number;
};

export type SupplierOrderReceiveLine = {
  productId: number;
  quantity: number;
  price?: number;
};

export type SupplierOrderPendingItem = {
  productId: number;
  ordered: number;
  received: number;
  pending: number;
  price: number;
};

export type CustomerOrderStatus =
  | "new"
  | "in_work"
  | "ordered"
  | "expected"
  | "arrived"
  | "issued"
  | "closed"
  | "cancelled";

export type Customer = {
  id: number;
  name: string;
  phone: string;
  email: string;
  comment: string;
  createdAt: string;
  updatedAt: string;
};

export type CustomerReminderStatus = "pending" | "done";

export type CustomerReminder = {
  id: number;
  customerId: number;
  text: string;
  dueAt?: string;
  status: CustomerReminderStatus;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type ServiceOrderStatus = "new" | "in_progress" | "done" | "cancelled";

export type ServiceOrder = {
  id: number;
  customerId: number;
  productId?: number;
  title: string;
  description: string;
  technician: string;
  laborMin: number;
  status: ServiceOrderStatus;
  price: number;
  partsTotal: number;
  total: number;
  paid: number;
  debt: number;
  totalUah: number;
  paidUah: number;
  debtUah: number;
  currency: CurrencyCode;
  parts: ServiceOrderPart[];
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type ServiceOrderPart = {
  id: number;
  serviceOrderId: number;
  productId: number;
  quantity: number;
  price: number;
  total: number;
  createdAt: string;
};

export type ServiceCategory = {
  id: number;
  name: string;
  createdAt: string;
};

export type Service = {
  id: number;
  categoryId: number;
  name: string;
  price: number;
  currency: CurrencyCode;
  durationMin: number;
  createdAt: string;
};

export type CustomerOrder = {
  id: number;
  customerName: string;
  status: CustomerOrderStatus;
  currency: CurrencyCode;
  total: number;
  totalUah: number;
  dueDate?: string;
  items: SaleItem[];
  createdAt: string;
  updatedAt: string;
};

export type Supplier = {
  id: number;
  name: string;
  contact: string;
  phone: string;
  email: string;
  comments: string;
  createdAt: string;
};

export type SupplierOrderStatus =
  | "draft"
  | "sent"
  | "confirmed"
  | "in_transit"
  | "received"
  | "closed"
  | "cancelled";

export type SupplierOrder = {
  id: number;
  supplierId: number;
  customerOrderId?: number;
  status: SupplierOrderStatus;
  currency: CurrencyCode;
  total: number;
  totalUah: number;
  items: PurchaseItem[];
  createdAt: string;
  updatedAt: string;
};

export type SupplierOrderPendingSummary = {
  orderId: number;
  supplierId: number;
  currency: CurrencyCode;
  status: SupplierOrderStatus;
  pendingItems: SupplierOrderPendingItem[];
  pendingTotal: number;
  pendingTotalUah: number;
  updatedAt: string;
};

export type PurchaseRecommendation = {
  productId: number;
  productName: string;
  sku: string;
  supplier: string;
  suggestedSupplierId?: number;
  currentStock: number;
  reserved: number;
  available: number;
  minStock: number;
  soldLast30Days: number;
  recommendedQty: number;
};

export type PurchaseRecommendationOrderLine = {
  productId: number;
  quantity: number;
  price?: number;
};

export type PurchaseRecommendationGroup = {
  supplierId?: number;
  supplierName: string;
  items: PurchaseRecommendation[];
  totalRecommended: number;
  productsCount: number;
};

export type PurchaseRecommendationCreateOrderRequest = {
  supplierId: number;
  currency?: CurrencyCode;
  items: PurchaseRecommendationOrderLine[];
};

export type Purchase = {
  id: number;
  supplierId: number;
  supplierOrderId?: number;
  currency: CurrencyCode;
  total: number;
  totalUah: number;
  items: PurchaseItem[];
  note: string;
  createdAt: string;
};

export type Reservation = {
  id: number;
  orderId: number;
  productId: number;
  quantity: number;
  status: "active" | "released" | "consumed";
  expiresAt?: string;
  releasedAt?: string;
  createdAt: string;
};

export type Payment = {
  id: number;
  orderId?: number;
  saleId?: number;
  serviceOrderId?: number;
  cashboxId: number;
  amount: number;
  currency: CurrencyCode;
  amountUah: number;
  method: PaymentMethod;
  note: string;
  createdAt: string;
};

export type DebtSummary = {
  entityType: "order" | "sale" | "service_order" | "supplier_order";
  entityId: number;
  currency: CurrencyCode;
  total: number;
  paid: number;
  debt: number;
  totalUah: number;
  paidUah: number;
  debtUah: number;
  dueDate?: string;
  isOverdue: boolean;
  overdueDays: number;
  lastPaymentAt?: string;
};

export type PaymentMethod = "cash" | "card" | "bank" | "virtual";
export type CurrencyCode = "UAH" | "USD" | "EUR" | "EUR";

export type Cashbox = {
  id: number;
  name: string;
  type: PaymentMethod;
  currency: string;
  balance: number;
  createdAt: string;
};

export type CashOperation = {
  id: number;
  cashboxId: number;
  type: "incoming" | "outgoing";
  amount: number;
  currency: CurrencyCode;
  amountUah: number;
  method: PaymentMethod;
  paymentId?: number;
  description: string;
  createdAt: string;
};

export type CashShiftStatus = "open" | "closed";

export type CashShift = {
  id: number;
  cashboxId: number;
  status: CashShiftStatus;
  openedBy: string;
  closedBy?: string;
  openingBalance: number;
  closingBalance?: number;
  note: string;
  openedAt: string;
  closedAt?: string;
};

export type ExchangeRate = {
  currency: CurrencyCode;
  rateToUah: number;
  updatedAt: string;
};

export type DebtPaymentHistoryEntry = {
  paymentId: number;
  amount: number;
  amountUah: number;
  currency: CurrencyCode;
  method: PaymentMethod;
  note: string;
  createdAt: string;
  remainingDebt: number;
  remainingDebtUah: number;
};

export type NotificationTemplate = {
  id: number;
  code: string;
  channel: "email" | "telegram" | "sms" | "viber";
  subject: string;
  body: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type Notification = {
  id: number;
  channel: "email" | "telegram";
  recipient: string;
  subject: string;
  body: string;
  entityType: string;
  entityId: number;
  status: "queued" | "sent" | "sent_stub" | "failed";
  attempts: number;
  errorMessage?: string;
  sentAt?: string;
  createdAt: string;
};

export type BackgroundJob = {
  id: number;
  jobType: string;
  status: "queued" | "running" | "completed" | "failed";
  attempts: number;
  maxAttempts: number;
  nextRetryAt?: string;
  payload: string;
  result: string;
  errorMessage?: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt: string;
};

export type AuditLog = {
  id: number;
  user: string;
  action: string;
  entity: string;
  details: string;
  createdAt: string;
};

export type RolesResponse = {
  roles: RolePermissions[];
  permissions: string[];
};

export type Counterparty = {
  id: number;
  name: string;
  phone: string;
  email: string;
  comment: string;
  isCustomer: boolean;
  isSupplier: boolean;
  customerId?: number;
  supplierId?: number;
  createdAt: string;
  updatedAt: string;
};

export type OrderChainNode = {
  type: "customer_order" | "supplier_order" | "purchase" | "sale" | "payment" | "supplier_payment";
  id: number;
  label: string;
  status?: string;
  amount?: number;
  currency?: string;
  date: string;
  children?: OrderChainNode[];
};

export type OrderChain = {
  root: OrderChainNode;
};

export type DocumentRegistryItem = {
  docType: string;
  id: number;
  number: string;
  status?: string;
  counterName?: string;
  total?: number;
  currency?: string;
  productHits?: string[];
  note?: string;
  date: string;
};

export interface ProductLifecycleEvent {
  eventType: 'created' | 'purchased' | 'received' | 'sold' | 'returned_from_customer' | 'returned_to_supplier' | 'movement';
  eventDate: string;
  refId?: number;
  quantity?: number;
  price?: number;
  currency?: string;
  supplierId?: number;
  supplierName?: string;
  customerName?: string;
  warehouseName?: string;
  note?: string;
}

export interface ProductLifecycle {
  productId: number;
  events: ProductLifecycleEvent[];
}

export interface AttachmentItem {
  id: number;
  entityType: string;
  entityId: number;
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  createdAt: string;
}

// ── Report types ──────────────────────────────────────────────────────────────

export type SupplierReportRow = {
  supplierId: number;
  supplierName: string;
  ordersCount: number;
  purchasedUah: number;
  paidUah: number;
  debtUah: number;
};

export type SupplierReport = {
  rows: SupplierReportRow[];
  totalPurchasedUah: number;
  totalPaidUah: number;
  totalDebtUah: number;
};

export type CounterpartyReportRow = {
  counterpartyId: number;
  counterpartyName: string;
  isCustomer: boolean;
  isSupplier: boolean;
  salesUah: number;
  purchasedUah: number;
  paidUah: number;
  debtUah: number;
};

export type CounterpartyReport = {
  rows: CounterpartyReportRow[];
  totalSalesUah: number;
  totalPurchasedUah: number;
  totalPaidUah: number;
  totalDebtUah: number;
};
