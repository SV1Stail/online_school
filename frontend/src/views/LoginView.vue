<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import { api } from '../api.js'

const router = useRouter()
const login = ref('')
const password = ref('')
const currentPassword = ref('')
const newPassword = ref('')
const loading = ref(false)
const error = ref('')
const mustChange = ref(false)
const changed = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const user = await api.login(login.value.trim(), password.value)
    if (user.mustChangePassword) {
      currentPassword.value = password.value
      mustChange.value = true
    } else {
      router.push('/app')
    }
  } catch (err) {
    error.value = err.code === 'invalid_credentials'
      ? 'Проверьте логин и пароль'
      : 'Не удалось войти. Попробуйте ещё раз.'
  } finally {
    loading.value = false
  }
}

async function changePassword() {
  error.value = ''
  loading.value = true
  try {
    await api.changePassword(currentPassword.value, newPassword.value)
    changed.value = true
    mustChange.value = false
    password.value = ''
  } catch {
    error.value = 'Пароль должен быть надёжным и отличаться от временного.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <RouterLink class="brand login-brand" to="/">
      <span class="brand-mark">A</span>
      <span><strong>Александра</strong><small>математика · физика</small></span>
    </RouterLink>

    <section class="login-card">
      <div class="login-decoration">x²</div>
      <div class="round-icon login-icon"><AppIcon name="user" :size="30" /></div>
      <template v-if="!mustChange">
        <p class="section-kicker">Личный кабинет</p>
        <h1>{{ changed ? 'Пароль изменён' : 'Рады видеть вас' }}</h1>
        <p>{{ changed ? 'Войдите ещё раз с новым паролем.' : 'Расписание, занятия и ссылки на конференции — в одном месте.' }}</p>
        <form class="login-form" @submit.prevent="submit">
          <label>Логин<input v-model="login" required autocomplete="username" placeholder="Введите логин" /></label>
          <label>Пароль<input v-model="password" required type="password" autocomplete="current-password" placeholder="Введите пароль" /></label>
          <p v-if="error" class="form-error">{{ error }}</p>
          <button class="button button--ink button--full" :disabled="loading" type="submit">
            {{ loading ? 'Входим…' : 'Войти' }} <AppIcon name="arrow" :size="18" />
          </button>
        </form>
      </template>
      <template v-else>
        <p class="section-kicker">Первый вход</p>
        <h1>Придумайте новый пароль</h1>
        <p>Временный пароль нужно заменить перед началом работы.</p>
        <form class="login-form" @submit.prevent="changePassword">
          <label>Новый пароль<input v-model="newPassword" required minlength="10" type="password" autocomplete="new-password" placeholder="Не менее 10 символов" /></label>
          <p v-if="error" class="form-error">{{ error }}</p>
          <button class="button button--ink button--full" :disabled="loading" type="submit">Сохранить пароль</button>
        </form>
      </template>
    </section>
    <p class="hand-note login-note">Всё нужное рядом ♡</p>
  </main>
</template>
