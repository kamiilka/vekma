<script setup lang="ts">
import type { OrderChainNode } from "../types";

const props = defineProps<{
  node: OrderChainNode;
  depth?: number;
  emitNavigate?: (payload: { type: string; id: number }) => void;
}>();

const depth = props.depth ?? 0;

const typeIcon: Record<string, string> = {
  customer_order:  "🛍",
  supplier_order:  "📋",
  purchase:        "📦",
  sale:            "🧾",
  payment:         "💳",
  supplier_payment:"💸",
};

const typeColor: Record<string, string> = {
  customer_order:  "#7ab99a",
  supplier_order:  "#60c0e0",
  purchase:        "#a78bfa",
  sale:            "#f59e0b",
  payment:         "#6dd4a0",
  supplier_payment:"#e0a060",
};

const statusColor: Record<string, string> = {
  new: "#94a3b8", in_work: "#60a5fa", ordered: "#a78bfa",
  expected: "#f59e0b", arrived: "#34d399", issued: "#10b981",
  closed: "#6b7280", cancelled: "#f87171",
  draft: "#94a3b8", sent: "#60a5fa", confirmed: "#818cf8",
  in_transit: "#f59e0b", received: "#34d399",
  paid: "#4ecb8d", completed: "#3cb87a", processing: "#e0c060",
};

const statusLabel: Record<string, string> = {
  new: "Нове", in_work: "В роботі", ordered: "Замовлено",
  expected: "Очікується", arrived: "Надійшло", issued: "Видано",
  closed: "Закрито", cancelled: "Скасовано",
  draft: "Чернетка", sent: "Відправлено", confirmed: "Підтверджено",
  in_transit: "В дорозі", received: "Отримано",
  paid: "Оплачено", completed: "Завершено", processing: "В обробці",
};

function navigate() {
  if (props.emitNavigate) {
    props.emitNavigate({ type: props.node.type, id: props.node.id });
  }
}
</script>

<template>
  <div :style="`padding-left:${depth * 16}px`">
    <!-- Node row -->
    <div
      :style="`
        display:flex;align-items:flex-start;gap:0.5rem;
        padding:0.4rem 0.6rem;
        margin-bottom:0.3rem;
        background:${typeColor[node.type]}10;
        border-left:2px solid ${typeColor[node.type] ?? '#666'};
        border-radius:0 6px 6px 0;
        cursor:pointer;
        font-size:0.82rem;
      `"
      @click="navigate"
    >
      <span style="font-size:1rem;flex-shrink:0">{{ typeIcon[node.type] ?? '📄' }}</span>
      <div style="flex:1;min-width:0">
        <div style="display:flex;align-items:center;gap:0.4rem;flex-wrap:wrap">
          <span style="font-weight:600" :style="`color:${typeColor[node.type] ?? 'var(--text)'}`">
            {{ node.label }}
          </span>
          <span v-if="node.status"
            style="font-size:0.7rem;padding:0.1rem 0.4rem;border-radius:8px;font-weight:600"
            :style="`background:${statusColor[node.status]??'#64748b'}22;color:${statusColor[node.status]??'#94a3b8'}`">
            {{ statusLabel[node.status] ?? node.status }}
          </span>
        </div>
        <div style="display:flex;gap:0.6rem;margin-top:0.15rem;color:var(--text-muted);font-size:0.75rem;flex-wrap:wrap">
          <span v-if="node.amount != null">{{ node.amount.toFixed(2) }} {{ node.currency ?? '' }}</span>
          <span>{{ new Date(node.date).toLocaleDateString('uk') }}</span>
        </div>
      </div>
      <span style="font-size:0.7rem;color:var(--text-muted);align-self:center;flex-shrink:0">→</span>
    </div>

    <!-- Children recursively -->
    <template v-if="node.children?.length">
      <!-- connector line -->
      <div :style="`margin-left:${depth * 16 + 8}px;padding-left:8px;border-left:1px dashed rgba(255,255,255,0.1)`">
        <OrderChainTree
          v-for="child in node.children"
          :key="`${child.type}-${child.id}`"
          :node="child"
          :depth="0"
          :emit-navigate="emitNavigate"
        />
      </div>
    </template>
  </div>
</template>
