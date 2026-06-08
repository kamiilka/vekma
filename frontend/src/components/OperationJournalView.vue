<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { api } from "../api";
import type { UserSession, AuditLog } from "../types";

const props = defineProps<{ session: UserSession }>();
const emit  = defineEmits<{ (e: "session-expired"): void }>();
const token = computed(() => props.session.token);

// ── State ──────────────────────────────────────────────────────
const logs    = ref<AuditLog[]>([]);
const loading = ref(false);
const error   = ref("");

const search   = ref("");
const filterUser   = ref("");
const filterEntity = ref("");
const filterFrom   = ref("");
const filterTo     = ref("");

// ── Human-readable labels ──────────────────────────────────────
const ACTION_LABELS: Record<string, string> = {
  "auth.login":                          "Вхід у систему",
  "order.create":                        "Створено замовлення покупця",
  "order.update":                        "Змінено замовлення покупця",
  "order.status.update":                 "Змінено статус замовлення",
  "sale.create":                         "Проведено продаж",
  "purchase.create":                     "Оформлено надходження",
  "purchase.import_csv":                 "Імпорт надходжень з CSV",
  "supplier_order.create":               "Створено замовлення постачальнику",
  "supplier_order.status.update":        "Змінено статус замовлення постачальнику",
  "supplier_order.receive":              "Отримано товар від постачальника",
  "supplier_order.update":               "Змінено замовлення постачальнику",
  "supplier_order.recommendation.create":       "Сформовано рекомендацію закупівлі",
  "supplier_order.recommendation.bulk_create":  "Масове формування рекомендацій",
  "payment.create":                      "Проведена оплата",
  "cash.operation.create":               "Касова операція",
  "cash_shift.open":                     "Відкрито зміну",
  "cash_shift.close":                    "Закрито зміну",
  "cashbox.create":                      "Додано касу",
  "product.create":                      "Додано товар",
  "product.update":                      "Змінено товар",
  "product.archive":                     "Товар архівовано",
  "product.import_xlsx":                 "Імпорт товарів з Excel",
  "product.import_csv":                  "Імпорт товарів з CSV",
  "product.bulk_update_prices":          "Масове оновлення цін",
  "product.merge_duplicates":            "Об'єднання дублікатів товарів",
  "product.barcode.generate":            "Генерація штрих-коду",
  "service_order.create":                "Відкрито ремонт",
  "service_order.update":                "Змінено ремонт",
  "service_order.status.update":         "Змінено статус ремонту",
  "service_order.reopen":                "Ремонт повторно відкрито",
  "service_order.part.add":              "Додано запчастину до ремонту",
  "service_order.auto_act.create":       "Автоматично створено акт",
  "document.create":                     "Створено документ",
  "document.post":                       "Проведено документ",
  "document.return_from_customer.create":"Повернення від покупця",
  "document.return_to_supplier.create":  "Повернення постачальнику",
  "document.service_order_act.create":   "Створено акт виконаних робіт",
  "document.service_order_act.cancel":   "Скасовано акт виконаних робіт",
  "document.import_csv":                 "Імпорт документів з CSV",
  "transfer.create":                     "Переміщення товарів",
  "transfer.cell.create":                "Переміщення між комірками",
  "transfer.cell.fifo.create":           "Переміщення (FIFO)",
  "inventory.create":                    "Розпочато інвентаризацію",
  "inventory.apply":                     "Проведено інвентаризацію",
  "stock.movement.create":               "Рух товару",
  "stock.movement.adjustment":           "Коригування залишку",
  "customer.create":                     "Додано клієнта",
  "supplier.create":                     "Додано постачальника",
  "counterparty.create":                 "Додано контрагента",
  "counterparty.update":                 "Змінено контрагента",
  "exchange_rate.upsert":                "Оновлено курс валюти",
  "user.create":                         "Додано користувача",
  "role.permissions.update":             "Змінено права ролі",
  "receipt.send":                        "Відправлено чек",
  "receipt.retry":                       "Повторна відправка чека",
  "receipt.retry.bulk":                  "Масова відправка чеків",
};

// Icon per action prefix
function actionIcon(action: string): string {
  if (action.startsWith("order."))           return "📋";
  if (action.startsWith("sale."))            return "🧾";
  if (action.startsWith("purchase."))        return "🚚";
  if (action.startsWith("supplier_order."))  return "📦";
  if (action.startsWith("payment.") || action.startsWith("cash")) return "💳";
  if (action.startsWith("service_order."))   return "⚙️";
  if (action.startsWith("document."))        return "📄";
  if (action.startsWith("transfer."))        return "🔄";
  if (action.startsWith("inventory."))       return "📊";
  if (action.startsWith("stock."))           return "📉";
  if (action.startsWith("product."))         return "🏷️";
  if (action.startsWith("customer."))        return "👤";
  if (action.startsWith("supplier.") || action.startsWith("counterparty.")) return "🤝";
  if (action.startsWith("auth."))            return "🔑";
  if (action.startsWith("user.") || action.startsWith("role.")) return "👥";
  if (action.startsWith("receipt."))         return "🧾";
  return "📌";
}

// Colour accent per action group
function actionColor(action: string): string {
  if (action.startsWith("order."))           return "#48bb78";
  if (action.startsWith("sale."))            return "#48bb78";
  if (action.startsWith("purchase."))        return "#6db8f0";
  if (action.startsWith("supplier_order."))  return "#6db8f0";
  if (action.startsWith("payment.") || action.startsWith("cash")) return "#e0c050";
  if (action.startsWith("service_order."))   return "#c084fc";
  if (action.startsWith("document."))        return "#94a3b8";
  if (action.startsWith("transfer."))        return "#a78bfa";
  if (action.startsWith("inventory."))       return "#38bdf8";
  if (action.startsWith("stock."))           return "#94a3b8";
  if (action.startsWith("product."))         return "#f97316";
  if (action.startsWith("customer.") || action.startsWith("counterparty.") || action.startsWith("supplier.")) return "#06b6d4";
  return "#6b7280";
}

function humanAction(action: string): string {
  return ACTION_LABELS[action] ?? action.replace(/\./g, " › ");
}

// Parse details key=value pairs for display
function parseDetails(details: string): { key: string; val: string }[] {
  if (!details) return [];
  const parts = details.split(" ");
  const result: { key: string; val: string }[] = [];
  const labelMap: Record<string, string> = {
    order_id: "Замовлення №", sale_id: "Продаж №", purchase_id: "Надходження №",
    supplier_order_id: "Зам. пост. №", service_order_id: "Ремонт №",
    product_id: "Товар №", user: "Користувач", cashbox_id: "Каса №",
    status: "Статус", sku: "Артикул", items: "Позицій",
    imported: "Імпортовано", updated: "Оновлено", skipped: "Пропущено",
    payment_id: "Оплата №", transfer_id: "Переміщення №",
    inventory_id: "Інвентаризація №", amount: "Сума",
    name: "Назва", id: "ID",
  };
  for (const p of parts) {
    const [k, v] = p.split("=");
    if (k && v !== undefined) {
      result.push({ key: labelMap[k] ?? k, val: v });
    } else if (p) {
      result.push({ key: "", val: p });
    }
  }
  return result;
}

// ── Date grouping ──────────────────────────────────────────────
type DayGroup = { dateKey: string; label: string; entries: AuditLog[] };

const grouped = computed((): DayGroup[] => {
  const map = new Map<string, AuditLog[]>();
  for (const log of logs.value) {
    const d = new Date(log.createdAt);
    const key = d.toLocaleDateString("uk-UA", { year: "numeric", month: "2-digit", day: "2-digit" });
    const list = map.get(key) ?? [];
    list.push(log);
    map.set(key, list);
  }
  const today     = new Date().toLocaleDateString("uk-UA", { year: "numeric", month: "2-digit", day: "2-digit" });
  const yesterday = new Date(Date.now() - 86400000).toLocaleDateString("uk-UA", { year: "numeric", month: "2-digit", day: "2-digit" });

  return [...map.entries()].map(([dateKey, entries]) => ({
    dateKey,
    label: dateKey === today ? "Сьогодні" : dateKey === yesterday ? "Вчора" : dateKey,
    entries,
  }));
});

const uniqueUsers = computed(() =>
  [...new Set(logs.value.map(l => l.user))].sort()
);

const uniqueEntities = computed(() =>
  [...new Set(logs.value.map(l => l.entity))].filter(Boolean).sort()
);

const ENTITY_LABELS: Record<string, string> = {
  customer_order: "Замовлення покупця", supplier_order: "Замовлення пост.",
  purchase: "Надходження", sale: "Продаж", service_order: "Ремонт",
  product: "Товар", payment: "Оплата", cashbox: "Каса",
  document: "Документ", transfer: "Переміщення", inventory: "Інвентаризація",
  customer: "Клієнт", supplier: "Постачальник", counterparty: "Контрагент",
  session: "Сесія", user: "Користувач",
};

// ── Load ──────────────────────────────────────────────────────
async function load() {
  loading.value = true; error.value = "";
  try {
    logs.value = await api.journal(token.value, {
      search:   search.value   || undefined,
      user:     filterUser.value   || undefined,
      entity:   filterEntity.value || undefined,
      from:     filterFrom.value   || undefined,
      to:       filterTo.value     || undefined,
      limit:    300,
    });
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

let timer: ReturnType<typeof setTimeout> | null = null;
watch([search, filterUser, filterEntity, filterFrom, filterTo], () => {
  if (timer) clearTimeout(timer);
  timer = setTimeout(load, 280);
});

onMounted(load);

function fmtTime(s: string) {
  return new Date(s).toLocaleTimeString("uk-UA", { hour: "2-digit", minute: "2-digit" });
}
function clearFilters() {
  search.value = ""; filterUser.value = ""; filterEntity.value = "";
  filterFrom.value = ""; filterTo.value = "";
}
const hasFilters = computed(() =>
  search.value || filterUser.value || filterEntity.value || filterFrom.value || filterTo.value
);
</script>

<template>
  <div class="page-content">

    <!-- Header -->
    <div class="page-header" style="margin-bottom:1.25rem">
      <div>
        <h2 style="margin:0 0 0.2rem">Журнал операцій</h2>
        <div class="subtle" style="font-size:0.8rem">Усі дії в системі — хто, що і коли зробив</div>
      </div>
      <div class="subtle" style="font-size:0.82rem;align-self:flex-end">
        {{ logs.length }} записів
      </div>
    </div>

    <!-- Filter bar -->
    <div class="panel" style="padding:0.75rem 1rem;margin-bottom:1.25rem">
      <div style="display:grid;grid-template-columns:1fr repeat(4,auto);gap:0.6rem;align-items:center;flex-wrap:wrap">

        <div style="position:relative">
          <span style="position:absolute;left:0.75rem;top:50%;transform:translateY(-50%);opacity:0.45;font-size:0.9rem">🔍</span>
          <input v-model="search" placeholder="Пошук по користувачу, дії, деталях..."
            style="width:100%;padding:0.5rem 0.75rem 0.5rem 2.2rem;box-sizing:border-box;font-size:0.85rem" />
        </div>

        <select v-model="filterUser" style="font-size:0.82rem;padding:0.5rem 0.6rem;min-width:120px">
          <option value="">Усі користувачі</option>
          <option v-for="u in uniqueUsers" :key="u" :value="u">{{ u }}</option>
        </select>

        <select v-model="filterEntity" style="font-size:0.82rem;padding:0.5rem 0.6rem;min-width:130px">
          <option value="">Усі типи</option>
          <option v-for="e in uniqueEntities" :key="e" :value="e">{{ ENTITY_LABELS[e] ?? e }}</option>
        </select>

        <div style="display:flex;align-items:center;gap:0.35rem">
          <span class="subtle" style="font-size:0.75rem;white-space:nowrap">Від</span>
          <input type="date" v-model="filterFrom" style="font-size:0.8rem;padding:0.45rem 0.5rem" />
        </div>
        <div style="display:flex;align-items:center;gap:0.35rem">
          <span class="subtle" style="font-size:0.75rem;white-space:nowrap">До</span>
          <input type="date" v-model="filterTo" style="font-size:0.8rem;padding:0.45rem 0.5rem" />
          <button v-if="hasFilters" class="ghost-button" style="padding:0.45rem 0.6rem;font-size:0.75rem;white-space:nowrap"
            @click="clearFilters">✕ Скинути</button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" style="text-align:center;padding:3rem;color:var(--text-subtle)">
      <div style="font-size:1.5rem;margin-bottom:0.5rem;opacity:0.4">⏳</div>
      Завантаження журналу...
    </div>

    <p v-else-if="error" class="error-text">{{ error }}</p>

    <!-- Empty -->
    <div v-else-if="!logs.length"
      style="text-align:center;padding:4rem;color:var(--text-subtle)">
      <div style="font-size:2.5rem;margin-bottom:0.75rem;opacity:0.35">📋</div>
      <div style="font-size:0.95rem">Записів не знайдено</div>
      <div style="font-size:0.8rem;margin-top:0.35rem">Спробуйте змінити фільтри</div>
    </div>

    <!-- Timeline grouped by date -->
    <div v-else>
      <div v-for="group in grouped" :key="group.dateKey" style="margin-bottom:2rem">

        <!-- Date separator -->
        <div style="display:flex;align-items:center;gap:0.75rem;margin-bottom:1rem;position:sticky;top:0;z-index:2;padding:0.3rem 0">
          <div style="height:1px;flex:1;background:var(--color-border-secondary)"></div>
          <div style="font-size:0.8rem;font-weight:600;color:var(--text-subtle);padding:0.25rem 0.8rem;background:var(--color-background-secondary);border-radius:20px;border:1px solid var(--color-border-secondary);white-space:nowrap">
            {{ group.label }}
          </div>
          <div style="height:1px;flex:1;background:var(--color-border-secondary)"></div>
        </div>

        <!-- Entries -->
        <div style="position:relative;padding-left:2rem">
          <!-- Vertical line -->
          <div style="position:absolute;left:0.65rem;top:0;bottom:0;width:2px;background:var(--color-border-secondary);border-radius:2px"></div>

          <div v-for="(log, idx) in group.entries" :key="log.id"
            style="position:relative;margin-bottom:0.9rem">

            <!-- Dot on timeline -->
            <div style="position:absolute;left:-1.6rem;top:0.55rem;width:10px;height:10px;border-radius:50%;border:2px solid var(--color-background-primary)"
              :style="`background:${actionColor(log.action)}`"></div>

            <!-- Card -->
            <div class="panel" style="padding:0.65rem 0.9rem;display:grid;grid-template-columns:auto 1fr auto;gap:0.5rem 0.75rem;align-items:start">

              <!-- Icon -->
              <div style="font-size:1.2rem;line-height:1;padding-top:0.1rem">{{ actionIcon(log.action) }}</div>

              <!-- Main content -->
              <div>
                <div style="display:flex;align-items:center;gap:0.5rem;flex-wrap:wrap;margin-bottom:0.2rem">
                  <span style="font-weight:600;font-size:0.88rem">{{ humanAction(log.action) }}</span>
                  <span v-if="log.entity" class="chip"
                    style="font-size:0.68rem;padding:0.1rem 0.45rem;opacity:0.75">
                    {{ ENTITY_LABELS[log.entity] ?? log.entity }}
                  </span>
                </div>

                <!-- User badge -->
                <div style="display:flex;align-items:center;gap:0.4rem;margin-bottom:0.25rem">
                  <span style="width:20px;height:20px;border-radius:50%;display:inline-flex;align-items:center;justify-content:center;font-size:0.7rem;font-weight:700;color:#fff;flex-shrink:0"
                    :style="`background:${actionColor(log.action)}`">
                    {{ (log.user || '?')[0].toUpperCase() }}
                  </span>
                  <span style="font-size:0.8rem;font-weight:500">{{ log.user || 'система' }}</span>
                </div>

                <!-- Parsed details as chips -->
                <div v-if="log.details" style="display:flex;flex-wrap:wrap;gap:0.3rem;margin-top:0.15rem">
                  <span v-for="d in parseDetails(log.details)" :key="d.key+d.val"
                    style="font-size:0.72rem;background:var(--color-background-secondary);border:1px solid var(--color-border-secondary);border-radius:4px;padding:0.1rem 0.4rem;color:var(--text-subtle)">
                    <span v-if="d.key" style="opacity:0.7">{{ d.key }}</span>{{ d.val }}
                  </span>
                </div>
              </div>

              <!-- Time -->
              <div style="font-size:0.75rem;color:var(--text-subtle);white-space:nowrap;padding-top:0.15rem">
                {{ fmtTime(log.createdAt) }}
              </div>

            </div>
          </div>
        </div>
      </div>

      <!-- Load more hint -->
      <div v-if="logs.length >= 300"
        style="text-align:center;padding:1rem;color:var(--text-subtle);font-size:0.8rem">
        Показано {{ logs.length }} записів. Використайте фільтр дат для звуження результатів.
      </div>
    </div>

  </div>
</template>
