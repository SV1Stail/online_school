# Frontend

Vue 3 / Vite frontend онлайн-школы.

## Локальный запуск

```sh
cp .env.example .env
npm ci
npm run dev
```

Vite проксирует `/api` и `/health` на backend по адресу `127.0.0.1:8080`.

## Сборка

```sh
npm run build
```

Результат записывается в `dist/`. Значения `VITE_TELEGRAM_URL` и
`VITE_WHATSAPP_URL` задают ссылки кнопок на публичной странице.
