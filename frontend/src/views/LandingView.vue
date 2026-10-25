<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import AppIcon from '../components/AppIcon.vue'

const menuOpen = ref(false)
const scrolled = ref(false)
const telegramUrl = import.meta.env.VITE_TELEGRAM_URL || 'https://t.me/ms_repeat'
const whatsappUrl = import.meta.env.VITE_WHATSAPP_URL || 'https://wa.me/79052438437'

const updateHeader = () => { scrolled.value = window.scrollY > 24 }
onMounted(() => window.addEventListener('scroll', updateHeader, { passive: true }))
onBeforeUnmount(() => window.removeEventListener('scroll', updateHeader))

const benefits = [
  { icon: 'book', title: 'Понятная теория', text: 'Сложные темы разбираем простым языком, через логику и наглядные примеры.' },
  { icon: 'user', title: 'Свой темп', text: 'Программа подстраивается под уровень, цели и темп конкретного ученика.' },
  { icon: 'chart', title: 'До результата', text: 'Учимся видеть ход решения, грамотно оформлять ответ и уверенно применять знания.' },
]

const process = [
  ['01', 'Знакомимся', 'Определяем текущий уровень, цель и комфортный темп обучения.'],
  ['02', 'Строим маршрут', 'Собираем персональный план без хаоса и случайных заданий.'],
  ['03', 'Разбираемся', 'На занятии не переписываем теорию, а понимаем логику и тренируемся.'],
  ['04', 'Закрепляем', 'Отслеживаем прогресс и возвращаемся к теме, если остались вопросы.'],
]

const closeMenu = () => { menuOpen.value = false }
</script>

<template>
  <div class="landing-shell">
    <header class="site-header" :class="{ 'site-header--scrolled': scrolled }">
      <RouterLink class="brand" to="/" aria-label="На главную">
        <span class="brand-mark">A</span>
        <span><strong>Александра</strong><small>математика · физика</small></span>
      </RouterLink>
      <nav class="desktop-nav" aria-label="Основная навигация">
        <a href="#approach">Подход</a>
        <a href="#materials">Материалы</a>
        <a href="#process">Как проходят занятия</a>
      </nav>
      <RouterLink class="button button--small button--ink desktop-login" to="/login">
        Личный кабинет <AppIcon name="arrow" :size="17" />
      </RouterLink>
      <button class="icon-button mobile-menu-button" type="button" aria-label="Открыть меню" @click="menuOpen = !menuOpen">
        <AppIcon :name="menuOpen ? 'close' : 'menu'" />
      </button>
    </header>

    <div v-if="menuOpen" class="mobile-menu">
      <a href="#approach" @click="closeMenu">Подход</a>
      <a href="#materials" @click="closeMenu">Материалы</a>
      <a href="#process" @click="closeMenu">Как проходят занятия</a>
      <RouterLink to="/login" @click="closeMenu">Личный кабинет</RouterLink>
    </div>

    <main>
      <section class="hero section-wrap">
        <div class="hero-copy">
          <div class="eyebrow"><span></span> Онлайн-занятия · 9–11 классы</div>
          <h1>Математика<br /><em>без хаоса</em></h1>
          <p class="hero-lead">Понятно, спокойно и по шагам. Готовимся к ОГЭ и ЕГЭ так, чтобы видеть не просто ответ, а весь ход решения.</p>
          <div class="hero-actions">
            <a class="button button--ink" :href="telegramUrl" target="_blank" rel="noreferrer">
              Написать в Telegram <AppIcon name="arrow" :size="18" />
            </a>
            <a class="button button--ghost" :href="whatsappUrl" target="_blank" rel="noreferrer">WhatsApp</a>
          </div>
          <p class="hand-note hero-note">Знания с тобой дальше ♡</p>
        </div>

        <div class="hero-visual" aria-label="Александра — репетитор по математике и физике">
          <div class="blue-blob blue-blob--one"></div>
          <div class="blue-blob blue-blob--two"></div>
          <div class="hero-photo-frame">
            <img src="/images/story-about.png" alt="Александра, репетитор по математике и физике" />
          </div>
          <div class="formula-card formula-card--top">x² + y² = r²</div>
          <div class="formula-card formula-card--bottom">F = ma</div>
          <div class="paper-plane" aria-hidden="true">⌁</div>
        </div>
      </section>

      <section id="approach" class="benefits section-wrap">
        <div class="section-heading section-heading--center">
          <p class="hand-note">Больше, чем просто объяснения</p>
          <h2>Спокойная подготовка<br />с опорой на понимание</h2>
        </div>
        <div class="benefit-grid">
          <article v-for="item in benefits" :key="item.title" class="soft-card benefit-card">
            <div class="round-icon"><AppIcon :name="item.icon" :size="30" /></div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.text }}</p>
          </article>
        </div>
      </section>

      <section id="materials" class="materials-section">
        <div class="section-wrap materials-inner">
          <div class="materials-copy">
            <p class="section-kicker">Авторские материалы</p>
            <h2>Теория, которую<br /><em>удобно повторять</em></h2>
            <p>Конспекты, формулы, схемы и примеры собраны аккуратно и наглядно. Ученик может сосредоточиться на логике темы, а не на переписывании.</p>
            <ul class="check-list">
              <li><span><AppIcon name="check" :size="17" /></span> Математика и физика без лишней воды</li>
              <li><span><AppIcon name="check" :size="17" /></span> Пошаговые объяснения и рисунки</li>
              <li><span><AppIcon name="check" :size="17" /></span> Материалы остаются у ученика</li>
            </ul>
          </div>
          <div class="story-stack" aria-label="Примеры учебных материалов">
            <figure class="story-card story-card--left"><img loading="lazy" src="/images/story-notes-math.png" alt="Конспект по производной" /></figure>
            <figure class="story-card story-card--center"><img loading="lazy" src="/images/story-notes-geometry.png" alt="Конспект по геометрии" /></figure>
            <figure class="story-card story-card--right"><img loading="lazy" src="/images/story-notes-physics.png" alt="Конспект по физике" /></figure>
          </div>
        </div>
      </section>

      <section id="process" class="process section-wrap">
        <div class="section-heading">
          <p class="hand-note">Наглядно. По шагам. Без стресса.</p>
          <h2>Как строится обучение</h2>
        </div>
        <div class="process-grid">
          <article v-for="item in process" :key="item[0]" class="process-item">
            <span class="process-number">{{ item[0] }}</span>
            <div><h3>{{ item[1] }}</h3><p>{{ item[2] }}</p></div>
          </article>
        </div>
      </section>

      <section class="cta-section section-wrap" id="contacts">
        <div class="cta-card">
          <div>
            <p class="hand-note">Готовы начать?</p>
            <h2>Давайте познакомимся<br />на пробном занятии</h2>
          </div>
          <div class="cta-actions">
            <a class="button button--white" :href="telegramUrl" target="_blank" rel="noreferrer">Telegram <AppIcon name="external" :size="18" /></a>
            <a class="button button--outline-white" :href="whatsappUrl" target="_blank" rel="noreferrer">WhatsApp</a>
          </div>
          <span class="cta-formula">y = kx + b</span>
        </div>
      </section>
    </main>

    <footer class="site-footer section-wrap">
      <div class="brand brand--footer"><span class="brand-mark">A</span><span><strong>Александра</strong><small>математика · физика</small></span></div>
      <p>Понятно · спокойно · результативно</p>
      <RouterLink to="/login">Войти в кабинет <AppIcon name="arrow" :size="16" /></RouterLink>
    </footer>
  </div>
</template>
