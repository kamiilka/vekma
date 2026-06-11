<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { api } from "../api";
import type { UserSession } from "../types";

const props = defineProps<{ session: UserSession }>();
const emit = defineEmits<{ (e: "session-expired"): void }>();

const token = computed(() => props.session.token);

// --- Tabs ---
const activeTab = ref<"send" | "settings">("send");

// --- Send tab ---
const channel = ref<"email" | "sms" | "telegram" | "viber">("email");
const composeSender = ref("");
const composeRecipient = ref("");
const composeSubject = ref("");
const composeBody = ref("");
const sending = ref(false);
const sendResult = ref<{ ok: boolean; message: string } | null>(null);

const channelConfig: Record<string, {
  label: string;
  senderLabel: string;
  senderPlaceholder: string;
  recipientLabel: string;
  recipientPlaceholder: string;
  hasSubject: boolean;
  senderNote?: string;
}> = {
  email: {
    label: "Email",
    senderLabel: "Від (адреса відправника)",
    senderPlaceholder: "from@example.com",
    recipientLabel: "Кому (email адреса)",
    recipientPlaceholder: "to@example.com",
    hasSubject: true,
  },
  sms: {
    label: "SMS",
    senderLabel: "Ім'я або номер відправника",
    senderPlaceholder: "MyApp або +380XXXXXXXXX",
    recipientLabel: "Номер телефону отримувача",
    recipientPlaceholder: "+380XXXXXXXXX",
    hasSubject: false,
  },
  telegram: {
    label: "Telegram",
    senderLabel: "Bot Token (необов'язково)",
    senderPlaceholder: "Залиште порожнім — використається збережений токен",
    recipientLabel: "Chat ID отримувача",
    recipientPlaceholder: "-1001234567890",
    hasSubject: false,
    senderNote: "Chat ID знайдіть через @userinfobot або @getidsbot у Telegram",
  },
  viber: {
    label: "Viber",
    senderLabel: "Bot Token (необов'язково)",
    senderPlaceholder: "Залиште порожнім — використається збережений токен",
    recipientLabel: "Viber User ID отримувача",
    recipientPlaceholder: "Числовий Viber User ID",
    hasSubject: false,
    senderNote: "Viber User ID — це числовий ідентифікатор, НЕ номер телефону",
  },
};

const cfg = computed(() => channelConfig[channel.value]);

function resetResult() { sendResult.value = null; }

async function sendMessage() {
  if (!composeRecipient.value.trim() || !composeBody.value.trim()) return;
  sending.value = true;
  sendResult.value = null;
  try {
    await api.sendQuickMessage(token.value, {
      channel: channel.value,
      sender: composeSender.value.trim() || undefined,
      recipient: composeRecipient.value.trim(),
      subject: composeSubject.value.trim() || undefined,
      body: composeBody.value.trim(),
    });
    sendResult.value = { ok: true, message: "Повідомлення надіслано!" };
    composeRecipient.value = "";
    composeBody.value = "";
    composeSubject.value = "";
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    sendResult.value = { ok: false, message: e.message || "Помилка надсилання" };
  } finally {
    sending.value = false;
  }
}

// --- Settings tab ---
const settingsLoading = ref(false);
const settingsSaving = ref(false);
const settingsResult = ref<{ ok: boolean; message: string } | null>(null);
const settingsTab = ref<"email" | "telegram" | "sms" | "viber">("email");

const smtpAddr = ref("");
const smtpUser = ref("");
const smtpPass = ref("");
const smtpFrom = ref("");
const telegramToken = ref("");
const telegramChatId = ref("");
const smsGatewayUrl = ref("");
const smsGatewayToken = ref("");
const smsPhoneTo = ref("");
const viberToken = ref("");
const viberRecipient = ref("");

async function loadSettings() {
  settingsLoading.value = true;
  try {
    const c = await api.getNotificationConfig(token.value);
    smtpAddr.value = c.smtpAddr ?? "";
    smtpUser.value = c.smtpUser ?? "";
    smtpPass.value = c.smtpPass ?? "";
    smtpFrom.value = c.smtpFrom ?? "";
    telegramToken.value = c.telegramToken ?? "";
    telegramChatId.value = c.telegramChatId ?? "";
    smsGatewayUrl.value = c.smsGatewayUrl ?? "";
    smsGatewayToken.value = c.smsGatewayToken ?? "";
    smsPhoneTo.value = c.smsPhoneTo ?? "";
    viberToken.value = c.viberToken ?? "";
    viberRecipient.value = c.viberRecipient ?? "";
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") emit("session-expired");
  } finally {
    settingsLoading.value = false;
  }
}

async function saveSettings() {
  settingsSaving.value = true;
  settingsResult.value = null;
  try {
    await api.saveNotificationConfig(token.value, {
      smtpAddr: smtpAddr.value,
      smtpUser: smtpUser.value,
      smtpPass: smtpPass.value,
      smtpFrom: smtpFrom.value,
      telegramToken: telegramToken.value,
      telegramChatId: telegramChatId.value,
      smsGatewayUrl: smsGatewayUrl.value,
      smsGatewayToken: smsGatewayToken.value,
      smsPhoneTo: smsPhoneTo.value,
      viberToken: viberToken.value,
      viberRecipient: viberRecipient.value,
    });
    settingsResult.value = { ok: true, message: "Налаштування збережено" };
  } catch (e: any) {
    if (e.message === "SESSION_EXPIRED") { emit("session-expired"); return; }
    settingsResult.value = { ok: false, message: e.message || "Помилка збереження" };
  } finally {
    settingsSaving.value = false;
  }
}

onMounted(() => { loadSettings(); });
</script>

<template>
  <div class="page-content">
    <div class="page-header" style="margin-bottom:1.5rem">
      <h2 style="margin:0">Повідомлення</h2>
    </div>

    <!-- Tab switcher -->
    <div style="display:flex;gap:0.4rem;margin-bottom:1.5rem">
      <button :class="['ghost-button', activeTab==='send' ? 'active' : '']" @click="activeTab='send'">
        Надіслати повідомлення
      </button>
      <button :class="['ghost-button', activeTab==='settings' ? 'active' : '']" @click="activeTab='settings'">
        Налаштування каналів
      </button>
    </div>

    <!-- SEND TAB -->
    <div v-if="activeTab==='send'" style="max-width:560px">
      <p v-if="sendResult" :class="sendResult.ok ? 'ok-text' : 'error-text'" style="margin-bottom:1rem">
        {{ sendResult.message }}
      </p>
      <div class="panel" style="padding:1.5rem">
        <div style="margin-bottom:1.2rem">
          <div style="font-size:0.82rem;font-weight:600;color:#7ab99a;text-transform:uppercase;letter-spacing:0.05em;margin-bottom:0.6rem">
            Канал
          </div>
          <div style="display:grid;grid-template-columns:1fr 1fr 1fr 1fr;gap:0.4rem">
            <button v-for="ch in ['email','sms','telegram','viber']" :key="ch"
              :class="['ghost-button', channel === ch ? 'active' : '']"
              style="padding:0.5rem;font-size:0.85rem;text-align:center"
              @click="channel = ch as any; resetResult()">
              {{ channelConfig[ch].label }}
            </button>
          </div>
        </div>

        <div class="grid">
          <label>{{ cfg.senderLabel }}
            <input v-model="composeSender" :placeholder="cfg.senderPlaceholder" @input="resetResult">
          </label>
          <label>{{ cfg.recipientLabel }} *
            <input v-model="composeRecipient" :placeholder="cfg.recipientPlaceholder" @input="resetResult">
          </label>
          <div v-if="cfg.senderNote" style="font-size:0.8rem;color:#e0a060;background:rgba(224,160,96,0.1);border-radius:6px;padding:0.5rem 0.7rem;margin-top:-0.4rem">
            {{ cfg.senderNote }}
          </div>
          <label v-if="cfg.hasSubject">Тема листа
            <input v-model="composeSubject" placeholder="Тема листа..." @input="resetResult">
          </label>
          <label>Текст повідомлення *
            <textarea v-model="composeBody" rows="6" placeholder="Введіть текст повідомлення..." @input="resetResult"></textarea>
          </label>
        </div>

        <div style="display:flex;gap:0.5rem;margin-top:1.2rem;align-items:center">
          <button @click="sendMessage" :disabled="sending || !composeRecipient.trim() || !composeBody.trim()">
            {{ sending ? 'Надсилання...' : 'Надіслати' }}
          </button>
          <span v-if="sendResult" :class="sendResult.ok ? 'ok-text' : 'error-text'" style="font-size:0.88rem">
            {{ sendResult.message }}
          </span>
        </div>
      </div>
    </div>

    <!-- SETTINGS TAB -->
    <div v-if="activeTab==='settings'" style="max-width:600px">
      <p v-if="settingsLoading" class="subtle">Завантаження...</p>
      <div v-else>
        <p v-if="settingsResult" :class="settingsResult.ok ? 'ok-text' : 'error-text'" style="margin-bottom:1rem">
          {{ settingsResult.message }}
        </p>
        <!-- Channel sub-tabs -->
        <div style="display:flex;gap:0.4rem;margin-bottom:1rem">
          <button v-for="ch in ['email','telegram','sms','viber']" :key="ch"
            :class="['ghost-button', settingsTab===ch ? 'active' : '']"
            style="padding:0.4rem 0.8rem;font-size:0.85rem"
            @click="settingsTab = ch as any">
            {{ channelConfig[ch].label }}
          </button>
        </div>

        <div class="panel" style="padding:1.5rem">
          <!-- Email -->
          <div v-if="settingsTab==='email'" class="grid">
            <label>SMTP-сервер (адреса:порт)
              <input v-model="smtpAddr" placeholder="smtp.gmail.com:587">
            </label>
            <label>Логін
              <input v-model="smtpUser" placeholder="your@gmail.com">
            </label>
            <label>Пароль / токен застосунку
              <input v-model="smtpPass" type="password" placeholder="••••••••">
            </label>
            <label>Адреса відправника
              <input v-model="smtpFrom" placeholder="your@gmail.com">
            </label>
          </div>

          <!-- Telegram -->
          <div v-if="settingsTab==='telegram'" class="grid">
            <label>Bot Token
              <input v-model="telegramToken" type="password" placeholder="••••••••">
            </label>
            <label>Chat ID отримувача за замовчуванням
              <input v-model="telegramChatId" placeholder="-1001234567890">
            </label>
            <div style="font-size:0.82rem;color:#6a9e84">
              Chat ID знайдіть через @userinfobot або @getidsbot у Telegram.
            </div>
          </div>

          <!-- SMS -->
          <div v-if="settingsTab==='sms'" class="grid">
            <label>URL шлюзу
              <input v-model="smsGatewayUrl" placeholder="https://api.turbosms.ua/message/send.json">
            </label>
            <label>Токен авторизації
              <input v-model="smsGatewayToken" type="password" placeholder="••••••••">
            </label>
            <label>Номер отримувача за замовчуванням
              <input v-model="smsPhoneTo" placeholder="+380XXXXXXXXX">
            </label>
          </div>

          <!-- Viber -->
          <div v-if="settingsTab==='viber'" class="grid">
            <label>Bot Token
              <input v-model="viberToken" type="password" placeholder="••••••••">
            </label>
            <label>Viber User ID за замовчуванням
              <input v-model="viberRecipient" placeholder="Числовий ID">
            </label>
            <div style="font-size:0.82rem;color:#6a9e84">
              Viber User ID — це числовий ідентифікатор, НЕ номер телефону.
            </div>
          </div>

          <div style="margin-top:1.5rem">
            <button @click="saveSettings" :disabled="settingsSaving">
              {{ settingsSaving ? 'Збереження...' : 'Зберегти налаштування' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
