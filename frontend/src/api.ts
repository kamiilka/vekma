import type {
  AppUser,
  AuditLog,
  Customer,
  CustomerReminder,
  CustomerOrder,
  CustomerOrderStatus,
  CashOperation,
  CashShift,
  CashShiftStatus,
  Cashbox,
  BackgroundJob,
  DebtPaymentHistoryEntry,
  DebtSummary,
  ExchangeRate,
  Notification,
  NotificationTemplate,
  Payment,
  PaymentMethod,
  ProfitabilityReport,
  CurrencyCode,
  Product,
  ProductCSVImportResult,
  PurchaseCSVImportResult,
  ProductPriceBulkUpdateRequest,
  ProductPriceBulkUpdateResult,
  ProductMergeResult,
  ProductDuplicate,
  Warehouse,
  WarehouseStock,
  WarehouseZone,
  WarehouseCell,
  CellStock,
  CellTransferFIFOItem,
  StockTransfer,
  StockTransferItem,
  Inventory,
  InventoryItem,
  Document,
  DocumentCSVImportResult,
  DocumentItem,
  DocumentStatus,
  DocumentTemplate,
  DocumentTemplatePreview,
  DocumentTemplateValidation,
  DocumentType,
  ReturnAvailability,
  TemplatePlaceholder,
  Reservation,
  Supplier,
  SupplierOrder,
  SupplierOrderStatus,
  SupplierOrderReceiveLine,
  SupplierOrderPendingSummary,
  PurchaseRecommendation,
  PurchaseRecommendationGroup,
  PurchaseRecommendationCreateOrderRequest,
  PurchaseRecommendationOrderLine,
  Purchase,
  PurchaseItem,
  RolesResponse,
  Service,
  ServiceCategory,
  ServiceOrder,
  ServiceOrderStatus,
  ReceiptBulkRetryResult,
  Receipt,
  ReceiptStatus,
  Sale,
  SaleItem,
  StockMovement,
  Summary,
  UserSession,
  UserSessionInfo,
  DocumentRegistryItem,
  OrderChain,
  Counterparty,
  SupplierReport,
  CounterpartyReport,
} from "./types";

const API_BASE = import.meta.env.VITE_API_URL ?? "/api/v1";
export const SESSION_EXPIRED = "SESSION_EXPIRED";

async function request<T>(path: string, options: RequestInit = {}, token?: string): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error(SESSION_EXPIRED);
    }
    const data = await response.json().catch(() => ({ error: "Request failed" }));
    throw new Error(data.error ?? "Request failed");
  }

  return response.json() as Promise<T>;
}

export const api = {
  login(username: string, password: string) {
    return request<UserSession>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password })
    });
  },
  me(token: string) {
    return request<UserSessionInfo>("/auth/me", {}, token);
  },
  products(token: string) {
    return request<Product[]>("/products", {}, token);
  },
  createProduct(token: string, payload: Partial<Product>) {
    return request<Product>(
      "/products",
      {
        method: "POST",
        body: JSON.stringify(payload)
      },
      token
    );
  },
  updateProduct(token: string, productId: number, payload: Partial<Product>) {
    return request<Product>(
      `/products/${productId}`,
      {
        method: "PUT",
        body: JSON.stringify(payload)
      },
      token
    );
  },
  archiveProduct(token: string, productId: number, archived: boolean) {
    return request<{ status: string }>(
      `/products/${productId}/archive`,
      {
        method: "POST",
        body: JSON.stringify({ archived })
      },
      token
    );
  },
  generateProductBarcode(token: string, productId: number) {
    return request<Product>(`/products/${productId}/generate-barcode`, {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  async labelsPdf(token: string, ids: number[], format: "small" | "large" = "small") {
    const idsStr = ids.join(",");
    const response = await fetch(`${API_BASE}/products/labels.pdf?ids=${idsStr}&format=${format}`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    if (!response.ok) {
      if (response.status === 401) throw new Error(SESSION_EXPIRED);
      throw new Error("Failed to generate labels PDF");
    }
    return response.blob();
  },
  productQrSvgUrl(token: string, productId: number) {
    return `${API_BASE}/products/${productId}/qr.svg`;
  },
  productDuplicates(token: string) {
    return request<ProductDuplicate[]>("/products/duplicates", {}, token);
  },
  async exportProductsCsv(token: string, includeArchived = false) {
    const query = includeArchived ? "?includeArchived=true" : "";
    const response = await fetch(`${API_BASE}/products/export-csv${query}`, {
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    if (!response.ok) {
      if (response.status === 401) {
        throw new Error(SESSION_EXPIRED);
      }
      const data = await response.json().catch(() => ({ error: "Request failed" }));
      throw new Error(data.error ?? "Request failed");
    }
    return response.blob();
  },
  importProductsCsv(token: string, payload: { csv: string; updateExisting: boolean }) {
    return request<ProductCSVImportResult>("/products/import-csv", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  async exportProductsXlsx(token: string, includeArchived = false) {
    const query = includeArchived ? "?includeArchived=true" : "";
    const response = await fetch(`${API_BASE}/products/export-xlsx${query}`, {
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    if (!response.ok) {
      if (response.status === 401) {
        throw new Error(SESSION_EXPIRED);
      }
      const data = await response.json().catch(() => ({ error: "Request failed" }));
      throw new Error(data.error ?? "Request failed");
    }
    return response.blob();
  },
  async importProductsXlsx(token: string, payload: { file: File; updateExisting: boolean }) {
    const form = new FormData();
    form.set("file", payload.file);
    form.set("updateExisting", payload.updateExisting ? "true" : "false");
    const response = await fetch(`${API_BASE}/products/import-xlsx`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`
      },
      body: form
    });
    if (!response.ok) {
      if (response.status === 401) {
        throw new Error(SESSION_EXPIRED);
      }
      const data = await response.json().catch(() => ({ error: "Request failed" }));
      throw new Error(data.error ?? "Request failed");
    }
    return response.json() as Promise<ProductCSVImportResult>;
  },
  bulkUpdateProductPrices(token: string, payload: ProductPriceBulkUpdateRequest) {
    return request<ProductPriceBulkUpdateResult>("/products/prices/bulk", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  mergeDuplicateProducts(token: string, payload: { targetProductId: number; sourceProductIds: number[] }) {
    return request<ProductMergeResult>("/products/merge-duplicates", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createMovement(token: string, payload: { productId: number; warehouseId: number; type: string; quantity: number; note: string }) {
    return request("/stock/movements", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  stockMovements(token: string, params?: { productId?: number; warehouseId?: number }) {
    const query = new URLSearchParams();
    if (params?.productId) {
      query.set("productId", String(params.productId));
    }
    if (params?.warehouseId) {
      query.set("warehouseId", String(params.warehouseId));
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<StockMovement[]>(`/stock/movements${suffix}`, {}, token);
  },
  // --- Attachments ---
  listAttachments(token: string, entityType: string, entityId: number) {
    return request<import('./types').AttachmentItem[]>(
      `/attachments?entityType=${entityType}&entityId=${entityId}`, {}, token
    );
  },
  async uploadAttachment(token: string, entityType: string, entityId: number, file: File): Promise<import('./types').AttachmentItem> {
    const form = new FormData();
    form.append('entityType', entityType);
    form.append('entityId', String(entityId));
    form.append('file', file);
    const response = await fetch(`${API_BASE}/attachments`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: form
    });
    if (!response.ok) { const d = await response.json().catch(() => ({})); throw new Error(d.error ?? 'Upload failed'); }
    return response.json();
  },
  async downloadAttachment(token: string, id: number): Promise<Blob> {
    const response = await fetch(`${API_BASE}/attachments/${id}/download`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    if (!response.ok) { const d = await response.json().catch(() => ({})); throw new Error(d.error ?? 'Download failed'); }
    return response.blob();
  },
  deleteAttachment(token: string, id: number) {
    return request<void>(`/attachments/${id}`, { method: 'DELETE' }, token);
  },

  productLifecycle(token: string, productId: number) {
    return request<import('./types').ProductLifecycle>(`/products/${productId}/lifecycle`, {}, token);
  },
  createSale(token: string, payload: { items: SaleItem[]; orderId?: number; currency?: CurrencyCode }) {
    return request<Sale>("/sales", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  receipts(token: string, params?: { saleId?: number; status?: ReceiptStatus }) {
    const query = new URLSearchParams();
    if (params?.saleId) {
      query.set("saleId", String(params.saleId));
    }
    if (params?.status) {
      query.set("status", params.status);
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<Receipt[]>(`/receipts${suffix}`, {}, token);
  },
  receipt(token: string, receiptId: number) {
    return request<Receipt>(`/receipts/${receiptId}`, {}, token);
  },
  retryReceipt(token: string, receiptId: number) {
    return request<Receipt>(`/receipts/${receiptId}/retry`, {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  sendReceiptForSale(token: string, saleId: number) {
    return request<Receipt>(`/sales/${saleId}/send-receipt`, {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  retryReceiptsBulk(token: string, payload?: { status?: "pending" | "failed"; limit?: number }) {
    return request<ReceiptBulkRetryResult>("/receipts/retry-bulk", {
      method: "POST",
      body: JSON.stringify(payload ?? {})
    }, token);
  },
  createCustomer(token: string, payload: { name: string; phone: string; email: string; comment: string }) {
    return request<Customer>("/customers", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  customers(token: string, search?: string) {
    const query = new URLSearchParams();
    if (search && search.trim() !== "") {
      query.set("search", search.trim());
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<Customer[]>(`/customers${suffix}`, {}, token);
  },
  createCustomerReminder(token: string, payload: { customerId: number; text: string; dueAt?: string }) {
    return request<CustomerReminder>("/customer-reminders", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  customerReminders(token: string, params?: { customerId?: number; status?: "pending" | "done"; overdueOnly?: boolean }) {
    const query = new URLSearchParams();
    if (params?.customerId) {
      query.set("customerId", String(params.customerId));
    }
    if (params?.status) {
      query.set("status", params.status);
    }
    if (params?.overdueOnly) {
      query.set("overdueOnly", "true");
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<CustomerReminder[]>(`/customer-reminders${suffix}`, {}, token);
  },
  completeCustomerReminder(token: string, reminderId: number) {
    return request<CustomerReminder>(`/customer-reminders/${reminderId}/complete`, {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  createServiceOrder(token: string, payload: {
    customerId: number;
    productId?: number;
    title: string;
    description: string;
    technician?: string;
    laborMin?: number;
    price: number;
    currency?: CurrencyCode;
  }) {
    return request<ServiceOrder>("/service-orders", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  serviceOrders(token: string, params?: { customerId?: number; status?: ServiceOrderStatus }) {
    const query = new URLSearchParams();
    if (params?.customerId) {
      query.set("customerId", String(params.customerId));
    }
    if (params?.status) {
      query.set("status", params.status);
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<ServiceOrder[]>(`/service-orders${suffix}`, {}, token);
  },
  updateServiceOrderStatus(token: string, orderId: number, status: ServiceOrderStatus) {
    return request<ServiceOrder>(`/service-orders/${orderId}/status`, {
      method: "PUT",
      body: JSON.stringify({ status })
    }, token);
  },
  updateServiceOrder(token: string, orderId: number, payload: {
    productId?: number;
    title: string;
    description: string;
    technician: string;
    laborMin: number;
    price: number;
    currency?: CurrencyCode;
  }) {
    return request<ServiceOrder>(`/service-orders/${orderId}`, {
      method: "PUT",
      body: JSON.stringify(payload)
    }, token);
  },
  addServiceOrderPart(token: string, orderId: number, payload: {
    productId: number;
    quantity: number;
    price: number;
  }) {
    return request<ServiceOrder>(`/service-orders/${orderId}/parts`, {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createServiceOrderActDocument(token: string, orderId: number, payload: { note: string; autoPost?: boolean }) {
    return request<Document>(`/service-orders/${orderId}/act`, {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  async serviceOrderActPdf(token: string, orderId: number) {
    const response = await fetch(`${API_BASE}/service-orders/${orderId}/act/pdf`, {
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    if (!response.ok) {
      if (response.status === 401) {
        throw new Error(SESSION_EXPIRED);
      }
      const data = await response.json().catch(() => ({ error: "Request failed" }));
      throw new Error(data.error ?? "Request failed");
    }
    return response.blob();
  },
  cancelServiceOrderActDocument(token: string, orderId: number) {
    return request<Document>(`/service-orders/${orderId}/act/cancel`, {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  serviceCategories(token: string) {
    return request<ServiceCategory[]>("/service-categories", {}, token);
  },
  createServiceCategory(token: string, payload: { name: string }) {
    return request<ServiceCategory>("/service-categories", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  services(token: string) {
    return request<Service[]>("/services", {}, token);
  },
  createService(token: string, payload: { categoryId: number; name: string; price: number; currency?: CurrencyCode; durationMin: number }) {
    return request<Service>("/services", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createOrder(token: string, payload: { customerName: string; items: SaleItem[]; reserve: boolean; reserveUntil?: string; dueDate?: string; currency?: CurrencyCode }) {
    return request<CustomerOrder>("/orders", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  orders(token: string) {
    return request<CustomerOrder[]>("/orders", {}, token);
  },
  updateOrderStatus(token: string, orderId: number, status: CustomerOrderStatus) {
    return request<{ status: string }>(
      `/orders/${orderId}/status`,
      {
        method: "PUT",
        body: JSON.stringify({ status })
      },
      token
    );
  },
  updateOrder(token: string, orderId: number, payload: { customerName: string; currency: CurrencyCode; dueDate?: string; items: SaleItem[] }) {
    return request<CustomerOrder>(
      `/orders/${orderId}`,
      {
        method: "PUT",
        body: JSON.stringify(payload)
      },
      token
    );
  },
  reservations(token: string, status?: string) {
    const suffix = status ? `?status=${encodeURIComponent(status)}` : "";
    return request<Reservation[]>(`/reservations${suffix}`, {}, token);
  },
  createWarehouse(token: string, payload: { name: string; isVirtual?: boolean; locationType?: string }) {
    return request<Warehouse>("/warehouses", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  warehouses(token: string) {
    return request<Warehouse[]>("/warehouses", {}, token);
  },
  warehouseStocks(token: string, params?: { warehouseId?: number; productId?: number }) {
    const query = new URLSearchParams();
    if (params?.warehouseId) {
      query.set("warehouseId", String(params.warehouseId));
    }
    if (params?.productId) {
      query.set("productId", String(params.productId));
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<WarehouseStock[]>(`/warehouses/stocks${suffix}`, {}, token);
  },
  createWarehouseZone(token: string, payload: { warehouseId: number; name: string }) {
    return request<WarehouseZone>("/warehouse-zones", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  warehouseZones(token: string, warehouseId?: number) {
    const suffix = warehouseId ? `?warehouseId=${warehouseId}` : "";
    return request<WarehouseZone[]>(`/warehouse-zones${suffix}`, {}, token);
  },
  createWarehouseCell(token: string, payload: { warehouseId: number; zoneId: number; code: string }) {
    return request<WarehouseCell>("/warehouse-cells", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  warehouseCells(token: string, zoneId?: number) {
    const suffix = zoneId ? `?zoneId=${zoneId}` : "";
    return request<WarehouseCell[]>(`/warehouse-cells${suffix}`, {}, token);
  },
  cellStocks(token: string, params?: { cellId?: number; productId?: number }) {
    const query = new URLSearchParams();
    if (params?.cellId) {
      query.set("cellId", String(params.cellId));
    }
    if (params?.productId) {
      query.set("productId", String(params.productId));
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<CellStock[]>(`/warehouse-cells/stocks${suffix}`, {}, token);
  },
  createTransfer(token: string, payload: { fromWarehouseId: number; toWarehouseId: number; items: StockTransferItem[]; note: string }) {
    return request<StockTransfer>("/transfers", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createCellTransfer(token: string, payload: { fromCellId: number; toCellId: number; items: StockTransferItem[]; note: string }) {
    return request<StockTransfer>("/transfers/cell", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createCellTransferFIFO(token: string, payload: { toCellId: number; items: CellTransferFIFOItem[]; note: string }) {
    return request<StockTransfer>("/transfers/cell/fifo", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createInventory(token: string, payload: { warehouseId: number; items: InventoryItem[]; note: string }) {
    return request<Inventory>("/inventories", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  sales(token: string) {
    return request<Sale[]>("/sales", {}, token);
  },
  inventories(token: string) {
    return request<Inventory[]>("/inventories", {}, token);
  },
  applyInventory(token: string, inventoryId: number) {
    return request<Inventory>(`/inventories/${inventoryId}/apply`, {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  createDocument(
    token: string,
    payload: {
      type: DocumentType;
      warehouseId?: number;
      cashboxId?: number;
      currency?: CurrencyCode;
      total?: number;
      items: DocumentItem[];
      note: string;
    }
  ) {
    return request<Document>("/documents", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  importDocumentCsv(
    token: string,
    payload: {
      type: DocumentType;
      warehouseId?: number;
      cashboxId?: number;
      currency?: CurrencyCode;
      total?: number;
      csv: string;
      note: string;
    }
  ) {
    return request<DocumentCSVImportResult>("/documents/import-csv", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  documents(token: string, params?: { type?: DocumentType; status?: DocumentStatus }) {
    const query = new URLSearchParams();
    if (params?.type) {
      query.set("type", params.type);
    }
    if (params?.status) {
      query.set("status", params.status);
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<Document[]>(`/documents${suffix}`, {}, token);
  },
  postDocument(token: string, documentId: number) {
    return request<Document>(`/documents/${documentId}/post`, {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  customerReturnAvailability(token: string, saleId: number) {
    return request<ReturnAvailability>(`/documents/returns/customer/${saleId}/available`, {}, token);
  },
  createCustomerReturnDocument(
    token: string,
    payload: {
      saleId: number;
      warehouseId: number;
      currency?: CurrencyCode;
      items: DocumentItem[];
      note: string;
    }
  ) {
    return request<Document>("/documents/returns/customer", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  supplierReturnAvailability(token: string, purchaseId: number) {
    return request<ReturnAvailability>(`/documents/returns/supplier/${purchaseId}/available`, {}, token);
  },
  createSupplierReturnDocument(
    token: string,
    payload: {
      purchaseId: number;
      warehouseId: number;
      currency?: CurrencyCode;
      items: DocumentItem[];
      note: string;
    }
  ) {
    return request<Document>("/documents/returns/supplier", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  async fetchDocumentPdf(token: string, documentId: number) {
    const response = await fetch(`${API_BASE}/documents/${documentId}/pdf`, {
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    if (!response.ok) {
      if (response.status === 401) {
        throw new Error(SESSION_EXPIRED);
      }
      const data = await response.json().catch(() => ({ error: "Request failed" }));
      throw new Error(data.error ?? "Request failed");
    }
    return response.blob();
  },
  documentTemplates(token: string) {
    return request<DocumentTemplate[]>("/document-templates", {}, token);
  },
  upsertDocumentTemplate(
    token: string,
    payload: {
      code: string;
      name: string;
      body: string;
      isActive: boolean;
    }
  ) {
    return request<DocumentTemplate>("/document-templates", {
      method: "PUT",
      body: JSON.stringify(payload)
    }, token);
  },
  documentTemplatePlaceholders(token: string) {
    return request<TemplatePlaceholder[]>("/document-templates/placeholders", {}, token);
  },
  validateDocumentTemplate(
    token: string,
    payload: {
      code: string;
      body: string;
      documentId?: number;
      strict?: boolean;
    }
  ) {
    return request<DocumentTemplateValidation>("/document-templates/validate", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  previewDocumentTemplate(
    token: string,
    payload: {
      code: string;
      body: string;
      documentId?: number;
      strict?: boolean;
    }
  ) {
    return request<DocumentTemplatePreview>("/document-templates/preview", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createSupplier(token: string, payload: { name: string; contact?: string; phone?: string; email?: string; comments?: string }) {
    return request<Supplier>("/suppliers", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  suppliers(token: string) {
    return request<Supplier[]>("/suppliers", {}, token);
  },
  createSupplierOrder(token: string, payload: { supplierId: number; currency?: CurrencyCode; items: PurchaseItem[] }) {
    return request<SupplierOrder>("/supplier-orders", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  supplierOrders(token: string) {
    return request<SupplierOrder[]>("/supplier-orders", {}, token);
  },
  supplierOrdersPending(token: string) {
    return request<SupplierOrderPendingSummary[]>("/supplier-orders/pending", {}, token);
  },
  supplierOrderRecommendations(token: string, limit?: number) {
    const suffix = limit ? `?limit=${limit}` : "";
    return request<PurchaseRecommendation[]>(`/supplier-orders/recommendations${suffix}`, {}, token);
  },
  supplierOrderRecommendationsGrouped(token: string, limit?: number) {
    const suffix = limit ? `?limit=${limit}` : "";
    return request<PurchaseRecommendationGroup[]>(`/supplier-orders/recommendations/grouped${suffix}`, {}, token);
  },
  createSupplierOrderFromRecommendations(
    token: string,
    payload: {
      supplierId: number;
      currency?: CurrencyCode;
      items: PurchaseRecommendationOrderLine[];
    }
  ) {
    return request<SupplierOrder>("/supplier-orders/recommendations/create-order", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createSupplierOrdersBulkFromRecommendations(
    token: string,
    payload: {
      orders: PurchaseRecommendationCreateOrderRequest[];
    }
  ) {
    return request<SupplierOrder[]>("/supplier-orders/recommendations/create-orders-bulk", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  updateSupplierOrderStatus(token: string, supplierOrderId: number, status: SupplierOrderStatus) {
    return request<{ status: string }>(
      `/supplier-orders/${supplierOrderId}/status`,
      {
        method: "PUT",
        body: JSON.stringify({ status })
      },
      token
    );
  },
  updateSupplierOrder(token: string, supplierOrderId: number, payload: { customerOrderId?: number | null; supplierId?: number }) {
    return request<{ status: string }>(
      `/supplier-orders/${supplierOrderId}`,
      {
        method: "PUT",
        body: JSON.stringify(payload)
      },
      token
    );
  },
  journal(token: string, params?: { search?: string; user?: string; entity?: string; from?: string; to?: string; limit?: number }) {
    const p = new URLSearchParams();
    if (params?.search) p.set("search", params.search);
    if (params?.user)   p.set("user",   params.user);
    if (params?.entity) p.set("entity", params.entity);
    if (params?.from)   p.set("from",   params.from);
    if (params?.to)     p.set("to",     params.to);
    if (params?.limit)  p.set("limit",  String(params.limit));
    return request<AuditLog[]>(`/journal?${p}`, {}, token);
  },
  searchDocuments(token: string, query: string, types?: string[], limit?: number) {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (types?.length) params.set("types", types.join(","));
    if (limit) params.set("limit", String(limit));
    return request<DocumentRegistryItem[]>(`/documents/registry?${params}`, {}, token);
  },
  getOrderChain(token: string, orderId: number) {
    return request<OrderChain>(`/orders/${orderId}/chain`, {}, token);
  },
  counterparties(token: string) {
    return request<Counterparty[]>("/counterparties", {}, token);
  },
  createCounterparty(token: string, payload: { name: string; phone: string; email: string; comment: string; isCustomer: boolean; isSupplier: boolean }) {
    return request<Counterparty>("/counterparties", { method: "POST", body: JSON.stringify(payload) }, token);
  },
  updateCounterparty(token: string, id: number, payload: { name: string; phone: string; email: string; comment: string; isCustomer: boolean; isSupplier: boolean }) {
    return request<Counterparty>(`/counterparties/${id}`, { method: "PUT", body: JSON.stringify(payload) }, token);
  },
  createPurchase(token: string, payload: { supplierId: number; supplierOrderId?: number; currency?: CurrencyCode; items: PurchaseItem[]; note: string }) {
    return request<Purchase>("/purchases", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  importPurchaseCsv(token: string, payload: {
    supplierId: number;
    supplierOrderId?: number;
    currency?: CurrencyCode;
    csv: string;
    note: string;
  }) {
    return request<PurchaseCSVImportResult>("/purchases/import-csv", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  purchases(token: string, supplierOrderId?: number) {
    const suffix = supplierOrderId ? `?supplierOrderId=${supplierOrderId}` : "";
    return request<Purchase[]>(`/purchases${suffix}`, {}, token);
  },
  receiveSupplierOrderByLines(
    token: string,
    supplierOrderId: number,
    payload: { currency?: CurrencyCode; lines: SupplierOrderReceiveLine[]; note: string }
  ) {
    return request<Purchase>(`/supplier-orders/${supplierOrderId}/receive`, {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  createPayment(token: string, payload: { orderId?: number; saleId?: number; serviceOrderId?: number; supplierOrderId?: number; cashboxId?: number; amount: number; currency?: CurrencyCode; method: PaymentMethod; note: string }) {
    return request<Payment>("/payments", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  payments(token: string, params?: { orderId?: number; saleId?: number; serviceOrderId?: number; supplierOrderId?: number }) {
    const query = new URLSearchParams();
    if (params?.orderId) {
      query.set("orderId", String(params.orderId));
    }
    if (params?.saleId) {
      query.set("saleId", String(params.saleId));
    }
    if (params?.serviceOrderId) {
      query.set("serviceOrderId", String(params.serviceOrderId));
    }
    if (params?.supplierOrderId) {
      query.set("supplierOrderId", String(params.supplierOrderId));
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<Payment[]>(`/payments${suffix}`, {}, token);
  },
  paymentsForOrder(token: string, orderId: number) {
    return request<Payment[]>(`/payments?orderId=${orderId}`, {}, token);
  },
  salesGrouped(token: string, groupBy: "day" | "month" = "month") {
    return request<Array<{ period: string; salesQty: number; revenue: number; profit: number }>>(
      `/analytics/sales-grouped?groupBy=${groupBy}`, {}, token
    );
  },
  async exportVATCsv(token: string) {
    const response = await fetch(`${API_BASE}/documents/export-vat-csv`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    if (!response.ok) {
      if (response.status === 401) throw new Error(SESSION_EXPIRED);
      throw new Error("Помилка експорту ПДВ");
    }
    return response.blob();
  },
  async exportVATXlsx(token: string) {
    const response = await fetch(`${API_BASE}/documents/export-vat-xlsx`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    if (!response.ok) {
      if (response.status === 401) throw new Error(SESSION_EXPIRED);
      throw new Error("Помилка експорту ПДВ XLSX");
    }
    return response.blob();
  },
  supplierPortalUrl() {
    // The static token must be set in SUPPLIER_PORTAL_TOKEN env var on the server.
    // Frontend just builds the shareable link; actual auth is server-side.
    return `${API_BASE}/public/supplier-deficit?token=`;
  },
  debts(token: string) {
    return request<DebtSummary[]>("/debts", {}, token);
  },
  overdueDebts(token: string, asOf?: string) {
    const suffix = asOf ? `?asOf=${encodeURIComponent(asOf)}` : "";
    return request<DebtSummary[]>(`/debts/overdue${suffix}`, {}, token);
  },
  debtPaymentHistory(token: string, entityType: "order" | "sale" | "service_order", entityId: number) {
    const query = `?entityType=${encodeURIComponent(entityType)}&entityId=${entityId}`;
    return request<DebtPaymentHistoryEntry[]>(`/debts/history${query}`, {}, token);
  },
  notificationTemplates(token: string, code?: string) {
    const suffix = code ? `?code=${encodeURIComponent(code)}` : "";
    return request<NotificationTemplate[]>(`/notification-templates${suffix}`, {}, token);
  },
  upsertNotificationTemplate(
    token: string,
    payload: { code: string; channel: "email" | "telegram"; subject: string; body: string; isActive: boolean }
  ) {
    return request<NotificationTemplate>("/notification-templates", {
      method: "PUT",
      body: JSON.stringify(payload)
    }, token);
  },
  notifications(token: string, limit = 50) {
    return request<Notification[]>(`/notifications?limit=${limit}`, {}, token);
  },
  enqueueOverdueReminderJob(token: string, asOf?: string) {
    return request<BackgroundJob>("/jobs/overdue-reminders", {
      method: "POST",
      body: JSON.stringify(asOf ? { asOf } : {})
    }, token);
  },
  enqueueReservationExpiryJob(token: string, asOf?: string) {
    return request<BackgroundJob>("/jobs/reservation-expiry", {
      method: "POST",
      body: JSON.stringify(asOf ? { asOf } : {})
    }, token);
  },
  enqueueReceiptRetryJob(token: string, payload?: { status?: "pending" | "failed"; limit?: number }) {
    return request<BackgroundJob>("/jobs/receipts-retry", {
      method: "POST",
      body: JSON.stringify(payload ?? {})
    }, token);
  },
  runBackgroundJobs(token: string) {
    return request<{ processed: BackgroundJob[]; count: number }>("/jobs/run", {
      method: "POST",
      body: JSON.stringify({})
    }, token);
  },
  backgroundJobs(token: string, limit = 50) {
    return request<BackgroundJob[]>(`/jobs?limit=${limit}`, {}, token);
  },
  createCashbox(token: string, payload: { name: string; type: PaymentMethod; currency: CurrencyCode }) {
    return request<Cashbox>("/cashboxes", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  cashboxes(token: string) {
    return request<Cashbox[]>("/cashboxes", {}, token);
  },
  createCashOperation(token: string, payload: { cashboxId: number; type: "incoming" | "outgoing"; amount: number; method: PaymentMethod; description: string }) {
    return request<CashOperation>("/cash-operations", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  cashOperations(token: string, cashboxId?: number) {
    const suffix = cashboxId ? `?cashboxId=${cashboxId}` : "";
    return request<CashOperation[]>(`/cash-operations${suffix}`, {}, token);
  },
  openCashShift(token: string, payload: { cashboxId: number; note?: string }) {
    return request<CashShift>("/cash-shifts/open", {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  closeCashShift(token: string, shiftId: number, payload: { note?: string }) {
    return request<CashShift>(`/cash-shifts/${shiftId}/close`, {
      method: "POST",
      body: JSON.stringify(payload)
    }, token);
  },
  cashShifts(token: string, params?: { cashboxId?: number; status?: CashShiftStatus }) {
    const query = new URLSearchParams();
    if (params?.cashboxId) {
      query.set("cashboxId", String(params.cashboxId));
    }
    if (params?.status) {
      query.set("status", params.status);
    }
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return request<CashShift[]>(`/cash-shifts${suffix}`, {}, token);
  },
  exchangeRates(token: string) {
    return request<ExchangeRate[]>("/exchange-rates", {}, token);
  },
  upsertExchangeRate(token: string, payload: { currency: CurrencyCode; rateToUah: number }) {
    return request<ExchangeRate>("/exchange-rates", {
      method: "PUT",
      body: JSON.stringify(payload)
    }, token);
  },
  summary(token: string) {
    return request<Summary>("/analytics/summary", {}, token);
  },
  profitability(token: string) {
    return request<ProfitabilityReport>("/analytics/profitability", {}, token);
  },
  auditLogs(token: string, limit = 30) {
    return request<AuditLog[]>(`/audit/logs?limit=${limit}`, {}, token);
  },
  users(token: string) {
    return request<AppUser[]>("/users", {}, token);
  },
  createUser(token: string, payload: { username: string; password: string; role: string }) {
    return request<AppUser>(
      "/users",
      {
        method: "POST",
        body: JSON.stringify(payload)
      },
      token
    );
  },
  roles(token: string) {
    return request<RolesResponse>("/roles", {}, token);
  },
  updateRolePermissions(token: string, role: string, permissions: string[]) {
    return request<{ status: string }>(
      `/roles/${role}/permissions`,
      {
        method: "PUT",
        body: JSON.stringify({ permissions })
      },
      token
    );
  },
  sendQuickMessage(token: string, payload: { channel: string; recipient: string; sender?: string; subject?: string; body: string; entityType?: string; entityId?: number }) {
    return request<{ id: number; status: string }>(
      "/notifications/quick-message",
      {
        method: "POST",
        body: JSON.stringify(payload)
      },
      token
    );
  },
  getNotificationConfig(token: string) {
    return request<{
      smtpAddr: string; smtpUser: string; smtpPass: string; smtpFrom: string;
      telegramToken: string; telegramChatId: string;
      smsGatewayUrl: string; smsGatewayToken: string; smsPhoneTo: string;
      viberToken: string; viberRecipient: string;
    }>("/settings/notification", {}, token);
  },
  saveNotificationConfig(token: string, cfg: Record<string, string>) {
    return request<{ status: string }>("/settings/notification", { method: "PUT", body: JSON.stringify(cfg) }, token);
  },
  supplierReport(token: string) {
    return request<SupplierReport>("/reports/suppliers", {}, token);
  },
  counterpartyReport(token: string) {
    return request<CounterpartyReport>("/reports/counterparties", {}, token);
  },
};
