# Bug Report Service

Сервис для приема и обработки баг-репортов. Пользователи отправляют обращения через публичную форму, модераторы просматривают, меняют статусы, оставляют внутренние заметки.

## Структура проекта

```
backend/            Go API сервер
frontend/           React SPA (Vite + TypeScript)
.env.example        Шаблон переменных окружения
```

## Архитектура бэкенда

Бэкенд написан на Go и построен по принципу Clean Architecture с четырьмя слоями:

```
backend/internal/
  domain/           Доменные сущности и бизнес-правила
  usecase/          Сценарии использования (по одному пакету на операцию)
  api/              HTTP слой: роутер, хендлеры, middleware авторизации
  infrastructure/   Реализации: БД, S3, JWT, логгер, middleware
```

**domain** содержит структуры данных (Report, Attachment, Note, User), валидаторы статусов и приоритетов, политики доступа. Этот слой не зависит ни от чего.

**usecase** содержит бизнес-логику. Каждая операция живет в своем пакете: `create_report`, `mod_login`, `list_attachments` и т.д. Каждый пакет определяет интерфейсы зависимостей в `contract.go` и реализует логику в `usecase.go`.

**api** принимает HTTP запросы, парсит параметры, вызывает нужный usecase и формирует JSON ответ. Хендлеры тоже разбиты по пакетам. Роутинг на chi.

**infrastructure** реализует интерфейсы из usecase: PostgreSQL репозитории, S3 хранилище, JWT менеджер, bcrypt хешер, Zap логгер, HTTP middleware (CORS, rate limit, request ID, recovery, logging).

Зависимости собираются в `bootstrap/app.go`: создаются репозитории, usecase, хендлеры, роутер и HTTP сервер.

## Технологии

- Go 1.25, chi router, pgx (PostgreSQL), JWT, bcrypt
- TUS протокол для resumable загрузки файлов в S3/MinIO
- React, TypeScript, Vite
- Docker Compose (PostgreSQL, MinIO, миграции, API, Swagger UI)

## Быстрый старт

1. Скопировать `.env.example` в `.env` и заполнить секреты
2. Запустить:

```bash
make up
```

Это поднимет PostgreSQL, MinIO, прогонит миграции и запустит API на `http://localhost:8080`.

Проверка здоровья: `curl http://localhost:8080/healthz`

Swagger UI: `http://localhost:8081` (запуск через `make swagger`)

## Makefile

Все команды работают из корня проекта (проксируются в `backend/`).

### Тесты

| Команда | Что делает |
|---------|-----------|
| `make test` | Юнит-тесты |
| `make test-v` | Юнит-тесты с подробным выводом |
| `make test-cover` | Покрытие по хендлерам, usecase, middleware и domain |
| `make test-integration` | Интеграционные тесты (нужен Docker с PostgreSQL) |

### Качество кода

| Команда | Что делает |
|---------|-----------|
| `make fmt` | Форматирование кода |
| `make fmt-check` | Проверка форматирования (без изменений) |
| `make vet` | Статический анализ go vet |
| `make lint` | Линтер golangci-lint |
| `make check` | Все проверки разом: fmt + vet + lint + test |

### Сборка

| Команда | Что делает |
|---------|-----------|
| `make build` | Собрать бинарник |
| `make run` | Запустить локально |
| `make clean` | Удалить бинарник и coverage.out |
| `make deps` | Скачать зависимости |
| `make mod-tidy` | go mod tidy |

### CI

| Команда | Что делает |
|---------|-----------|
| `make ci` | Полный пайплайн: fmt + vet + lint + test + coverage |
| `make pre-commit` | Быстрая проверка перед коммитом: fmt + vet + test |

### Docker

| Команда | Что делает |
|---------|-----------|
| `make up` | Поднять проект со всеми зависимостями |
| `make down` | Остановить и удалить volumes |
| `make docker-restart` | Перезапустить контейнеры |
| `make docker-build` | Пересобрать образы без кеша |
| `make docker-logs` | Логи контейнеров |
| `make docker-ps` | Статус контейнеров |
| `make docker-clean` | Остановить и удалить все, включая volumes |
| `make swagger` | Поднять Swagger UI |

## API

### Публичные эндпоинты

```
POST   /api/v1/public/upload-sessions                         Создать сессию загрузки
DELETE /api/v1/public/upload-sessions/{id}/uploads/{uploadId}  Удалить файл из сессии
POST   /api/v1/public/reports                                 Отправить баг-репорт
```

### Загрузка файлов (TUS протокол)

```
POST   /api/v1/uploads          Создать загрузку
PATCH  /api/v1/uploads/{id}     Продолжить загрузку
HEAD   /api/v1/uploads/{id}     Статус загрузки
```

### Авторизация модератора

```
POST   /api/v1/mod/auth/login     Логин (email + пароль)
POST   /api/v1/mod/auth/refresh   Обновить токен
```

### Модератор (нужен Bearer токен)

```
GET    /api/v1/mod/me                          Профиль
GET    /api/v1/mod/reports                     Список репортов (фильтры, пагинация)
GET    /api/v1/mod/reports/{id}                Детали репорта
PATCH  /api/v1/mod/reports/{id}/status         Изменить статус
PATCH  /api/v1/mod/reports/{id}/meta           Изменить приоритет и влияние
GET    /api/v1/mod/reports/{id}/notes          Заметки к репорту
POST   /api/v1/mod/reports/{id}/notes          Добавить заметку
GET    /api/v1/mod/reports/{id}/attachments    Вложения репорта
```

## Переменные окружения

Полный список в `.env.example`. Основные:

| Переменная | Описание | По умолчанию |
|-----------|---------|-------------|
| `APP_ENV` | Окружение (local/production) | local |
| `HTTP_ADDR` | Адрес сервера | :8080 |
| `DATABASE_URL` | PostgreSQL connection string | |
| `JWT_SECRET` | Секрет для подписи токенов | |
| `S3_ENDPOINT` | Эндпоинт S3/MinIO | http://minio:9000 |
| `S3_BUCKET` | Бакет для вложений | bug-attachments |
| `CORS_ALLOWED_ORIGINS` | Разрешенные origin через запятую | * |

В local режиме сервер запускается без БД и S3 (только health эндпоинты). В production все переменные обязательны.

## Модераторы

Создаются автоматически при старте из переменных окружения:

```
MOD_SEED_1_EMAIL=admin@example.com
MOD_SEED_1_NAME=Admin
MOD_SEED_1_PASSWORD=secret123
```

Поддерживается до 5 аккаунтов (MOD_SEED_1 ... MOD_SEED_5). Для продакшена можно указать `MOD_SEED_N_PASSWORD_HASH` вместо пароля в открытом виде.

Также есть CLI утилита:

```bash
go run ./cmd/moderatorctl -email admin@example.com -password secret123
```

## Фронтенд

```bash
cd frontend
npm ci
npm run dev    # dev сервер
npm run build  # production сборка
```
