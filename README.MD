# Bingo — Gamification Backend

REST API сервис для управления геймификационными сезонами: задачи, баланс монет, призы, заявки.

---

## Стек

- **Go 1.26** · **chi** (роутер) · **bun** (ORM) · **PostgreSQL**
- Аутентификация через Telegram Login Widget + gorilla/sessions (cookie)

---

## Запуск

### Переменные окружения

Создайте файл `dev.env` (или `prod.env`) в корне `backend/`:

```env
DB_CONNECTION_STRING=<connection string для PostgreSQL>
API_URL=:4010
TG_BOT_TOKEN=<telegram bot token>
SESSION_SECRET=<32+ random bytes>
SESSION_TTL_MINUTES=43200
CLIENT_URL=http://localhost:3000
```

### Миграции

```bash
# Применить все pending-миграции (используются embedded SQL-файлы)
go run ./cmd/migrator -env dev

# С внешней директорией миграций
go run ./cmd/migrator -env dev -migrations-path ./internal/db/migrations
```

### Бот в Telegram
1. Создайте бота через @BotFather, получите токен.
2. Установите `TG_BOT_TOKEN` в `.env`.
3. В настройках бота укажите домен вашего приложения (например, `https://your.domain.com`) и разрешите Telegram Login Widget (для теста можно использовать сервис ngrok).

### Сервер

```bash
go run ./cmd/api -env dev
# или
go build -o bin/api ./cmd/api && ./bin/api -env dev
```

### Тесты

```bash
go test ./...
# Сценарный тест (без БД, in-memory)
go test ./internal/api/ -run TestPlatformScenario -v
```

---

## Аутентификация

Все маршруты кроме `/auth/login` требуют активной cookie-сессии.  
Маршруты, помеченные **\[manager\]**, дополнительно требуют роль `manager`.

---

## API Reference

### Auth

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `POST` | `/auth/login` | public | Вход через Telegram |
| `POST` | `/auth/logout` | auth | Выход, очистка сессии |
| `GET` | `/auth/me` | auth | Данные текущего пользователя |

#### `POST /auth/login`

```json
{
  "id": 123456789,
  "first_name": "Ivan",
  "last_name": "Petrov",
  "username": "ivanpetrov",
  "photo_url": "https://t.me/...",
  "auth_date": 1712345678,
  "hash": "<HMAC-SHA256 от Telegram>"
}
```

Возвращает `domain.User`. Устанавливает cookie `sid`.

---

### Seasons

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `POST` | `/seasons` | auth | Создать сезон |
| `GET` | `/seasons/active` | auth | Активный сезон |
| `PATCH` | `/seasons/{id}` | auth | Обновить сезон |
| `DELETE` | `/seasons/{id}` | auth | Удалить сезон |

#### `POST /seasons`

```json
{
  "title": "Spring 2026",
  "start_date": "2026-03-01T00:00:00Z",
  "end_date": "2026-05-31T23:59:59Z",
  "is_active": true
}
```

#### `PATCH /seasons/{id}` — все поля опциональны

```json
{
  "title": "Spring 2026 (updated)",
  "is_active": false
}
```

`DELETE` возвращает `409 Conflict` если у сезона есть связанные записи.

---

### Tasks

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `GET` | `/tasks?season_id={id}` | auth | Список задач сезона |
| `GET` | `/tasks/{id}` | auth | Задача по ID |
| `POST` | `/tasks` | **\[manager\]** | Создать задачу |
| `PATCH` | `/tasks/{id}` | **\[manager\]** | Обновить задачу |
| `DELETE` | `/tasks/{id}` | **\[manager\]** | Удалить задачу |

#### `POST /tasks`

```json
{
  "season_id": 1,
  "title": "Написать пост в соцсетях",
  "category": "social",
  "description": "Любая площадка, минимум 200 слов",
  "reward_coins": 100,
  "sort_order": 3,
  "is_active": true
}
```

#### `PATCH /tasks/{id}` — все поля опциональны

```json
{
  "title": "Новый заголовок",
  "reward_coins": 150,
  "is_active": false
}
```

---

### Balance

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `GET` | `/balance/{seasonID}` | auth | Баланс текущего пользователя в сезоне |
| `GET` | `/balance/{seasonID}/transactions` | auth | История транзакций |
| `POST` | `/balance/{seasonID}/adjust` | **\[manager\]** | Ручная корректировка баланса |
| `POST` | `/balance/{seasonID}/join` | auth | Вступить в сезон |

`GET /balance/{seasonID}` — если баланса ещё нет, возвращает нулевой объект (не 404).  
`GET /balance/{seasonID}/transactions` — если баланса нет, возвращает пустой массив.

#### `POST /balance/{seasonID}/adjust`

```json
{
  "user_id": 42,
  "amount": -200,
  "note": "Reversal: задача отклонена"
}
```

`amount` — положительное (начисление) или отрицательное (списание). Возвращает `Transaction`.

---

### Submissions (отчёты по задачам)

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `GET` | `/submissions` | auth | Свои отчёты; `?user_id=X` — отчёты конкретного пользователя |
| `GET` | `/submissions/{id}` | auth | Отчёт по ID |
| `POST` | `/submissions` | **\[manager\]** | Подтвердить выполнение задачи (начисляет монеты) |
| `DELETE` | `/submissions/{id}` | **\[manager\]** | Удалить отчёт (монеты НЕ списываются автоматически) |

#### `POST /submissions`

```json
{
  "user_id": 42,
  "task_id": 7,
  "season_id": 1
}
```

Создаёт submission со статусом `approved` и немедленно начисляет `task.reward_coins` на баланс пользователя.

---

### Rewards (призы)

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `GET` | `/rewards?season_id={id}` | auth | Список призов сезона |
| `GET` | `/rewards/{id}` | auth | Приз по ID |
| `POST` | `/rewards` | **\[manager\]** | Создать приз |
| `PATCH` | `/rewards/{id}` | **\[manager\]** | Обновить приз |
| `DELETE` | `/rewards/{id}` | **\[manager\]** | Удалить приз |

#### `POST /rewards`

```json
{
  "season_id": 1,
  "title": "Coffee",
  "description": "Кофе от менеджера",
  "cost_coins": 50,
  "limit": 10,
  "status": "available"
}
```

`limit` — максимальное число заявок (`null` = безлимитно).  
`status`: `available` | `hidden`.

#### `PATCH /rewards/{id}` — все поля опциональны

```json
{
  "title": "Premium Coffee",
  "cost_coins": 75,
  "status": "hidden"
}
```

---

### Claims (заявки на призы)

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `POST` | `/claims` | auth | Подать заявку на приз (списывает монеты) |
| `GET` | `/claims` | auth | Свои заявки; `?user_id=X` — заявки конкретного пользователя |
| `GET` | `/claims/{id}` | auth | Заявка по ID |
| `PATCH` | `/claims/{id}/status` | **\[manager\]** | Обновить статус заявки |

#### `POST /claims`

```json
{
  "reward_id": 3,
  "season_id": 1
}
```

Монеты списываются сразу. Коды ошибок:
- `400` — приз недоступен (`status != available`)
- `409` — лимит исчерпан
- `422` — недостаточно монет

#### `PATCH /claims/{id}/status`

```json
{
  "status": "completed",
  "season_id": 1
}
```

`status`: `pending` | `completed` | `cancelled`.  
При переводе в `cancelled` монеты возвращаются на баланс.

---

## Коды ответов

| Код | Значение |
|---|---|
| `200` | OK |
| `201` | Created |
| `204` | No Content (успешное удаление) |
| `400` | Bad Request — невалидные параметры |
| `401` | Unauthorized — нет сессии |
| `403` | Forbidden — недостаточно прав |
| `404` | Not Found |
| `409` | Conflict — нарушение ограничений (лимит, связанные записи) |
| `422` | Unprocessable Entity — недостаточно монет |
| `500` | Internal Server Error |

Ошибки возвращаются в формате:

```json
{ "error": "описание ошибки" }
```

---

## Деплой на продакшн сервер

Проект деплоится через GitHub Actions при пуше тега вида `v*.*.*` или вручную через `workflow_dispatch`.

### Архитектура хостинга

```
Internet
  │
  ▼
nginx (443/80)          — SSL termination, static files, API proxy
  │
  ├─ /api/* ──────────► Go API  (127.0.0.1:8080, systemd service)
  │                          │
  │                          └─► PostgreSQL (localhost:5432)
  │
  └─ /* ──────────────► /var/www/bingo  (Vite build, SPA fallback)
```

### Требования к серверу

- Ubuntu 22.04+ / Debian 12+
- 1 CPU, 1 GB RAM минимум
- Открытые порты: 22 (SSH), 80 (HTTP), 443 (HTTPS)
- Зарегистрированный домен, A-запись которого указывает на IP сервера

---

### Шаг 1 — Первоначальная настройка сервера

Запусти один раз как root (или под sudo):

```bash
# Клонируй репозиторий на свою машину или скопируй deploy/ на сервер
scp -r deploy/ root@YOUR_SERVER:/tmp/bingo-deploy/

# На сервере:
CERTBOT_EMAIL=you@example.com \
DB_PASSWORD=your_strong_password \
bash /tmp/bingo-deploy/setup.sh --domain your.domain.com
```

Скрипт:
- Устанавливает PostgreSQL, nginx, certbot
- Создаёт БД и пользователя `bingo`
- Создаёт системного пользователя `bingo`
- Разворачивает nginx config с SSL
- Устанавливает systemd unit `bingo-api`

> **Важно:** скрипт выведет строку подключения к БД — скопируй её, она понадобится на следующем шаге.

---

### Шаг 2 — Настройка SSH-доступа для GitHub Actions

Создай отдельную SSH-пару для деплоя:

```bash
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/bingo_deploy
```

Добавь публичный ключ на сервер:

```bash
ssh-copy-id -i ~/.ssh/bingo_deploy.pub bingo@YOUR_SERVER
# или вручную добавь в /home/bingo/.ssh/authorized_keys
```

---

### Шаг 3 — GitHub Secrets

Перейди в `Settings → Secrets and variables → Actions` репозитория и добавь:

| Секрет | Пример значения | Описание |
|---|---|---|
| `SSH_HOST` | `1.2.3.4` или `your.domain.com` | IP или хост сервера |
| `SSH_USER` | `bingo` | SSH-пользователь (создан скриптом setup.sh) |
| `SSH_PRIVATE_KEY` | `-----BEGIN OPENSSH...` | Приватный ключ из шага 2 |
| `DB_CONNECTION_STRING` | `postgres://bingo:pass@localhost:5432/bingo` | Строка из вывода setup.sh |
| `TG_BOT_TOKEN` | `123456:ABC...` | Токен Telegram-бота |
| `SESSION_SECRET` | `<32+ случайных символов>` | Секрет для cookie-сессий |
| `CLIENT_URL` | `https://your.domain.com` | CORS: разрешённый origin |
| `API_URL` | `:8080` | Внутренний адрес Go-сервера |
| `VITE_API_BASE_URL` | `https://your.domain.com/api` | Куда фронтенд шлёт запросы |

Сгенерировать SESSION_SECRET:
```bash
openssl rand -base64 32
```

---

### Шаг 4 — Первый деплой

После настройки секретов запушь в main:

```bash
git push origin main
```

GitHub Actions выполнит:
1. Тесты бэкенда
2. Сборку Go-бинарников (linux/amd64, статические)
3. Сборку фронтенда (Vite)
4. Загрузку бинарников и `prod.env` на сервер
5. Загрузку статики в `/var/www/bingo`
6. Запуск миграций БД
7. Перезапуск `bingo-api` systemd-сервиса

---

### Управление сервисом вручную

```bash
# Статус
systemctl status bingo-api

# Логи (live)
journalctl -u bingo-api -f

# Перезапуск
systemctl restart bingo-api

# Проверить nginx
nginx -t && systemctl reload nginx
```

---

### Структура deploy/

```
deploy/
  nginx.conf        — конфиг nginx (YOUR_DOMAIN заменяется скриптом)
  setup.sh          — скрипт первоначальной настройки сервера
  bingo-api.service — systemd unit для Go API
```

### Структура .github/

```
.github/
  workflows/
    deploy.yml  — CI/CD pipeline (test → build → deploy)
```

### TODO:
- Сервис работы с telegram-ботом (уведомления и дополнительные)
- Безшовная аутентификация и авторизация в случае открытия приложения в TelegramMiniApp
- Добалвение метрики и трассировки приложения (prometheus + grafana)
- Добавление Redis-Cache для развертывания в кластере


