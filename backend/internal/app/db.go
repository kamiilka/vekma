package app

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func InitPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}
	if err := seedExchangeRates(db); err != nil {
		return nil, err
	}
	if err := seedNotificationTemplates(db); err != nil {
		return nil, err
	}
	if err := seedDocumentTemplates(db); err != nil {
		return nil, err
	}
	if err := seedDefaultCashboxes(db); err != nil {
		return nil, err
	}
	if err := seedDefaultWarehouse(db); err != nil {
		return nil, err
	}
	if err := seedDefaultPermissions(db); err != nil {
		return nil, err
	}

	if err := seedDefaultUsers(db); err != nil {
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS roles (
			name TEXT PRIMARY KEY
		)`,
		`CREATE TABLE IF NOT EXISTS permissions (
			name TEXT PRIMARY KEY
		)`,
		`CREATE TABLE IF NOT EXISTS role_permissions (
			role TEXT NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
			permission TEXT NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
			PRIMARY KEY (role, permission)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password TEXT NOT NULL,
			role TEXT NOT NULL REFERENCES roles(name)
		)`,
		`CREATE TABLE IF NOT EXISTS user_warehouse_scopes (
			username TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			warehouse_id BIGINT NOT NULL,
			PRIMARY KEY (username, warehouse_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_cashbox_scopes (
			username TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			cashbox_id BIGINT NOT NULL,
			PRIMARY KEY (username, cashbox_id)
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			code TEXT NOT NULL DEFAULT '',
			sku TEXT NOT NULL UNIQUE,
			article TEXT NOT NULL DEFAULT '',
			barcode TEXT NOT NULL DEFAULT '',
			serial_number TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			brand TEXT NOT NULL DEFAULT '',
			supplier TEXT NOT NULL DEFAULT '',
			purchase_price DOUBLE PRECISION NOT NULL DEFAULT 0,
			retail_price DOUBLE PRECISION NOT NULL DEFAULT 0,
			wholesale_price DOUBLE PRECISION NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'UAH',
			vat_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			min_stock INTEGER NOT NULL DEFAULT 0,
			warehouse_position TEXT NOT NULL DEFAULT '',
			comments TEXT NOT NULL DEFAULT '',
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS code TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS article TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS serial_number TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS wholesale_price DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'UAH'`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS vat_percent DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS warehouse_position TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS comments TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS archived BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS supplier_sku TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS supplier_name_ext TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS stock_movements (
			id BIGSERIAL PRIMARY KEY,
			product_id BIGINT NOT NULL REFERENCES products(id),
			from_warehouse_id BIGINT NULL,
			to_warehouse_id BIGINT NULL,
			from_cell_id BIGINT NULL,
			to_cell_id BIGINT NULL,
			movement_type TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS from_warehouse_id BIGINT NULL`,
		`ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS to_warehouse_id BIGINT NULL`,
		`ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS from_cell_id BIGINT NULL`,
		`ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS to_cell_id BIGINT NULL`,
		`CREATE TABLE IF NOT EXISTS warehouses (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS warehouse_stocks (
			warehouse_id BIGINT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			quantity INTEGER NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (warehouse_id, product_id)
		)`,
		`CREATE TABLE IF NOT EXISTS warehouse_zones (
			id BIGSERIAL PRIMARY KEY,
			warehouse_id BIGINT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (warehouse_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS warehouse_cells (
			id BIGSERIAL PRIMARY KEY,
			warehouse_id BIGINT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
			zone_id BIGINT NOT NULL REFERENCES warehouse_zones(id) ON DELETE CASCADE,
			code TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (zone_id, code)
		)`,
		`CREATE TABLE IF NOT EXISTS cell_stocks (
			cell_id BIGINT NOT NULL REFERENCES warehouse_cells(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			quantity INTEGER NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (cell_id, product_id)
		)`,
		`CREATE TABLE IF NOT EXISTS stock_transfers (
			id BIGSERIAL PRIMARY KEY,
			from_warehouse_id BIGINT NOT NULL REFERENCES warehouses(id),
			to_warehouse_id BIGINT NOT NULL REFERENCES warehouses(id),
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS stock_transfer_items (
			id BIGSERIAL PRIMARY KEY,
			transfer_id BIGINT NOT NULL REFERENCES stock_transfers(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			from_cell_id BIGINT NULL REFERENCES warehouse_cells(id) ON DELETE SET NULL,
			to_cell_id BIGINT NULL REFERENCES warehouse_cells(id) ON DELETE SET NULL,
			quantity INTEGER NOT NULL
		)`,
		`ALTER TABLE stock_transfer_items ADD COLUMN IF NOT EXISTS from_cell_id BIGINT NULL REFERENCES warehouse_cells(id) ON DELETE SET NULL`,
		`ALTER TABLE stock_transfer_items ADD COLUMN IF NOT EXISTS to_cell_id BIGINT NULL REFERENCES warehouse_cells(id) ON DELETE SET NULL`,
		`CREATE TABLE IF NOT EXISTS inventories (
			id BIGSERIAL PRIMARY KEY,
			warehouse_id BIGINT NOT NULL REFERENCES warehouses(id),
			status TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			applied_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS inventory_items (
			id BIGSERIAL PRIMARY KEY,
			inventory_id BIGINT NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			system_quantity INTEGER NOT NULL,
			actual_quantity INTEGER NOT NULL,
			adjustment INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sales (
			id BIGSERIAL PRIMARY KEY,
			order_id BIGINT NULL,
			total DOUBLE PRECISION NOT NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			total_uah DOUBLE PRECISION NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE sales ADD COLUMN IF NOT EXISTS order_id BIGINT NULL`,
		`ALTER TABLE sales ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'UAH'`,
		`ALTER TABLE sales ADD COLUMN IF NOT EXISTS total_uah DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`CREATE TABLE IF NOT EXISTS sale_items (
			id BIGSERIAL PRIMARY KEY,
			sale_id BIGINT NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS receipts (
			id BIGSERIAL PRIMARY KEY,
			sale_id BIGINT NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
			provider TEXT NOT NULL DEFAULT 'checkbox',
			status TEXT NOT NULL DEFAULT 'pending',
			external_id TEXT NOT NULL DEFAULT '',
			fiscal_number TEXT NOT NULL DEFAULT '',
			qr_url TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			payload TEXT NOT NULL DEFAULT '',
			sent_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (sale_id)
		)`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'checkbox'`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending'`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS external_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS fiscal_number TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS qr_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS error_message TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS payload TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ NULL`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
		`ALTER TABLE receipts ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_receipts_sale_id_unique ON receipts(sale_id)`,
		`CREATE INDEX IF NOT EXISTS idx_receipts_status ON receipts(status)`,
		`CREATE TABLE IF NOT EXISTS customer_orders (
			id BIGSERIAL PRIMARY KEY,
			customer_name TEXT NOT NULL,
			status TEXT NOT NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			total DOUBLE PRECISION NOT NULL DEFAULT 0,
			total_uah DOUBLE PRECISION NOT NULL DEFAULT 0,
			due_date TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS customers (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			comment TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS customer_reminders (
			id BIGSERIAL PRIMARY KEY,
			customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			text TEXT NOT NULL,
			due_at TIMESTAMPTZ NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			completed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS service_orders (
			id BIGSERIAL PRIMARY KEY,
			customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
			product_id BIGINT NULL REFERENCES products(id) ON DELETE SET NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			technician TEXT NOT NULL DEFAULT '',
			labor_min INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'new',
			price DOUBLE PRECISION NOT NULL DEFAULT 0,
			parts_total DOUBLE PRECISION NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'UAH',
			completed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS service_order_parts (
			id BIGSERIAL PRIMARY KEY,
			service_order_id BIGINT NOT NULL REFERENCES service_orders(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
			quantity INTEGER NOT NULL,
			price DOUBLE PRECISION NOT NULL,
			total DOUBLE PRECISION NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE service_orders ADD COLUMN IF NOT EXISTS technician TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE service_orders ADD COLUMN IF NOT EXISTS labor_min INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE service_orders ADD COLUMN IF NOT EXISTS parts_total DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE customer_orders ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'UAH'`,
		`ALTER TABLE customer_orders ADD COLUMN IF NOT EXISTS total DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE customer_orders ADD COLUMN IF NOT EXISTS total_uah DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE customer_orders ADD COLUMN IF NOT EXISTS due_date TIMESTAMPTZ NULL`,
		`CREATE TABLE IF NOT EXISTS customer_order_items (
			id BIGSERIAL PRIMARY KEY,
			order_id BIGINT NOT NULL REFERENCES customer_orders(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS reservations (
			id BIGSERIAL PRIMARY KEY,
			order_id BIGINT NOT NULL REFERENCES customer_orders(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			status TEXT NOT NULL,
			expires_at TIMESTAMPTZ NULL,
			released_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS payments (
			id BIGSERIAL PRIMARY KEY,
			order_id BIGINT NULL REFERENCES customer_orders(id) ON DELETE SET NULL,
			sale_id BIGINT NULL REFERENCES sales(id) ON DELETE SET NULL,
			service_order_id BIGINT NULL REFERENCES service_orders(id) ON DELETE SET NULL,
			cashbox_id BIGINT NULL,
			amount DOUBLE PRECISION NOT NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			amount_uah DOUBLE PRECISION NOT NULL DEFAULT 0,
			payment_method TEXT NOT NULL DEFAULT '',
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE payments ADD COLUMN IF NOT EXISTS cashbox_id BIGINT NULL`,
		`ALTER TABLE payments ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'UAH'`,
		`ALTER TABLE payments ADD COLUMN IF NOT EXISTS amount_uah DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE payments ADD COLUMN IF NOT EXISTS service_order_id BIGINT NULL REFERENCES service_orders(id) ON DELETE SET NULL`,
		`CREATE TABLE IF NOT EXISTS cashboxes (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			balance DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS cash_operations (
			id BIGSERIAL PRIMARY KEY,
			cashbox_id BIGINT NOT NULL REFERENCES cashboxes(id),
			operation_type TEXT NOT NULL,
			amount DOUBLE PRECISION NOT NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			amount_uah DOUBLE PRECISION NOT NULL DEFAULT 0,
			payment_method TEXT NOT NULL DEFAULT '',
			payment_id BIGINT NULL REFERENCES payments(id) ON DELETE SET NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE cash_operations ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'UAH'`,
		`ALTER TABLE cash_operations ADD COLUMN IF NOT EXISTS amount_uah DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`CREATE TABLE IF NOT EXISTS cash_shifts (
			id BIGSERIAL PRIMARY KEY,
			cashbox_id BIGINT NOT NULL REFERENCES cashboxes(id),
			status TEXT NOT NULL,
			opened_by TEXT NOT NULL,
			closed_by TEXT NOT NULL DEFAULT '',
			opening_balance DOUBLE PRECISION NOT NULL DEFAULT 0,
			closing_balance DOUBLE PRECISION NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			closed_at TIMESTAMPTZ NULL
		)`,
		`CREATE TABLE IF NOT EXISTS exchange_rates (
			currency TEXT PRIMARY KEY,
			rate_to_uah DOUBLE PRECISION NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS suppliers (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			contact TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			comments TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS service_categories (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS services (
			id BIGSERIAL PRIMARY KEY,
			category_id BIGINT NOT NULL REFERENCES service_categories(id),
			name TEXT NOT NULL,
			price DOUBLE PRECISION NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'UAH',
			duration_min INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS supplier_orders (
			id BIGSERIAL PRIMARY KEY,
			supplier_id BIGINT NOT NULL REFERENCES suppliers(id),
			status TEXT NOT NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			total DOUBLE PRECISION NOT NULL DEFAULT 0,
			total_uah DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS supplier_order_items (
			id BIGSERIAL PRIMARY KEY,
			supplier_order_id BIGINT NOT NULL REFERENCES supplier_orders(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS purchases (
			id BIGSERIAL PRIMARY KEY,
			supplier_id BIGINT NOT NULL REFERENCES suppliers(id),
			supplier_order_id BIGINT NULL REFERENCES supplier_orders(id) ON DELETE SET NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			total DOUBLE PRECISION NOT NULL DEFAULT 0,
			total_uah DOUBLE PRECISION NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS purchase_items (
			id BIGSERIAL PRIMARY KEY,
			purchase_id BIGINT NOT NULL REFERENCES purchases(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS notification_templates (
			id BIGSERIAL PRIMARY KEY,
			code TEXT NOT NULL,
			channel TEXT NOT NULL,
			subject TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (code, channel)
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id BIGSERIAL PRIMARY KEY,
			channel TEXT NOT NULL,
			recipient TEXT NOT NULL DEFAULT '',
			subject TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id BIGINT NOT NULL,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			sent_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0`,
		`CREATE TABLE IF NOT EXISTS background_jobs (
			id BIGSERIAL PRIMARY KEY,
			job_type TEXT NOT NULL,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 5,
			next_retry_at TIMESTAMPTZ NULL,
			payload TEXT NOT NULL DEFAULT '',
			result TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			started_at TIMESTAMPTZ NULL,
			finished_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE background_jobs ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE background_jobs ADD COLUMN IF NOT EXISTS max_attempts INTEGER NOT NULL DEFAULT 5`,
		`ALTER TABLE background_jobs ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ NULL`,
		`CREATE TABLE IF NOT EXISTS documents (
			id BIGSERIAL PRIMARY KEY,
			doc_type TEXT NOT NULL,
			doc_number TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			source_sale_id BIGINT NULL REFERENCES sales(id) ON DELETE SET NULL,
			source_purchase_id BIGINT NULL REFERENCES purchases(id) ON DELETE SET NULL,
			source_service_order_id BIGINT NULL REFERENCES service_orders(id) ON DELETE SET NULL,
			warehouse_id BIGINT NULL REFERENCES warehouses(id) ON DELETE SET NULL,
			cashbox_id BIGINT NULL REFERENCES cashboxes(id) ON DELETE SET NULL,
			currency TEXT NOT NULL DEFAULT 'UAH',
			total DOUBLE PRECISION NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			posted_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS source_sale_id BIGINT NULL REFERENCES sales(id) ON DELETE SET NULL`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS source_purchase_id BIGINT NULL REFERENCES purchases(id) ON DELETE SET NULL`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS source_service_order_id BIGINT NULL REFERENCES service_orders(id) ON DELETE SET NULL`,
		`CREATE TABLE IF NOT EXISTS document_items (
			id BIGSERIAL PRIMARY KEY,
			document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			product_id BIGINT NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS document_templates (
			id BIGSERIAL PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			body TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id BIGSERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			action TEXT NOT NULL,
			entity TEXT NOT NULL,
			details TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS location_type TEXT NOT NULL DEFAULT 'warehouse'`,
		`ALTER TABLE supplier_orders ADD COLUMN IF NOT EXISTS customer_order_id BIGINT NULL REFERENCES customer_orders(id) ON DELETE SET NULL`,
		`ALTER TABLE payments ADD COLUMN IF NOT EXISTS supplier_order_id BIGINT NULL REFERENCES supplier_orders(id) ON DELETE SET NULL`,
		`ALTER TABLE payments ADD COLUMN IF NOT EXISTS purchase_id BIGINT NULL REFERENCES purchases(id) ON DELETE SET NULL`,
		`CREATE TABLE IF NOT EXISTS counterparties (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			comment TEXT NOT NULL DEFAULT '',
			is_customer BOOLEAN NOT NULL DEFAULT FALSE,
			is_supplier BOOLEAN NOT NULL DEFAULT FALSE,
			customer_id BIGINT NULL REFERENCES customers(id) ON DELETE SET NULL,
			supplier_id BIGINT NULL REFERENCES suppliers(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS attachments (
			id BIGSERIAL PRIMARY KEY,
			entity_type TEXT NOT NULL,
			entity_id BIGINT NOT NULL,
			file_name TEXT NOT NULL,
			mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			size_bytes INTEGER NOT NULL DEFAULT 0,
			data BYTEA NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_entity ON attachments(entity_type, entity_id)`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func seedDefaultUsers(db *sql.DB) error {
	users := []User{
		{
			Username: "admin",
			Password: "admin123",
			Role:     "admin",
		},
		{
			Username: "seller",
			Password: "seller123",
			Role:     "seller",
		},
	}

	for _, user := range users {
		if _, err := db.Exec(`
			INSERT INTO users (username, password, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (username) DO NOTHING
		`,
			user.Username,
			user.Password,
			user.Role,
		); err != nil {
			return err
		}
	}

	return nil
}

func seedDefaultCashboxes(db *sql.DB) error {
	defaultCashboxes := []Cashbox{
		{Name: "Main Cash", Type: PaymentMethodCash, Currency: "UAH"},
		{Name: "Card Terminal", Type: PaymentMethodCard, Currency: "UAH"},
		{Name: "Bank Account", Type: PaymentMethodBank, Currency: "UAH"},
		{Name: "Virtual Wallet", Type: PaymentMethodVirtual, Currency: "UAH"},
	}

	for _, cashbox := range defaultCashboxes {
		if _, err := db.Exec(`
			INSERT INTO cashboxes (name, type, currency)
			VALUES ($1, $2, $3)
			ON CONFLICT (name) DO NOTHING
		`,
			cashbox.Name,
			cashbox.Type,
			cashbox.Currency,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedDefaultWarehouse(db *sql.DB) error {
	if _, err := db.Exec(`
		INSERT INTO warehouses (name, is_virtual)
		VALUES ('Main Warehouse', FALSE)
		ON CONFLICT (name) DO NOTHING
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO warehouse_stocks (warehouse_id, product_id, quantity, updated_at)
		SELECT w.id, p.id, p.stock, NOW()
		FROM warehouses w
		CROSS JOIN products p
		WHERE w.name = 'Main Warehouse'
		ON CONFLICT (warehouse_id, product_id) DO NOTHING
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO warehouse_zones (warehouse_id, name)
		SELECT id, 'DEFAULT'
		FROM warehouses
		WHERE name = 'Main Warehouse'
		ON CONFLICT (warehouse_id, name) DO NOTHING
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO warehouse_cells (warehouse_id, zone_id, code)
		SELECT z.warehouse_id, z.id, 'MAIN'
		FROM warehouse_zones z
		JOIN warehouses w ON w.id = z.warehouse_id
		WHERE w.name = 'Main Warehouse' AND z.name = 'DEFAULT'
		ON CONFLICT (zone_id, code) DO NOTHING
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO cell_stocks (cell_id, product_id, quantity, updated_at)
		SELECT c.id, p.id, p.stock, NOW()
		FROM warehouse_cells c
		JOIN warehouse_zones z ON z.id = c.zone_id
		JOIN warehouses w ON w.id = z.warehouse_id
		CROSS JOIN products p
		WHERE w.name = 'Main Warehouse' AND z.name = 'DEFAULT' AND c.code = 'MAIN'
		ON CONFLICT (cell_id, product_id) DO NOTHING
	`); err != nil {
		return err
	}

	return nil
}

func seedExchangeRates(db *sql.DB) error {
	defaultRates := []ExchangeRate{
		{Currency: "UAH", RateToUAH: 1},
		{Currency: "USD", RateToUAH: 40},
	}
	for _, rate := range defaultRates {
		if _, err := db.Exec(`
			INSERT INTO exchange_rates (currency, rate_to_uah)
			VALUES ($1, $2)
			ON CONFLICT (currency) DO NOTHING
		`,
			rate.Currency,
			rate.RateToUAH,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedNotificationTemplates(db *sql.DB) error {
	templates := []NotificationTemplate{
		{
			Code:     BackgroundJobTypeOverdueReminders,
			Channel:  NotificationChannelEmail,
			Subject:  "Overdue payment reminder",
			Body:     "Dear client, debt for {{entityType}} #{{entityId}} is {{debt}} {{currency}} ({{debtUah}} UAH). Overdue by {{overdueDays}} days.",
			IsActive: true,
		},
		{
			Code:     BackgroundJobTypeOverdueReminders,
			Channel:  NotificationChannelTelegram,
			Subject:  "Overdue payment reminder",
			Body:     "Reminder: {{entityType}} #{{entityId}} debt {{debt}} {{currency}} / {{debtUah}} UAH, overdue {{overdueDays}} days.",
			IsActive: true,
		},
	}

	for _, tpl := range templates {
		if _, err := db.Exec(`
			INSERT INTO notification_templates (code, channel, subject, body, is_active)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (code, channel) DO NOTHING
		`,
			tpl.Code,
			tpl.Channel,
			tpl.Subject,
			tpl.Body,
			tpl.IsActive,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedDocumentTemplates(db *sql.DB) error {
	templates := []DocumentTemplate{
		{
			Code:     DocumentTypeInvoice,
			Name:     "Invoice Template",
			Body:     "Invoice {{number}} from {{createdAt}}\nTotal: {{total}} {{currency}}\n{{items}}",
			IsActive: true,
		},
		{
			Code:     DocumentTypeAct,
			Name:     "Act Template",
			Body:     "Act {{number}}\nDate: {{createdAt}}\nTotal: {{total}} {{currency}}",
			IsActive: true,
		},
		{
			Code:     DocumentTypeCashInOrder,
			Name:     "PKO Template",
			Body:     "PKO {{number}}\nAmount: {{total}} {{currency}}\n{{note}}",
			IsActive: true,
		},
		{
			Code:     DocumentTypeCashOutOrder,
			Name:     "VKO Template",
			Body:     "VKO {{number}}\nAmount: {{total}} {{currency}}\n{{note}}",
			IsActive: true,
		},
	}

	for _, tpl := range templates {
		if _, err := db.Exec(`
			INSERT INTO document_templates (code, name, body, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (code) DO NOTHING
		`,
			tpl.Code,
			tpl.Name,
			tpl.Body,
			tpl.IsActive,
		); err != nil {
			return err
		}
	}

	return nil
}

func seedDefaultPermissions(db *sql.DB) error {
	for role := range defaultRolePermissions() {
		if _, err := db.Exec(`INSERT INTO roles (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, role); err != nil {
			return err
		}
	}

	for _, permission := range AllPermissions() {
		if _, err := db.Exec(`INSERT INTO permissions (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, permission); err != nil {
			return err
		}
	}

	for role, permissions := range defaultRolePermissions() {
		for permission := range permissions {
			if _, err := db.Exec(`
				INSERT INTO role_permissions (role, permission)
				VALUES ($1, $2)
				ON CONFLICT (role, permission) DO NOTHING
			`,
				role,
				permission,
			); err != nil {
				return err
			}
		}
	}

	return nil
}
