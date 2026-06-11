<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { api } from "./api";
import AdminPanelView from "./components/AdminPanelView.vue";
import AnalyticsView from "./components/AnalyticsView.vue";
import BuyerPortalView from "./components/BuyerPortalView.vue";
import CounterpartiesView from "./components/CounterpartiesView.vue";
import DirectoryView from "./components/DirectoryView.vue";
import DocumentRegistryView from "./components/DocumentRegistryView.vue";
import OperationJournalView from "./components/OperationJournalView.vue";
import CashboxView from "./components/CashboxView.vue";
import DashboardView from "./components/DashboardView.vue";
import LoginView from "./components/LoginView.vue";
import NotificationsView from "./components/NotificationsView.vue";
import OrdersView from "./components/OrdersView.vue";
import ProcurementView from "./components/ProcurementView.vue";
import ProductsView from "./components/ProductsView.vue";
import RepairsView from "./components/RepairsView.vue";
import ReportsView from "./components/ReportsView.vue";
import SalesView from "./components/SalesView.vue";
import type { UserSession } from "./types";

type Tab =
  // Довідники
  | "products"
  | "categories"
  | "counterparties"
  | "currencies"
  | "warehouses"
  // Документи
  | "orders"
  | "supplier_orders"
  | "sales"
  | "purchases"
  | "returns"
  | "cash_orders"
  | "repairs"
  // Звіти
  | "report_debts"
  | "report_sales"
  | "report_purchases"
  | "report_profit"
  // Інше
  | "dashboard"
  | "analytics"
  | "notifications"
  | "admin"
  | "buyer"
  | "journal"
  | "registry"
  | "reports"
  | "orders_combined"
  | "directory_clients"
  | "directory_suppliers";

const session = ref<UserSession | null>(null);
const activeTab = ref<Tab>("dashboard");
const initialProductId = ref<number | undefined>(undefined);
const initialOrderId = ref<number | undefined>(undefined);

function can(p: string) { return (session.value?.permissions ?? []).includes(p); }
function canAny(ps: string[]) { return ps.some(can); }

// ── Theme ─────────────────────────────────────────────────────
const theme = ref<"dark" | "light">((localStorage.getItem("erp_theme") as any) ?? "dark");
function applyTheme(t: "dark" | "light") {
  document.documentElement.setAttribute("data-theme", t === "light" ? "light" : "dark");
  localStorage.setItem("erp_theme", t);
}
function toggleTheme() {
  theme.value = theme.value === "dark" ? "light" : "dark";
  applyTheme(theme.value);
}

// ── Nav groups ────────────────────────────────────────────────
const navGroups = computed(() => [
  {
    label: "Головна",
    items: [
      { id: "dashboard" as Tab, label: "Дашборд", icon: "⊞", show: true },
    ]
  },
  {
    label: "Довідники",
    items: [
      { id: "products" as Tab,       label: "Номенклатура", icon: "▦", show: canAny(["products:read", "warehouses:read"]) },
      { id: "counterparties" as Tab, label: "Контрагенти",  icon: "◎", show: canAny(["sales:read", "suppliers:read"]) },
    ]
  },
  {
    label: "Документи",
    items: [
      { id: "orders_combined" as Tab, label: "Замовлення",                  icon: "📦", show: canAny(["orders:read", "sales:read", "suppliers:read", "supplier_orders:read"]) },
      { id: "sales" as Tab,           label: "Продажі",                     icon: "▷", show: canAny(["sales:read", "sales:write"]) },
      { id: "cash_orders" as Tab,     label: "Касові ордери",               icon: "≡", show: canAny(["cashboxes:read", "cashops:read"]) },
    ]
  },
  {
    label: "Налаштування",
    items: [
      { id: "notifications" as Tab, label: "Сповіщення",    icon: "◉", show: canAny(["notifications:read", "notifications:write"]) },
      { id: "admin" as Tab,         label: "Адміністратор", icon: "◆", show: session.value?.role === "admin" || canAny(["users:read", "roles:read", "audit:read"]) },
    ]
  },
  {
    label: "Кабінет",
    items: [
      { id: "buyer" as Tab, label: "Мої замовлення", icon: "◇", show: session.value?.role === "buyer" },
    ]
  },
]);

// ── Routing helpers ───────────────────────────────────────────
// Some tabs are sub-sections of existing views; we route them here.

// Which "view component" to render for a given tab
type ViewKey = "dashboard" | "products" | "sales" | "procurement" | "repairs" | "cashbox" | "analytics" | "notifications" | "admin" | "buyer" | "counterparties" | "registry" | "journal" | "reports" | "orders_combined" | "directory";

const tabToView: Record<Tab, ViewKey> = {
  dashboard:       "dashboard",
  // Довідники
  products:        "products",
  categories:      "products",      // sub-tab inside ProductsView
  counterparties:  "counterparties",
  currencies:      "analytics",     // placeholder — no dedicated view yet
  warehouses:      "products",      // warehouses are managed inside ProductsView
  // Документи
  orders:          "sales",         // SalesView sub-tab "orders"
  supplier_orders: "procurement",   // ProcurementView sub-tab "orders"
  orders_combined: "orders_combined",
  sales:           "sales",         // SalesView sub-tab "sales"
  purchases:       "procurement",   // ProcurementView sub-tab "purchases"
  returns:         "sales",         // SalesView (returns handled there)
  cash_orders:     "cashbox",       // CashboxView
  repairs:         "repairs",
  // Звіти
  report_debts:    "cashbox",       // Debts tab inside CashboxView
  report_sales:    "analytics",
  report_purchases:"analytics",
  report_profit:   "analytics",
  reports:          "reports",
  // Довідник
  directory_clients:   "directory",
  directory_suppliers: "directory",
  // Інше
  analytics:       "analytics",
  notifications:   "notifications",
  admin:           "admin",
  buyer:           "buyer",
  journal:         "journal",
  registry:        "registry",
};

// Initial sub-tab hints to pass down to views
const tabToSubTab: Partial<Record<Tab, string>> = {
  orders:          "orders",
  sales:           "sales",
  purchases:       "purchases",
  supplier_orders: "orders",
  report_debts:    "debts",
  report_sales:    "sales",
  report_purchases:"purchases",
  report_profit:   "profitability",
  reports:         "suppliers",
  cash_orders:     "cashorders",
  warehouses:      "warehouses",
  categories:      "categories",
  directory_clients:   "customers",
  directory_suppliers: "suppliers",
};

const activeView = computed(() => tabToView[activeTab.value] ?? "dashboard");
const activeSubTab = computed(() => tabToSubTab[activeTab.value] ?? null);

async function enrichSession(value: UserSession) {
  if (!value.token) { session.value = value; return; }
  try {
    const info = await api.me(value.token);
    session.value = { ...value, role: info.role, user: info.user, permissions: info.permissions, scopes: info.scopes, features: info.features };
  } catch (e: any) {
    // If token is expired/invalid — clear session so user sees login screen
    if (e?.message === "SESSION_EXPIRED") {
      localStorage.removeItem("erp_session");
      session.value = null;
    } else {
      // Network error or other issue — keep session so app still works offline
      session.value = value;
    }
  }
}

async function onAuthenticated(value: UserSession) {
  await enrichSession(value);
  activeTab.value = "dashboard";
}

function logout() { session.value = null; activeTab.value = "dashboard"; }
function handleSessionExpired() { logout(); }

// Allow child views to navigate to a tab
function handleNavigate(payload: { type: string; id: number } | string) {
  if (typeof payload === "string") {
    activeTab.value = payload as Tab;
    return;
  }
  // Map entity type → tab
  const typeMap: Record<string, Tab> = {
    customer_order:   "orders_combined",
    supplier_order:   "supplier_orders",
    purchase:         "purchases",
    sale:             "sales",
    payment:          "cash_orders",
    supplier_payment: "cash_orders",
    product:          "products",
  };
  const tab = typeMap[payload.type];
  if (tab) {
    if (payload.type === "product") {
      initialProductId.value = payload.id;
    }
    if (payload.type === "supplier_order") {
      initialOrderId.value = payload.id;
    }
    activeTab.value = tab;
  }
}

onMounted(async () => {
  applyTheme(theme.value);
  const raw = localStorage.getItem("erp_session");
  if (!raw) return;
  try { await enrichSession(JSON.parse(raw) as UserSession); }
  catch { localStorage.removeItem("erp_session"); }
});

watch(session, (value) => {
  if (!value) { localStorage.removeItem("erp_session"); return; }
  localStorage.setItem("erp_session", JSON.stringify(value));
});
</script>

<template>
  <template v-if="session">
    <div class="app-shell">
      <!-- Sidebar -->
      <aside class="sidebar">
        <div class="sidebar__logo">
          <div style="font-size:1.1rem;font-weight:800;color:var(--accent);letter-spacing:-0.02em">VEKMA</div>
          <div style="font-size:0.65rem;color:var(--text-subtle)">automate. manage. grow.</div>
        </div>

        <div v-for="group in navGroups" :key="group.label">
          <template v-if="group.items.some(i => i.show)">
            <div class="sidebar__section">
              <div class="sidebar__section-label">{{ group.label }}</div>
              <template v-for="item in group.items" :key="item.id">
                <button v-if="item.show"
                  :class="['sidebar__nav-item', activeTab === item.id && 'sidebar__nav-item--active']"
                  @click="activeTab = item.id">
                  <span class="sidebar__nav-icon">{{ item.icon }}</span>
                  {{ item.label }}
                </button>
              </template>
            </div>
          </template>
        </div>

        <!-- Footer -->
        <div class="sidebar__footer">
          <div class="sidebar__user">
            <div class="sidebar__username">{{ session.user }}</div>
            <div class="sidebar__role">{{ session.role }}</div>
          </div>
          <button class="ghost-button" style="padding:0.4rem 0.5rem;font-size:1rem;flex-shrink:0" @click="toggleTheme" :title="theme === 'dark' ? 'Світла тема' : 'Темна тема'">
            {{ theme === 'dark' ? '☀️' : '🌙' }}
          </button>
          <button class="ghost-button" style="padding:0.4rem 0.6rem;font-size:0.8rem;flex-shrink:0" @click="logout">Вийти</button>
        </div>
      </aside>

      <!-- Main -->
      <main class="main-content">
        <DashboardView
          v-if="activeView === 'dashboard'"
          :session="session"
          @session-expired="handleSessionExpired"
          @navigate="handleNavigate" />

        <ProductsView
          v-else-if="activeView === 'products'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          :initial-product-id="initialProductId"
          @session-expired="handleSessionExpired" />

        <OrdersView
          v-else-if="activeView === 'orders_combined'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          @session-expired="handleSessionExpired" />

        <!-- SalesView handles: sales, orders, returns -->
        <SalesView
          v-else-if="activeView === 'sales'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          @session-expired="handleSessionExpired"
          @navigate="handleNavigate" />

        <!-- ProcurementView handles: supplier orders, purchases, recommendations, suppliers -->
        <ProcurementView
          v-else-if="activeView === 'procurement'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          :initial-order-id="initialOrderId"
          @session-expired="handleSessionExpired"
          @navigate="handleNavigate" />

        <RepairsView
          v-else-if="activeView === 'repairs'"
          :session="session"
          @session-expired="handleSessionExpired" />

        <!-- CashboxView handles: cashbox, cash orders, debts -->
        <CashboxView
          v-else-if="activeView === 'cashbox'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          @session-expired="handleSessionExpired"
          @navigate="handleNavigate" />

        <!-- AnalyticsView handles: sales report, purchases report, profitability -->
        <AnalyticsView
          v-else-if="activeView === 'analytics'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          @session-expired="handleSessionExpired" />

        <NotificationsView
          v-else-if="activeView === 'notifications'"
          :session="session"
          @session-expired="handleSessionExpired" />

        <OperationJournalView
          v-else-if="activeView === 'journal'"
          :session="session!"
          @session-expired="handleSessionExpired" />

        <DocumentRegistryView
          v-else-if="activeView === 'registry'"
          :session="session!"
          @session-expired="handleSessionExpired" />

        <CounterpartiesView
          v-else-if="activeView === 'counterparties'"
          :session="session"
          @session-expired="handleSessionExpired" />

        <AdminPanelView
          v-else-if="activeView === 'admin'"
          :session="session"
          @session-expired="handleSessionExpired" />

        <ReportsView
          v-else-if="activeView === 'reports'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          @session-expired="handleSessionExpired" />

        <BuyerPortalView
          v-else-if="activeView === 'buyer'"
          :session="session"
          @session-expired="handleSessionExpired" />

        <DirectoryView
          v-else-if="activeView === 'directory'"
          :session="session"
          :initial-sub-tab="activeSubTab ?? undefined"
          @session-expired="handleSessionExpired" />
      </main>
    </div>
  </template>
  <LoginView v-else @authenticated="onAuthenticated" />
</template>
