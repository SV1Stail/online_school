.PHONY: build fmt lint migrate-up migrate-down run bootstrap

GOCACHE ?= /tmp/online-school-go-cache
GOMODCACHE ?= /tmp/online-school-go-mod

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build ./...

fmt:
	gofmt -w cmd internal

lint:
	golangci-lint run ./...

migrate-up:
	docker run --rm --network host -v "$(CURDIR)/migration:/migrations:ro" migrate/migrate:v4.20.1 \
		-path=/migrations -database "$(DATABASE_URL)" up

migrate-down:
	docker run --rm --network host -v "$(CURDIR)/migration:/migrations:ro" migrate/migrate:v4.20.1 \
		-path=/migrations -database "$(DATABASE_URL)" down 1

run:
	go run ./cmd/api

bootstrap:
	go run ./cmd/bootstrap
