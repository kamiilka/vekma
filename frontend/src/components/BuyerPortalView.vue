<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import type { UserSession, CustomerOrder, DebtSummary, Payment } from "../types";

const props = defineProps<{ session: UserSession }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);

const orders = ref<CustomerOrder[]>([]);
const debts = ref<DebtSummary[]>([]);
const payments = ref<Payment[]>([]);
const loading = ref(false);
const error = ref("");
const selected = ref<CustomerOrder | null>(null);

const statusLabel: Record<string, string> = {
  draft: "Чернетка",
  confirmed: "Підтверджено",
  processing: "Готується",
  paid: "Оплачено",
  completed: "Завершено",
  cancelled: "Скасовано",
};

const statusColor: Record<string, string> = {
  draft: "#7ab99a",
  confirmed: "#fad07a",
  processing: "#9fe8c4",
  paid: "#6dd4a0",
  completed: "#6dd4a0",
  cancelled: "#ff9ca0",
};

async function load() {
  loading.value = true; error.value = "";
  try {
    [orders.value, debts.value] = await Promise.all([
      api.orders(token.value),
      api.debts(token.value),
    ]);
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    error.value = e.message;
  } finally { loading.value = false; }
}

async function loadPayments(orderId: number) {
  try {
    payments.value = await api.paymentsForOrder(token.value, orderId);
  } catch { payments.value = []; }
}

function selectOrder(o: CustomerOrder) {
  selected.value = o;
  loadPayments(o.id);
}

function debtForOrder(orderId: number): DebtSummary | undefined {
  return debts.value.find(d => d.entityType === "order" && d.entityId === orderId);
}

const openOrders = computed(() => orders.value.filter(o => o.status !== "completed" && o.status !== "cancelled"));
const closedOrders = computed(() => orders.value.filter(o => o.status === "completed" || o.status === "cancelled"));

onMounted(load);
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1.5rem">
      <div>
        <h2 style="margin:0">Мої замовлення</h2>
        <p class="subtle" style="margin:0.2rem 0 0">Особистий кабінет покупця</p>
      </div>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="loading" class="subtle">Завантаження...</p>

    <div style="display:grid;grid-template-columns:300px 1fr;gap:1rem;align-items:start">
      <!-- Order list -->
      <div>
        <div v-if="openOrders.length">
          <div style="font-size:0.8rem;font-weight:600;color:#7ab99a;margin-bottom:0.5rem;text-transform:uppercase;letter-spacing:0.05em">Активні</div>
          <div v-for="o in openOrders" :key="o.id"
            class="panel" style="padding:0.8rem;cursor:pointer;margin-bottom:0.5rem"
            :style="selected?.id===o.id?'border-color:rgba(72,187,120,0.6)':''"
            @click="selectOrder(o)">
            <div style="display:flex;justify-content:space-between;align-items:center;gap:0.5rem">
              <span style="font-weight:600;font-size:0.9rem">Замовлення #{{ o.id }}</span>
              <span class="chip" :style="`background:${statusColor[o.status]}22;color:${statusColor[o.status]};font-size:0.75rem;padding:0.2rem 0.5rem`">
                {{ statusLabel[o.status] ?? o.status }}
              </span>
            </div>
            <div class="subtle" style="font-size:0.82rem;margin-top:0.3rem">
              {{ new Date(o.createdAt).toLocaleDateString('uk') }}
              <span v-if="o.dueDate"> · до {{ new Date(o.dueDate).toLocaleDateString('uk') }}</span>
            </div>
            <div style="margin-top:0.4rem;font-size:0.9rem">
              Сума: <strong>{{ o.total }} {{ o.currency }}</strong>
            </div>
            <div v-if="debtForOrder(o.id)" style="font-size:0.82rem;margin-top:0.2rem">
              Залишок до оплати:
              <span :style="`color:${(debtForOrder(o.id)?.debt??0)>0?'#ff9ca0':'#6dd4a0'};font-weight:700`">
                {{ debtForOrder(o.id)?.debt ?? 0 }} {{ o.currency }}
              </span>
            </div>
          </div>
        </div>

        <div v-if="closedOrders.length" style="margin-top:1rem">
          <div style="font-size:0.8rem;font-weight:600;color:#7ab99a;margin-bottom:0.5rem;text-transform:uppercase;letter-spacing:0.05em">Завершені</div>
          <div v-for="o in closedOrders" :key="o.id"
            class="panel" style="padding:0.7rem;cursor:pointer;margin-bottom:0.4rem;opacity:0.7"
            :style="selected?.id===o.id?'border-color:rgba(72,187,120,0.6)':''"
            @click="selectOrder(o)">
            <div style="display:flex;justify-content:space-between">
              <span style="font-size:0.88rem">#{{ o.id }}</span>
              <span class="subtle" style="font-size:0.78rem">{{ statusLabel[o.status] }}</span>
            </div>
            <div class="subtle" style="font-size:0.78rem">{{ new Date(o.createdAt).toLocaleDateString('uk') }}</div>
          </div>
        </div>

        <div v-if="!orders.length && !loading" class="subtle" style="text-align:center;padding:2rem 0">
          У вас ще немає замовлень
        </div>
      </div>

      <!-- Order detail -->
      <div v-if="selected">
        <div class="panel" style="padding:1.2rem">
          <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:0.5rem;margin-bottom:1rem">
            <h3 style="margin:0">Замовлення #{{ selected.id }}</h3>
            <span class="chip" :style="`background:${statusColor[selected.status]}22;color:${statusColor[selected.status]};padding:0.3rem 0.8rem;font-size:0.88rem`">
              {{ statusLabel[selected.status] ?? selected.status }}
            </span>
          </div>

          <!-- Status progress bar -->
          <div style="display:flex;gap:0;margin-bottom:1.2rem;border-radius:6px;overflow:hidden">
            <div v-for="(s, i) in ['draft','confirmed','processing','paid','completed']" :key="s"
              style="flex:1;padding:0.4rem 0;text-align:center;font-size:0.72rem;font-weight:600;transition:background 0.2s"
              :style="`background:${ ['draft','confirmed','processing','paid','completed'].indexOf(selected.status) >= i ? statusColor[s]+'44' : 'rgba(186,192,230,0.06)' }; color:${ ['draft','confirmed','processing','paid','completed'].indexOf(selected.status) >= i ? statusColor[s] : '#5a6080' }`">
              {{ statusLabel[s] }}
            </div>
          </div>

          <div style="display:grid;grid-template-columns:1fr 1fr;gap:0.6rem;font-size:0.9rem;margin-bottom:1rem">
            <div><span class="subtle">Дата:</span> {{ new Date(selected.createdAt).toLocaleString('uk') }}</div>
            <div v-if="selected.dueDate"><span class="subtle">Термін:</span> {{ new Date(selected.dueDate).toLocaleDateString('uk') }}</div>
            <div><span class="subtle">Сума:</span> <strong>{{ selected.total }} {{ selected.currency }}</strong></div>
            <div v-if="debtForOrder(selected.id)">
              <span class="subtle">До оплати:</span>
              <span :style="`font-weight:700;color:${(debtForOrder(selected.id)?.debt??0)>0?'#ff9ca0':'#6dd4a0'}`">
                {{ debtForOrder(selected.id)?.debt ?? 0 }} {{ selected.currency }}
              </span>
            </div>
          </div>

          <!-- Items -->
          <div v-if="selected.items?.length" style="margin-bottom:1rem">
            <div style="font-size:0.85rem;color:#7ab99a;margin-bottom:0.4rem;font-weight:600">Позиції замовлення</div>
            <table style="font-size:0.88rem">
              <thead><tr><th>Товар ID</th><th>К-сть</th><th>Ціна</th><th>Сума</th></tr></thead>
              <tbody>
                <tr v-for="item in selected.items" :key="item.productId">
                  <td>#{{ item.productId }}</td>
                  <td>{{ item.quantity }}</td>
                  <td>{{ item.price }}</td>
                  <td>{{ (item.quantity * item.price).toFixed(2) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Payments -->
          <div>
            <div style="font-size:0.85rem;color:#7ab99a;margin-bottom:0.4rem;font-weight:600">Оплати</div>
            <div v-if="!payments.length" class="subtle" style="font-size:0.85rem">Оплат ще немає</div>
            <div v-for="p in payments" :key="p.id"
              style="display:flex;justify-content:space-between;align-items:center;padding:0.4rem 0;border-bottom:0.5px solid rgba(186,192,230,0.1);font-size:0.88rem">
              <div>
                <span style="color:#6dd4a0;font-weight:600">+{{ p.amount }} {{ p.currency }}</span>
                <span class="subtle" style="margin-left:0.5rem">{{ p.method }}</span>
              </div>
              <span class="subtle">{{ new Date(p.createdAt).toLocaleString('uk') }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="panel" style="padding:2rem;text-align:center">
        <p class="subtle">Оберіть замовлення зі списку, щоб побачити деталі</p>
      </div>
    </div>
  </div>
</template>
