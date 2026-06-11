# ERP Core (Go + Vue)

MVP ERP-система за вашим ТЗ: авторизація JWT, granular RBAC, товари, складські рухи, продажі, базова аналітика, аудит дій, PostgreSQL-персистентність, адаптивний UI, тести та Docker-запуск.

## Що реалізовано

- **Auth + RBAC база**: логін/пароль, JWT, ролі `admin` і `seller`.
- **RBAC permissions**: доступ до ендпоінтів за правами (`products:read/write`, `stock:write`, `sales:write`, `analytics:read`, `audit:read`, `users:*`, `roles:*`).
- **Номенклатура (розширено)**: створення, редагування, архівація, пошук товарів і перевірка дублів.
- **Складський облік (MVP)**: надходження/списання з перевіркою залишків.
- **Продажі (MVP)**: проведення продажу з авто-зменшенням залишку.
- **Замовлення + резерви (базово)**: створення замовлень, резервування товару, контроль статусів, авто-зняття резерву при скасуванні/продажі.
- **Оплати та борги (базово)**: часткові оплати, доплати, перегляд історії оплат та поточних боргів по замовленнях/продажах.
- **Каси та рух коштів (базово)**: каси/рахунки за типами (`cash`, `card`, `bank`, `virtual`), авто-рух коштів при оплатах, ручні касові операції, журнал руху коштів.
- **CashShift (MVP)**: відкриття/закриття касової зміни з фіксацією `opening/closing balance`, `openedBy/closedBy`, фільтрація змін по касі та статусу.
- **Мультивалютність (базово)**: валюти `UAH/USD`, довідник курсів, перерахунок у `UAH` для оплат, касових рухів та боргів.
- **Постачальники та закупівлі (базово)**: довідник постачальників, замовлення постачальнику (`supplier orders`), надходження товару (`purchases`) з автоматичним збільшенням залишків і оновленням статусу замовлення постачальнику.
- **Автозакупівля (MVP)**: рекомендації на закупівлю за дефіцитом (`min stock` + buffer від продажів за 30 днів) і швидке створення `supplier order` з рекомендацій.
- **Автозакупівля (phase 2)**: grouped-рекомендації по постачальниках і bulk-створення кількох `supplier orders` одним запитом.
- **SupplierOrder UX-правила**: заборона переприймання понад замовлену кількість, окреме приймання по лініях замовлення постачальнику, аналітика `pending to receive`.
- **Мультисклад + інвентаризація (базово)**: довідник складів, залишки по складах, переміщення між складами, інвентаризації з автоматичними корекціями залишків.
- **Phase 2 адресного зберігання (базово)**: зони складу (`WarehouseZone`), комірки (`WarehouseCell`) і залишки на рівні комірок (`cell-level stock`).
- **Документообіг (базово)**: документи (`invoice`, `act`, `cash in/out`, `returns`), проведення документів зі складськими/фінансовими ефектами, шаблони документів і генерація PDF.
- **Конструктор шаблонів (phase 2)**: довідник доступних плейсхолдерів, валідація шаблону перед збереженням і preview рендеру (з реальним або демо-документом).
- **Strict mode шаблонів**: для кожного типу документа задаються обов’язкові плейсхолдери; збереження шаблону працює у strict-режимі, validate/preview підтримують `strict=true`.
- **Прострочення та історія погашень (базово)**: `dueDate` для замовлень, ознака та дні прострочення, звіт по прострочених боргах і таймлайн погашень.
- **Нагадування та background jobs**: шаблони повідомлень (`email`/`telegram`), реальна відправка через `SMTP`/`Telegram Bot` (за наявності env-конфігурації), retry-політика з backoff для failed jobs.
- **Аналітика (MVP)**: KPI по товарах, залишках, продажах і виторгу.
- **Audit log**: журнал ключових дій (логін, створення товарів, складські рухи, продажі).
- **PostgreSQL**: автоматичні міграції таблиць і seed демо-користувачів при старті backend.
- **UI (Vue)**: сучасний dashboard з формами операцій, журналом дій для admin і модулем керування користувачами/правами.
- **UX-сесія**: persist сесії у `localStorage`, `Logout`, авто-вихід при `401 SESSION_EXPIRED`.
- **Навігація**: окремі вкладки `Dashboard` і `Admin` (admin-only).
- **Тести**: backend unit + handler tests, frontend component test.

## Техстек

- Backend: `Go`, `chi`, `jwt/v5`
- Frontend: `Vue 3`, `Vite`, `TypeScript`, `Vitest`
- Контейнеризація: `Docker`, `docker-compose`

## Запуск через Docker

1. Перейдіть у корінь проєкту:

```bash
cd /Users/uraaruta/go/src/с1PR
```

2. Підніміть сервіси:

```bash
docker compose up --build
```

3. Відкрийте:
- Frontend: [http://localhost:5173](http://localhost:5173)
- Backend health: [http://localhost:8080/healthz](http://localhost:8080/healthz)
- PostgreSQL: `localhost:5432` (`erp/erp`, db `erp`)

Демо-облікові записи:
- `admin / admin123`
- `seller / seller123`

## Локальний запуск без Docker

### Backend

```bash
cd backend
go mod tidy
export DATABASE_URL="postgres://erp:erp@localhost:5432/erp?sslmode=disable"
# optional notification providers
export SMTP_ADDR="smtp.example.com:587"
export SMTP_USER="user"
export SMTP_PASS="pass"
export SMTP_FROM="noreply@example.com"
export NOTIFY_EMAIL_TO="finance@example.com"
export TELEGRAM_BOT_TOKEN="<bot-token>"
export TELEGRAM_CHAT_ID="<chat-id>"
go run ./cmd/server
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Тести

### Backend

```bash
cd backend
go test ./...
```

### Frontend

```bash
cd frontend
npm test
```

## Основні API-ендпоінти

- `POST /api/v1/auth/login`
- `GET /api/v1/products`
- `POST /api/v1/products`
- `PUT /api/v1/products/{id}`
- `POST /api/v1/products/{id}/archive`
- `GET /api/v1/products/duplicates`
- `POST /api/v1/stock/movements`
- `POST /api/v1/sales`
- `POST /api/v1/orders`
- `GET /api/v1/orders`
- `PUT /api/v1/orders/{id}/status`
- `GET /api/v1/reservations`
- `POST /api/v1/warehouses`
- `GET /api/v1/warehouses`
- `GET /api/v1/warehouses/stocks`
- `POST /api/v1/warehouse-zones`
- `GET /api/v1/warehouse-zones`
- `POST /api/v1/warehouse-cells`
- `GET /api/v1/warehouse-cells`
- `GET /api/v1/warehouse-cells/stocks`
- `POST /api/v1/transfers`
- `POST /api/v1/transfers/cell`
- `POST /api/v1/transfers/cell/fifo`
- `POST /api/v1/inventories`
- `GET /api/v1/inventories`
- `POST /api/v1/inventories/{id}/apply`
- `POST /api/v1/documents`
- `GET /api/v1/documents`
- `POST /api/v1/documents/{id}/post`
- `GET /api/v1/documents/{id}/pdf`
- `GET /api/v1/document-templates`
- `PUT /api/v1/document-templates`
- `GET /api/v1/document-templates/placeholders`
- `POST /api/v1/document-templates/validate`
- `POST /api/v1/document-templates/preview`
- `POST /api/v1/suppliers`
- `GET /api/v1/suppliers`
- `POST /api/v1/supplier-orders`
- `GET /api/v1/supplier-orders`
- `GET /api/v1/supplier-orders/pending`
- `GET /api/v1/supplier-orders/recommendations`
- `POST /api/v1/supplier-orders/recommendations/create-order`
- `GET /api/v1/supplier-orders/recommendations/grouped`
- `POST /api/v1/supplier-orders/recommendations/create-orders-bulk`
- `POST /api/v1/supplier-orders/{id}/receive`
- `PUT /api/v1/supplier-orders/{id}/status`
- `POST /api/v1/purchases`
- `GET /api/v1/purchases`
- `POST /api/v1/payments`
- `GET /api/v1/payments`
- `GET /api/v1/debts`
- `GET /api/v1/debts/overdue`
- `GET /api/v1/debts/history?entityType=order|sale&entityId=...`
- `GET /api/v1/notification-templates`
- `PUT /api/v1/notification-templates`
- `GET /api/v1/notifications`
- `POST /api/v1/jobs/overdue-reminders`
- `POST /api/v1/jobs/run`
- `GET /api/v1/jobs`
- `GET /api/v1/cashboxes`
- `POST /api/v1/cashboxes`
- `GET /api/v1/cash-operations`
- `POST /api/v1/cash-operations`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `GET /api/v1/cash-shifts`
- `GET /api/v1/exchange-rates`
- `PUT /api/v1/exchange-rates`
- `GET /api/v1/analytics/summary`
- `GET /api/v1/audit/logs?limit=20`
- `GET /api/v1/users`
- `POST /api/v1/users`
- `GET /api/v1/roles`
- `PUT /api/v1/roles/{role}/permissions`

## Права доступу (поточні ролі)

- `admin`: усі дії, керування користувачами і правами ролей, перегляд журналу аудиту.
- `seller`: перегляд товарів, рух товару, продаж, аналітика (без створення товарів, без user/role management, без audit logs).

## Що далі для повної ERP-версії

ТЗ охоплює велику full ERP-систему (CRM, ремонти, документообіг, борги, інтеграції, імпорт/експорт, повідомлення тощо). Поточна версія є стабільним стартовим ядром, яке можна нарощувати модульно.
