<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from "vue";
import { api } from "../api";
import BarcodeScanner from "./BarcodeScanner.vue";
import type { UserSession, Product, Warehouse, WarehouseStock, StockMovement, Supplier } from "../types";

const props = defineProps<{ session: UserSession; initialSubTab?: string; initialProductId?: number }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);
const can = (p: string) => props.session.permissions?.includes(p) ?? false;

// ── Invoice (накладна) import ──────────────────────────────────────────────
const showInvoiceImport = ref(false);
const invoiceFile = ref<File | null>(null);
const invoiceImporting = ref(false);
const invoiceResult = ref<{ created: number; found: number; supplierName: string; purchaseId?: number } | null>(null);
const invoiceError = ref("");
const invoiceSuppliers = ref<Supplier[]>([]);
const productFormSuppliers = ref<Supplier[]>([]);
const supplierSearch = ref('');
const showSupplierDropdown = ref(false);

function hideDropdown(setter: (v: boolean) => void) {
  setTimeout(() => setter(false), 150);
}

function onInvoiceFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  invoiceFile.value = input.files?.[0] ?? null;
  invoiceResult.value = null;
  invoiceError.value = "";
}

interface InvoiceRow { barcode: string; name: string; qty: number; price: number; }

async function loadSheetJS(): Promise<any> {
  // @ts-ignore
  if (typeof window.XLSX !== "undefined") return window.XLSX;
  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = "https://cdn.sheetjs.com/xlsx-0.20.3/package/dist/xlsx.full.min.js";
    script.onload = () => resolve((window as any).XLSX);
    script.onerror = () => reject(new Error("Не вдалося завантажити бібліотеку xlsx"));
    document.head.appendChild(script);
  });
}

function parseInvoiceSheet(XLSX: any, buffer: ArrayBuffer): { supplierName: string; rows: InvoiceRow[] } {
  const wb = XLSX.read(buffer, { type: "array" });
  const ws = wb.Sheets[wb.SheetNames[0]];
  const raw: any[][] = XLSX.utils.sheet_to_json(ws, { header: 1, defval: "" });

  let supplierName = "";
  for (let r = 0; r < Math.min(10, raw.length); r++) {
    for (let c = 0; c < raw[r].length; c++) {
      const cell = String(raw[r][c]).trim();
      if (cell === "Постачальник") {
        for (let cc = c + 1; cc < raw[r].length; cc++) {
          const v = String(raw[r][cc]).trim();
          if (v) { supplierName = v; break; }
        }
        break;
      }
    }
    if (supplierName) break;
  }

  let headerRow = -1;
  let colCode = -1, colName = -1, colQty = -1, colPrice = -1;
  for (let r = 0; r < raw.length; r++) {
    const row = raw[r].map((c: any) => String(c).trim().toLowerCase());
    const codeIdx = row.findIndex((c: string) => c === "код");
    const nameIdx = row.findIndex((c: string) => c.startsWith("товари") || c === "назва");
    const qtyIdx = row.findIndex((c: string) => c === "кількість" || c === "к-сть");
    const priceIdx = row.findIndex((c: string) => c === "ціна" || c === "цiна");
    if (codeIdx >= 0 && nameIdx >= 0) {
      headerRow = r;
      colCode = codeIdx;
      colName = nameIdx;
      colQty = qtyIdx >= 0 ? qtyIdx : nameIdx + 1;
      colPrice = priceIdx >= 0 ? priceIdx : colQty + 2;
      break;
    }
  }
  if (headerRow < 0) throw new Error("Не знайдено рядок заголовку (Код / Товари) в файлі");

  const rows: InvoiceRow[] = [];
  for (let r = headerRow + 1; r < raw.length; r++) {
    const row = raw[r];
    const numCell = String(row[0]).trim();
    if (!numCell || isNaN(Number(numCell))) continue;
    const barcode = String(row[colCode] ?? "").trim();
    const name = String(row[colName] ?? "").trim();
    const qty = parseFloat(String(row[colQty] ?? "1").replace(",", ".")) || 1;
    const price = parseFloat(String(row[colPrice] ?? "0").replace(",", ".")) || 0;
    if (barcode && name) rows.push({ barcode, name, qty, price });
  }
  return { supplierName, rows };
}

async function importInvoice() {
  if (!invoiceFile.value) return;
  invoiceImporting.value = true;
  invoiceError.value = "";
  invoiceResult.value = null;
  try {
    const XLSX = await loadSheetJS();
    const buffer = await invoiceFile.value.arrayBuffer();
    const { supplierName, rows } = parseInvoiceSheet(XLSX, buffer);

    if (!rows.length) throw new Error("Файл не містить жодного товару");

    if (!invoiceSuppliers.value.length) {
      invoiceSuppliers.value = await api.suppliers(token.value);
    }

    let supplier = invoiceSuppliers.value.find(s => s.name.toLowerCase() === supplierName.toLowerCase());
    if (!supplier) {
      supplier = await api.createSupplier(token.value, { name: supplierName || "Невідомий постачальник" });
      invoiceSuppliers.value.push(supplier);
    }

    let createdCount = 0;
    let foundCount = 0;
    const resolvedItems: { productId: number; quantity: number; price: number }[] = [];

    for (const row of rows) {
      let product = products.value.find(p =>
        (row.barcode && p.barcode === row.barcode) ||
        (row.barcode && p.sku === row.barcode)
      );
      if (!product) {
        product = await api.createProduct(token.value, {
          name: row.name,
          barcode: row.barcode,
          sku: row.barcode,
          supplier: supplierName,
          purchasePrice: row.price,
          retailPrice: row.price,
          category: "",
          brand: "",
        });
        products.value.push(product);
        createdCount++;
      } else {
        foundCount++;
      }
      resolvedItems.push({ productId: product.id, quantity: row.qty, price: row.price });
    }

    const csvLines = ["productId,quantity,price",
      ...resolvedItems.map(i => `${i.productId},${i.quantity},${i.price}`)];
    const result = await api.importPurchaseCsv(token.value, {
      supplierId: supplier.id,
      currency: "UAH",
      csv: csvLines.join("\n"),
      note: `Імпорт накладної: ${invoiceFile.value.name}`,
    });

    invoiceResult.value = {
      created: createdCount,
      found: foundCount,
      supplierName: supplier.name,
      purchaseId: result.purchase?.id,
    };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    invoiceError.value = e.message;
  } finally {
    invoiceImporting.value = false;
  }
}
// ──────────────────────────────────────────────────────────────────────────
// Barcode camera in product form
const barcodeScanning = ref(false);
const barcodeScanError = ref("");
const barcodeVideoRef = ref<HTMLVideoElement | null>(null);
let barcodeCameraReader: any = null;

async function startBarcodeCamera() {
  barcodeScanError.value = "";
  barcodeScanning.value = true;
  await new Promise(r => setTimeout(r, 80));
  try {
    const { BrowserMultiFormatReader } = await import("@zxing/library");
    barcodeCameraReader = new BrowserMultiFormatReader();
    const devices = await barcodeCameraReader.listVideoInputDevices();
    const deviceId = devices.find((d: any) =>
      d.label.toLowerCase().includes("back") ||
      d.label.toLowerCase().includes("rear") ||
      d.label.toLowerCase().includes("environment")
    )?.deviceId ?? devices[0]?.deviceId;
    if (!deviceId) {
      barcodeScanError.value = "Камеру не знайдено. Скористайтесь USB-сканером або введіть вручну.";
      stopBarcodeCamera(); return;
    }
    await barcodeCameraReader.decodeFromVideoDevice(deviceId, barcodeVideoRef.value!, (result: any) => {
      if (result) {
        newProduct.value.barcode = result.getText();
        stopBarcodeCamera();
      }
    });
  } catch (e: any) {
    barcodeScanError.value = e?.message ?? "Помилка камери";
    stopBarcodeCamera();
  }
}

function stopBarcodeCamera() {
  barcodeCameraReader?.reset();
  barcodeCameraReader = null;
  barcodeScanning.value = false;
}

onBeforeUnmount(() => { stopBarcodeCamera(); });

const products = ref<Product[]>([]);
const warehouses = ref<Warehouse[]>([]);
const warehouseStocks = ref<WarehouseStock[]>([]);
const movements = ref<StockMovement[]>([]);
const loading = ref(false);
const error = ref("");
const subTab = ref<"list" | "movement" | "transfer" | "warehouses">((props.initialSubTab as any) ?? "list");

// filters
const search = ref("");
const filterCategory = ref("");
const filterArchived = ref(false);
const filterStockMin = ref<number | "">("");
const filterStockMax = ref<number | "">("");
const viewMode = ref<"table" | "grid">("table");

// product detail modal
const selectedProduct = ref<Product | null>(null);
const productDetailLifecycle = ref<ProductLifecycle | null>(null);
const productDetailLoading = ref(false);

async function openProductDetail(p: Product) {
  selectedProduct.value = p;
  productDetailLifecycle.value = null;
  productDetailLoading.value = true;
  try {
    productDetailLifecycle.value = await api.productLifecycle(token.value, p.id);
  } catch { /* ignore */ } finally {
    productDetailLoading.value = false;
  }
}

function closeProductDetail() {
  selectedProduct.value = null;
  productDetailLifecycle.value = null;
}

// pagination
const PAGE_SIZE = 40;
const currentPage = ref(1);

// forms
const showAddProduct = ref(false);
const showEditProduct = ref<Product | null>(null);
const showMovement = ref(false);
const showReturnToSupplier = ref(false);
const returnToSupplierProductId = ref(0);
const returnToSupplierQty = ref(1);
const showTransfer = ref(false);
const showAddWarehouse = ref(false);

// lifecycle history
import type { ProductLifecycleEvent, ProductLifecycle } from '../types';

const newProduct = ref<Partial<Product> & { initialWarehouseId?: number }>({ currency: "UAH", stock: 0, minStock: 1, initialWarehouseId: 0 });

function generateSku() {
  if (newProduct.value.sku) return; // не перезаписуємо якщо вже є
  const existingSkus = products.value
    .map((p: any) => p.sku)
    .filter((s: string) => /^SKU-\d+$/.test(s))
    .map((s: string) => parseInt(s.replace("SKU-", ""), 10));
  const maxNum = existingSkus.length > 0 ? Math.max(...existingSkus) : 0;
  newProduct.value.sku = `SKU-${String(maxNum + 1).padStart(4, "0")}`;
}
const movForm = ref({ productId: 0, warehouseId: 0, type: "receipt", quantity: 1, note: "" });
const transferForm = ref({ fromWarehouseId: 0, toWarehouseId: 0, productId: 0, quantity: 1, note: "" });
const newWarehouse = ref({ name: "", isVirtual: false, locationType: "warehouse" });

// Warehouse/shop detail view
const selectedWarehouseId = ref<number | null>(null);
const editingStock = ref<{ productId: number; qty: number } | null>(null);

const saving = ref(false);

async function load() {
  loading.value = true;
  error.value = "";
  try {
    [products.value, warehouses.value, warehouseStocks.value] = await Promise.all([
      api.products(token.value),
      api.warehouses(token.value),
      api.warehouseStocks(token.value),
    ]);
    currentPage.value = 1;
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}

// ── Movement tab state ───────────────────────────────────────────────────
const movSearch = ref("");
const movTypeFilter = ref("");

// Enriched movement row for display
interface EnrichedMovement {
  id: number;
  productId: number;
  productName: string;
  productSku: string;
  type: string;       // receipt | sale | transfer | writeoff | adjustment | purchase | ...
  quantity: number;
  note: string;
  fromWarehouseId?: number;
  toWarehouseId?: number;
  fromWarehouseName: string;
  toWarehouseName: string;
  createdAt: string;
}

const movSelectedEvent = ref<EnrichedMovement | null>(null);
const movEventsLoading = ref(false);

// Maps type → display label + icon + color
const movTypeConfig: Record<string, { label: string; icon: string; color: string }> = {
  receipt:    { label: "Надходження",         icon: "📦", color: "rgba(72,187,120,0.12);color:#6dd4a0" },
  purchase:   { label: "Закупівля",            icon: "🛒", color: "rgba(72,187,120,0.12);color:#6dd4a0" },
  sale:       { label: "Продаж",               icon: "💰", color: "rgba(224,112,112,0.12);color:#e07070" },
  transfer:   { label: "Переміщення",          icon: "🔄", color: "rgba(90,180,220,0.12);color:#60c0e0" },
  writeoff:   { label: "Списання",             icon: "🗑",  color: "rgba(224,160,96,0.12);color:#e0a060" },
  write_off:  { label: "Списання",             icon: "🗑",  color: "rgba(224,160,96,0.12);color:#e0a060" },
  adjustment: { label: "Коригування",          icon: "✏️", color: "rgba(90,180,220,0.12);color:#60c0e0" },
  return_from_customer: { label: "Повернення від клієнта",    icon: "↩️", color: "rgba(122,185,154,0.12);color:#7ab99a" },
  return_to_supplier:   { label: "Повернення постачальнику",  icon: "↪️", color: "rgba(224,160,96,0.12);color:#e0a060" },
};

function movCfg(type: string) {
  return movTypeConfig[type] ?? { label: type, icon: "🔄", color: "rgba(90,180,220,0.12);color:#60c0e0" };
}

// Enrich movements with product and warehouse names (all data already loaded)
const enrichedMovements = computed((): EnrichedMovement[] => {
  return movements.value.map(m => {
    const prod = products.value.find(p => p.id === m.productId);
    const wFrom = warehouses.value.find(w => w.id === m.fromWarehouseId);
    const wTo   = warehouses.value.find(w => w.id === m.toWarehouseId);
    return {
      ...m,
      productName:      prod?.name ?? `Товар #${m.productId}`,
      productSku:       prod?.sku  ?? "",
      fromWarehouseName: wFrom?.name ?? "",
      toWarehouseName:   wTo?.name   ?? "",
    };
  });
});

const filteredMovements = computed(() => {
  const s = movSearch.value.toLowerCase();
  const t = movTypeFilter.value;
  return enrichedMovements.value.filter(m => {
    if (t && m.type !== t) return false;
    if (s && !m.productName.toLowerCase().includes(s)
          && !m.productSku.toLowerCase().includes(s)
          && !m.note.toLowerCase().includes(s)
          && !m.fromWarehouseName.toLowerCase().includes(s)
          && !m.toWarehouseName.toLowerCase().includes(s)) return false;
    return true;
  });
});

async function loadMovements() {
  movEventsLoading.value = true;
  try {
    movements.value = await api.stockMovements(token.value);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    movEventsLoading.value = false;
  }
}

// keep for backward compatibility - no-op now
async function loadAllLifecycleEvents() {}

const categories = computed(() => [...new Set(products.value.map(p => p.category).filter(Boolean))]);

const filteredProducts = computed(() => {
  return products.value.filter(p => {
    if (!filterArchived.value && p.archived) return false;
    if (filterCategory.value && p.category !== filterCategory.value) return false;
    if (search.value) {
      const s = search.value.toLowerCase();
      if (!p.name.toLowerCase().includes(s) && !p.sku.toLowerCase().includes(s) && !(p.barcode || "").includes(s)) return false;
    }
    const stock = stockForProduct(p.id);
    if (filterStockMin.value !== "" && stock < Number(filterStockMin.value)) return false;
    if (filterStockMax.value !== "" && stock > Number(filterStockMax.value)) return false;
    return true;
  });
});

// Reset to page 1 whenever filters change
watch([search, filterCategory, filterArchived, filterStockMin, filterStockMax], () => {
  currentPage.value = 1;
});

const totalPages = computed(() => Math.max(1, Math.ceil(filteredProducts.value.length / PAGE_SIZE)));
const pagedProducts = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE;
  return filteredProducts.value.slice(start, start + PAGE_SIZE);
});
const pageNumbers = computed(() => {
  const pages = [];
  const total = totalPages.value;
  const cur = currentPage.value;
  for (let i = Math.max(1, cur - 2); i <= Math.min(total, cur + 2); i++) pages.push(i);
  return pages;
});

function stockForProduct(productId: number) {
  return warehouseStocks.value.filter(s => s.productId === productId).reduce((a, s) => a + s.quantity, 0);
}

function stockByWarehouse(productId: number) {
  return warehouses.value.map(w => {
    const s = warehouseStocks.value.find(s => s.productId === productId && s.warehouseId === w.id);
    return { warehouse: w.name, qty: s?.quantity ?? 0 };
  }).filter(x => x.qty > 0);
}

async function saveProduct() {
  saving.value = true; error.value = "";
  try {
    if (showEditProduct.value) {
      await api.updateProduct(token.value, showEditProduct.value.id, newProduct.value);
    } else {
      await api.createProduct(token.value, newProduct.value);
    }
    showAddProduct.value = false;
    showEditProduct.value = null;
    newProduct.value = { currency: "UAH", stock: 0, minStock: 1, initialWarehouseId: 0 };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

function startEdit(p: Product) {
  showEditProduct.value = p;
  newProduct.value = { ...p };
  supplierSearch.value = p.supplier || '';
  showAddProduct.value = true;
  if (!productFormSuppliers.value.length) {
    api.suppliers(token.value).then(r => { productFormSuppliers.value = r; });
  }
}

async function toggleArchive(p: Product) {
  try {
    await api.archiveProduct(token.value, p.id, !p.archived);
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

async function genBarcode(p: Product) {
  try {
    await api.generateProductBarcode(token.value, p.id);
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

async function doMovement() {
  saving.value = true; error.value = "";
  try {
    await api.createMovement(token.value, movForm.value);
    showMovement.value = false;
    movForm.value = { productId: 0, warehouseId: 0, type: "receipt", quantity: 1, note: "" };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function doReturnToSupplier() {
  saving.value = true; error.value = "";
  try {
    await api.createMovement(token.value, {
      productId: returnToSupplierProductId.value,
      warehouseId: 0,
      type: "return_to_supplier",
      quantity: returnToSupplierQty.value,
      note: ""
    });
    showReturnToSupplier.value = false;
    returnToSupplierQty.value = 1;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function doTransfer() {
  saving.value = true; error.value = "";
  try {
    await api.createTransfer(token.value, {
      fromWarehouseId: transferForm.value.fromWarehouseId,
      toWarehouseId: transferForm.value.toWarehouseId,
      items: [{ productId: transferForm.value.productId, quantity: transferForm.value.quantity }],
      note: transferForm.value.note
    });
    showTransfer.value = false;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function addWarehouse() {
  saving.value = true; error.value = "";
  try {
    await api.createWarehouse(token.value, newWarehouse.value);
    newWarehouse.value = { name: "", isVirtual: false, locationType: "warehouse" };
    showAddWarehouse.value = false;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function handleImportXlsx(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0];
  if (!file) return;
  try {
    const result = await api.importProductsXlsx(token.value, { file, updateExisting: true });
    alert(`Імпортовано: ${result.imported}, оновлено: ${result.updated}, пропущено: ${result.skipped}`);
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  }
}

const showQrProduct = ref<Product | null>(null);
const movPage = ref(1);
const selectedForLabel = ref<number[]>([]);

// ── Bulk transfer for selected products ──────────────────────────────────
const showBulkTransfer = ref(false);
const bulkTransferForm = ref({ fromWarehouseId: 0, toWarehouseId: 0, note: "" });
const bulkTransferSaving = ref(false);
const bulkTransferError = ref("");

async function doBulkTransfer() {
  if (!bulkTransferForm.value.fromWarehouseId || !bulkTransferForm.value.toWarehouseId) return;
  if (bulkTransferForm.value.fromWarehouseId === bulkTransferForm.value.toWarehouseId) {
    bulkTransferError.value = "Склад відправки і склад призначення мають бути різними";
    return;
  }
  bulkTransferSaving.value = true;
  bulkTransferError.value = "";
  try {
    const items = selectedForLabel.value.map(productId => {
      const stock = warehouseStocks.value.find(
        s => s.productId === productId && s.warehouseId === bulkTransferForm.value.fromWarehouseId
      );
      return { productId, quantity: stock?.quantity ?? 0 };
    }).filter(i => i.quantity > 0);

    if (!items.length) {
      bulkTransferError.value = "Жоден з вибраних товарів не має залишку на обраному складі";
      bulkTransferSaving.value = false;
      return;
    }

    await api.createTransfer(token.value, {
      fromWarehouseId: bulkTransferForm.value.fromWarehouseId,
      toWarehouseId: bulkTransferForm.value.toWarehouseId,
      items,
      note: bulkTransferForm.value.note,
    });
    showBulkTransfer.value = false;
    selectedForLabel.value = [];
    bulkTransferForm.value = { fromWarehouseId: 0, toWarehouseId: 0, note: "" };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    bulkTransferError.value = e.message;
  } finally {
    bulkTransferSaving.value = false;
  }
}
// ─────────────────────────────────────────────────────────────────────────

function toggleLabelSelect(id: number) {
  const idx = selectedForLabel.value.indexOf(id);
  if (idx >= 0) selectedForLabel.value.splice(idx, 1);
  else selectedForLabel.value.push(id);
}

async function printLabels(format: "small" | "large" = "small", overrideIds?: number[]) {
  const ids = overrideIds ?? (selectedForLabel.value.length > 0
    ? [...selectedForLabel.value]
    : filteredProducts.value.map(p => p.id));
  if (!ids.length) return;
  try {
    const blob = await api.labelsPdf(token.value, ids, format);
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank");
    URL.revokeObjectURL(url);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  }
}

const showBulkPrice = ref(false);
const bulkPriceForm = ref<{ markupPct: number; markupFixed: number; roundMode: "none"|"nearest"|"up"|"down"; roundTo: number; applyToCategory: string }>({
  markupPct: 0, markupFixed: 0, roundMode: "none", roundTo: 1, applyToCategory: ""
});

async function doBulkPrice() {
  saving.value = true; error.value = "";
  try {
    const payload: any = {
      roundMode: bulkPriceForm.value.roundMode,
      roundTo: bulkPriceForm.value.roundTo,
    };
    if (bulkPriceForm.value.markupPct) payload.markupPercent = bulkPriceForm.value.markupPct;
    if (bulkPriceForm.value.markupFixed) payload.markupFixed = bulkPriceForm.value.markupFixed;
    if (bulkPriceForm.value.applyToCategory) payload.category = bulkPriceForm.value.applyToCategory;
    const result = await api.bulkUpdateProductPrices(token.value, payload);
    showBulkPrice.value = false;
    alert(`Оновлено ${result.updatedCount} товарів`);
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function exportXlsx() {
  try {
    // If specific products are selected — export only them
    if (selectedForLabel.value.length > 0) {
      const XLSX = await loadSheetJS();
      const selected = products.value.filter(p => selectedForLabel.value.includes(p.id));
      const rows = selected.map(p => ({
        "Назва": p.name,
        "SKU": p.sku,
        "Штрихкод": p.barcode || "",
        "Категорія": p.category || "",
        "Бренд": p.brand || "",
        "Ціна закупівлі": p.purchasePrice,
        "Ціна роздрібна": p.retailPrice,
        "Валюта": p.currency || "UAH",
        "Залишок": stockForProduct(p.id),
        "Мін. залишок": p.minStock,
        "Постачальник": p.supplier || "",
        "SKU постачальника": p.supplierSku || "",
      }));
      const ws = XLSX.utils.json_to_sheet(rows);
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, "Товари");
      XLSX.writeFile(wb, `products_selected_${selected.length}.xlsx`);
      return;
    }
    // Otherwise export all (via API)
    const blob = await api.exportProductsXlsx(token.value, filterArchived.value);
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = "products.xlsx"; a.click();
    URL.revokeObjectURL(url);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  }
}

// Warehouse/shop detail helpers
const selectedWarehouse = computed(() =>
  warehouses.value.find(w => w.id === selectedWarehouseId.value) ?? null
);

const warehouseProducts = computed(() => {
  if (!selectedWarehouseId.value) return [];
  return products.value.filter(p => {
    const stock = warehouseStocks.value.find(s => s.productId === p.id && s.warehouseId === selectedWarehouseId.value);
    return stock && stock.quantity > 0;
  }).map(p => {
    const stock = warehouseStocks.value.find(s => s.productId === p.id && s.warehouseId === selectedWarehouseId.value);
    return { ...p, warehouseQty: stock?.quantity ?? 0 };
  });
});

const warehouseProductSearch = ref("");
const warehouseFilterCategory = ref("");
const warehouseFilterBrand = ref("");
const warehouseFilterQtyMin = ref<number | "">("");
const warehouseFilterQtyMax = ref<number | "">("");

const warehouseCategories = computed(() =>
  [...new Set(warehouseProducts.value.map(p => p.category).filter(Boolean))]
);
const warehouseBrands = computed(() =>
  [...new Set(warehouseProducts.value.map(p => p.brand).filter(Boolean))]
);

const filteredWarehouseProducts = computed(() => {
  const s = warehouseProductSearch.value.toLowerCase();
  return warehouseProducts.value.filter(p => {
    if (s && !p.name.toLowerCase().includes(s) && !p.sku.toLowerCase().includes(s)) return false;
    if (warehouseFilterCategory.value && p.category !== warehouseFilterCategory.value) return false;
    if (warehouseFilterBrand.value && (p.brand ?? "") !== warehouseFilterBrand.value) return false;
    if (warehouseFilterQtyMin.value !== "" && p.warehouseQty < Number(warehouseFilterQtyMin.value)) return false;
    if (warehouseFilterQtyMax.value !== "" && p.warehouseQty > Number(warehouseFilterQtyMax.value)) return false;
    return true;
  });
});

const editingWarehouseProduct = ref<(typeof warehouseProducts.value[0]) | null>(null);
const editWarehouseProductForm = ref<Partial<typeof products.value[0]>>({});

function startEditWarehouseProduct(p: typeof warehouseProducts.value[0]) {
  editingWarehouseProduct.value = p;
  editWarehouseProductForm.value = { ...p };
}

async function saveWarehouseProduct() {
  if (!editingWarehouseProduct.value) return;
  saving.value = true; error.value = "";
  try {
    await api.updateProduct(token.value, editingWarehouseProduct.value.id, editWarehouseProductForm.value);
    editingWarehouseProduct.value = null;
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

onMounted(async () => {
  await load();
  loadMovements();
  if (props.initialProductId) {
    const p = products.value.find(x => x.id === props.initialProductId);
    if (p) openProductDetail(p);
  }
});
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Товари та склад</h2>
      <div style="display:flex;gap:0.5rem">
        <button v-if="can('products:write')" class="ghost-button" @click="showAddProduct=true;showEditProduct=null;newProduct={currency:'UAH',stock:0,minStock:1};supplierSearch='';if(!productFormSuppliers.length)api.suppliers(token).then(r=>productFormSuppliers.value=r)">+ Товар</button>
        <button class="ghost-button" @click="showMovement=true">Рух товару</button>
        <button class="ghost-button"
          :style="selectedForLabel.length > 0 ? 'border-color:rgba(90,180,220,0.6);color:#60c0e0' : ''"
          :title="selectedForLabel.length > 0 ? `Перемістити ${selectedForLabel.length} вибраних товарів` : 'Переміщення між складами'"
          @click="selectedForLabel.length > 0 ? (showBulkTransfer=true) : (showTransfer=true)">
          {{ selectedForLabel.length > 0 ? `🔄 Перемістити (${selectedForLabel.length})` : 'Переміщення' }}
        </button>
        <button class="ghost-button" @click="showInvoiceImport=true">📥 Імпорт накладної</button>
        <button class="ghost-button"
          :style="selectedForLabel.length > 0 ? 'border-color:rgba(72,187,120,0.6);color:#6dd4a0' : ''"
          :title="selectedForLabel.length > 0 ? `Експорт ${selectedForLabel.length} вибраних товарів у XLSX` : 'Експорт всіх товарів у XLSX'"
          @click="exportXlsx">
          {{ selectedForLabel.length > 0 ? `XLSX (${selectedForLabel.length})` : 'XLSX' }}
        </button>
        
        <button v-if="can('products:write')" class="ghost-button" @click="showBulkPrice=true"> Ціни масово</button>
        <button class="ghost-button" @click="printLabels('small')" title="Друк цінників (малий формат)"> Цінники малі</button>
        <button class="ghost-button" @click="printLabels('large')" title="Друк цінників (великий формат)"> Цінники великі</button>
      </div>
    </div>
    <div v-if="selectedForLabel.length > 0" style="margin-bottom:0.5rem;font-size:0.85rem;color:#9fe8c4;display:flex;align-items:center;gap:0.5rem;flex-wrap:wrap;background:rgba(72,187,120,0.07);border-radius:8px;padding:0.45rem 0.75rem;border:1px solid rgba(72,187,120,0.2)">
      <span>✓ Обрано <strong>{{ selectedForLabel.length }}</strong> товарів</span>
      <span class="subtle" style="font-size:0.8rem">— використайте кнопки «Перемістити» або «XLSX» для дії над вибраними</span>
      <button class="ghost-button" style="padding:0.2rem 0.5rem;font-size:0.8rem;margin-left:auto" @click="selectedForLabel = []">✕ Зняти вибір</button>
    </div>

    <div class="tab-row" style="margin-bottom:1rem">
      <button :class="['tab-button', subTab==='list'&&'tab-button--active']" @click="subTab='list'">Список</button>
      <button :class="['tab-button', subTab==='movement'&&'tab-button--active']" @click="subTab='movement';loadMovements();loadAllLifecycleEvents()">Рухи</button>
      <button :class="['tab-button', subTab==='warehouses'&&'tab-button--active']" @click="subTab='warehouses'">Склади та Магазини</button>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <!-- PRODUCT LIST -->
    <div v-if="subTab==='list'">
      <div style="display:flex;gap:0.5rem;margin-bottom:1rem;flex-wrap:wrap">
        <div style="flex:2;min-width:180px">
          <BarcodeScanner
            v-model="search"
            placeholder="Пошук (назва, SKU, штрихкод)..."
            @scanned="search = $event"
          />
        </div>
        <select v-model="filterCategory" style="min-width:130px">
          <option value="">Всі категорії</option>
          <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
        </select>
        <div style="display:flex;align-items:center;gap:0.3rem;min-width:200px">
          <span style="font-size:0.82rem;color:var(--text-subtle);white-space:nowrap">Кількість:</span>
          <input
            v-model.number="filterStockMin"
            type="number" min="0" placeholder="від"
            style="width:64px;padding:0.35rem 0.5rem;font-size:0.85rem"
            class="input"
          />
          <span style="color:var(--text-subtle)">—</span>
          <input
            v-model.number="filterStockMax"
            type="number" min="0" placeholder="до"
            style="width:64px;padding:0.35rem 0.5rem;font-size:0.85rem"
            class="input"
          />
          <button v-if="filterStockMin !== '' || filterStockMax !== ''"
            class="ghost-button" style="padding:0.3rem 0.5rem;font-size:0.8rem"
            @click="filterStockMin=''; filterStockMax=''">✕</button>
        </div>
        <label style="display:flex;align-items:center;gap:0.4rem;font-size:0.9rem">
          <input type="checkbox" v-model="filterArchived"> Архів
        </label>
        <button class="ghost-button" style="padding:0.5rem 0.75rem"
          @click="viewMode = viewMode==='table' ? 'grid' : 'table'"
          :title="viewMode==='table' ? 'Вигляд сіткою' : 'Вигляд таблицею'">
          {{ viewMode === 'table' ? '⊞' : '☰' }}
        </button>
      </div>
      <!-- GRID VIEW (tablet / toggle) -->
      <div v-if="viewMode==='grid'" class="products-grid" style="display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:0.75rem">
        <div v-for="p in pagedProducts" :key="p.id"
          class="panel" style="padding:0.9rem;margin-top:0;cursor:pointer;position:relative"
          :style="p.archived ? 'opacity:0.5' : ''"
          @click="openProductDetail(p)">
          <div style="position:absolute;top:0.5rem;right:0.5rem">
            <input type="checkbox" :checked="selectedForLabel.includes(p.id)"
              @click.stop="toggleLabelSelect(p.id)">
          </div>
          <div style="font-weight:600;font-size:0.88rem;margin-bottom:0.25rem;padding-right:1.2rem">{{ p.name }}</div>
          <div class="subtle" style="font-size:0.75rem;margin-bottom:0.4rem">{{ p.sku }}</div>
          <div style="font-size:0.8rem;color:var(--text-muted)">{{ p.category }}</div>
          <div style="margin:0.5rem 0;font-size:1.05rem;font-weight:700;color:var(--accent)">
            {{ p.retailPrice }} {{ p.currency || 'UAH' }}
          </div>
          <div v-if="can('products:write')" class="subtle" style="font-size:0.75rem">
            Закупівля: {{ p.purchasePrice }}
          </div>
          <div style="margin-top:0.4rem;font-size:0.82rem"
            :style="stockForProduct(p.id) <= (p.minStock||0) ? 'color:var(--danger);font-weight:700' : 'color:var(--ok)'">
            Залишок: {{ stockForProduct(p.id) }}
          </div>
          <div style="display:flex;gap:0.3rem;margin-top:0.6rem;flex-wrap:wrap">
            <button class="ghost-button" style="padding:0.25rem 0.5rem;font-size:0.75rem" @click.stop="showQrProduct=p"> QR</button>
            <button v-if="can('products:write')" class="ghost-button" style="padding:0.25rem 0.5rem;font-size:0.75rem" @click.stop="toggleArchive(p)">
              {{ p.archived ? '↩' : '🗄' }}
            </button>
          </div>
        </div>
      </div>

      <!-- TABLE VIEW -->
      <div v-else class="products-table-wrap table-wrap">
        <table>
          <thead>
            <tr>
              <th style="width:32px"></th>
              <th>Назва</th><th>SKU</th><th>Категорія</th>
              <th>Закупівля</th><th>Роздріб</th>
              <th>Залишок</th><th>Мін</th><th>SKU пост.</th><th>Дії</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in pagedProducts" :key="p.id" :class="p.archived?'row-muted':''" style="cursor:pointer" @click="openProductDetail(p)">
              <td>
                <input type="checkbox"
                  :checked="selectedForLabel.includes(p.id)"
                  @change="toggleLabelSelect(p.id)">
              </td>
              <td>
                <div>{{ p.name }}</div>
                <div v-if="p.barcode" style="font-family:monospace;font-size:0.78rem;color:#7ab99a">{{ p.barcode }}</div>
                <div v-else style="font-size:0.75rem;color:#e0a060;font-style:italic">Немає штрихкоду</div>
              </td>
              <td>{{ p.sku }}</td>
              <td>{{ p.category }}</td>
              <td v-if="can('products:write')">{{ p.purchasePrice }} {{ p.currency || 'UAH' }}</td>
              <td v-else>—</td>
              <td>{{ p.retailPrice }} {{ p.currency || 'UAH' }}</td>
              <td :style="stockForProduct(p.id) <= (p.minStock||0) ? 'color:var(--danger);font-weight:700' : ''">
                {{ stockForProduct(p.id) }}
                <span v-if="stockForProduct(p.id) > 0" class="subtle" style="font-size:0.75rem">
                  ({{ stockByWarehouse(p.id).map(x=>x.warehouse+':'+x.qty).join(', ') }})
                </span>
              </td>
              <td>{{ p.minStock }}</td>
              <td>
                <div style="font-size:0.82rem">{{ p.supplierSku || '—' }}</div>
                <div class="subtle" style="font-size:0.75rem">{{ p.supplierNameExt }}</div>
              </td>
              <td @click.stop>
                <div style="display:flex;gap:0.3rem;flex-wrap:wrap">
                  <button v-if="can('products:write')" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem" @click="startEdit(p)">Ред.</button>
                  <button v-if="can('products:write')" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem;color:#6dd4a0;border-color:rgba(72,187,120,0.4)"
                    @click="movForm.productId=p.id; movForm.type='receipt'; movForm.quantity=1; movForm.note=''; showMovement=true"
                    title="Змінити кількість на складі">
                    Кількість
                  </button>
                  <button v-if="can('products:write') && !p.barcode" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem;color:#e0a060;border-color:rgba(224,160,96,0.4)" @click="genBarcode(p)" title="Згенерувати штрихкод автоматично">+ Штрихкод</button>
                  <button v-if="p.barcode || p.sku" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem" @click="showQrProduct=p" title="Показати та роздрукувати штрихкод">Друк</button>
                  <button v-if="can('products:write')" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.8rem" @click="toggleArchive(p)">{{ p.archived ? 'Відновити' : 'Архів' }}</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- PAGINATION -->
      <div class="pagination" v-if="totalPages > 1">
        <button class="pagination__btn" :disabled="currentPage===1" @click="currentPage=1">«</button>
        <button class="pagination__btn" :disabled="currentPage===1" @click="currentPage--">‹</button>
        <button v-for="n in pageNumbers" :key="n"
          :class="['pagination__btn', n===currentPage && 'pagination__btn--active']"
          @click="currentPage=n">{{ n }}</button>
        <button class="pagination__btn" :disabled="currentPage===totalPages" @click="currentPage++">›</button>
        <button class="pagination__btn" :disabled="currentPage===totalPages" @click="currentPage=totalPages">»</button>
        <span class="subtle" style="font-size:0.82rem;margin-left:0.3rem">
          {{ (currentPage-1)*PAGE_SIZE+1 }}–{{ Math.min(currentPage*PAGE_SIZE, filteredProducts.length) }} з {{ filteredProducts.length }}
        </span>
      </div>
    </div>

    <!-- MOVEMENTS -->
    <div v-if="subTab==='movement'">
      <!-- Filter bar -->
      <div style="display:flex;gap:0.5rem;margin-bottom:1rem;flex-wrap:wrap;align-items:center">
        <input v-model="movSearch" placeholder="🔍 Назва товару, SKU, склад, примітка..." style="flex:2;min-width:180px">
        <select v-model="movTypeFilter" style="min-width:170px">
          <option value="">Всі типи рухів</option>
          <option value="receipt">📦 Надходження</option>
          <option value="purchase">🛒 Закупівля</option>
          <option value="sale">💰 Продаж</option>
          <option value="transfer">🔄 Переміщення</option>
          <option value="writeoff">🗑 Списання</option>
          <option value="write_off">🗑 Списання</option>
          <option value="adjustment">✏️ Коригування</option>
          <option value="return_from_customer">↩️ Повернення від клієнта</option>
          <option value="return_to_supplier">↪️ Повернення постачальнику</option>
        </select>
        <button v-if="movSearch || movTypeFilter" class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.82rem"
          @click="movSearch='';movTypeFilter=''">✕ Скинути</button>
        <span class="subtle" style="font-size:0.82rem;margin-left:auto">{{ filteredMovements.length }} записів</span>
      </div>

      <div v-if="movEventsLoading" style="text-align:center;padding:3rem;color:#7ab99a">
        <div style="font-size:1.5rem;margin-bottom:0.5rem">⏳</div>
        Завантаження...
      </div>
      <div v-else-if="!movements.length" style="text-align:center;padding:3rem;color:#888">
        <div style="font-size:2rem;margin-bottom:0.5rem">📋</div>
        Рухів товарів не знайдено
      </div>
      <div v-else-if="!filteredMovements.length" style="text-align:center;padding:2rem;color:#888">
        Нічого не знайдено за вашим запитом
      </div>

      <div v-else class="table-wrap">
        <table style="font-size:0.87rem">
          <thead>
            <tr>
              <th style="width:36px"></th>
              <th>Товар</th>
              <th>Тип руху</th>
              <th>К-сть</th>
              <th>Склад (з)</th>
              <th>Склад (до)</th>
              <th>Примітка</th>
              <th>Дата</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in filteredMovements.slice((movPage-1)*30, movPage*30)" :key="m.id"
              style="cursor:pointer"
              @click="movSelectedEvent=m"
              @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(122,185,154,0.07)'"
              @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
              <td style="text-align:center;font-size:1.1rem">{{ movCfg(m.type).icon }}</td>
              <td>
                <div style="font-weight:600">{{ m.productName }}</div>
                <div style="font-size:0.75rem;color:var(--text-subtle);font-family:monospace">{{ m.productSku }}</div>
              </td>
              <td>
                <span style="font-size:0.8rem;padding:0.2rem 0.55rem;border-radius:12px"
                  :style="movCfg(m.type).color">
                  {{ movCfg(m.type).label }}
                </span>
              </td>
              <td style="font-weight:600"
                :style="m.type==='sale'||m.type==='writeoff'||m.type==='write_off'||m.type==='return_to_supplier' ? 'color:#e07070' : 'color:#6dd4a0'">
                {{ m.type==='sale'||m.type==='writeoff'||m.type==='write_off'||m.type==='return_to_supplier' ? '−' : '+' }}{{ m.quantity }}
              </td>
              <td style="font-size:0.84rem;color:var(--text-subtle)">{{ m.fromWarehouseName || '—' }}</td>
              <td style="font-size:0.84rem;color:var(--text-subtle)">{{ m.toWarehouseName || '—' }}</td>
              <td style="font-size:0.84rem;color:var(--text-subtle);max-width:160px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap"
                :title="m.note">{{ m.note || '—' }}</td>
              <td style="font-size:0.78rem;color:var(--text-subtle);white-space:nowrap">
                {{ new Date(m.createdAt).toLocaleDateString('uk', {day:'2-digit',month:'short',year:'numeric'}) }}
                <div style="color:#666">{{ new Date(m.createdAt).toLocaleTimeString('uk', {hour:'2-digit',minute:'2-digit'}) }}</div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination" v-if="filteredMovements.length > 30">
        <button class="pagination__btn" :disabled="movPage===1" @click="movPage=1">«</button>
        <button class="pagination__btn" :disabled="movPage===1" @click="movPage--">‹</button>
        <span class="subtle" style="font-size:0.82rem">{{ movPage }} / {{ Math.ceil(filteredMovements.length/30) }}</span>
        <button class="pagination__btn" :disabled="movPage>=Math.ceil(filteredMovements.length/30)" @click="movPage++">›</button>
        <button class="pagination__btn" :disabled="movPage>=Math.ceil(filteredMovements.length/30)" @click="movPage=Math.ceil(filteredMovements.length/30)">»</button>
      </div>

      <!-- MODAL: movement detail -->
      <div v-if="movSelectedEvent" class="modal-backdrop" @click.self="movSelectedEvent=null">
        <div class="panel modal-box" style="max-width:500px;width:100%">
          <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:1.2rem">
            <div>
              <div style="font-size:1.5rem;margin-bottom:0.3rem">{{ movCfg(movSelectedEvent.type).icon }}</div>
              <h3 style="margin:0 0 0.25rem">{{ movCfg(movSelectedEvent.type).label }}</h3>
              <div style="font-size:0.82rem;color:var(--text-subtle)">
                {{ new Date(movSelectedEvent.createdAt).toLocaleString('uk', {day:'2-digit',month:'long',year:'numeric',hour:'2-digit',minute:'2-digit'}) }}
              </div>
            </div>
            <button class="ghost-button" style="padding:0.3rem 0.6rem" @click="movSelectedEvent=null">✕</button>
          </div>

          <div style="background:rgba(122,185,154,0.06);border-radius:8px;padding:0.75rem 1rem;margin-bottom:1rem">
            <div style="font-size:0.67rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.06em;margin-bottom:0.3rem">ТОВАР</div>
            <div style="font-weight:700;font-size:1rem">{{ movSelectedEvent.productName }}</div>
            <div style="font-family:monospace;font-size:0.8rem;color:#7ab99a">{{ movSelectedEvent.productSku }}</div>
          </div>

          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.75rem;font-size:0.88rem;margin-bottom:1rem">
            <div>
              <div style="font-size:0.67rem;color:var(--text-subtle);font-weight:600;margin-bottom:0.2rem">КІЛЬКІСТЬ</div>
              <div style="font-weight:700;font-size:1.1rem"
                :style="movSelectedEvent.type==='sale'||movSelectedEvent.type==='writeoff'||movSelectedEvent.type==='write_off'||movSelectedEvent.type==='return_to_supplier' ? 'color:#e07070' : 'color:#6dd4a0'">
                {{ movSelectedEvent.type==='sale'||movSelectedEvent.type==='writeoff'||movSelectedEvent.type==='write_off'||movSelectedEvent.type==='return_to_supplier' ? '−' : '+' }}{{ movSelectedEvent.quantity }} шт.
              </div>
            </div>
            <div>
              <div style="font-size:0.67rem;color:var(--text-subtle);font-weight:600;margin-bottom:0.2rem">ID РУХУ</div>
              <div style="font-weight:500">#{{ movSelectedEvent.id }}</div>
            </div>
            <div v-if="movSelectedEvent.fromWarehouseName">
              <div style="font-size:0.67rem;color:var(--text-subtle);font-weight:600;margin-bottom:0.2rem">СКЛАД (З)</div>
              <div style="font-weight:500">{{ movSelectedEvent.fromWarehouseName }}</div>
            </div>
            <div v-if="movSelectedEvent.toWarehouseName">
              <div style="font-size:0.67rem;color:var(--text-subtle);font-weight:600;margin-bottom:0.2rem">СКЛАД (ДО)</div>
              <div style="font-weight:500">{{ movSelectedEvent.toWarehouseName }}</div>
            </div>
          </div>

          <div v-if="movSelectedEvent.note"
            style="font-size:0.85rem;color:var(--text-subtle);font-style:italic;padding:0.55rem 0.75rem;background:rgba(122,185,154,0.05);border-radius:6px;margin-bottom:1rem">
            📝 {{ movSelectedEvent.note }}
          </div>

          <div style="display:flex;justify-content:flex-end">
            <button class="ghost-button" @click="movSelectedEvent=null">Закрити</button>
          </div>
        </div>
      </div>
    </div>

    <!-- WAREHOUSES & SHOPS -->
    <div v-if="subTab==='warehouses'">
      <div style="display:flex;gap:1rem;align-items:center;margin-bottom:1rem;flex-wrap:wrap">
        <button v-if="can('products:write')" class="ghost-button" @click="showAddWarehouse=true;selectedWarehouseId=null">+ Склад / Магазин</button>
        <div v-if="selectedWarehouseId" style="margin-left:auto">
          <button class="ghost-button" @click="selectedWarehouseId=null">← Назад до списку</button>
        </div>
      </div>

      <!-- Warehouse/shop list -->
      <div v-if="!selectedWarehouseId">
        <!-- Shops -->
        <div v-if="warehouses.filter(w=>w.locationType==='shop').length" style="margin-bottom:1.5rem">
          <div style="font-size:0.8rem;font-weight:600;color:#7ab99a;text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.5rem">🏪 Магазини</div>
          <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:0.8rem">
            <div v-for="w in warehouses.filter(x=>x.locationType==='shop')" :key="w.id"
              class="panel" style="padding:1rem;cursor:pointer;transition:border-color 0.2s"
              @click="selectedWarehouseId=w.id">
              <div style="font-size:1.1rem;font-weight:600;margin-bottom:0.3rem">🏪 {{ w.name }}</div>
              <div class="subtle" style="font-size:0.8rem">
                {{ warehouseStocks.filter(s=>s.warehouseId===w.id && s.quantity>0).length }} позицій
              </div>
              <div class="subtle" style="font-size:0.75rem">{{ w.isVirtual ? 'Віртуальний' : 'Фізичний' }}</div>
            </div>
          </div>
        </div>
        <!-- Warehouses -->
        <div>
          <div style="font-size:0.8rem;font-weight:600;color:#7ab99a;text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.5rem">🏭 Склади</div>
          <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:0.8rem">
            <div v-for="w in warehouses.filter(x=>x.locationType!=='shop')" :key="w.id"
              class="panel" style="padding:1rem;cursor:pointer;transition:border-color 0.2s"
              @click="selectedWarehouseId=w.id">
              <div style="font-size:1.1rem;font-weight:600;margin-bottom:0.3rem">🏭 {{ w.name }}</div>
              <div class="subtle" style="font-size:0.8rem">
                {{ warehouseStocks.filter(s=>s.warehouseId===w.id && s.quantity>0).length }} позицій
              </div>
              <div class="subtle" style="font-size:0.75rem">{{ w.isVirtual ? 'Віртуальний' : 'Фізичний' }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Warehouse detail: products + edit -->
      <div v-else-if="selectedWarehouse">
        <div style="display:flex;align-items:center;gap:0.8rem;margin-bottom:1rem;flex-wrap:wrap">
          <h3 style="margin:0">
            {{ selectedWarehouse.locationType === 'shop' ? '🏪' : '🏭' }} {{ selectedWarehouse.name }}
          </h3>
          <span class="chip">{{ filteredWarehouseProducts.length }} позицій</span>
        </div>

        <!-- Warehouse filter bar -->
        <div style="display:flex;gap:0.5rem;margin-bottom:1rem;flex-wrap:wrap;align-items:center">
          <input v-model="warehouseProductSearch" placeholder="🔍 Пошук за назвою, SKU..." style="flex:2;min-width:160px">
          <select v-model="warehouseFilterCategory" style="min-width:130px">
            <option value="">Всі категорії</option>
            <option v-for="c in warehouseCategories" :key="c" :value="c">{{ c }}</option>
          </select>
          <select v-model="warehouseFilterBrand" style="min-width:120px">
            <option value="">Всі бренди</option>
            <option v-for="b in warehouseBrands" :key="b" :value="b">{{ b }}</option>
          </select>
          <div style="display:flex;align-items:center;gap:0.3rem">
            <span style="font-size:0.82rem;color:var(--text-subtle);white-space:nowrap">К-сть:</span>
            <input v-model.number="warehouseFilterQtyMin" type="number" min="0" placeholder="від"
              style="width:60px;padding:0.35rem 0.5rem;font-size:0.85rem">
            <span style="color:var(--text-subtle)">—</span>
            <input v-model.number="warehouseFilterQtyMax" type="number" min="0" placeholder="до"
              style="width:60px;padding:0.35rem 0.5rem;font-size:0.85rem">
          </div>
          <button v-if="warehouseProductSearch || warehouseFilterCategory || warehouseFilterBrand || warehouseFilterQtyMin !== '' || warehouseFilterQtyMax !== ''"
            class="ghost-button" style="padding:0.3rem 0.6rem;font-size:0.82rem"
            @click="warehouseProductSearch=''; warehouseFilterCategory=''; warehouseFilterBrand=''; warehouseFilterQtyMin=''; warehouseFilterQtyMax=''"
          >✕ Скинути</button>
        </div>

        <!-- Product edit modal inline -->
        <div v-if="editingWarehouseProduct" class="panel" style="padding:1rem;margin-bottom:1rem;border:1px solid rgba(122,185,154,0.4)">
          <h4 style="margin-top:0"> Редагувати: {{ editingWarehouseProduct.name }}</h4>
          <div class="grid" style="grid-template-columns:1fr 1fr">
            <label>Назва <input v-model="editWarehouseProductForm.name"></label>
            <label>SKU <input v-model="editWarehouseProductForm.sku"></label>
            <label>Категорія <input v-model="editWarehouseProductForm.category"></label>
            <label>Бренд <input v-model="editWarehouseProductForm.brand"></label>
            <label>Ціна закупівлі <input type="number" v-model.number="editWarehouseProductForm.purchasePrice"></label>
            <label>Ціна роздрібна <input type="number" v-model.number="editWarehouseProductForm.retailPrice"></label>
            <label>Мін. залишок <input type="number" v-model.number="editWarehouseProductForm.minStock"></label>
            <label>Позиція на полиці <input v-model="editWarehouseProductForm.warehousePosition"></label>
            <label style="grid-column:1/-1">Коментар <textarea v-model="editWarehouseProductForm.comments" rows="2"></textarea></label>
          </div>
          <div style="display:flex;gap:0.5rem;margin-top:0.8rem">
            <button @click="saveWarehouseProduct" :disabled="saving">{{ saving ? 'Збереження...' : 'Зберегти' }}</button>
            <button class="ghost-button" @click="editingWarehouseProduct=null">Скасувати</button>
          </div>
          <p v-if="error" class="error-text">{{ error }}</p>
        </div>

        <table v-if="filteredWarehouseProducts.length" style="font-size:0.87rem">
          <thead>
            <tr>
              <th>Товар</th>
              <th>SKU</th>
              <th>Категорія</th>
              <th>Бренд</th>
              <th>К-сть</th>
              <th>Роздрібна ціна</th>
              <th>Дія</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in filteredWarehouseProducts" :key="p.id"
              style="cursor:pointer"
              :style="editingWarehouseProduct?.id===p.id ? 'background:rgba(122,185,154,0.08)' : ''"
              @click="openProductDetail(p)"
              @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(122,185,154,0.07)'"
              @mouseleave="($event.currentTarget as HTMLElement).style.background=editingWarehouseProduct?.id===p.id ? 'rgba(122,185,154,0.08)' : ''">
              <td style="font-weight:500">{{ p.name }}</td>
              <td style="font-family:monospace;font-size:0.8rem">{{ p.sku }}</td>
              <td>{{ p.category || '—' }}</td>
              <td>{{ p.brand || '—' }}</td>
              <td>
                <span :class="p.warehouseQty <= p.minStock ? 'chip chip--warn' : 'chip chip--ok'">
                  {{ p.warehouseQty }}
                </span>
              </td>
              <td>{{ p.retailPrice?.toFixed(2) }} {{ p.currency }}</td>
              <td @click.stop>
                <button v-if="can('products:write')" class="ghost-button" style="padding:0.2rem 0.5rem;font-size:0.8rem"
                  @click="startEditWarehouseProduct(p)"> Редагувати</button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="subtle">У цьому місці немає товарів із залишком.</p>
      </div>
    </div>

    <!-- MODAL: Add/Edit Product -->
    <div v-if="showAddProduct" class="modal-backdrop" @click.self="showAddProduct=false;showEditProduct=null;stopBarcodeCamera()">
      <div class="panel modal-box">
        <h3>{{ showEditProduct ? 'Редагувати товар' : 'Новий товар' }}</h3>
        <div class="grid">
          <label>Назва <input v-model="newProduct.name" placeholder="Назва товару"></label>
          <div>
            <div style="font-size:0.8rem;font-weight:500;margin-bottom:0.3rem">SKU</div>
            <div style="display:flex;gap:0.4rem;align-items:center">
              <input v-model="newProduct.sku" placeholder="SKU" style="flex:1">
              <button v-if="!showEditProduct" type="button" class="ghost-button"
                style="padding:0.4rem 0.75rem;font-size:0.8rem;white-space:nowrap"
                :style="newProduct.sku ? 'opacity:0.4;cursor:not-allowed' : ''"
                :disabled="!!newProduct.sku"
                :title="newProduct.sku ? 'SKU вже заповнено' : 'Згенерувати SKU автоматично'"
                @click="generateSku">
                Згенерувати
              </button>
            </div>
          </div>
          <div>
            <div style="font-size:0.8rem;font-weight:500;margin-bottom:0.3rem">Штрихкод</div>
            <div style="display:flex;gap:0.4rem;align-items:center">
              <input v-model="newProduct.barcode" placeholder="Введіть або скануйте штрихкод" style="flex:1">
              <button type="button" class="ghost-button"
                style="padding:0.4rem 0.75rem;font-size:0.8rem;white-space:nowrap"
                :style="barcodeScanning ? 'border-color:rgba(72,187,120,0.6);color:#6dd4a0' : ''"
                @click="barcodeScanning ? stopBarcodeCamera() : startBarcodeCamera()">
                {{ barcodeScanning ? 'Зупинити' : 'Сканувати' }}
              </button>
              <button v-if="!newProduct.barcode && showEditProduct" type="button" class="ghost-button"
                style="padding:0.4rem 0.75rem;font-size:0.8rem;white-space:nowrap"
                @click="genBarcode(showEditProduct); showAddProduct=false"
                title="Згенерувати штрихкод автоматично">
                Згенерувати
              </button>
            </div>
            <div v-if="barcodeScanning" style="margin-top:0.5rem">
              <div style="font-size:0.78rem;color:#6dd4a0;margin-bottom:0.3rem">Наведіть камеру на штрихкод…</div>
              <video ref="barcodeVideoRef"
                style="width:100%;max-width:320px;border-radius:0.6rem;border:2px solid rgba(72,187,120,0.4);display:block"
                autoplay muted playsinline />
              <div style="font-size:0.75rem;color:#aaa;margin-top:0.25rem">Штрихкод буде вставлено автоматично</div>
            </div>
            <div v-if="barcodeScanError" style="font-size:0.78rem;color:#e07070;margin-top:0.25rem">{{ barcodeScanError }}</div>
            <div style="font-size:0.75rem;color:#6a9e84;margin-top:0.25rem">
              Можна сканувати USB-сканером — просто клікніть у поле і скануйте
            </div>
          </div>
          <label>Категорія <input v-model="newProduct.category" placeholder="Категорія"></label>
          <label>Бренд <input v-model="newProduct.brand" placeholder="Бренд"></label>

          <label>Постачальник
            <div style="position:relative">
              <input
                v-model="supplierSearch"
                placeholder="Введіть назву постачальника"
                autocomplete="off"
                @focus="showSupplierDropdown=true"
                @input="showSupplierDropdown=true; newProduct.supplier=supplierSearch"
                @blur="hideDropdown(v => showSupplierDropdown = v)"
              >
              <div v-if="showSupplierDropdown && productFormSuppliers.length"
                style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                <div
                  v-for="s in productFormSuppliers.filter(s => !supplierSearch || s.name.toLowerCase().includes(supplierSearch.toLowerCase()))"
                  :key="s.id"
                  style="padding:0.45rem 0.75rem;cursor:pointer;font-size:0.88rem;border-bottom:1px solid rgba(122,185,154,0.08)"
                  @mouseenter="($event.currentTarget as HTMLElement).style.background='rgba(122,185,154,0.1)'"
                  @mouseleave="($event.currentTarget as HTMLElement).style.background=''"
                  @mousedown.prevent="newProduct.supplier=s.name; supplierSearch=s.name; showSupplierDropdown=false"
                >{{ s.name }}</div>
                <div v-if="!productFormSuppliers.filter(s => !supplierSearch || s.name.toLowerCase().includes(supplierSearch.toLowerCase())).length"
                  style="padding:0.45rem 0.75rem;font-size:0.82rem;color:var(--text-muted)">Не знайдено</div>
              </div>
            </div>
          </label>

          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem">
            <label>Ціна закупівлі <input type="number" v-model.number="newProduct.purchasePrice"></label>
            <label>Ціна роздрібна <input type="number" v-model.number="newProduct.retailPrice"></label>
          </div>
          <label>Валюта
            <select v-model="newProduct.currency">
              <option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option>
            </select>
          </label>
          <label>Позиція на полиці <input v-model="newProduct.warehousePosition" placeholder="A1-B3"></label>
          <label>Коментар <textarea v-model="newProduct.comments" rows="2"></textarea></label>
          <hr style="border-color:rgba(100,200,140,0.15);margin:0.3rem 0">

          <!-- NEW PRODUCT: initial stock placement -->
          <div v-if="!showEditProduct">
            <div style="font-size:0.88rem;font-weight:600;color:#7ab99a;margin-bottom:0.6rem">Початкова кількість на складі</div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem">
              <label>Кількість (шт.)
                <input type="number" min="0" v-model.number="newProduct.stock" placeholder="0">
                <span style="font-size:0.75rem;color:#6a9e84;margin-top:0.15rem;display:block">Скільки одиниць є в наявності</span>
              </label>
              <label>Мін. залишок (шт.)
                <input type="number" min="0" v-model.number="newProduct.minStock" placeholder="1">
                <span style="font-size:0.75rem;color:#6a9e84;margin-top:0.15rem;display:block">Попередження при меншому залишку</span>
              </label>
            </div>
            <label style="margin-top:0.5rem">Розмістити на складі / магазині
              <select v-model.number="newProduct.initialWarehouseId">
                <option :value="0">— Не обирати —</option>
                <optgroup label="Магазини">
                  <option v-for="w in warehouses.filter(w => w.locationType === 'shop')" :key="w.id" :value="w.id">
                    {{ w.name }}
                  </option>
                </optgroup>
                <optgroup label="Склади">
                  <option v-for="w in warehouses.filter(w => w.locationType !== 'shop')" :key="w.id" :value="w.id">
                    {{ w.name }}
                  </option>
                </optgroup>
              </select>
            </label>
            <div v-if="newProduct.initialWarehouseId && (newProduct.stock || 0) > 0"
              style="font-size:0.8rem;color:#6dd4a0;background:rgba(72,187,120,0.08);border-radius:6px;padding:0.4rem 0.6rem;margin-top:0.3rem">
              {{ newProduct.stock }} шт. буде зараховано на склад при збереженні
            </div>
            <div v-else-if="!newProduct.initialWarehouseId && (newProduct.stock || 0) > 0"
              style="font-size:0.8rem;color:#e0a060;background:rgba(224,160,96,0.08);border-radius:6px;padding:0.4rem 0.6rem;margin-top:0.3rem">
              Оберіть склад, щоб зарахувати кількість
            </div>
          </div>

          <!-- EDIT PRODUCT: show current stock + quick adjust -->
          <div v-if="showEditProduct">
            <div style="font-size:0.88rem;font-weight:600;color:#7ab99a;margin-bottom:0.5rem">Залишок на складі</div>
            <div style="background:rgba(72,187,120,0.07);border-radius:8px;padding:0.6rem 0.8rem;margin-bottom:0.5rem">
              <div style="font-size:1.1rem;font-weight:700;color:var(--accent)">
                {{ stockForProduct(showEditProduct.id) }} шт.
              </div>
              <div style="font-size:0.78rem;color:#aaa;margin-top:0.2rem">
                <span v-for="s in stockByWarehouse(showEditProduct.id)" :key="s.warehouse" style="margin-right:0.6rem">
                  {{ s.warehouse }}: {{ s.qty }}
                </span>
              </div>
            </div>
            <label style="margin-top:0.4rem;margin-bottom:0.5rem">Склад для операції
              <select v-model.number="movForm.warehouseId">
                <option :value="0">— Основний склад —</option>
                <optgroup label="Магазини">
                  <option v-for="w in warehouses.filter(w => w.locationType === 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                </optgroup>
                <optgroup label="Склади">
                  <option v-for="w in warehouses.filter(w => w.locationType !== 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                </optgroup>
              </select>
            </label>
            <div style="display:flex;gap:0.5rem">
              <button type="button" class="ghost-button" style="font-size:0.82rem;padding:0.35rem 0.75rem"
                @click="movForm.productId=showEditProduct!.id; movForm.type='receipt'; movForm.quantity=1; movForm.note=''; showMovement=true; showAddProduct=false">
                + Додати кількість
              </button>
              <button type="button" class="ghost-button" style="font-size:0.82rem;padding:0.35rem 0.75rem"
                @click="movForm.productId=showEditProduct!.id; movForm.type='writeoff'; movForm.quantity=1; movForm.note=''; showMovement=true; showAddProduct=false">
                − Списати
              </button>
              <button type="button" class="ghost-button" style="font-size:0.82rem;padding:0.35rem 0.75rem"
                @click="movForm.productId=showEditProduct!.id; movForm.type='adjustment'; movForm.quantity=stockForProduct(showEditProduct!.id); movForm.note=''; showMovement=true; showAddProduct=false">
                Скоригувати
              </button>
            </div>
            <label style="margin-top:0.6rem">Мін. залишок (шт.)
              <input type="number" min="0" v-model.number="newProduct.minStock" placeholder="1">
              <span style="font-size:0.75rem;color:#6a9e84;margin-top:0.15rem;display:block">Попередження при меншому залишку</span>
            </label>
          </div>

          <hr style="border-color:rgba(100,200,140,0.15);margin:0.3rem 0">
          <div style="font-size:0.8rem;color:#7ab99a;margin-bottom:0.2rem">Мапінг назв постачальника</div>
          <label>SKU постачальника <input v-model="newProduct.supplierSku" placeholder="Артикул у постачальника"></label>
          <label>Назва у постачальника <input v-model="newProduct.supplierNameExt" placeholder="Як цей товар називає постачальник"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="saveProduct" :disabled="saving">{{ saving ? 'Збереження...' : 'Зберегти' }}</button>
          <button class="ghost-button" @click="showAddProduct=false;showEditProduct=null;stopBarcodeCamera()">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Movement -->
    <!-- MODAL: Повернення постачальнику -->
    <div v-if="showReturnToSupplier" class="modal-backdrop" @click.self="showReturnToSupplier=false">
      <div class="panel modal-box" style="max-width:400px;width:100%">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:1rem;border-bottom:1px solid var(--border);padding-bottom:0.75rem">
          <h3 style="margin:0;font-size:1rem">↪️ Повернення постачальнику</h3>
          <button class="ghost-button" style="padding:0 0.4rem;font-size:1.1rem;line-height:1" @click="showReturnToSupplier=false">✕</button>
        </div>
        <div v-if="returnToSupplierProductId" style="margin-bottom:1rem">
          <div style="font-size:0.82rem;color:#aaa;margin-bottom:0.25rem">Товар:</div>
          <div style="font-weight:600;font-size:0.95rem">{{ products.find(p => p.id === returnToSupplierProductId)?.name }}</div>
          <div style="font-size:0.8rem;color:#aaa;margin-top:0.3rem">Зараз на складі: <strong style="color:var(--accent)">{{ stockForProduct(returnToSupplierProductId) }} шт.</strong></div>
        </div>
        <div class="grid">
          <label>
            Кількість для повернення (шт.)
            <input type="number" min="1" :max="stockForProduct(returnToSupplierProductId)" v-model.number="returnToSupplierQty" autofocus>
            <span style="font-size:0.78rem;color:#e07070;margin-top:0.15rem;display:block">
              Буде після повернення: {{ stockForProduct(returnToSupplierProductId) - (returnToSupplierQty || 0) }} шт.
            </span>
          </label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="doReturnToSupplier"
            :disabled="saving || !returnToSupplierQty || returnToSupplierQty <= 0 || returnToSupplierQty > stockForProduct(returnToSupplierProductId)"
            style="background:#e0a060;color:#fff;border:none">
            {{ saving ? 'Збереження...' : '↪️ Підтвердити повернення' }}
          </button>
          <button class="ghost-button" @click="showReturnToSupplier=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <div v-if="showMovement" class="modal-backdrop" @click.self="showMovement=false">
      <div class="panel modal-box">
        <h3>Змінити кількість товару</h3>
        <div class="grid">
          <label>Товар
            <select v-model.number="movForm.productId">
              <option value="0" disabled>Оберіть товар</option>
              <option v-for="p in products" :key="p.id" :value="p.id">{{ p.name }} ({{ p.sku }})</option>
            </select>
          </label>

          <label>Склад / Магазин
            <select v-model.number="movForm.warehouseId">
              <option :value="0">— Основний склад —</option>
              <optgroup label="Магазини">
                <option v-for="w in warehouses.filter(w => w.locationType === 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
              </optgroup>
              <optgroup label="Склади">
                <option v-for="w in warehouses.filter(w => w.locationType !== 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
              </optgroup>
            </select>
          </label>

          <!-- Current stock display -->
          <div v-if="movForm.productId" style="background:rgba(72,187,120,0.07);border-radius:8px;padding:0.55rem 0.8rem">
            <span style="font-size:0.82rem;color:#aaa">Зараз на складі: </span>
            <span style="font-weight:700;font-size:1rem;color:var(--accent)">{{ stockForProduct(movForm.productId) }} шт.</span>
          </div>

          <label>Що зробити?
            <select v-model="movForm.type">
              <option value="receipt">Надходження — збільшити кількість (прийшов товар)</option>
              <option value="writeoff">Списання — зменшити кількість (брак, втрата)</option>
              <option value="adjustment">Коригування — встановити точну кількість</option>
            </select>
          </label>

          <label>
            {{ movForm.type === 'adjustment' ? 'Нова кількість (шт.)' : 'Кількість (шт.)' }}
            <input type="number" min="0" v-model.number="movForm.quantity">
            <span v-if="movForm.productId && movForm.type === 'receipt'" style="font-size:0.78rem;color:#6dd4a0;margin-top:0.15rem;display:block">
              Буде: {{ stockForProduct(movForm.productId) + (movForm.quantity || 0) }} шт.
            </span>
            <span v-else-if="movForm.productId && movForm.type === 'writeoff'" style="font-size:0.78rem;color:#e07070;margin-top:0.15rem;display:block">
              Буде: {{ stockForProduct(movForm.productId) - (movForm.quantity || 0) }} шт.
            </span>
            <span v-else-if="movForm.productId && movForm.type === 'adjustment'" style="font-size:0.78rem;color:#e0c060;margin-top:0.15rem;display:block">
              Зміна: {{ (movForm.quantity || 0) - stockForProduct(movForm.productId) >= 0 ? '+' : '' }}{{ (movForm.quantity || 0) - stockForProduct(movForm.productId) }} шт.
            </span>
          </label>
          <label>Примітка (необов'язково)
            <input v-model="movForm.note" placeholder="Наприклад: інвентаризація, повернення від клієнта...">
          </label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="doMovement" :disabled="saving || !movForm.productId || !movForm.quantity">
            {{ saving ? 'Збереження...' : 'Зберегти зміни' }}
          </button>
          <button class="ghost-button" @click="showMovement=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Transfer -->
    <div v-if="showTransfer" class="modal-backdrop" @click.self="showTransfer=false">
      <div class="panel modal-box">
        <h3>Переміщення між складами</h3>
        <div class="grid">
          <label>Зі складу
            <select v-model.number="transferForm.fromWarehouseId">
              <option value="0" disabled>Оберіть склад</option>
              <option v-for="w in warehouses" :key="w.id" :value="w.id">{{ w.name }}</option>
            </select>
          </label>
          <label>На склад
            <select v-model.number="transferForm.toWarehouseId">
              <option value="0" disabled>Оберіть склад</option>
              <option v-for="w in warehouses" :key="w.id" :value="w.id">{{ w.name }}</option>
            </select>
          </label>
          <label>Товар
            <select v-model.number="transferForm.productId">
              <option value="0" disabled>Оберіть товар</option>
              <option v-for="p in products" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </label>
          <label>Кількість <input type="number" min="1" v-model.number="transferForm.quantity"></label>
          <label>Примітка <input v-model="transferForm.note"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="doTransfer" :disabled="saving">{{ saving?'..':'Перемістити' }}</button>
          <button class="ghost-button" @click="showTransfer=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: New Warehouse -->
    <div v-if="showAddWarehouse" class="modal-backdrop" @click.self="showAddWarehouse=false">
      <div class="panel modal-box">
        <h3>Нове місце зберігання</h3>
        <div class="grid">
          <label>Назва <input v-model="newWarehouse.name" placeholder="Назва складу / магазину"></label>
          <label>Тип
            <select v-model="newWarehouse.locationType">
              <option value="warehouse">🏭 Склад</option>
              <option value="shop">🏪 Магазин</option>
            </select>
          </label>
          <label style="flex-direction:row;align-items:center;gap:0.5rem">
            <input type="checkbox" v-model="newWarehouse.isVirtual"> Віртуальний
          </label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="addWarehouse" :disabled="saving||!newWarehouse.name">{{ saving?'..':'Зберегти' }}</button>
          <button class="ghost-button" @click="showAddWarehouse=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>
    <!-- MODAL: QR / Barcode -->
    <div v-if="showQrProduct" class="modal-backdrop" @click.self="showQrProduct=null">
      <div class="panel modal-box" style="max-width:340px;text-align:center">
        <h3 style="margin-bottom:0.3rem">{{ showQrProduct.name }}</h3>
        <div style="font-family:monospace;font-size:0.88rem;color:#7ab99a;margin-bottom:0.3rem">{{ showQrProduct.barcode || showQrProduct.sku }}</div>
        <div class="subtle" style="font-size:0.78rem;margin-bottom:1rem">Штрихкод / QR для сканування на касі</div>
        <img
          :src="api.productQrSvgUrl(token, showQrProduct.id)"
          :alt="showQrProduct.barcode || showQrProduct.sku"
          style="width:100%;max-width:280px;border-radius:0.6rem;background:#fff;padding:0.5rem"
        >
        <div style="font-size:0.78rem;color:#6a9e84;margin-top:0.7rem">
          Роздрукуйте та наклейте на товар.<br>Сканер або камера на касі розпізнає його автоматично.
        </div>
        <div style="margin-top:1rem;display:flex;gap:0.5rem;justify-content:center">
          <button class="ghost-button" @click="printLabels('small', [showQrProduct!.id])">Роздрукувати цінник</button>
          <button class="ghost-button" @click="showQrProduct=null">Закрити</button>
        </div>
      </div>
    </div>

    <!-- MODAL: Bulk Price Update -->
    <div v-if="showBulkPrice" class="modal-backdrop" @click.self="showBulkPrice=false">
      <div class="panel modal-box">
        <h3>Масове оновлення цін</h3>
        <div class="grid">
          <label>Категорія (порожньо = всі)
            <select v-model="bulkPriceForm.applyToCategory">
              <option value="">Всі категорії</option>
              <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
            </select>
          </label>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem">
            <label>Націнка %
              <input type="number" v-model.number="bulkPriceForm.markupPct" placeholder="0">
            </label>
            <label>Фіксована націнка
              <input type="number" v-model.number="bulkPriceForm.markupFixed" placeholder="0">
            </label>
          </div>
          <label>Округлення ціни
            <select v-model="bulkPriceForm.roundMode">
              <option value="none">Без округлення</option>
              <option value="nearest">До найближчого</option>
              <option value="up">Вгору</option>
              <option value="down">Вниз</option>
            </select>
          </label>
          <label v-if="bulkPriceForm.roundMode !== 'none'">Округляти до
            <select v-model.number="bulkPriceForm.roundTo">
              <option :value="1">1 (без копійок)</option>
              <option :value="5">5</option>
              <option :value="10">10</option>
              <option :value="50">50</option>
              <option :value="100">100</option>
            </select>
          </label>
        </div>
        <p style="font-size:0.82rem;color:#7ab99a;margin-top:0.5rem">
          Буде оновлено роздрібну ціну на основі закупівельної ціни + націнка.
        </p>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="doBulkPrice" :disabled="saving">{{ saving ? 'Оновлення...' : 'Застосувати' }}</button>
          <button class="ghost-button" @click="showBulkPrice=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>
  </div>

    <!-- ═══ PRODUCT DETAIL MODAL ════════════════════════════════════════════ -->
    <div v-if="selectedProduct"
      style="position:fixed;inset:0;background:rgba(0,0,0,0.6);z-index:200;display:flex;align-items:flex-start;justify-content:center;padding:2rem 1rem;overflow-y:auto"
      @click.self="closeProductDetail">
      <div style="background:var(--bg-card);border-radius:12px;border:1px solid var(--border);width:100%;max-width:720px;padding:1.5rem;position:relative">

        <!-- Header -->
        <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:1.2rem">
          <div>
            <h2 style="margin:0 0 0.25rem">{{ selectedProduct.name }}</h2>
            <div style="display:flex;gap:0.5rem;flex-wrap:wrap">
              <span class="chip" style="font-size:0.75rem">SKU: {{ selectedProduct.sku }}</span>
              <span v-if="selectedProduct.barcode" class="chip" style="font-size:0.75rem">Штрихкод: {{ selectedProduct.barcode }}</span>
              <span v-if="selectedProduct.category" class="chip" style="font-size:0.75rem">{{ selectedProduct.category }}</span>
              <span v-if="selectedProduct.brand" class="chip" style="font-size:0.75rem">{{ selectedProduct.brand }}</span>
              <span v-if="selectedProduct.archived" class="chip chip--required" style="font-size:0.75rem">Архів</span>
            </div>
          </div>
          <button class="ghost-button" style="padding:0.3rem 0.7rem;flex-shrink:0" @click="closeProductDetail">✕</button>
        </div>

        <!-- Info grid -->
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-bottom:1.2rem">

          <!-- Prices -->
          <div class="panel" style="padding:1rem">
            <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.7rem">ЦІНИ</div>
            <div style="display:flex;justify-content:space-between;margin-bottom:0.4rem">
              <span class="subtle">Роздрібна:</span>
              <span style="font-weight:700;color:var(--accent)">{{ selectedProduct.retailPrice }} {{ selectedProduct.currency || 'UAH' }}</span>
            </div>
            <div v-if="can('products:write')" style="display:flex;justify-content:space-between;margin-bottom:0.4rem">
              <span class="subtle">Закупівельна:</span>
              <span style="font-weight:600">{{ selectedProduct.purchasePrice }} {{ selectedProduct.currency || 'UAH' }}</span>
            </div>
            <div v-if="selectedProduct.wholesalePrice" style="display:flex;justify-content:space-between">
              <span class="subtle">Оптова:</span>
              <span style="font-weight:600">{{ selectedProduct.wholesalePrice }} {{ selectedProduct.currency || 'UAH' }}</span>
            </div>
            <div v-if="selectedProduct.vatPercent" style="display:flex;justify-content:space-between;margin-top:0.4rem">
              <span class="subtle">ПДВ:</span>
              <span>{{ selectedProduct.vatPercent }}%</span>
            </div>
          </div>

          <!-- Supplier -->
          <div class="panel" style="padding:1rem">
            <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.7rem">ПОСТАЧАЛЬНИК</div>
            <div style="margin-bottom:0.3rem;font-weight:600">{{ selectedProduct.supplier || selectedProduct.supplierNameExt || '—' }}</div>
            <div v-if="selectedProduct.supplierSku" class="subtle" style="font-size:0.82rem">Арт. постач.: {{ selectedProduct.supplierSku }}</div>
            <div v-if="selectedProduct.article" class="subtle" style="font-size:0.82rem;margin-top:0.2rem">Артикул: {{ selectedProduct.article }}</div>
          </div>

          <!-- Stock per warehouse -->
          <div class="panel" style="padding:1rem;grid-column:1/-1">
            <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.7rem">СКЛАДИ</div>
            <div style="display:flex;gap:0.5rem;flex-wrap:wrap;margin-bottom:0.6rem">
              <div v-for="w in stockByWarehouse(selectedProduct.id).filter(x=>x.qty>0)" :key="w.warehouse"
                style="background:rgba(72,187,120,0.1);border:1px solid rgba(72,187,120,0.25);border-radius:8px;padding:0.4rem 0.8rem;font-size:0.85rem">
                <span style="color:var(--text-subtle)">{{ w.warehouse }}:</span>
                <span style="font-weight:700;margin-left:0.4rem"
                  :style="w.qty <= (selectedProduct.minStock||0) ? 'color:var(--danger)' : 'color:var(--ok)'">
                  {{ w.qty }}
                </span>
              </div>
              <div v-if="!stockByWarehouse(selectedProduct.id).filter(x=>x.qty>0).length"
                class="subtle" style="font-size:0.85rem">Товар відсутній на всіх складах</div>
            </div>
            <div style="display:flex;justify-content:space-between;font-size:0.85rem;border-top:1px solid var(--border);padding-top:0.5rem">
              <span class="subtle">Загальний залишок:</span>
              <span style="font-weight:700"
                :style="stockForProduct(selectedProduct.id) <= (selectedProduct.minStock||0) ? 'color:var(--danger)' : ''">
                {{ stockForProduct(selectedProduct.id) }}
                <span class="subtle" style="font-weight:400"> (мін: {{ selectedProduct.minStock }})</span>
              </span>
            </div>
            <div v-if="selectedProduct.warehousePosition" class="subtle" style="font-size:0.82rem;margin-top:0.4rem">
              📍 Місце зберігання: {{ selectedProduct.warehousePosition }}
            </div>
          </div>
        </div>

        <!-- Comments -->
        <div v-if="selectedProduct.comments" class="panel" style="padding:0.8rem;margin-bottom:1rem;font-size:0.88rem">
          <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.4rem">КОМЕНТАР</div>
          {{ selectedProduct.comments }}
        </div>

        <!-- Lifecycle documents -->
        <div class="panel" style="padding:1rem;margin-bottom:1rem">
          <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.7rem">ДОКУМЕНТИ / ІСТОРІЯ РУХУ</div>
          <div v-if="productDetailLoading" class="subtle" style="font-size:0.85rem">Завантаження...</div>
          <div v-else-if="productDetailLifecycle && productDetailLifecycle.events.length">
            <div v-for="ev in productDetailLifecycle.events" :key="ev.eventDate + ev.eventType + (ev.refId||'')"
              style="display:flex;gap:0.75rem;align-items:flex-start;padding:0.5rem 0;border-bottom:1px solid rgba(122,185,154,0.08)">
              <div style="width:28px;height:28px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:1rem;flex-shrink:0;background:rgba(72,187,120,0.08)">
                <span v-if="ev.eventType==='purchased'">📦</span>
                <span v-else-if="ev.eventType==='received'">🚚</span>
                <span v-else-if="ev.eventType==='sold'">🧾</span>
                <span v-else-if="ev.eventType==='returned_from_customer'">↩</span>
                <span v-else-if="ev.eventType==='returned_to_supplier'">↪</span>
                <span v-else-if="ev.eventType==='movement'">🔄</span>
                <span v-else>📋</span>
              </div>
              <div style="flex:1;font-size:0.85rem">
                <div style="font-weight:600;margin-bottom:0.15rem">
                  <span v-if="ev.eventType==='created'">Створено</span>
                  <span v-else-if="ev.eventType==='purchased'">Закуплено{{ ev.refId ? ` (Замовлення #${ev.refId})` : '' }}</span>
                  <span v-else-if="ev.eventType==='received'">Надходження{{ ev.refId ? ` #${ev.refId}` : '' }}</span>
                  <span v-else-if="ev.eventType==='sold'">Продано{{ ev.refId ? ` (Продаж #${ev.refId})` : '' }}</span>
                  <span v-else-if="ev.eventType==='returned_from_customer'">Повернення від клієнта{{ ev.refId ? ` #${ev.refId}` : '' }}</span>
                  <span v-else-if="ev.eventType==='returned_to_supplier'">Повернення постачальнику{{ ev.refId ? ` #${ev.refId}` : '' }}</span>
                  <span v-else-if="ev.eventType==='movement'">Переміщення на склад</span>
                  <span v-else>{{ ev.eventType }}</span>
                </div>
                <div style="display:flex;gap:0.75rem;flex-wrap:wrap;color:var(--text-subtle);font-size:0.78rem">
                  <span v-if="ev.quantity != null">Кількість: <strong>{{ ev.quantity }}</strong></span>
                  <span v-if="ev.price != null">Ціна: <strong>{{ ev.price }} {{ ev.currency }}</strong></span>
                  <span v-if="ev.supplierName">Постачальник: {{ ev.supplierName }}</span>
                  <span v-if="ev.customerName">Клієнт: {{ ev.customerName }}</span>
                  <span v-if="ev.warehouseName">Склад: {{ ev.warehouseName }}</span>
                  <span v-if="ev.note">📝 {{ ev.note }}</span>
                </div>
              </div>
              <div style="font-size:0.75rem;color:#888;white-space:nowrap;text-align:right;min-width:80px">
                {{ new Date(ev.eventDate).toLocaleDateString('uk', {day:'2-digit',month:'short'}) }}
                <div>{{ new Date(ev.eventDate).toLocaleTimeString('uk', {hour:'2-digit',minute:'2-digit'}) }}</div>
              </div>
            </div>
          </div>
          <div v-else class="subtle" style="font-size:0.85rem">Документів не знайдено</div>
        </div>

        <!-- Actions -->
        <div style="display:flex;gap:0.5rem;justify-content:flex-end;flex-wrap:wrap">
          <button v-if="can('products:write')" class="ghost-button"
            @click="startEdit(selectedProduct!); closeProductDetail()">
            ✏️ Редагувати
          </button>
          <button class="ghost-button"
            @click="movForm.productId=selectedProduct!.id; movForm.type='receipt'; movForm.quantity=1; movForm.note=''; showMovement=true; closeProductDetail()">
            📦 Рух товару
          </button>
          <button class="ghost-button" style="color:#e0a060;border-color:rgba(224,160,96,0.4)"
            @click="returnToSupplierProductId=selectedProduct!.id; returnToSupplierQty=1; showReturnToSupplier=true; closeProductDetail()">
            ↪️ Повернення постачальнику
          </button>
          <button v-if="selectedProduct.barcode || selectedProduct.sku" class="ghost-button"
            @click="showQrProduct=selectedProduct; closeProductDetail()">
            🖨 Друк
          </button>
          <button class="ghost-button" @click="closeProductDetail">Закрити</button>
        </div>
      </div>
    </div>


    <!-- MODAL: Bulk Transfer (переміщення вибраних товарів) -->
    <teleport to="body">
      <div v-if="showBulkTransfer" class="modal-backdrop" @click.self="showBulkTransfer=false;bulkTransferError=''">
        <div class="panel modal-box" style="max-width:520px;width:100%">
          <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:1rem;border-bottom:1px solid var(--border);padding-bottom:0.75rem">
            <h3 style="margin:0;font-size:1rem">🔄 Переміщення вибраних товарів</h3>
            <button class="ghost-button" style="padding:0 0.4rem;font-size:1.1rem;line-height:1" @click="showBulkTransfer=false;bulkTransferError=''">✕</button>
          </div>
          <div style="margin-bottom:1rem;max-height:200px;overflow-y:auto">
            <div style="font-size:0.72rem;color:var(--text-subtle);font-weight:600;letter-spacing:0.05em;margin-bottom:0.5rem">
              ВИБРАНІ ТОВАРИ ({{ selectedForLabel.length }})
            </div>
            <div v-for="id in selectedForLabel" :key="id"
              style="display:flex;justify-content:space-between;align-items:center;font-size:0.85rem;padding:0.3rem 0;border-bottom:1px solid rgba(122,185,154,0.08)">
              <span>{{ products.find(p => p.id === id)?.name || `#${id}` }}</span>
              <span style="font-size:0.78rem;color:var(--text-subtle)">
                {{ bulkTransferForm.fromWarehouseId
                    ? (warehouseStocks.find(s => s.productId === id && s.warehouseId === bulkTransferForm.fromWarehouseId)?.quantity ?? 0) + ' шт.'
                    : stockForProduct(id) + ' шт. (всього)' }}
              </span>
            </div>
          </div>
          <div class="grid">
            <label>Зі складу
              <select v-model.number="bulkTransferForm.fromWarehouseId">
                <option :value="0" disabled>Оберіть склад</option>
                <optgroup label="Магазини">
                  <option v-for="w in warehouses.filter(w => w.locationType === 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                </optgroup>
                <optgroup label="Склади">
                  <option v-for="w in warehouses.filter(w => w.locationType !== 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                </optgroup>
              </select>
            </label>
            <label>На склад
              <select v-model.number="bulkTransferForm.toWarehouseId">
                <option :value="0" disabled>Оберіть склад</option>
                <optgroup label="Магазини">
                  <option v-for="w in warehouses.filter(w => w.locationType === 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                </optgroup>
                <optgroup label="Склади">
                  <option v-for="w in warehouses.filter(w => w.locationType !== 'shop')" :key="w.id" :value="w.id">{{ w.name }}</option>
                </optgroup>
              </select>
            </label>
            <label>Примітка (необов'язково)
              <input v-model="bulkTransferForm.note" placeholder="Наприклад: інвентаризація, сезон...">
            </label>
          </div>
          <div v-if="bulkTransferForm.fromWarehouseId && bulkTransferForm.toWarehouseId && bulkTransferForm.fromWarehouseId !== bulkTransferForm.toWarehouseId"
            style="font-size:0.82rem;color:#60c0e0;background:rgba(90,180,220,0.08);border-radius:6px;padding:0.4rem 0.7rem;margin-top:0.5rem">
            Буде переміщено {{ selectedForLabel.length }} товарів зі складу
            «{{ warehouses.find(w => w.id === bulkTransferForm.fromWarehouseId)?.name }}»
            → «{{ warehouses.find(w => w.id === bulkTransferForm.toWarehouseId)?.name }}»
          </div>
          <p v-if="bulkTransferError" class="error-text">{{ bulkTransferError }}</p>
          <div style="display:flex;gap:0.5rem;margin-top:1.25rem;padding-top:0.75rem;border-top:1px solid var(--border);justify-content:flex-end">
            <button class="ghost-button" @click="showBulkTransfer=false;bulkTransferError=''">Скасувати</button>
            <button
              style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.45rem 1rem;font-size:0.85rem;cursor:pointer;font-weight:500"
              @click="doBulkTransfer"
              :disabled="bulkTransferSaving || !bulkTransferForm.fromWarehouseId || !bulkTransferForm.toWarehouseId || bulkTransferForm.fromWarehouseId === bulkTransferForm.toWarehouseId"
            >
              {{ bulkTransferSaving ? 'Переміщення...' : '🔄 Перемістити' }}
            </button>
          </div>
        </div>
      </div>
    </teleport>

    <!-- MODAL: Invoice Import (накладна) -->
    <teleport to="body">
      <div v-if="showInvoiceImport" class="modal-backdrop" @click.self="showInvoiceImport=false; invoiceResult=null; invoiceError=''">
        <div class="modal-box" style="max-width:500px;width:100%">
          <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:1rem;border-bottom:1px solid var(--border);padding-bottom:0.75rem">
            <h3 style="margin:0;font-size:1rem">📥 Імпорт видаткової накладної</h3>
            <button class="ghost-button" style="padding:0 0.4rem;font-size:1.1rem;line-height:1" @click="showInvoiceImport=false; invoiceResult=null; invoiceError=''">✕</button>
          </div>
          <p style="font-size:0.88rem;color:var(--text-muted);margin-bottom:0.8rem">
            Оберіть xlsx-файл накладної. Постачальник та відсутні номенклатури будуть створені автоматично.
          </p>
          <label style="display:block;margin-bottom:0.8rem;font-size:0.85rem">
            Файл xlsx
            <input type="file" accept=".xlsx" @change="onInvoiceFileChange" style="display:block;margin-top:0.4rem">
          </label>
          <div v-if="invoiceResult" style="background:rgba(109,212,160,0.1);border:1px solid rgba(109,212,160,0.4);border-radius:8px;padding:0.8rem;margin-bottom:0.8rem;font-size:0.9rem">
            <div style="color:#6dd4a0;font-weight:600;margin-bottom:0.4rem">✓ Надходження #{{ invoiceResult.purchaseId }} створено</div>
            <div>Постачальник: <strong>{{ invoiceResult.supplierName }}</strong></div>
            <div>Нових номенклатур: <strong>{{ invoiceResult.created }}</strong></div>
            <div>Знайдено в базі: <strong>{{ invoiceResult.found }}</strong></div>
          </div>
          <p v-if="invoiceError" class="error-text">{{ invoiceError }}</p>
          <div style="display:flex;justify-content:flex-end;gap:0.5rem;margin-top:1.25rem;padding-top:0.75rem;border-top:1px solid var(--border)">
            <button class="ghost-button" @click="showInvoiceImport=false; invoiceResult=null; invoiceError=''">Закрити</button>
            <button
              style="background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);padding:0.45rem 1rem;font-size:0.85rem;cursor:pointer;font-weight:500"
              @click="importInvoice"
              :disabled="invoiceImporting || !invoiceFile"
            >
              {{ invoiceImporting ? 'Імпортуємо...' : 'Імпортувати' }}
            </button>
          </div>
        </div>
      </div>
    </teleport>
</template>