# Makefile in backend/
.PHONY: run dev test build clean

# Запуск в dev режиме (автосоздание БД и миграции)
dev:
	go run cmd/api/main.go

# Тесты
test:
	go test ./... -v

# E2E тесты
e2e:
	go test ./tests/e2e -v

# Сборка бинарника
build:
	go build -o bin/api cmd/api/main.go

# Заполнение БД тестовыми данными
seed:
	go run cmd/seed/main.go

# Очистка
clean:
	rm -f studio.db
	rm -rf bin/
	rm -rf uploads/

migrate-up:
	migrate -path migrations -database "$(MIGRATION_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATION_URL)" down 1
