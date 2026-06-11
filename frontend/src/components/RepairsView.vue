<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import AttachmentsPanel from "./AttachmentsPanel.vue";
import { api } from "../api";
import type { UserSession, ServiceOrder, ServiceOrderStatus, Customer, Product, CurrencyCode } from "../types";

const props = defineProps<{ session: UserSession }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);

const orders = ref<ServiceOrder[]>([]);
const customers = ref<Customer[]>([]);
const products = ref<Product[]>([]);
const loading = ref(false);
const error = ref("");
const saving = ref(false);

const filterStatus = ref<ServiceOrderStatus | "">("");
const selectedOrder = ref<ServiceOrder | null>(null);

const showNewOrder = ref(false);
const newOrder = ref({
  customerId: 0, productId: undefined as number | undefined,
  title: "", description: "", technician: "",
  laborMin: 0, price: 0, currency: "UAH" as CurrencyCode
});
const customerNameInput = ref("");

const showAddPart = ref(false);
const partForm = ref({ productId: 0, quantity: 1, price: 0 });
const partProductSearch = ref("");
const showPartProductDropdown = ref(false);
const showPartCreateProduct = ref(false);
const partNewProductForm = ref({ name: "", category: "", brand: "", retailPrice: 0, currency: "UAH" });
const savingPartProduct = ref(false);

const filteredPartProducts = computed(() => {
  const s = partProductSearch.value.trim().toLowerCase();
  if (!s) return products.value.slice(0, 8);
  return products.value.filter(p => p.name.toLowerCase().includes(s) || (p.sku || "").toLowerCase().includes(s)).slice(0, 10);
});

async function createPartProduct() {
  if (!partNewProductForm.value.name.trim()) { error.value = "Введіть назву товару"; return; }
  savingPartProduct.value = true; error.value = "";
  try {
    const created = await api.createProduct(token.value, {
      name: partNewProductForm.value.name,
      category: partNewProductForm.value.category,
      brand: partNewProductForm.value.brand,
      retailPrice: partNewProductForm.value.retailPrice,
      currency: partNewProductForm.value.currency || "UAH",
      stock: 0, minStock: 0,
    } as any);
    products.value = [...products.value, created];
    partForm.value.productId = created.id;
    partForm.value.price = created.retailPrice;
    partProductSearch.value = created.name;
    showPartCreateProduct.value = false;
    partNewProductForm.value = { name: "", category: "", brand: "", retailPrice: 0, currency: "UAH" };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { savingPartProduct.value = false; }
}

const productSearchInput = ref("");
const showCreateProduct = ref(false);
const newProductForm = ref({ name: "", category: "", brand: "", retailPrice: 0, currency: "UAH" });
const savingProduct = ref(false);

const filteredProducts = computed(() => {
  const s = productSearchInput.value.trim().toLowerCase();
  if (!s) return products.value;
  return products.value.filter(p => p.name.toLowerCase().includes(s) || (p.sku || "").toLowerCase().includes(s));
});

const showPayment = ref(false);
const paymentAmount = ref(0);

const statusFlow: ServiceOrderStatus[] = ["new", "in_progress", "done", "cancelled"];
const statusLabel: Record<string, string> = {
  new: "Нове", in_progress: "В роботі", done: "Готово", cancelled: "Скасовано"
};
const statusColor: Record<string, string> = {
  new: "#9fe8c4", in_progress: "#fad07a", done: "#7cf2b5", cancelled: "#ff9ca0"
};

async function load() {
  loading.value = true; error.value = "";
  try {
    const params: any = {};
    if (filterStatus.value) params.status = filterStatus.value;
    [orders.value, customers.value, products.value] = await Promise.all([
      api.serviceOrders(token.value, params),
      api.customers(token.value),
      api.products(token.value),
    ]);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

async function createOrder() {
  saving.value = true; error.value = "";
  try {
    const name = customerNameInput.value.trim();
    if (!name) { error.value = "Введіть ім'я клієнта"; saving.value = false; return; }
    let customer = customers.value.find(c => c.name.toLowerCase() === name.toLowerCase());
    if (!customer) {
      customer = await api.createCustomer(token.value, { name, phone: "", email: "", comment: "" });
      await load();
    }
    await api.createServiceOrder(token.value, { ...newOrder.value, customerId: customer.id });
    showNewOrder.value = false;
    customerNameInput.value = "";
    newOrder.value = { customerId: 0, productId: undefined, title: "", description: "", technician: "", laborMin: 0, price: 0, currency: "UAH" };
    await load();
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function updateStatus(order: ServiceOrder, status: ServiceOrderStatus) {
  try {
    const updated = await api.updateServiceOrderStatus(token.value, order.id, status);
    const idx = orders.value.findIndex(o => o.id === order.id);
    if (idx >= 0) orders.value[idx] = updated;
    if (selectedOrder.value?.id === order.id) selectedOrder.value = updated;
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
    else error.value = e.message;
  }
}

async function addPart() {
  if (!selectedOrder.value) return;
  saving.value = true; error.value = "";
  try {
    const updated = await api.addServiceOrderPart(token.value, selectedOrder.value.id, partForm.value);
    selectedOrder.value = updated;
    const idx = orders.value.findIndex(o => o.id === updated.id);
    if (idx >= 0) orders.value[idx] = updated;
    showAddPart.value = false;
    partForm.value = { productId: 0, quantity: 1, price: 0 };
    partProductSearch.value = "";
    showPartCreateProduct.value = false;
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function downloadPdf() {
  if (!selectedOrder.value) return;
  try {
    const blob = await api.serviceOrderActPdf(token.value, selectedOrder.value.id);
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = `act-${selectedOrder.value.id}.pdf`; a.click();
    URL.revokeObjectURL(url);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
    else if (e.message?.includes("акт виконаних робіт не знайдено") || e.message?.includes("act not found"))
      error.value = "Акт ще не створено. Натисніть «Акт (PDF)», щоб створити його.";
    else error.value = e.message;
  }
}

async function createAct() {
  if (!selectedOrder.value) return;
  saving.value = true;
  try {
    await api.createServiceOrderActDocument(token.value, selectedOrder.value.id, { note: "", autoPost: true });
    alert("Акт створено і проведено");
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { saving.value = false; }
}

async function createNewProduct() {
  if (!newProductForm.value.name.trim()) { error.value = "Введіть назву товару"; return; }
  savingProduct.value = true; error.value = "";
  try {
    const created = await api.createProduct(token.value, {
      name: newProductForm.value.name,
      category: newProductForm.value.category,
      brand: newProductForm.value.brand,
      retailPrice: newProductForm.value.retailPrice,
      currency: newProductForm.value.currency || "UAH",
      stock: 0, minStock: 0,
    } as any);
    products.value = [...products.value, created];
    newOrder.value.productId = created.id;
    productSearchInput.value = created.name;
    showCreateProduct.value = false;
    newProductForm.value = { name: "", category: "", brand: "", retailPrice: 0, currency: "UAH" };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { savingProduct.value = false; }
}

function selectProduct(p: Product | undefined) {
  newOrder.value.productId = p?.id;
  productSearchInput.value = p?.name ?? "";
}


function customerName(id: number) {
  return customers.value.find(c => c.id === id)?.name ?? `#${id}`;
}
function productName(id?: number) {
  if (!id) return "—";
  return products.value.find(p => p.id === id)?.name ?? `#${id}`;
}

const filteredOrders = computed(() => {
  if (!filterStatus.value) return orders.value;
  return orders.value.filter(o => o.status === filterStatus.value);
});

onMounted(load);
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1rem">
      <h2 style="margin:0">Сервіс та ремонти</h2>
      <button class="ghost-button" @click="showNewOrder=true">+ Нове замовлення</button>
    </div>

    <div style="display:grid;grid-template-columns:280px 1fr;gap:1rem;align-items:start">
      <!-- Left: list -->
      <div>
        <div style="margin-bottom:0.7rem">
          <select v-model="filterStatus" style="width:100%" @change="load">
            <option value="">Всі статуси</option>
            <option v-for="s in statusFlow" :key="s" :value="s">{{ statusLabel[s] }}</option>
          </select>
        </div>
        <p v-if="loading" class="subtle">Завантаження...</p>
        <div v-for="o in filteredOrders" :key="o.id"
          class="panel" style="padding:0.7rem;cursor:pointer;margin-bottom:0.5rem"
          :style="selectedOrder?.id === o.id ? 'border-color:rgba(72,187,120,0.6)' : ''"
          @click="selectedOrder=o">
          <div style="display:flex;justify-content:space-between;align-items:center">
            <span style="font-weight:600">#{{ o.id }} {{ o.title }}</span>
            <span class="chip" :style="`background:${statusColor[o.status]}22;color:${statusColor[o.status]};font-size:0.75rem;padding:0.2rem 0.5rem`">
              {{ statusLabel[o.status] }}
            </span>
          </div>
          <div class="subtle" style="font-size:0.82rem;margin-top:0.3rem">
            {{ customerName(o.customerId) }} · {{ new Date(o.createdAt).toLocaleDateString('uk') }}
          </div>
          <div style="margin-top:0.3rem;font-size:0.82rem">
            Разом: <strong>{{ o.total }} {{ o.currency }}</strong>
            · Борг: <span :style="o.debt>0?'color:#ff9ca0':''">{{ o.debt }}</span>
          </div>
        </div>
      </div>

      <!-- Right: detail -->
      <div v-if="selectedOrder">
        <div class="panel" style="padding:1rem">
          <div style="display:flex;justify-content:space-between;align-items:flex-start;flex-wrap:wrap;gap:0.5rem">
            <div>
              <h3 style="margin:0 0 0.3rem">#{{ selectedOrder.id }} — {{ selectedOrder.title }}</h3>
              <div class="subtle" style="font-size:0.88rem">
                Клієнт: {{ customerName(selectedOrder.customerId) }} ·
                Майстер: {{ selectedOrder.technician || '—' }}
              </div>
            </div>
            <div style="display:flex;gap:0.4rem;flex-wrap:wrap">
              <button v-for="s in statusFlow" :key="s"
                :class="['ghost-button', selectedOrder.status===s?'tab-button--active':'']"
                style="padding:0.3rem 0.6rem;font-size:0.8rem"
                @click="updateStatus(selectedOrder!, s)">
                {{ statusLabel[s] }}
              </button>
            </div>
          </div>

          <div style="margin-top:1rem;display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;font-size:0.9rem">
            <div><span class="subtle">Пристрій:</span> {{ productName(selectedOrder.productId) }}</div>
            <div><span class="subtle">Тривалість:</span> {{ selectedOrder.laborMin }} хв</div>
            <div><span class="subtle">Вартість роботи:</span> {{ selectedOrder.price }} {{ selectedOrder.currency }}</div>
            <div><span class="subtle">Запчастини:</span> {{ selectedOrder.partsTotal }} {{ selectedOrder.currency }}</div>
            <div><span class="subtle">Разом:</span> <strong>{{ selectedOrder.total }} {{ selectedOrder.currency }}</strong></div>
            <div><span class="subtle">Оплачено:</span> <span style="color:#6dd4a0">{{ selectedOrder.paid }}</span></div>
            <div><span class="subtle">Борг:</span> <span :style="selectedOrder.debt>0?'color:#ff9ca0;font-weight:700':''">{{ selectedOrder.debt }}</span></div>
          </div>

          <div style="margin-top:1rem">
            <div style="font-size:0.85rem;color:#7ab99a;margin-bottom:0.3rem">Опис несправності:</div>
            <div style="font-size:0.9rem">{{ selectedOrder.description || '—' }}</div>
          </div>

          <!-- Parts -->
          <div style="margin-top:1rem">
            <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
              <strong style="font-size:0.9rem">Запчастини</strong>
              <button class="ghost-button" style="padding:0.3rem 0.7rem;font-size:0.8rem" @click="showAddPart=true">+ Додати</button>
            </div>
            <table style="font-size:0.85rem">
              <thead><tr><th>Товар</th><th>К-сть</th><th>Ціна</th><th>Сума</th></tr></thead>
              <tbody>
                <tr v-if="!selectedOrder.parts?.length"><td colspan="4" class="subtle">Немає запчастин</td></tr>
                <tr v-for="part in selectedOrder.parts" :key="part.id">
                  <td>{{ productName(part.productId) }}</td>
                  <td>{{ part.quantity }}</td>
                  <td>{{ part.price }}</td>
                  <td>{{ part.total }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Вкладення -->
          <div style="margin-top:1.2rem">
            <AttachmentsPanel
              :token="token"
              entity-type="service_order"
              :entity-id="selectedOrder.id"
              :can-delete="true"
            />
          </div>

          <div style="display:flex;gap:0.5rem;margin-top:1rem;flex-wrap:wrap">
            <button class="ghost-button" @click="createAct"
              :disabled="saving || selectedOrder.status !== 'done'"
              :title="selectedOrder.status !== 'done' ? 'Акт можна створити лише після переведення замовлення у статус «Готово»' : 'Створити акт виконаних робіт'">
              Акт (PDF)
            </button>
            <button class="ghost-button" @click="downloadPdf">Завантажити акт</button>
          </div>
          <p v-if="error" class="error-text">{{ error }}</p>
        </div>
      </div>
      <div v-else class="panel" style="padding:2rem;text-align:center">
        <p class="subtle">Оберіть замовлення зі списку</p>
      </div>
    </div>

    <!-- MODAL: New Order -->
    <div v-if="showNewOrder" class="modal-backdrop" @click.self="showNewOrder=false; customerNameInput=''">
      <div class="panel modal-box">
        <h3>Нове сервісне замовлення</h3>
        <div class="grid">
          <label>Клієнт
            <input v-model="customerNameInput" list="repair-customers-list" placeholder="Введіть або оберіть клієнта">
            <datalist id="repair-customers-list">
              <option v-for="c in customers" :key="c.id" :value="c.name" />
            </datalist>
          </label>
          <label>Заголовок <input v-model="newOrder.title" placeholder="Опис замовлення"></label>
          <label>Опис несправності <textarea v-model="newOrder.description" rows="3"></textarea></label>
          <label>Майстер <input v-model="newOrder.technician"></label>
          <label>Пристрій (товар)
            <div style="position:relative">
              <input v-model="productSearchInput" placeholder="Пошук товару за назвою..." style="width:100%"
                @input="newOrder.productId = undefined">
              <div v-if="productSearchInput && !newOrder.productId"
                style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                <div v-if="filteredProducts.length === 0 && productSearchInput.trim()"
                  style="padding:0.5rem 0.7rem;font-size:0.85rem;color:#7ab99a">
                  Товар не знайдено
                </div>
                <div v-for="p in filteredProducts" :key="p.id"
                  style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.88rem"
                  @mousedown.prevent="selectProduct(p)"
                  @mouseover="($event.target as HTMLElement).style.background='rgba(72,187,120,0.12)'"
                  @mouseleave="($event.target as HTMLElement).style.background=''">
                  {{ p.name }} <span class="subtle" style="font-size:0.78rem">{{ p.sku }}</span>
                </div>
              </div>
            </div>
            <div style="display:flex;align-items:center;gap:0.5rem;margin-top:0.3rem">
              <span class="subtle" style="font-size:0.8rem">
                <template v-if="newOrder.productId">
                  ✓ Обрано: {{ products.find(p => p.id === newOrder.productId)?.name }}
                  <button class="ghost-button" style="padding:0.1rem 0.4rem;font-size:0.75rem;margin-left:0.3rem"
                    @click="newOrder.productId=undefined;productSearchInput=''">✕</button>
                </template>
                <template v-else>Або</template>
              </span>
              <button v-if="!newOrder.productId" class="ghost-button" style="padding:0.2rem 0.6rem;font-size:0.8rem"
                @click="showCreateProduct=true">+ Створити новий товар</button>
            </div>
          </label>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem">
            <label>Вартість роботи <input type="number" v-model.number="newOrder.price"></label>
            <label>Тривалість (хв) <input type="number" v-model.number="newOrder.laborMin"></label>
          </div>
          <label>Валюта
            <select v-model="newOrder.currency"><option value="UAH">UAH (грн)</option><option value="USD">USD ($)</option><option value="EUR">EUR (€)</option></select>
          </label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="createOrder" :disabled="saving||!customerNameInput.trim()||!newOrder.title">{{ saving?'..':'Створити' }}</button>
          <button class="ghost-button" @click="showNewOrder=false; customerNameInput=''">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- MODAL: Add Part -->
    <div v-if="showAddPart" class="modal-backdrop" @click.self="showAddPart=false">
      <div class="panel modal-box">
        <h3>Додати запчастину</h3>
        <div class="grid">
          <label>Товар
            <div v-if="partForm.productId > 0"
              style="display:flex;align-items:center;gap:0.4rem;padding:0.4rem 0.6rem;background:rgba(72,187,120,0.1);border-radius:6px;font-size:0.88rem;margin-top:0.3rem">
              <span style="flex:1;font-weight:500">✓ {{ products.find(p => p.id === partForm.productId)?.name }}</span>
              <button class="ghost-button" style="padding:0.1rem 0.4rem;font-size:0.75rem"
                @click="partForm.productId=0;partProductSearch='';partForm.price=0">✕ змінити</button>
            </div>
            <div v-else style="position:relative;margin-top:0.3rem">
              <input v-model="partProductSearch" placeholder="Пошук товару за назвою, SKU..."
                style="width:100%" @focus="showPartProductDropdown=true" @input="showPartProductDropdown=true">
              <div v-if="showPartProductDropdown && filteredPartProducts.length > 0"
                style="position:absolute;top:100%;left:0;right:0;background:var(--panel-bg);border:1px solid var(--accent,rgba(72,187,120,0.5));border-radius:var(--radius-sm,6px);z-index:9999;max-height:220px;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,0.45),0 2px 8px rgba(0,0,0,0.25)">
                <div v-for="p in filteredPartProducts" :key="p.id"
                  style="padding:0.45rem 0.7rem;cursor:pointer;font-size:0.86rem;border-bottom:1px solid rgba(72,187,120,0.08)"
                  @mousedown.prevent="partForm.productId=p.id;partForm.price=p.retailPrice;partProductSearch=p.name;showPartProductDropdown=false"
                  @mouseover="($event.currentTarget as HTMLElement).style.background='rgba(72,187,120,0.12)'"
                  @mouseleave="($event.currentTarget as HTMLElement).style.background=''">
                  <span style="font-weight:600">{{ p.name }}</span>
                  <span class="subtle" style="font-size:0.75rem;margin-left:0.4rem">{{ p.sku }}</span>
                  <span style="float:right;color:var(--accent);font-size:0.8rem">{{ p.retailPrice }} {{ p.currency }}</span>
                </div>
                <div style="padding:0.4rem 0.7rem;border-top:1px solid rgba(72,187,120,0.15)">
                  <button class="ghost-button" style="font-size:0.78rem;width:100%"
                    @mousedown.prevent="showPartCreateProduct=true;showPartProductDropdown=false">
                    + Створити новий товар
                  </button>
                </div>
              </div>
              <div v-if="partProductSearch && !showPartProductDropdown && filteredPartProducts.length===0"
                style="font-size:0.78rem;color:var(--text-muted);margin-top:0.3rem">
                Товар не знайдено —
                <button class="ghost-button" style="font-size:0.78rem;padding:0.1rem 0.3rem;display:inline"
                  @click="showPartCreateProduct=true">+ створити новий</button>
              </div>
            </div>
          </label>

          <!-- Quick create product for part -->
          <div v-if="showPartCreateProduct"
            style="background:rgba(72,187,120,0.06);border:1px solid rgba(72,187,120,0.2);border-radius:8px;padding:0.75rem">
            <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
              <span style="font-size:0.8rem;font-weight:600;color:#7ab99a">Новий товар</span>
              <button class="ghost-button" style="font-size:0.8rem" @click="showPartCreateProduct=false">✕</button>
            </div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
              <label style="font-size:0.78rem">Назва *<input v-model="partNewProductForm.name" class="input" placeholder="Назва товару" style="margin-top:0.2rem"></label>
              <label style="font-size:0.78rem">Категорія<input v-model="partNewProductForm.category" class="input" style="margin-top:0.2rem"></label>
            </div>
            <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:0.5rem;margin-bottom:0.5rem">
              <label style="font-size:0.78rem">Бренд<input v-model="partNewProductForm.brand" class="input" style="margin-top:0.2rem"></label>
              <label style="font-size:0.78rem">Роздрібна ціна<input type="number" v-model.number="partNewProductForm.retailPrice" min="0" class="input" style="margin-top:0.2rem"></label>
              <label style="font-size:0.78rem">Валюта
                <select v-model="partNewProductForm.currency" class="input" style="margin-top:0.2rem">
                  <option value="UAH">UAH</option><option value="USD">USD</option><option value="EUR">EUR</option>
                </select>
              </label>
            </div>
            <button style="font-size:0.8rem;padding:0.35rem 0.8rem" :disabled="savingPartProduct||!partNewProductForm.name.trim()"
              @click="createPartProduct">
              {{ savingPartProduct ? '..' : '✓ Зберегти товар' }}
            </button>
          </div>

          <label>Кількість <input type="number" min="1" v-model.number="partForm.quantity"></label>
          <label>Ціна <input type="number" min="0" v-model.number="partForm.price"></label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="addPart" :disabled="saving||!partForm.productId">{{ saving?'..':'Додати' }}</button>
          <button class="ghost-button" @click="showAddPart=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>
    <!-- MODAL: Create New Product -->
    <div v-if="showCreateProduct" class="modal-backdrop modal-backdrop--top" @click.self="showCreateProduct=false">
      <div class="panel modal-box">
        <h3>Новий товар</h3>
        <div class="grid">
          <label>Назва <input v-model="newProductForm.name" placeholder="Назва товару"></label>
          <label>Категорія <input v-model="newProductForm.category" placeholder="Категорія"></label>
          <label>Бренд <input v-model="newProductForm.brand" placeholder="Бренд"></label>
          <label>Роздрібна ціна <input type="number" min="0" v-model.number="newProductForm.retailPrice"></label>
          <label>Валюта
            <select v-model="newProductForm.currency">
              <option value="UAH">UAH (грн)</option>
              <option value="USD">USD ($)</option>
              <option value="EUR">EUR (€)</option>
            </select>
          </label>
        </div>
        <div style="display:flex;gap:0.5rem;margin-top:1rem">
          <button @click="createNewProduct" :disabled="savingProduct||!newProductForm.name.trim()">
            {{ savingProduct ? '..' : 'Створити товар' }}
          </button>
          <button class="ghost-button" @click="showCreateProduct=false">Скасувати</button>
        </div>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

  </div>
</template>
