# Makefile in backend/
.PHONY: run dev test build clean

# Запуск в dev режиме (автосоздание БД и миграции)
dev:
	go run cmd/api/main.go

# Тесты
test:
	go test ./... -v

# Сборка бинарника
build:
	go build -o bin/api cmd/api/main.go

# Очистка
clean:
	rm -f studio.db
	rm -rf bin/
	rm -rf uploads/

migrate-up:
	migrate -path migrations -database "$(MIGRATION_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATION_URL)" down 1
