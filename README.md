# 🎬 StudioBooking Backend

**StudioBooking** — это B2C платформа для бронирования профессиональных фотостудий в Казахстане (MVP).  
Клиенты ищут студии, смотрят свободные слоты, бронируют, оставляют отзывы.  
Владельцы управляют студиями, комнатами, видят бронирования и обновляют статус оплаты.  
Администраторы верифицируют студии, модерируют контент и смотрят статистику.

**Технологии**:
- Go 1.24+ (Gin, GORM, JWT, bcrypt)
- База данных: SQLite (локально) + PostgreSQL (production/Docker)
- Файлы: локальное хранилище (готово к S3)
- Тесты: testify (unit + mock)
- Инфраструктура: Makefile, Docker, seed-скрипт

## 🚀 Быстрый старт (локальная разработка)

### Требования
- Go 1.24+
- Git
- Docker (опционально — для PostgreSQL)

### 1. Клонируй репозиторий
```bash
git clone https://github.com/your-org/photostudio-main.git
cd photostudio-main/backend
```

### 2. Установи зависимости
```bash
go mod download
```

### 3. Настрой окружение
Скопируй шаблон:
```bash
cp .env.example .env
```

По умолчанию используется **SQLite** — ничего менять не нужно!  
Если хочешь PostgreSQL — раскомментируй строку в `.env`:
```env
# SQLite (по умолчанию, для локальной разработки)
DATABASE_URL=studio.db

# PostgreSQL (раскомментируй для Docker/production)
# DATABASE_URL=postgres://photostudio:photostudio123@localhost:5432/photostudio?sslmode=disable

# Обязательно смени в production!
JWT_SECRET=your-super-secret-jwt-key-change-me

PORT=3001
ENV=development
```

### 4. Запусти сервер
```bash
# Вариант 1 — самый простой (рекомендуется на Windows)
go run cmd/api/main.go

# Вариант 2 — через Makefile (если make установлен)
make dev
```

Сервер запустится на: **http://localhost:3001**  
При первом запуске создастся файл `studio.db` и все таблицы (AutoMigrate).

### 5. Загрузи тестовые данные (обязательно!)
```bash
go run cmd/seed/main.go
# или
make seed
```

**Что будет создано:**
- 1 администратор
- 3 клиента
- 3 владельца студий
- 5 студий с 3 комнатами
- 10 бронирований
- 5 отзывов
- Несколько уведомлений

**Тестовые аккаунты** (все пароли: `client123` / `owner123` / `admin123`):

| Роль               | Email                              | Пароль     | Описание                          |
|--------------------|------------------------------------|------------|-----------------------------------|
| 👑 Администратор   | `admin@photostudio.kz`             | `admin123` | Полный доступ к админ-панели     |
| 👤 Клиент 1        | `asel@mail.kz`                     | `client123`| Тестовый клиент                  |
| 👤 Клиент 2        | `bekzat@gmail.com`                 | `client123`| Тестовый клиент                  |
| 👤 Клиент 3        | `dina@yandex.kz`                   | `client123`| Тестовый клиент                  |
| 🏢 Владелец 1      | `aidar@lightpro.kz`                | `owner123` | Владелец нескольких студий       |
| 🏢 Владелец 2      | `gulnaz@creativespace.kz`          | `owner123` | Владелец студий                  |
| 🏢 Владелец 3      | `yerlan@fashionstudio.kz`          | `owner123` | Владелец студий                  |

> **Важно**: После запуска seed-скрипта всегда используйте **один из этих аккаунтов** для тестирования.  
> Не создавайте новых пользователей вручную для демо — используйте готовые, чтобы всё работало сразу.

### 6. (Опционально) PostgreSQL через Docker
```bash
# Запусти PostgreSQL
docker-compose up -d

# Обнови .env (раскомментируй PostgreSQL строку)
# DATABASE_URL=postgres://photostudio:photostudio123@localhost:5432/photostudio?sslmode=disable

# Перезапусти backend
go run cmd/api/main.go
```

## 🛠 Полезные команды (Makefile)

```bash
make dev       # запуск сервера
make test      # все тесты
make seed      # загрузка тестовых данных
make build     # сборка бинарника (bin/api)
make clean     # удалить studio.db, bin/, uploads/
```

## 🔐 Основные API Endpoints

### Auth (публичные)
- `POST /api/v1/auth/register/client` — регистрация клиента
- `POST /api/v1/auth/register/studio` — регистрация владельца студии
- `POST /api/v1/auth/login` — вход (возвращает JWT)

### Catalog (публичные)
- `GET /api/v1/studios` — список студий (фильтры: city, min_price, max_price, room_type)
- `GET /api/v1/studios/:id` — детали студии
- `GET /api/v1/rooms/:id/availability?date=YYYY-MM-DD` — свободные слоты

### Booking (защищённые)
- `POST /api/v1/bookings` — создать бронирование
- `GET /api/v1/users/me/bookings` — мои бронирования
- `PATCH /api/v1/bookings/:id/payment` — обновить статус оплаты (владелец)
- `PATCH /api/v1/bookings/:id/status` — обновить статус (владелец)

### Owner (защищённые, роль `studio_owner`)
- `GET /api/v1/studios/my` — мои студии
- `POST /api/v1/studios/:id/photos` — загрузить фото студии
- `GET /api/v1/studios/:id/bookings` — бронирования моей студии

### Admin (защищённые, роль `admin`)
- `GET /api/v1/admin/studios/pending` — студии на верификации
- `POST /api/v1/admin/studios/:id/verify` — верифицировать студию
- `POST /api/v1/admin/studios/:id/reject` — отклонить студию
- `GET /api/v1/admin/statistics` — статистика платформы

## 🐳 Docker (PostgreSQL + API)

```bash
docker-compose up --build
```

- API: **http://localhost:3001**
- PostgreSQL: localhost:5432 (user: photostudio, pass: photostudio123, db: photostudio)

## 🔐 GitHub Secrets (CI/CD)

- `PROD_HOST`
- `PROD_USER`
- `PROD_SSH_KEY`

## 📂 Структура проекта

```text
backend/
├── cmd/
│   ├── api/         # главная точка входа
│   └── seed/        # скрипт тестовых данных
├── internal/
│   ├── database/    # подключение + SQLite fallback
│   ├── domain/      # бизнес-сущности (User, Studio, Booking...)
│   ├── middleware/  # JWT, роли, CORS
│   ├── modules/     # модули: auth, catalog, booking, admin...
│   ├── pkg/         # утилиты (jwt, response)
│   └── repository/  # доступ к БД
├── migrations/      # SQL миграции
├── uploads/         # загруженные фото
├── studio.db        # SQLite (в .gitignore)
├── .env.example
├── Makefile
└── docker-compose.yml
```

## ⚠️ Важные заметки

- **Локально** — всегда SQLite (не нужно устанавливать PostgreSQL)
- **Production** — обязательно PostgreSQL + Docker
- **Безопасность** — **смените JWT_SECRET** в production!
- **Тесты** — `make test` или `go test ./... -v`
- **Файлы** — загружаются в `./uploads/` (gitignore) 

## 📌 Статус проекта

- ✅ MVP завершён
- ✅ Полная аутентификация и роли
- ✅ Каталог, бронирование, отзывы, уведомления
- ✅ Admin-панель
- ✅ Локальная разработка без внешней БД
- ✅ Docker + seed + документация

## 💳 RoboKassa payments and recurring subscriptions

Required env vars:
- `ROBOKASSA_MERCHANT_LOGIN`
- `ROBOKASSA_IS_TEST`
- `ROBOKASSA_TEST_PASSWORD_1`, `ROBOKASSA_TEST_PASSWORD_2`
- `ROBOKASSA_PROD_PASSWORD_1`, `ROBOKASSA_PROD_PASSWORD_2`
- `ROBOKASSA_HASH_ALGO` (`sha256` by default, supported: `md5`, `sha256`, `sha512`)
- `ROBOKASSA_RESULT_URL`, `ROBOKASSA_SUCCESS_URL`, `ROBOKASSA_FAIL_URL`
- `ROBOKASSA_FRONTEND_SUCCESS_URL`, `ROBOKASSA_FRONTEND_FAIL_URL`

Flow:
1. Client creates booking payment via `POST /api/v1/payments/robokassa/create`.
2. Backend validates booking ownership and amount, stores payment, returns signed RoboKassa URL.
3. RoboKassa sends server-to-server callback to `POST /webhooks/robokassa/result`.
4. Backend verifies signature (`PASSWORD_2`), checks amount, marks payment paid idempotently.
5. For recurring subscription first payment, `first_invoice_id` is persisted and subscription becomes active.


Production readiness checklist (RoboKassa):
- Use only public webhook endpoint `POST /webhooks/robokassa/result` (single ResultURL target).
- Keep `ROBOKASSA_IS_TEST=0` in production and set only PROD passwords in runtime secrets.
- Verify DB migration `000035` is applied before rollout.
- Ensure application can write booking/payment/subscription state updates (callback handlers now fail closed on write errors).
- Monitor webhook 4xx/5xx rates and replay detection (`ErrReplayDetected`) as fraud/security signals.
