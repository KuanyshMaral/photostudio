# Makefile in backend/
migrate-up:
	migrate -path migrations -database "$(MIGRATION_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATION_URL)" down 1