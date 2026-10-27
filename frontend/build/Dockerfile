FROM node:24-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_TELEGRAM_URL=https://t.me/ms_repeat
ARG VITE_WHATSAPP_URL=https://wa.me/79052438437
ENV VITE_TELEGRAM_URL=${VITE_TELEGRAM_URL}
ENV VITE_WHATSAPP_URL=${VITE_WHATSAPP_URL}
RUN npm run build

FROM caddy:2-alpine
COPY build/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv
EXPOSE 80 443 443/udp
