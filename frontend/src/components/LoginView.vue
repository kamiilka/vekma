<script setup lang="ts">
import { ref } from "vue";
import { api } from "../api";
import type { UserSession } from "../types";

const emit = defineEmits<{
  authenticated: [session: UserSession];
}>();

const username = ref("admin");
const password = ref("admin123");
const errorText = ref("");
const isLoading = ref(false);

async function onLogin() {
  try {
    isLoading.value = true;
    errorText.value = "";
    const session = await api.login(username.value, password.value);
    emit("authenticated", session);
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : "Помилка входу";
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="login-page">
    <section class="login-card">
      <div style="text-align:center;margin-bottom:1.5rem">
        <div style="font-size:1.8rem;font-weight:800;color:var(--accent);letter-spacing:-0.02em">VEKMA</div>
        <div style="font-size:0.82rem;color:var(--text-subtle);margin-top:0.2rem">automate. manage. grow.</div>
      </div>
      <h1 style="font-size:1.15rem;margin-bottom:1.2rem">Вхід до системи</h1>
      <form class="grid" @submit.prevent="onLogin">
        <label>
          Логін
          <input v-model="username" type="text" placeholder="Введіть логін" />
        </label>
        <label>
          Пароль
          <input v-model="password" type="password" placeholder="Введіть пароль" />
        </label>
        <button type="submit" :disabled="isLoading">
          {{ isLoading ? "Вхід..." : "Увійти" }}
        </button>
      </form>
      <p v-if="errorText" class="error-text">{{ errorText }}</p>
    </section>
  </div>
</template>
