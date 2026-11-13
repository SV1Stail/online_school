SHELL := /bin/sh
.DEFAULT_GOAL := help
export COMPOSE_PARALLEL_LIMIT ?= 1

LOCAL_ENV ?= build/.env.local
PROD_ENV ?= build/.env.prod

COMPOSE_BASE := -f build/docker-compose.yml
COMPOSE_LOCAL := $(COMPOSE_BASE) -f build/docker-compose.local.yml
COMPOSE_PROD := $(COMPOSE_BASE) -f build/docker-compose.prod.yml

LOCAL_DC := docker compose --env-file $(LOCAL_ENV) $(COMPOSE_LOCAL)
PROD_DC := docker compose --env-file $(PROD_ENV) $(COMPOSE_PROD)

.PHONY: help check-tools \
	local-init local-config local-build local-up local-down local-restart local-logs local-ps local-bootstrap \
	prod-init prod-check prod-config prod-build prod-up prod-deploy prod-down prod-restart prod-logs prod-ps prod-bootstrap

help: ## Показать доступные команды
	@printf '%s\n' \
		'Локальная разработка:' \
		'  make local-up          Собрать и запустить приложение' \
		'  make local-down        Остановить приложение без удаления данных' \
		'  make local-restart     Перезапустить сервисы' \
		'  make local-logs        Следить за логами' \
		'  make local-ps          Показать состояние сервисов' \
		'  make local-bootstrap   Создать первую учетную запись модератора' \
		'  make local-config      Проверить итоговую Compose-конфигурацию' \
		'' \
		'Production:' \
		'  make prod-init         Создать build/.env.prod из примера' \
		'  make prod-deploy       Проверить настройки, собрать и запустить стек' \
		'  make prod-down         Остановить стек без удаления данных' \
		'  make prod-restart      Перезапустить сервисы' \
		'  make prod-logs         Следить за логами' \
		'  make prod-ps           Показать состояние сервисов' \
		'  make prod-bootstrap    Создать первую учетную запись модератора' \
		'  make prod-config       Проверить итоговую Compose-конфигурацию'

check-tools:
	@command -v docker >/dev/null 2>&1 || { echo 'Ошибка: Docker не установлен.' >&2; exit 1; }
	@docker compose version >/dev/null 2>&1 || { echo 'Ошибка: Docker Compose v2 недоступен.' >&2; exit 1; }

local-init:
	@if [ ! -f "$(LOCAL_ENV)" ]; then \
		cp build/.env.local.example "$(LOCAL_ENV)" || exit 1; \
		echo "Создан $(LOCAL_ENV). При необходимости измените значения."; \
	fi

local-config: check-tools local-init
	@$(LOCAL_DC) config --quiet
	@echo 'Локальная Compose-конфигурация корректна.'

local-build: check-tools local-init
	$(LOCAL_DC) build

local-up: check-tools local-init
	$(LOCAL_DC) up -d --build --remove-orphans
	@port=$$(sed -n 's/^LOCAL_HTTP_PORT=//p' "$(LOCAL_ENV)"); \
		test -n "$$port" || port=3000; \
		echo "Приложение доступно: http://localhost:$$port"

local-down: check-tools local-init
	$(LOCAL_DC) down --remove-orphans

local-restart: check-tools local-init
	$(LOCAL_DC) restart

local-logs: check-tools local-init
	$(LOCAL_DC) logs -f --tail=200

local-ps: check-tools local-init
	$(LOCAL_DC) ps

local-bootstrap: check-tools local-init
	$(LOCAL_DC) --profile bootstrap run --rm bootstrap

prod-init:
	@if [ -f "$(PROD_ENV)" ]; then \
		echo "$(PROD_ENV) уже существует; файл не изменен."; \
	else \
		cp build/.env.prod.example "$(PROD_ENV)" || exit 1; \
		echo "Создан $(PROD_ENV). Заполните домен и секреты перед развертыванием."; \
	fi

prod-check: check-tools
	@test -f "$(PROD_ENV)" || { echo "Ошибка: нет $(PROD_ENV). Выполните make prod-init и заполните файл." >&2; exit 1; }
	@! grep -Eq '(^SITE_ADDRESS=school\.example\.com$$|^ALLOWED_ORIGINS=https://school\.example\.com$$|^POSTGRES_PASSWORD=change-me$$|^DATABASE_URL=.*change-me|^BOOTSTRAP_PASSWORD=change-this)' "$(PROD_ENV)" || { echo "Ошибка: замените тестовые домен и пароли в $(PROD_ENV)." >&2; exit 1; }
	@test -n "$$(sed -n 's/^SITE_ADDRESS=//p' "$(PROD_ENV)")" || { echo "Ошибка: SITE_ADDRESS не задан в $(PROD_ENV)." >&2; exit 1; }

prod-config: prod-check
	@$(PROD_DC) config --quiet
	@echo 'Production Compose-конфигурация корректна.'

prod-build: prod-check
	$(PROD_DC) build --pull

prod-up: prod-check
	$(PROD_DC) up -d --build --remove-orphans

prod-deploy: prod-up
	@echo 'Production-стек развернут. Проверьте состояние: make prod-ps'

prod-down: prod-check
	$(PROD_DC) down --remove-orphans

prod-restart: prod-check
	$(PROD_DC) restart

prod-logs: prod-check
	$(PROD_DC) logs -f --tail=200

prod-ps: prod-check
	$(PROD_DC) ps

prod-bootstrap: prod-check
	$(PROD_DC) --profile bootstrap run --rm bootstrap
