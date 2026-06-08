package app

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrSaleNotFound      = errors.New("sale not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrReceiptNotFound   = errors.New("receipt not found")

	ErrReceiptSenderNotConfigured = errors.New("receipt sender is not configured")
)

// GetProduct returns a single product by ID (public wrapper around getProductByID).
func (s *Store) GetProduct(id int64) (Product, error) {
	return s.getProductByID(id)
}

// GetSale returns a single sale by ID (public wrapper around saleByID).
func (s *Store) GetSale(id int64) (Sale, error) {
	sale, err := s.saleByID(id)
	if err != nil {
		return Sale{}, ErrSaleNotFound
	}
	return sale, nil
}

const defaultWarehouseName = "Main Warehouse"

var templatePlaceholderPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

type Store struct {
	mu sync.RWMutex
	db *sql.DB

	users                 map[string]User
	rolePermissions       map[string]map[string]struct{}
	userWarehouseScopes   map[string]map[int64]struct{}
	userCashboxScopes     map[string]map[int64]struct{}
	products              []Product
	warehouses            []Warehouse
	warehouseZones        []WarehouseZone
	warehouseCells        []WarehouseCell
	warehouseStocks       []WarehouseStock
	cellStocks            []CellStock
	movements             []StockMovement
	transfers             []StockTransfer
	inventories           []Inventory
	documents             []Document
	documentTemplates     []DocumentTemplate
	sales                 []Sale
	receipts              []Receipt
	customers             []Customer
	customerReminders     []CustomerReminder
	serviceOrders         []ServiceOrder
	serviceOrderParts     []ServiceOrderPart
	serviceCategories     []ServiceCategory
	services              []Service
	orders                []CustomerOrder
	suppliers             []Supplier
	supplierOrders        []SupplierOrder
	purchases             []Purchase
	reservations          []Reservation
	payments              []Payment
	cashboxes             []Cashbox
	cashOperations        []CashOperation
	cashShifts            []CashShift
	notificationTemplates []NotificationTemplate
	notifications         []Notification
	backgroundJobs        []BackgroundJob
	exchangeRates         map[string]float64
	notificationSender    NotificationSender
	receiptSender         ReceiptSender
	auditLogs             []AuditLog
	settings              map[string]string

	productSeq          int64
	warehouseSeq        int64
	zoneSeq             int64
	cellSeq             int64
	movementSeq         int64
	transferSeq         int64
	inventorySeq        int64
	documentSeq         int64
	saleSeq             int64
	receiptSeq          int64
	customerSeq         int64
	customerRemSeq      int64
	serviceOrderSeq     int64
	serviceOrderPartSeq int64
	serviceCategorySeq  int64
	serviceSeq          int64
	orderSeq            int64
	supplierSeq         int64
	supplierOrderSeq    int64
	purchaseSeq         int64
	reservationSeq      int64
	paymentSeq          int64
	cashboxSeq          int64
	cashOperationSeq    int64
	cashShiftSeq        int64
	templateSeq         int64
	documentTplSeq      int64
	notificationSeq     int64
	jobSeq              int64
	auditSeq            int64
}

func NewStore() *Store {
	return &Store{
		users:               defaultUsers(),
		rolePermissions:     defaultRolePermissions(),
		userWarehouseScopes: map[string]map[int64]struct{}{},
		userCashboxScopes:   map[string]map[int64]struct{}{},
		products:            []Product{},
		warehouses: []Warehouse{
			{ID: 1, Name: "Main Warehouse", IsVirtual: false, LocationType: "warehouse", CreatedAt: time.Now().UTC()},
		},
		warehouseZones: []WarehouseZone{
			{ID: 1, WarehouseID: 1, Name: "DEFAULT", CreatedAt: time.Now().UTC()},
		},
		warehouseCells: []WarehouseCell{
			{ID: 1, WarehouseID: 1, ZoneID: 1, Code: "MAIN", CreatedAt: time.Now().UTC()},
		},
		warehouseStocks: []WarehouseStock{},
		cellStocks:      []CellStock{},
		movements:       []StockMovement{},
		transfers:       []StockTransfer{},
		inventories:     []Inventory{},
		documents:       []Document{},
		documentTemplates: []DocumentTemplate{
			{
				ID:        1,
				Code:      DocumentTypeInvoice,
				Name:      "Invoice Template",
				Body:      "Invoice {{number}} from {{createdAt}}\nTotal: {{total}} {{currency}}\n{{items}}",
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				ID:        2,
				Code:      DocumentTypeAct,
				Name:      "Act Template",
				Body:      "Act {{number}}\nDate: {{createdAt}}\nTotal: {{total}} {{currency}}",
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				ID:        3,
				Code:      DocumentTypeCashInOrder,
				Name:      "PKO Template",
				Body:      "PKO {{number}}\nAmount: {{total}} {{currency}}\n{{note}}",
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				ID:        4,
				Code:      DocumentTypeCashOutOrder,
				Name:      "VKO Template",
				Body:      "VKO {{number}}\nAmount: {{total}} {{currency}}\n{{note}}",
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		},
		sales:             []Sale{},
		receipts:          []Receipt{},
		customers:         []Customer{},
		customerReminders: []CustomerReminder{},
		serviceOrders:     []ServiceOrder{},
		serviceOrderParts: []ServiceOrderPart{},
		serviceCategories: []ServiceCategory{},
		services:          []Service{},
		orders:            []CustomerOrder{},
		suppliers:         []Supplier{},
		supplierOrders:    []SupplierOrder{},
		purchases:         []Purchase{},
		reservations:      []Reservation{},
		payments:          []Payment{},
		cashboxes: []Cashbox{
			{ID: 1, Name: "Main Cash", Type: PaymentMethodCash, Currency: "UAH", Balance: 0, CreatedAt: time.Now().UTC()},
			{ID: 2, Name: "Card Terminal", Type: PaymentMethodCard, Currency: "UAH", Balance: 0, CreatedAt: time.Now().UTC()},
			{ID: 3, Name: "Bank Account", Type: PaymentMethodBank, Currency: "UAH", Balance: 0, CreatedAt: time.Now().UTC()},
			{ID: 4, Name: "Virtual Wallet", Type: PaymentMethodVirtual, Currency: "UAH", Balance: 0, CreatedAt: time.Now().UTC()},
		},
		cashOperations: []CashOperation{},
		cashShifts:     []CashShift{},
		notificationTemplates: []NotificationTemplate{
			{
				ID:        1,
				Code:      BackgroundJobTypeOverdueReminders,
				Channel:   NotificationChannelEmail,
				Subject:   "Overdue payment reminder",
				Body:      "Dear client, debt for {{entityType}} #{{entityId}} is {{debt}} {{currency}} ({{debtUah}} UAH). Overdue by {{overdueDays}} days.",
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				ID:        2,
				Code:      BackgroundJobTypeOverdueReminders,
				Channel:   NotificationChannelTelegram,
				Subject:   "Overdue payment reminder",
				Body:      "Reminder: {{entityType}} #{{entityId}} debt {{debt}} {{currency}} / {{debtUah}} UAH, overdue {{overdueDays}} days.",
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		},
		notifications:  []Notification{},
		backgroundJobs: []BackgroundJob{},
		exchangeRates: map[string]float64{
			"UAH": 1,
			"USD": 40,
		},
		warehouseSeq:   1,
		zoneSeq:        1,
		cellSeq:        1,
		cashboxSeq:     4,
		templateSeq:    2,
		documentTplSeq: 4,
		auditLogs:      []AuditLog{},
		settings:       map[string]string{},
	}
}

func NewStoreWithDB(db *sql.DB) *Store {
	return &Store{
		db:                  db,
		users:               defaultUsers(),
		rolePermissions:     defaultRolePermissions(),
		userWarehouseScopes: map[string]map[int64]struct{}{},
		userCashboxScopes:   map[string]map[int64]struct{}{},
		exchangeRates: map[string]float64{
			"UAH": 1,
			"USD": 40,
		},
		settings: map[string]string{},
	}
}

func (s *Store) SetNotificationSender(sender NotificationSender) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notificationSender = sender
}

func (s *Store) SetReceiptSender(sender ReceiptSender) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receiptSender = sender
}

// NotificationConfig holds all credentials for notification channels.
type NotificationConfig struct {
	SMTPAddr        string `json:"smtpAddr"`
	SMTPUser        string `json:"smtpUser"`
	SMTPPass        string `json:"smtpPass"`
	SMTPFrom        string `json:"smtpFrom"`
	TelegramToken   string `json:"telegramToken"`
	TelegramChatID  string `json:"telegramChatId"`
	SMSGatewayURL   string `json:"smsGatewayUrl"`
	SMSGatewayToken string `json:"smsGatewayToken"`
	SMSPhoneTo      string `json:"smsPhoneTo"`
	ViberToken      string `json:"viberToken"`
	ViberRecipient  string `json:"viberRecipient"`
}

func (s *Store) GetSetting(key string) string {
	if s.db != nil {
		var value string
		err := s.db.QueryRow(`SELECT value FROM app_settings WHERE key = $1`, key).Scan(&value)
		if err == nil {
			return value
		}
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings[key]
}

func (s *Store) SetSetting(key, value string) error {
	if s.db != nil {
		_, err := s.db.Exec(`
			INSERT INTO app_settings (key, value, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		`, key, value)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[key] = value
	return nil
}

func (s *Store) GetNotificationConfig() NotificationConfig {
	raw := s.GetSetting("notification_config")
	if raw != "" {
		var cfg NotificationConfig
		if err := json.Unmarshal([]byte(raw), &cfg); err == nil {
			return cfg
		}
	}
	return NotificationConfig{}
}

func (s *Store) SaveNotificationConfig(cfg NotificationConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := s.SetSetting("notification_config", string(raw)); err != nil {
		return err
	}
	// Rebuild the live notifier from the saved config
	notifier := NewProviderNotifierFromConfig(cfg)
	s.mu.Lock()
	s.notificationSender = notifier
	s.mu.Unlock()
	return nil
}

// RebuildNotificationSenderFromDB reads saved config from DB and activates it.
// Call this once at startup after the store is initialized.
func (s *Store) RebuildNotificationSenderFromDB() {
	cfg := s.GetNotificationConfig()
	notifier := NewProviderNotifierFromConfig(cfg)
	if notifier.Enabled() {
		s.mu.Lock()
		s.notificationSender = notifier
		s.mu.Unlock()
	}
}

func defaultUsers() map[string]User {
	return map[string]User{
		"admin": {
			Username: "admin",
			Password: "admin123",
			Role:     "admin",
		},
		"seller": {
			Username: "seller",
			Password: "seller123",
			Role:     "seller",
		},
	}
}

func (s *Store) HasPermission(role, permission string) bool {
	if s.db != nil {
		var exists bool
		err := s.db.QueryRow(
			`SELECT EXISTS (
				SELECT 1 FROM role_permissions
				WHERE role = $1 AND permission = $2
			)`,
			role,
			permission,
		).Scan(&exists)
		return err == nil && exists
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	permissions, ok := s.rolePermissions[role]
	if !ok {
		return false
	}
	_, exists := permissions[permission]
	return exists
}

func (s *Store) PermissionsForRole(role string) ([]string, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT permission
			FROM role_permissions
			WHERE role = $1
			ORDER BY permission
		`, role)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		permissions := []string{}
		for rows.Next() {
			var permission string
			if err := rows.Scan(&permission); err != nil {
				return nil, err
			}
			permissions = append(permissions, permission)
		}
		return permissions, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	rolePermissions, ok := s.rolePermissions[role]
	if !ok {
		return []string{}, nil
	}
	permissions := make([]string, 0, len(rolePermissions))
	for permission := range rolePermissions {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return permissions, nil
}

func (s *Store) SessionInfo(username, role string) (UserSessionInfo, error) {
	permissions, err := s.PermissionsForRole(role)
	if err != nil {
		return UserSessionInfo{}, err
	}
	scopes, err := s.UserAccessScopes(username)
	if err != nil {
		return UserSessionInfo{}, err
	}
	return UserSessionInfo{
		User:        username,
		Role:        role,
		Permissions: permissions,
		Scopes:      scopes,
	}, nil
}

func (s *Store) UserAccessScopes(username string) (UserAccessScopes, error) {
	if s.db != nil {
		warehouseIDs, err := s.userScopeIDs("user_warehouse_scopes", username)
		if err != nil {
			return UserAccessScopes{}, err
		}
		cashboxIDs, err := s.userScopeIDs("user_cashbox_scopes", username)
		if err != nil {
			return UserAccessScopes{}, err
		}
		return UserAccessScopes{
			WarehouseIDs: warehouseIDs,
			CashboxIDs:   cashboxIDs,
		}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return UserAccessScopes{
		WarehouseIDs: sortedScopeIDs(s.userWarehouseScopes[username]),
		CashboxIDs:   sortedScopeIDs(s.userCashboxScopes[username]),
	}, nil
}

func (s *Store) userScopeIDs(table, username string) ([]int64, error) {
	idColumn := "warehouse_id"
	if table == "user_cashbox_scopes" {
		idColumn = "cashbox_id"
	}
	rows, err := s.db.Query(
		fmt.Sprintf("SELECT %s FROM %s WHERE username = $1 ORDER BY %s", idColumn, table, idColumn),
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func sortedScopeIDs(scope map[int64]struct{}) []int64 {
	if len(scope) == 0 {
		return []int64{}
	}
	ids := make([]int64, 0, len(scope))
	for id := range scope {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (s *Store) HasWarehouseAccess(username, role string, warehouseID int64) bool {
	if role == "admin" || warehouseID <= 0 {
		return true
	}
	if s.db != nil {
		return s.hasScopedAccess("user_warehouse_scopes", "warehouse_id", username, warehouseID)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return hasInMemoryScopedAccess(s.userWarehouseScopes[username], warehouseID)
}

func (s *Store) HasCashboxAccess(username, role string, cashboxID int64) bool {
	if role == "admin" || cashboxID <= 0 {
		return true
	}
	if s.db != nil {
		return s.hasScopedAccess("user_cashbox_scopes", "cashbox_id", username, cashboxID)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return hasInMemoryScopedAccess(s.userCashboxScopes[username], cashboxID)
}

func (s *Store) hasScopedAccess(table, idColumn, username string, id int64) bool {
	var scopedCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE username = $1", table)
	if err := s.db.QueryRow(countQuery, username).Scan(&scopedCount); err != nil {
		return false
	}
	if scopedCount == 0 {
		return true
	}

	var exists bool
	existsQuery := fmt.Sprintf(
		"SELECT EXISTS (SELECT 1 FROM %s WHERE username = $1 AND %s = $2)",
		table,
		idColumn,
	)
	if err := s.db.QueryRow(existsQuery, username, id).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func hasInMemoryScopedAccess(scope map[int64]struct{}, id int64) bool {
	if len(scope) == 0 {
		return true
	}
	_, ok := scope[id]
	return ok
}

func (s *Store) ValidateUser(username, password string) (User, bool) {
	if s.db != nil {
		var user User
		err := s.db.QueryRow(
			`SELECT username, password, role FROM users WHERE username = $1`,
			username,
		).Scan(&user.Username, &user.Password, &user.Role)
		if err != nil || user.Password != password {
			return User{}, false
		}
		return user, true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	if !ok || user.Password != password {
		return User{}, false
	}

	return user, true
}

func (s *Store) ListProducts() ([]Product, error) {
	return s.ListProductsFiltered("", false)
}

func (s *Store) CreateCustomer(input Customer) (Customer, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Email = strings.TrimSpace(input.Email)
	input.Comment = strings.TrimSpace(input.Comment)
	if input.Name == "" {
		return Customer{}, errors.New("customer name is required")
	}

	if s.db != nil {
		if err := s.db.QueryRow(`
			INSERT INTO customers (name, phone, email, comment)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`, input.Name, input.Phone, input.Email, input.Comment).Scan(&input.ID, &input.CreatedAt, &input.UpdatedAt); err != nil {
			return Customer{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.customerSeq++
	now := time.Now().UTC()
	input.ID = s.customerSeq
	input.CreatedAt = now
	input.UpdatedAt = now
	s.customers = append(s.customers, input)
	return input, nil
}

func (s *Store) ListCustomers(search string) ([]Customer, error) {
	search = strings.TrimSpace(search)
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, name, phone, email, comment, created_at, updated_at
			FROM customers
			WHERE ($1 = '' OR
				LOWER(name) LIKE '%' || LOWER($1) || '%' OR
				LOWER(phone) LIKE '%' || LOWER($1) || '%' OR
				LOWER(email) LIKE '%' || LOWER($1) || '%')
			ORDER BY id DESC
		`, search)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := []Customer{}
		for rows.Next() {
			var item Customer
			if err := rows.Scan(&item.ID, &item.Name, &item.Phone, &item.Email, &item.Comment, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Customer, 0, len(s.customers))
	for _, customer := range s.customers {
		if search != "" {
			needle := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(customer.Name), needle) &&
				!strings.Contains(strings.ToLower(customer.Phone), needle) &&
				!strings.Contains(strings.ToLower(customer.Email), needle) {
				continue
			}
		}
		result = append(result, customer)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

func (s *Store) CreateCustomerReminder(input CustomerReminder) (CustomerReminder, error) {
	input.Text = strings.TrimSpace(input.Text)
	if input.CustomerID <= 0 {
		return CustomerReminder{}, errors.New("customerId is required")
	}
	if input.Text == "" {
		return CustomerReminder{}, errors.New("reminder text is required")
	}
	input.Status = CustomerReminderStatusPending

	if s.db != nil {
		var exists bool
		if err := s.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM customers WHERE id = $1)`, input.CustomerID).Scan(&exists); err != nil {
			return CustomerReminder{}, err
		}
		if !exists {
			return CustomerReminder{}, errors.New("customer not found")
		}
		if err := s.db.QueryRow(`
			INSERT INTO customer_reminders (customer_id, text, due_at, status)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`, input.CustomerID, input.Text, input.DueAt, input.Status).Scan(&input.ID, &input.CreatedAt, &input.UpdatedAt); err != nil {
			return CustomerReminder{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	customerExists := false
	for _, customer := range s.customers {
		if customer.ID == input.CustomerID {
			customerExists = true
			break
		}
	}
	if !customerExists {
		return CustomerReminder{}, errors.New("customer not found")
	}
	s.customerRemSeq++
	now := time.Now().UTC()
	input.ID = s.customerRemSeq
	input.CreatedAt = now
	input.UpdatedAt = now
	s.customerReminders = append(s.customerReminders, input)
	return input, nil
}

func (s *Store) CompleteCustomerReminder(reminderID int64) (CustomerReminder, error) {
	if reminderID <= 0 {
		return CustomerReminder{}, errors.New("reminder id is required")
	}
	completedAt := time.Now().UTC()
	if s.db != nil {
		var item CustomerReminder
		if err := s.db.QueryRow(`
			UPDATE customer_reminders
			SET status = $2, completed_at = $3, updated_at = $3
			WHERE id = $1
			RETURNING id, customer_id, text, due_at, status, completed_at, created_at, updated_at
		`, reminderID, CustomerReminderStatusDone, completedAt).Scan(
			&item.ID, &item.CustomerID, &item.Text, &item.DueAt, &item.Status, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CustomerReminder{}, errors.New("customer reminder not found")
			}
			return CustomerReminder{}, err
		}
		return item, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.customerReminders {
		if s.customerReminders[i].ID == reminderID {
			s.customerReminders[i].Status = CustomerReminderStatusDone
			s.customerReminders[i].CompletedAt = &completedAt
			s.customerReminders[i].UpdatedAt = completedAt
			return s.customerReminders[i], nil
		}
	}
	return CustomerReminder{}, errors.New("customer reminder not found")
}

func (s *Store) ListCustomerReminders(customerID *int64, status *string, overdueOnly bool) ([]CustomerReminder, error) {
	statusFilter := ""
	if status != nil {
		statusFilter = strings.TrimSpace(*status)
	}

	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, customer_id, text, due_at, status, completed_at, created_at, updated_at
			FROM customer_reminders
			WHERE ($1::BIGINT IS NULL OR customer_id = $1)
			  AND ($2 = '' OR status = $2)
			  AND ($3 = FALSE OR (status = 'pending' AND due_at IS NOT NULL AND due_at < NOW()))
			ORDER BY id DESC
		`, customerID, statusFilter, overdueOnly)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []CustomerReminder{}
		for rows.Next() {
			var item CustomerReminder
			if err := rows.Scan(&item.ID, &item.CustomerID, &item.Text, &item.DueAt, &item.Status, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	now := time.Now().UTC()
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []CustomerReminder{}
	for _, reminder := range s.customerReminders {
		if customerID != nil && reminder.CustomerID != *customerID {
			continue
		}
		if statusFilter != "" && reminder.Status != statusFilter {
			continue
		}
		if overdueOnly {
			if reminder.Status != CustomerReminderStatusPending || reminder.DueAt == nil || !reminder.DueAt.Before(now) {
				continue
			}
		}
		result = append(result, reminder)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

func (s *Store) CreateServiceOrder(input ServiceOrder) (ServiceOrder, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Technician = strings.TrimSpace(input.Technician)
	input.Currency = normalizeCurrency(input.Currency)
	if input.CustomerID <= 0 {
		return ServiceOrder{}, errors.New("customerId is required")
	}
	if input.Title == "" {
		return ServiceOrder{}, errors.New("service order title is required")
	}
	if input.Price < 0 {
		return ServiceOrder{}, errors.New("service order price must be greater or equal zero")
	}
	if input.LaborMin < 0 {
		return ServiceOrder{}, errors.New("laborMin must be greater or equal zero")
	}
	if input.Currency == "" {
		input.Currency = "UAH"
	}
	input.Status = ServiceOrderStatusNew
	input.Parts = []ServiceOrderPart{}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return ServiceOrder{}, err
		}
		defer tx.Rollback()

		var customerExists bool
		if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM customers WHERE id = $1)`, input.CustomerID).Scan(&customerExists); err != nil {
			return ServiceOrder{}, err
		}
		if !customerExists {
			return ServiceOrder{}, errors.New("customer not found")
		}
		if input.ProductID != nil {
			var productExists bool
			if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, *input.ProductID).Scan(&productExists); err != nil {
				return ServiceOrder{}, err
			}
			if !productExists {
				return ServiceOrder{}, ErrProductNotFound
			}
		}
		if _, err := s.rateToUAHInTx(tx, input.Currency); err != nil {
			return ServiceOrder{}, err
		}

		if err := tx.QueryRow(`
			INSERT INTO service_orders (customer_id, product_id, title, description, technician, labor_min, status, price, parts_total, currency)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id, created_at, updated_at
		`, input.CustomerID, input.ProductID, input.Title, input.Description, input.Technician, input.LaborMin, input.Status, input.Price, input.PartsTotal, input.Currency).Scan(&input.ID, &input.CreatedAt, &input.UpdatedAt); err != nil {
			return ServiceOrder{}, err
		}

		if err := tx.Commit(); err != nil {
			return ServiceOrder{}, err
		}
		enriched, err := s.enrichServiceOrderFinanceFromDB(input)
		if err != nil {
			return ServiceOrder{}, err
		}
		return enriched, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	customerExists := false
	for _, customer := range s.customers {
		if customer.ID == input.CustomerID {
			customerExists = true
			break
		}
	}
	if !customerExists {
		return ServiceOrder{}, errors.New("customer not found")
	}
	if input.ProductID != nil && !s.productExistsLocked(*input.ProductID) {
		return ServiceOrder{}, ErrProductNotFound
	}
	if _, err := s.rateToUAHLocked(input.Currency); err != nil {
		return ServiceOrder{}, err
	}

	s.serviceOrderSeq++
	now := time.Now().UTC()
	input.ID = s.serviceOrderSeq
	input.CreatedAt = now
	input.UpdatedAt = now
	input.Total = input.Price + input.PartsTotal
	input.Debt = input.Total
	rateToUAH, err := s.rateToUAHLocked(input.Currency)
	if err != nil {
		return ServiceOrder{}, err
	}
	input.TotalUAH = input.Total * rateToUAH
	input.DebtUAH = input.TotalUAH
	s.serviceOrders = append(s.serviceOrders, input)
	return input, nil
}

func (s *Store) ListServiceOrders(customerID *int64, status *string) ([]ServiceOrder, error) {
	statusFilter := ""
	if status != nil {
		statusFilter = strings.TrimSpace(*status)
	}
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, customer_id, product_id, title, description, technician, labor_min, status, price, parts_total, currency, completed_at, created_at, updated_at
			FROM service_orders
			WHERE ($1::BIGINT IS NULL OR customer_id = $1)
			  AND ($2 = '' OR status = $2)
			ORDER BY id DESC
		`, customerID, statusFilter)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []ServiceOrder{}
		for rows.Next() {
			var item ServiceOrder
			if err := rows.Scan(&item.ID, &item.CustomerID, &item.ProductID, &item.Title, &item.Description, &item.Technician, &item.LaborMin, &item.Status, &item.Price, &item.PartsTotal, &item.Currency, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return nil, err
			}
			parts, err := s.listServiceOrderPartsDB(item.ID)
			if err != nil {
				return nil, err
			}
			item.Parts = parts
			item, err = s.enrichServiceOrderFinanceFromDB(item)
			if err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []ServiceOrder{}
	for _, order := range s.serviceOrders {
		if customerID != nil && order.CustomerID != *customerID {
			continue
		}
		if statusFilter != "" && order.Status != statusFilter {
			continue
		}
		order.Parts = s.serviceOrderPartsForLocked(order.ID)
		order = s.enrichServiceOrderFinanceLocked(order)
		result = append(result, order)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

func (s *Store) ServiceOrderByID(orderID int64) (ServiceOrder, error) {
	if orderID <= 0 {
		return ServiceOrder{}, errors.New("service order id is required")
	}
	if s.db != nil {
		var item ServiceOrder
		if err := s.db.QueryRow(`
			SELECT id, customer_id, product_id, title, description, technician, labor_min, status, price, parts_total, currency, completed_at, created_at, updated_at
			FROM service_orders
			WHERE id = $1
		`, orderID).Scan(&item.ID, &item.CustomerID, &item.ProductID, &item.Title, &item.Description, &item.Technician, &item.LaborMin, &item.Status, &item.Price, &item.PartsTotal, &item.Currency, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ServiceOrder{}, errors.New("service order not found")
			}
			return ServiceOrder{}, err
		}
		parts, err := s.listServiceOrderPartsDB(item.ID)
		if err != nil {
			return ServiceOrder{}, err
		}
		item.Parts = parts
		return s.enrichServiceOrderFinanceFromDB(item)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.serviceOrders {
		if item.ID != orderID {
			continue
		}
		item.Parts = s.serviceOrderPartsForLocked(item.ID)
		item = s.enrichServiceOrderFinanceLocked(item)
		return item, nil
	}
	return ServiceOrder{}, errors.New("service order not found")
}

func (s *Store) UpdateServiceOrderStatus(orderID int64, status string) (ServiceOrder, error) {
	status = strings.TrimSpace(status)
	if !isValidServiceOrderStatus(status) {
		return ServiceOrder{}, errors.New("invalid service order status")
	}
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return ServiceOrder{}, err
		}
		defer tx.Rollback()

		var item ServiceOrder
		if err := tx.QueryRow(`
			SELECT id, customer_id, product_id, title, description, technician, labor_min, status, price, parts_total, currency, completed_at, created_at, updated_at
			FROM service_orders
			WHERE id = $1
			FOR UPDATE
		`, orderID).Scan(&item.ID, &item.CustomerID, &item.ProductID, &item.Title, &item.Description, &item.Technician, &item.LaborMin, &item.Status, &item.Price, &item.PartsTotal, &item.Currency, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ServiceOrder{}, errors.New("service order not found")
			}
			return ServiceOrder{}, err
		}
		if err := validateServiceOrderTransition(item.Status, status); err != nil {
			if !(item.Status == ServiceOrderStatusDone && status == ServiceOrderStatusInProgress) {
				return ServiceOrder{}, err
			}
			if _, actErr := s.ServiceOrderActDocument(orderID); actErr == nil {
				return ServiceOrder{}, errors.New("скасуйте акт виконаних робіт перед повторним відкриттям")
			} else if actErr.Error() != "акт виконаних робіт не знайдено" {
				return ServiceOrder{}, actErr
			}
		}
		now := time.Now().UTC()
		var completedAt *time.Time
		if status == ServiceOrderStatusDone {
			completedAt = &now
		}
		if _, err := tx.Exec(`
			UPDATE service_orders
			SET status = $2, completed_at = $3, updated_at = $4
			WHERE id = $1
		`, orderID, status, completedAt, now); err != nil {
			return ServiceOrder{}, err
		}
		item.Status = status
		item.CompletedAt = completedAt
		item.UpdatedAt = now
		item.Parts, err = s.listServiceOrderPartsDB(item.ID)
		if err != nil {
			return ServiceOrder{}, err
		}
		if err := tx.Commit(); err != nil {
			return ServiceOrder{}, err
		}
		enriched, err := s.enrichServiceOrderFinanceFromDB(item)
		if err != nil {
			return ServiceOrder{}, err
		}
		if status == ServiceOrderStatusDone {
			_ = s.ensureServiceOrderActPosted(orderID)
		}
		return enriched, nil
	}

	result, shouldEnsureAct, err := func() (ServiceOrder, bool, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		for i := range s.serviceOrders {
			if s.serviceOrders[i].ID != orderID {
				continue
			}
			if err := validateServiceOrderTransition(s.serviceOrders[i].Status, status); err != nil {
				if !(s.serviceOrders[i].Status == ServiceOrderStatusDone && status == ServiceOrderStatusInProgress) {
					return ServiceOrder{}, false, err
				}
				if _, actErr := s.serviceOrderActDocumentLocked(orderID); actErr == nil {
					return ServiceOrder{}, false, errors.New("скасуйте акт виконаних робіт перед повторним відкриттям")
				}
			}
			now := time.Now().UTC()
			s.serviceOrders[i].Status = status
			if status == ServiceOrderStatusDone {
				s.serviceOrders[i].CompletedAt = &now
			} else {
				s.serviceOrders[i].CompletedAt = nil
			}
			s.serviceOrders[i].UpdatedAt = now
			s.serviceOrders[i].Parts = s.serviceOrderPartsForLocked(s.serviceOrders[i].ID)
			s.serviceOrders[i] = s.enrichServiceOrderFinanceLocked(s.serviceOrders[i])
			return s.serviceOrders[i], status == ServiceOrderStatusDone, nil
		}
		return ServiceOrder{}, false, errors.New("service order not found")
	}()
	if err != nil {
		return ServiceOrder{}, err
	}
	if shouldEnsureAct {
		_ = s.ensureServiceOrderActPosted(orderID)
	}
	return result, nil
}

func (s *Store) UpdateServiceOrderDetails(orderID int64, input ServiceOrder) (ServiceOrder, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Technician = strings.TrimSpace(input.Technician)
	input.Currency = normalizeCurrency(input.Currency)
	if input.Title == "" {
		return ServiceOrder{}, errors.New("service order title is required")
	}
	if input.Price < 0 {
		return ServiceOrder{}, errors.New("service order price must be greater or equal zero")
	}
	if input.LaborMin < 0 {
		return ServiceOrder{}, errors.New("laborMin must be greater or equal zero")
	}
	if input.Currency == "" {
		input.Currency = "UAH"
	}
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return ServiceOrder{}, err
		}
		defer tx.Rollback()
		var item ServiceOrder
		err = tx.QueryRow(`
			SELECT id, customer_id, product_id, title, description, technician, labor_min, status, price, parts_total, currency, completed_at, created_at, updated_at
			FROM service_orders
			WHERE id = $1
			FOR UPDATE
		`, orderID).Scan(&item.ID, &item.CustomerID, &item.ProductID, &item.Title, &item.Description, &item.Technician, &item.LaborMin, &item.Status, &item.Price, &item.PartsTotal, &item.Currency, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ServiceOrder{}, errors.New("service order not found")
		}
		if err != nil {
			return ServiceOrder{}, err
		}
		if item.Status == ServiceOrderStatusDone || item.Status == ServiceOrderStatusCancelled {
			return ServiceOrder{}, errors.New("service order is closed")
		}
		if input.ProductID != nil {
			var productExists bool
			if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, *input.ProductID).Scan(&productExists); err != nil {
				return ServiceOrder{}, err
			}
			if !productExists {
				return ServiceOrder{}, ErrProductNotFound
			}
		}
		if _, err := s.rateToUAHInTx(tx, input.Currency); err != nil {
			return ServiceOrder{}, err
		}
		now := time.Now().UTC()
		if _, err := tx.Exec(`
			UPDATE service_orders
			SET product_id = $2,
				title = $3,
				description = $4,
				technician = $5,
				labor_min = $6,
				price = $7,
				currency = $8,
				updated_at = $9
			WHERE id = $1
		`, orderID, input.ProductID, input.Title, input.Description, input.Technician, input.LaborMin, input.Price, input.Currency, now); err != nil {
			return ServiceOrder{}, err
		}
		item.ProductID = input.ProductID
		item.Title = input.Title
		item.Description = input.Description
		item.Technician = input.Technician
		item.LaborMin = input.LaborMin
		item.Price = input.Price
		item.Currency = input.Currency
		item.UpdatedAt = now
		item.Parts, err = s.listServiceOrderPartsDB(item.ID)
		if err != nil {
			return ServiceOrder{}, err
		}
		if err := tx.Commit(); err != nil {
			return ServiceOrder{}, err
		}
		enriched, err := s.enrichServiceOrderFinanceFromDB(item)
		if err != nil {
			return ServiceOrder{}, err
		}
		return enriched, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.serviceOrders {
		if s.serviceOrders[i].ID != orderID {
			continue
		}
		if s.serviceOrders[i].Status == ServiceOrderStatusDone || s.serviceOrders[i].Status == ServiceOrderStatusCancelled {
			return ServiceOrder{}, errors.New("service order is closed")
		}
		if input.ProductID != nil && !s.productExistsLocked(*input.ProductID) {
			return ServiceOrder{}, ErrProductNotFound
		}
		if _, err := s.rateToUAHLocked(input.Currency); err != nil {
			return ServiceOrder{}, err
		}
		s.serviceOrders[i].ProductID = input.ProductID
		s.serviceOrders[i].Title = input.Title
		s.serviceOrders[i].Description = input.Description
		s.serviceOrders[i].Technician = input.Technician
		s.serviceOrders[i].LaborMin = input.LaborMin
		s.serviceOrders[i].Price = input.Price
		s.serviceOrders[i].Currency = input.Currency
		s.serviceOrders[i].UpdatedAt = time.Now().UTC()
		s.serviceOrders[i].Parts = s.serviceOrderPartsForLocked(orderID)
		s.serviceOrders[i] = s.enrichServiceOrderFinanceLocked(s.serviceOrders[i])
		return s.serviceOrders[i], nil
	}
	return ServiceOrder{}, errors.New("service order not found")
}

func (s *Store) AddServiceOrderPart(orderID int64, input ServiceOrderPart) (ServiceOrder, ServiceOrderPart, error) {
	if input.ProductID <= 0 {
		return ServiceOrder{}, ServiceOrderPart{}, errors.New("productId is required")
	}
	if input.Quantity <= 0 {
		return ServiceOrder{}, ServiceOrderPart{}, errors.New("quantity must be greater than zero")
	}
	if input.Price < 0 {
		return ServiceOrder{}, ServiceOrderPart{}, errors.New("price must be greater or equal zero")
	}
	input.Total = input.Price * float64(input.Quantity)
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		defer tx.Rollback()

		var order ServiceOrder
		err = tx.QueryRow(`
			SELECT id, customer_id, product_id, title, description, technician, labor_min, status, price, parts_total, currency, completed_at, created_at, updated_at
			FROM service_orders
			WHERE id = $1
			FOR UPDATE
		`, orderID).Scan(&order.ID, &order.CustomerID, &order.ProductID, &order.Title, &order.Description, &order.Technician, &order.LaborMin, &order.Status, &order.Price, &order.PartsTotal, &order.Currency, &order.CompletedAt, &order.CreatedAt, &order.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ServiceOrder{}, ServiceOrderPart{}, errors.New("service order not found")
		}
		if err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		if order.Status != ServiceOrderStatusNew && order.Status != ServiceOrderStatusInProgress {
			return ServiceOrder{}, ServiceOrderPart{}, errors.New("cannot add parts to closed service order")
		}

		var currentStock int
		if err := tx.QueryRow(`SELECT stock FROM products WHERE id = $1 FOR UPDATE`, input.ProductID).Scan(&currentStock); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ServiceOrder{}, ServiceOrderPart{}, ErrProductNotFound
			}
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		reservedByOthers, err := s.activeReservedByOtherOrdersInTx(tx, input.ProductID, nil)
		if err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		if currentStock-reservedByOthers < input.Quantity {
			return ServiceOrder{}, ServiceOrderPart{}, ErrInsufficientStock
		}
		if _, err := tx.Exec(`UPDATE products SET stock = $1 WHERE id = $2`, currentStock-input.Quantity, input.ProductID); err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		defaultWarehouseID, err := s.defaultWarehouseIDInTx(tx)
		if err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		if err := s.adjustWarehouseStockInTx(tx, defaultWarehouseID, input.ProductID, "write_off", input.Quantity); err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		if _, err := tx.Exec(`
			INSERT INTO stock_movements (product_id, from_warehouse_id, movement_type, quantity, note)
			VALUES ($1, $2, $3, $4, $5)
		`, input.ProductID, defaultWarehouseID, "write_off", input.Quantity, fmt.Sprintf("service_order:%d part", orderID)); err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}

		if err := tx.QueryRow(`
			INSERT INTO service_order_parts (service_order_id, product_id, quantity, price, total)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`, orderID, input.ProductID, input.Quantity, input.Price, input.Total).Scan(&input.ID, &input.CreatedAt); err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		input.ServiceOrderID = orderID

		now := time.Now().UTC()
		if _, err := tx.Exec(`
			UPDATE service_orders
			SET parts_total = parts_total + $2, updated_at = $3
			WHERE id = $1
		`, orderID, input.Total, now); err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		order.PartsTotal += input.Total
		order.UpdatedAt = now
		order.Parts, err = s.listServiceOrderPartsInTx(tx, orderID)
		if err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		if err := tx.Commit(); err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		enriched, err := s.enrichServiceOrderFinanceFromDB(order)
		if err != nil {
			return ServiceOrder{}, ServiceOrderPart{}, err
		}
		return enriched, input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	orderIndex := -1
	for i := range s.serviceOrders {
		if s.serviceOrders[i].ID == orderID {
			orderIndex = i
			break
		}
	}
	if orderIndex < 0 {
		return ServiceOrder{}, ServiceOrderPart{}, errors.New("service order not found")
	}
	if s.serviceOrders[orderIndex].Status != ServiceOrderStatusNew && s.serviceOrders[orderIndex].Status != ServiceOrderStatusInProgress {
		return ServiceOrder{}, ServiceOrderPart{}, errors.New("cannot add parts to closed service order")
	}
	productIndex := -1
	for i := range s.products {
		if s.products[i].ID == input.ProductID {
			productIndex = i
			break
		}
	}
	if productIndex < 0 {
		return ServiceOrder{}, ServiceOrderPart{}, ErrProductNotFound
	}
	available := s.products[productIndex].Stock - s.activeReservedByOtherOrdersLocked(input.ProductID, nil)
	if available < input.Quantity {
		return ServiceOrder{}, ServiceOrderPart{}, ErrInsufficientStock
	}
	s.products[productIndex].Stock -= input.Quantity
	warehouseID := int64(1)
	s.adjustWarehouseStockLocked(warehouseID, input.ProductID, "write_off", input.Quantity)
	s.movementSeq++
	s.movements = append(s.movements, StockMovement{
		ID:              s.movementSeq,
		ProductID:       input.ProductID,
		FromWarehouseID: &warehouseID,
		Type:            "write_off",
		Quantity:        input.Quantity,
		Note:            fmt.Sprintf("service_order:%d part", orderID),
		CreatedAt:       time.Now().UTC(),
	})

	s.serviceOrderPartSeq++
	input.ID = s.serviceOrderPartSeq
	input.ServiceOrderID = orderID
	input.CreatedAt = time.Now().UTC()
	s.serviceOrderParts = append(s.serviceOrderParts, input)
	s.serviceOrders[orderIndex].PartsTotal += input.Total
	s.serviceOrders[orderIndex].UpdatedAt = time.Now().UTC()
	s.serviceOrders[orderIndex].Parts = s.serviceOrderPartsForLocked(orderID)
	s.serviceOrders[orderIndex] = s.enrichServiceOrderFinanceLocked(s.serviceOrders[orderIndex])
	return s.serviceOrders[orderIndex], input, nil
}

func isValidServiceOrderStatus(status string) bool {
	switch status {
	case ServiceOrderStatusNew, ServiceOrderStatusInProgress, ServiceOrderStatusDone, ServiceOrderStatusCancelled:
		return true
	default:
		return false
	}
}

func (s *Store) CreateServiceCategory(input ServiceCategory) (ServiceCategory, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return ServiceCategory{}, errors.New("service category name is required")
	}
	if s.db != nil {
		if err := s.db.QueryRow(`
			INSERT INTO service_categories (name)
			VALUES ($1)
			RETURNING id, created_at
		`, input.Name).Scan(&input.ID, &input.CreatedAt); err != nil {
			return ServiceCategory{}, err
		}
		return input, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serviceCategorySeq++
	input.ID = s.serviceCategorySeq
	input.CreatedAt = time.Now().UTC()
	s.serviceCategories = append(s.serviceCategories, input)
	return input, nil
}

func (s *Store) ListServiceCategories() ([]ServiceCategory, error) {
	if s.db != nil {
		rows, err := s.db.Query(`SELECT id, name, created_at FROM service_categories ORDER BY id ASC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []ServiceCategory{}
		for rows.Next() {
			var item ServiceCategory
			if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ServiceCategory, len(s.serviceCategories))
	copy(result, s.serviceCategories)
	return result, nil
}

func (s *Store) CreateService(input Service) (Service, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Currency = normalizeCurrency(input.Currency)
	if input.Name == "" || input.CategoryID <= 0 {
		return Service{}, errors.New("необхідно вказати категорію і назву послуги")
	}
	if input.Price < 0 || input.DurationMin < 0 {
		return Service{}, errors.New("service price and duration must be greater or equal zero")
	}
	if s.db != nil {
		if err := s.db.QueryRow(`
			INSERT INTO services (category_id, name, price, currency, duration_min)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`, input.CategoryID, input.Name, input.Price, input.Currency, input.DurationMin).Scan(&input.ID, &input.CreatedAt); err != nil {
			return Service{}, err
		}
		return input, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serviceSeq++
	input.ID = s.serviceSeq
	input.CreatedAt = time.Now().UTC()
	s.services = append(s.services, input)
	return input, nil
}

func (s *Store) ListServices() ([]Service, error) {
	if s.db != nil {
		rows, err := s.db.Query(`SELECT id, category_id, name, price, currency, duration_min, created_at FROM services ORDER BY id ASC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []Service{}
		for rows.Next() {
			var item Service
			if err := rows.Scan(&item.ID, &item.CategoryID, &item.Name, &item.Price, &item.Currency, &item.DurationMin, &item.CreatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Service, len(s.services))
	copy(result, s.services)
	return result, nil
}

func validateServiceOrderTransition(current, next string) error {
	if current == next {
		return nil
	}
	allowed := map[string]map[string]struct{}{
		ServiceOrderStatusNew: {
			ServiceOrderStatusInProgress: {},
			ServiceOrderStatusCancelled:  {},
		},
		ServiceOrderStatusInProgress: {
			ServiceOrderStatusDone:      {},
			ServiceOrderStatusCancelled: {},
		},
		ServiceOrderStatusDone:      {},
		ServiceOrderStatusCancelled: {},
	}
	if _, ok := allowed[current][next]; !ok {
		return fmt.Errorf("invalid service order status transition: %s -> %s", current, next)
	}
	return nil
}

func (s *Store) listServiceOrderPartsDB(orderID int64) ([]ServiceOrderPart, error) {
	if s.db == nil {
		return []ServiceOrderPart{}, nil
	}
	rows, err := s.db.Query(`
		SELECT id, service_order_id, product_id, quantity, price, total, created_at
		FROM service_order_parts
		WHERE service_order_id = $1
		ORDER BY id ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ServiceOrderPart{}
	for rows.Next() {
		var part ServiceOrderPart
		if err := rows.Scan(&part.ID, &part.ServiceOrderID, &part.ProductID, &part.Quantity, &part.Price, &part.Total, &part.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, part)
	}
	return result, rows.Err()
}

func (s *Store) listServiceOrderPartsInTx(tx *sql.Tx, orderID int64) ([]ServiceOrderPart, error) {
	rows, err := tx.Query(`
		SELECT id, service_order_id, product_id, quantity, price, total, created_at
		FROM service_order_parts
		WHERE service_order_id = $1
		ORDER BY id ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ServiceOrderPart{}
	for rows.Next() {
		var part ServiceOrderPart
		if err := rows.Scan(&part.ID, &part.ServiceOrderID, &part.ProductID, &part.Quantity, &part.Price, &part.Total, &part.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, part)
	}
	return result, rows.Err()
}

func (s *Store) serviceOrderPartsForLocked(orderID int64) []ServiceOrderPart {
	result := []ServiceOrderPart{}
	for _, part := range s.serviceOrderParts {
		if part.ServiceOrderID != orderID {
			continue
		}
		result = append(result, part)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *Store) enrichServiceOrderFinanceFromDB(item ServiceOrder) (ServiceOrder, error) {
	rateToUAH, err := s.rateToUAHInDB(item.Currency)
	if err != nil {
		return ServiceOrder{}, err
	}
	item.Total = item.Price + item.PartsTotal
	item.TotalUAH = item.Total * rateToUAH
	if s.db == nil {
		item.Paid = 0
		item.PaidUAH = 0
		item.Debt = item.Total
		item.DebtUAH = item.TotalUAH
		return item, nil
	}
	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0), COALESCE(SUM(amount_uah), 0)
		FROM payments
		WHERE service_order_id = $1
	`, item.ID).Scan(&item.Paid, &item.PaidUAH); err != nil {
		return ServiceOrder{}, err
	}
	item.Debt = item.Total - item.Paid
	if item.Debt < 0 {
		item.Debt = 0
	}
	item.DebtUAH = item.TotalUAH - item.PaidUAH
	if item.DebtUAH < 0 {
		item.DebtUAH = 0
	}
	return item, nil
}

func (s *Store) enrichServiceOrderFinanceLocked(item ServiceOrder) ServiceOrder {
	item.Total = item.Price + item.PartsTotal
	rateToUAH, err := s.rateToUAHLocked(item.Currency)
	if err != nil {
		return item
	}
	item.TotalUAH = item.Total * rateToUAH
	paidUAH, _ := s.paidForTargetUAHLocked(nil, nil, &item.ID)
	item.PaidUAH = paidUAH
	item.Paid = paidUAH / rateToUAH
	item.Debt = item.Total - item.Paid
	if item.Debt < 0 {
		item.Debt = 0
	}
	item.DebtUAH = item.TotalUAH - item.PaidUAH
	if item.DebtUAH < 0 {
		item.DebtUAH = 0
	}
	return item
}

func (s *Store) ListProductsFiltered(search string, includeArchived bool) ([]Product, error) {
	if s.db != nil {
		query := `
			SELECT id, name, code, sku, article, barcode, serial_number, category, brand, supplier,
				purchase_price, retail_price, wholesale_price, currency, vat_percent,
				stock, min_stock, warehouse_position, comments, archived, supplier_sku, supplier_name_ext, created_at
			FROM products
			WHERE ($1 = '' OR
				LOWER(name) LIKE '%' || LOWER($1) || '%' OR
				LOWER(sku) LIKE '%' || LOWER($1) || '%' OR
				LOWER(barcode) LIKE '%' || LOWER($1) || '%' OR
				LOWER(article) LIKE '%' || LOWER($1) || '%')
				AND ($2 OR archived = FALSE)
			ORDER BY id ASC
		`
		rows, err := s.db.Query(query, search, includeArchived)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		products := []Product{}
		for rows.Next() {
			var product Product
			if err := rows.Scan(
				&product.ID,
				&product.Name,
				&product.Code,
				&product.SKU,
				&product.Article,
				&product.Barcode,
				&product.SerialNumber,
				&product.Category,
				&product.Brand,
				&product.Supplier,
				&product.PurchasePrice,
				&product.RetailPrice,
				&product.WholesalePrice,
				&product.Currency,
				&product.VATPercent,
				&product.Stock,
				&product.MinStock,
				&product.WarehousePosition,
				&product.Comments,
				&product.Archived,
				&product.SupplierSKU,
				&product.SupplierNameExt,
				&product.CreatedAt,
			); err != nil {
				return nil, err
			}
			products = append(products, product)
		}

		return products, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Product, 0, len(s.products))
	for _, product := range s.products {
		if !includeArchived && product.Archived {
			continue
		}
		if search != "" {
			target := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(product.Name), target) &&
				!strings.Contains(strings.ToLower(product.SKU), target) &&
				!strings.Contains(strings.ToLower(product.Barcode), target) &&
				!strings.Contains(strings.ToLower(product.Article), target) {
				continue
			}
		}
		result = append(result, product)
	}
	return result, nil
}

func (s *Store) CreateProduct(input Product) (Product, error) {
	if s.db != nil {
		if input.Stock < 0 {
			input.Stock = 0
		}
		if input.Currency == "" {
			input.Currency = "UAH"
		}
		err := s.db.QueryRow(`
			INSERT INTO products
				(name, code, sku, article, barcode, serial_number, category, brand, supplier,
				purchase_price, retail_price, wholesale_price, currency, vat_percent,
				stock, min_stock, warehouse_position, comments, archived, supplier_sku, supplier_name_ext)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
			RETURNING id, created_at
		`,
			input.Name,
			input.Code,
			input.SKU,
			input.Article,
			input.Barcode,
			input.SerialNumber,
			input.Category,
			input.Brand,
			input.Supplier,
			input.PurchasePrice,
			input.RetailPrice,
			input.WholesalePrice,
			input.Currency,
			input.VATPercent,
			input.Stock,
			input.MinStock,
			input.WarehousePosition,
			input.Comments,
			input.Archived,
			input.SupplierSKU,
			input.SupplierNameExt,
		).Scan(&input.ID, &input.CreatedAt)
		if err != nil {
			return Product{}, err
		}
		if err := s.upsertDefaultWarehouseStockForProduct(input.ID, input.Stock); err != nil {
			return Product{}, err
		}

		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.productSeq++
	input.ID = s.productSeq
	input.CreatedAt = time.Now().UTC()
	if input.Stock < 0 {
		input.Stock = 0
	}
	if input.Currency == "" {
		input.Currency = "UAH"
	}

	s.products = append(s.products, input)
	s.setWarehouseStockLocked(1, input.ID, input.Stock)
	return input, nil
}

func (s *Store) UpdateProduct(id int64, input Product) (Product, error) {
	if s.db != nil {
		current, err := s.getProductByID(id)
		if err != nil {
			return Product{}, err
		}
		if input.Currency == "" {
			input.Currency = current.Currency
		}
		if input.Stock < 0 {
			input.Stock = 0
		}

		var updated Product
		err = s.db.QueryRow(`
			UPDATE products
			SET name = $2, code = $3, sku = $4, article = $5, barcode = $6, serial_number = $7,
				category = $8, brand = $9, supplier = $10, purchase_price = $11, retail_price = $12,
				wholesale_price = $13, currency = $14, vat_percent = $15, stock = $16, min_stock = $17,
				warehouse_position = $18, comments = $19, archived = $20,
				supplier_sku = $21, supplier_name_ext = $22
			WHERE id = $1
			RETURNING id, name, code, sku, article, barcode, serial_number, category, brand, supplier,
				purchase_price, retail_price, wholesale_price, currency, vat_percent, stock, min_stock,
				warehouse_position, comments, archived, supplier_sku, supplier_name_ext, created_at
		`,
			id,
			input.Name,
			input.Code,
			input.SKU,
			input.Article,
			input.Barcode,
			input.SerialNumber,
			input.Category,
			input.Brand,
			input.Supplier,
			input.PurchasePrice,
			input.RetailPrice,
			input.WholesalePrice,
			input.Currency,
			input.VATPercent,
			input.Stock,
			input.MinStock,
			input.WarehousePosition,
			input.Comments,
			input.Archived,
			input.SupplierSKU,
			input.SupplierNameExt,
		).Scan(
			&updated.ID,
			&updated.Name,
			&updated.Code,
			&updated.SKU,
			&updated.Article,
			&updated.Barcode,
			&updated.SerialNumber,
			&updated.Category,
			&updated.Brand,
			&updated.Supplier,
			&updated.PurchasePrice,
			&updated.RetailPrice,
			&updated.WholesalePrice,
			&updated.Currency,
			&updated.VATPercent,
			&updated.Stock,
			&updated.MinStock,
			&updated.WarehousePosition,
			&updated.Comments,
			&updated.Archived,
			&updated.SupplierSKU,
			&updated.SupplierNameExt,
			&updated.CreatedAt,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrProductNotFound
		}
		if err == nil {
			if syncErr := s.upsertDefaultWarehouseStockForProduct(updated.ID, updated.Stock); syncErr != nil {
				return Product{}, syncErr
			}
		}
		return updated, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.products {
		if s.products[i].ID != id {
			continue
		}
		input.ID = id
		input.CreatedAt = s.products[i].CreatedAt
		if input.Currency == "" {
			input.Currency = s.products[i].Currency
		}
		if input.Stock < 0 {
			input.Stock = 0
		}
		s.products[i] = input
		s.setWarehouseStockLocked(1, input.ID, input.Stock)
		return input, nil
	}
	return Product{}, ErrProductNotFound
}

func (s *Store) ArchiveProduct(id int64, archived bool) error {
	if s.db != nil {
		res, err := s.db.Exec(`UPDATE products SET archived = $2 WHERE id = $1`, id, archived)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrProductNotFound
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.products {
		if s.products[i].ID == id {
			s.products[i].Archived = archived
			return nil
		}
	}
	return ErrProductNotFound
}

func (s *Store) GenerateProductBarcode(id int64) (Product, error) {
	barcode := generateEAN13Barcode(id)
	if s.db != nil {
		var product Product
		err := s.db.QueryRow(`
			UPDATE products
			SET barcode = $2
			WHERE id = $1
			RETURNING id, name, code, sku, article, barcode, serial_number, category, brand, supplier,
				purchase_price, retail_price, wholesale_price, currency, vat_percent, stock, min_stock,
				warehouse_position, comments, archived, created_at
		`, id, barcode).Scan(
			&product.ID,
			&product.Name,
			&product.Code,
			&product.SKU,
			&product.Article,
			&product.Barcode,
			&product.SerialNumber,
			&product.Category,
			&product.Brand,
			&product.Supplier,
			&product.PurchasePrice,
			&product.RetailPrice,
			&product.WholesalePrice,
			&product.Currency,
			&product.VATPercent,
			&product.Stock,
			&product.MinStock,
			&product.WarehousePosition,
			&product.Comments,
			&product.Archived,
			&product.SupplierSKU,
			&product.SupplierNameExt,
			&product.CreatedAt,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrProductNotFound
		}
		return product, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.products {
		if s.products[i].ID == id {
			s.products[i].Barcode = barcode
			return s.products[i], nil
		}
	}
	return Product{}, ErrProductNotFound
}

func (s *Store) FindProductDuplicates() ([]ProductDuplicate, error) {
	products, err := s.ListProductsFiltered("", true)
	if err != nil {
		return nil, err
	}

	type item struct {
		field string
		value string
	}
	groups := map[item][]Product{}
	for _, product := range products {
		candidates := []item{
			{field: "name", value: strings.TrimSpace(strings.ToLower(product.Name))},
			{field: "barcode", value: strings.TrimSpace(strings.ToLower(product.Barcode))},
			{field: "article", value: strings.TrimSpace(strings.ToLower(product.Article))},
			{field: "serialNumber", value: strings.TrimSpace(strings.ToLower(product.SerialNumber))},
		}
		for _, candidate := range candidates {
			if candidate.value == "" {
				continue
			}
			groups[candidate] = append(groups[candidate], product)
		}
	}

	result := make([]ProductDuplicate, 0)
	for key, grouped := range groups {
		if len(grouped) < 2 {
			continue
		}
		ids := make([]int64, 0, len(grouped))
		skus := make([]string, 0, len(grouped))
		names := make([]string, 0, len(grouped))
		for _, product := range grouped {
			ids = append(ids, product.ID)
			skus = append(skus, product.SKU)
			names = append(names, product.Name)
		}
		result = append(result, ProductDuplicate{
			Field:      key.field,
			Value:      key.value,
			ProductIDs: ids,
			SKUs:       skus,
			Names:      names,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Field == result[j].Field {
			return result[i].Value < result[j].Value
		}
		return result[i].Field < result[j].Field
	})
	return result, nil
}

func (s *Store) ExportProductsCSV(includeArchived bool) (string, error) {
	products, err := s.ListProductsFiltered("", includeArchived)
	if err != nil {
		return "", err
	}

	builder := &strings.Builder{}
	writer := csv.NewWriter(builder)
	header := []string{
		"name", "code", "sku", "article", "barcode", "serialNumber",
		"category", "brand", "supplier", "purchasePrice", "retailPrice", "wholesalePrice",
		"currency", "vatPercent", "stock", "minStock", "warehousePosition", "comments", "archived",
	}
	if err := writer.Write(header); err != nil {
		return "", err
	}
	for _, product := range products {
		row := []string{
			product.Name,
			product.Code,
			product.SKU,
			product.Article,
			product.Barcode,
			product.SerialNumber,
			product.Category,
			product.Brand,
			product.Supplier,
			fmt.Sprintf("%.2f", product.PurchasePrice),
			fmt.Sprintf("%.2f", product.RetailPrice),
			fmt.Sprintf("%.2f", product.WholesalePrice),
			product.Currency,
			fmt.Sprintf("%.2f", product.VATPercent),
			strconv.Itoa(product.Stock),
			strconv.Itoa(product.MinStock),
			product.WarehousePosition,
			product.Comments,
			strconv.FormatBool(product.Archived),
		}
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func (s *Store) ExportProductsXLSX(includeArchived bool) ([]byte, error) {
	products, err := s.ListProductsFiltered("", includeArchived)
	if err != nil {
		return nil, err
	}

	file := excelize.NewFile()
	const sheet = "Products"
	file.SetSheetName("Sheet1", sheet)
	header := []string{
		"name", "code", "sku", "article", "barcode", "serialNumber",
		"category", "brand", "supplier", "purchasePrice", "retailPrice", "wholesalePrice",
		"currency", "vatPercent", "stock", "minStock", "warehousePosition", "comments", "archived",
	}
	for col, value := range header {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := file.SetCellValue(sheet, cell, value); err != nil {
			return nil, err
		}
	}
	for rowIdx, product := range products {
		row := []interface{}{
			product.Name,
			product.Code,
			product.SKU,
			product.Article,
			product.Barcode,
			product.SerialNumber,
			product.Category,
			product.Brand,
			product.Supplier,
			product.PurchasePrice,
			product.RetailPrice,
			product.WholesalePrice,
			product.Currency,
			product.VATPercent,
			product.Stock,
			product.MinStock,
			product.WarehousePosition,
			product.Comments,
			product.Archived,
		}
		for col, value := range row {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx+2)
			if err := file.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s *Store) ImportProductsCSV(csvText string, updateExisting bool) (ProductCSVImportResult, error) {
	reader := csv.NewReader(strings.NewReader(csvText))
	rows, err := reader.ReadAll()
	if err != nil {
		return ProductCSVImportResult{}, errors.New("invalid csv format")
	}
	if len(rows) < 2 {
		return ProductCSVImportResult{}, errors.New("csv must contain header and at least one row")
	}

	headerMap := map[string]int{}
	for idx, header := range rows[0] {
		headerMap[normalizeCSVHeader(header)] = idx
	}
	if _, ok := headerMap["sku"]; !ok {
		return ProductCSVImportResult{}, errors.New("csv header must include sku")
	}
	if _, ok := headerMap["name"]; !ok {
		return ProductCSVImportResult{}, errors.New("csv header must include name")
	}

	result := ProductCSVImportResult{
		Errors: make([]ProductCSVImportRowError, 0),
	}
	for i := 1; i < len(rows); i++ {
		line := i + 1
		productInput, rowErr := parseProductCSVRow(rows[i], headerMap)
		if rowErr != nil {
			result.Skipped++
			result.Errors = append(result.Errors, ProductCSVImportRowError{
				Line:  line,
				Error: rowErr.Error(),
			})
			continue
		}

		existing, exists, err := s.productBySKU(productInput.SKU)
		if err != nil {
			return ProductCSVImportResult{}, err
		}
		if exists {
			if !updateExisting {
				result.Skipped++
				continue
			}
			merged := mergeProductForCSVUpdate(existing, productInput)
			if _, err := s.UpdateProduct(existing.ID, merged); err != nil {
				result.Skipped++
				result.Errors = append(result.Errors, ProductCSVImportRowError{
					Line:  line,
					Error: err.Error(),
				})
				continue
			}
			result.Updated++
			continue
		}

		if _, err := s.CreateProduct(productInput); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, ProductCSVImportRowError{
				Line:  line,
				Error: err.Error(),
			})
			continue
		}
		result.Imported++
	}

	return result, nil
}

func (s *Store) ImportProductsXLSX(content []byte, updateExisting bool) (ProductCSVImportResult, error) {
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return ProductCSVImportResult{}, errors.New("invalid xlsx format")
	}
	sheetName := file.GetSheetName(0)
	if sheetName == "" {
		return ProductCSVImportResult{}, errors.New("xlsx file does not contain sheets")
	}
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return ProductCSVImportResult{}, err
	}
	if len(rows) < 2 {
		return ProductCSVImportResult{}, errors.New("xlsx must contain header and at least one row")
	}

	headerMap := map[string]int{}
	for idx, header := range rows[0] {
		headerMap[normalizeCSVHeader(header)] = idx
	}
	if _, ok := headerMap["sku"]; !ok {
		return ProductCSVImportResult{}, errors.New("xlsx header must include sku")
	}
	if _, ok := headerMap["name"]; !ok {
		return ProductCSVImportResult{}, errors.New("xlsx header must include name")
	}

	result := ProductCSVImportResult{
		Errors: make([]ProductCSVImportRowError, 0),
	}
	for i := 1; i < len(rows); i++ {
		line := i + 1
		productInput, rowErr := parseProductCSVRow(rows[i], headerMap)
		if rowErr != nil {
			result.Skipped++
			result.Errors = append(result.Errors, ProductCSVImportRowError{
				Line:  line,
				Error: rowErr.Error(),
			})
			continue
		}

		existing, exists, err := s.productBySKU(productInput.SKU)
		if err != nil {
			return ProductCSVImportResult{}, err
		}
		if exists {
			if !updateExisting {
				result.Skipped++
				continue
			}
			merged := mergeProductForCSVUpdate(existing, productInput)
			if _, err := s.UpdateProduct(existing.ID, merged); err != nil {
				result.Skipped++
				result.Errors = append(result.Errors, ProductCSVImportRowError{
					Line:  line,
					Error: err.Error(),
				})
				continue
			}
			result.Updated++
			continue
		}

		if _, err := s.CreateProduct(productInput); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, ProductCSVImportRowError{
				Line:  line,
				Error: err.Error(),
			})
			continue
		}
		result.Imported++
	}

	return result, nil
}

func (s *Store) BulkUpdateProductPrices(req ProductPriceBulkUpdateRequest) (ProductPriceBulkUpdateResult, error) {
	req.Mode = strings.TrimSpace(strings.ToLower(req.Mode))
	req.PriceField = strings.TrimSpace(req.PriceField)
	req.RoundMode = strings.TrimSpace(strings.ToLower(req.RoundMode))
	req.Category = strings.TrimSpace(req.Category)
	req.Brand = strings.TrimSpace(req.Brand)
	req.Supplier = strings.TrimSpace(req.Supplier)
	req.Search = strings.TrimSpace(req.Search)

	if req.Mode != "percent" && req.Mode != "fixed" {
		return ProductPriceBulkUpdateResult{}, errors.New("mode must be percent or fixed")
	}
	if req.Value == 0 {
		return ProductPriceBulkUpdateResult{}, errors.New("value must not be zero")
	}
	if req.PriceField != "retailPrice" && req.PriceField != "purchasePrice" && req.PriceField != "wholesalePrice" {
		return ProductPriceBulkUpdateResult{}, errors.New("priceField must be retailPrice, purchasePrice or wholesalePrice")
	}
	if req.RoundMode == "" {
		req.RoundMode = "none"
	}
	if req.RoundMode != "none" && req.RoundMode != "nearest" && req.RoundMode != "up" && req.RoundMode != "down" {
		return ProductPriceBulkUpdateResult{}, errors.New("roundMode must be none, nearest, up or down")
	}
	if req.RoundMode != "none" && req.RoundTo <= 0 {
		return ProductPriceBulkUpdateResult{}, errors.New("roundTo must be greater than zero when roundMode is used")
	}

	products, err := s.ListProductsFiltered(req.Search, req.IncludeArchived)
	if err != nil {
		return ProductPriceBulkUpdateResult{}, err
	}

	matches := make([]Product, 0, len(products))
	for _, product := range products {
		if req.Category != "" && !strings.EqualFold(strings.TrimSpace(product.Category), req.Category) {
			continue
		}
		if req.Brand != "" && !strings.EqualFold(strings.TrimSpace(product.Brand), req.Brand) {
			continue
		}
		if req.Supplier != "" && !strings.EqualFold(strings.TrimSpace(product.Supplier), req.Supplier) {
			continue
		}
		matches = append(matches, product)
	}

	updatedCount := 0
	for _, product := range matches {
		nextProduct := product
		current := currentPriceFieldValue(nextProduct, req.PriceField)
		next := current
		if req.Mode == "percent" {
			next = current * (1 + req.Value/100.0)
		} else {
			next = current + req.Value
		}
		if next < 0 {
			next = 0
		}
		next = roundPriceValue(next, req.RoundMode, req.RoundTo)
		setPriceFieldValue(&nextProduct, req.PriceField, next)
		if _, err := s.UpdateProduct(product.ID, nextProduct); err != nil {
			return ProductPriceBulkUpdateResult{}, err
		}
		updatedCount++
	}

	return ProductPriceBulkUpdateResult{Updated: updatedCount}, nil
}

func roundPriceValue(value float64, mode string, step float64) float64 {
	if mode == "none" || step <= 0 {
		return value
	}
	ratio := value / step
	switch mode {
	case "nearest":
		return math.Round(ratio) * step
	case "up":
		return math.Ceil(ratio) * step
	case "down":
		return math.Floor(ratio) * step
	default:
		return value
	}
}

func (s *Store) MergeDuplicateProducts(targetProductID int64, sourceProductIDs []int64) (ProductMergeResult, error) {
	if targetProductID <= 0 {
		return ProductMergeResult{}, errors.New("targetProductId is required")
	}
	sources := normalizeSourceProductIDs(targetProductID, sourceProductIDs)
	if len(sources) == 0 {
		return ProductMergeResult{}, errors.New("необхідно вказати товари для об'єднання")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return ProductMergeResult{}, err
		}
		defer tx.Rollback()

		var targetStock int
		if err := tx.QueryRow(`SELECT stock FROM products WHERE id = $1 FOR UPDATE`, targetProductID).Scan(&targetStock); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ProductMergeResult{}, ErrProductNotFound
			}
			return ProductMergeResult{}, err
		}

		totalSourceStock := 0
		for _, sourceID := range sources {
			var sourceStock int
			if err := tx.QueryRow(`SELECT stock FROM products WHERE id = $1 FOR UPDATE`, sourceID).Scan(&sourceStock); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ProductMergeResult{}, fmt.Errorf("source product not found: %d", sourceID)
				}
				return ProductMergeResult{}, err
			}
			totalSourceStock += sourceStock

			if _, err := tx.Exec(`
				INSERT INTO warehouse_stocks (warehouse_id, product_id, quantity)
				SELECT warehouse_id, $1, quantity
				FROM warehouse_stocks
				WHERE product_id = $2
				ON CONFLICT (warehouse_id, product_id)
				DO UPDATE SET quantity = warehouse_stocks.quantity + EXCLUDED.quantity, updated_at = NOW()
			`, targetProductID, sourceID); err != nil {
				return ProductMergeResult{}, err
			}
			if _, err := tx.Exec(`DELETE FROM warehouse_stocks WHERE product_id = $1`, sourceID); err != nil {
				return ProductMergeResult{}, err
			}

			if _, err := tx.Exec(`
				INSERT INTO cell_stocks (cell_id, product_id, quantity)
				SELECT cell_id, $1, quantity
				FROM cell_stocks
				WHERE product_id = $2
				ON CONFLICT (cell_id, product_id)
				DO UPDATE SET quantity = cell_stocks.quantity + EXCLUDED.quantity, updated_at = NOW()
			`, targetProductID, sourceID); err != nil {
				return ProductMergeResult{}, err
			}
			if _, err := tx.Exec(`DELETE FROM cell_stocks WHERE product_id = $1`, sourceID); err != nil {
				return ProductMergeResult{}, err
			}

			updateQueries := []string{
				`UPDATE stock_movements SET product_id = $1 WHERE product_id = $2`,
				`UPDATE stock_transfer_items SET product_id = $1 WHERE product_id = $2`,
				`UPDATE inventory_items SET product_id = $1 WHERE product_id = $2`,
				`UPDATE sale_items SET product_id = $1 WHERE product_id = $2`,
				`UPDATE customer_order_items SET product_id = $1 WHERE product_id = $2`,
				`UPDATE reservations SET product_id = $1 WHERE product_id = $2`,
				`UPDATE supplier_order_items SET product_id = $1 WHERE product_id = $2`,
				`UPDATE purchase_items SET product_id = $1 WHERE product_id = $2`,
				`UPDATE document_items SET product_id = $1 WHERE product_id = $2`,
			}
			for _, query := range updateQueries {
				if _, err := tx.Exec(query, targetProductID, sourceID); err != nil {
					return ProductMergeResult{}, err
				}
			}
		}

		if _, err := tx.Exec(`UPDATE products SET stock = $2 WHERE id = $1`, targetProductID, targetStock+totalSourceStock); err != nil {
			return ProductMergeResult{}, err
		}
		for _, sourceID := range sources {
			if _, err := tx.Exec(`DELETE FROM products WHERE id = $1`, sourceID); err != nil {
				return ProductMergeResult{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return ProductMergeResult{}, err
		}
		return ProductMergeResult{
			TargetProductID:  targetProductID,
			MergedProductIDs: sources,
		}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	targetIndex := -1
	indexByProductID := map[int64]int{}
	for i := range s.products {
		indexByProductID[s.products[i].ID] = i
		if s.products[i].ID == targetProductID {
			targetIndex = i
		}
	}
	if targetIndex == -1 {
		return ProductMergeResult{}, ErrProductNotFound
	}
	for _, sourceID := range sources {
		if _, ok := indexByProductID[sourceID]; !ok {
			return ProductMergeResult{}, fmt.Errorf("source product not found: %d", sourceID)
		}
	}

	remap := map[int64]int64{}
	for _, sourceID := range sources {
		remap[sourceID] = targetProductID
		s.products[targetIndex].Stock += s.products[indexByProductID[sourceID]].Stock
	}

	applyRemap := func(productID int64) int64 {
		if target, ok := remap[productID]; ok {
			return target
		}
		return productID
	}

	for i := range s.warehouseStocks {
		s.warehouseStocks[i].ProductID = applyRemap(s.warehouseStocks[i].ProductID)
	}
	s.warehouseStocks = mergeWarehouseStocksByKey(s.warehouseStocks)

	for i := range s.cellStocks {
		s.cellStocks[i].ProductID = applyRemap(s.cellStocks[i].ProductID)
	}
	s.cellStocks = mergeCellStocksByKey(s.cellStocks)

	for i := range s.movements {
		s.movements[i].ProductID = applyRemap(s.movements[i].ProductID)
	}
	for i := range s.transfers {
		for j := range s.transfers[i].Items {
			s.transfers[i].Items[j].ProductID = applyRemap(s.transfers[i].Items[j].ProductID)
		}
	}
	for i := range s.inventories {
		for j := range s.inventories[i].Items {
			s.inventories[i].Items[j].ProductID = applyRemap(s.inventories[i].Items[j].ProductID)
		}
	}
	for i := range s.sales {
		for j := range s.sales[i].Items {
			s.sales[i].Items[j].ProductID = applyRemap(s.sales[i].Items[j].ProductID)
		}
	}
	for i := range s.orders {
		for j := range s.orders[i].Items {
			s.orders[i].Items[j].ProductID = applyRemap(s.orders[i].Items[j].ProductID)
		}
	}
	for i := range s.reservations {
		s.reservations[i].ProductID = applyRemap(s.reservations[i].ProductID)
	}
	for i := range s.supplierOrders {
		for j := range s.supplierOrders[i].Items {
			s.supplierOrders[i].Items[j].ProductID = applyRemap(s.supplierOrders[i].Items[j].ProductID)
		}
	}
	for i := range s.purchases {
		for j := range s.purchases[i].Items {
			s.purchases[i].Items[j].ProductID = applyRemap(s.purchases[i].Items[j].ProductID)
		}
	}
	for i := range s.documents {
		for j := range s.documents[i].Items {
			s.documents[i].Items[j].ProductID = applyRemap(s.documents[i].Items[j].ProductID)
		}
	}

	nextProducts := make([]Product, 0, len(s.products)-len(sources))
	for _, product := range s.products {
		if _, remove := remap[product.ID]; remove {
			continue
		}
		nextProducts = append(nextProducts, product)
	}
	s.products = nextProducts

	return ProductMergeResult{
		TargetProductID:  targetProductID,
		MergedProductIDs: sources,
	}, nil
}

func normalizeSourceProductIDs(targetProductID int64, sourceProductIDs []int64) []int64 {
	seen := map[int64]struct{}{}
	result := make([]int64, 0, len(sourceProductIDs))
	for _, sourceID := range sourceProductIDs {
		if sourceID <= 0 || sourceID == targetProductID {
			continue
		}
		if _, exists := seen[sourceID]; exists {
			continue
		}
		seen[sourceID] = struct{}{}
		result = append(result, sourceID)
	}
	return result
}

func mergeWarehouseStocksByKey(stocks []WarehouseStock) []WarehouseStock {
	type key struct {
		WarehouseID int64
		ProductID   int64
	}
	grouped := map[key]WarehouseStock{}
	for _, stock := range stocks {
		k := key{WarehouseID: stock.WarehouseID, ProductID: stock.ProductID}
		if existing, ok := grouped[k]; ok {
			existing.Quantity += stock.Quantity
			existing.UpdatedAt = time.Now().UTC()
			grouped[k] = existing
			continue
		}
		grouped[k] = stock
	}
	result := make([]WarehouseStock, 0, len(grouped))
	for _, stock := range grouped {
		result = append(result, stock)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].WarehouseID == result[j].WarehouseID {
			return result[i].ProductID < result[j].ProductID
		}
		return result[i].WarehouseID < result[j].WarehouseID
	})
	return result
}

func mergeCellStocksByKey(stocks []CellStock) []CellStock {
	type key struct {
		CellID    int64
		ProductID int64
	}
	grouped := map[key]CellStock{}
	for _, stock := range stocks {
		k := key{CellID: stock.CellID, ProductID: stock.ProductID}
		if existing, ok := grouped[k]; ok {
			existing.Quantity += stock.Quantity
			existing.UpdatedAt = time.Now().UTC()
			grouped[k] = existing
			continue
		}
		grouped[k] = stock
	}
	result := make([]CellStock, 0, len(grouped))
	for _, stock := range grouped {
		result = append(result, stock)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CellID == result[j].CellID {
			return result[i].ProductID < result[j].ProductID
		}
		return result[i].CellID < result[j].CellID
	})
	return result
}

func currentPriceFieldValue(product Product, field string) float64 {
	switch field {
	case "purchasePrice":
		return product.PurchasePrice
	case "wholesalePrice":
		return product.WholesalePrice
	default:
		return product.RetailPrice
	}
}

func setPriceFieldValue(product *Product, field string, value float64) {
	switch field {
	case "purchasePrice":
		product.PurchasePrice = value
	case "wholesalePrice":
		product.WholesalePrice = value
	default:
		product.RetailPrice = value
	}
}

func generateEAN13Barcode(productID int64) string {
	if productID < 0 {
		productID = -productID
	}
	body := fmt.Sprintf("200%09d", productID%1_000_000_000)
	sum := 0
	for idx, char := range body {
		digit := int(char - '0')
		if idx%2 == 0 {
			sum += digit
			continue
		}
		sum += digit * 3
	}
	checkDigit := (10 - (sum % 10)) % 10
	return fmt.Sprintf("%s%d", body, checkDigit)
}

func parseProductCSVRow(row []string, header map[string]int) (Product, error) {
	get := func(key string) string {
		idx, ok := header[key]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	product := Product{
		Name:              get("name"),
		Code:              get("code"),
		SKU:               get("sku"),
		Article:           get("article"),
		Barcode:           get("barcode"),
		SerialNumber:      get("serialnumber"),
		Category:          get("category"),
		Brand:             get("brand"),
		Supplier:          get("supplier"),
		Currency:          get("currency"),
		WarehousePosition: get("warehouseposition"),
		Comments:          get("comments"),
	}

	if product.Name == "" {
		return Product{}, errors.New("name is required")
	}
	if product.SKU == "" {
		return Product{}, errors.New("sku is required")
	}
	if product.Currency == "" {
		product.Currency = "UAH"
	}

	var err error
	if product.PurchasePrice, err = csvFloat(get("purchaseprice")); err != nil {
		return Product{}, fmt.Errorf("invalid purchasePrice: %w", err)
	}
	if product.RetailPrice, err = csvFloat(get("retailprice")); err != nil {
		return Product{}, fmt.Errorf("invalid retailPrice: %w", err)
	}
	if product.WholesalePrice, err = csvFloat(get("wholesaleprice")); err != nil {
		return Product{}, fmt.Errorf("invalid wholesalePrice: %w", err)
	}
	if product.VATPercent, err = csvFloat(get("vatpercent")); err != nil {
		return Product{}, fmt.Errorf("invalid vatPercent: %w", err)
	}
	if product.Stock, err = csvInt(get("stock")); err != nil {
		return Product{}, fmt.Errorf("invalid stock: %w", err)
	}
	if product.MinStock, err = csvInt(get("minstock")); err != nil {
		return Product{}, fmt.Errorf("invalid minStock: %w", err)
	}
	if product.Archived, err = csvBool(get("archived")); err != nil {
		return Product{}, fmt.Errorf("invalid archived: %w", err)
	}

	return product, nil
}

func parsePurchaseCSVRow(row []string, header map[string]int) (PurchaseItem, error) {
	get := func(key string) string {
		idx, ok := header[key]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	productID, err := csvInt64(get("productid"))
	if err != nil || productID <= 0 {
		return PurchaseItem{}, errors.New("invalid productId")
	}
	quantity, err := csvInt(get("quantity"))
	if err != nil || quantity <= 0 {
		return PurchaseItem{}, errors.New("invalid quantity")
	}
	price, err := csvFloat(get("price"))
	if err != nil || price < 0 {
		return PurchaseItem{}, errors.New("invalid price")
	}
	return PurchaseItem{
		ProductID: productID,
		Quantity:  quantity,
		Price:     price,
	}, nil
}

func parseDocumentCSVRow(row []string, header map[string]int) (DocumentItem, error) {
	purchaseItem, err := parsePurchaseCSVRow(row, header)
	if err != nil {
		return DocumentItem{}, err
	}
	return DocumentItem{
		ProductID: purchaseItem.ProductID,
		Quantity:  purchaseItem.Quantity,
		Price:     purchaseItem.Price,
	}, nil
}

func normalizeCSVHeader(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func csvFloat(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseFloat(raw, 64)
}

func csvInt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func csvInt64(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func csvBool(raw string) (bool, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return false, nil
	}
	switch raw {
	case "1", "true", "yes", "y":
		return true, nil
	case "0", "false", "no", "n":
		return false, nil
	default:
		return false, errors.New("expected true/false")
	}
}

func mergeProductForCSVUpdate(existing Product, incoming Product) Product {
	merged := existing
	merged.Name = incoming.Name
	merged.Code = incoming.Code
	merged.SKU = incoming.SKU
	merged.Article = incoming.Article
	merged.Barcode = incoming.Barcode
	merged.SerialNumber = incoming.SerialNumber
	merged.Category = incoming.Category
	merged.Brand = incoming.Brand
	merged.Supplier = incoming.Supplier
	merged.PurchasePrice = incoming.PurchasePrice
	merged.RetailPrice = incoming.RetailPrice
	merged.WholesalePrice = incoming.WholesalePrice
	merged.Currency = incoming.Currency
	merged.VATPercent = incoming.VATPercent
	merged.Stock = incoming.Stock
	merged.MinStock = incoming.MinStock
	merged.WarehousePosition = incoming.WarehousePosition
	merged.Comments = incoming.Comments
	merged.Archived = incoming.Archived
	return merged
}

func (s *Store) getProductByID(id int64) (Product, error) {
	if s.db == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, product := range s.products {
			if product.ID == id {
				return product, nil
			}
		}
		return Product{}, ErrProductNotFound
	}

	var product Product
	err := s.db.QueryRow(`
		SELECT id, name, code, sku, article, barcode, serial_number, category, brand, supplier,
			purchase_price, retail_price, wholesale_price, currency, vat_percent,
			stock, min_stock, warehouse_position, comments, archived, supplier_sku, supplier_name_ext, created_at
		FROM products
		WHERE id = $1
	`,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Code,
		&product.SKU,
		&product.Article,
		&product.Barcode,
		&product.SerialNumber,
		&product.Category,
		&product.Brand,
		&product.Supplier,
		&product.PurchasePrice,
		&product.RetailPrice,
		&product.WholesalePrice,
		&product.Currency,
		&product.VATPercent,
		&product.Stock,
		&product.MinStock,
		&product.WarehousePosition,
		&product.Comments,
		&product.Archived,
		&product.SupplierSKU,
		&product.SupplierNameExt,
		&product.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrProductNotFound
	}
	return product, err
}

func (s *Store) productBySKU(sku string) (Product, bool, error) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return Product{}, false, nil
	}
	if s.db != nil {
		var product Product
		err := s.db.QueryRow(`
			SELECT id, name, code, sku, article, barcode, serial_number, category, brand, supplier,
				purchase_price, retail_price, wholesale_price, currency, vat_percent,
				stock, min_stock, warehouse_position, comments, archived, supplier_sku, supplier_name_ext, created_at
			FROM products
			WHERE sku = $1
		`, sku).Scan(
			&product.ID,
			&product.Name,
			&product.Code,
			&product.SKU,
			&product.Article,
			&product.Barcode,
			&product.SerialNumber,
			&product.Category,
			&product.Brand,
			&product.Supplier,
			&product.PurchasePrice,
			&product.RetailPrice,
			&product.WholesalePrice,
			&product.Currency,
			&product.VATPercent,
			&product.Stock,
			&product.MinStock,
			&product.WarehousePosition,
			&product.Comments,
			&product.Archived,
			&product.SupplierSKU,
			&product.SupplierNameExt,
			&product.CreatedAt,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, false, nil
		}
		if err != nil {
			return Product{}, false, err
		}
		return product, true, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, product := range s.products {
		if product.SKU == sku {
			return product, true, nil
		}
	}
	return Product{}, false, nil
}

func (s *Store) CreateStockMovement(movement StockMovement, warehouseID int64) (StockMovement, error) {
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return StockMovement{}, err
		}
		defer tx.Rollback()

		var currentStock int
		err = tx.QueryRow(
			`SELECT stock FROM products WHERE id = $1 FOR UPDATE`,
			movement.ProductID,
		).Scan(&currentStock)
		if errors.Is(err, sql.ErrNoRows) {
			return StockMovement{}, ErrProductNotFound
		}
		if err != nil {
			return StockMovement{}, err
		}

		switch movement.Type {
		case "incoming":
			currentStock += movement.Quantity
		case "write_off", "sale", "return_to_supplier":
			reservedByOthers, err := s.activeReservedByOtherOrdersInTx(tx, movement.ProductID, nil)
			if err != nil {
				return StockMovement{}, err
			}
			if currentStock-reservedByOthers < movement.Quantity {
				return StockMovement{}, ErrInsufficientStock
			}
			currentStock -= movement.Quantity
		default:
			return StockMovement{}, fmt.Errorf("unknown movement type: %s", movement.Type)
		}

		if _, err := tx.Exec(
			`UPDATE products SET stock = $1 WHERE id = $2`,
			currentStock,
			movement.ProductID,
		); err != nil {
			return StockMovement{}, err
		}
		if warehouseID <= 0 {
			warehouseID, err = s.defaultWarehouseIDInTx(tx)
			if err != nil {
				return StockMovement{}, err
			}
		} else {
			if err := s.ensureWarehouseExistsTx(tx, warehouseID); err != nil {
				return StockMovement{}, err
			}
		}
		if err := s.adjustWarehouseStockInTx(tx, warehouseID, movement.ProductID, movement.Type, movement.Quantity); err != nil {
			return StockMovement{}, err
		}
		movement.ToWarehouseID = &warehouseID
		if movement.Type != "incoming" {
			movement.FromWarehouseID = &warehouseID
		}

		err = tx.QueryRow(`
			INSERT INTO stock_movements (product_id, from_warehouse_id, to_warehouse_id, movement_type, quantity, note)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at
		`,
			movement.ProductID,
			movement.FromWarehouseID,
			movement.ToWarehouseID,
			movement.Type,
			movement.Quantity,
			movement.Note,
		).Scan(&movement.ID, &movement.CreatedAt)
		if err != nil {
			return StockMovement{}, err
		}

		if err := tx.Commit(); err != nil {
			return StockMovement{}, err
		}
		return movement, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for idx := range s.products {
		if s.products[idx].ID != movement.ProductID {
			continue
		}

		switch movement.Type {
		case "incoming":
			s.products[idx].Stock += movement.Quantity
			wid := warehouseID
			if wid <= 0 {
				wid = int64(1)
			}
			movement.ToWarehouseID = &wid
			s.adjustWarehouseStockLocked(wid, movement.ProductID, movement.Type, movement.Quantity)
		case "write_off", "sale", "return_to_supplier":
			available := s.products[idx].Stock - s.activeReservedByOtherOrdersLocked(movement.ProductID, nil)
			if available < movement.Quantity {
				return StockMovement{}, ErrInsufficientStock
			}
			s.products[idx].Stock -= movement.Quantity
			wid := warehouseID
			if wid <= 0 {
				wid = int64(1)
			}
			movement.FromWarehouseID = &wid
			s.adjustWarehouseStockLocked(wid, movement.ProductID, movement.Type, movement.Quantity)
		default:
			return StockMovement{}, fmt.Errorf("unknown movement type: %s", movement.Type)
		}

		s.movementSeq++
		movement.ID = s.movementSeq
		movement.CreatedAt = time.Now().UTC()
		s.movements = append(s.movements, movement)
		return movement, nil
	}

	return StockMovement{}, ErrProductNotFound
}

func (s *Store) ListStockMovements(productID, warehouseID *int64) ([]StockMovement, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, product_id, from_warehouse_id, to_warehouse_id, from_cell_id, to_cell_id,
				movement_type, quantity, note, created_at
			FROM stock_movements
			WHERE ($1::BIGINT IS NULL OR product_id = $1)
				AND ($2::BIGINT IS NULL OR from_warehouse_id = $2 OR to_warehouse_id = $2)
			ORDER BY id DESC
		`, productID, warehouseID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStockMovements(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []StockMovement{}
	for _, movement := range s.movements {
		if productID != nil && movement.ProductID != *productID {
			continue
		}
		if warehouseID != nil && !movementTouchesWarehouse(movement, *warehouseID) {
			continue
		}
		result = append(result, movement)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})
	return result, nil
}

func (s *Store) CreateOrder(
	customerName string,
	items []SaleItem,
	reserve bool,
	expiresAt *time.Time,
	currency string,
	dueDate *time.Time,
) (CustomerOrder, error) {
	if len(items) == 0 {
		return CustomerOrder{}, errors.New("необхідно додати хоча б один товар до замовлення")
	}

	status := OrderStatusDraft
	if reserve {
		status = OrderStatusConfirmed
	}
	currency = normalizeCurrency(currency)

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return CustomerOrder{}, err
		}
		defer tx.Rollback()

		for _, item := range items {
			if item.Quantity <= 0 {
				return CustomerOrder{}, errors.New("кількість товару повинна бути більше нуля")
			}
			if reserve {
				available, err := s.availableStockInTx(tx, item.ProductID)
				if err != nil {
					return CustomerOrder{}, err
				}
				if available < item.Quantity {
					return CustomerOrder{}, ErrInsufficientStock
				}
			}
		}
		total := 0.0
		for _, item := range items {
			total += float64(item.Quantity) * item.Price
		}
		rateToUAH, err := s.rateToUAHInTx(tx, currency)
		if err != nil {
			return CustomerOrder{}, err
		}

		order := CustomerOrder{
			CustomerName: customerName,
			Status:       status,
			Currency:     currency,
			Total:        total,
			TotalUAH:     total * rateToUAH,
			DueDate:      dueDate,
			Items:        items,
		}
		if err := tx.QueryRow(`
			INSERT INTO customer_orders (customer_name, status, currency, total, total_uah, due_date)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at, updated_at
		`,
			order.CustomerName,
			order.Status,
			order.Currency,
			order.Total,
			order.TotalUAH,
			order.DueDate,
		).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return CustomerOrder{}, err
		}

		for _, item := range items {
			if _, err := tx.Exec(`
				INSERT INTO customer_order_items (order_id, product_id, quantity, price)
				VALUES ($1, $2, $3, $4)
			`,
				order.ID,
				item.ProductID,
				item.Quantity,
				item.Price,
			); err != nil {
				return CustomerOrder{}, err
			}

			if reserve {
				if _, err := tx.Exec(`
					INSERT INTO reservations (order_id, product_id, quantity, status, expires_at)
					VALUES ($1, $2, $3, $4, $5)
				`,
					order.ID,
					item.ProductID,
					item.Quantity,
					ReservationStatusActive,
					expiresAt,
				); err != nil {
					return CustomerOrder{}, err
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return CustomerOrder{}, err
		}
		return order, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	positions := map[int64]int{}
	for index := range s.products {
		positions[s.products[index].ID] = index
	}

	for _, item := range items {
		pos, ok := positions[item.ProductID]
		if !ok {
			return CustomerOrder{}, ErrProductNotFound
		}
		if reserve {
			available := s.products[pos].Stock - s.activeReservedForProductLocked(item.ProductID)
			if available < item.Quantity {
				return CustomerOrder{}, ErrInsufficientStock
			}
		}
	}
	total := 0.0
	for _, item := range items {
		total += float64(item.Quantity) * item.Price
	}
	rateToUAH, err := s.rateToUAHLocked(currency)
	if err != nil {
		return CustomerOrder{}, err
	}

	s.orderSeq++
	now := time.Now().UTC()
	order := CustomerOrder{
		ID:           s.orderSeq,
		CustomerName: customerName,
		Status:       status,
		Currency:     currency,
		Total:        total,
		TotalUAH:     total * rateToUAH,
		DueDate:      dueDate,
		Items:        items,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.orders = append(s.orders, order)

	if reserve {
		for _, item := range items {
			s.reservationSeq++
			s.reservations = append(s.reservations, Reservation{
				ID:        s.reservationSeq,
				OrderID:   order.ID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Status:    ReservationStatusActive,
				ExpiresAt: expiresAt,
				CreatedAt: now,
			})
		}
	}

	return order, nil
}

func (s *Store) ListOrders() ([]CustomerOrder, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, customer_name, status, currency, total, total_uah, due_date, created_at, updated_at
			FROM customer_orders
			ORDER BY id DESC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		orders := []CustomerOrder{}
		for rows.Next() {
			var order CustomerOrder
			if err := rows.Scan(
				&order.ID,
				&order.CustomerName,
				&order.Status,
				&order.Currency,
				&order.Total,
				&order.TotalUAH,
				&order.DueDate,
				&order.CreatedAt,
				&order.UpdatedAt,
			); err != nil {
				return nil, err
			}

			orderItems, err := s.orderItemsByOrderID(order.ID)
			if err != nil {
				return nil, err
			}
			order.Items = orderItems
			orders = append(orders, order)
		}
		return orders, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]CustomerOrder, len(s.orders))
	copy(orders, s.orders)
	return orders, nil
}

func (s *Store) CreateSupplier(input Supplier) (Supplier, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Supplier{}, errors.New("supplier name is required")
	}

	if s.db != nil {
		err := s.db.QueryRow(`
			INSERT INTO suppliers (name, contact, phone, email, comments)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`,
			input.Name,
			input.Contact,
			input.Phone,
			input.Email,
			input.Comments,
		).Scan(&input.ID, &input.CreatedAt)
		return input, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.supplierSeq++
	input.ID = s.supplierSeq
	input.CreatedAt = time.Now().UTC()
	s.suppliers = append(s.suppliers, input)
	return input, nil
}

func (s *Store) ListSuppliers() ([]Supplier, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, name, contact, phone, email, comments, created_at
			FROM suppliers
			ORDER BY id ASC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		suppliers := []Supplier{}
		for rows.Next() {
			var item Supplier
			if err := rows.Scan(
				&item.ID,
				&item.Name,
				&item.Contact,
				&item.Phone,
				&item.Email,
				&item.Comments,
				&item.CreatedAt,
			); err != nil {
				return nil, err
			}
			suppliers = append(suppliers, item)
		}
		return suppliers, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Supplier, len(s.suppliers))
	copy(result, s.suppliers)
	return result, nil
}

func (s *Store) CreateSupplierOrder(
	supplierID int64,
	items []PurchaseItem,
	currency string,
) (SupplierOrder, error) {
	if supplierID <= 0 {
		return SupplierOrder{}, errors.New("supplierId is required")
	}
	currency = normalizeCurrency(currency)

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return SupplierOrder{}, err
		}
		defer tx.Rollback()

		var supplierExists bool
		if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM suppliers WHERE id = $1)`, supplierID).Scan(&supplierExists); err != nil {
			return SupplierOrder{}, err
		}
		if !supplierExists {
			return SupplierOrder{}, errors.New("supplier not found")
		}

		total := 0.0
		for _, item := range items {
			if item.ProductID <= 0 {
				return SupplierOrder{}, errors.New("productId is required")
			}
			if item.Quantity <= 0 {
				return SupplierOrder{}, errors.New("supplier order item quantity must be greater than zero")
			}
			if item.Price < 0 {
				return SupplierOrder{}, errors.New("supplier order item price must be greater or equal zero")
			}
			var productExists bool
			if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, item.ProductID).Scan(&productExists); err != nil {
				return SupplierOrder{}, err
			}
			if !productExists {
				return SupplierOrder{}, ErrProductNotFound
			}
			total += float64(item.Quantity) * item.Price
		}

		rateToUAH, err := s.rateToUAHInTx(tx, currency)
		if err != nil {
			return SupplierOrder{}, err
		}

		order := SupplierOrder{
			SupplierID: supplierID,
			Status:     SupplierOrderStatusDraft,
			Currency:   currency,
			Total:      total,
			TotalUAH:   total * rateToUAH,
			Items:      items,
		}
		if err := tx.QueryRow(`
			INSERT INTO supplier_orders (supplier_id, status, currency, total, total_uah)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at, updated_at
		`,
			order.SupplierID,
			order.Status,
			order.Currency,
			order.Total,
			order.TotalUAH,
		).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return SupplierOrder{}, err
		}

		for _, item := range items {
			if _, err := tx.Exec(`
				INSERT INTO supplier_order_items (supplier_order_id, product_id, quantity, price)
				VALUES ($1, $2, $3, $4)
			`, order.ID, item.ProductID, item.Quantity, item.Price); err != nil {
				return SupplierOrder{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return SupplierOrder{}, err
		}
		return order, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	supplierExists := false
	for _, supplier := range s.suppliers {
		if supplier.ID == supplierID {
			supplierExists = true
			break
		}
	}
	if !supplierExists {
		return SupplierOrder{}, errors.New("supplier not found")
	}

	productIndex := map[int64]int{}
	for i := range s.products {
		productIndex[s.products[i].ID] = i
	}

	total := 0.0
	for _, item := range items {
		if item.ProductID <= 0 {
			return SupplierOrder{}, errors.New("productId is required")
		}
		if item.Quantity <= 0 {
			return SupplierOrder{}, errors.New("supplier order item quantity must be greater than zero")
		}
		if item.Price < 0 {
			return SupplierOrder{}, errors.New("supplier order item price must be greater or equal zero")
		}
		if _, ok := productIndex[item.ProductID]; !ok {
			return SupplierOrder{}, ErrProductNotFound
		}
		total += float64(item.Quantity) * item.Price
	}

	rateToUAH, err := s.rateToUAHLocked(currency)
	if err != nil {
		return SupplierOrder{}, err
	}

	now := time.Now().UTC()
	s.supplierOrderSeq++
	order := SupplierOrder{
		ID:         s.supplierOrderSeq,
		SupplierID: supplierID,
		Status:     SupplierOrderStatusDraft,
		Currency:   currency,
		Total:      total,
		TotalUAH:   total * rateToUAH,
		Items:      items,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.supplierOrders = append(s.supplierOrders, order)
	return order, nil
}

func (s *Store) ListSupplierOrders() ([]SupplierOrder, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, supplier_id, customer_order_id, status, currency, total, total_uah, created_at, updated_at
			FROM supplier_orders
			ORDER BY id DESC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		orders := []SupplierOrder{}
		for rows.Next() {
			var item SupplierOrder
			if err := rows.Scan(
				&item.ID,
				&item.SupplierID,
				&item.CustomerOrderID,
				&item.Status,
				&item.Currency,
				&item.Total,
				&item.TotalUAH,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {
				return nil, err
			}
			orderItems, err := s.supplierOrderItemsByOrderID(item.ID)
			if err != nil {
				return nil, err
			}
			item.Items = orderItems
			orders = append(orders, item)
		}
		return orders, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]SupplierOrder, len(s.supplierOrders))
	copy(orders, s.supplierOrders)
	return orders, nil
}

func (s *Store) UpdateSupplierOrderStatus(orderID int64, status string) error {
	if !isValidSupplierOrderStatus(status) {
		return fmt.Errorf("invalid supplier order status: %s", status)
	}

	if s.db != nil {
		var currentStatus string
		if err := s.db.QueryRow(`
			SELECT status
			FROM supplier_orders
			WHERE id = $1
		`, orderID).Scan(&currentStatus); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("supplier order not found")
			}
			return err
		}
		if err := validateManualSupplierOrderTransition(currentStatus, status); err != nil {
			return err
		}

		if status == SupplierOrderStatusReceived {
			order, err := s.getSupplierOrderByID(orderID)
			if err != nil {
				return err
			}
			pendingItems, err := s.pendingItemsForSupplierOrder(order)
			if err != nil {
				return err
			}
			if len(pendingItems) > 0 {
				return errors.New("cannot mark supplier order as received while pending items exist")
			}
		}

		result, err := s.db.Exec(`
			UPDATE supplier_orders
			SET status = $2, updated_at = NOW()
			WHERE id = $1
		`, orderID, status)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("supplier order not found")
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	orderIndex := -1
	for i := range s.supplierOrders {
		if s.supplierOrders[i].ID == orderID {
			orderIndex = i
			break
		}
	}
	if orderIndex == -1 {
		return errors.New("supplier order not found")
	}
	if err := validateManualSupplierOrderTransition(s.supplierOrders[orderIndex].Status, status); err != nil {
		return err
	}

	if status == SupplierOrderStatusReceived {
		pending := s.pendingItemsForSupplierOrderLocked(orderID)
		if len(pending) > 0 {
			return errors.New("cannot mark supplier order as received while pending items exist")
		}
	}

	s.supplierOrders[orderIndex].Status = status
	s.supplierOrders[orderIndex].UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) CreatePurchase(input Purchase) (Purchase, error) {
	if input.SupplierID <= 0 {
		return Purchase{}, errors.New("supplierId is required")
	}
	if len(input.Items) == 0 {
		return Purchase{}, errors.New("необхідно додати хоча б один товар до закупівлі")
	}
	input.Currency = normalizeCurrency(input.Currency)

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Purchase{}, err
		}
		defer tx.Rollback()

		var supplierExists bool
		if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM suppliers WHERE id = $1)`, input.SupplierID).Scan(&supplierExists); err != nil {
			return Purchase{}, err
		}
		if !supplierExists {
			return Purchase{}, errors.New("supplier not found")
		}

		if input.SupplierOrderID != nil {
			var orderSupplierID int64
			var orderStatus string
			if err := tx.QueryRow(`
				SELECT supplier_id, status
				FROM supplier_orders
				WHERE id = $1
				FOR UPDATE
			`, *input.SupplierOrderID).Scan(&orderSupplierID, &orderStatus); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return Purchase{}, errors.New("supplier order not found")
				}
				return Purchase{}, err
			}
			if orderSupplierID != input.SupplierID {
				return Purchase{}, errors.New("supplier order belongs to another supplier")
			}
			if orderStatus == SupplierOrderStatusCancelled {
				return Purchase{}, errors.New("cannot receive cancelled supplier order")
			}
			if err := s.validateSupplierOrderReceiptTx(tx, *input.SupplierOrderID, input.Items); err != nil {
				return Purchase{}, err
			}
		}

		total := 0.0
		for _, item := range input.Items {
			if item.ProductID <= 0 {
				return Purchase{}, errors.New("productId is required")
			}
			if item.Quantity <= 0 {
				return Purchase{}, errors.New("purchase item quantity must be greater than zero")
			}
			if item.Price < 0 {
				return Purchase{}, errors.New("purchase item price must be greater or equal zero")
			}
			var productID int64
			if err := tx.QueryRow(`SELECT id FROM products WHERE id = $1 FOR UPDATE`, item.ProductID).Scan(&productID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return Purchase{}, ErrProductNotFound
				}
				return Purchase{}, err
			}
			total += float64(item.Quantity) * item.Price
		}

		rateToUAH, err := s.rateToUAHInTx(tx, input.Currency)
		if err != nil {
			return Purchase{}, err
		}
		input.Total = total
		input.TotalUAH = total * rateToUAH

		if err := tx.QueryRow(`
			INSERT INTO purchases (supplier_id, supplier_order_id, currency, total, total_uah, note)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at
		`,
			input.SupplierID,
			input.SupplierOrderID,
			input.Currency,
			input.Total,
			input.TotalUAH,
			input.Note,
		).Scan(&input.ID, &input.CreatedAt); err != nil {
			return Purchase{}, err
		}
		defaultWarehouseID, err := s.defaultWarehouseIDInTx(tx)
		if err != nil {
			return Purchase{}, err
		}

		for _, item := range input.Items {
			if _, err := tx.Exec(`
				INSERT INTO purchase_items (purchase_id, product_id, quantity, price)
				VALUES ($1, $2, $3, $4)
			`, input.ID, item.ProductID, item.Quantity, item.Price); err != nil {
				return Purchase{}, err
			}
			if _, err := tx.Exec(`
				UPDATE products
				SET stock = stock + $2
				WHERE id = $1
			`, item.ProductID, item.Quantity); err != nil {
				return Purchase{}, err
			}
			if err := s.adjustWarehouseStockInTx(tx, defaultWarehouseID, item.ProductID, "incoming", item.Quantity); err != nil {
				return Purchase{}, err
			}
			if _, err := tx.Exec(`
				INSERT INTO stock_movements (product_id, from_warehouse_id, to_warehouse_id, movement_type, quantity, note)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, item.ProductID, nil, defaultWarehouseID, "incoming", item.Quantity, fmt.Sprintf("purchase_id=%d", input.ID)); err != nil {
				return Purchase{}, err
			}
		}

		if input.SupplierOrderID != nil {
			if err := s.refreshSupplierOrderReceiptStatusTx(tx, *input.SupplierOrderID); err != nil {
				return Purchase{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return Purchase{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	supplierExists := false
	for _, supplier := range s.suppliers {
		if supplier.ID == input.SupplierID {
			supplierExists = true
			break
		}
	}
	if !supplierExists {
		return Purchase{}, errors.New("supplier not found")
	}

	if input.SupplierOrderID != nil {
		orderIndex := -1
		for i := range s.supplierOrders {
			if s.supplierOrders[i].ID == *input.SupplierOrderID {
				orderIndex = i
				break
			}
		}
		if orderIndex == -1 {
			return Purchase{}, errors.New("supplier order not found")
		}
		if s.supplierOrders[orderIndex].SupplierID != input.SupplierID {
			return Purchase{}, errors.New("supplier order belongs to another supplier")
		}
		if s.supplierOrders[orderIndex].Status == SupplierOrderStatusCancelled {
			return Purchase{}, errors.New("cannot receive cancelled supplier order")
		}
		if err := s.validateSupplierOrderReceiptLocked(*input.SupplierOrderID, input.Items); err != nil {
			return Purchase{}, err
		}
	}

	productIndex := map[int64]int{}
	for i := range s.products {
		productIndex[s.products[i].ID] = i
	}

	total := 0.0
	for _, item := range input.Items {
		if item.ProductID <= 0 {
			return Purchase{}, errors.New("productId is required")
		}
		if item.Quantity <= 0 {
			return Purchase{}, errors.New("purchase item quantity must be greater than zero")
		}
		if item.Price < 0 {
			return Purchase{}, errors.New("purchase item price must be greater or equal zero")
		}
		if _, ok := productIndex[item.ProductID]; !ok {
			return Purchase{}, ErrProductNotFound
		}
		total += float64(item.Quantity) * item.Price
	}

	rateToUAH, err := s.rateToUAHLocked(input.Currency)
	if err != nil {
		return Purchase{}, err
	}
	input.Total = total
	input.TotalUAH = total * rateToUAH

	s.purchaseSeq++
	input.ID = s.purchaseSeq
	input.CreatedAt = time.Now().UTC()
	s.purchases = append(s.purchases, input)

	for _, item := range input.Items {
		s.products[productIndex[item.ProductID]].Stock += item.Quantity
		defaultWarehouseID := int64(1)
		s.adjustWarehouseStockLocked(defaultWarehouseID, item.ProductID, "incoming", item.Quantity)
		s.movementSeq++
		s.movements = append(s.movements, StockMovement{
			ID:            s.movementSeq,
			ProductID:     item.ProductID,
			ToWarehouseID: &defaultWarehouseID,
			Type:          "incoming",
			Quantity:      item.Quantity,
			Note:          fmt.Sprintf("purchase_id=%d", input.ID),
			CreatedAt:     input.CreatedAt,
		})
	}

	if input.SupplierOrderID != nil {
		s.refreshSupplierOrderReceiptStatusLocked(*input.SupplierOrderID)
	}

	return input, nil
}

func (s *Store) ImportPurchaseCSV(input Purchase, csvText string) (PurchaseCSVImportResult, error) {
	reader := csv.NewReader(strings.NewReader(csvText))
	rows, err := reader.ReadAll()
	if err != nil {
		return PurchaseCSVImportResult{}, errors.New("invalid csv format")
	}
	if len(rows) < 2 {
		return PurchaseCSVImportResult{}, errors.New("csv must contain header and at least one row")
	}

	headerMap := map[string]int{}
	for idx, header := range rows[0] {
		headerMap[normalizeCSVHeader(header)] = idx
	}
	if _, ok := headerMap["productid"]; !ok {
		return PurchaseCSVImportResult{}, errors.New("csv header must include productId")
	}
	if _, ok := headerMap["quantity"]; !ok {
		return PurchaseCSVImportResult{}, errors.New("csv header must include quantity")
	}

	result := PurchaseCSVImportResult{
		Errors: []ProductCSVImportRowError{},
	}
	items := []PurchaseItem{}
	for i := 1; i < len(rows); i++ {
		line := i + 1
		item, rowErr := parsePurchaseCSVRow(rows[i], headerMap)
		if rowErr != nil {
			result.Skipped++
			result.Errors = append(result.Errors, ProductCSVImportRowError{
				Line:  line,
				Error: rowErr.Error(),
			})
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return result, errors.New("csv contains no valid purchase rows")
	}

	input.Items = items
	purchase, err := s.CreatePurchase(input)
	if err != nil {
		return PurchaseCSVImportResult{}, err
	}
	result.Purchase = purchase
	result.Imported = len(items)
	return result, nil
}

func (s *Store) ListPurchases(supplierOrderID *int64) ([]Purchase, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, supplier_id, supplier_order_id, currency, total, total_uah, note, created_at
			FROM purchases
			WHERE ($1::BIGINT IS NULL OR supplier_order_id = $1)
			ORDER BY id DESC
		`, supplierOrderID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		items := []Purchase{}
		for rows.Next() {
			var purchase Purchase
			if err := rows.Scan(
				&purchase.ID,
				&purchase.SupplierID,
				&purchase.SupplierOrderID,
				&purchase.Currency,
				&purchase.Total,
				&purchase.TotalUAH,
				&purchase.Note,
				&purchase.CreatedAt,
			); err != nil {
				return nil, err
			}
			purchaseItems, err := s.purchaseItemsByPurchaseID(purchase.ID)
			if err != nil {
				return nil, err
			}
			purchase.Items = purchaseItems
			items = append(items, purchase)
		}
		return items, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []Purchase{}
	for _, purchase := range s.purchases {
		if supplierOrderID != nil {
			if purchase.SupplierOrderID == nil || *purchase.SupplierOrderID != *supplierOrderID {
				continue
			}
		}
		result = append(result, purchase)
	}
	return result, nil
}

func (s *Store) ReceiveSupplierOrderByLines(
	orderID int64,
	currency string,
	lines []SupplierOrderReceiveLine,
	note string,
) (Purchase, error) {
	if orderID <= 0 {
		return Purchase{}, errors.New("supplierOrderId is required")
	}
	if len(lines) == 0 {
		return Purchase{}, errors.New("необхідно вказати рядки отримання")
	}

	order, err := s.getSupplierOrderByID(orderID)
	if err != nil {
		return Purchase{}, err
	}
	if order.Status == SupplierOrderStatusCancelled {
		return Purchase{}, errors.New("cannot receive cancelled supplier order")
	}
	if order.Status == SupplierOrderStatusReceived {
		return Purchase{}, errors.New("supplier order already fully received")
	}

	priceByProduct := map[int64]float64{}
	for _, item := range order.Items {
		priceByProduct[item.ProductID] = item.Price
	}

	items := make([]PurchaseItem, 0, len(lines))
	for _, line := range lines {
		if line.ProductID <= 0 {
			return Purchase{}, errors.New("productId is required")
		}
		if line.Quantity <= 0 {
			return Purchase{}, errors.New("receive line quantity must be greater than zero")
		}
		price, ok := priceByProduct[line.ProductID]
		if !ok {
			return Purchase{}, errors.New("product is not part of supplier order")
		}
		if line.Price != nil {
			price = *line.Price
		}
		items = append(items, PurchaseItem{
			ProductID: line.ProductID,
			Quantity:  line.Quantity,
			Price:     price,
		})
	}

	if currency == "" {
		currency = order.Currency
	}
	return s.CreatePurchase(Purchase{
		SupplierID:      order.SupplierID,
		SupplierOrderID: &order.ID,
		Currency:        currency,
		Items:           items,
		Note:            note,
	})
}

func (s *Store) ListSupplierOrdersPending() ([]SupplierOrderPendingSummary, error) {
	orders, err := s.ListSupplierOrders()
	if err != nil {
		return nil, err
	}

	summaries := []SupplierOrderPendingSummary{}
	for _, order := range orders {
		if order.Status == SupplierOrderStatusCancelled || order.Status == SupplierOrderStatusReceived {
			continue
		}
		pendingItems, err := s.pendingItemsForSupplierOrder(order)
		if err != nil {
			return nil, err
		}
		if len(pendingItems) == 0 {
			continue
		}

		pendingTotal := 0.0
		for _, item := range pendingItems {
			pendingTotal += float64(item.Pending) * item.Price
		}
		rateToUAH, err := s.rateToUAH(order.Currency)
		if err != nil {
			return nil, err
		}

		summaries = append(summaries, SupplierOrderPendingSummary{
			OrderID:         order.ID,
			SupplierID:      order.SupplierID,
			Currency:        order.Currency,
			Status:          order.Status,
			PendingItems:    pendingItems,
			PendingTotal:    pendingTotal,
			PendingTotalUAH: pendingTotal * rateToUAH,
			UpdatedAt:       order.UpdatedAt,
		})
	}
	return summaries, nil
}

func (s *Store) ListPurchaseRecommendations(limit int) ([]PurchaseRecommendation, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT
				p.id,
				p.name,
				p.sku,
				p.supplier,
				p.stock,
				p.min_stock,
				COALESCE((
					SELECT SUM(r.quantity)
					FROM reservations r
					WHERE r.product_id = p.id AND r.status = $1
				), 0) AS reserved_qty,
				COALESCE((
					SELECT SUM(si.quantity)
					FROM sale_items si
					JOIN sales s ON s.id = si.sale_id
					WHERE si.product_id = p.id
					  AND s.created_at >= NOW() - INTERVAL '30 days'
				), 0) AS sold_last_30_days,
				(
					SELECT sup.id
					FROM suppliers sup
					WHERE LOWER(sup.name) = LOWER(p.supplier)
					LIMIT 1
				) AS suggested_supplier_id
			FROM products p
			WHERE p.archived = FALSE
			ORDER BY p.id ASC
		`, ReservationStatusActive)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := []PurchaseRecommendation{}
		for rows.Next() {
			var rec PurchaseRecommendation
			var suggestedSupplierID sql.NullInt64
			if err := rows.Scan(
				&rec.ProductID,
				&rec.ProductName,
				&rec.SKU,
				&rec.Supplier,
				&rec.CurrentStock,
				&rec.MinStock,
				&rec.Reserved,
				&rec.SoldLast30Days,
				&suggestedSupplierID,
			); err != nil {
				return nil, err
			}
			if suggestedSupplierID.Valid {
				rec.SuggestedSupplierID = &suggestedSupplierID.Int64
			}
			rec.Available = rec.CurrentStock - rec.Reserved
			rec.RecommendedQty = recommendedPurchaseQty(rec.Available, rec.MinStock, rec.SoldLast30Days)
			if rec.RecommendedQty <= 0 {
				continue
			}
			result = append(result, rec)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		sort.Slice(result, func(i, j int) bool {
			if result[i].RecommendedQty == result[j].RecommendedQty {
				return result[i].ProductID < result[j].ProductID
			}
			return result[i].RecommendedQty > result[j].RecommendedQty
		})
		if len(result) > limit {
			result = result[:limit]
		}
		return result, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []PurchaseRecommendation{}
	for _, product := range s.products {
		if product.Archived {
			continue
		}
		reserved := s.activeReservedForProductLocked(product.ID)
		available := product.Stock - reserved
		soldLast30Days := s.soldQuantityLast30DaysLocked(product.ID)
		rec := PurchaseRecommendation{
			ProductID:      product.ID,
			ProductName:    product.Name,
			SKU:            product.SKU,
			SupplierSKU:    product.SupplierSKU,
			Supplier:       product.Supplier,
			CurrentStock:   product.Stock,
			Reserved:       reserved,
			Available:      available,
			MinStock:       product.MinStock,
			SoldLast30Days: soldLast30Days,
		}
		if supplierID, ok := s.supplierIDByNameLocked(product.Supplier); ok {
			rec.SuggestedSupplierID = &supplierID
		}
		rec.RecommendedQty = recommendedPurchaseQty(rec.Available, rec.MinStock, rec.SoldLast30Days)
		if rec.RecommendedQty <= 0 {
			continue
		}
		result = append(result, rec)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RecommendedQty == result[j].RecommendedQty {
			return result[i].ProductID < result[j].ProductID
		}
		return result[i].RecommendedQty > result[j].RecommendedQty
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *Store) CreateSupplierOrderFromRecommendations(
	supplierID int64,
	currency string,
	lines []PurchaseRecommendationOrderLine,
) (SupplierOrder, error) {
	if supplierID <= 0 {
		return SupplierOrder{}, errors.New("supplierId is required")
	}
	if len(lines) == 0 {
		return SupplierOrder{}, errors.New("необхідно додати хоча б один рекомендований товар")
	}

	items := make([]PurchaseItem, 0, len(lines))
	for _, line := range lines {
		if line.ProductID <= 0 {
			return SupplierOrder{}, errors.New("productId is required")
		}
		if line.Quantity <= 0 {
			return SupplierOrder{}, errors.New("recommendation item quantity must be greater than zero")
		}
		price := 0.0
		if line.Price != nil {
			price = *line.Price
		} else {
			var err error
			price, err = s.productPurchasePrice(line.ProductID)
			if err != nil {
				return SupplierOrder{}, err
			}
		}
		items = append(items, PurchaseItem{
			ProductID: line.ProductID,
			Quantity:  line.Quantity,
			Price:     price,
		})
	}
	return s.CreateSupplierOrder(supplierID, items, currency)
}

func (s *Store) ListPurchaseRecommendationsGrouped(limit int) ([]PurchaseRecommendationGroup, error) {
	recommendations, err := s.ListPurchaseRecommendations(limit)
	if err != nil {
		return nil, err
	}

	type groupAgg struct {
		supplierID   *int64
		supplierName string
		items        []PurchaseRecommendation
		total        int
	}
	groupsMap := map[string]*groupAgg{}
	for _, rec := range recommendations {
		key := fmt.Sprintf("supplier:%s", strings.TrimSpace(strings.ToLower(rec.Supplier)))
		if rec.SuggestedSupplierID != nil {
			key = fmt.Sprintf("id:%d", *rec.SuggestedSupplierID)
		}
		group, ok := groupsMap[key]
		if !ok {
			group = &groupAgg{
				supplierID:   rec.SuggestedSupplierID,
				supplierName: rec.Supplier,
				items:        []PurchaseRecommendation{},
			}
			if strings.TrimSpace(group.supplierName) == "" {
				group.supplierName = "Unknown Supplier"
			}
			groupsMap[key] = group
		}
		group.items = append(group.items, rec)
		group.total += rec.RecommendedQty
	}

	result := []PurchaseRecommendationGroup{}
	for _, group := range groupsMap {
		sort.Slice(group.items, func(i, j int) bool {
			if group.items[i].RecommendedQty == group.items[j].RecommendedQty {
				return group.items[i].ProductID < group.items[j].ProductID
			}
			return group.items[i].RecommendedQty > group.items[j].RecommendedQty
		})
		result = append(result, PurchaseRecommendationGroup{
			SupplierID:       group.supplierID,
			SupplierName:     group.supplierName,
			Items:            group.items,
			TotalRecommended: group.total,
			ProductsCount:    len(group.items),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalRecommended == result[j].TotalRecommended {
			return result[i].SupplierName < result[j].SupplierName
		}
		return result[i].TotalRecommended > result[j].TotalRecommended
	})
	return result, nil
}

func (s *Store) CreateSupplierOrdersBulkFromRecommendations(
	requests []PurchaseRecommendationCreateOrderRequest,
) ([]SupplierOrder, error) {
	if len(requests) == 0 {
		return nil, errors.New("необхідно вказати замовлення")
	}
	orders := make([]SupplierOrder, 0, len(requests))
	for _, request := range requests {
		order, err := s.CreateSupplierOrderFromRecommendations(
			request.SupplierID,
			request.Currency,
			request.Items,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (s *Store) UpdateOrderStatus(orderID int64, status string) error {
	if !isValidOrderStatus(status) {
		return fmt.Errorf("invalid order status: %s", status)
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`
			UPDATE customer_orders
			SET status = $2, updated_at = NOW()
			WHERE id = $1
		`,
			orderID,
			status,
		); err != nil {
			return err
		}

		if status == OrderStatusCancelled {
			if _, err := tx.Exec(`
				UPDATE reservations
				SET status = $2, released_at = NOW()
				WHERE order_id = $1 AND status = $3
			`,
				orderID,
				ReservationStatusReleased,
				ReservationStatusActive,
			); err != nil {
				return err
			}
		}

		return tx.Commit()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.orders {
		if s.orders[i].ID != orderID {
			continue
		}
		s.orders[i].Status = status
		s.orders[i].UpdatedAt = time.Now().UTC()
		if status == OrderStatusCancelled {
			releasedAt := time.Now().UTC()
			for j := range s.reservations {
				if s.reservations[j].OrderID == orderID && s.reservations[j].Status == ReservationStatusActive {
					s.reservations[j].Status = ReservationStatusReleased
					s.reservations[j].ReleasedAt = &releasedAt
				}
			}
		}
		return nil
	}
	return fmt.Errorf("order not found")
}

func (s *Store) UpdateOrder(orderID int64, customerName string, currency string, dueDate *time.Time, items []SaleItem) (*CustomerOrder, error) {
	if customerName == "" {
		return nil, errors.New("customer name is required")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`
			UPDATE customer_orders
			SET customer_name = $2, currency = $3, due_date = $4, updated_at = NOW()
			WHERE id = $1
		`, orderID, customerName, currency, dueDate); err != nil {
			return nil, err
		}

		if _, err := tx.Exec(`DELETE FROM customer_order_items WHERE order_id = $1`, orderID); err != nil {
			return nil, err
		}

		var total float64
		for _, item := range items {
			if _, err := tx.Exec(`
				INSERT INTO customer_order_items (order_id, product_id, quantity, price)
				VALUES ($1, $2, $3, $4)
			`, orderID, item.ProductID, item.Quantity, item.Price); err != nil {
				return nil, err
			}
			total += float64(item.Quantity) * item.Price
		}

		if _, err := tx.Exec(`
			UPDATE customer_orders SET total = $2, total_uah = $2, updated_at = NOW() WHERE id = $1
		`, orderID, total); err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}

		var order CustomerOrder
		if err := s.db.QueryRow(`
			SELECT id, customer_name, status, currency, total, total_uah, due_date, created_at, updated_at
			FROM customer_orders WHERE id = $1
		`, orderID).Scan(
			&order.ID, &order.CustomerName, &order.Status, &order.Currency,
			&order.Total, &order.TotalUAH, &order.DueDate, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orderItems, err := s.orderItemsByOrderID(order.ID)
		if err != nil {
			return nil, err
		}
		order.Items = orderItems
		return &order, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.orders {
		if s.orders[i].ID != orderID {
			continue
		}
		s.orders[i].CustomerName = customerName
		s.orders[i].Currency = currency
		s.orders[i].DueDate = dueDate
		s.orders[i].Items = items
		s.orders[i].UpdatedAt = time.Now().UTC()
		o := s.orders[i]
		return &o, nil
	}
	return nil, errors.New("order not found")
}

func (s *Store) UpdateSupplierOrderCustomerLink(supplierOrderID int64, customerOrderID *int64) error {
	if s.db != nil {
		_, err := s.db.Exec(`
			UPDATE supplier_orders SET customer_order_id = $2, updated_at = NOW() WHERE id = $1
		`, supplierOrderID, customerOrderID)
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.supplierOrders {
		if s.supplierOrders[i].ID == supplierOrderID {
			s.supplierOrders[i].CustomerOrderID = customerOrderID
			s.supplierOrders[i].UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("supplier order not found")
}

func (s *Store) ListReservations(status string) ([]Reservation, error) {
	if s.db != nil {
		if status == "" {
			rows, err := s.db.Query(`
				SELECT id, order_id, product_id, quantity, status, expires_at, released_at, created_at
				FROM reservations
				ORDER BY id DESC
			`)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			return scanReservations(rows)
		}

		rows, err := s.db.Query(`
			SELECT id, order_id, product_id, quantity, status, expires_at, released_at, created_at
			FROM reservations
			WHERE status = $1
			ORDER BY id DESC
		`, status)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanReservations(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []Reservation{}
	for _, reservation := range s.reservations {
		if status != "" && reservation.Status != status {
			continue
		}
		result = append(result, reservation)
	}
	return result, nil
}

func (s *Store) ReleaseExpiredReservations(asOf time.Time) (int, error) {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	if s.db != nil {
		res, err := s.db.Exec(`
			UPDATE reservations
			SET status = $2, released_at = $3
			WHERE status = $1
				AND expires_at IS NOT NULL
				AND expires_at <= $3
		`, ReservationStatusActive, ReservationStatusExpired, asOf)
		if err != nil {
			return 0, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		return int(affected), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	released := 0
	for index := range s.reservations {
		reservation := &s.reservations[index]
		if reservation.Status != ReservationStatusActive || reservation.ExpiresAt == nil {
			continue
		}
		if reservation.ExpiresAt.After(asOf) {
			continue
		}
		releasedAt := asOf
		reservation.Status = ReservationStatusExpired
		reservation.ReleasedAt = &releasedAt
		released++
	}
	return released, nil
}

func (s *Store) releaseExpiredReservationsTx(tx *sql.Tx, asOf time.Time) (int, error) {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	res, err := tx.Exec(`
		UPDATE reservations
		SET status = $2, released_at = $3
		WHERE status = $1
			AND expires_at IS NOT NULL
			AND expires_at <= $3
	`, ReservationStatusActive, ReservationStatusExpired, asOf)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

func (s *Store) releaseExpiredReservationsLocked(asOf time.Time) int {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	released := 0
	for index := range s.reservations {
		reservation := &s.reservations[index]
		if reservation.Status != ReservationStatusActive || reservation.ExpiresAt == nil {
			continue
		}
		if reservation.ExpiresAt.After(asOf) {
			continue
		}
		releasedAt := asOf
		reservation.Status = ReservationStatusExpired
		reservation.ReleasedAt = &releasedAt
		released++
	}
	return released
}

func (s *Store) CreateWarehouse(input Warehouse) (Warehouse, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Warehouse{}, errors.New("warehouse name is required")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Warehouse{}, err
		}
		defer tx.Rollback()

		if err := tx.QueryRow(`
			INSERT INTO warehouses (name, is_virtual, location_type)
			VALUES ($1, $2, $3)
			RETURNING id, created_at
		`, input.Name, input.IsVirtual, input.LocationType).Scan(&input.ID, &input.CreatedAt); err != nil {
			return Warehouse{}, err
		}
		var zoneID int64
		if err := tx.QueryRow(`
			INSERT INTO warehouse_zones (warehouse_id, name)
			VALUES ($1, 'DEFAULT')
			RETURNING id
		`, input.ID).Scan(&zoneID); err != nil {
			return Warehouse{}, err
		}
		if _, err := tx.Exec(`
			INSERT INTO warehouse_cells (warehouse_id, zone_id, code)
			VALUES ($1, $2, 'MAIN')
		`, input.ID, zoneID); err != nil {
			return Warehouse{}, err
		}
		if err := tx.Commit(); err != nil {
			return Warehouse{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.warehouseSeq++
	input.ID = s.warehouseSeq
	input.CreatedAt = time.Now().UTC()
	s.warehouses = append(s.warehouses, input)
	s.zoneSeq++
	zoneID := s.zoneSeq
	s.warehouseZones = append(s.warehouseZones, WarehouseZone{
		ID:          zoneID,
		WarehouseID: input.ID,
		Name:        "DEFAULT",
		CreatedAt:   input.CreatedAt,
	})
	s.cellSeq++
	s.warehouseCells = append(s.warehouseCells, WarehouseCell{
		ID:          s.cellSeq,
		WarehouseID: input.ID,
		ZoneID:      zoneID,
		Code:        "MAIN",
		CreatedAt:   input.CreatedAt,
	})
	return input, nil
}

// DefaultWarehouseID returns the ID of the default warehouse, creating it if necessary.
func (s *Store) DefaultWarehouseID() (int64, error) {
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		id, err := s.defaultWarehouseIDInTx(tx)
		if err != nil {
			return 0, err
		}
		return id, tx.Commit()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.warehouses {
		if w.Name == defaultWarehouseName {
			return w.ID, nil
		}
	}
	return 1, nil
}

func (s *Store) ListWarehouses() ([]Warehouse, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, name, is_virtual, location_type, created_at
			FROM warehouses
			ORDER BY id ASC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := []Warehouse{}
		for rows.Next() {
			var item Warehouse
			if err := rows.Scan(&item.ID, &item.Name, &item.IsVirtual, &item.LocationType, &item.CreatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Warehouse, len(s.warehouses))
	copy(result, s.warehouses)
	return result, nil
}

func (s *Store) CreateWarehouseZone(input WarehouseZone) (WarehouseZone, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.WarehouseID <= 0 {
		return WarehouseZone{}, errors.New("warehouseId is required")
	}
	if input.Name == "" {
		return WarehouseZone{}, errors.New("zone name is required")
	}

	if s.db != nil {
		if err := s.db.QueryRow(`
			INSERT INTO warehouse_zones (warehouse_id, name)
			VALUES ($1, $2)
			RETURNING id, created_at
		`, input.WarehouseID, input.Name).Scan(&input.ID, &input.CreatedAt); err != nil {
			return WarehouseZone{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.warehouseExistsLocked(input.WarehouseID) {
		return WarehouseZone{}, errors.New("warehouse not found")
	}
	s.zoneSeq++
	input.ID = s.zoneSeq
	input.CreatedAt = time.Now().UTC()
	s.warehouseZones = append(s.warehouseZones, input)
	return input, nil
}

func (s *Store) ListWarehouseZones(warehouseID *int64) ([]WarehouseZone, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, warehouse_id, name, created_at
			FROM warehouse_zones
			WHERE ($1::BIGINT IS NULL OR warehouse_id = $1)
			ORDER BY warehouse_id ASC, id ASC
		`, warehouseID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []WarehouseZone{}
		for rows.Next() {
			var item WarehouseZone
			if err := rows.Scan(&item.ID, &item.WarehouseID, &item.Name, &item.CreatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []WarehouseZone{}
	for _, item := range s.warehouseZones {
		if warehouseID != nil && item.WarehouseID != *warehouseID {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Store) CreateWarehouseCell(input WarehouseCell) (WarehouseCell, error) {
	input.Code = strings.TrimSpace(input.Code)
	if input.WarehouseID <= 0 || input.ZoneID <= 0 {
		return WarehouseCell{}, errors.New("необхідно вказати склад і зону")
	}
	if input.Code == "" {
		return WarehouseCell{}, errors.New("cell code is required")
	}

	if s.db != nil {
		if err := s.db.QueryRow(`
			INSERT INTO warehouse_cells (warehouse_id, zone_id, code)
			VALUES ($1, $2, $3)
			RETURNING id, created_at
		`, input.WarehouseID, input.ZoneID, input.Code).Scan(&input.ID, &input.CreatedAt); err != nil {
			return WarehouseCell{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.warehouseExistsLocked(input.WarehouseID) {
		return WarehouseCell{}, errors.New("warehouse not found")
	}
	zoneFound := false
	for _, zone := range s.warehouseZones {
		if zone.ID == input.ZoneID && zone.WarehouseID == input.WarehouseID {
			zoneFound = true
			break
		}
	}
	if !zoneFound {
		return WarehouseCell{}, errors.New("zone not found")
	}
	s.cellSeq++
	input.ID = s.cellSeq
	input.CreatedAt = time.Now().UTC()
	s.warehouseCells = append(s.warehouseCells, input)
	return input, nil
}

func (s *Store) ListWarehouseCells(zoneID *int64) ([]WarehouseCell, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, warehouse_id, zone_id, code, created_at
			FROM warehouse_cells
			WHERE ($1::BIGINT IS NULL OR zone_id = $1)
			ORDER BY warehouse_id ASC, zone_id ASC, id ASC
		`, zoneID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []WarehouseCell{}
		for rows.Next() {
			var item WarehouseCell
			if err := rows.Scan(&item.ID, &item.WarehouseID, &item.ZoneID, &item.Code, &item.CreatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []WarehouseCell{}
	for _, item := range s.warehouseCells {
		if zoneID != nil && item.ZoneID != *zoneID {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Store) ListCellStocks(cellID, productID *int64) ([]CellStock, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT cell_id, product_id, quantity, updated_at
			FROM cell_stocks
			WHERE ($1::BIGINT IS NULL OR cell_id = $1)
				AND ($2::BIGINT IS NULL OR product_id = $2)
			ORDER BY cell_id ASC, product_id ASC
		`, cellID, productID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []CellStock{}
		for rows.Next() {
			var item CellStock
			if err := rows.Scan(&item.CellID, &item.ProductID, &item.Quantity, &item.UpdatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []CellStock{}
	for _, item := range s.cellStocks {
		if cellID != nil && item.CellID != *cellID {
			continue
		}
		if productID != nil && item.ProductID != *productID {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Store) ListWarehouseStocks(warehouseID, productID *int64) ([]WarehouseStock, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT warehouse_id, product_id, quantity, updated_at
			FROM warehouse_stocks
			WHERE ($1::BIGINT IS NULL OR warehouse_id = $1)
				AND ($2::BIGINT IS NULL OR product_id = $2)
			ORDER BY warehouse_id ASC, product_id ASC
		`, warehouseID, productID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := []WarehouseStock{}
		for rows.Next() {
			var item WarehouseStock
			if err := rows.Scan(&item.WarehouseID, &item.ProductID, &item.Quantity, &item.UpdatedAt); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []WarehouseStock{}
	for _, stock := range s.warehouseStocks {
		if warehouseID != nil && stock.WarehouseID != *warehouseID {
			continue
		}
		if productID != nil && stock.ProductID != *productID {
			continue
		}
		result = append(result, stock)
	}
	return result, nil
}

func (s *Store) CreateStockTransfer(input StockTransfer) (StockTransfer, error) {
	if input.FromWarehouseID <= 0 || input.ToWarehouseID <= 0 {
		return StockTransfer{}, errors.New("необхідно вказати склад-джерело і склад-призначення")
	}
	if len(input.Items) == 0 {
		return StockTransfer{}, errors.New("необхідно додати хоча б один товар для переміщення")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return StockTransfer{}, err
		}
		defer tx.Rollback()

		if err := s.ensureWarehouseExistsTx(tx, input.FromWarehouseID); err != nil {
			return StockTransfer{}, err
		}
		if err := s.ensureWarehouseExistsTx(tx, input.ToWarehouseID); err != nil {
			return StockTransfer{}, err
		}

		resolvedItems := make([]StockTransferItem, 0, len(input.Items))
		for _, item := range input.Items {
			if item.Quantity <= 0 {
				return StockTransfer{}, errors.New("transfer item quantity must be greater than zero")
			}
			fromCellID := item.FromCellID
			toCellID := item.ToCellID
			if fromCellID == nil {
				defaultFromCellID, err := s.defaultCellIDInTx(tx, input.FromWarehouseID)
				if err != nil {
					return StockTransfer{}, err
				}
				fromCellID = &defaultFromCellID
			}
			if toCellID == nil {
				defaultToCellID, err := s.defaultCellIDInTx(tx, input.ToWarehouseID)
				if err != nil {
					return StockTransfer{}, err
				}
				toCellID = &defaultToCellID
			}
			if err := s.ensureCellInWarehouseTx(tx, *fromCellID, input.FromWarehouseID); err != nil {
				return StockTransfer{}, err
			}
			if err := s.ensureCellInWarehouseTx(tx, *toCellID, input.ToWarehouseID); err != nil {
				return StockTransfer{}, err
			}
			if input.FromWarehouseID == input.ToWarehouseID && *fromCellID == *toCellID {
				return StockTransfer{}, errors.New("source and destination cells must differ")
			}
			if err := s.transferCellStockInTx(tx, *fromCellID, *toCellID, item.ProductID, item.Quantity); err != nil {
				return StockTransfer{}, err
			}
			if input.FromWarehouseID != input.ToWarehouseID {
				if err := s.transferWarehouseStockInTx(tx, input.FromWarehouseID, input.ToWarehouseID, item.ProductID, item.Quantity); err != nil {
					return StockTransfer{}, err
				}
			}
			resolvedItems = append(resolvedItems, StockTransferItem{
				ProductID:  item.ProductID,
				Quantity:   item.Quantity,
				FromCellID: fromCellID,
				ToCellID:   toCellID,
			})
		}
		input.Items = resolvedItems

		if err := tx.QueryRow(`
			INSERT INTO stock_transfers (from_warehouse_id, to_warehouse_id, note)
			VALUES ($1, $2, $3)
			RETURNING id, created_at
		`, input.FromWarehouseID, input.ToWarehouseID, input.Note).Scan(&input.ID, &input.CreatedAt); err != nil {
			return StockTransfer{}, err
		}

		for _, item := range input.Items {
			if _, err := tx.Exec(`
				INSERT INTO stock_transfer_items (transfer_id, product_id, from_cell_id, to_cell_id, quantity)
				VALUES ($1, $2, $3, $4, $5)
			`, input.ID, item.ProductID, item.FromCellID, item.ToCellID, item.Quantity); err != nil {
				return StockTransfer{}, err
			}
			if _, err := tx.Exec(`
				INSERT INTO stock_movements (product_id, from_warehouse_id, to_warehouse_id, from_cell_id, to_cell_id, movement_type, quantity, note)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, item.ProductID, input.FromWarehouseID, input.ToWarehouseID, item.FromCellID, item.ToCellID, "transfer", item.Quantity, fmt.Sprintf("transfer_id=%d", input.ID)); err != nil {
				return StockTransfer{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return StockTransfer{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.warehouseExistsLocked(input.FromWarehouseID) || !s.warehouseExistsLocked(input.ToWarehouseID) {
		return StockTransfer{}, errors.New("warehouse not found")
	}
	resolvedItems := make([]StockTransferItem, 0, len(input.Items))
	for _, item := range input.Items {
		if item.Quantity <= 0 {
			return StockTransfer{}, errors.New("transfer item quantity must be greater than zero")
		}
		fromCellID := item.FromCellID
		toCellID := item.ToCellID
		if fromCellID == nil {
			defaultFromCellID := s.defaultCellIDLocked(input.FromWarehouseID)
			fromCellID = &defaultFromCellID
		}
		if toCellID == nil {
			defaultToCellID := s.defaultCellIDLocked(input.ToWarehouseID)
			toCellID = &defaultToCellID
		}
		if !s.cellInWarehouseLocked(*fromCellID, input.FromWarehouseID) || !s.cellInWarehouseLocked(*toCellID, input.ToWarehouseID) {
			return StockTransfer{}, errors.New("cell not found in warehouse")
		}
		if input.FromWarehouseID == input.ToWarehouseID && *fromCellID == *toCellID {
			return StockTransfer{}, errors.New("source and destination cells must differ")
		}
		if err := s.transferCellStockLocked(*fromCellID, *toCellID, item.ProductID, item.Quantity); err != nil {
			return StockTransfer{}, err
		}
		if input.FromWarehouseID != input.ToWarehouseID {
			if err := s.transferWarehouseStockLocked(input.FromWarehouseID, input.ToWarehouseID, item.ProductID, item.Quantity); err != nil {
				return StockTransfer{}, err
			}
		}
		resolvedItems = append(resolvedItems, StockTransferItem{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			FromCellID: fromCellID,
			ToCellID:   toCellID,
		})
	}
	input.Items = resolvedItems

	s.transferSeq++
	input.ID = s.transferSeq
	input.CreatedAt = time.Now().UTC()
	s.transfers = append(s.transfers, input)
	for _, item := range input.Items {
		s.movementSeq++
		fromID := input.FromWarehouseID
		toID := input.ToWarehouseID
		s.movements = append(s.movements, StockMovement{
			ID:              s.movementSeq,
			ProductID:       item.ProductID,
			FromWarehouseID: &fromID,
			ToWarehouseID:   &toID,
			FromCellID:      item.FromCellID,
			ToCellID:        item.ToCellID,
			Type:            "transfer",
			Quantity:        item.Quantity,
			Note:            fmt.Sprintf("transfer_id=%d", input.ID),
			CreatedAt:       input.CreatedAt,
		})
	}
	return input, nil
}

func (s *Store) CreateCellTransfer(
	fromCellID int64,
	toCellID int64,
	items []StockTransferItem,
	note string,
) (StockTransfer, error) {
	if fromCellID <= 0 || toCellID <= 0 {
		return StockTransfer{}, errors.New("fromCellId and toCellId are required")
	}
	if fromCellID == toCellID {
		return StockTransfer{}, errors.New("source and destination cells must differ")
	}
	if len(items) == 0 {
		return StockTransfer{}, errors.New("необхідно додати хоча б один товар для переміщення")
	}

	fromWarehouseID, err := s.warehouseIDByCellID(fromCellID)
	if err != nil {
		return StockTransfer{}, err
	}
	toWarehouseID, err := s.warehouseIDByCellID(toCellID)
	if err != nil {
		return StockTransfer{}, err
	}

	normalized := make([]StockTransferItem, 0, len(items))
	for _, item := range items {
		if item.ProductID <= 0 {
			return StockTransfer{}, errors.New("productId is required")
		}
		if item.Quantity <= 0 {
			return StockTransfer{}, errors.New("transfer item quantity must be greater than zero")
		}
		fromID := fromCellID
		toID := toCellID
		normalized = append(normalized, StockTransferItem{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			FromCellID: &fromID,
			ToCellID:   &toID,
		})
	}

	return s.CreateStockTransfer(StockTransfer{
		FromWarehouseID: fromWarehouseID,
		ToWarehouseID:   toWarehouseID,
		Items:           normalized,
		Note:            note,
	})
}

func (s *Store) CreateCellTransferFIFO(
	toCellID int64,
	items []CellTransferFIFOItem,
	note string,
) (StockTransfer, error) {
	if toCellID <= 0 {
		return StockTransfer{}, errors.New("toCellId is required")
	}
	if len(items) == 0 {
		return StockTransfer{}, errors.New("необхідно додати хоча б один товар для переміщення")
	}

	warehouseID, err := s.warehouseIDByCellID(toCellID)
	if err != nil {
		return StockTransfer{}, err
	}

	resolved := []StockTransferItem{}
	for _, item := range items {
		if item.ProductID <= 0 {
			return StockTransfer{}, errors.New("productId is required")
		}
		if item.Quantity <= 0 {
			return StockTransfer{}, errors.New("transfer item quantity must be greater than zero")
		}

		allocations, err := s.allocateCellsFIFO(warehouseID, toCellID, item.ProductID, item.Quantity)
		if err != nil {
			return StockTransfer{}, err
		}
		for _, allocation := range allocations {
			fromID := allocation.cellID
			toID := toCellID
			resolved = append(resolved, StockTransferItem{
				ProductID:  item.ProductID,
				Quantity:   allocation.quantity,
				FromCellID: &fromID,
				ToCellID:   &toID,
			})
		}
	}

	return s.CreateStockTransfer(StockTransfer{
		FromWarehouseID: warehouseID,
		ToWarehouseID:   warehouseID,
		Items:           resolved,
		Note:            note,
	})
}

func (s *Store) CreateInventory(input Inventory) (Inventory, error) {
	if input.WarehouseID <= 0 {
		return Inventory{}, errors.New("warehouseId is required")
	}
	if len(input.Items) == 0 {
		return Inventory{}, errors.New("необхідно додати хоча б один товар до інвентаризації")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Inventory{}, err
		}
		defer tx.Rollback()

		if err := s.ensureWarehouseExistsTx(tx, input.WarehouseID); err != nil {
			return Inventory{}, err
		}

		input.Status = InventoryStatusDraft
		if err := tx.QueryRow(`
			INSERT INTO inventories (warehouse_id, status, note)
			VALUES ($1, $2, $3)
			RETURNING id, created_at
		`, input.WarehouseID, input.Status, input.Note).Scan(&input.ID, &input.CreatedAt); err != nil {
			return Inventory{}, err
		}

		for idx := range input.Items {
			systemQty, err := s.warehouseStockQuantityInTx(tx, input.WarehouseID, input.Items[idx].ProductID, true)
			if err != nil {
				return Inventory{}, err
			}
			input.Items[idx].SystemQuantity = systemQty
			input.Items[idx].Adjustment = input.Items[idx].ActualQuantity - systemQty
			if _, err := tx.Exec(`
				INSERT INTO inventory_items (inventory_id, product_id, system_quantity, actual_quantity, adjustment)
				VALUES ($1, $2, $3, $4, $5)
			`, input.ID, input.Items[idx].ProductID, input.Items[idx].SystemQuantity, input.Items[idx].ActualQuantity, input.Items[idx].Adjustment); err != nil {
				return Inventory{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return Inventory{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.warehouseExistsLocked(input.WarehouseID) {
		return Inventory{}, errors.New("warehouse not found")
	}
	input.Status = InventoryStatusDraft
	s.inventorySeq++
	input.ID = s.inventorySeq
	input.CreatedAt = time.Now().UTC()
	for idx := range input.Items {
		systemQty := s.warehouseStockQuantityLocked(input.WarehouseID, input.Items[idx].ProductID)
		input.Items[idx].SystemQuantity = systemQty
		input.Items[idx].Adjustment = input.Items[idx].ActualQuantity - systemQty
	}
	s.inventories = append(s.inventories, input)
	return input, nil
}

func (s *Store) ListInventories() ([]Inventory, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, warehouse_id, status, note, applied_at, created_at
			FROM inventories
			ORDER BY id DESC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := []Inventory{}
		for rows.Next() {
			var item Inventory
			if err := rows.Scan(&item.ID, &item.WarehouseID, &item.Status, &item.Note, &item.AppliedAt, &item.CreatedAt); err != nil {
				return nil, err
			}
			invItems, err := s.inventoryItemsByInventoryID(item.ID)
			if err != nil {
				return nil, err
			}
			item.Items = invItems
			result = append(result, item)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Inventory, len(s.inventories))
	copy(result, s.inventories)
	return result, nil
}

func (s *Store) CreateDocument(input Document) (Document, error) {
	input.Type = strings.TrimSpace(input.Type)
	input.Note = strings.TrimSpace(input.Note)
	input.Currency = normalizeCurrency(input.Currency)
	if !isValidDocumentType(input.Type) {
		return Document{}, errors.New("invalid document type")
	}
	if isStockDocumentType(input.Type) && input.WarehouseID == nil {
		return Document{}, errors.New("warehouseId is required for stock document")
	}
	if isCashDocumentType(input.Type) && input.CashboxID == nil {
		return Document{}, errors.New("cashboxId is required for cash document")
	}
	if isStockDocumentType(input.Type) && len(input.Items) == 0 {
		return Document{}, errors.New("необхідно додати хоча б один рядок до документа")
	}

	total := 0.0
	for _, item := range input.Items {
		if item.ProductID <= 0 {
			return Document{}, errors.New("productId is required")
		}
		if item.Quantity <= 0 {
			return Document{}, errors.New("document item quantity must be greater than zero")
		}
		if item.Price < 0 {
			return Document{}, errors.New("document item price must be greater or equal zero")
		}
		total += float64(item.Quantity) * item.Price
	}
	if total > 0 {
		input.Total = total
	}
	if input.Total <= 0 {
		return Document{}, errors.New("document total must be greater than zero")
	}

	now := time.Now().UTC()
	input.Status = DocumentStatusDraft
	input.Number = newDocumentNumber(input.Type, now)

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Document{}, err
		}
		defer tx.Rollback()

		if input.WarehouseID != nil {
			if err := s.ensureWarehouseExistsTx(tx, *input.WarehouseID); err != nil {
				return Document{}, err
			}
		}
		if input.CashboxID != nil {
			var cashboxCurrency string
			if err := tx.QueryRow(`
				SELECT currency
				FROM cashboxes
				WHERE id = $1
			`, *input.CashboxID).Scan(&cashboxCurrency); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return Document{}, errors.New("cashbox not found")
				}
				return Document{}, err
			}
			cashboxCurrency = normalizeCurrency(cashboxCurrency)
			if input.Currency == "" {
				input.Currency = cashboxCurrency
			}
			if input.Currency != cashboxCurrency {
				return Document{}, errors.New("document currency must match cashbox currency")
			}
		}
		if input.Currency == "" {
			input.Currency = "UAH"
		}
		if _, err := s.rateToUAHInTx(tx, input.Currency); err != nil {
			return Document{}, err
		}

		for _, item := range input.Items {
			var productExists bool
			if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, item.ProductID).Scan(&productExists); err != nil {
				return Document{}, err
			}
			if !productExists {
				return Document{}, ErrProductNotFound
			}
		}

		if err := tx.QueryRow(`
			INSERT INTO documents (doc_type, doc_number, status, source_sale_id, source_purchase_id, source_service_order_id, warehouse_id, cashbox_id, currency, total, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, created_at, updated_at
		`,
			input.Type,
			input.Number,
			input.Status,
			input.SourceSaleID,
			input.SourcePurchaseID,
			input.SourceServiceOrderID,
			input.WarehouseID,
			input.CashboxID,
			input.Currency,
			input.Total,
			input.Note,
		).Scan(&input.ID, &input.CreatedAt, &input.UpdatedAt); err != nil {
			return Document{}, err
		}

		for _, item := range input.Items {
			if _, err := tx.Exec(`
				INSERT INTO document_items (document_id, product_id, quantity, price)
				VALUES ($1, $2, $3, $4)
			`, input.ID, item.ProductID, item.Quantity, item.Price); err != nil {
				return Document{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return Document{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if input.WarehouseID != nil && !s.warehouseExistsLocked(*input.WarehouseID) {
		return Document{}, errors.New("warehouse not found")
	}
	if input.CashboxID != nil {
		cashbox := s.cashboxByIDLocked(*input.CashboxID)
		if cashbox == nil {
			return Document{}, errors.New("cashbox not found")
		}
		cashboxCurrency := normalizeCurrency(cashbox.Currency)
		if input.Currency == "" {
			input.Currency = cashboxCurrency
		}
		if input.Currency != cashboxCurrency {
			return Document{}, errors.New("document currency must match cashbox currency")
		}
	}
	if input.Currency == "" {
		input.Currency = "UAH"
	}
	if _, err := s.rateToUAHLocked(input.Currency); err != nil {
		return Document{}, err
	}
	for _, item := range input.Items {
		if !s.productExistsLocked(item.ProductID) {
			return Document{}, ErrProductNotFound
		}
	}

	s.documentSeq++
	input.ID = s.documentSeq
	input.CreatedAt = now
	input.UpdatedAt = now
	s.documents = append(s.documents, input)
	return input, nil
}

func (s *Store) ImportDocumentCSV(input Document, csvText string) (DocumentCSVImportResult, error) {
	reader := csv.NewReader(strings.NewReader(csvText))
	rows, err := reader.ReadAll()
	if err != nil {
		return DocumentCSVImportResult{}, errors.New("invalid csv format")
	}
	if len(rows) < 2 {
		return DocumentCSVImportResult{}, errors.New("csv must contain header and at least one row")
	}

	headerMap := map[string]int{}
	for idx, header := range rows[0] {
		headerMap[normalizeCSVHeader(header)] = idx
	}
	if _, ok := headerMap["productid"]; !ok {
		return DocumentCSVImportResult{}, errors.New("csv header must include productId")
	}
	if _, ok := headerMap["quantity"]; !ok {
		return DocumentCSVImportResult{}, errors.New("csv header must include quantity")
	}

	result := DocumentCSVImportResult{
		Errors: []ProductCSVImportRowError{},
	}
	items := []DocumentItem{}
	for i := 1; i < len(rows); i++ {
		line := i + 1
		item, rowErr := parseDocumentCSVRow(rows[i], headerMap)
		if rowErr != nil {
			result.Skipped++
			result.Errors = append(result.Errors, ProductCSVImportRowError{
				Line:  line,
				Error: rowErr.Error(),
			})
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return result, errors.New("csv contains no valid document rows")
	}

	input.Items = items
	doc, err := s.CreateDocument(input)
	if err != nil {
		return DocumentCSVImportResult{}, err
	}
	result.Document = doc
	result.Imported = len(items)
	return result, nil
}

func (s *Store) CreateReturnFromCustomerDocument(
	saleID int64,
	warehouseID int64,
	currency string,
	items []DocumentItem,
	note string,
) (Document, error) {
	if saleID <= 0 {
		return Document{}, errors.New("saleId is required")
	}
	if warehouseID <= 0 {
		return Document{}, errors.New("warehouseId is required")
	}
	if len(items) == 0 {
		return Document{}, errors.New("необхідно додати хоча б один товар до повернення")
	}

	maxByProduct, saleCurrency, salePriceByProduct, err := s.returnableFromSale(saleID)
	if err != nil {
		return Document{}, err
	}
	normalized := make([]DocumentItem, 0, len(items))
	for _, item := range items {
		if item.ProductID <= 0 {
			return Document{}, errors.New("productId is required")
		}
		if item.Quantity <= 0 {
			return Document{}, errors.New("return item quantity must be greater than zero")
		}
		maxQty, ok := maxByProduct[item.ProductID]
		if !ok || maxQty <= 0 {
			return Document{}, errors.New("product is not returnable for this sale")
		}
		if item.Quantity > maxQty {
			return Document{}, fmt.Errorf("return quantity exceeds sale remainder for product %d", item.ProductID)
		}
		price := item.Price
		if price <= 0 {
			price = salePriceByProduct[item.ProductID]
		}
		if price < 0 {
			return Document{}, errors.New("return item price must be greater or equal zero")
		}
		normalized = append(normalized, DocumentItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     price,
		})
		maxByProduct[item.ProductID] -= item.Quantity
	}

	if currency == "" {
		currency = saleCurrency
	}
	return s.CreateDocument(Document{
		Type:         DocumentTypeReturnFromCustomer,
		SourceSaleID: &saleID,
		WarehouseID:  &warehouseID,
		Currency:     currency,
		Items:        normalized,
		Note:         note,
	})
}

func (s *Store) CreateReturnToSupplierDocument(
	purchaseID int64,
	warehouseID int64,
	currency string,
	items []DocumentItem,
	note string,
) (Document, error) {
	if purchaseID <= 0 {
		return Document{}, errors.New("purchaseId is required")
	}
	if warehouseID <= 0 {
		return Document{}, errors.New("warehouseId is required")
	}
	if len(items) == 0 {
		return Document{}, errors.New("необхідно додати хоча б один товар до повернення")
	}

	maxByProduct, purchaseCurrency, purchasePriceByProduct, err := s.returnableFromPurchase(purchaseID)
	if err != nil {
		return Document{}, err
	}
	normalized := make([]DocumentItem, 0, len(items))
	for _, item := range items {
		if item.ProductID <= 0 {
			return Document{}, errors.New("productId is required")
		}
		if item.Quantity <= 0 {
			return Document{}, errors.New("return item quantity must be greater than zero")
		}
		maxQty, ok := maxByProduct[item.ProductID]
		if !ok || maxQty <= 0 {
			return Document{}, errors.New("product is not returnable for this purchase")
		}
		if item.Quantity > maxQty {
			return Document{}, fmt.Errorf("return quantity exceeds purchase remainder for product %d", item.ProductID)
		}
		price := item.Price
		if price <= 0 {
			price = purchasePriceByProduct[item.ProductID]
		}
		if price < 0 {
			return Document{}, errors.New("return item price must be greater or equal zero")
		}
		normalized = append(normalized, DocumentItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     price,
		})
		maxByProduct[item.ProductID] -= item.Quantity
	}

	if currency == "" {
		currency = purchaseCurrency
	}
	return s.CreateDocument(Document{
		Type:             DocumentTypeReturnToSupplier,
		SourcePurchaseID: &purchaseID,
		WarehouseID:      &warehouseID,
		Currency:         currency,
		Items:            normalized,
		Note:             note,
	})
}

func (s *Store) CreateServiceOrderActDocument(serviceOrderID int64, note string) (Document, error) {
	if serviceOrderID <= 0 {
		return Document{}, errors.New("serviceOrderId is required")
	}
	note = strings.TrimSpace(note)
	if s.db != nil {
		var status string
		var currency string
		var price float64
		var partsTotal float64
		if err := s.db.QueryRow(`
			SELECT status, currency, price, parts_total
			FROM service_orders
			WHERE id = $1
		`, serviceOrderID).Scan(&status, &currency, &price, &partsTotal); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Document{}, errors.New("service order not found")
			}
			return Document{}, err
		}
		if status != ServiceOrderStatusDone {
			return Document{}, errors.New("service order must be done before creating act")
		}
		var exists bool
		if err := s.db.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM documents
				WHERE doc_type = $1
				  AND source_service_order_id = $2
				  AND status <> $3
			)
		`, DocumentTypeAct, serviceOrderID, DocumentStatusCancelled).Scan(&exists); err != nil {
			return Document{}, err
		}
		if exists {
			return Document{}, errors.New("акт виконаних робіт вже існує")
		}
		total := price + partsTotal
		if total <= 0 {
			return Document{}, errors.New("service order total must be greater than zero")
		}
		return s.CreateDocument(Document{
			Type:                 DocumentTypeAct,
			SourceServiceOrderID: &serviceOrderID,
			Currency:             currency,
			Total:                total,
			Note:                 note,
		})
	}

	s.mu.RLock()
	var (
		found    bool
		status   string
		currency string
		total    float64
	)
	for _, order := range s.serviceOrders {
		if order.ID != serviceOrderID {
			continue
		}
		found = true
		status = order.Status
		currency = order.Currency
		total = order.Price + order.PartsTotal
		break
	}
	if !found {
		s.mu.RUnlock()
		return Document{}, errors.New("service order not found")
	}
	if status != ServiceOrderStatusDone {
		s.mu.RUnlock()
		return Document{}, errors.New("service order must be done before creating act")
	}
	for _, doc := range s.documents {
		if doc.Type == DocumentTypeAct && doc.SourceServiceOrderID != nil && *doc.SourceServiceOrderID == serviceOrderID && doc.Status != DocumentStatusCancelled {
			s.mu.RUnlock()
			return Document{}, errors.New("акт виконаних робіт вже існує")
		}
	}
	s.mu.RUnlock()
	if total <= 0 {
		return Document{}, errors.New("service order total must be greater than zero")
	}
	return s.CreateDocument(Document{
		Type:                 DocumentTypeAct,
		SourceServiceOrderID: &serviceOrderID,
		Currency:             currency,
		Total:                total,
		Note:                 note,
	})
}

func (s *Store) ServiceOrderActDocument(serviceOrderID int64) (Document, error) {
	if serviceOrderID <= 0 {
		return Document{}, errors.New("serviceOrderId is required")
	}
	if s.db != nil {
		var doc Document
		row := s.db.QueryRow(`
			SELECT id, doc_type, doc_number, status, source_sale_id, source_purchase_id, source_service_order_id, warehouse_id, cashbox_id, currency, total, note, posted_at, created_at, updated_at
			FROM documents
			WHERE doc_type = $1
			  AND source_service_order_id = $2
			  AND status <> $3
			ORDER BY id DESC
			LIMIT 1
		`, DocumentTypeAct, serviceOrderID, DocumentStatusCancelled)
		if err := row.Scan(
			&doc.ID,
			&doc.Type,
			&doc.Number,
			&doc.Status,
			&doc.SourceSaleID,
			&doc.SourcePurchaseID,
			&doc.SourceServiceOrderID,
			&doc.WarehouseID,
			&doc.CashboxID,
			&doc.Currency,
			&doc.Total,
			&doc.Note,
			&doc.PostedAt,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Document{}, errors.New("акт виконаних робіт не знайдено")
			}
			return Document{}, err
		}
		items, err := s.documentItemsByDocumentID(doc.ID)
		if err != nil {
			return Document{}, err
		}
		doc.Items = items
		return doc, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serviceOrderActDocumentLocked(serviceOrderID)
}

func (s *Store) ensureServiceOrderActPosted(serviceOrderID int64) error {
	_, err := s.ServiceOrderActDocument(serviceOrderID)
	if err == nil {
		return nil
	}
	if err.Error() != "акт виконаних робіт не знайдено" {
		return err
	}
	doc, err := s.CreateServiceOrderActDocument(serviceOrderID, "Auto-created on service order completion")
	if err != nil {
		if err.Error() == "акт виконаних робіт вже існує" {
			return nil
		}
		return err
	}
	if doc.Status == DocumentStatusPosted {
		return nil
	}
	_, err = s.PostDocument(doc.ID)
	return err
}

func (s *Store) CancelServiceOrderActDocument(serviceOrderID int64) (Document, error) {
	if serviceOrderID <= 0 {
		return Document{}, errors.New("serviceOrderId is required")
	}
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Document{}, err
		}
		defer tx.Rollback()
		doc, err := s.serviceOrderActDocumentTx(tx, serviceOrderID, true)
		if err != nil {
			return Document{}, err
		}
		now := time.Now().UTC()
		if _, err := tx.Exec(`
			UPDATE documents
			SET status = $2, updated_at = $3
			WHERE id = $1
		`, doc.ID, DocumentStatusCancelled, now); err != nil {
			return Document{}, err
		}
		doc.Status = DocumentStatusCancelled
		doc.UpdatedAt = now
		if err := tx.Commit(); err != nil {
			return Document{}, err
		}
		return doc, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.documents) - 1; i >= 0; i-- {
		doc := s.documents[i]
		if doc.Type != DocumentTypeAct {
			continue
		}
		if doc.Status == DocumentStatusCancelled {
			continue
		}
		if doc.SourceServiceOrderID == nil || *doc.SourceServiceOrderID != serviceOrderID {
			continue
		}
		s.documents[i].Status = DocumentStatusCancelled
		s.documents[i].UpdatedAt = time.Now().UTC()
		return s.documents[i], nil
	}
	return Document{}, errors.New("акт виконаних робіт не знайдено")
}

func (s *Store) serviceOrderActDocumentLocked(serviceOrderID int64) (Document, error) {
	for i := len(s.documents) - 1; i >= 0; i-- {
		doc := s.documents[i]
		if doc.Type != DocumentTypeAct {
			continue
		}
		if doc.Status == DocumentStatusCancelled {
			continue
		}
		if doc.SourceServiceOrderID == nil || *doc.SourceServiceOrderID != serviceOrderID {
			continue
		}
		return doc, nil
	}
	return Document{}, errors.New("акт виконаних робіт не знайдено")
}

func (s *Store) serviceOrderActDocumentTx(tx *sql.Tx, serviceOrderID int64, lock bool) (Document, error) {
	query := `
		SELECT id, doc_type, doc_number, status, source_sale_id, source_purchase_id, source_service_order_id, warehouse_id, cashbox_id, currency, total, note, posted_at, created_at, updated_at
		FROM documents
		WHERE doc_type = $1
		  AND source_service_order_id = $2
		  AND status <> $3
		ORDER BY id DESC
		LIMIT 1
	`
	if lock {
		query += ` FOR UPDATE`
	}
	var doc Document
	if err := tx.QueryRow(query, DocumentTypeAct, serviceOrderID, DocumentStatusCancelled).Scan(
		&doc.ID,
		&doc.Type,
		&doc.Number,
		&doc.Status,
		&doc.SourceSaleID,
		&doc.SourcePurchaseID,
		&doc.SourceServiceOrderID,
		&doc.WarehouseID,
		&doc.CashboxID,
		&doc.Currency,
		&doc.Total,
		&doc.Note,
		&doc.PostedAt,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Document{}, errors.New("акт виконаних робіт не знайдено")
		}
		return Document{}, err
	}
	items, err := s.documentItemsByDocumentID(doc.ID)
	if err != nil {
		return Document{}, err
	}
	doc.Items = items
	return doc, nil
}

func (s *Store) returnableFromSale(saleID int64) (map[int64]int, string, map[int64]float64, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT si.product_id, SUM(si.quantity) AS qty, MIN(si.price) AS price, s.currency
			FROM sales s
			JOIN sale_items si ON si.sale_id = s.id
			WHERE s.id = $1
			GROUP BY si.product_id, s.currency
		`, saleID)
		if err != nil {
			return nil, "", nil, err
		}
		defer rows.Close()

		available := map[int64]int{}
		prices := map[int64]float64{}
		currency := ""
		for rows.Next() {
			var productID int64
			var qty int
			var price float64
			var rowCurrency string
			if err := rows.Scan(&productID, &qty, &price, &rowCurrency); err != nil {
				return nil, "", nil, err
			}
			available[productID] = qty
			prices[productID] = price
			currency = normalizeCurrency(rowCurrency)
		}
		if err := rows.Err(); err != nil {
			return nil, "", nil, err
		}
		if len(available) == 0 {
			return nil, "", nil, errors.New("sale not found")
		}

		returnedRows, err := s.db.Query(`
			SELECT di.product_id, COALESCE(SUM(di.quantity), 0)
			FROM documents d
			JOIN document_items di ON di.document_id = d.id
			WHERE d.doc_type = $1
			  AND d.source_sale_id = $2
			  AND d.status <> $3
			GROUP BY di.product_id
		`, DocumentTypeReturnFromCustomer, saleID, DocumentStatusCancelled)
		if err != nil {
			return nil, "", nil, err
		}
		defer returnedRows.Close()

		for returnedRows.Next() {
			var productID int64
			var returnedQty int
			if err := returnedRows.Scan(&productID, &returnedQty); err != nil {
				return nil, "", nil, err
			}
			available[productID] -= returnedQty
			if available[productID] < 0 {
				available[productID] = 0
			}
		}
		if err := returnedRows.Err(); err != nil {
			return nil, "", nil, err
		}
		return available, currency, prices, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	var source *Sale
	for i := range s.sales {
		if s.sales[i].ID == saleID {
			source = &s.sales[i]
			break
		}
	}
	if source == nil {
		return nil, "", nil, errors.New("sale not found")
	}

	available := map[int64]int{}
	prices := map[int64]float64{}
	for _, item := range source.Items {
		available[item.ProductID] += item.Quantity
		if _, exists := prices[item.ProductID]; !exists || prices[item.ProductID] <= 0 {
			prices[item.ProductID] = item.Price
		}
	}

	for _, doc := range s.documents {
		if doc.Type != DocumentTypeReturnFromCustomer || doc.SourceSaleID == nil || *doc.SourceSaleID != saleID || doc.Status == DocumentStatusCancelled {
			continue
		}
		for _, item := range doc.Items {
			available[item.ProductID] -= item.Quantity
			if available[item.ProductID] < 0 {
				available[item.ProductID] = 0
			}
		}
	}

	return available, source.Currency, prices, nil
}

func (s *Store) returnableFromPurchase(purchaseID int64) (map[int64]int, string, map[int64]float64, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT pi.product_id, SUM(pi.quantity) AS qty, MIN(pi.price) AS price, p.currency
			FROM purchases p
			JOIN purchase_items pi ON pi.purchase_id = p.id
			WHERE p.id = $1
			GROUP BY pi.product_id, p.currency
		`, purchaseID)
		if err != nil {
			return nil, "", nil, err
		}
		defer rows.Close()

		available := map[int64]int{}
		prices := map[int64]float64{}
		currency := ""
		for rows.Next() {
			var productID int64
			var qty int
			var price float64
			var rowCurrency string
			if err := rows.Scan(&productID, &qty, &price, &rowCurrency); err != nil {
				return nil, "", nil, err
			}
			available[productID] = qty
			prices[productID] = price
			currency = normalizeCurrency(rowCurrency)
		}
		if err := rows.Err(); err != nil {
			return nil, "", nil, err
		}
		if len(available) == 0 {
			return nil, "", nil, errors.New("purchase not found")
		}

		returnedRows, err := s.db.Query(`
			SELECT di.product_id, COALESCE(SUM(di.quantity), 0)
			FROM documents d
			JOIN document_items di ON di.document_id = d.id
			WHERE d.doc_type = $1
			  AND d.source_purchase_id = $2
			  AND d.status <> $3
			GROUP BY di.product_id
		`, DocumentTypeReturnToSupplier, purchaseID, DocumentStatusCancelled)
		if err != nil {
			return nil, "", nil, err
		}
		defer returnedRows.Close()

		for returnedRows.Next() {
			var productID int64
			var returnedQty int
			if err := returnedRows.Scan(&productID, &returnedQty); err != nil {
				return nil, "", nil, err
			}
			available[productID] -= returnedQty
			if available[productID] < 0 {
				available[productID] = 0
			}
		}
		if err := returnedRows.Err(); err != nil {
			return nil, "", nil, err
		}
		return available, currency, prices, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	var source *Purchase
	for i := range s.purchases {
		if s.purchases[i].ID == purchaseID {
			source = &s.purchases[i]
			break
		}
	}
	if source == nil {
		return nil, "", nil, errors.New("purchase not found")
	}

	available := map[int64]int{}
	prices := map[int64]float64{}
	for _, item := range source.Items {
		available[item.ProductID] += item.Quantity
		if _, exists := prices[item.ProductID]; !exists || prices[item.ProductID] <= 0 {
			prices[item.ProductID] = item.Price
		}
	}

	for _, doc := range s.documents {
		if doc.Type != DocumentTypeReturnToSupplier || doc.SourcePurchaseID == nil || *doc.SourcePurchaseID != purchaseID || doc.Status == DocumentStatusCancelled {
			continue
		}
		for _, item := range doc.Items {
			available[item.ProductID] -= item.Quantity
			if available[item.ProductID] < 0 {
				available[item.ProductID] = 0
			}
		}
	}

	return available, source.Currency, prices, nil
}

func (s *Store) CustomerReturnAvailability(saleID int64) (ReturnAvailability, error) {
	availableByProduct, currency, pricesByProduct, err := s.returnableFromSale(saleID)
	if err != nil {
		return ReturnAvailability{}, err
	}
	result := ReturnAvailability{
		SourceID: saleID,
		Currency: currency,
		Items:    make([]ReturnAvailabilityItem, 0, len(availableByProduct)),
	}
	for productID, available := range availableByProduct {
		price := pricesByProduct[productID]
		sourceQty := available
		returnedQty := 0
		if s.db != nil {
			if err := s.db.QueryRow(`
				SELECT COALESCE(SUM(quantity), 0)
				FROM sale_items
				WHERE sale_id = $1 AND product_id = $2
			`, saleID, productID).Scan(&sourceQty); err != nil {
				return ReturnAvailability{}, err
			}
		} else {
			for _, sale := range s.sales {
				if sale.ID != saleID {
					continue
				}
				sourceQty = 0
				for _, item := range sale.Items {
					if item.ProductID == productID {
						sourceQty += item.Quantity
					}
				}
				break
			}
		}
		returnedQty = sourceQty - available
		if returnedQty < 0 {
			returnedQty = 0
		}
		result.Items = append(result.Items, ReturnAvailabilityItem{
			ProductID:    productID,
			SourceQty:    sourceQty,
			ReturnedQty:  returnedQty,
			AvailableQty: available,
			Price:        price,
		})
	}
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].ProductID < result.Items[j].ProductID
	})
	return result, nil
}

func (s *Store) SupplierReturnAvailability(purchaseID int64) (ReturnAvailability, error) {
	availableByProduct, currency, pricesByProduct, err := s.returnableFromPurchase(purchaseID)
	if err != nil {
		return ReturnAvailability{}, err
	}
	result := ReturnAvailability{
		SourceID: purchaseID,
		Currency: currency,
		Items:    make([]ReturnAvailabilityItem, 0, len(availableByProduct)),
	}
	for productID, available := range availableByProduct {
		price := pricesByProduct[productID]
		sourceQty := available
		returnedQty := 0
		if s.db != nil {
			if err := s.db.QueryRow(`
				SELECT COALESCE(SUM(quantity), 0)
				FROM purchase_items
				WHERE purchase_id = $1 AND product_id = $2
			`, purchaseID, productID).Scan(&sourceQty); err != nil {
				return ReturnAvailability{}, err
			}
		} else {
			for _, purchase := range s.purchases {
				if purchase.ID != purchaseID {
					continue
				}
				sourceQty = 0
				for _, item := range purchase.Items {
					if item.ProductID == productID {
						sourceQty += item.Quantity
					}
				}
				break
			}
		}
		returnedQty = sourceQty - available
		if returnedQty < 0 {
			returnedQty = 0
		}
		result.Items = append(result.Items, ReturnAvailabilityItem{
			ProductID:    productID,
			SourceQty:    sourceQty,
			ReturnedQty:  returnedQty,
			AvailableQty: available,
			Price:        price,
		})
	}
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].ProductID < result.Items[j].ProductID
	})
	return result, nil
}

func (s *Store) ListDocuments(docType, status *string) ([]Document, error) {
	if s.db != nil {
		typeFilter := ""
		statusFilter := ""
		if docType != nil {
			typeFilter = strings.TrimSpace(*docType)
		}
		if status != nil {
			statusFilter = strings.TrimSpace(*status)
		}
		rows, err := s.db.Query(`
			SELECT id, doc_type, doc_number, status, source_sale_id, source_purchase_id, source_service_order_id, warehouse_id, cashbox_id, currency, total, note, posted_at, created_at, updated_at
			FROM documents
			WHERE ($1 = '' OR doc_type = $1)
			  AND ($2 = '' OR status = $2)
			ORDER BY id DESC
		`, typeFilter, statusFilter)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := []Document{}
		for rows.Next() {
			var doc Document
			if err := rows.Scan(
				&doc.ID,
				&doc.Type,
				&doc.Number,
				&doc.Status,
				&doc.SourceSaleID,
				&doc.SourcePurchaseID,
				&doc.SourceServiceOrderID,
				&doc.WarehouseID,
				&doc.CashboxID,
				&doc.Currency,
				&doc.Total,
				&doc.Note,
				&doc.PostedAt,
				&doc.CreatedAt,
				&doc.UpdatedAt,
			); err != nil {
				return nil, err
			}
			items, err := s.documentItemsByDocumentID(doc.ID)
			if err != nil {
				return nil, err
			}
			doc.Items = items
			result = append(result, doc)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []Document{}
	for _, doc := range s.documents {
		if docType != nil && strings.TrimSpace(*docType) != "" && doc.Type != strings.TrimSpace(*docType) {
			continue
		}
		if status != nil && strings.TrimSpace(*status) != "" && doc.Status != strings.TrimSpace(*status) {
			continue
		}
		result = append(result, doc)
	}
	return result, nil
}

func (s *Store) PostDocument(documentID int64) (Document, error) {
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Document{}, err
		}
		defer tx.Rollback()

		doc, err := s.documentByIDTx(tx, documentID, true)
		if err != nil {
			return Document{}, err
		}
		if doc.Status != DocumentStatusDraft {
			return Document{}, errors.New("only draft document can be posted")
		}

		if err := s.applyDocumentEffectsTx(tx, doc); err != nil {
			return Document{}, err
		}

		postedAt := time.Now().UTC()
		if _, err := tx.Exec(`
			UPDATE documents
			SET status = $2, posted_at = $3, updated_at = $3
			WHERE id = $1
		`, doc.ID, DocumentStatusPosted, postedAt); err != nil {
			return Document{}, err
		}
		doc.Status = DocumentStatusPosted
		doc.PostedAt = &postedAt
		doc.UpdatedAt = postedAt

		if err := tx.Commit(); err != nil {
			return Document{}, err
		}
		return doc, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index := -1
	for i := range s.documents {
		if s.documents[i].ID == documentID {
			index = i
			break
		}
	}
	if index == -1 {
		return Document{}, errors.New("document not found")
	}
	if s.documents[index].Status != DocumentStatusDraft {
		return Document{}, errors.New("only draft document can be posted")
	}
	if err := s.applyDocumentEffectsLocked(&s.documents[index]); err != nil {
		return Document{}, err
	}
	postedAt := time.Now().UTC()
	s.documents[index].Status = DocumentStatusPosted
	s.documents[index].PostedAt = &postedAt
	s.documents[index].UpdatedAt = postedAt
	return s.documents[index], nil
}

func (s *Store) ListDocumentTemplates() ([]DocumentTemplate, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, code, name, body, is_active, created_at, updated_at
			FROM document_templates
			ORDER BY id ASC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		templates := []DocumentTemplate{}
		for rows.Next() {
			var tpl DocumentTemplate
			if err := rows.Scan(&tpl.ID, &tpl.Code, &tpl.Name, &tpl.Body, &tpl.IsActive, &tpl.CreatedAt, &tpl.UpdatedAt); err != nil {
				return nil, err
			}
			templates = append(templates, tpl)
		}
		return templates, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]DocumentTemplate, len(s.documentTemplates))
	copy(result, s.documentTemplates)
	return result, nil
}

func (s *Store) UpsertDocumentTemplate(input DocumentTemplate) (DocumentTemplate, error) {
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.Body = strings.TrimSpace(input.Body)
	if input.Code == "" {
		return DocumentTemplate{}, errors.New("template code is required")
	}
	if input.Name == "" {
		return DocumentTemplate{}, errors.New("template name is required")
	}
	if input.Body == "" {
		return DocumentTemplate{}, errors.New("template body is required")
	}

	if s.db != nil {
		now := time.Now().UTC()
		if err := s.db.QueryRow(`
			INSERT INTO document_templates (code, name, body, is_active, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (code)
			DO UPDATE SET
				name = EXCLUDED.name,
				body = EXCLUDED.body,
				is_active = EXCLUDED.is_active,
				updated_at = EXCLUDED.updated_at
			RETURNING id, created_at, updated_at
		`, input.Code, input.Name, input.Body, input.IsActive, now).Scan(&input.ID, &input.CreatedAt, &input.UpdatedAt); err != nil {
			return DocumentTemplate{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.documentTemplates {
		if s.documentTemplates[i].Code == input.Code {
			s.documentTemplates[i].Name = input.Name
			s.documentTemplates[i].Body = input.Body
			s.documentTemplates[i].IsActive = input.IsActive
			s.documentTemplates[i].UpdatedAt = now
			return s.documentTemplates[i], nil
		}
	}
	s.documentTplSeq++
	input.ID = s.documentTplSeq
	input.CreatedAt = now
	input.UpdatedAt = now
	s.documentTemplates = append(s.documentTemplates, input)
	return input, nil
}

func (s *Store) DocumentTemplatePlaceholders() []TemplatePlaceholder {
	return []TemplatePlaceholder{
		{Name: "number", Description: "Document number"},
		{Name: "createdAt", Description: "Document creation date"},
		{Name: "updatedAt", Description: "Last update date"},
		{Name: "postedAt", Description: "Posting date"},
		{Name: "type", Description: "Document type"},
		{Name: "status", Description: "Document status"},
		{Name: "currency", Description: "Document currency"},
		{Name: "total", Description: "Document total"},
		{Name: "note", Description: "Document note"},
		{Name: "warehouseId", Description: "Warehouse id"},
		{Name: "cashboxId", Description: "Cashbox id"},
		{Name: "items", Description: "Document items text"},
	}
}

func (s *Store) ValidateDocumentTemplate(code, body string, strict bool) (DocumentTemplateValidation, error) {
	code = strings.TrimSpace(code)
	body = strings.TrimSpace(body)
	if !isValidDocumentType(code) {
		return DocumentTemplateValidation{}, errors.New("invalid document type")
	}
	if body == "" {
		return DocumentTemplateValidation{}, errors.New("template body is required")
	}

	allowedMap := map[string]struct{}{}
	allowed := []string{}
	for _, placeholder := range s.DocumentTemplatePlaceholders() {
		allowed = append(allowed, placeholder.Name)
		allowedMap[placeholder.Name] = struct{}{}
	}

	usedSet := map[string]struct{}{}
	unknownSet := map[string]struct{}{}
	matches := templatePlaceholderPattern.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		usedSet[name] = struct{}{}
		if _, ok := allowedMap[name]; !ok {
			unknownSet[name] = struct{}{}
		}
	}

	used := mapKeysSorted(usedSet)
	unknown := mapKeysSorted(unknownSet)
	required := requiredTemplatePlaceholders(code)
	missingRequired := []string{}
	if strict {
		requiredSet := map[string]struct{}{}
		for _, name := range required {
			requiredSet[name] = struct{}{}
		}
		for name := range requiredSet {
			if _, ok := usedSet[name]; !ok {
				missingRequired = append(missingRequired, name)
			}
		}
		sort.Strings(missingRequired)
	}
	return DocumentTemplateValidation{
		Valid:           len(unknown) == 0 && len(missingRequired) == 0,
		Strict:          strict,
		Used:            used,
		Unknown:         unknown,
		MissingRequired: missingRequired,
		Allowed:         allowed,
		Required:        required,
		TemplateLen:     len(body),
	}, nil
}

func (s *Store) PreviewDocumentTemplate(code, body string, documentID *int64, strict bool) (DocumentTemplatePreview, error) {
	validation, err := s.ValidateDocumentTemplate(code, body, strict)
	if err != nil {
		return DocumentTemplatePreview{}, err
	}
	if !validation.Valid {
		return DocumentTemplatePreview{}, errors.New("template validation failed")
	}

	var doc Document
	if documentID != nil && *documentID > 0 {
		doc, err = s.documentByID(*documentID)
		if err != nil {
			return DocumentTemplatePreview{}, err
		}
	} else {
		doc = sampleDocumentForTemplate(code)
	}

	content := renderDocumentTemplate(
		doc,
		DocumentTemplate{
			Code: code,
			Body: body,
		},
	)
	return DocumentTemplatePreview{
		Content:    content,
		Validation: validation,
	}, nil
}

func (s *Store) RenderDocumentPDF(documentID int64) ([]byte, error) {
	doc, err := s.documentByID(documentID)
	if err != nil {
		return nil, err
	}
	template := s.documentTemplateByCode(doc.Type)
	return buildDocumentPDF(doc, template)
}

func (s *Store) ApplyInventory(inventoryID int64) (Inventory, error) {
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Inventory{}, err
		}
		defer tx.Rollback()

		var inv Inventory
		if err := tx.QueryRow(`
			SELECT id, warehouse_id, status, note, applied_at, created_at
			FROM inventories
			WHERE id = $1
			FOR UPDATE
		`, inventoryID).Scan(&inv.ID, &inv.WarehouseID, &inv.Status, &inv.Note, &inv.AppliedAt, &inv.CreatedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Inventory{}, errors.New("inventory not found")
			}
			return Inventory{}, err
		}
		if inv.Status != InventoryStatusDraft {
			return Inventory{}, errors.New("only draft inventory can be applied")
		}

		items, err := s.inventoryItemsByInventoryIDTx(tx, inv.ID)
		if err != nil {
			return Inventory{}, err
		}
		inv.Items = items

		for _, item := range items {
			if err := s.setWarehouseStockQuantityInTx(tx, inv.WarehouseID, item.ProductID, item.ActualQuantity); err != nil {
				return Inventory{}, err
			}
			if _, err := tx.Exec(`UPDATE products SET stock = stock + $2 WHERE id = $1`, item.ProductID, item.Adjustment); err != nil {
				return Inventory{}, err
			}
			mType := "inventory_adjustment"
			if _, err := tx.Exec(`
				INSERT INTO stock_movements (product_id, from_warehouse_id, to_warehouse_id, movement_type, quantity, note)
				VALUES ($1, $2, $3, $4, $5, $6)
			`,
				item.ProductID,
				nil,
				inv.WarehouseID,
				mType,
				absInt(item.Adjustment),
				fmt.Sprintf("inventory_id=%d adjustment=%d", inv.ID, item.Adjustment),
			); err != nil {
				return Inventory{}, err
			}
		}

		appliedAt := time.Now().UTC()
		if _, err := tx.Exec(`
			UPDATE inventories
			SET status = $2, applied_at = $3
			WHERE id = $1
		`, inv.ID, InventoryStatusApplied, appliedAt); err != nil {
			return Inventory{}, err
		}
		inv.Status = InventoryStatusApplied
		inv.AppliedAt = &appliedAt

		if err := tx.Commit(); err != nil {
			return Inventory{}, err
		}
		return inv, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.inventories {
		if s.inventories[i].ID != inventoryID {
			continue
		}
		if s.inventories[i].Status != InventoryStatusDraft {
			return Inventory{}, errors.New("only draft inventory can be applied")
		}
		for _, item := range s.inventories[i].Items {
			s.setWarehouseStockLocked(s.inventories[i].WarehouseID, item.ProductID, item.ActualQuantity)
			for p := range s.products {
				if s.products[p].ID == item.ProductID {
					s.products[p].Stock += item.Adjustment
					break
				}
			}
			s.movementSeq++
			toID := s.inventories[i].WarehouseID
			s.movements = append(s.movements, StockMovement{
				ID:            s.movementSeq,
				ProductID:     item.ProductID,
				ToWarehouseID: &toID,
				Type:          "inventory_adjustment",
				Quantity:      absInt(item.Adjustment),
				Note:          fmt.Sprintf("inventory_id=%d adjustment=%d", inventoryID, item.Adjustment),
				CreatedAt:     time.Now().UTC(),
			})
		}
		appliedAt := time.Now().UTC()
		s.inventories[i].Status = InventoryStatusApplied
		s.inventories[i].AppliedAt = &appliedAt
		return s.inventories[i], nil
	}
	return Inventory{}, errors.New("inventory not found")
}

func (s *Store) CreateCashbox(input Cashbox) (Cashbox, error) {
	if input.Name == "" {
		return Cashbox{}, errors.New("cashbox name is required")
	}
	if !isValidPaymentMethod(input.Type) {
		return Cashbox{}, errors.New("invalid cashbox type")
	}
	input.Currency = normalizeCurrency(input.Currency)

	if s.db != nil {
		err := s.db.QueryRow(`
			INSERT INTO cashboxes (name, type, currency, balance)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at
		`,
			input.Name,
			input.Type,
			input.Currency,
			input.Balance,
		).Scan(&input.ID, &input.CreatedAt)
		return input, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cashboxSeq++
	input.ID = s.cashboxSeq
	input.CreatedAt = time.Now().UTC()
	s.cashboxes = append(s.cashboxes, input)
	return input, nil
}

func (s *Store) ListCashboxes() ([]Cashbox, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, name, type, currency, balance, created_at
			FROM cashboxes
			ORDER BY id ASC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cashboxes := []Cashbox{}
		for rows.Next() {
			var item Cashbox
			if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Currency, &item.Balance, &item.CreatedAt); err != nil {
				return nil, err
			}
			cashboxes = append(cashboxes, item)
		}
		return cashboxes, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Cashbox, len(s.cashboxes))
	copy(result, s.cashboxes)
	return result, nil
}

func (s *Store) ListCashOperations(cashboxID *int64) ([]CashOperation, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, cashbox_id, operation_type, amount, currency, amount_uah, payment_method, payment_id, description, created_at
			FROM cash_operations
			WHERE ($1::BIGINT IS NULL OR cashbox_id = $1)
			ORDER BY id DESC
		`, cashboxID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanCashOperations(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []CashOperation{}
	for _, item := range s.cashOperations {
		if cashboxID != nil && item.CashboxID != *cashboxID {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Store) CreateCashOperation(input CashOperation) (CashOperation, error) {
	if input.CashboxID <= 0 {
		return CashOperation{}, errors.New("cashboxId is required")
	}
	if input.Amount <= 0 {
		return CashOperation{}, errors.New("amount must be greater than zero")
	}
	if input.Type != CashOperationTypeIncoming && input.Type != CashOperationTypeOutgoing {
		return CashOperation{}, errors.New("invalid cash operation type")
	}
	if input.Method != "" && !isValidPaymentMethod(input.Method) {
		return CashOperation{}, errors.New("invalid payment method")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return CashOperation{}, err
		}
		defer tx.Rollback()

		var balance float64
		var cashboxCurrency string
		if err := tx.QueryRow(`SELECT balance, currency FROM cashboxes WHERE id = $1 FOR UPDATE`, input.CashboxID).Scan(&balance, &cashboxCurrency); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CashOperation{}, errors.New("cashbox not found")
			}
			return CashOperation{}, err
		}
		input.Currency = normalizeCurrency(cashboxCurrency)
		rateToUAH, err := s.rateToUAHInTx(tx, input.Currency)
		if err != nil {
			return CashOperation{}, err
		}
		input.AmountUAH = input.Amount * rateToUAH
		nextBalance := balance
		if input.Type == CashOperationTypeIncoming {
			nextBalance += input.Amount
		} else {
			if balance < input.Amount {
				return CashOperation{}, errors.New("insufficient cashbox balance")
			}
			nextBalance -= input.Amount
		}

		if _, err := tx.Exec(`UPDATE cashboxes SET balance = $2 WHERE id = $1`, input.CashboxID, nextBalance); err != nil {
			return CashOperation{}, err
		}

		err = tx.QueryRow(`
			INSERT INTO cash_operations (cashbox_id, operation_type, amount, currency, amount_uah, payment_method, payment_id, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, created_at
		`,
			input.CashboxID,
			input.Type,
			input.Amount,
			input.Currency,
			input.AmountUAH,
			input.Method,
			input.PaymentID,
			input.Description,
		).Scan(&input.ID, &input.CreatedAt)
		if err != nil {
			return CashOperation{}, err
		}

		if err := tx.Commit(); err != nil {
			return CashOperation{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cashboxIndex := -1
	for i := range s.cashboxes {
		if s.cashboxes[i].ID == input.CashboxID {
			cashboxIndex = i
			break
		}
	}
	if cashboxIndex == -1 {
		return CashOperation{}, errors.New("cashbox not found")
	}
	input.Currency = normalizeCurrency(s.cashboxes[cashboxIndex].Currency)
	rateToUAH, err := s.rateToUAHLocked(input.Currency)
	if err != nil {
		return CashOperation{}, err
	}
	input.AmountUAH = input.Amount * rateToUAH

	if input.Type == CashOperationTypeIncoming {
		s.cashboxes[cashboxIndex].Balance += input.Amount
	} else {
		if s.cashboxes[cashboxIndex].Balance < input.Amount {
			return CashOperation{}, errors.New("insufficient cashbox balance")
		}
		s.cashboxes[cashboxIndex].Balance -= input.Amount
	}

	s.cashOperationSeq++
	input.ID = s.cashOperationSeq
	input.CreatedAt = time.Now().UTC()
	s.cashOperations = append(s.cashOperations, input)
	return input, nil
}

func (s *Store) OpenCashShift(cashboxID int64, openedBy, note string) (CashShift, error) {
	if cashboxID <= 0 {
		return CashShift{}, errors.New("cashboxId is required")
	}
	openedBy = strings.TrimSpace(openedBy)
	if openedBy == "" {
		return CashShift{}, errors.New("openedBy is required")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return CashShift{}, err
		}
		defer tx.Rollback()

		var openingBalance float64
		if err := tx.QueryRow(`SELECT balance FROM cashboxes WHERE id = $1 FOR UPDATE`, cashboxID).Scan(&openingBalance); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CashShift{}, errors.New("cashbox not found")
			}
			return CashShift{}, err
		}
		var openExists bool
		if err := tx.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM cash_shifts
				WHERE cashbox_id = $1 AND status = $2
			)
		`, cashboxID, CashShiftStatusOpen).Scan(&openExists); err != nil {
			return CashShift{}, err
		}
		if openExists {
			return CashShift{}, errors.New("cash shift already open for cashbox")
		}

		shift := CashShift{
			CashboxID:      cashboxID,
			Status:         CashShiftStatusOpen,
			OpenedBy:       openedBy,
			OpeningBalance: openingBalance,
			Note:           note,
		}
		if err := tx.QueryRow(`
			INSERT INTO cash_shifts (cashbox_id, status, opened_by, opening_balance, note)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, opened_at
		`, shift.CashboxID, shift.Status, shift.OpenedBy, shift.OpeningBalance, shift.Note).Scan(&shift.ID, &shift.OpenedAt); err != nil {
			return CashShift{}, err
		}
		if err := tx.Commit(); err != nil {
			return CashShift{}, err
		}
		return shift, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cashbox := s.cashboxByIDLocked(cashboxID)
	if cashbox == nil {
		return CashShift{}, errors.New("cashbox not found")
	}
	for _, shift := range s.cashShifts {
		if shift.CashboxID == cashboxID && shift.Status == CashShiftStatusOpen {
			return CashShift{}, errors.New("cash shift already open for cashbox")
		}
	}
	s.cashShiftSeq++
	shift := CashShift{
		ID:             s.cashShiftSeq,
		CashboxID:      cashboxID,
		Status:         CashShiftStatusOpen,
		OpenedBy:       openedBy,
		OpeningBalance: cashbox.Balance,
		Note:           note,
		OpenedAt:       time.Now().UTC(),
	}
	s.cashShifts = append(s.cashShifts, shift)
	return shift, nil
}

func (s *Store) CloseCashShift(shiftID int64, closedBy, note string) (CashShift, error) {
	if shiftID <= 0 {
		return CashShift{}, errors.New("shiftId is required")
	}
	closedBy = strings.TrimSpace(closedBy)
	if closedBy == "" {
		return CashShift{}, errors.New("closedBy is required")
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return CashShift{}, err
		}
		defer tx.Rollback()

		shift := CashShift{}
		if err := tx.QueryRow(`
			SELECT id, cashbox_id, status, opened_by, closed_by, opening_balance, closing_balance, note, opened_at, closed_at
			FROM cash_shifts
			WHERE id = $1
			FOR UPDATE
		`, shiftID).Scan(
			&shift.ID,
			&shift.CashboxID,
			&shift.Status,
			&shift.OpenedBy,
			&shift.ClosedBy,
			&shift.OpeningBalance,
			&shift.ClosingBalance,
			&shift.Note,
			&shift.OpenedAt,
			&shift.ClosedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CashShift{}, errors.New("cash shift not found")
			}
			return CashShift{}, err
		}
		if shift.Status != CashShiftStatusOpen {
			return CashShift{}, errors.New("only open cash shift can be closed")
		}
		var closingBalance float64
		if err := tx.QueryRow(`SELECT balance FROM cashboxes WHERE id = $1 FOR UPDATE`, shift.CashboxID).Scan(&closingBalance); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CashShift{}, errors.New("cashbox not found")
			}
			return CashShift{}, err
		}
		closedAt := time.Now().UTC()
		if _, err := tx.Exec(`
			UPDATE cash_shifts
			SET status = $2, closed_by = $3, closing_balance = $4, note = $5, closed_at = $6
			WHERE id = $1
		`, shift.ID, CashShiftStatusClosed, closedBy, closingBalance, note, closedAt); err != nil {
			return CashShift{}, err
		}
		shift.Status = CashShiftStatusClosed
		shift.ClosedBy = closedBy
		shift.ClosingBalance = closingBalance
		shift.Note = note
		shift.ClosedAt = &closedAt
		if err := tx.Commit(); err != nil {
			return CashShift{}, err
		}
		return shift, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.cashShifts {
		if s.cashShifts[i].ID != shiftID {
			continue
		}
		if s.cashShifts[i].Status != CashShiftStatusOpen {
			return CashShift{}, errors.New("only open cash shift can be closed")
		}
		cashbox := s.cashboxByIDLocked(s.cashShifts[i].CashboxID)
		if cashbox == nil {
			return CashShift{}, errors.New("cashbox not found")
		}
		closedAt := time.Now().UTC()
		s.cashShifts[i].Status = CashShiftStatusClosed
		s.cashShifts[i].ClosedBy = closedBy
		s.cashShifts[i].ClosingBalance = cashbox.Balance
		s.cashShifts[i].Note = note
		s.cashShifts[i].ClosedAt = &closedAt
		return s.cashShifts[i], nil
	}
	return CashShift{}, errors.New("cash shift not found")
}

func (s *Store) ListCashShifts(cashboxID *int64, status string) ([]CashShift, error) {
	status = strings.TrimSpace(status)
	if status != "" && status != CashShiftStatusOpen && status != CashShiftStatusClosed {
		return nil, errors.New("invalid cash shift status")
	}
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, cashbox_id, status, opened_by, closed_by, opening_balance, closing_balance, note, opened_at, closed_at
			FROM cash_shifts
			WHERE ($1::BIGINT IS NULL OR cashbox_id = $1)
			  AND ($2 = '' OR status = $2)
			ORDER BY id DESC
		`, cashboxID, status)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		shifts := []CashShift{}
		for rows.Next() {
			var shift CashShift
			if err := rows.Scan(
				&shift.ID,
				&shift.CashboxID,
				&shift.Status,
				&shift.OpenedBy,
				&shift.ClosedBy,
				&shift.OpeningBalance,
				&shift.ClosingBalance,
				&shift.Note,
				&shift.OpenedAt,
				&shift.ClosedAt,
			); err != nil {
				return nil, err
			}
			shifts = append(shifts, shift)
		}
		return shifts, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	shifts := []CashShift{}
	for _, shift := range s.cashShifts {
		if cashboxID != nil && shift.CashboxID != *cashboxID {
			continue
		}
		if status != "" && shift.Status != status {
			continue
		}
		shifts = append(shifts, shift)
	}
	sort.Slice(shifts, func(i, j int) bool { return shifts[i].ID > shifts[j].ID })
	return shifts, nil
}

func (s *Store) CreatePayment(input Payment) (Payment, error) {
	if input.Amount <= 0 {
		return Payment{}, errors.New("amount must be greater than zero")
	}
	targetCount := 0
	if input.OrderID != nil {
		targetCount++
	}
	if input.SaleID != nil {
		targetCount++
	}
	if input.ServiceOrderID != nil {
		targetCount++
	}
	if input.SupplierOrderID != nil {
		targetCount++
	}
	if input.PurchaseID != nil {
		targetCount++
	}
	if targetCount != 1 {
		return Payment{}, errors.New("exactly one target must be provided: orderId, saleId, serviceOrderId, supplierOrderId or purchaseId")
	}
	if !isValidPaymentMethod(input.Method) {
		return Payment{}, errors.New("invalid payment method")
	}
	input.Currency = normalizeCurrency(input.Currency)

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Payment{}, err
		}
		defer tx.Rollback()

		rateToUAH, err := s.rateToUAHInTx(tx, input.Currency)
		if err != nil {
			return Payment{}, err
		}
		input.AmountUAH = input.Amount * rateToUAH

		// Only enforce debt ceiling for customer-side payments (not supplier outgoing)
		if input.SupplierOrderID == nil && input.PurchaseID == nil {
			totalUAH, paidUAH, err := s.targetPaymentTotalsInTx(
				tx,
				input.OrderID,
				input.SaleID,
				input.ServiceOrderID,
			)
			if err != nil {
				return Payment{}, err
			}
			if paidUAH+input.AmountUAH > totalUAH {
				return Payment{}, errors.New("payment amount exceeds outstanding debt")
			}
		}

		cashboxID, cashboxCurrency, err := s.resolveCashboxInTx(tx, input.CashboxID, input.Method)
		if err != nil {
			return Payment{}, err
		}
		input.CashboxID = cashboxID
		if input.Currency != cashboxCurrency {
			return Payment{}, errors.New("payment currency must match cashbox currency")
		}

		// For outgoing supplier payments, deduct from cashbox instead of add
		isSupplierPayment := input.SupplierOrderID != nil || input.PurchaseID != nil
		_ = isSupplierPayment

		err = tx.QueryRow(`
			INSERT INTO payments (order_id, sale_id, service_order_id, supplier_order_id, purchase_id, cashbox_id, amount, currency, amount_uah, payment_method, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, created_at
		`,
			input.OrderID,
			input.SaleID,
			input.ServiceOrderID,
			input.SupplierOrderID,
			input.PurchaseID,
			input.CashboxID,
			input.Amount,
			input.Currency,
			input.AmountUAH,
			input.Method,
			input.Note,
		).Scan(&input.ID, &input.CreatedAt)
		if err != nil {
			return Payment{}, err
		}

		// Supplier payments are outgoing (deduct from cashbox); others are incoming
		balanceDelta := input.Amount
		cashOpType := CashOperationTypeIncoming
		cashOpDesc := "Auto incoming from payment"
		if input.SupplierOrderID != nil || input.PurchaseID != nil {
			balanceDelta = -input.Amount
			cashOpType = CashOperationTypeOutgoing
			cashOpDesc = "Outgoing payment to supplier"
		}

		if _, err := tx.Exec(`
			UPDATE cashboxes
			SET balance = balance + $2
			WHERE id = $1
		`,
			input.CashboxID,
			balanceDelta,
		); err != nil {
			return Payment{}, err
		}

		if _, err := tx.Exec(`
			INSERT INTO cash_operations (cashbox_id, operation_type, amount, currency, amount_uah, payment_method, payment_id, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			input.CashboxID,
			cashOpType,
			input.Amount,
			input.Currency,
			input.AmountUAH,
			input.Method,
			input.ID,
			cashOpDesc,
		); err != nil {
			return Payment{}, err
		}

		if input.OrderID != nil {
			if err := s.refreshOrderPaymentStatusInTx(tx, *input.OrderID); err != nil {
				return Payment{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return Payment{}, err
		}
		return input, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Only enforce debt ceiling for customer-side payments (not supplier outgoing)
	isSupplierPaymentMem := input.SupplierOrderID != nil || input.PurchaseID != nil
	if !isSupplierPaymentMem {
		totalUAH, paidUAH, err := s.targetPaymentTotalsLocked(input.OrderID, input.SaleID, input.ServiceOrderID)
		if err != nil {
			return Payment{}, err
		}
		rateToUAH, err := s.rateToUAHLocked(input.Currency)
		if err != nil {
			return Payment{}, err
		}
		input.AmountUAH = input.Amount * rateToUAH
		if paidUAH+input.AmountUAH > totalUAH {
			return Payment{}, errors.New("payment amount exceeds outstanding debt")
		}
	} else {
		rateToUAH, err := s.rateToUAHLocked(input.Currency)
		if err != nil {
			return Payment{}, err
		}
		input.AmountUAH = input.Amount * rateToUAH
	}

	cashboxIndex, err := s.resolveCashboxIndexLocked(input.CashboxID, input.Method)
	if err != nil {
		return Payment{}, err
	}
	input.CashboxID = s.cashboxes[cashboxIndex].ID
	if input.Currency != s.cashboxes[cashboxIndex].Currency {
		return Payment{}, errors.New("payment currency must match cashbox currency")
	}

	s.paymentSeq++
	input.ID = s.paymentSeq
	input.CreatedAt = time.Now().UTC()
	s.payments = append(s.payments, input)

	// Supplier payments are outgoing (deduct from cashbox); others are incoming
	cashOpType := CashOperationTypeIncoming
	cashOpDesc := "Auto incoming from payment"
	if isSupplierPaymentMem {
		s.cashboxes[cashboxIndex].Balance -= input.Amount
		cashOpType = CashOperationTypeOutgoing
		cashOpDesc = "Outgoing payment to supplier"
	} else {
		s.cashboxes[cashboxIndex].Balance += input.Amount
	}

	s.cashOperationSeq++
	s.cashOperations = append(s.cashOperations, CashOperation{
		ID:          s.cashOperationSeq,
		CashboxID:   input.CashboxID,
		Type:        cashOpType,
		Amount:      input.Amount,
		Currency:    input.Currency,
		AmountUAH:   input.AmountUAH,
		Method:      input.Method,
		PaymentID:   &input.ID,
		Description: cashOpDesc,
		CreatedAt:   input.CreatedAt,
	})

	if input.OrderID != nil {
		s.refreshOrderPaymentStatusLocked(*input.OrderID)
	}

	return input, nil
}

func (s *Store) ListPayments(orderID, saleID, serviceOrderID, supplierOrderID *int64) ([]Payment, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, order_id, sale_id, service_order_id, supplier_order_id, purchase_id, cashbox_id, amount, currency, amount_uah, payment_method, note, created_at
			FROM payments
			WHERE ($1::BIGINT IS NULL OR order_id = $1)
				AND ($2::BIGINT IS NULL OR sale_id = $2)
				AND ($3::BIGINT IS NULL OR service_order_id = $3)
				AND ($4::BIGINT IS NULL OR supplier_order_id = $4)
			ORDER BY id DESC
		`, orderID, saleID, serviceOrderID, supplierOrderID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanPayments(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []Payment{}
	for _, payment := range s.payments {
		if orderID != nil {
			if payment.OrderID == nil || *payment.OrderID != *orderID {
				continue
			}
		}
		if saleID != nil {
			if payment.SaleID == nil || *payment.SaleID != *saleID {
				continue
			}
		}
		if serviceOrderID != nil {
			if payment.ServiceOrderID == nil || *payment.ServiceOrderID != *serviceOrderID {
				continue
			}
		}
		if supplierOrderID != nil {
			if payment.SupplierOrderID == nil || *payment.SupplierOrderID != *supplierOrderID {
				continue
			}
		}
		result = append(result, payment)
	}
	return result, nil
}

func (s *Store) ListDebts() ([]DebtSummary, error) {
	if s.db != nil {
		orderDebts, err := s.listOrderDebtsFromDB()
		if err != nil {
			return nil, err
		}
		saleDebts, err := s.listSaleDebtsFromDB()
		if err != nil {
			return nil, err
		}
		serviceOrderDebts, err := s.listServiceOrderDebtsFromDB()
		if err != nil {
			return nil, err
		}
		return append(append(orderDebts, saleDebts...), serviceOrderDebts...), nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	debts := []DebtSummary{}
	for _, order := range s.orders {
		if order.Status == OrderStatusCancelled {
			continue
		}
		paidUAH, lastPayment := s.paidForTargetUAHLocked(&order.ID, nil, nil)
		if order.TotalUAH-paidUAH <= 0 {
			continue
		}
		rateToUAH, err := s.rateToUAHLocked(order.Currency)
		if err != nil {
			return nil, err
		}
		paid := paidUAH / rateToUAH
		debts = append(debts, DebtSummary{
			EntityType:    "order",
			EntityID:      order.ID,
			Currency:      order.Currency,
			Total:         order.Total,
			Paid:          paid,
			Debt:          order.Total - paid,
			TotalUAH:      order.TotalUAH,
			PaidUAH:       paidUAH,
			DebtUAH:       order.TotalUAH - paidUAH,
			DueDate:       order.DueDate,
			IsOverdue:     isDebtOverdue(order.DueDate, order.TotalUAH-paidUAH, time.Now().UTC()),
			OverdueDays:   debtOverdueDays(order.DueDate, order.TotalUAH-paidUAH, time.Now().UTC()),
			LastPaymentAt: lastPayment,
		})
	}

	for _, sale := range s.sales {
		paidUAH, lastPayment := s.paidForTargetUAHLocked(nil, &sale.ID, nil)
		if sale.TotalUAH-paidUAH <= 0 {
			continue
		}
		rateToUAH, err := s.rateToUAHLocked(sale.Currency)
		if err != nil {
			return nil, err
		}
		paid := paidUAH / rateToUAH
		debts = append(debts, DebtSummary{
			EntityType:    "sale",
			EntityID:      sale.ID,
			Currency:      sale.Currency,
			Total:         sale.Total,
			Paid:          paid,
			Debt:          sale.Total - paid,
			TotalUAH:      sale.TotalUAH,
			PaidUAH:       paidUAH,
			DebtUAH:       sale.TotalUAH - paidUAH,
			IsOverdue:     false,
			OverdueDays:   0,
			LastPaymentAt: lastPayment,
		})
	}

	for _, order := range s.serviceOrders {
		if order.Status == ServiceOrderStatusCancelled {
			continue
		}
		total := order.Price + order.PartsTotal
		rateToUAH, err := s.rateToUAHLocked(order.Currency)
		if err != nil {
			return nil, err
		}
		totalUAH := total * rateToUAH
		paidUAH, lastPayment := s.paidForTargetUAHLocked(nil, nil, &order.ID)
		if totalUAH-paidUAH <= 0 {
			continue
		}
		paid := paidUAH / rateToUAH
		debts = append(debts, DebtSummary{
			EntityType:    "service_order",
			EntityID:      order.ID,
			Currency:      order.Currency,
			Total:         total,
			Paid:          paid,
			Debt:          total - paid,
			TotalUAH:      totalUAH,
			PaidUAH:       paidUAH,
			DebtUAH:       totalUAH - paidUAH,
			IsOverdue:     false,
			OverdueDays:   0,
			LastPaymentAt: lastPayment,
		})
	}

	for _, so := range s.supplierOrders {
		if so.Status == SupplierOrderStatusClosed || so.Status == SupplierOrderStatusCancelled {
			continue
		}
		if so.TotalUAH <= 0 {
			continue
		}
		paidUAH, lastPayment := s.paidForTargetUAHLocked(nil, nil, nil, &so.ID)
		if so.TotalUAH-paidUAH <= 0 {
			continue
		}
		rateToUAH, err := s.rateToUAHLocked(so.Currency)
		if err != nil || rateToUAH == 0 {
			rateToUAH = 1
		}
		paid := paidUAH / rateToUAH
		debts = append(debts, DebtSummary{
			EntityType:    "supplier_order",
			EntityID:      so.ID,
			Currency:      so.Currency,
			Total:         so.Total,
			Paid:          paid,
			Debt:          so.Total - paid,
			TotalUAH:      so.TotalUAH,
			PaidUAH:       paidUAH,
			DebtUAH:       so.TotalUAH - paidUAH,
			IsOverdue:     false,
			OverdueDays:   0,
			LastPaymentAt: lastPayment,
		})
	}

	return debts, nil
}

func (s *Store) ListOverdueDebts(asOf time.Time) ([]DebtSummary, error) {
	debts, err := s.ListDebts()
	if err != nil {
		return nil, err
	}

	overdue := []DebtSummary{}
	for _, debt := range debts {
		if !isDebtOverdue(debt.DueDate, debt.DebtUAH, asOf) {
			continue
		}
		debt.IsOverdue = true
		debt.OverdueDays = debtOverdueDays(debt.DueDate, debt.DebtUAH, asOf)
		overdue = append(overdue, debt)
	}
	return overdue, nil
}

func (s *Store) DebtPaymentHistory(entityType string, entityID int64) ([]DebtPaymentHistoryEntry, error) {
	if entityID <= 0 {
		return nil, errors.New("entityId must be greater than zero")
	}
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	if entityType != "order" && entityType != "sale" && entityType != "service_order" {
		return nil, errors.New("entityType must be order, sale or service_order")
	}

	var orderID *int64
	var saleID *int64
	var serviceOrderID *int64
	var total float64
	var totalUAH float64
	var currency string

	if s.db != nil {
		if entityType == "order" {
			orderID = &entityID
			if err := s.db.QueryRow(`
				SELECT total, total_uah, currency
				FROM customer_orders
				WHERE id = $1
			`, entityID).Scan(&total, &totalUAH, &currency); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, errors.New("order not found")
				}
				return nil, err
			}
		} else if entityType == "sale" {
			saleID = &entityID
			if err := s.db.QueryRow(`
				SELECT total, total_uah, currency
				FROM sales
				WHERE id = $1
			`, entityID).Scan(&total, &totalUAH, &currency); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, errors.New("sale not found")
				}
				return nil, err
			}
		} else {
			serviceOrderID = &entityID
			if err := s.db.QueryRow(`
				SELECT (price + parts_total), (price + parts_total) * COALESCE(er.rate_to_uah, 1), so.currency
				FROM service_orders so
				LEFT JOIN exchange_rates er ON er.currency = so.currency
				WHERE so.id = $1
			`, entityID).Scan(&total, &totalUAH, &currency); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, errors.New("service order not found")
				}
				return nil, err
			}
		}

		payments, err := s.ListPayments(orderID, saleID, serviceOrderID, nil)
		if err != nil {
			return nil, err
		}
		sort.Slice(payments, func(i, j int) bool {
			return payments[i].CreatedAt.Before(payments[j].CreatedAt)
		})

		remaining := total
		remainingUAH := totalUAH
		history := []DebtPaymentHistoryEntry{}
		for _, payment := range payments {
			remaining -= payment.Amount
			remainingUAH -= payment.AmountUAH
			if remaining < 0 {
				remaining = 0
			}
			if remainingUAH < 0 {
				remainingUAH = 0
			}
			history = append(history, DebtPaymentHistoryEntry{
				PaymentID:        payment.ID,
				Amount:           payment.Amount,
				AmountUAH:        payment.AmountUAH,
				Currency:         payment.Currency,
				Method:           payment.Method,
				Note:             payment.Note,
				CreatedAt:        payment.CreatedAt,
				RemainingDebt:    remaining,
				RemainingDebtUAH: remainingUAH,
			})
		}
		return history, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if entityType == "order" {
		orderID = &entityID
		found := false
		for _, order := range s.orders {
			if order.ID != entityID {
				continue
			}
			total = order.Total
			totalUAH = order.TotalUAH
			currency = order.Currency
			found = true
			break
		}
		if !found {
			return nil, errors.New("order not found")
		}
	} else if entityType == "sale" {
		saleID = &entityID
		found := false
		for _, sale := range s.sales {
			if sale.ID != entityID {
				continue
			}
			total = sale.Total
			totalUAH = sale.TotalUAH
			currency = sale.Currency
			found = true
			break
		}
		if !found {
			return nil, errors.New("sale not found")
		}
	} else {
		serviceOrderID = &entityID
		found := false
		for _, serviceOrder := range s.serviceOrders {
			if serviceOrder.ID != entityID {
				continue
			}
			total = serviceOrder.Price + serviceOrder.PartsTotal
			rateToUAH, err := s.rateToUAHLocked(serviceOrder.Currency)
			if err != nil {
				return nil, err
			}
			totalUAH = total * rateToUAH
			currency = serviceOrder.Currency
			found = true
			break
		}
		if !found {
			return nil, errors.New("service order not found")
		}
	}

	payments := []Payment{}
	for _, payment := range s.payments {
		if orderID != nil {
			if payment.OrderID == nil || *payment.OrderID != *orderID {
				continue
			}
		}
		if saleID != nil {
			if payment.SaleID == nil || *payment.SaleID != *saleID {
				continue
			}
		}
		if serviceOrderID != nil {
			if payment.ServiceOrderID == nil || *payment.ServiceOrderID != *serviceOrderID {
				continue
			}
		}
		payments = append(payments, payment)
	}
	sort.Slice(payments, func(i, j int) bool {
		return payments[i].CreatedAt.Before(payments[j].CreatedAt)
	})

	history := []DebtPaymentHistoryEntry{}
	remaining := total
	remainingUAH := totalUAH
	for _, payment := range payments {
		remaining -= payment.Amount
		remainingUAH -= payment.AmountUAH
		if remaining < 0 {
			remaining = 0
		}
		if remainingUAH < 0 {
			remainingUAH = 0
		}
		history = append(history, DebtPaymentHistoryEntry{
			PaymentID:        payment.ID,
			Amount:           payment.Amount,
			AmountUAH:        payment.AmountUAH,
			Currency:         currency,
			Method:           payment.Method,
			Note:             payment.Note,
			CreatedAt:        payment.CreatedAt,
			RemainingDebt:    remaining,
			RemainingDebtUAH: remainingUAH,
		})
	}
	return history, nil
}

func (s *Store) ListNotificationTemplates(code string) ([]NotificationTemplate, error) {
	if s.db != nil {
		if code == "" {
			rows, err := s.db.Query(`
				SELECT id, code, channel, subject, body, is_active, created_at, updated_at
				FROM notification_templates
				ORDER BY id ASC
			`)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			return scanNotificationTemplates(rows)
		}
		rows, err := s.db.Query(`
			SELECT id, code, channel, subject, body, is_active, created_at, updated_at
			FROM notification_templates
			WHERE code = $1
			ORDER BY id ASC
		`, code)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanNotificationTemplates(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []NotificationTemplate{}
	for _, tpl := range s.notificationTemplates {
		if code != "" && tpl.Code != code {
			continue
		}
		result = append(result, tpl)
	}
	return result, nil
}

func (s *Store) UpsertNotificationTemplate(input NotificationTemplate) (NotificationTemplate, error) {
	input.Code = strings.TrimSpace(input.Code)
	input.Channel = strings.TrimSpace(input.Channel)
	if input.Code == "" || input.Channel == "" || input.Body == "" {
		return NotificationTemplate{}, errors.New("необхідно вказати код, канал і текст шаблону")
	}
	if input.Channel != NotificationChannelEmail &&
		input.Channel != NotificationChannelTelegram &&
		input.Channel != NotificationChannelSMS &&
		input.Channel != NotificationChannelViber {
		return NotificationTemplate{}, errors.New("unsupported notification channel")
	}

	if s.db != nil {
		tpl := NotificationTemplate{}
		if err := s.db.QueryRow(`
			INSERT INTO notification_templates (code, channel, subject, body, is_active, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (code, channel) DO UPDATE
			SET subject = EXCLUDED.subject,
				body = EXCLUDED.body,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()
			RETURNING id, code, channel, subject, body, is_active, created_at, updated_at
		`,
			input.Code,
			input.Channel,
			input.Subject,
			input.Body,
			input.IsActive,
		).Scan(
			&tpl.ID,
			&tpl.Code,
			&tpl.Channel,
			&tpl.Subject,
			&tpl.Body,
			&tpl.IsActive,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		); err != nil {
			return NotificationTemplate{}, err
		}
		return tpl, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.notificationTemplates {
		if s.notificationTemplates[i].Code == input.Code && s.notificationTemplates[i].Channel == input.Channel {
			s.notificationTemplates[i].Subject = input.Subject
			s.notificationTemplates[i].Body = input.Body
			s.notificationTemplates[i].IsActive = input.IsActive
			s.notificationTemplates[i].UpdatedAt = time.Now().UTC()
			return s.notificationTemplates[i], nil
		}
	}
	s.templateSeq++
	input.ID = s.templateSeq
	now := time.Now().UTC()
	input.CreatedAt = now
	input.UpdatedAt = now
	s.notificationTemplates = append(s.notificationTemplates, input)
	return input, nil
}

func (s *Store) ListNotifications(limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, channel, recipient, subject, body, entity_type, entity_id, status, attempts, error_message, sent_at, created_at
			FROM notifications
			ORDER BY id DESC
			LIMIT $1
		`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanNotifications(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit > len(s.notifications) {
		limit = len(s.notifications)
	}
	result := make([]Notification, 0, limit)
	for i := len(s.notifications) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, s.notifications[i])
	}
	return result, nil
}

func (s *Store) ListBackgroundJobs(limit int) ([]BackgroundJob, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, job_type, status, attempts, max_attempts, next_retry_at, payload, result, error_message, started_at, finished_at, created_at
			FROM background_jobs
			ORDER BY id DESC
			LIMIT $1
		`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanBackgroundJobs(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit > len(s.backgroundJobs) {
		limit = len(s.backgroundJobs)
	}
	result := make([]BackgroundJob, 0, limit)
	for i := len(s.backgroundJobs) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, s.backgroundJobs[i])
	}
	return result, nil
}

func (s *Store) SendQuickMessage(input QuickMessageRequest) (Notification, error) {
	input.Channel = strings.TrimSpace(strings.ToLower(input.Channel))
	input.Recipient = strings.TrimSpace(input.Recipient)
	input.Sender = strings.TrimSpace(input.Sender)
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	if input.Channel != NotificationChannelEmail &&
		input.Channel != NotificationChannelTelegram &&
		input.Channel != NotificationChannelSMS &&
		input.Channel != NotificationChannelViber {
		return Notification{}, errors.New("unsupported notification channel")
	}
	if input.Recipient == "" || input.Body == "" {
		return Notification{}, errors.New("необхідно вказати отримувача і текст повідомлення")
	}

	// Actually send the message
	if s.notificationSender == nil || !s.notificationSender.Enabled() {
		return Notification{}, errors.New("сповіщення не налаштовано: перевірте змінні середовища (.env)")
	}
	if err := s.notificationSender.SendFrom(input.Channel, input.Sender, input.Recipient, input.Subject, input.Body); err != nil {
		return Notification{}, err
	}

	now := time.Now().UTC()
	notification := Notification{
		Channel:    input.Channel,
		Recipient:  input.Recipient,
		Subject:    input.Subject,
		Body:       input.Body,
		EntityType: input.EntityType,
		EntityID:   input.EntityID,
		Status:     NotificationStatusSentStub,
		SentAt:     &now,
		CreatedAt:  now,
	}
	if s.db != nil {
		if err := s.db.QueryRow(`
			INSERT INTO notifications (channel, recipient, subject, body, entity_type, entity_id, status, attempts, sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8)
			RETURNING id, created_at
		`, notification.Channel, notification.Recipient, notification.Subject, notification.Body, notification.EntityType, notification.EntityID, notification.Status, notification.SentAt).Scan(&notification.ID, &notification.CreatedAt); err != nil {
			return Notification{}, err
		}
		return notification, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notificationSeq++
	notification.ID = s.notificationSeq
	s.notifications = append(s.notifications, notification)
	return notification, nil
}

func (s *Store) EnqueueOverdueReminderJob(asOf time.Time) (BackgroundJob, error) {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	payload := asOf.Format(time.RFC3339)
	if s.db != nil {
		job := BackgroundJob{}
		if err := s.db.QueryRow(`
			INSERT INTO background_jobs (job_type, status, attempts, max_attempts, next_retry_at, payload)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, job_type, status, attempts, max_attempts, next_retry_at, payload, result, error_message, started_at, finished_at, created_at
		`,
			BackgroundJobTypeOverdueReminders,
			BackgroundJobStatusQueued,
			0,
			5,
			asOf,
			payload,
		).Scan(
			&job.ID,
			&job.JobType,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.NextRetryAt,
			&job.Payload,
			&job.Result,
			&job.ErrorMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.CreatedAt,
		); err != nil {
			return BackgroundJob{}, err
		}
		return job, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobSeq++
	job := BackgroundJob{
		ID:          s.jobSeq,
		JobType:     BackgroundJobTypeOverdueReminders,
		Status:      BackgroundJobStatusQueued,
		Attempts:    0,
		MaxAttempts: 5,
		NextRetryAt: &asOf,
		Payload:     payload,
		Result:      "",
		CreatedAt:   time.Now().UTC(),
	}
	s.backgroundJobs = append(s.backgroundJobs, job)
	return job, nil
}

func (s *Store) EnqueueReservationExpiryJob(asOf time.Time) (BackgroundJob, error) {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	payload := asOf.Format(time.RFC3339)
	if s.db != nil {
		job := BackgroundJob{}
		if err := s.db.QueryRow(`
			INSERT INTO background_jobs (job_type, status, attempts, max_attempts, next_retry_at, payload)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, job_type, status, attempts, max_attempts, next_retry_at, payload, result, error_message, started_at, finished_at, created_at
		`,
			BackgroundJobTypeReservationExpiry,
			BackgroundJobStatusQueued,
			0,
			3,
			asOf,
			payload,
		).Scan(
			&job.ID,
			&job.JobType,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.NextRetryAt,
			&job.Payload,
			&job.Result,
			&job.ErrorMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.CreatedAt,
		); err != nil {
			return BackgroundJob{}, err
		}
		return job, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobSeq++
	job := BackgroundJob{
		ID:          s.jobSeq,
		JobType:     BackgroundJobTypeReservationExpiry,
		Status:      BackgroundJobStatusQueued,
		Attempts:    0,
		MaxAttempts: 3,
		NextRetryAt: &asOf,
		Payload:     payload,
		Result:      "",
		CreatedAt:   time.Now().UTC(),
	}
	s.backgroundJobs = append(s.backgroundJobs, job)
	return job, nil
}

func (s *Store) EnqueueReceiptRetryJob(status string, limit int) (BackgroundJob, error) {
	normalizedStatus, err := normalizeReceiptRetryJobStatus(status)
	if err != nil {
		return BackgroundJob{}, err
	}
	normalizedLimit := normalizeReceiptRetryJobLimit(limit)
	payload, err := encodeReceiptRetryJobPayload(normalizedStatus, normalizedLimit)
	if err != nil {
		return BackgroundJob{}, err
	}
	nextRunAt := time.Now().UTC()

	if s.db != nil {
		job := BackgroundJob{}
		if err := s.db.QueryRow(`
			INSERT INTO background_jobs (job_type, status, attempts, max_attempts, next_retry_at, payload)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, job_type, status, attempts, max_attempts, next_retry_at, payload, result, error_message, started_at, finished_at, created_at
		`,
			BackgroundJobTypeReceiptRetries,
			BackgroundJobStatusQueued,
			0,
			5,
			nextRunAt,
			payload,
		).Scan(
			&job.ID,
			&job.JobType,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.NextRetryAt,
			&job.Payload,
			&job.Result,
			&job.ErrorMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.CreatedAt,
		); err != nil {
			return BackgroundJob{}, err
		}
		return job, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobSeq++
	job := BackgroundJob{
		ID:          s.jobSeq,
		JobType:     BackgroundJobTypeReceiptRetries,
		Status:      BackgroundJobStatusQueued,
		Attempts:    0,
		MaxAttempts: 5,
		NextRetryAt: &nextRunAt,
		Payload:     payload,
		Result:      "",
		CreatedAt:   time.Now().UTC(),
	}
	s.backgroundJobs = append(s.backgroundJobs, job)
	return job, nil
}

func (s *Store) RunDueBackgroundJobs() ([]BackgroundJob, error) {
	if s.db != nil {
		jobs, err := s.ListBackgroundJobs(200)
		if err != nil {
			return nil, err
		}
		processed := []BackgroundJob{}
		now := time.Now().UTC()
		for _, job := range jobs {
			if job.Status != BackgroundJobStatusQueued && job.Status != BackgroundJobStatusFailed {
				continue
			}
			if job.NextRetryAt != nil && job.NextRetryAt.After(now) {
				continue
			}
			if job.Attempts >= job.MaxAttempts {
				continue
			}
			updated, err := s.runBackgroundJobByID(job.ID)
			if err != nil {
				return nil, err
			}
			processed = append(processed, updated)
		}
		return processed, nil
	}

	s.mu.Lock()
	queue := []BackgroundJob{}
	now := time.Now().UTC()
	for _, job := range s.backgroundJobs {
		if job.Status != BackgroundJobStatusQueued && job.Status != BackgroundJobStatusFailed {
			continue
		}
		if job.NextRetryAt != nil && job.NextRetryAt.After(now) {
			continue
		}
		if job.Attempts >= job.MaxAttempts {
			continue
		}
		queue = append(queue, job)
	}
	s.mu.Unlock()

	processed := []BackgroundJob{}
	for _, job := range queue {
		updated, err := s.runBackgroundJobByID(job.ID)
		if err != nil {
			return nil, err
		}
		processed = append(processed, updated)
	}
	return processed, nil
}

func (s *Store) runBackgroundJobByID(jobID int64) (BackgroundJob, error) {
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return BackgroundJob{}, err
		}
		defer tx.Rollback()

		job := BackgroundJob{}
		if err := tx.QueryRow(`
			SELECT id, job_type, status, attempts, max_attempts, next_retry_at, payload, result, error_message, started_at, finished_at, created_at
			FROM background_jobs
			WHERE id = $1
			FOR UPDATE
		`, jobID).Scan(
			&job.ID,
			&job.JobType,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.NextRetryAt,
			&job.Payload,
			&job.Result,
			&job.ErrorMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.CreatedAt,
		); err != nil {
			return BackgroundJob{}, err
		}
		now := time.Now().UTC()
		if job.Status != BackgroundJobStatusQueued && job.Status != BackgroundJobStatusFailed {
			return job, tx.Commit()
		}
		if job.NextRetryAt != nil && job.NextRetryAt.After(now) {
			return job, tx.Commit()
		}
		if job.Attempts >= job.MaxAttempts {
			return job, tx.Commit()
		}

		started := now
		if _, err := tx.Exec(`
			UPDATE background_jobs
			SET status = $2, started_at = $3, attempts = attempts + 1
			WHERE id = $1
		`, jobID, BackgroundJobStatusRunning, started); err != nil {
			return BackgroundJob{}, err
		}
		job.Attempts++

		if job.JobType == BackgroundJobTypeReservationExpiry {
			asOf := started
			if parsed, err := time.Parse(time.RFC3339, job.Payload); err == nil {
				asOf = parsed
			}
			released, err := s.releaseExpiredReservationsTx(tx, asOf)
			if err != nil {
				return BackgroundJob{}, err
			}
			result := fmt.Sprintf("expired_reservations=%d", released)
			if _, err := tx.Exec(`
				UPDATE background_jobs
				SET status = $2, result = $3, next_retry_at = NULL, finished_at = $4, error_message = ''
				WHERE id = $1
			`, jobID, BackgroundJobStatusCompleted, result, time.Now().UTC()); err != nil {
				return BackgroundJob{}, err
			}
			if err := tx.Commit(); err != nil {
				return BackgroundJob{}, err
			}
			return s.getBackgroundJobByID(jobID)
		}

		if job.JobType == BackgroundJobTypeReceiptRetries {
			result, err := s.runReceiptRetryBackgroundJobTx(tx, job.Payload)
			if err != nil {
				nextRetry := started.Add(backgroundJobBackoff(job.Attempts))
				nextStatus := BackgroundJobStatusFailed
				if job.Attempts >= job.MaxAttempts {
					nextRetry = time.Time{}
				}
				if _, updateErr := tx.Exec(`
					UPDATE background_jobs
					SET status = $2, error_message = $3, next_retry_at = $4, finished_at = $5
					WHERE id = $1
				`, jobID, nextStatus, err.Error(), nullableTime(nextRetry), time.Now().UTC()); updateErr != nil {
					return BackgroundJob{}, updateErr
				}
				if commitErr := tx.Commit(); commitErr != nil {
					return BackgroundJob{}, commitErr
				}
				return s.getBackgroundJobByID(jobID)
			}
			if _, err := tx.Exec(`
				UPDATE background_jobs
				SET status = $2, result = $3, next_retry_at = NULL, finished_at = $4, error_message = ''
				WHERE id = $1
			`, jobID, BackgroundJobStatusCompleted, result, time.Now().UTC()); err != nil {
				return BackgroundJob{}, err
			}
			if err := tx.Commit(); err != nil {
				return BackgroundJob{}, err
			}
			return s.getBackgroundJobByID(jobID)
		}

		asOf := started
		if parsed, err := time.Parse(time.RFC3339, job.Payload); err == nil {
			asOf = parsed
		}

		templates, err := s.loadActiveReminderTemplatesTx(tx)
		if err != nil {
			return BackgroundJob{}, err
		}
		overdue, err := s.listOverdueDebtsTx(tx, asOf)
		if err != nil {
			return BackgroundJob{}, err
		}
		sentCount, err := s.sendOverdueNotificationsTx(tx, templates, overdue)
		if err != nil {
			nextRetry := started.Add(backgroundJobBackoff(job.Attempts))
			nextStatus := BackgroundJobStatusFailed
			if job.Attempts >= job.MaxAttempts {
				nextRetry = time.Time{}
			}
			if _, updateErr := tx.Exec(`
				UPDATE background_jobs
				SET status = $2, error_message = $3, next_retry_at = $4, finished_at = $5
				WHERE id = $1
			`, jobID, nextStatus, err.Error(), nullableTime(nextRetry), time.Now().UTC()); updateErr != nil {
				return BackgroundJob{}, updateErr
			}
			if commitErr := tx.Commit(); commitErr != nil {
				return BackgroundJob{}, commitErr
			}
			return s.getBackgroundJobByID(jobID)
		}

		result := fmt.Sprintf("overdue=%d notifications=%d", len(overdue), sentCount)
		if _, err := tx.Exec(`
			UPDATE background_jobs
			SET status = $2, result = $3, next_retry_at = NULL, finished_at = $4
			WHERE id = $1
		`, jobID, BackgroundJobStatusCompleted, result, time.Now().UTC()); err != nil {
			return BackgroundJob{}, err
		}

		if err := tx.Commit(); err != nil {
			return BackgroundJob{}, err
		}
		return s.getBackgroundJobByID(jobID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i := range s.backgroundJobs {
		if s.backgroundJobs[i].ID == jobID {
			index = i
			break
		}
	}
	if index == -1 {
		return BackgroundJob{}, errors.New("job not found")
	}
	job := s.backgroundJobs[index]
	now := time.Now().UTC()
	if job.Status != BackgroundJobStatusQueued && job.Status != BackgroundJobStatusFailed {
		return job, nil
	}
	if job.NextRetryAt != nil && job.NextRetryAt.After(now) {
		return job, nil
	}
	if job.Attempts >= job.MaxAttempts {
		return job, nil
	}
	s.backgroundJobs[index].Attempts++
	s.backgroundJobs[index].Status = BackgroundJobStatusRunning
	s.backgroundJobs[index].StartedAt = &now

	if job.JobType == BackgroundJobTypeReservationExpiry {
		asOf := now
		if parsed, err := time.Parse(time.RFC3339, job.Payload); err == nil {
			asOf = parsed
		}
		released := s.releaseExpiredReservationsLocked(asOf)
		finished := time.Now().UTC()
		s.backgroundJobs[index].Status = BackgroundJobStatusCompleted
		s.backgroundJobs[index].Result = fmt.Sprintf("expired_reservations=%d", released)
		s.backgroundJobs[index].ErrorMessage = ""
		s.backgroundJobs[index].NextRetryAt = nil
		s.backgroundJobs[index].FinishedAt = &finished
		return s.backgroundJobs[index], nil
	}

	if job.JobType == BackgroundJobTypeReceiptRetries {
		result, err := s.runReceiptRetryBackgroundJobLocked(job.Payload)
		finished := time.Now().UTC()
		if err != nil {
			s.backgroundJobs[index].Status = BackgroundJobStatusFailed
			s.backgroundJobs[index].ErrorMessage = err.Error()
			nextRetry := finished.Add(backgroundJobBackoff(s.backgroundJobs[index].Attempts))
			if s.backgroundJobs[index].Attempts < s.backgroundJobs[index].MaxAttempts {
				s.backgroundJobs[index].NextRetryAt = &nextRetry
			}
			s.backgroundJobs[index].FinishedAt = &finished
			return s.backgroundJobs[index], nil
		}

		s.backgroundJobs[index].Status = BackgroundJobStatusCompleted
		s.backgroundJobs[index].Result = result
		s.backgroundJobs[index].ErrorMessage = ""
		s.backgroundJobs[index].NextRetryAt = nil
		s.backgroundJobs[index].FinishedAt = &finished
		return s.backgroundJobs[index], nil
	}

	asOf := now
	if parsed, err := time.Parse(time.RFC3339, job.Payload); err == nil {
		asOf = parsed
	}
	overdue := []DebtSummary{}
	for _, debt := range s.mustListDebtsLocked() {
		if isDebtOverdue(debt.DueDate, debt.DebtUAH, asOf) {
			overdue = append(overdue, debt)
		}
	}
	sentCount := 0
	var sendErr error
	for _, tpl := range s.notificationTemplates {
		if tpl.Code != BackgroundJobTypeOverdueReminders || !tpl.IsActive {
			continue
		}
		for _, debt := range overdue {
			s.notificationSeq++
			body := renderTemplateBody(tpl.Body, debt)
			recipient := notificationRecipientForDebt(tpl.Channel, debt, s.notificationSender)
			sentAt := time.Now().UTC()
			status := NotificationStatusSentStub
			errMessage := ""
			attempts := 0
			if s.notificationSender != nil && s.notificationSender.Enabled() {
				attempts = 1
				if err := s.notificationSender.Send(tpl.Channel, recipient, tpl.Subject, body); err != nil {
					status = NotificationStatusFailed
					errMessage = err.Error()
					if sendErr == nil {
						sendErr = err
					}
				} else {
					status = NotificationStatusSent
				}
			}
			s.notifications = append(s.notifications, Notification{
				ID:           s.notificationSeq,
				Channel:      tpl.Channel,
				Recipient:    recipient,
				Subject:      tpl.Subject,
				Body:         body,
				EntityType:   debt.EntityType,
				EntityID:     debt.EntityID,
				Status:       status,
				Attempts:     attempts,
				ErrorMessage: errMessage,
				SentAt:       &sentAt,
				CreatedAt:    sentAt,
			})
			sentCount++
		}
	}
	finished := time.Now().UTC()
	if sendErr != nil {
		s.backgroundJobs[index].Status = BackgroundJobStatusFailed
		s.backgroundJobs[index].ErrorMessage = sendErr.Error()
		nextRetry := finished.Add(backgroundJobBackoff(s.backgroundJobs[index].Attempts))
		if s.backgroundJobs[index].Attempts < s.backgroundJobs[index].MaxAttempts {
			s.backgroundJobs[index].NextRetryAt = &nextRetry
		}
	} else {
		s.backgroundJobs[index].Status = BackgroundJobStatusCompleted
		s.backgroundJobs[index].Result = fmt.Sprintf("overdue=%d notifications=%d", len(overdue), sentCount)
		s.backgroundJobs[index].NextRetryAt = nil
	}
	s.backgroundJobs[index].FinishedAt = &finished
	return s.backgroundJobs[index], nil
}

func (s *Store) CreateSale(items []SaleItem) (Sale, error) {
	return s.CreateSaleFromOrder(items, nil, "UAH")
}

func (s *Store) ListSales() ([]Sale, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, order_id, total, currency, total_uah, status, created_at
			FROM sales
			ORDER BY id DESC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result := []Sale{}
		for rows.Next() {
			var sale Sale
			if err := rows.Scan(&sale.ID, &sale.OrderID, &sale.Total, &sale.Currency, &sale.TotalUAH, &sale.Status, &sale.CreatedAt); err != nil {
				return nil, err
			}
			items, err := s.saleItemsBySaleID(sale.ID)
			if err != nil {
				return nil, err
			}
			sale.Items = items
			result = append(result, sale)
		}
		return result, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Sale, len(s.sales))
	copy(result, s.sales)
	sort.Slice(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

func (s *Store) CreateSaleFromOrder(items []SaleItem, orderID *int64, currency string) (Sale, error) {
	currency = normalizeCurrency(currency)
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Sale{}, err
		}
		defer tx.Rollback()

		if len(items) == 0 {
			return Sale{}, errors.New("sale items are required")
		}

		var total float64
		saleCurrency := currency
		if orderID != nil {
			if err := tx.QueryRow(`SELECT currency FROM customer_orders WHERE id = $1`, *orderID).Scan(&saleCurrency); err != nil {
				return Sale{}, err
			}
		}
		for _, item := range items {
			var currentStock int
			err := tx.QueryRow(
				`SELECT stock FROM products WHERE id = $1 FOR UPDATE`,
				item.ProductID,
			).Scan(&currentStock)
			if errors.Is(err, sql.ErrNoRows) {
				return Sale{}, ErrProductNotFound
			}
			if err != nil {
				return Sale{}, err
			}

			available := currentStock
			if orderID == nil {
				reservedByOthers, err := s.activeReservedByOtherOrdersInTx(tx, item.ProductID, nil)
				if err != nil {
					return Sale{}, err
				}
				available -= reservedByOthers
			}

			if available < item.Quantity {
				return Sale{}, ErrInsufficientStock
			}
			total += float64(item.Quantity) * item.Price
		}
		rateToUAH, err := s.rateToUAHInTx(tx, saleCurrency)
		if err != nil {
			return Sale{}, err
		}
		defaultWarehouseID, err := s.defaultWarehouseIDInTx(tx)
		if err != nil {
			return Sale{}, err
		}

		for _, item := range items {
			if _, err := tx.Exec(
				`UPDATE products SET stock = stock - $1 WHERE id = $2`,
				item.Quantity,
				item.ProductID,
			); err != nil {
				return Sale{}, err
			}
			if err := s.adjustWarehouseStockInTx(tx, defaultWarehouseID, item.ProductID, "sale", item.Quantity); err != nil {
				return Sale{}, err
			}

			if _, err := tx.Exec(`
				INSERT INTO stock_movements (product_id, from_warehouse_id, to_warehouse_id, movement_type, quantity, note)
				VALUES ($1, $2, $3, 'sale', $4, $5)
			`,
				item.ProductID,
				defaultWarehouseID,
				nil,
				item.Quantity,
				"Auto movement from sale",
			); err != nil {
				return Sale{}, err
			}
		}

		sale := Sale{
			OrderID:  orderID,
			Items:    items,
			Total:    total,
			Currency: saleCurrency,
			TotalUAH: total * rateToUAH,
			Status:   "completed",
		}
		err = tx.QueryRow(`
			INSERT INTO sales (order_id, total, currency, total_uah, status)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`,
			orderID,
			sale.Total,
			sale.Currency,
			sale.TotalUAH,
			sale.Status,
		).Scan(&sale.ID, &sale.CreatedAt)
		if err != nil {
			return Sale{}, err
		}

		for _, item := range items {
			if _, err := tx.Exec(`
				INSERT INTO sale_items (sale_id, product_id, quantity, price)
				VALUES ($1, $2, $3, $4)
			`,
				sale.ID,
				item.ProductID,
				item.Quantity,
				item.Price,
			); err != nil {
				return Sale{}, err
			}
		}

		if orderID != nil {
			if _, err := tx.Exec(`
				UPDATE reservations
				SET status = $2, released_at = NOW()
				WHERE order_id = $1 AND status = $3
			`,
				*orderID,
				ReservationStatusConsumed,
				ReservationStatusActive,
			); err != nil {
				return Sale{}, err
			}

			if _, err := tx.Exec(`
				UPDATE customer_orders
				SET status = $2, updated_at = NOW()
				WHERE id = $1
			`,
				*orderID,
				OrderStatusIssued,
			); err != nil {
				return Sale{}, err
			}
		}

		if err := tx.Commit(); err != nil {
			return Sale{}, err
		}
		receipt, err := s.ensureSaleReceiptForSale(sale)
		if err != nil {
			log.Printf("failed to create pending receipt for sale %d: %v", sale.ID, err)
		} else {
			s.tryAutoSendReceipt(receipt.ID)
		}
		return sale, nil
	}

	sale, receiptID, err := func() (Sale, int64, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if len(items) == 0 {
			return Sale{}, 0, errors.New("sale items are required")
		}

		positions := map[int64]int{}
		for index := range s.products {
			positions[s.products[index].ID] = index
		}

		var total float64
		saleCurrency := currency
		if orderID != nil {
			for _, order := range s.orders {
				if order.ID == *orderID {
					saleCurrency = order.Currency
					break
				}
			}
		}
		for _, item := range items {
			pos, ok := positions[item.ProductID]
			if !ok {
				return Sale{}, 0, ErrProductNotFound
			}
			available := s.products[pos].Stock
			if orderID == nil {
				available -= s.activeReservedByOtherOrdersLocked(item.ProductID, nil)
			}
			if available < item.Quantity {
				return Sale{}, 0, ErrInsufficientStock
			}
			total += float64(item.Quantity) * item.Price
		}
		rateToUAH, err := s.rateToUAHLocked(saleCurrency)
		if err != nil {
			return Sale{}, 0, err
		}

		for _, item := range items {
			pos := positions[item.ProductID]
			s.products[pos].Stock -= item.Quantity
			defaultWarehouseID := int64(1)
			s.adjustWarehouseStockLocked(defaultWarehouseID, item.ProductID, "sale", item.Quantity)

			s.movementSeq++
			movement := StockMovement{
				ID:              s.movementSeq,
				ProductID:       item.ProductID,
				FromWarehouseID: &defaultWarehouseID,
				Type:            "sale",
				Quantity:        item.Quantity,
				Note:            "Auto movement from sale",
				CreatedAt:       time.Now().UTC(),
			}
			s.movements = append(s.movements, movement)
		}

		s.saleSeq++
		sale := Sale{
			ID:        s.saleSeq,
			OrderID:   orderID,
			Items:     items,
			Total:     total,
			Currency:  saleCurrency,
			TotalUAH:  total * rateToUAH,
			Status:    "completed",
			CreatedAt: time.Now().UTC(),
		}
		s.sales = append(s.sales, sale)

		if orderID != nil {
			releasedAt := time.Now().UTC()
			for index := range s.reservations {
				if s.reservations[index].OrderID == *orderID && s.reservations[index].Status == ReservationStatusActive {
					s.reservations[index].Status = ReservationStatusConsumed
					s.reservations[index].ReleasedAt = &releasedAt
				}
			}
			for index := range s.orders {
				if s.orders[index].ID == *orderID {
					s.orders[index].Status = OrderStatusIssued
					s.orders[index].UpdatedAt = releasedAt
				}
			}
		}

		receipt, err := s.ensureSaleReceiptForSaleLocked(sale)
		if err != nil {
			log.Printf("failed to create pending receipt for sale %d: %v", sale.ID, err)
			return sale, 0, nil
		}
		return sale, receipt.ID, nil
	}()
	if err != nil {
		return Sale{}, err
	}
	s.tryAutoSendReceipt(receiptID)
	return sale, nil
}

func (s *Store) ensureSaleReceiptForSale(sale Sale) (Receipt, error) {
	if sale.ID <= 0 {
		return Receipt{}, errors.New("sale id is required")
	}
	if s.db != nil {
		if _, err := s.db.Exec(`
			INSERT INTO receipts (sale_id, provider, status, payload)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (sale_id) DO NOTHING
		`,
			sale.ID,
			ReceiptProviderCheckbox,
			ReceiptStatusPending,
			receiptPayloadForSale(sale),
		); err != nil {
			return Receipt{}, err
		}
		return s.ReceiptBySaleID(sale.ID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensureSaleReceiptForSaleLocked(sale)
}

func (s *Store) ensureSaleReceiptForSaleLocked(sale Sale) (Receipt, error) {
	for _, receipt := range s.receipts {
		if receipt.SaleID == sale.ID {
			return receipt, nil
		}
	}

	s.receiptSeq++
	now := time.Now().UTC()
	receipt := Receipt{
		ID:        s.receiptSeq,
		SaleID:    sale.ID,
		Provider:  ReceiptProviderCheckbox,
		Status:    ReceiptStatusPending,
		Payload:   receiptPayloadForSale(sale),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.receipts = append(s.receipts, receipt)
	return receipt, nil
}

func receiptPayloadForSale(sale Sale) string {
	return fmt.Sprintf(
		"sale_id=%d total=%.2f currency=%s items=%d",
		sale.ID,
		sale.Total,
		sale.Currency,
		len(sale.Items),
	)
}

func (s *Store) tryAutoSendReceipt(receiptID int64) {
	if receiptID <= 0 {
		return
	}
	if _, err := s.RetryReceipt(receiptID); err != nil {
		if errors.Is(err, ErrReceiptSenderNotConfigured) {
			return
		}
		log.Printf("failed to auto-send receipt %d: %v", receiptID, err)
		if _, enqueueErr := s.EnqueueReceiptRetryJob(ReceiptStatusFailed, 20); enqueueErr != nil {
			log.Printf("failed to enqueue receipt retry background job: %v", enqueueErr)
		}
	}
}

func isValidReceiptStatus(status string) bool {
	switch status {
	case ReceiptStatusPending, ReceiptStatusSent, ReceiptStatusFailed:
		return true
	default:
		return false
	}
}

func (s *Store) ListReceipts(saleID *int64, status *string) ([]Receipt, error) {
	var filterStatus *string
	if status != nil {
		normalized := strings.TrimSpace(strings.ToLower(*status))
		if normalized != "" {
			if !isValidReceiptStatus(normalized) {
				return nil, errors.New("invalid receipt status")
			}
			filterStatus = &normalized
		}
	}

	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, sale_id, provider, status, external_id, fiscal_number, qr_url, error_message, payload, sent_at, created_at, updated_at
			FROM receipts
			WHERE ($1::BIGINT IS NULL OR sale_id = $1)
				AND ($2::TEXT IS NULL OR status = $2)
			ORDER BY id DESC
		`, saleID, filterStatus)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanReceipts(rows)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []Receipt{}
	for _, receipt := range s.receipts {
		if saleID != nil && receipt.SaleID != *saleID {
			continue
		}
		if filterStatus != nil && receipt.Status != *filterStatus {
			continue
		}
		result = append(result, receipt)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

func (s *Store) ReceiptByID(receiptID int64) (Receipt, error) {
	if receiptID <= 0 {
		return Receipt{}, ErrReceiptNotFound
	}
	if s.db != nil {
		row := s.db.QueryRow(`
			SELECT id, sale_id, provider, status, external_id, fiscal_number, qr_url, error_message, payload, sent_at, created_at, updated_at
			FROM receipts
			WHERE id = $1
		`, receiptID)
		receipt, err := scanReceipt(row)
		if errors.Is(err, sql.ErrNoRows) {
			return Receipt{}, ErrReceiptNotFound
		}
		return receipt, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, receipt := range s.receipts {
		if receipt.ID == receiptID {
			return receipt, nil
		}
	}
	return Receipt{}, ErrReceiptNotFound
}

func (s *Store) ReceiptBySaleID(saleID int64) (Receipt, error) {
	if saleID <= 0 {
		return Receipt{}, ErrReceiptNotFound
	}
	if s.db != nil {
		row := s.db.QueryRow(`
			SELECT id, sale_id, provider, status, external_id, fiscal_number, qr_url, error_message, payload, sent_at, created_at, updated_at
			FROM receipts
			WHERE sale_id = $1
		`, saleID)
		receipt, err := scanReceipt(row)
		if errors.Is(err, sql.ErrNoRows) {
			return Receipt{}, ErrReceiptNotFound
		}
		return receipt, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, receipt := range s.receipts {
		if receipt.SaleID == saleID {
			return receipt, nil
		}
	}
	return Receipt{}, ErrReceiptNotFound
}

func (s *Store) RetryReceipt(receiptID int64) (Receipt, error) {
	receipt, err := s.ReceiptByID(receiptID)
	if err != nil {
		return Receipt{}, err
	}

	sale, err := s.saleByID(receipt.SaleID)
	if err != nil {
		return Receipt{}, err
	}

	sender := s.currentReceiptSender()
	if sender == nil || !sender.Enabled() {
		return receipt, ErrReceiptSenderNotConfigured
	}

	result, err := sender.SendSaleReceipt(sale)
	if err != nil {
		updated, updateErr := s.markReceiptFailed(receiptID, err.Error())
		if updateErr != nil {
			return Receipt{}, updateErr
		}
		return updated, err
	}

	return s.markReceiptSent(receiptID, result)
}

func (s *Store) RetryReceiptsBulk(limit int, status *string) (ReceiptBulkRetryResult, error) {
	result := ReceiptBulkRetryResult{
		Items: []Receipt{},
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	receipts, err := s.ListReceipts(nil, nil)
	if err != nil {
		return result, err
	}

	var filterStatus string
	if status != nil {
		filterStatus = strings.TrimSpace(strings.ToLower(*status))
		if filterStatus != "" && !isValidReceiptStatus(filterStatus) {
			return result, errors.New("invalid receipt status")
		}
	}

	for _, receipt := range receipts {
		if result.Attempted >= limit {
			break
		}
		if !shouldRetryReceiptStatus(receipt.Status, filterStatus) {
			continue
		}

		updated, retryErr := s.RetryReceipt(receipt.ID)
		result.Attempted++
		if retryErr != nil {
			if errors.Is(retryErr, ErrReceiptSenderNotConfigured) {
				return result, retryErr
			}
			result.Failed++
			if updated.ID > 0 {
				result.Items = append(result.Items, updated)
			}
			continue
		}

		result.Succeeded++
		result.Items = append(result.Items, updated)
	}

	return result, nil
}

func shouldRetryReceiptStatus(currentStatus string, filterStatus string) bool {
	status := strings.TrimSpace(strings.ToLower(currentStatus))
	if filterStatus == "" {
		return status == ReceiptStatusPending || status == ReceiptStatusFailed
	}
	if filterStatus == ReceiptStatusSent {
		return false
	}
	return status == filterStatus
}

type receiptRetryJobPayload struct {
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func normalizeReceiptRetryJobStatus(status string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(status))
	switch normalized {
	case "":
		return "", nil
	case ReceiptStatusPending, ReceiptStatusFailed:
		return normalized, nil
	default:
		return "", errors.New("invalid receipt retry status")
	}
}

func normalizeReceiptRetryJobLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func encodeReceiptRetryJobPayload(status string, limit int) (string, error) {
	raw, err := json.Marshal(receiptRetryJobPayload{
		Status: status,
		Limit:  limit,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func parseReceiptRetryJobPayload(payload string) (receiptRetryJobPayload, error) {
	cfg := receiptRetryJobPayload{
		Limit: 20,
	}
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(trimmed), &cfg); err != nil {
		return receiptRetryJobPayload{}, err
	}
	status, err := normalizeReceiptRetryJobStatus(cfg.Status)
	if err != nil {
		return receiptRetryJobPayload{}, err
	}
	cfg.Status = status
	cfg.Limit = normalizeReceiptRetryJobLimit(cfg.Limit)
	return cfg, nil
}

func (s *Store) currentReceiptSender() ReceiptSender {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.receiptSender
}

func (s *Store) IsReceiptSenderEnabled() bool {
	sender := s.currentReceiptSender()
	return sender != nil && sender.Enabled()
}

func (s *Store) markReceiptFailed(receiptID int64, message string) (Receipt, error) {
	if s.db != nil {
		row := s.db.QueryRow(`
			UPDATE receipts
			SET status = $2, error_message = $3, sent_at = NULL, updated_at = NOW()
			WHERE id = $1
			RETURNING id, sale_id, provider, status, external_id, fiscal_number, qr_url, error_message, payload, sent_at, created_at, updated_at
		`, receiptID, ReceiptStatusFailed, message)
		updated, err := scanReceipt(row)
		if errors.Is(err, sql.ErrNoRows) {
			return Receipt{}, ErrReceiptNotFound
		}
		return updated, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.receipts {
		if s.receipts[index].ID != receiptID {
			continue
		}
		s.receipts[index].Status = ReceiptStatusFailed
		s.receipts[index].ErrorMessage = message
		s.receipts[index].SentAt = nil
		s.receipts[index].UpdatedAt = time.Now().UTC()
		return s.receipts[index], nil
	}
	return Receipt{}, ErrReceiptNotFound
}

func (s *Store) markReceiptSent(receiptID int64, result ReceiptSendResult) (Receipt, error) {
	if s.db != nil {
		row := s.db.QueryRow(`
			UPDATE receipts
			SET status = $2, external_id = $3, fiscal_number = $4, qr_url = $5, error_message = '', sent_at = NOW(), updated_at = NOW()
			WHERE id = $1
			RETURNING id, sale_id, provider, status, external_id, fiscal_number, qr_url, error_message, payload, sent_at, created_at, updated_at
		`,
			receiptID,
			ReceiptStatusSent,
			result.ExternalID,
			result.FiscalNumber,
			result.QRURL,
		)
		updated, err := scanReceipt(row)
		if errors.Is(err, sql.ErrNoRows) {
			return Receipt{}, ErrReceiptNotFound
		}
		return updated, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.receipts {
		if s.receipts[index].ID != receiptID {
			continue
		}
		sentAt := time.Now().UTC()
		s.receipts[index].Status = ReceiptStatusSent
		s.receipts[index].ExternalID = result.ExternalID
		s.receipts[index].FiscalNumber = result.FiscalNumber
		s.receipts[index].QRURL = result.QRURL
		s.receipts[index].ErrorMessage = ""
		s.receipts[index].SentAt = &sentAt
		s.receipts[index].UpdatedAt = sentAt
		return s.receipts[index], nil
	}
	return Receipt{}, ErrReceiptNotFound
}

func (s *Store) saleByID(saleID int64) (Sale, error) {
	if saleID <= 0 {
		return Sale{}, errors.New("sale not found")
	}
	if s.db != nil {
		var sale Sale
		if err := s.db.QueryRow(`
			SELECT id, order_id, total, currency, total_uah, status, created_at
			FROM sales
			WHERE id = $1
		`, saleID).Scan(
			&sale.ID,
			&sale.OrderID,
			&sale.Total,
			&sale.Currency,
			&sale.TotalUAH,
			&sale.Status,
			&sale.CreatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Sale{}, errors.New("sale not found")
			}
			return Sale{}, err
		}
		items, err := s.saleItemsBySaleID(sale.ID)
		if err != nil {
			return Sale{}, err
		}
		sale.Items = items
		return sale, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sale := range s.sales {
		if sale.ID == saleID {
			return sale, nil
		}
	}
	return Sale{}, errors.New("sale not found")
}

func (s *Store) runReceiptRetryBackgroundJobTx(tx *sql.Tx, payload string) (string, error) {
	cfg, err := parseReceiptRetryJobPayload(payload)
	if err != nil {
		return "", err
	}
	sender := s.currentReceiptSender()
	if sender == nil || !sender.Enabled() {
		return "", ErrReceiptSenderNotConfigured
	}

	receipts := []Receipt{}
	if cfg.Status == "" {
		rows, err := tx.Query(`
			SELECT id, sale_id, provider, status, external_id, fiscal_number, qr_url, error_message, payload, sent_at, created_at, updated_at
			FROM receipts
			WHERE status = $1 OR status = $2
			ORDER BY id DESC
			LIMIT $3
			FOR UPDATE
		`, ReceiptStatusPending, ReceiptStatusFailed, cfg.Limit)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		receipts, err = scanReceipts(rows)
		if err != nil {
			return "", err
		}
	} else {
		rows, err := tx.Query(`
			SELECT id, sale_id, provider, status, external_id, fiscal_number, qr_url, error_message, payload, sent_at, created_at, updated_at
			FROM receipts
			WHERE status = $1
			ORDER BY id DESC
			LIMIT $2
			FOR UPDATE
		`, cfg.Status, cfg.Limit)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		receipts, err = scanReceipts(rows)
		if err != nil {
			return "", err
		}
	}

	attempted := 0
	succeeded := 0
	failed := 0
	for _, receipt := range receipts {
		if !shouldRetryReceiptStatus(receipt.Status, cfg.Status) {
			continue
		}
		attempted++

		sale, err := s.saleByIDInTx(tx, receipt.SaleID)
		if err != nil {
			failed++
			if _, updateErr := tx.Exec(`
				UPDATE receipts
				SET status = $2, error_message = $3, sent_at = NULL, updated_at = NOW()
				WHERE id = $1
			`, receipt.ID, ReceiptStatusFailed, err.Error()); updateErr != nil {
				return "", updateErr
			}
			continue
		}

		sendResult, err := sender.SendSaleReceipt(sale)
		if err != nil {
			failed++
			if _, updateErr := tx.Exec(`
				UPDATE receipts
				SET status = $2, error_message = $3, sent_at = NULL, updated_at = NOW()
				WHERE id = $1
			`, receipt.ID, ReceiptStatusFailed, err.Error()); updateErr != nil {
				return "", updateErr
			}
			continue
		}

		succeeded++
		if _, updateErr := tx.Exec(`
			UPDATE receipts
			SET status = $2, external_id = $3, fiscal_number = $4, qr_url = $5, error_message = '', sent_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, receipt.ID, ReceiptStatusSent, sendResult.ExternalID, sendResult.FiscalNumber, sendResult.QRURL); updateErr != nil {
			return "", updateErr
		}
	}
	if failed > 0 {
		return "", fmt.Errorf("receipt retries failed: attempted=%d succeeded=%d failed=%d", attempted, succeeded, failed)
	}

	return fmt.Sprintf(
		"receipts attempted=%d succeeded=%d failed=%d status=%s limit=%d",
		attempted,
		succeeded,
		failed,
		cfg.Status,
		cfg.Limit,
	), nil
}

func (s *Store) runReceiptRetryBackgroundJobLocked(payload string) (string, error) {
	cfg, err := parseReceiptRetryJobPayload(payload)
	if err != nil {
		return "", err
	}
	if s.receiptSender == nil || !s.receiptSender.Enabled() {
		return "", ErrReceiptSenderNotConfigured
	}

	indices := []int{}
	for i := len(s.receipts) - 1; i >= 0 && len(indices) < cfg.Limit; i-- {
		if shouldRetryReceiptStatus(s.receipts[i].Status, cfg.Status) {
			indices = append(indices, i)
		}
	}

	attempted := 0
	succeeded := 0
	failed := 0
	for _, index := range indices {
		receipt := s.receipts[index]
		attempted++

		sale, found := s.saleByIDLocked(receipt.SaleID)
		if !found {
			failed++
			s.receipts[index].Status = ReceiptStatusFailed
			s.receipts[index].ErrorMessage = "sale not found"
			s.receipts[index].SentAt = nil
			s.receipts[index].UpdatedAt = time.Now().UTC()
			continue
		}

		sendResult, err := s.receiptSender.SendSaleReceipt(sale)
		if err != nil {
			failed++
			s.receipts[index].Status = ReceiptStatusFailed
			s.receipts[index].ErrorMessage = err.Error()
			s.receipts[index].SentAt = nil
			s.receipts[index].UpdatedAt = time.Now().UTC()
			continue
		}

		succeeded++
		sentAt := time.Now().UTC()
		s.receipts[index].Status = ReceiptStatusSent
		s.receipts[index].ExternalID = sendResult.ExternalID
		s.receipts[index].FiscalNumber = sendResult.FiscalNumber
		s.receipts[index].QRURL = sendResult.QRURL
		s.receipts[index].ErrorMessage = ""
		s.receipts[index].SentAt = &sentAt
		s.receipts[index].UpdatedAt = sentAt
	}
	if failed > 0 {
		return "", fmt.Errorf("receipt retries failed: attempted=%d succeeded=%d failed=%d", attempted, succeeded, failed)
	}

	return fmt.Sprintf(
		"receipts attempted=%d succeeded=%d failed=%d status=%s limit=%d",
		attempted,
		succeeded,
		failed,
		cfg.Status,
		cfg.Limit,
	), nil
}

func (s *Store) saleByIDInTx(tx *sql.Tx, saleID int64) (Sale, error) {
	var sale Sale
	if err := tx.QueryRow(`
		SELECT id, order_id, total, currency, total_uah, status, created_at
		FROM sales
		WHERE id = $1
	`, saleID).Scan(
		&sale.ID,
		&sale.OrderID,
		&sale.Total,
		&sale.Currency,
		&sale.TotalUAH,
		&sale.Status,
		&sale.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Sale{}, errors.New("sale not found")
		}
		return Sale{}, err
	}
	items, err := s.saleItemsBySaleIDInTx(tx, sale.ID)
	if err != nil {
		return Sale{}, err
	}
	sale.Items = items
	return sale, nil
}

func (s *Store) saleItemsBySaleIDInTx(tx *sql.Tx, saleID int64) ([]SaleItem, error) {
	rows, err := tx.Query(`
		SELECT product_id, quantity, price
		FROM sale_items
		WHERE sale_id = $1
		ORDER BY id ASC
	`, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []SaleItem{}
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) saleByIDLocked(saleID int64) (Sale, bool) {
	for _, sale := range s.sales {
		if sale.ID == saleID {
			return sale, true
		}
	}
	return Sale{}, false
}

func (s *Store) orderItemsByOrderID(orderID int64) ([]SaleItem, error) {
	rows, err := s.db.Query(`
		SELECT product_id, quantity, price
		FROM customer_order_items
		WHERE order_id = $1
		ORDER BY id ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []SaleItem{}
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) saleItemsBySaleID(saleID int64) ([]SaleItem, error) {
	rows, err := s.db.Query(`
		SELECT product_id, quantity, price
		FROM sale_items
		WHERE sale_id = $1
		ORDER BY id ASC
	`, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []SaleItem{}
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) supplierOrderItemsByOrderID(orderID int64) ([]PurchaseItem, error) {
	rows, err := s.db.Query(`
		SELECT product_id, quantity, price
		FROM supplier_order_items
		WHERE supplier_order_id = $1
		ORDER BY id ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []PurchaseItem{}
	for rows.Next() {
		var item PurchaseItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) purchaseItemsByPurchaseID(purchaseID int64) ([]PurchaseItem, error) {
	rows, err := s.db.Query(`
		SELECT product_id, quantity, price
		FROM purchase_items
		WHERE purchase_id = $1
		ORDER BY id ASC
	`, purchaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []PurchaseItem{}
	for rows.Next() {
		var item PurchaseItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) getSupplierOrderByID(orderID int64) (SupplierOrder, error) {
	if s.db != nil {
		var item SupplierOrder
		if err := s.db.QueryRow(`
			SELECT id, supplier_id, customer_order_id, status, currency, total, total_uah, created_at, updated_at
			FROM supplier_orders
			WHERE id = $1
		`, orderID).Scan(
			&item.ID,
			&item.SupplierID,
			&item.CustomerOrderID,
			&item.Status,
			&item.Currency,
			&item.Total,
			&item.TotalUAH,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return SupplierOrder{}, errors.New("supplier order not found")
			}
			return SupplierOrder{}, err
		}
		orderItems, err := s.supplierOrderItemsByOrderID(item.ID)
		if err != nil {
			return SupplierOrder{}, err
		}
		item.Items = orderItems
		return item, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, order := range s.supplierOrders {
		if order.ID == orderID {
			return order, nil
		}
	}
	return SupplierOrder{}, errors.New("supplier order not found")
}

func (s *Store) pendingItemsForSupplierOrder(order SupplierOrder) ([]SupplierOrderPendingItem, error) {
	purchases, err := s.ListPurchases(&order.ID)
	if err != nil {
		return nil, err
	}

	receivedByProduct := map[int64]int{}
	for _, purchase := range purchases {
		for _, item := range purchase.Items {
			receivedByProduct[item.ProductID] += item.Quantity
		}
	}

	pending := []SupplierOrderPendingItem{}
	for _, item := range order.Items {
		received := receivedByProduct[item.ProductID]
		if received < 0 {
			received = 0
		}
		pendingQty := item.Quantity - received
		if pendingQty <= 0 {
			continue
		}
		pending = append(pending, SupplierOrderPendingItem{
			ProductID: item.ProductID,
			Ordered:   item.Quantity,
			Received:  received,
			Pending:   pendingQty,
			Price:     item.Price,
		})
	}
	return pending, nil
}

func (s *Store) pendingItemsForSupplierOrderLocked(orderID int64) []SupplierOrderPendingItem {
	orderIndex := -1
	for i := range s.supplierOrders {
		if s.supplierOrders[i].ID == orderID {
			orderIndex = i
			break
		}
	}
	if orderIndex == -1 {
		return []SupplierOrderPendingItem{}
	}

	receivedByProduct := map[int64]int{}
	for _, purchase := range s.purchases {
		if purchase.SupplierOrderID == nil || *purchase.SupplierOrderID != orderID {
			continue
		}
		for _, item := range purchase.Items {
			receivedByProduct[item.ProductID] += item.Quantity
		}
	}

	pending := []SupplierOrderPendingItem{}
	for _, item := range s.supplierOrders[orderIndex].Items {
		received := receivedByProduct[item.ProductID]
		pendingQty := item.Quantity - received
		if pendingQty <= 0 {
			continue
		}
		pending = append(pending, SupplierOrderPendingItem{
			ProductID: item.ProductID,
			Ordered:   item.Quantity,
			Received:  received,
			Pending:   pendingQty,
			Price:     item.Price,
		})
	}
	return pending
}

func (s *Store) validateSupplierOrderReceiptTx(tx *sql.Tx, orderID int64, items []PurchaseItem) error {
	orderedByProduct := map[int64]int{}
	rows, err := tx.Query(`
		SELECT product_id, quantity
		FROM supplier_order_items
		WHERE supplier_order_id = $1
	`, orderID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var productID int64
		var quantity int
		if err := rows.Scan(&productID, &quantity); err != nil {
			rows.Close()
			return err
		}
		orderedByProduct[productID] += quantity
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	receivedByProduct := map[int64]int{}
	rows, err = tx.Query(`
		SELECT pi.product_id, COALESCE(SUM(pi.quantity), 0)
		FROM purchases p
		JOIN purchase_items pi ON pi.purchase_id = p.id
		WHERE p.supplier_order_id = $1
		GROUP BY pi.product_id
	`, orderID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var productID int64
		var quantity int
		if err := rows.Scan(&productID, &quantity); err != nil {
			rows.Close()
			return err
		}
		receivedByProduct[productID] = quantity
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, item := range items {
		orderedQty, ok := orderedByProduct[item.ProductID]
		if !ok {
			return errors.New("purchase contains product outside supplier order")
		}
		nextReceived := receivedByProduct[item.ProductID] + item.Quantity
		if nextReceived > orderedQty {
			return fmt.Errorf("received quantity exceeds ordered for product %d", item.ProductID)
		}
		receivedByProduct[item.ProductID] = nextReceived
	}
	return nil
}

func (s *Store) validateSupplierOrderReceiptLocked(orderID int64, items []PurchaseItem) error {
	orderedByProduct := map[int64]int{}
	for _, order := range s.supplierOrders {
		if order.ID != orderID {
			continue
		}
		for _, item := range order.Items {
			orderedByProduct[item.ProductID] += item.Quantity
		}
		break
	}

	receivedByProduct := map[int64]int{}
	for _, purchase := range s.purchases {
		if purchase.SupplierOrderID == nil || *purchase.SupplierOrderID != orderID {
			continue
		}
		for _, item := range purchase.Items {
			receivedByProduct[item.ProductID] += item.Quantity
		}
	}

	for _, item := range items {
		orderedQty, ok := orderedByProduct[item.ProductID]
		if !ok {
			return errors.New("purchase contains product outside supplier order")
		}
		nextReceived := receivedByProduct[item.ProductID] + item.Quantity
		if nextReceived > orderedQty {
			return fmt.Errorf("received quantity exceeds ordered for product %d", item.ProductID)
		}
		receivedByProduct[item.ProductID] = nextReceived
	}
	return nil
}

func (s *Store) refreshSupplierOrderReceiptStatusTx(tx *sql.Tx, orderID int64) error {
	rows, err := tx.Query(`
		SELECT product_id, quantity
		FROM supplier_order_items
		WHERE supplier_order_id = $1
	`, orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	expected := map[int64]int{}
	for rows.Next() {
		var productID int64
		var quantity int
		if err := rows.Scan(&productID, &quantity); err != nil {
			return err
		}
		expected[productID] += quantity
	}
	if err := rows.Err(); err != nil {
		return err
	}

	receivedRows, err := tx.Query(`
		SELECT pi.product_id, COALESCE(SUM(pi.quantity), 0)
		FROM purchases p
		JOIN purchase_items pi ON pi.purchase_id = p.id
		WHERE p.supplier_order_id = $1
		GROUP BY pi.product_id
	`, orderID)
	if err != nil {
		return err
	}
	defer receivedRows.Close()

	received := map[int64]int{}
	for receivedRows.Next() {
		var productID int64
		var quantity int
		if err := receivedRows.Scan(&productID, &quantity); err != nil {
			return err
		}
		received[productID] = quantity
	}
	if err := receivedRows.Err(); err != nil {
		return err
	}

	allReceived := len(expected) > 0
	anyReceived := false
	for productID, expectedQty := range expected {
		receivedQty := received[productID]
		if receivedQty > 0 {
			anyReceived = true
		}
		if receivedQty < expectedQty {
			allReceived = false
		}
	}

	nextStatus := SupplierOrderStatusSent
	switch {
	case allReceived:
		nextStatus = SupplierOrderStatusReceived
	case anyReceived:
		nextStatus = SupplierOrderStatusPartiallyReceived
	}

	_, err = tx.Exec(`
		UPDATE supplier_orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, orderID, nextStatus)
	return err
}

func (s *Store) refreshSupplierOrderReceiptStatusLocked(orderID int64) {
	orderIndex := -1
	for i := range s.supplierOrders {
		if s.supplierOrders[i].ID == orderID {
			orderIndex = i
			break
		}
	}
	if orderIndex == -1 {
		return
	}

	expected := map[int64]int{}
	for _, item := range s.supplierOrders[orderIndex].Items {
		expected[item.ProductID] += item.Quantity
	}

	received := map[int64]int{}
	for _, purchase := range s.purchases {
		if purchase.SupplierOrderID == nil || *purchase.SupplierOrderID != orderID {
			continue
		}
		for _, item := range purchase.Items {
			received[item.ProductID] += item.Quantity
		}
	}

	allReceived := len(expected) > 0
	anyReceived := false
	for productID, expectedQty := range expected {
		receivedQty := received[productID]
		if receivedQty > 0 {
			anyReceived = true
		}
		if receivedQty < expectedQty {
			allReceived = false
		}
	}

	nextStatus := SupplierOrderStatusSent
	switch {
	case allReceived:
		nextStatus = SupplierOrderStatusReceived
	case anyReceived:
		nextStatus = SupplierOrderStatusPartiallyReceived
	}

	s.supplierOrders[orderIndex].Status = nextStatus
	s.supplierOrders[orderIndex].UpdatedAt = time.Now().UTC()
}

func scanReservations(rows *sql.Rows) ([]Reservation, error) {
	reservations := []Reservation{}
	for rows.Next() {
		var reservation Reservation
		if err := rows.Scan(
			&reservation.ID,
			&reservation.OrderID,
			&reservation.ProductID,
			&reservation.Quantity,
			&reservation.Status,
			&reservation.ExpiresAt,
			&reservation.ReleasedAt,
			&reservation.CreatedAt,
		); err != nil {
			return nil, err
		}
		reservations = append(reservations, reservation)
	}
	return reservations, rows.Err()
}

func scanStockMovements(rows *sql.Rows) ([]StockMovement, error) {
	movements := []StockMovement{}
	for rows.Next() {
		var movement StockMovement
		if err := rows.Scan(
			&movement.ID,
			&movement.ProductID,
			&movement.FromWarehouseID,
			&movement.ToWarehouseID,
			&movement.FromCellID,
			&movement.ToCellID,
			&movement.Type,
			&movement.Quantity,
			&movement.Note,
			&movement.CreatedAt,
		); err != nil {
			return nil, err
		}
		movements = append(movements, movement)
	}
	return movements, rows.Err()
}

func movementTouchesWarehouse(movement StockMovement, warehouseID int64) bool {
	return (movement.FromWarehouseID != nil && *movement.FromWarehouseID == warehouseID) ||
		(movement.ToWarehouseID != nil && *movement.ToWarehouseID == warehouseID)
}

func isValidOrderStatus(status string) bool {
	switch status {
	case OrderStatusNew,
		OrderStatusInWork,
		OrderStatusOrdered,
		OrderStatusExpected,
		OrderStatusArrived,
		OrderStatusIssued,
		OrderStatusClosed,
		OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func isValidSupplierOrderStatus(status string) bool {
	switch status {
	case SupplierOrderStatusDraft,
		SupplierOrderStatusSent,
		SupplierOrderStatusConfirmed,
		SupplierOrderStatusInTransit,
		SupplierOrderStatusReceived,
		SupplierOrderStatusClosed,
		SupplierOrderStatusCancelled:
		return true
	default:
		return false
	}
}

func validateManualSupplierOrderTransition(currentStatus, nextStatus string) error {
	if currentStatus == nextStatus {
		return nil
	}

	allowed := map[string]map[string]struct{}{
		SupplierOrderStatusDraft: {
			SupplierOrderStatusSent:      {},
			SupplierOrderStatusCancelled: {},
		},
		SupplierOrderStatusSent: {
			SupplierOrderStatusConfirmed:  {},
			SupplierOrderStatusInTransit:  {},
			SupplierOrderStatusReceived:   {},
			SupplierOrderStatusCancelled:  {},
		},
		SupplierOrderStatusConfirmed: {
			SupplierOrderStatusInTransit:  {},
			SupplierOrderStatusReceived:   {},
			SupplierOrderStatusCancelled:  {},
		},
		SupplierOrderStatusInTransit: {
			SupplierOrderStatusReceived:   {},
			SupplierOrderStatusCancelled:  {},
		},
		SupplierOrderStatusReceived: {
			SupplierOrderStatusClosed:     {},
		},
		SupplierOrderStatusClosed:    {},
		SupplierOrderStatusCancelled: {},
	}

	nextAllowed, ok := allowed[currentStatus]
	if !ok {
		return fmt.Errorf("unknown current supplier order status: %s", currentStatus)
	}
	if _, ok := nextAllowed[nextStatus]; ok {
		return nil
	}
	return fmt.Errorf("manual status transition %s -> %s is not allowed", currentStatus, nextStatus)
}

func isValidDocumentType(documentType string) bool {
	switch documentType {
	case DocumentTypeInvoice,
		DocumentTypeAct,
		DocumentTypeCashInOrder,
		DocumentTypeCashOutOrder,
		DocumentTypeVAT,
		DocumentTypeReturnFromCustomer,
		DocumentTypeReturnToSupplier:
		return true
	default:
		return false
	}
}

func isStockDocumentType(documentType string) bool {
	switch documentType {
	case DocumentTypeInvoice,
		DocumentTypeReturnFromCustomer,
		DocumentTypeReturnToSupplier:
		return true
	default:
		return false
	}
}

func isCashDocumentType(documentType string) bool {
	switch documentType {
	case DocumentTypeCashInOrder, DocumentTypeCashOutOrder:
		return true
	default:
		return false
	}
}

func requiredTemplatePlaceholders(documentType string) []string {
	switch documentType {
	case DocumentTypeInvoice:
		return []string{"number", "items", "total", "currency"}
	case DocumentTypeAct:
		return []string{"number", "total", "currency"}
	case DocumentTypeCashInOrder, DocumentTypeCashOutOrder:
		return []string{"number", "total", "currency"}
	case DocumentTypeReturnFromCustomer, DocumentTypeReturnToSupplier:
		return []string{"number", "items", "total", "currency"}
	case DocumentTypeVAT:
		return []string{"number", "total"}
	default:
		return []string{}
	}
}

func newDocumentNumber(documentType string, now time.Time) string {
	code := strings.ToUpper(strings.ReplaceAll(documentType, "_", "-"))
	return fmt.Sprintf("%s-%s", code, now.Format("20060102150405"))
}

func (s *Store) cashboxByIDLocked(cashboxID int64) *Cashbox {
	for i := range s.cashboxes {
		if s.cashboxes[i].ID == cashboxID {
			return &s.cashboxes[i]
		}
	}
	return nil
}

func (s *Store) productExistsLocked(productID int64) bool {
	for _, product := range s.products {
		if product.ID == productID {
			return true
		}
	}
	return false
}

func (s *Store) documentItemsByDocumentID(documentID int64) ([]DocumentItem, error) {
	rows, err := s.db.Query(`
		SELECT product_id, quantity, price
		FROM document_items
		WHERE document_id = $1
		ORDER BY id ASC
	`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []DocumentItem{}
	for rows.Next() {
		var item DocumentItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) documentItemsByDocumentIDTx(tx *sql.Tx, documentID int64) ([]DocumentItem, error) {
	rows, err := tx.Query(`
		SELECT product_id, quantity, price
		FROM document_items
		WHERE document_id = $1
		ORDER BY id ASC
	`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []DocumentItem{}
	for rows.Next() {
		var item DocumentItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) documentByID(documentID int64) (Document, error) {
	if s.db != nil {
		row := s.db.QueryRow(`
			SELECT id, doc_type, doc_number, status, source_sale_id, source_purchase_id, source_service_order_id, warehouse_id, cashbox_id, currency, total, note, posted_at, created_at, updated_at
			FROM documents
			WHERE id = $1
		`, documentID)
		var doc Document
		if err := row.Scan(
			&doc.ID,
			&doc.Type,
			&doc.Number,
			&doc.Status,
			&doc.SourceSaleID,
			&doc.SourcePurchaseID,
			&doc.SourceServiceOrderID,
			&doc.WarehouseID,
			&doc.CashboxID,
			&doc.Currency,
			&doc.Total,
			&doc.Note,
			&doc.PostedAt,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Document{}, errors.New("document not found")
			}
			return Document{}, err
		}
		items, err := s.documentItemsByDocumentID(documentID)
		if err != nil {
			return Document{}, err
		}
		doc.Items = items
		return doc, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, doc := range s.documents {
		if doc.ID == documentID {
			return doc, nil
		}
	}
	return Document{}, errors.New("document not found")
}

func (s *Store) documentByIDTx(tx *sql.Tx, documentID int64, lock bool) (Document, error) {
	query := `
		SELECT id, doc_type, doc_number, status, source_sale_id, source_purchase_id, source_service_order_id, warehouse_id, cashbox_id, currency, total, note, posted_at, created_at, updated_at
		FROM documents
		WHERE id = $1
	`
	if lock {
		query += ` FOR UPDATE`
	}
	var doc Document
	if err := tx.QueryRow(query, documentID).Scan(
		&doc.ID,
		&doc.Type,
		&doc.Number,
		&doc.Status,
		&doc.SourceSaleID,
		&doc.SourcePurchaseID,
		&doc.SourceServiceOrderID,
		&doc.WarehouseID,
		&doc.CashboxID,
		&doc.Currency,
		&doc.Total,
		&doc.Note,
		&doc.PostedAt,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Document{}, errors.New("document not found")
		}
		return Document{}, err
	}
	items, err := s.documentItemsByDocumentIDTx(tx, documentID)
	if err != nil {
		return Document{}, err
	}
	doc.Items = items
	return doc, nil
}

func (s *Store) applyDocumentEffectsTx(tx *sql.Tx, doc Document) error {
	switch doc.Type {
	case DocumentTypeInvoice:
		if doc.WarehouseID == nil {
			return errors.New("warehouseId is required for invoice")
		}
		for _, item := range doc.Items {
			if err := s.adjustWarehouseStockInTx(tx, *doc.WarehouseID, item.ProductID, "incoming", item.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(`
				UPDATE products
				SET stock = stock + $2, purchase_price = $3
				WHERE id = $1
			`, item.ProductID, item.Quantity, item.Price); err != nil {
				return err
			}
			if _, err := tx.Exec(`
				INSERT INTO stock_movements (product_id, to_warehouse_id, movement_type, quantity, note)
				VALUES ($1, $2, 'incoming', $3, $4)
			`, item.ProductID, *doc.WarehouseID, item.Quantity, "posted_document="+doc.Number); err != nil {
				return err
			}
		}
	case DocumentTypeReturnToSupplier:
		if doc.WarehouseID == nil {
			return errors.New("warehouseId is required for return_to_supplier")
		}
		for _, item := range doc.Items {
			if err := s.adjustWarehouseStockInTx(tx, *doc.WarehouseID, item.ProductID, "sale", item.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(`
				UPDATE products
				SET stock = stock - $2
				WHERE id = $1
			`, item.ProductID, item.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(`
				INSERT INTO stock_movements (product_id, from_warehouse_id, movement_type, quantity, note)
				VALUES ($1, $2, 'return_to_supplier', $3, $4)
			`, item.ProductID, *doc.WarehouseID, item.Quantity, "posted_document="+doc.Number); err != nil {
				return err
			}
		}
	case DocumentTypeReturnFromCustomer:
		if doc.WarehouseID == nil {
			return errors.New("warehouseId is required for return_from_customer")
		}
		for _, item := range doc.Items {
			if err := s.adjustWarehouseStockInTx(tx, *doc.WarehouseID, item.ProductID, "incoming", item.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(`
				UPDATE products
				SET stock = stock + $2
				WHERE id = $1
			`, item.ProductID, item.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(`
				INSERT INTO stock_movements (product_id, to_warehouse_id, movement_type, quantity, note)
				VALUES ($1, $2, 'return_from_customer', $3, $4)
			`, item.ProductID, *doc.WarehouseID, item.Quantity, "posted_document="+doc.Number); err != nil {
				return err
			}
		}
	case DocumentTypeCashInOrder, DocumentTypeCashOutOrder:
		if doc.CashboxID == nil {
			return errors.New("cashboxId is required for cash document")
		}
		var opType string
		if doc.Type == DocumentTypeCashInOrder {
			opType = CashOperationTypeIncoming
		} else {
			opType = CashOperationTypeOutgoing
		}
		if _, err := s.createCashOperationInTx(tx, CashOperation{
			CashboxID:   *doc.CashboxID,
			Type:        opType,
			Amount:      doc.Total,
			Currency:    doc.Currency,
			Method:      PaymentMethodCash,
			Description: "posted_document=" + doc.Number,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) applyDocumentEffectsLocked(doc *Document) error {
	switch doc.Type {
	case DocumentTypeInvoice:
		if doc.WarehouseID == nil {
			return errors.New("warehouseId is required for invoice")
		}
		for _, item := range doc.Items {
			s.adjustWarehouseStockLocked(*doc.WarehouseID, item.ProductID, "incoming", item.Quantity)
			for p := range s.products {
				if s.products[p].ID == item.ProductID {
					s.products[p].Stock += item.Quantity
					s.products[p].PurchasePrice = item.Price
					break
				}
			}
		}
	case DocumentTypeReturnToSupplier:
		if doc.WarehouseID == nil {
			return errors.New("warehouseId is required for return_to_supplier")
		}
		for _, item := range doc.Items {
			stock := s.warehouseStockQuantityLocked(*doc.WarehouseID, item.ProductID)
			if stock < item.Quantity {
				return ErrInsufficientStock
			}
			s.adjustWarehouseStockLocked(*doc.WarehouseID, item.ProductID, "sale", item.Quantity)
			for p := range s.products {
				if s.products[p].ID == item.ProductID {
					if s.products[p].Stock < item.Quantity {
						return ErrInsufficientStock
					}
					s.products[p].Stock -= item.Quantity
					break
				}
			}
		}
	case DocumentTypeReturnFromCustomer:
		if doc.WarehouseID == nil {
			return errors.New("warehouseId is required for return_from_customer")
		}
		for _, item := range doc.Items {
			s.adjustWarehouseStockLocked(*doc.WarehouseID, item.ProductID, "incoming", item.Quantity)
			for p := range s.products {
				if s.products[p].ID == item.ProductID {
					s.products[p].Stock += item.Quantity
					break
				}
			}
		}
	case DocumentTypeCashInOrder, DocumentTypeCashOutOrder:
		if doc.CashboxID == nil {
			return errors.New("cashboxId is required for cash document")
		}
		cashbox := s.cashboxByIDLocked(*doc.CashboxID)
		if cashbox == nil {
			return errors.New("cashbox not found")
		}
		if doc.Type == DocumentTypeCashInOrder {
			cashbox.Balance += doc.Total
		} else {
			if cashbox.Balance < doc.Total {
				return errors.New("insufficient cashbox balance")
			}
			cashbox.Balance -= doc.Total
		}
	}
	return nil
}

func (s *Store) createCashOperationInTx(tx *sql.Tx, input CashOperation) (CashOperation, error) {
	var balance float64
	var cashboxCurrency string
	if err := tx.QueryRow(`SELECT balance, currency FROM cashboxes WHERE id = $1 FOR UPDATE`, input.CashboxID).Scan(&balance, &cashboxCurrency); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CashOperation{}, errors.New("cashbox not found")
		}
		return CashOperation{}, err
	}
	input.Currency = normalizeCurrency(cashboxCurrency)
	rateToUAH, err := s.rateToUAHInTx(tx, input.Currency)
	if err != nil {
		return CashOperation{}, err
	}
	input.AmountUAH = input.Amount * rateToUAH
	nextBalance := balance
	if input.Type == CashOperationTypeIncoming {
		nextBalance += input.Amount
	} else {
		if balance < input.Amount {
			return CashOperation{}, errors.New("insufficient cashbox balance")
		}
		nextBalance -= input.Amount
	}
	if _, err := tx.Exec(`UPDATE cashboxes SET balance = $2 WHERE id = $1`, input.CashboxID, nextBalance); err != nil {
		return CashOperation{}, err
	}
	if err := tx.QueryRow(`
		INSERT INTO cash_operations (cashbox_id, operation_type, amount, currency, amount_uah, payment_method, payment_id, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`,
		input.CashboxID,
		input.Type,
		input.Amount,
		input.Currency,
		input.AmountUAH,
		input.Method,
		input.PaymentID,
		input.Description,
	).Scan(&input.ID, &input.CreatedAt); err != nil {
		return CashOperation{}, err
	}
	return input, nil
}

func (s *Store) documentTemplateByCode(code string) DocumentTemplate {
	if s.db != nil {
		var tpl DocumentTemplate
		err := s.db.QueryRow(`
			SELECT id, code, name, body, is_active, created_at, updated_at
			FROM document_templates
			WHERE code = $1
		`, code).Scan(
			&tpl.ID,
			&tpl.Code,
			&tpl.Name,
			&tpl.Body,
			&tpl.IsActive,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		)
		if err == nil {
			return tpl
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, tpl := range s.documentTemplates {
		if tpl.Code == code {
			return tpl
		}
	}
	return DocumentTemplate{
		Code: code,
		Name: code,
		Body: "{{number}}\n{{createdAt}}\n{{total}} {{currency}}\n{{items}}\n{{note}}",
	}
}

func buildDocumentPDF(doc Document, tpl DocumentTemplate) ([]byte, error) {
	body := renderDocumentTemplate(doc, tpl)

	pdf := fmt.Sprintf("%%PDF-1.1\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n4 0 obj<</Length %d>>stream\nBT /F1 12 Tf 50 740 Td (%s) Tj ET\nendstream\nendobj\n5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\nxref\n0 6\n0000000000 65535 f \ntrailer<</Root 1 0 R/Size 6>>\nstartxref\n0\n%%%%EOF\n", len(body)+32, sanitizePDFText(body))
	return []byte(pdf), nil
}

func renderDocumentTemplate(doc Document, tpl DocumentTemplate) string {
	body := tpl.Body
	if strings.TrimSpace(body) == "" {
		body = "{{number}}"
	}
	body = strings.ReplaceAll(body, "{{number}}", doc.Number)
	body = strings.ReplaceAll(body, "{{createdAt}}", doc.CreatedAt.Format(time.RFC3339))
	body = strings.ReplaceAll(body, "{{updatedAt}}", doc.UpdatedAt.Format(time.RFC3339))
	if doc.PostedAt != nil {
		body = strings.ReplaceAll(body, "{{postedAt}}", doc.PostedAt.Format(time.RFC3339))
	} else {
		body = strings.ReplaceAll(body, "{{postedAt}}", "")
	}
	body = strings.ReplaceAll(body, "{{type}}", doc.Type)
	body = strings.ReplaceAll(body, "{{status}}", doc.Status)
	body = strings.ReplaceAll(body, "{{currency}}", doc.Currency)
	body = strings.ReplaceAll(body, "{{total}}", fmt.Sprintf("%.2f", doc.Total))
	body = strings.ReplaceAll(body, "{{note}}", doc.Note)
	if doc.WarehouseID != nil {
		body = strings.ReplaceAll(body, "{{warehouseId}}", fmt.Sprintf("%d", *doc.WarehouseID))
	} else {
		body = strings.ReplaceAll(body, "{{warehouseId}}", "")
	}
	if doc.CashboxID != nil {
		body = strings.ReplaceAll(body, "{{cashboxId}}", fmt.Sprintf("%d", *doc.CashboxID))
	} else {
		body = strings.ReplaceAll(body, "{{cashboxId}}", "")
	}

	itemLines := []string{}
	var vatTotal float64
	var subtotal float64
	for _, item := range doc.Items {
		lineTotal := float64(item.Quantity) * item.Price
		subtotal += lineTotal
		vatAmt := lineTotal * item.VATPercent / 100
		vatTotal += vatAmt
		itemLines = append(itemLines, fmt.Sprintf("P#%d x%d @ %.2f = %.2f (ПДВ %.0f%%: %.2f)",
			item.ProductID, item.Quantity, item.Price, lineTotal, item.VATPercent, vatAmt))
	}
	body = strings.ReplaceAll(body, "{{items}}", strings.Join(itemLines, "; "))
	body = strings.ReplaceAll(body, "{{subtotal}}", fmt.Sprintf("%.2f", subtotal))
	body = strings.ReplaceAll(body, "{{vatTotal}}", fmt.Sprintf("%.2f", vatTotal))
	body = strings.ReplaceAll(body, "{{totalWithVat}}", fmt.Sprintf("%.2f", subtotal+vatTotal))
	if strings.TrimSpace(body) == "" {
		return doc.Number
	}
	return body
}

func sampleDocumentForTemplate(code string) Document {
	now := time.Now().UTC()
	warehouseID := int64(1)
	cashboxID := int64(1)
	return Document{
		ID:          0,
		Type:        code,
		Number:      "PREVIEW-001",
		Status:      DocumentStatusDraft,
		WarehouseID: &warehouseID,
		CashboxID:   &cashboxID,
		Currency:    "UAH",
		Total:       1234.56,
		Items: []DocumentItem{
			{ProductID: 1001, Quantity: 2, Price: 200},
			{ProductID: 1002, Quantity: 1, Price: 834.56},
		},
		Note:      "Preview document",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func mapKeysSorted(values map[string]struct{}) []string {
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func recommendedPurchaseQty(available, minStock, soldLast30Days int) int {
	deficit := minStock - available
	if deficit < 0 {
		deficit = 0
	}
	buffer := soldLast30Days / 4
	if soldLast30Days > 0 && buffer == 0 {
		buffer = 1
	}
	qty := deficit + buffer
	if qty == 0 && available <= minStock && soldLast30Days > 0 {
		qty = 1
	}
	return qty
}

func (s *Store) productPurchasePrice(productID int64) (float64, error) {
	if s.db != nil {
		var price float64
		if err := s.db.QueryRow(`SELECT purchase_price FROM products WHERE id = $1`, productID).Scan(&price); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrProductNotFound
			}
			return 0, err
		}
		return price, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, product := range s.products {
		if product.ID == productID {
			return product.PurchasePrice, nil
		}
	}
	return 0, ErrProductNotFound
}

func (s *Store) soldQuantityLast30DaysLocked(productID int64) int {
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	total := 0
	for _, sale := range s.sales {
		if sale.CreatedAt.Before(cutoff) {
			continue
		}
		for _, item := range sale.Items {
			if item.ProductID == productID {
				total += item.Quantity
			}
		}
	}
	return total
}

func (s *Store) supplierIDByNameLocked(name string) (int64, bool) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return 0, false
	}
	for _, supplier := range s.suppliers {
		if strings.TrimSpace(strings.ToLower(supplier.Name)) == name {
			return supplier.ID, true
		}
	}
	return 0, false
}

func sanitizePDFText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}

func (s *Store) activeReservedForProductLocked(productID int64) int {
	reserved := 0
	for _, reservation := range s.reservations {
		if reservation.ProductID == productID && reservation.Status == ReservationStatusActive {
			reserved += reservation.Quantity
		}
	}
	return reserved
}

func (s *Store) activeReservedByOtherOrdersLocked(productID int64, orderID *int64) int {
	reserved := 0
	for _, reservation := range s.reservations {
		if reservation.ProductID != productID || reservation.Status != ReservationStatusActive {
			continue
		}
		if orderID != nil && reservation.OrderID == *orderID {
			continue
		}
		reserved += reservation.Quantity
	}
	return reserved
}

func (s *Store) availableStockInTx(tx *sql.Tx, productID int64) (int, error) {
	var currentStock int
	err := tx.QueryRow(`SELECT stock FROM products WHERE id = $1 FOR UPDATE`, productID).Scan(&currentStock)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrProductNotFound
	}
	if err != nil {
		return 0, err
	}

	var reserved int
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0)
		FROM reservations
		WHERE product_id = $1 AND status = $2
	`,
		productID,
		ReservationStatusActive,
	).Scan(&reserved); err != nil {
		return 0, err
	}
	return currentStock - reserved, nil
}

func (s *Store) activeReservedByOtherOrdersInTx(tx *sql.Tx, productID int64, orderID *int64) (int, error) {
	if orderID == nil {
		var reserved int
		if err := tx.QueryRow(`
			SELECT COALESCE(SUM(quantity), 0)
			FROM reservations
			WHERE product_id = $1 AND status = $2
		`,
			productID,
			ReservationStatusActive,
		).Scan(&reserved); err != nil {
			return 0, err
		}
		return reserved, nil
	}

	var reserved int
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0)
		FROM reservations
		WHERE product_id = $1 AND status = $2 AND order_id <> $3
	`,
		productID,
		ReservationStatusActive,
		*orderID,
	).Scan(&reserved); err != nil {
		return 0, err
	}
	return reserved, nil
}

func scanPayments(rows *sql.Rows) ([]Payment, error) {
	payments := []Payment{}
	for rows.Next() {
		var payment Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.SaleID,
			&payment.ServiceOrderID,
			&payment.SupplierOrderID,
			&payment.PurchaseID,
			&payment.CashboxID,
			&payment.Amount,
			&payment.Currency,
			&payment.AmountUAH,
			&payment.Method,
			&payment.Note,
			&payment.CreatedAt,
		); err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}
	return payments, rows.Err()
}

func scanReceipts(rows *sql.Rows) ([]Receipt, error) {
	receipts := []Receipt{}
	for rows.Next() {
		receipt, err := scanReceipt(rows)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	return receipts, rows.Err()
}

func scanReceipt(scanner interface{ Scan(dest ...any) error }) (Receipt, error) {
	var receipt Receipt
	var externalID sql.NullString
	var fiscalNumber sql.NullString
	var qrURL sql.NullString
	var errorMessage sql.NullString
	var payload sql.NullString
	var sentAt sql.NullTime
	if err := scanner.Scan(
		&receipt.ID,
		&receipt.SaleID,
		&receipt.Provider,
		&receipt.Status,
		&externalID,
		&fiscalNumber,
		&qrURL,
		&errorMessage,
		&payload,
		&sentAt,
		&receipt.CreatedAt,
		&receipt.UpdatedAt,
	); err != nil {
		return Receipt{}, err
	}
	if externalID.Valid {
		receipt.ExternalID = externalID.String
	}
	if fiscalNumber.Valid {
		receipt.FiscalNumber = fiscalNumber.String
	}
	if qrURL.Valid {
		receipt.QRURL = qrURL.String
	}
	if errorMessage.Valid {
		receipt.ErrorMessage = errorMessage.String
	}
	if payload.Valid {
		receipt.Payload = payload.String
	}
	if sentAt.Valid {
		sent := sentAt.Time
		receipt.SentAt = &sent
	}
	return receipt, nil
}

func scanCashOperations(rows *sql.Rows) ([]CashOperation, error) {
	operations := []CashOperation{}
	for rows.Next() {
		var operation CashOperation
		if err := rows.Scan(
			&operation.ID,
			&operation.CashboxID,
			&operation.Type,
			&operation.Amount,
			&operation.Currency,
			&operation.AmountUAH,
			&operation.Method,
			&operation.PaymentID,
			&operation.Description,
			&operation.CreatedAt,
		); err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, rows.Err()
}

func scanNotificationTemplates(rows *sql.Rows) ([]NotificationTemplate, error) {
	templates := []NotificationTemplate{}
	for rows.Next() {
		var tpl NotificationTemplate
		if err := rows.Scan(
			&tpl.ID,
			&tpl.Code,
			&tpl.Channel,
			&tpl.Subject,
			&tpl.Body,
			&tpl.IsActive,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		templates = append(templates, tpl)
	}
	return templates, rows.Err()
}

func scanNotifications(rows *sql.Rows) ([]Notification, error) {
	notifications := []Notification{}
	for rows.Next() {
		var item Notification
		if err := rows.Scan(
			&item.ID,
			&item.Channel,
			&item.Recipient,
			&item.Subject,
			&item.Body,
			&item.EntityType,
			&item.EntityID,
			&item.Status,
			&item.Attempts,
			&item.ErrorMessage,
			&item.SentAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, item)
	}
	return notifications, rows.Err()
}

func scanBackgroundJobs(rows *sql.Rows) ([]BackgroundJob, error) {
	jobs := []BackgroundJob{}
	for rows.Next() {
		var job BackgroundJob
		if err := rows.Scan(
			&job.ID,
			&job.JobType,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.NextRetryAt,
			&job.Payload,
			&job.Result,
			&job.ErrorMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.CreatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) upsertDefaultWarehouseStockForProduct(productID int64, quantity int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	warehouseID, err := s.defaultWarehouseIDInTx(tx)
	if err != nil {
		return err
	}
	if err := s.setWarehouseStockQuantityInTx(tx, warehouseID, productID, quantity); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) defaultWarehouseIDInTx(tx *sql.Tx) (int64, error) {
	var warehouseID int64
	err := tx.QueryRow(`SELECT id FROM warehouses WHERE name = $1`, defaultWarehouseName).Scan(&warehouseID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRow(`
			INSERT INTO warehouses (name, is_virtual, location_type)
			VALUES ($1, FALSE, 'warehouse')
			RETURNING id
		`, defaultWarehouseName).Scan(&warehouseID); err != nil {
			return 0, err
		}
		return warehouseID, nil
	}
	return warehouseID, err
}

func (s *Store) ensureWarehouseExistsTx(tx *sql.Tx, warehouseID int64) error {
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM warehouses WHERE id = $1)`, warehouseID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("warehouse not found")
	}
	return nil
}

func (s *Store) warehouseExistsLocked(warehouseID int64) bool {
	for _, warehouse := range s.warehouses {
		if warehouse.ID == warehouseID {
			return true
		}
	}
	return false
}

func (s *Store) warehouseStockQuantityInTx(tx *sql.Tx, warehouseID, productID int64, lock bool) (int, error) {
	query := `SELECT quantity FROM warehouse_stocks WHERE warehouse_id = $1 AND product_id = $2`
	if lock {
		query += ` FOR UPDATE`
	}
	var quantity int
	err := tx.QueryRow(query, warehouseID, productID).Scan(&quantity)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return quantity, err
}

func (s *Store) warehouseStockQuantityLocked(warehouseID, productID int64) int {
	for _, stock := range s.warehouseStocks {
		if stock.WarehouseID == warehouseID && stock.ProductID == productID {
			return stock.Quantity
		}
	}
	return 0
}

func (s *Store) setWarehouseStockQuantityInTx(tx *sql.Tx, warehouseID, productID int64, quantity int) error {
	if err := s.setWarehouseStockOnlyInTx(tx, warehouseID, productID, quantity); err != nil {
		return err
	}
	cellID, err := s.defaultCellIDInTx(tx, warehouseID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO cell_stocks (cell_id, product_id, quantity, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (cell_id, product_id)
		DO UPDATE SET quantity = $3, updated_at = NOW()
	`, cellID, productID, quantity)
	return err
}

func (s *Store) setWarehouseStockOnlyInTx(tx *sql.Tx, warehouseID, productID int64, quantity int) error {
	_, err := tx.Exec(`
		INSERT INTO warehouse_stocks (warehouse_id, product_id, quantity, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (warehouse_id, product_id)
		DO UPDATE SET quantity = $3, updated_at = NOW()
	`, warehouseID, productID, quantity)
	return err
}

func (s *Store) setWarehouseStockLocked(warehouseID, productID int64, quantity int) {
	s.setWarehouseStockOnlyLocked(warehouseID, productID, quantity)
	cellID := s.defaultCellIDLocked(warehouseID)
	if cellID == 0 {
		return
	}
	for i := range s.cellStocks {
		if s.cellStocks[i].CellID == cellID && s.cellStocks[i].ProductID == productID {
			s.cellStocks[i].Quantity = quantity
			s.cellStocks[i].UpdatedAt = time.Now().UTC()
			return
		}
	}
	s.cellStocks = append(s.cellStocks, CellStock{
		CellID:    cellID,
		ProductID: productID,
		Quantity:  quantity,
		UpdatedAt: time.Now().UTC(),
	})
}

func (s *Store) setWarehouseStockOnlyLocked(warehouseID, productID int64, quantity int) {
	for i := range s.warehouseStocks {
		if s.warehouseStocks[i].WarehouseID == warehouseID && s.warehouseStocks[i].ProductID == productID {
			s.warehouseStocks[i].Quantity = quantity
			s.warehouseStocks[i].UpdatedAt = time.Now().UTC()
			return
		}
	}
	s.warehouseStocks = append(s.warehouseStocks, WarehouseStock{
		WarehouseID: warehouseID,
		ProductID:   productID,
		Quantity:    quantity,
		UpdatedAt:   time.Now().UTC(),
	})
}

func (s *Store) defaultCellIDInTx(tx *sql.Tx, warehouseID int64) (int64, error) {
	var cellID int64
	err := tx.QueryRow(`
		SELECT c.id
		FROM warehouse_cells c
		JOIN warehouse_zones z ON z.id = c.zone_id
		WHERE c.warehouse_id = $1 AND z.name = 'DEFAULT' AND c.code = 'MAIN'
		LIMIT 1
	`, warehouseID).Scan(&cellID)
	if errors.Is(err, sql.ErrNoRows) {
		var zoneID int64
		if err := tx.QueryRow(`
			INSERT INTO warehouse_zones (warehouse_id, name)
			VALUES ($1, 'DEFAULT')
			ON CONFLICT (warehouse_id, name)
			DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, warehouseID).Scan(&zoneID); err != nil {
			return 0, err
		}
		if err := tx.QueryRow(`
			INSERT INTO warehouse_cells (warehouse_id, zone_id, code)
			VALUES ($1, $2, 'MAIN')
			ON CONFLICT (zone_id, code)
			DO UPDATE SET code = EXCLUDED.code
			RETURNING id
		`, warehouseID, zoneID).Scan(&cellID); err != nil {
			return 0, err
		}
		return cellID, nil
	}
	return cellID, err
}

func (s *Store) defaultCellIDLocked(warehouseID int64) int64 {
	zoneID := int64(0)
	for _, zone := range s.warehouseZones {
		if zone.WarehouseID == warehouseID && zone.Name == "DEFAULT" {
			zoneID = zone.ID
			break
		}
	}
	if zoneID == 0 {
		s.zoneSeq++
		zoneID = s.zoneSeq
		s.warehouseZones = append(s.warehouseZones, WarehouseZone{
			ID:          zoneID,
			WarehouseID: warehouseID,
			Name:        "DEFAULT",
			CreatedAt:   time.Now().UTC(),
		})
	}

	for _, cell := range s.warehouseCells {
		if cell.ZoneID == zoneID && cell.Code == "MAIN" {
			return cell.ID
		}
	}
	s.cellSeq++
	cellID := s.cellSeq
	s.warehouseCells = append(s.warehouseCells, WarehouseCell{
		ID:          cellID,
		WarehouseID: warehouseID,
		ZoneID:      zoneID,
		Code:        "MAIN",
		CreatedAt:   time.Now().UTC(),
	})
	return cellID
}

func (s *Store) adjustWarehouseStockInTx(tx *sql.Tx, warehouseID, productID int64, movementType string, quantity int) error {
	currentQty, err := s.warehouseStockQuantityInTx(tx, warehouseID, productID, true)
	if err != nil {
		return err
	}
	switch movementType {
	case "incoming":
		currentQty += quantity
	case "sale", "write_off", "return_to_supplier":
			if currentQty < quantity {
				return ErrInsufficientStock
			}
			currentQty -= quantity
		default:
			return fmt.Errorf("unknown movement type: %s", movementType)
	}
	return s.setWarehouseStockQuantityInTx(tx, warehouseID, productID, currentQty)
}

func (s *Store) adjustWarehouseStockLocked(warehouseID, productID int64, movementType string, quantity int) {
	current := s.warehouseStockQuantityLocked(warehouseID, productID)
	switch movementType {
	case "incoming":
		current += quantity
	case "sale", "write_off", "return_to_supplier":
		current -= quantity
	default:
		return
	}
	if current < 0 {
		current = 0
	}
	s.setWarehouseStockLocked(warehouseID, productID, current)
}

func (s *Store) transferWarehouseStockInTx(tx *sql.Tx, fromWarehouseID, toWarehouseID, productID int64, quantity int) error {
	sourceQty, err := s.warehouseStockQuantityInTx(tx, fromWarehouseID, productID, true)
	if err != nil {
		return err
	}
	if sourceQty < quantity {
		return ErrInsufficientStock
	}
	destQty, err := s.warehouseStockQuantityInTx(tx, toWarehouseID, productID, true)
	if err != nil {
		return err
	}
	if err := s.setWarehouseStockOnlyInTx(tx, fromWarehouseID, productID, sourceQty-quantity); err != nil {
		return err
	}
	return s.setWarehouseStockOnlyInTx(tx, toWarehouseID, productID, destQty+quantity)
}

func (s *Store) transferWarehouseStockLocked(fromWarehouseID, toWarehouseID, productID int64, quantity int) error {
	source := s.warehouseStockQuantityLocked(fromWarehouseID, productID)
	if source < quantity {
		return ErrInsufficientStock
	}
	dest := s.warehouseStockQuantityLocked(toWarehouseID, productID)
	s.setWarehouseStockOnlyLocked(fromWarehouseID, productID, source-quantity)
	s.setWarehouseStockOnlyLocked(toWarehouseID, productID, dest+quantity)
	return nil
}

func (s *Store) ensureCellInWarehouseTx(tx *sql.Tx, cellID, warehouseID int64) error {
	var exists bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM warehouse_cells
			WHERE id = $1 AND warehouse_id = $2
		)
	`, cellID, warehouseID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("cell not found in warehouse")
	}
	return nil
}

func (s *Store) cellInWarehouseLocked(cellID, warehouseID int64) bool {
	for _, cell := range s.warehouseCells {
		if cell.ID == cellID && cell.WarehouseID == warehouseID {
			return true
		}
	}
	return false
}

func (s *Store) warehouseIDByCellID(cellID int64) (int64, error) {
	if s.db != nil {
		var warehouseID int64
		if err := s.db.QueryRow(`SELECT warehouse_id FROM warehouse_cells WHERE id = $1`, cellID).Scan(&warehouseID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, errors.New("cell not found")
			}
			return 0, err
		}
		return warehouseID, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, cell := range s.warehouseCells {
		if cell.ID == cellID {
			return cell.WarehouseID, nil
		}
	}
	return 0, errors.New("cell not found")
}

type cellAllocation struct {
	cellID   int64
	quantity int
}

func (s *Store) allocateCellsFIFO(
	warehouseID int64,
	toCellID int64,
	productID int64,
	required int,
) ([]cellAllocation, error) {
	if s.db != nil {
		return s.allocateCellsFIFOFromDB(warehouseID, toCellID, productID, required)
	}
	allocations := s.allocateCellsFIFOLocked(warehouseID, toCellID, productID, required)
	total := 0
	for _, allocation := range allocations {
		total += allocation.quantity
	}
	if total < required {
		return nil, ErrInsufficientStock
	}
	return allocations, nil
}

func (s *Store) allocateCellsFIFOFromDB(
	warehouseID int64,
	toCellID int64,
	productID int64,
	required int,
) ([]cellAllocation, error) {
	rows, err := s.db.Query(`
		SELECT cs.cell_id, cs.quantity
		FROM cell_stocks cs
		JOIN warehouse_cells c ON c.id = cs.cell_id
		WHERE c.warehouse_id = $1
			AND cs.product_id = $2
			AND cs.cell_id <> $3
			AND cs.quantity > 0
		ORDER BY cs.updated_at ASC, cs.cell_id ASC
	`, warehouseID, productID, toCellID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allocations := []cellAllocation{}
	remaining := required
	for rows.Next() && remaining > 0 {
		var cellID int64
		var quantity int
		if err := rows.Scan(&cellID, &quantity); err != nil {
			return nil, err
		}
		take := quantity
		if take > remaining {
			take = remaining
		}
		allocations = append(allocations, cellAllocation{
			cellID:   cellID,
			quantity: take,
		})
		remaining -= take
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if remaining > 0 {
		return nil, ErrInsufficientStock
	}
	return allocations, nil
}

func (s *Store) allocateCellsFIFOLocked(
	warehouseID int64,
	toCellID int64,
	productID int64,
	required int,
) []cellAllocation {
	type candidate struct {
		cellID    int64
		quantity  int
		updatedAt time.Time
	}
	candidates := []candidate{}
	for _, stock := range s.cellStocks {
		if stock.ProductID != productID || stock.CellID == toCellID || stock.Quantity <= 0 {
			continue
		}
		cellWarehouseID, err := s.warehouseIDByCellID(stock.CellID)
		if err != nil || cellWarehouseID != warehouseID {
			continue
		}
		candidates = append(candidates, candidate{
			cellID:    stock.CellID,
			quantity:  stock.Quantity,
			updatedAt: stock.UpdatedAt,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].updatedAt.Equal(candidates[j].updatedAt) {
			return candidates[i].cellID < candidates[j].cellID
		}
		return candidates[i].updatedAt.Before(candidates[j].updatedAt)
	})

	allocations := []cellAllocation{}
	remaining := required
	for _, candidate := range candidates {
		if remaining <= 0 {
			break
		}
		take := candidate.quantity
		if take > remaining {
			take = remaining
		}
		allocations = append(allocations, cellAllocation{
			cellID:   candidate.cellID,
			quantity: take,
		})
		remaining -= take
	}
	if remaining > 0 {
		return []cellAllocation{}
	}
	return allocations
}

func (s *Store) cellStockQuantityInTx(tx *sql.Tx, cellID, productID int64, lock bool) (int, error) {
	query := `SELECT quantity FROM cell_stocks WHERE cell_id = $1 AND product_id = $2`
	if lock {
		query += ` FOR UPDATE`
	}
	var quantity int
	err := tx.QueryRow(query, cellID, productID).Scan(&quantity)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return quantity, err
}

func (s *Store) setCellStockQuantityInTx(tx *sql.Tx, cellID, productID int64, quantity int) error {
	_, err := tx.Exec(`
		INSERT INTO cell_stocks (cell_id, product_id, quantity, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (cell_id, product_id)
		DO UPDATE SET quantity = $3, updated_at = NOW()
	`, cellID, productID, quantity)
	return err
}

func (s *Store) transferCellStockInTx(tx *sql.Tx, fromCellID, toCellID, productID int64, quantity int) error {
	sourceQty, err := s.cellStockQuantityInTx(tx, fromCellID, productID, true)
	if err != nil {
		return err
	}
	if sourceQty < quantity {
		return ErrInsufficientStock
	}
	destQty, err := s.cellStockQuantityInTx(tx, toCellID, productID, true)
	if err != nil {
		return err
	}
	if err := s.setCellStockQuantityInTx(tx, fromCellID, productID, sourceQty-quantity); err != nil {
		return err
	}
	return s.setCellStockQuantityInTx(tx, toCellID, productID, destQty+quantity)
}

func (s *Store) cellStockQuantityLocked(cellID, productID int64) int {
	for _, stock := range s.cellStocks {
		if stock.CellID == cellID && stock.ProductID == productID {
			return stock.Quantity
		}
	}
	return 0
}

func (s *Store) setCellStockQuantityLocked(cellID, productID int64, quantity int) {
	for i := range s.cellStocks {
		if s.cellStocks[i].CellID == cellID && s.cellStocks[i].ProductID == productID {
			s.cellStocks[i].Quantity = quantity
			s.cellStocks[i].UpdatedAt = time.Now().UTC()
			return
		}
	}
	s.cellStocks = append(s.cellStocks, CellStock{
		CellID:    cellID,
		ProductID: productID,
		Quantity:  quantity,
		UpdatedAt: time.Now().UTC(),
	})
}

func (s *Store) transferCellStockLocked(fromCellID, toCellID, productID int64, quantity int) error {
	source := s.cellStockQuantityLocked(fromCellID, productID)
	if source < quantity {
		return ErrInsufficientStock
	}
	dest := s.cellStockQuantityLocked(toCellID, productID)
	s.setCellStockQuantityLocked(fromCellID, productID, source-quantity)
	s.setCellStockQuantityLocked(toCellID, productID, dest+quantity)
	return nil
}

func (s *Store) inventoryItemsByInventoryID(inventoryID int64) ([]InventoryItem, error) {
	rows, err := s.db.Query(`
		SELECT product_id, system_quantity, actual_quantity, adjustment
		FROM inventory_items
		WHERE inventory_id = $1
		ORDER BY id ASC
	`, inventoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInventoryItems(rows)
}

func (s *Store) inventoryItemsByInventoryIDTx(tx *sql.Tx, inventoryID int64) ([]InventoryItem, error) {
	rows, err := tx.Query(`
		SELECT product_id, system_quantity, actual_quantity, adjustment
		FROM inventory_items
		WHERE inventory_id = $1
		ORDER BY id ASC
	`, inventoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInventoryItems(rows)
}

func scanInventoryItems(rows *sql.Rows) ([]InventoryItem, error) {
	items := []InventoryItem{}
	for rows.Next() {
		var item InventoryItem
		if err := rows.Scan(&item.ProductID, &item.SystemQuantity, &item.ActualQuantity, &item.Adjustment); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (s *Store) getBackgroundJobByID(jobID int64) (BackgroundJob, error) {
	if s.db != nil {
		job := BackgroundJob{}
		if err := s.db.QueryRow(`
			SELECT id, job_type, status, attempts, max_attempts, next_retry_at, payload, result, error_message, started_at, finished_at, created_at
			FROM background_jobs
			WHERE id = $1
		`, jobID).Scan(
			&job.ID,
			&job.JobType,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.NextRetryAt,
			&job.Payload,
			&job.Result,
			&job.ErrorMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.CreatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return BackgroundJob{}, errors.New("job not found")
			}
			return BackgroundJob{}, err
		}
		return job, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, job := range s.backgroundJobs {
		if job.ID == jobID {
			return job, nil
		}
	}
	return BackgroundJob{}, errors.New("job not found")
}

func (s *Store) loadActiveReminderTemplatesTx(tx *sql.Tx) ([]NotificationTemplate, error) {
	rows, err := tx.Query(`
		SELECT id, code, channel, subject, body, is_active, created_at, updated_at
		FROM notification_templates
		WHERE code = $1 AND is_active = TRUE
		ORDER BY id ASC
	`, BackgroundJobTypeOverdueReminders)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotificationTemplates(rows)
}

func (s *Store) listOverdueDebtsTx(tx *sql.Tx, asOf time.Time) ([]DebtSummary, error) {
	rows, err := tx.Query(`
		SELECT
			o.id,
			o.currency,
			o.total,
			o.total_uah,
			o.due_date,
			COALESCE(er.rate_to_uah, 1),
			COALESCE((
				SELECT SUM(p.amount_uah)
				FROM payments p
				WHERE p.order_id = o.id
			), 0) AS paid_uah,
			(
				SELECT MAX(p.created_at)
				FROM payments p
				WHERE p.order_id = o.id
			) AS last_payment_at
		FROM customer_orders o
		LEFT JOIN exchange_rates er ON er.currency = o.currency
		WHERE o.status <> $1
			AND o.due_date IS NOT NULL
			AND o.due_date < $2
		GROUP BY o.id, o.currency, o.total, o.total_uah, o.due_date, er.rate_to_uah
		HAVING o.total_uah >
			COALESCE((SELECT SUM(p2.amount_uah) FROM payments p2 WHERE p2.order_id = o.id), 0)
		ORDER BY o.id DESC
	`,
		OrderStatusCancelled,
		asOf,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overdue := []DebtSummary{}
	for rows.Next() {
		var item DebtSummary
		var rateToUAH float64
		item.EntityType = "order"
		if err := rows.Scan(
			&item.EntityID,
			&item.Currency,
			&item.Total,
			&item.TotalUAH,
			&item.DueDate,
			&rateToUAH,
			&item.PaidUAH,
			&item.LastPaymentAt,
		); err != nil {
			return nil, err
		}
		if rateToUAH <= 0 {
			rateToUAH = 1
		}
		item.Paid = item.PaidUAH / rateToUAH
		item.Debt = item.Total - item.Paid
		item.DebtUAH = item.TotalUAH - item.PaidUAH
		item.IsOverdue = true
		item.OverdueDays = debtOverdueDays(item.DueDate, item.DebtUAH, asOf)
		overdue = append(overdue, item)
	}
	return overdue, rows.Err()
}

func (s *Store) sendOverdueNotificationsTx(
	tx *sql.Tx,
	templates []NotificationTemplate,
	overdue []DebtSummary,
) (int, error) {
	sentCount := 0
	var firstErr error
	for _, tpl := range templates {
		for _, debt := range overdue {
			body := renderTemplateBody(tpl.Body, debt)
			recipient := notificationRecipientForDebt(tpl.Channel, debt, s.notificationSender)
			status := NotificationStatusSentStub
			attempts := 0
			errMessage := ""
			if s.notificationSender != nil && s.notificationSender.Enabled() {
				attempts = 1
				if err := s.notificationSender.Send(tpl.Channel, recipient, tpl.Subject, body); err != nil {
					status = NotificationStatusFailed
					errMessage = err.Error()
					if firstErr == nil {
						firstErr = err
					}
				} else {
					status = NotificationStatusSent
				}
			}
			if _, err := tx.Exec(`
				INSERT INTO notifications
					(channel, recipient, subject, body, entity_type, entity_id, status, attempts, error_message, sent_at)
				VALUES
					($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			`,
				tpl.Channel,
				recipient,
				tpl.Subject,
				body,
				debt.EntityType,
				debt.EntityID,
				status,
				attempts,
				errMessage,
			); err != nil {
				return sentCount, err
			}
			sentCount++
		}
	}
	if firstErr != nil {
		return sentCount, firstErr
	}
	return sentCount, nil
}

func renderTemplateBody(template string, debt DebtSummary) string {
	body := template
	replacements := map[string]string{
		"{{entityType}}":  debt.EntityType,
		"{{entityId}}":    fmt.Sprintf("%d", debt.EntityID),
		"{{currency}}":    debt.Currency,
		"{{debt}}":        fmt.Sprintf("%.2f", debt.Debt),
		"{{debtUah}}":     fmt.Sprintf("%.2f", debt.DebtUAH),
		"{{overdueDays}}": fmt.Sprintf("%d", debt.OverdueDays),
	}
	for key, value := range replacements {
		body = strings.ReplaceAll(body, key, value)
	}
	return body
}

func notificationRecipientForDebt(channel string, debt DebtSummary, sender NotificationSender) string {
	if sender != nil {
		if recipient := strings.TrimSpace(sender.DefaultRecipient(channel)); recipient != "" {
			return recipient
		}
	}
	switch channel {
	case NotificationChannelEmail:
		return fmt.Sprintf("customer+%s-%d@example.local", debt.EntityType, debt.EntityID)
	case NotificationChannelTelegram:
		return fmt.Sprintf("telegram://debt-reminder/%s/%d", debt.EntityType, debt.EntityID)
	default:
		return ""
	}
}

func (s *Store) mustListDebtsLocked() []DebtSummary {
	debts := []DebtSummary{}
	now := time.Now().UTC()
	for _, order := range s.orders {
		if order.Status == OrderStatusCancelled {
			continue
		}
		paidUAH, lastPayment := s.paidForTargetUAHLocked(&order.ID, nil, nil)
		if order.TotalUAH-paidUAH <= 0 {
			continue
		}
		rateToUAH := s.exchangeRates[normalizeCurrency(order.Currency)]
		if rateToUAH == 0 {
			rateToUAH = 1
		}
		paid := paidUAH / rateToUAH
		debts = append(debts, DebtSummary{
			EntityType:    "order",
			EntityID:      order.ID,
			Currency:      order.Currency,
			Total:         order.Total,
			Paid:          paid,
			Debt:          order.Total - paid,
			TotalUAH:      order.TotalUAH,
			PaidUAH:       paidUAH,
			DebtUAH:       order.TotalUAH - paidUAH,
			DueDate:       order.DueDate,
			IsOverdue:     isDebtOverdue(order.DueDate, order.TotalUAH-paidUAH, now),
			OverdueDays:   debtOverdueDays(order.DueDate, order.TotalUAH-paidUAH, now),
			LastPaymentAt: lastPayment,
		})
	}
	return debts
}

func isValidPaymentMethod(method string) bool {
	switch method {
	case PaymentMethodCash, PaymentMethodCard, PaymentMethodBank, PaymentMethodVirtual:
		return true
	default:
		return false
	}
}

func (s *Store) resolveCashboxInTx(tx *sql.Tx, explicitID int64, method string) (int64, string, error) {
	if explicitID > 0 {
		var currency string
		if err := tx.QueryRow(`SELECT currency FROM cashboxes WHERE id = $1`, explicitID).Scan(&currency); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, "", errors.New("cashbox not found")
			}
			return 0, "", err
		}
		return explicitID, normalizeCurrency(currency), nil
	}

	var id int64
	var currency string
	err := tx.QueryRow(`
		SELECT id, currency
		FROM cashboxes
		WHERE type = $1
		ORDER BY id ASC
		LIMIT 1
	`, method).Scan(&id, &currency)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", errors.New("cashbox for payment method not found")
	}
	return id, normalizeCurrency(currency), err
}

func (s *Store) resolveCashboxIndexLocked(explicitID int64, method string) (int, error) {
	if explicitID > 0 {
		for i := range s.cashboxes {
			if s.cashboxes[i].ID == explicitID {
				return i, nil
			}
		}
		return -1, errors.New("cashbox not found")
	}

	for i := range s.cashboxes {
		if s.cashboxes[i].Type == method {
			return i, nil
		}
	}
	return -1, errors.New("cashbox for payment method not found")
}

func normalizeCurrency(currency string) string {
	value := strings.ToUpper(strings.TrimSpace(currency))
	if value == "" {
		return "UAH"
	}
	return value
}

func isDebtOverdue(dueDate *time.Time, debtUAH float64, asOf time.Time) bool {
	if dueDate == nil || debtUAH <= 0 {
		return false
	}
	return dueDate.Before(asOf)
}

func debtOverdueDays(dueDate *time.Time, debtUAH float64, asOf time.Time) int {
	if !isDebtOverdue(dueDate, debtUAH, asOf) {
		return 0
	}
	duration := asOf.Sub(*dueDate)
	days := int(duration.Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}

func backgroundJobBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	seconds := 30 * (1 << (attempt - 1))
	if seconds > 1800 {
		seconds = 1800
	}
	return time.Duration(seconds) * time.Second
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value
}

func (s *Store) ListExchangeRates() ([]ExchangeRate, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT currency, rate_to_uah, updated_at
			FROM exchange_rates
			ORDER BY currency ASC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		rates := []ExchangeRate{}
		for rows.Next() {
			var rate ExchangeRate
			if err := rows.Scan(&rate.Currency, &rate.RateToUAH, &rate.UpdatedAt); err != nil {
				return nil, err
			}
			rates = append(rates, rate)
		}
		return rates, rows.Err()
	}

	rates := []ExchangeRate{}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for currency, rateToUAH := range s.exchangeRates {
		rates = append(rates, ExchangeRate{
			Currency:  currency,
			RateToUAH: rateToUAH,
		})
	}
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Currency < rates[j].Currency
	})
	return rates, nil
}

func (s *Store) UpsertExchangeRate(currency string, rateToUAH float64) (ExchangeRate, error) {
	currency = normalizeCurrency(currency)
	if rateToUAH <= 0 {
		return ExchangeRate{}, errors.New("rateToUah must be greater than zero")
	}

	if s.db != nil {
		entry := ExchangeRate{}
		if err := s.db.QueryRow(`
			INSERT INTO exchange_rates (currency, rate_to_uah, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (currency) DO UPDATE
			SET rate_to_uah = EXCLUDED.rate_to_uah, updated_at = NOW()
			RETURNING currency, rate_to_uah, updated_at
		`,
			currency,
			rateToUAH,
		).Scan(&entry.Currency, &entry.RateToUAH, &entry.UpdatedAt); err != nil {
			return ExchangeRate{}, err
		}
		return entry, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.exchangeRates[currency] = rateToUAH
	return ExchangeRate{
		Currency:  currency,
		RateToUAH: rateToUAH,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *Store) rateToUAHInTx(tx *sql.Tx, currency string) (float64, error) {
	currency = normalizeCurrency(currency)
	var rate float64
	if err := tx.QueryRow(`SELECT rate_to_uah FROM exchange_rates WHERE currency = $1`, currency).Scan(&rate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("exchange rate not found for currency: %s", currency)
		}
		return 0, err
	}
	return rate, nil
}

func (s *Store) rateToUAH(currency string) (float64, error) {
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		rate, err := s.rateToUAHInTx(tx, currency)
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return rate, nil
	}
	return s.rateToUAHLocked(currency)
}

func (s *Store) rateToUAHInDB(currency string) (float64, error) {
	if s.db == nil {
		return s.rateToUAHLocked(currency)
	}
	currency = normalizeCurrency(currency)
	var rate float64
	if err := s.db.QueryRow(`SELECT rate_to_uah FROM exchange_rates WHERE currency = $1`, currency).Scan(&rate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("exchange rate not found for currency: %s", currency)
		}
		return 0, err
	}
	return rate, nil
}

func (s *Store) rateToUAHLocked(currency string) (float64, error) {
	currency = normalizeCurrency(currency)
	rateToUAH, ok := s.exchangeRates[currency]
	if !ok {
		return 0, fmt.Errorf("exchange rate not found for currency: %s", currency)
	}
	return rateToUAH, nil
}

func (s *Store) targetPaymentTotalsInTx(tx *sql.Tx, orderID, saleID, serviceOrderID *int64) (float64, float64, error) {
	if orderID != nil {
		var totalUAH float64
		if err := tx.QueryRow(`
			SELECT total_uah
			FROM customer_orders
			WHERE id = $1
		`, *orderID).Scan(&totalUAH); err != nil {
			return 0, 0, err
		}

		var paidUAH float64
		if err := tx.QueryRow(`
			SELECT COALESCE(SUM(amount_uah), 0)
			FROM payments
			WHERE order_id = $1
		`, *orderID).Scan(&paidUAH); err != nil {
			return 0, 0, err
		}
		return totalUAH, paidUAH, nil
	}

	if saleID != nil {
		var totalUAH float64
		if err := tx.QueryRow(`SELECT total_uah FROM sales WHERE id = $1`, *saleID).Scan(&totalUAH); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, 0, errors.New("sale not found")
			}
			return 0, 0, err
		}

		var paidUAH float64
		if err := tx.QueryRow(`
			SELECT COALESCE(SUM(amount_uah), 0)
			FROM payments
			WHERE sale_id = $1
		`, *saleID).Scan(&paidUAH); err != nil {
			return 0, 0, err
		}
		return totalUAH, paidUAH, nil
	}

	var serviceOrderTotal float64
	if serviceOrderID == nil {
		return 0, 0, errors.New("no payment target specified")
	}
	var serviceOrderCurrency string
	if err := tx.QueryRow(`
		SELECT (price + parts_total), currency
		FROM service_orders
		WHERE id = $1
	`, *serviceOrderID).Scan(&serviceOrderTotal, &serviceOrderCurrency); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, errors.New("service order not found")
		}
		return 0, 0, err
	}
	rateToUAH, err := s.rateToUAHInTx(tx, serviceOrderCurrency)
	if err != nil {
		return 0, 0, err
	}
	totalUAH := serviceOrderTotal * rateToUAH

	var paidUAH float64
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(amount_uah), 0)
		FROM payments
		WHERE service_order_id = $1
	`, *serviceOrderID).Scan(&paidUAH); err != nil {
		return 0, 0, err
	}
	return totalUAH, paidUAH, nil
}

func (s *Store) refreshOrderPaymentStatusInTx(tx *sql.Tx, orderID int64) error {
	totalUAH, paidUAH, err := s.targetPaymentTotalsInTx(tx, &orderID, nil, nil)
	if err != nil {
		return err
	}

	nextStatus := ""
	if paidUAH >= totalUAH {
		nextStatus = OrderStatusArrived // fully paid → Надійшло
	} else if paidUAH > 0 {
		nextStatus = OrderStatusInWork // partial payment → В роботі
	}
	if nextStatus == "" {
		return nil
	}

	_, err = tx.Exec(`
		UPDATE customer_orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1 AND status <> $3
	`,
		orderID,
		nextStatus,
		OrderStatusCancelled,
	)
	return err
}

func (s *Store) targetPaymentTotalsLocked(orderID, saleID, serviceOrderID *int64) (float64, float64, error) {
	if orderID != nil {
		totalUAH := 0.0
		for _, order := range s.orders {
			if order.ID == *orderID {
				totalUAH = order.TotalUAH
				break
			}
		}
		if totalUAH == 0 {
			return 0, 0, errors.New("order not found")
		}
		paidUAH, _ := s.paidForTargetUAHLocked(orderID, nil, nil)
		return totalUAH, paidUAH, nil
	}

	if saleID != nil {
		for _, sale := range s.sales {
			if sale.ID == *saleID {
				paidUAH, _ := s.paidForTargetUAHLocked(nil, saleID, nil)
				return sale.TotalUAH, paidUAH, nil
			}
		}
		return 0, 0, errors.New("sale not found")
	}

	if serviceOrderID == nil {
		return 0, 0, errors.New("no payment target specified")
	}
	for _, serviceOrder := range s.serviceOrders {
		if serviceOrder.ID != *serviceOrderID {
			continue
		}
		total := serviceOrder.Price + serviceOrder.PartsTotal
		rateToUAH, err := s.rateToUAHLocked(serviceOrder.Currency)
		if err != nil {
			return 0, 0, err
		}
		paidUAH, _ := s.paidForTargetUAHLocked(nil, nil, serviceOrderID)
		return total * rateToUAH, paidUAH, nil
	}
	return 0, 0, errors.New("service order not found")
}

func (s *Store) orderTotalLocked(orderID int64) float64 {
	for _, order := range s.orders {
		if order.ID != orderID {
			continue
		}
		total := 0.0
		for _, item := range order.Items {
			total += float64(item.Quantity) * item.Price
		}
		return total
	}
	return 0
}

func (s *Store) paidForTargetUAHLocked(orderID, saleID, serviceOrderID *int64, extraIDs ...interface{}) (float64, *time.Time) {
	paidUAH := 0.0
	var lastPayment *time.Time

	// optional: supplierOrderID passed as extraIDs[0]
	var supplierOrderID *int64
	if len(extraIDs) > 0 {
		if v, ok := extraIDs[0].(*int64); ok {
			supplierOrderID = v
		}
	}

	for _, payment := range s.payments {
		if orderID != nil {
			if payment.OrderID == nil || *payment.OrderID != *orderID {
				continue
			}
		}
		if saleID != nil {
			if payment.SaleID == nil || *payment.SaleID != *saleID {
				continue
			}
		}
		if serviceOrderID != nil {
			if payment.ServiceOrderID == nil || *payment.ServiceOrderID != *serviceOrderID {
				continue
			}
		}
		if supplierOrderID != nil {
			if payment.SupplierOrderID == nil || *payment.SupplierOrderID != *supplierOrderID {
				continue
			}
		}
		paidUAH += payment.AmountUAH
		if lastPayment == nil || payment.CreatedAt.After(*lastPayment) {
			copyTime := payment.CreatedAt
			lastPayment = &copyTime
		}
	}

	return paidUAH, lastPayment
}

func (s *Store) refreshOrderPaymentStatusLocked(orderID int64) {
	totalUAH := 0.0
	for _, order := range s.orders {
		if order.ID == orderID {
			totalUAH = order.TotalUAH
			break
		}
	}
	if totalUAH <= 0 {
		return
	}
	paidUAH, _ := s.paidForTargetUAHLocked(&orderID, nil, nil)

	nextStatus := ""
	if paidUAH >= totalUAH {
		nextStatus = OrderStatusArrived
	} else if paidUAH > 0 {
		nextStatus = OrderStatusInWork
	}
	if nextStatus == "" {
		return
	}

	for i := range s.orders {
		if s.orders[i].ID != orderID || s.orders[i].Status == OrderStatusCancelled {
			continue
		}
		s.orders[i].Status = nextStatus
		s.orders[i].UpdatedAt = time.Now().UTC()
		return
	}
}

func (s *Store) listOrderDebtsFromDB() ([]DebtSummary, error) {
	rows, err := s.db.Query(`
		SELECT
			o.id,
			o.currency,
			o.total,
			o.total_uah,
			o.due_date,
			COALESCE(er.rate_to_uah, 1),
			COALESCE((
				SELECT SUM(p.amount_uah)
				FROM payments p
				WHERE p.order_id = o.id
			), 0) AS paid_uah,
			(
				SELECT MAX(p.created_at)
				FROM payments p
				WHERE p.order_id = o.id
			) AS last_payment_at
		FROM customer_orders o
		LEFT JOIN exchange_rates er ON er.currency = o.currency
		WHERE o.status <> $1
		GROUP BY o.id, o.currency, o.total, o.total_uah, o.due_date, er.rate_to_uah
		HAVING o.total_uah >
			COALESCE((SELECT SUM(p2.amount_uah) FROM payments p2 WHERE p2.order_id = o.id), 0)
		ORDER BY o.id DESC
	`, OrderStatusCancelled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	debts := []DebtSummary{}
	for rows.Next() {
		var item DebtSummary
		var rateToUAH float64
		item.EntityType = "order"
		if err := rows.Scan(
			&item.EntityID,
			&item.Currency,
			&item.Total,
			&item.TotalUAH,
			&item.DueDate,
			&rateToUAH,
			&item.PaidUAH,
			&item.LastPaymentAt,
		); err != nil {
			return nil, err
		}
		if rateToUAH <= 0 {
			rateToUAH = 1
		}
		item.Paid = item.PaidUAH / rateToUAH
		item.Debt = item.Total - item.Paid
		item.DebtUAH = item.TotalUAH - item.PaidUAH
		now := time.Now().UTC()
		item.IsOverdue = isDebtOverdue(item.DueDate, item.DebtUAH, now)
		item.OverdueDays = debtOverdueDays(item.DueDate, item.DebtUAH, now)
		debts = append(debts, item)
	}
	return debts, rows.Err()
}

func (s *Store) listSaleDebtsFromDB() ([]DebtSummary, error) {
	rows, err := s.db.Query(`
		SELECT
			s.id,
			s.currency,
			s.total,
			s.total_uah,
			COALESCE(er.rate_to_uah, 1),
			COALESCE((
				SELECT SUM(p.amount_uah)
				FROM payments p
				WHERE p.sale_id = s.id
			), 0) AS paid_uah,
			(
				SELECT MAX(p.created_at)
				FROM payments p
				WHERE p.sale_id = s.id
			) AS last_payment_at
		FROM sales s
		LEFT JOIN exchange_rates er ON er.currency = s.currency
		WHERE s.total_uah >
			COALESCE((SELECT SUM(p2.amount_uah) FROM payments p2 WHERE p2.sale_id = s.id), 0)
		ORDER BY s.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	debts := []DebtSummary{}
	for rows.Next() {
		var item DebtSummary
		var rateToUAH float64
		item.EntityType = "sale"
		if err := rows.Scan(
			&item.EntityID,
			&item.Currency,
			&item.Total,
			&item.TotalUAH,
			&rateToUAH,
			&item.PaidUAH,
			&item.LastPaymentAt,
		); err != nil {
			return nil, err
		}
		if rateToUAH <= 0 {
			rateToUAH = 1
		}
		item.Paid = item.PaidUAH / rateToUAH
		item.Debt = item.Total - item.Paid
		item.DebtUAH = item.TotalUAH - item.PaidUAH
		item.IsOverdue = false
		item.OverdueDays = 0
		debts = append(debts, item)
	}
	return debts, rows.Err()
}

func (s *Store) listServiceOrderDebtsFromDB() ([]DebtSummary, error) {
	rows, err := s.db.Query(`
		SELECT
			so.id,
			so.currency,
			(so.price + so.parts_total) AS total,
			(so.price + so.parts_total) * COALESCE(er.rate_to_uah, 1) AS total_uah,
			COALESCE(er.rate_to_uah, 1) AS rate_to_uah,
			COALESCE((
				SELECT SUM(p.amount)
				FROM payments p
				WHERE p.service_order_id = so.id
			), 0) AS paid,
			COALESCE((
				SELECT SUM(p.amount_uah)
				FROM payments p
				WHERE p.service_order_id = so.id
			), 0) AS paid_uah,
			(
				SELECT MAX(p.created_at)
				FROM payments p
				WHERE p.service_order_id = so.id
			) AS last_payment_at
		FROM service_orders so
		LEFT JOIN exchange_rates er ON er.currency = so.currency
		WHERE so.status <> $1
		  AND ((so.price + so.parts_total) * COALESCE(er.rate_to_uah, 1)) >
			COALESCE((SELECT SUM(p2.amount_uah) FROM payments p2 WHERE p2.service_order_id = so.id), 0)
		ORDER BY so.id DESC
	`, ServiceOrderStatusCancelled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	debts := []DebtSummary{}
	for rows.Next() {
		var item DebtSummary
		var rateToUAH float64
		item.EntityType = "service_order"
		if err := rows.Scan(
			&item.EntityID,
			&item.Currency,
			&item.Total,
			&item.TotalUAH,
			&rateToUAH,
			&item.Paid,
			&item.PaidUAH,
			&item.LastPaymentAt,
		); err != nil {
			return nil, err
		}
		if rateToUAH <= 0 {
			rateToUAH = 1
		}
		item.Debt = item.Total - item.Paid
		if item.Debt < 0 {
			item.Debt = 0
		}
		item.DebtUAH = item.TotalUAH - item.PaidUAH
		if item.DebtUAH < 0 {
			item.DebtUAH = 0
		}
		item.IsOverdue = false
		item.OverdueDays = 0
		debts = append(debts, item)
	}
	return debts, rows.Err()
}

func (s *Store) AnalyticsSummary() (AnalyticsSummary, error) {
	if s.db != nil {
		var summary AnalyticsSummary
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&summary.ProductCount); err != nil {
			return AnalyticsSummary{}, err
		}
		if err := s.db.QueryRow(`SELECT COALESCE(SUM(stock), 0) FROM products`).Scan(&summary.TotalStock); err != nil {
			return AnalyticsSummary{}, err
		}
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM products WHERE stock <= min_stock`).Scan(&summary.LowStock); err != nil {
			return AnalyticsSummary{}, err
		}
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM sales`).Scan(&summary.SalesCount); err != nil {
			return AnalyticsSummary{}, err
		}
		if err := s.db.QueryRow(`SELECT COALESCE(SUM(total_uah), 0) FROM sales`).Scan(&summary.Revenue); err != nil {
			return AnalyticsSummary{}, err
		}
		return summary, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var lowStock int
	var totalStock int
	var revenue float64
	for _, p := range s.products {
		totalStock += p.Stock
		if p.Stock <= p.MinStock {
			lowStock++
		}
	}

	for _, sale := range s.sales {
		revenue += sale.TotalUAH
	}

	return AnalyticsSummary{
		ProductCount: len(s.products),
		LowStock:     lowStock,
		TotalStock:   totalStock,
		SalesCount:   len(s.sales),
		Revenue:      revenue,
	}, nil
}

func (s *Store) ProfitabilityReport() (ProfitabilityReport, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT
				p.id,
				p.name,
				p.sku,
				COALESCE(SUM(si.quantity), 0) AS qty,
				COALESCE(SUM(si.quantity * si.price * COALESCE(sr.rate_to_uah, 1)), 0) AS revenue_uah,
				COALESCE(SUM(si.quantity * p.purchase_price * COALESCE(pr.rate_to_uah, 1)), 0) AS cost_uah
			FROM sale_items si
			JOIN sales s ON s.id = si.sale_id
			JOIN products p ON p.id = si.product_id
			LEFT JOIN exchange_rates sr ON sr.currency = s.currency
			LEFT JOIN exchange_rates pr ON pr.currency = p.currency
			GROUP BY p.id, p.name, p.sku
			ORDER BY revenue_uah DESC
		`)
		if err != nil {
			return ProfitabilityReport{}, err
		}
		defer rows.Close()

		report := ProfitabilityReport{
			Items: make([]ProductProfitability, 0),
		}
		for rows.Next() {
			var item ProductProfitability
			if err := rows.Scan(&item.ProductID, &item.ProductName, &item.SKU, &item.QuantitySold, &item.RevenueUAH, &item.CostUAH); err != nil {
				return ProfitabilityReport{}, err
			}
			item.ProfitUAH = item.RevenueUAH - item.CostUAH
			if item.RevenueUAH > 0 {
				item.MarginPct = (item.ProfitUAH / item.RevenueUAH) * 100
			}
			report.TotalRevenue += item.RevenueUAH
			report.TotalCost += item.CostUAH
			report.TotalProfit += item.ProfitUAH
			report.Items = append(report.Items, item)
		}
		if err := rows.Err(); err != nil {
			return ProfitabilityReport{}, err
		}
		if report.TotalRevenue > 0 {
			report.MarginPct = (report.TotalProfit / report.TotalRevenue) * 100
		}
		return report, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	productByID := map[int64]Product{}
	for _, product := range s.products {
		productByID[product.ID] = product
	}
	agg := map[int64]*ProductProfitability{}
	report := ProfitabilityReport{
		Items: make([]ProductProfitability, 0),
	}

	for _, sale := range s.sales {
		saleRate, err := s.rateToUAHLocked(sale.Currency)
		if err != nil {
			saleRate = 1
		}
		for _, saleItem := range sale.Items {
			product, ok := productByID[saleItem.ProductID]
			if !ok {
				continue
			}
			productRate, err := s.rateToUAHLocked(product.Currency)
			if err != nil {
				productRate = 1
			}

			entry := agg[saleItem.ProductID]
			if entry == nil {
				entry = &ProductProfitability{
					ProductID:   product.ID,
					ProductName: product.Name,
					SKU:         product.SKU,
				}
				agg[saleItem.ProductID] = entry
			}

			revenue := float64(saleItem.Quantity) * saleItem.Price * saleRate
			cost := float64(saleItem.Quantity) * product.PurchasePrice * productRate
			entry.QuantitySold += saleItem.Quantity
			entry.RevenueUAH += revenue
			entry.CostUAH += cost
		}
	}

	for _, item := range agg {
		item.ProfitUAH = item.RevenueUAH - item.CostUAH
		if item.RevenueUAH > 0 {
			item.MarginPct = (item.ProfitUAH / item.RevenueUAH) * 100
		}
		report.TotalRevenue += item.RevenueUAH
		report.TotalCost += item.CostUAH
		report.TotalProfit += item.ProfitUAH
		report.Items = append(report.Items, *item)
	}
	sort.Slice(report.Items, func(i, j int) bool {
		return report.Items[i].RevenueUAH > report.Items[j].RevenueUAH
	})
	if report.TotalRevenue > 0 {
		report.MarginPct = (report.TotalProfit / report.TotalRevenue) * 100
	}
	return report, nil
}

func (s *Store) CategoryAnalytics() ([]CategoryAnalyticsItem, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT COALESCE(NULLIF(p.category, ''), 'Uncategorized') AS category,
				COALESCE(SUM(si.quantity), 0) AS quantity,
				COALESCE(SUM(si.quantity * si.price * CASE WHEN sa.currency = 'USD' THEN 40 ELSE 1 END), 0) AS revenue_uah
			FROM sale_items si
			JOIN sales sa ON sa.id = si.sale_id
			JOIN products p ON p.id = si.product_id
			GROUP BY category
			ORDER BY revenue_uah DESC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []CategoryAnalyticsItem{}
		for rows.Next() {
			var item CategoryAnalyticsItem
			if err := rows.Scan(&item.Category, &item.Quantity, &item.RevenueUAH); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	productsByID := map[int64]Product{}
	for _, product := range s.products {
		productsByID[product.ID] = product
	}
	grouped := map[string]*CategoryAnalyticsItem{}
	for _, sale := range s.sales {
		for _, item := range sale.Items {
			product := productsByID[item.ProductID]
			category := product.Category
			if strings.TrimSpace(category) == "" {
				category = "Uncategorized"
			}
			if grouped[category] == nil {
				grouped[category] = &CategoryAnalyticsItem{Category: category}
			}
			grouped[category].Quantity += item.Quantity
			grouped[category].RevenueUAH += float64(item.Quantity) * item.Price
		}
	}
	result := []CategoryAnalyticsItem{}
	for _, item := range grouped {
		result = append(result, *item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RevenueUAH > result[j].RevenueUAH })
	return result, nil
}

func (s *Store) ChangeHistory(limit int) ([]ChangeHistoryEntry, error) {
	logs, err := s.ListAuditLogs(limit)
	if err != nil {
		return nil, err
	}
	result := make([]ChangeHistoryEntry, 0, len(logs))
	for _, logEntry := range logs {
		result = append(result, ChangeHistoryEntry{
			ID:         logEntry.ID,
			User:       logEntry.User,
			Action:     logEntry.Action,
			Entity:     logEntry.Entity,
			OldValue:   "",
			NewValue:   logEntry.Details,
			OccurredAt: logEntry.CreatedAt,
		})
	}
	return result, nil
}

func (s *Store) AddAuditLog(user, action, entity, details string) AuditLog {
	if s.db != nil {
		var logEntry AuditLog
		err := s.db.QueryRow(`
			INSERT INTO audit_logs (username, action, entity, details)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at
		`,
			user,
			action,
			entity,
			details,
		).Scan(&logEntry.ID, &logEntry.CreatedAt)
		if err != nil {
			log.Printf("failed to write audit log: %v", err)
			return AuditLog{}
		}
		logEntry.User = user
		logEntry.Action = action
		logEntry.Entity = entity
		logEntry.Details = details
		return logEntry
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.auditSeq++
	log := AuditLog{
		ID:        s.auditSeq,
		User:      user,
		Action:    action,
		Entity:    entity,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	}
	s.auditLogs = append(s.auditLogs, log)
	return log
}

func (s *Store) ListAuditLogs(limit int) ([]AuditLog, error) {
	if s.db != nil {
		if limit <= 0 {
			limit = 50
		}
		rows, err := s.db.Query(`
			SELECT id, username, action, entity, details, created_at
			FROM audit_logs
			ORDER BY id DESC
			LIMIT $1
		`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		logs := []AuditLog{}
		for rows.Next() {
			var entry AuditLog
			if err := rows.Scan(
				&entry.ID,
				&entry.User,
				&entry.Action,
				&entry.Entity,
				&entry.Details,
				&entry.CreatedAt,
			); err != nil {
				return nil, err
			}
			logs = append(logs, entry)
		}
		return logs, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.auditLogs) {
		limit = len(s.auditLogs)
	}

	start := len(s.auditLogs) - limit
	result := make([]AuditLog, limit)
	copy(result, s.auditLogs[start:])
	return result, nil
}

func (s *Store) ListUsers() ([]User, error) {
	if s.db != nil {
		rows, err := s.db.Query(`SELECT username, role FROM users ORDER BY username`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		users := []User{}
		for rows.Next() {
			var user User
			if err := rows.Scan(&user.Username, &user.Role); err != nil {
				return nil, err
			}
			users = append(users, user)
		}
		return users, rows.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, User{
			Username: user.Username,
			Role:     user.Role,
		})
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	return users, nil
}

func (s *Store) CreateUser(input User) (User, error) {
	if s.db != nil {
		if _, err := s.db.Exec(`
			INSERT INTO users (username, password, role)
			VALUES ($1, $2, $3)
		`,
			input.Username,
			input.Password,
			input.Role,
		); err != nil {
			return User{}, err
		}
		return User{
			Username: input.Username,
			Role:     input.Role,
		}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[input.Username]; exists {
		return User{}, errors.New("user already exists")
	}
	s.users[input.Username] = input
	return User{
		Username: input.Username,
		Role:     input.Role,
	}, nil
}

func (s *Store) ListRolePermissions() ([]RolePermissions, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT r.name, COALESCE(rp.permission, '')
			FROM roles r
			LEFT JOIN role_permissions rp ON rp.role = r.name
			ORDER BY r.name, rp.permission
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		byRole := map[string][]string{}
		for rows.Next() {
			var role string
			var permission string
			if err := rows.Scan(&role, &permission); err != nil {
				return nil, err
			}
			if permission != "" {
				byRole[role] = append(byRole[role], permission)
				continue
			}
			if _, exists := byRole[role]; !exists {
				byRole[role] = []string{}
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}

		result := make([]RolePermissions, 0, len(byRole))
		for role, permissions := range byRole {
			result = append(result, RolePermissions{
				Role:        role,
				Permissions: permissions,
			})
		}
		sort.Slice(result, func(i, j int) bool {
			return result[i].Role < result[j].Role
		})
		return result, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RolePermissions, 0, len(s.rolePermissions))
	for role, permissions := range s.rolePermissions {
		permissionList := make([]string, 0, len(permissions))
		for permission := range permissions {
			permissionList = append(permissionList, permission)
		}
		result = append(result, RolePermissions{
			Role:        role,
			Permissions: permissionList,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Role < result[j].Role
	})
	return result, nil
}

func (s *Store) UpdateRolePermissions(role string, permissions []string) error {
	validPermissions := map[string]struct{}{}
	for _, permission := range AllPermissions() {
		validPermissions[permission] = struct{}{}
	}
	for _, permission := range permissions {
		if _, ok := validPermissions[permission]; !ok {
			return fmt.Errorf("unknown permission: %s", permission)
		}
	}

	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`INSERT INTO roles (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, role); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM role_permissions WHERE role = $1`, role); err != nil {
			return err
		}
		for _, permission := range permissions {
			if _, err := tx.Exec(`
				INSERT INTO role_permissions (role, permission)
				VALUES ($1, $2)
			`,
				role,
				permission,
			); err != nil {
				return err
			}
		}

		return tx.Commit()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	permissionSet := map[string]struct{}{}
	for _, permission := range permissions {
		permissionSet[permission] = struct{}{}
	}
	s.rolePermissions[role] = permissionSet
	return nil
}

// EnsureAndSendReceipt creates a receipt for the sale if one does not exist,
// then immediately attempts to send it to the fiscal provider.
func (s *Store) EnsureAndSendReceipt(sale Sale) (Receipt, error) {
	receipt, err := s.ensureSaleReceiptForSale(sale)
	if err != nil {
		return Receipt{}, err
	}
	// If already sent, return as-is
	if receipt.Status == ReceiptStatusSent {
		return receipt, nil
	}
	// Trigger send
	return s.RetryReceipt(receipt.ID)
}

// SalesGrouped returns revenue, profit and count grouped by day or month.
func (s *Store) SalesGrouped(groupBy string) ([]map[string]interface{}, error) {
	if s.db == nil {
		// In-memory fallback — group s.sales by period
		periodMap := map[string]struct{ qty int; revenue, profit float64 }{}
		for _, sale := range s.sales {
			var key string
			if groupBy == "day" {
				key = sale.CreatedAt.Format("2006-01-02")
			} else {
				key = sale.CreatedAt.Format("2006-01")
			}
			e := periodMap[key]
			e.qty++
			e.revenue += sale.Total
			periodMap[key] = e
		}
		result := make([]map[string]interface{}, 0, len(periodMap))
		for k, v := range periodMap {
			result = append(result, map[string]interface{}{
				"period": k, "salesQty": v.qty, "revenue": v.revenue, "profit": 0,
			})
		}
		return result, nil
	}

	trunc := "month"
	if groupBy == "day" {
		trunc = "day"
	}
	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT
			TO_CHAR(DATE_TRUNC('%s', s.created_at), 'YYYY-MM-DD') AS period,
			COUNT(*)::int AS sales_qty,
			COALESCE(SUM(s.total), 0) AS revenue,
			COALESCE(SUM(s.total) - SUM(
				(SELECT COALESCE(SUM(si.quantity * p.purchase_price), 0)
				 FROM sale_items si
				 JOIN products p ON p.id = si.product_id
				 WHERE si.sale_id = s.id)
			), 0) AS profit
		FROM sales s
		WHERE s.status != 'cancelled'
		GROUP BY DATE_TRUNC('%s', s.created_at)
		ORDER BY DATE_TRUNC('%s', s.created_at) DESC
		LIMIT 60
	`, trunc, trunc, trunc))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var period string
		var qty int
		var revenue, profit float64
		if err := rows.Scan(&period, &qty, &revenue, &profit); err != nil {
			return nil, err
		}
		result = append(result, map[string]interface{}{
			"period": period, "salesQty": qty, "revenue": revenue, "profit": profit,
		})
	}
	return result, rows.Err()
}

// ── Counterparties ────────────────────────────────────────────

func (s *Store) ListCounterparties() ([]Counterparty, error) {
	if s.db != nil {
		rows, err := s.db.Query(`
			SELECT id, name, phone, email, comment, is_customer, is_supplier, customer_id, supplier_id, created_at, updated_at
			FROM counterparties
			ORDER BY name ASC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := []Counterparty{}
		for rows.Next() {
			var c Counterparty
			if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Comment, &c.IsCustomer, &c.IsSupplier, &c.CustomerID, &c.SupplierID, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return nil, err
			}
			result = append(result, c)
		}
		return result, rows.Err()
	}
	return []Counterparty{}, nil
}

func (s *Store) CreateCounterparty(input Counterparty) (Counterparty, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Counterparty{}, errors.New("name is required")
	}
	if s.db != nil {
		tx, err := s.db.Begin()
		if err != nil {
			return Counterparty{}, err
		}
		defer tx.Rollback()

		if input.IsCustomer {
			var custID int64
			if err := tx.QueryRow(`
				INSERT INTO customers (name, phone, email, comment)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, input.Name, input.Phone, input.Email, input.Comment).Scan(&custID); err != nil {
				return Counterparty{}, err
			}
			input.CustomerID = &custID
		}
		if input.IsSupplier {
			var supID int64
			if err := tx.QueryRow(`
				INSERT INTO suppliers (name, phone, email, comments)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, input.Name, input.Phone, input.Email, input.Comment).Scan(&supID); err != nil {
				return Counterparty{}, err
			}
			input.SupplierID = &supID
		}

		if err := tx.QueryRow(`
			INSERT INTO counterparties (name, phone, email, comment, is_customer, is_supplier, customer_id, supplier_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, created_at, updated_at
		`, input.Name, input.Phone, input.Email, input.Comment, input.IsCustomer, input.IsSupplier, input.CustomerID, input.SupplierID,
		).Scan(&input.ID, &input.CreatedAt, &input.UpdatedAt); err != nil {
			return Counterparty{}, err
		}
		if err := tx.Commit(); err != nil {
			return Counterparty{}, err
		}
		return input, nil
	}
	return Counterparty{}, errors.New("database not available")
}

func (s *Store) UpdateCounterparty(id int64, input Counterparty) (Counterparty, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Counterparty{}, errors.New("name is required")
	}
	if s.db != nil {
		if _, err := s.db.Exec(`
			UPDATE counterparties
			SET name=$2, phone=$3, email=$4, comment=$5, is_customer=$6, is_supplier=$7, updated_at=NOW()
			WHERE id=$1
		`, id, input.Name, input.Phone, input.Email, input.Comment, input.IsCustomer, input.IsSupplier); err != nil {
			return Counterparty{}, err
		}
		var c Counterparty
		if err := s.db.QueryRow(`
			SELECT id, name, phone, email, comment, is_customer, is_supplier, customer_id, supplier_id, created_at, updated_at
			FROM counterparties WHERE id=$1
		`, id).Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Comment, &c.IsCustomer, &c.IsSupplier, &c.CustomerID, &c.SupplierID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return Counterparty{}, err
		}
		return c, nil
	}
	return Counterparty{}, errors.New("database not available")
}

// ── Order chain (full document tree) ─────────────────────────

func (s *Store) GetOrderChain(orderID int64) (*OrderChain, error) {
	if s.db == nil {
		return &OrderChain{}, nil
	}

	// 1. Root: customer order
	var order CustomerOrder
	if err := s.db.QueryRow(`
		SELECT id, customer_name, status, currency, total, total_uah, due_date, created_at, updated_at
		FROM customer_orders WHERE id=$1
	`, orderID).Scan(&order.ID, &order.CustomerName, &order.Status, &order.Currency, &order.Total, &order.TotalUAH, &order.DueDate, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	root := OrderChainNode{
		Type:     "customer_order",
		ID:       order.ID,
		Label:    fmt.Sprintf("Замовлення покупця #%d — %s", order.ID, order.CustomerName),
		Status:   order.Status,
		Amount:   order.Total,
		Currency: order.Currency,
		Date:     order.CreatedAt,
	}

	// 2. Linked supplier orders
	soRows, err := s.db.Query(`
		SELECT id, status, currency, total, created_at
		FROM supplier_orders WHERE customer_order_id=$1 ORDER BY id
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer soRows.Close()
	for soRows.Next() {
		var so struct {
			ID        int64
			Status    string
			Currency  string
			Total     float64
			CreatedAt time.Time
		}
		if err := soRows.Scan(&so.ID, &so.Status, &so.Currency, &so.Total, &so.CreatedAt); err != nil {
			return nil, err
		}
		soNode := OrderChainNode{
			Type:     "supplier_order",
			ID:       so.ID,
			Label:    fmt.Sprintf("Замовлення постачальнику #%d", so.ID),
			Status:   so.Status,
			Amount:   so.Total,
			Currency: so.Currency,
			Date:     so.CreatedAt,
		}

		// 2a. Purchases under this supplier order
		prRows, err := s.db.Query(`
			SELECT id, currency, total, created_at FROM purchases WHERE supplier_order_id=$1 ORDER BY id
		`, so.ID)
		if err != nil {
			return nil, err
		}
		for prRows.Next() {
			var pr struct {
				ID        int64
				Currency  string
				Total     float64
				CreatedAt time.Time
			}
			if err := prRows.Scan(&pr.ID, &pr.Currency, &pr.Total, &pr.CreatedAt); err != nil {
				prRows.Close()
				return nil, err
			}
			prNode := OrderChainNode{
				Type:     "purchase",
				ID:       pr.ID,
				Label:    fmt.Sprintf("Надходження #%d", pr.ID),
				Amount:   pr.Total,
				Currency: pr.Currency,
				Date:     pr.CreatedAt,
			}
			soNode.Children = append(soNode.Children, prNode)
		}
		prRows.Close()

		// 2b. Payments to supplier for this supplier order
		spRows, err := s.db.Query(`
			SELECT id, amount, currency, payment_method, note, created_at
			FROM payments WHERE supplier_order_id=$1 ORDER BY id
		`, so.ID)
		if err != nil {
			return nil, err
		}
		for spRows.Next() {
			var p struct {
				ID        int64
				Amount    float64
				Currency  string
				Method    string
				Note      string
				CreatedAt time.Time
			}
			if err := spRows.Scan(&p.ID, &p.Amount, &p.Currency, &p.Method, &p.Note, &p.CreatedAt); err != nil {
				spRows.Close()
				return nil, err
			}
			note := p.Note
			if note == "" {
				note = p.Method
			}
			soNode.Children = append(soNode.Children, OrderChainNode{
				Type:     "supplier_payment",
				ID:       p.ID,
				Label:    fmt.Sprintf("Оплата постачальнику #%d (%s)", p.ID, note),
				Amount:   p.Amount,
				Currency: p.Currency,
				Date:     p.CreatedAt,
			})
		}
		spRows.Close()

		root.Children = append(root.Children, soNode)
	}
	soRows.Close()

	// 3. Linked sales (orderId on sale)
	saleRows, err := s.db.Query(`
		SELECT id, currency, total, status, created_at FROM sales WHERE order_id=$1 ORDER BY id
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer saleRows.Close()
	for saleRows.Next() {
		var sal struct {
			ID        int64
			Currency  string
			Total     float64
			Status    string
			CreatedAt time.Time
		}
		if err := saleRows.Scan(&sal.ID, &sal.Currency, &sal.Total, &sal.Status, &sal.CreatedAt); err != nil {
			return nil, err
		}
		root.Children = append(root.Children, OrderChainNode{
			Type:     "sale",
			ID:       sal.ID,
			Label:    fmt.Sprintf("Продаж #%d", sal.ID),
			Status:   sal.Status,
			Amount:   sal.Total,
			Currency: sal.Currency,
			Date:     sal.CreatedAt,
		})
	}

	// 4. Customer payments on this order
	pmtRows, err := s.db.Query(`
		SELECT id, amount, currency, payment_method, note, created_at
		FROM payments WHERE order_id=$1 ORDER BY id
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer pmtRows.Close()
	for pmtRows.Next() {
		var p struct {
			ID        int64
			Amount    float64
			Currency  string
			Method    string
			Note      string
			CreatedAt time.Time
		}
		if err := pmtRows.Scan(&p.ID, &p.Amount, &p.Currency, &p.Method, &p.Note, &p.CreatedAt); err != nil {
			return nil, err
		}
		note := p.Note
		if note == "" {
			note = p.Method
		}
		root.Children = append(root.Children, OrderChainNode{
			Type:     "payment",
			ID:       p.ID,
			Label:    fmt.Sprintf("Оплата покупця #%d (%s)", p.ID, note),
			Amount:   p.Amount,
			Currency: p.Currency,
			Date:     p.CreatedAt,
		})
	}

	return &OrderChain{Root: root}, nil
}

// ── Document Registry Search ──────────────────────────────────────────────

// SearchDocuments returns a unified list of documents matching the query across
// all document types. query is matched against: customer/supplier name, product
// name, note, doc number/title, status. Pass empty query to list recent docs.
func (s *Store) SearchDocuments(query string, docTypes []string, limit int) ([]DocumentRegistryItem, error) {
	if s.db == nil {
		return []DocumentRegistryItem{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	q := "%" + strings.ToLower(query) + "%"
	wantType := func(t string) bool {
		if len(docTypes) == 0 {
			return true
		}
		for _, dt := range docTypes {
			if dt == t {
				return true
			}
		}
		return false
	}

	var results []DocumentRegistryItem

	// 1. Customer orders
	if wantType("customer_order") {
		rows, err := s.db.Query(`
			SELECT co.id, co.customer_name, co.status, co.total, co.currency, co.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM customer_order_items coi
			           JOIN products p ON p.id = coi.product_id
			           WHERE coi.order_id = co.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM customer_orders co
			WHERE $1 = '%%'
			   OR LOWER(co.customer_name) LIKE $1
			   OR LOWER(co.status)        LIKE $1
			   OR CAST(co.id AS TEXT)     LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM customer_order_items coi
			       JOIN products p ON p.id = coi.product_id
			       WHERE coi.order_id = co.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY co.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("customer_orders search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var hits string
			if err := rows.Scan(&item.ID, &item.CounterName, &item.Status, &item.Total, &item.Currency, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "customer_order"
			item.Number = fmt.Sprintf("Замовлення покупця #%d", item.ID)
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// 2. Supplier orders
	if wantType("supplier_order") {
		rows, err := s.db.Query(`
			SELECT so.id, sup.name, so.status, so.total, so.currency, so.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM supplier_order_items soi
			           JOIN products p ON p.id = soi.product_id
			           WHERE soi.supplier_order_id = so.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM supplier_orders so
			JOIN suppliers sup ON sup.id = so.supplier_id
			WHERE $1 = '%%'
			   OR LOWER(sup.name)      LIKE $1
			   OR LOWER(so.status)     LIKE $1
			   OR CAST(so.id AS TEXT)  LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM supplier_order_items soi
			       JOIN products p ON p.id = soi.product_id
			       WHERE soi.supplier_order_id = so.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY so.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("supplier_orders search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var hits string
			if err := rows.Scan(&item.ID, &item.CounterName, &item.Status, &item.Total, &item.Currency, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "supplier_order"
			item.Number = fmt.Sprintf("Замовлення постачальнику #%d", item.ID)
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// 3. Purchases (надходження)
	if wantType("purchase") {
		rows, err := s.db.Query(`
			SELECT pu.id, sup.name, pu.total, pu.currency, pu.note, pu.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM purchase_items pi2
			           JOIN products p ON p.id = pi2.product_id
			           WHERE pi2.purchase_id = pu.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM purchases pu
			JOIN suppliers sup ON sup.id = pu.supplier_id
			WHERE $1 = '%%'
			   OR LOWER(sup.name)      LIKE $1
			   OR LOWER(pu.note)       LIKE $1
			   OR CAST(pu.id AS TEXT)  LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM purchase_items pi2
			       JOIN products p ON p.id = pi2.product_id
			       WHERE pi2.purchase_id = pu.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY pu.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("purchases search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var hits string
			if err := rows.Scan(&item.ID, &item.CounterName, &item.Total, &item.Currency, &item.Note, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "purchase"
			item.Number = fmt.Sprintf("Надходження #%d", item.ID)
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// 4. Sales
	if wantType("sale") {
		rows, err := s.db.Query(`
			SELECT s.id, s.status, s.total, s.currency, s.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM sale_items si
			           JOIN products p ON p.id = si.product_id
			           WHERE si.sale_id = s.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM sales s
			WHERE $1 = '%%'
			   OR LOWER(s.status)     LIKE $1
			   OR CAST(s.id AS TEXT)  LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM sale_items si
			       JOIN products p ON p.id = si.product_id
			       WHERE si.sale_id = s.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY s.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("sales search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var hits string
			if err := rows.Scan(&item.ID, &item.Status, &item.Total, &item.Currency, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "sale"
			item.Number = fmt.Sprintf("Продаж #%d", item.ID)
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// 5. Service orders (ремонти)
	if wantType("service_order") {
		rows, err := s.db.Query(`
			SELECT so.id, c.name, so.title, so.status,
			       (so.price + so.parts_total), so.currency, so.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM service_order_parts sop
			           JOIN products p ON p.id = sop.product_id
			           WHERE sop.service_order_id = so.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM service_orders so
			JOIN customers c ON c.id = so.customer_id
			WHERE $1 = '%%'
			   OR LOWER(c.name)        LIKE $1
			   OR LOWER(so.title)      LIKE $1
			   OR LOWER(so.description) LIKE $1
			   OR LOWER(so.status)     LIKE $1
			   OR CAST(so.id AS TEXT)  LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM service_order_parts sop
			       JOIN products p ON p.id = sop.product_id
			       WHERE sop.service_order_id = so.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY so.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("service_orders search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var hits string
			if err := rows.Scan(&item.ID, &item.CounterName, &item.Note, &item.Status, &item.Total, &item.Currency, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "service_order"
			item.Number = fmt.Sprintf("Ремонт #%d — %s", item.ID, item.Note)
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// 6. Cash operations — incoming (прибутковий касовий ордер)
	if wantType("payment_in") {
		rows, err := s.db.Query(`
			SELECT co.id, co.amount, co.currency, co.description, co.created_at
			FROM cash_operations co
			WHERE co.operation_type = 'incoming'
			  AND ($1 = '%%'
			   OR LOWER(co.description) LIKE $1
			   OR CAST(co.id AS TEXT)   LIKE $2)
			ORDER BY co.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("payment_in search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			if err := rows.Scan(&item.ID, &item.Total, &item.Currency, &item.Note, &item.Date); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "payment_in"
			item.Number = fmt.Sprintf("ПКО #%d", item.ID)
			results = append(results, item)
		}
		rows.Close()
	}

	// 7. Cash operations — outgoing (видатковий касовий ордер)
	if wantType("payment_out") {
		rows, err := s.db.Query(`
			SELECT co.id, co.amount, co.currency, co.description, co.created_at
			FROM cash_operations co
			WHERE co.operation_type = 'outgoing'
			  AND ($1 = '%%'
			   OR LOWER(co.description) LIKE $1
			   OR CAST(co.id AS TEXT)   LIKE $2)
			ORDER BY co.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("payment_out search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			if err := rows.Scan(&item.ID, &item.Total, &item.Currency, &item.Note, &item.Date); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "payment_out"
			item.Number = fmt.Sprintf("ВКО #%d", item.ID)
			results = append(results, item)
		}
		rows.Close()
	}

	// 8. Stock transfers (переміщення)
	if wantType("transfer") {
		rows, err := s.db.Query(`
			SELECT st.id, wf.name, wt.name, st.note, st.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM stock_transfer_items sti
			           JOIN products p ON p.id = sti.product_id
			           WHERE sti.transfer_id = st.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM stock_transfers st
			JOIN warehouses wf ON wf.id = st.from_warehouse_id
			JOIN warehouses wt ON wt.id = st.to_warehouse_id
			WHERE $1 = '%%'
			   OR LOWER(wf.name)       LIKE $1
			   OR LOWER(wt.name)       LIKE $1
			   OR LOWER(st.note)       LIKE $1
			   OR CAST(st.id AS TEXT)  LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM stock_transfer_items sti
			       JOIN products p ON p.id = sti.product_id
			       WHERE sti.transfer_id = st.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY st.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("transfers search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var fromWH, toWH, hits string
			if err := rows.Scan(&item.ID, &fromWH, &toWH, &item.Note, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "transfer"
			item.Number = fmt.Sprintf("Переміщення #%d", item.ID)
			item.CounterName = fromWH + " → " + toWH
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// 9. Inventories (інвентаризація / списання)
	if wantType("inventory") {
		rows, err := s.db.Query(`
			SELECT inv.id, w.name, inv.status, inv.note, inv.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM inventory_items ii
			           JOIN products p ON p.id = ii.product_id
			           WHERE ii.inventory_id = inv.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM inventories inv
			JOIN warehouses w ON w.id = inv.warehouse_id
			WHERE $1 = '%%'
			   OR LOWER(w.name)          LIKE $1
			   OR LOWER(inv.status)      LIKE $1
			   OR LOWER(inv.note)        LIKE $1
			   OR CAST(inv.id AS TEXT)   LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM inventory_items ii
			       JOIN products p ON p.id = ii.product_id
			       WHERE ii.inventory_id = inv.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY inv.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("inventories search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var hits string
			if err := rows.Scan(&item.ID, &item.CounterName, &item.Status, &item.Note, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			item.DocType = "inventory"
			item.Number = fmt.Sprintf("Інвентаризація #%d", item.ID)
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// 10. Documents (накладні / повернення)
	if wantType("document") {
		rows, err := s.db.Query(`
			SELECT d.id, d.doc_type, d.doc_number, d.status, d.total, d.currency, d.note, d.created_at,
			       COALESCE((
			           SELECT string_agg(p.name, ', ')
			           FROM document_items di
			           JOIN products p ON p.id = di.product_id
			           WHERE di.document_id = d.id AND LOWER(p.name) LIKE $1
			       ), '') AS product_hits
			FROM documents d
			WHERE $1 = '%%'
			   OR LOWER(d.doc_number)  LIKE $1
			   OR LOWER(d.doc_type)    LIKE $1
			   OR LOWER(d.status)      LIKE $1
			   OR LOWER(d.note)        LIKE $1
			   OR CAST(d.id AS TEXT)   LIKE $2
			   OR EXISTS (
			       SELECT 1 FROM document_items di
			       JOIN products p ON p.id = di.product_id
			       WHERE di.document_id = d.id AND LOWER(p.name) LIKE $1
			   )
			ORDER BY d.id DESC
			LIMIT $3
		`, q, "%"+strings.ToLower(query)+"%", limit)
		if err != nil {
			return nil, fmt.Errorf("documents search: %w", err)
		}
		for rows.Next() {
			var item DocumentRegistryItem
			var hits string
			if err := rows.Scan(&item.ID, &item.DocType, &item.Number, &item.Status, &item.Total, &item.Currency, &item.Note, &item.Date, &hits); err != nil {
				rows.Close()
				return nil, err
			}
			// Keep docType as returned (e.g. "customer_return", "supplier_return")
			if hits != "" {
				item.ProductHits = strings.Split(hits, ", ")
			}
			results = append(results, item)
		}
		rows.Close()
	}

	// Sort all results by date descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Date.After(results[j].Date)
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// FilteredListAuditLogs returns audit logs with optional filters.
// All filter params are optional — empty string / zero means "no filter".
func (s *Store) FilteredListAuditLogs(search, user, entity, dateFrom, dateTo string, limit int) ([]AuditLog, error) {
	if s.db == nil {
		return []AuditLog{}, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	query := `
		SELECT id, username, action, entity, details, created_at
		FROM audit_logs
		WHERE ($1 = '' OR username = $1)
		  AND ($2 = '' OR entity   = $2)
		  AND ($3 = '' OR (LOWER(username) LIKE $3 OR LOWER(action) LIKE $3 OR LOWER(details) LIKE $3))
		  AND ($4 = '' OR created_at >= $4::timestamptz)
		  AND ($5 = '' OR created_at <  ($5::date + interval '1 day')::timestamptz)
		ORDER BY id DESC
		LIMIT $6
	`
	searchQ := ""
	if search != "" {
		searchQ = "%" + strings.ToLower(search) + "%"
	}

	rows, err := s.db.Query(query, user, entity, searchQ, dateFrom, dateTo, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []AuditLog{}
	for rows.Next() {
		var entry AuditLog
		if err := rows.Scan(&entry.ID, &entry.User, &entry.Action, &entry.Entity, &entry.Details, &entry.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

// GetProductLifecycle collects the full lifecycle history for a single product.
// It queries: product creation, purchases, stock movements (non-purchase/sale),
// sales, customer-order returns and supplier returns (via documents).
func (s *Store) GetProductLifecycle(productID int64) (ProductLifecycle, error) {
	if s.db == nil {
		// in-memory mode: lifecycle queries require the DB; return empty lifecycle
		return ProductLifecycle{ProductID: productID, Events: []ProductLifecycleEvent{}}, nil
	}

	var events []ProductLifecycleEvent

	// 1. Created
	var createdAt time.Time
	err := s.db.QueryRow(`SELECT created_at FROM products WHERE id = $1`, productID).Scan(&createdAt)
	if err != nil {
		return ProductLifecycle{}, fmt.Errorf("product not found: %w", err)
	}
	events = append(events, ProductLifecycleEvent{
		EventType: "created",
		EventDate: createdAt,
	})

	// 2. Purchases (коли куплений, у кого куплений)
	rows, err := s.db.Query(`
		SELECT pi.purchase_id, pi.quantity, pi.price, p.currency, p.supplier_id, s.name, p.created_at
		FROM purchase_items pi
		JOIN purchases p ON p.id = pi.purchase_id
		JOIN suppliers s ON s.id = p.supplier_id
		WHERE pi.product_id = $1
		ORDER BY p.created_at ASC`, productID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e ProductLifecycleEvent
			var qty int
			var price float64
			var suppID int64
			if err2 := rows.Scan(&e.RefID, &qty, &price, &e.Currency, &suppID, &e.SupplierName, &e.EventDate); err2 == nil {
				e.EventType = "purchased"
				e.Quantity = &qty
				e.Price = &price
				e.SupplierID = &suppID
				events = append(events, e)
			}
		}
	}

	// 3. Stock movements with warehouse names (receipt, write_off, transfer, adjustment)
	rows2, err := s.db.Query(`
		SELECT sm.id, sm.movement_type, sm.quantity, sm.note, sm.created_at,
		       wf.name, wt.name
		FROM stock_movements sm
		LEFT JOIN warehouses wf ON wf.id = sm.from_warehouse_id
		LEFT JOIN warehouses wt ON wt.id = sm.to_warehouse_id
		WHERE sm.product_id = $1
		  AND sm.movement_type NOT IN ('sale', 'write_off')
		ORDER BY sm.created_at ASC`, productID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var e ProductLifecycleEvent
			var qty int
			var note string
			var wFrom, wTo *string
			var id int64
			var mvType string
			if err2 := rows2.Scan(&id, &mvType, &qty, &note, &e.EventDate, &wFrom, &wTo); err2 == nil {
				e.RefID = &id
				e.EventType = "movement"
				e.Note = note
				e.Quantity = &qty
				if wTo != nil && *wTo != "" {
					e.WarehouseName = *wTo
				} else if wFrom != nil && *wFrom != "" {
					e.WarehouseName = *wFrom
				}
				// Надходження на склад
				if mvType == "receipt" || mvType == "purchase" {
					e.EventType = "received"
				}
				events = append(events, e)
			}
		}
	}

	// 4. Sales (кому проданий) — через sale_items + customer_orders якщо є
	rows3, err := s.db.Query(`
		SELECT si.sale_id, si.quantity, si.price, sl.currency, sl.created_at,
		       COALESCE(co.customer_name, '')
		FROM sale_items si
		JOIN sales sl ON sl.id = si.sale_id
		LEFT JOIN customer_orders co ON co.id = sl.order_id
		WHERE si.product_id = $1
		ORDER BY sl.created_at ASC`, productID)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var e ProductLifecycleEvent
			var qty int
			var price float64
			if err2 := rows3.Scan(&e.RefID, &qty, &price, &e.Currency, &e.EventDate, &e.CustomerName); err2 == nil {
				e.EventType = "sold"
				e.Quantity = &qty
				e.Price = &price
				events = append(events, e)
			}
		}
	}

	// 5. Returns from customer (повернення від покупця)
	rows4, err := s.db.Query(`
		SELECT d.id, di.quantity, d.currency, d.created_at
		FROM documents d
		JOIN document_items di ON di.document_id = d.id
		WHERE di.product_id = $1
		  AND d.doc_type = 'return_from_customer'
		  AND d.status != 'cancelled'
		ORDER BY d.created_at ASC`, productID)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var e ProductLifecycleEvent
			var qty int
			var id int64
			if err2 := rows4.Scan(&id, &qty, &e.Currency, &e.EventDate); err2 == nil {
				e.RefID = &id
				e.EventType = "returned_from_customer"
				e.Quantity = &qty
				events = append(events, e)
			}
		}
	}

	// 6. Returns to supplier (повернення постачальнику)
	rows5, err := s.db.Query(`
		SELECT d.id, di.quantity, d.currency, d.created_at
		FROM documents d
		JOIN document_items di ON di.document_id = d.id
		WHERE di.product_id = $1
		  AND d.doc_type = 'return_to_supplier'
		  AND d.status != 'cancelled'
		ORDER BY d.created_at ASC`, productID)
	if err == nil {
		defer rows5.Close()
		for rows5.Next() {
			var e ProductLifecycleEvent
			var qty int
			var id int64
			if err2 := rows5.Scan(&id, &qty, &e.Currency, &e.EventDate); err2 == nil {
				e.RefID = &id
				e.EventType = "returned_to_supplier"
				e.Quantity = &qty
				events = append(events, e)
			}
		}
	}

	// Sort all events chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].EventDate.Before(events[j].EventDate)
	})

	return ProductLifecycle{ProductID: productID, Events: events}, nil
}

// --- Attachments ---

const maxAttachmentSize = 10 * 1024 * 1024 // 10 MB

func (s *Store) CreateAttachment(input CreateAttachmentInput) (AttachmentListItem, error) {
	if len(input.Data) > maxAttachmentSize {
		return AttachmentListItem{}, fmt.Errorf("файл перевищує ліміт 10 МБ")
	}
	if input.FileName == "" {
		return AttachmentListItem{}, fmt.Errorf("ім'я файлу обов'язкове")
	}
	validTypes := map[string]bool{
		"service_order": true, "customer_order": true, "purchase": true, "sale": true,
	}
	if !validTypes[input.EntityType] {
		return AttachmentListItem{}, fmt.Errorf("невідомий тип сутності: %s", input.EntityType)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var a AttachmentListItem
	err := s.db.QueryRow(`
		INSERT INTO attachments (entity_type, entity_id, file_name, mime_type, size_bytes, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, entity_type, entity_id, file_name, mime_type, size_bytes, created_at`,
		input.EntityType, input.EntityID, input.FileName, input.MimeType, len(input.Data), input.Data,
	).Scan(&a.ID, &a.EntityType, &a.EntityID, &a.FileName, &a.MimeType, &a.SizeBytes, &a.CreatedAt)
	if err != nil {
		return AttachmentListItem{}, fmt.Errorf("не вдалося зберегти вкладення: %w", err)
	}
	return a, nil
}

func (s *Store) ListAttachments(entityType string, entityID int64) ([]AttachmentListItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, entity_type, entity_id, file_name, mime_type, size_bytes, created_at
		FROM attachments
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at ASC`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AttachmentListItem
	for rows.Next() {
		var a AttachmentListItem
		if err := rows.Scan(&a.ID, &a.EntityType, &a.EntityID, &a.FileName, &a.MimeType, &a.SizeBytes, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	if items == nil {
		items = []AttachmentListItem{}
	}
	return items, nil
}

func (s *Store) GetAttachmentData(id int64) (Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var a Attachment
	err := s.db.QueryRow(`
		SELECT id, entity_type, entity_id, file_name, mime_type, size_bytes, data, created_at
		FROM attachments WHERE id = $1`, id,
	).Scan(&a.ID, &a.EntityType, &a.EntityID, &a.FileName, &a.MimeType, &a.SizeBytes, &a.Data, &a.CreatedAt)
	if err != nil {
		return Attachment{}, fmt.Errorf("вкладення не знайдено: %w", err)
	}
	return a, nil
}

func (s *Store) DeleteAttachment(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`DELETE FROM attachments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("вкладення %d не знайдено", id)
	}
	return nil
}

// ── Reports ───────────────────────────────────────────────────────────────────

// SupplierReport aggregates purchases, supplier-order payments and debts per supplier.
func (s *Store) SupplierReport() (SupplierReport, error) {
	if s.db == nil {
		return SupplierReport{Rows: []SupplierReportRow{}}, nil
	}
	// List suppliers without holding the mutex — listSuppliersLocked is only safe when
	// called with the lock already held by the caller, but all queries below go directly
	// to the DB so we must NOT hold the read-lock while calling ListDebts (which also
	// calls s.mu.RLock in its in-memory path), as that would deadlock.
	s.mu.RLock()
	suppliers, err := s.listSuppliersLocked()
	s.mu.RUnlock()
	if err != nil {
		return SupplierReport{}, err
	}

	// purchases totals per supplier
	purchasedBySupplier := map[int64]float64{}
	ordersCountBySupplier := map[int64]int{}
	rowsPurchases, err := s.db.Query(`SELECT supplier_id, total_uah FROM purchases`)
	if err != nil {
		return SupplierReport{}, err
	}
	defer rowsPurchases.Close()
	for rowsPurchases.Next() {
		var sid int64
		var totalUAH float64
		if err := rowsPurchases.Scan(&sid, &totalUAH); err != nil {
			continue
		}
		purchasedBySupplier[sid] += totalUAH
		ordersCountBySupplier[sid]++
	}

	// Paid amounts per supplier: sum payments linked to supplier_orders.
	// ListDebts() only returns customer-facing debts (orders/sales/service_orders) and
	// never produces EntityType=="supplier_order", so we query payments directly instead.
	paidBySupplier := map[int64]float64{}
	rowsPaid, err := s.db.Query(`
		SELECT so.supplier_id, COALESCE(SUM(p.amount_uah), 0)
		FROM supplier_orders so
		LEFT JOIN payments p ON p.supplier_order_id = so.id
		GROUP BY so.supplier_id
	`)
	if err != nil {
		return SupplierReport{}, err
	}
	defer rowsPaid.Close()
	for rowsPaid.Next() {
		var sid int64
		var paidUAH float64
		if err := rowsPaid.Scan(&sid, &paidUAH); err != nil {
			continue
		}
		paidBySupplier[sid] += paidUAH
	}

	// Debt per supplier: total of supplier_orders minus payments already counted above.
	debtsBySupplier := map[int64]float64{}
	rowsDebt, err := s.db.Query(`
		SELECT so.supplier_id,
		       COALESCE(SUM(so.total_uah), 0) - COALESCE(SUM(p.amount_uah), 0)
		FROM supplier_orders so
		LEFT JOIN payments p ON p.supplier_order_id = so.id
		GROUP BY so.supplier_id
		HAVING COALESCE(SUM(so.total_uah), 0) > COALESCE(SUM(p.amount_uah), 0)
	`)
	if err != nil {
		return SupplierReport{}, err
	}
	defer rowsDebt.Close()
	for rowsDebt.Next() {
		var sid int64
		var debtUAH float64
		if err := rowsDebt.Scan(&sid, &debtUAH); err != nil {
			continue
		}
		if debtUAH > 0 {
			debtsBySupplier[sid] = debtUAH
		}
	}

	var report SupplierReport
	for _, sup := range suppliers {
		row := SupplierReportRow{
			SupplierID:   sup.ID,
			SupplierName: sup.Name,
			OrdersCount:  ordersCountBySupplier[sup.ID],
			PurchasedUAH: purchasedBySupplier[sup.ID],
			PaidUAH:      paidBySupplier[sup.ID],
			DebtUAH:      debtsBySupplier[sup.ID],
		}
		report.Rows = append(report.Rows, row)
		report.TotalPurchased += row.PurchasedUAH
		report.TotalPaid += row.PaidUAH
		report.TotalDebt += row.DebtUAH
	}
	if report.Rows == nil {
		report.Rows = []SupplierReportRow{}
	}
	return report, nil
}

func (s *Store) listSuppliersLocked() ([]Supplier, error) {
	if s.db == nil {
		return []Supplier{}, nil
	}
	rows, err := s.db.Query(`SELECT id, name, contact, phone, email, comments, created_at FROM suppliers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Supplier
	for rows.Next() {
		var sup Supplier
		if err := rows.Scan(&sup.ID, &sup.Name, &sup.Contact, &sup.Phone, &sup.Email, &sup.Comments, &sup.CreatedAt); err != nil {
			continue
		}
		out = append(out, sup)
	}
	return out, nil
}

// CounterpartyReport aggregates sales, purchases, payments and debts per counterparty.
func (s *Store) CounterpartyReport() (CounterpartyReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counterparties, err := s.listCounterpartiesLocked()
	if err != nil {
		return CounterpartyReport{}, err
	}

	// Sales totals per customer (matched via counterparty.customer_id → orders/sales)
	salesByCustomer := map[int64]float64{}
	rowsSales, err := s.db.Query(`SELECT COALESCE(order_id, 0), total_uah FROM sales WHERE order_id IS NOT NULL`)
	if err == nil {
		defer rowsSales.Close()
		for rowsSales.Next() {
			var oid int64
			var total float64
			if err := rowsSales.Scan(&oid, &total); err != nil {
				continue
			}
			// get customer name from order
			var cname string
			_ = s.db.QueryRow(`SELECT customer_name FROM customer_orders WHERE id = $1`, oid).Scan(&cname)
			salesByCustomer[oid] += total
			_ = cname
		}
	}

	// Sales directly per customer_name matched to counterparty name
	salesByName := map[string]float64{}
	rowsSalesName, err2 := s.db.Query(`
		SELECT o.customer_name, COALESCE(SUM(sa.total_uah),0)
		FROM customer_orders o
		LEFT JOIN sales sa ON sa.order_id = o.id
		GROUP BY o.customer_name`)
	if err2 == nil {
		defer rowsSalesName.Close()
		for rowsSalesName.Next() {
			var name string
			var total float64
			if err := rowsSalesName.Scan(&name, &total); err != nil {
				continue
			}
			salesByName[name] += total
		}
	}

	// Purchases per supplier (matched via counterparty.supplier_id)
	purchasedBySupplier := map[int64]float64{}
	rowsPurch, err3 := s.db.Query(`SELECT supplier_id, SUM(total_uah) FROM purchases GROUP BY supplier_id`)
	if err3 == nil {
		defer rowsPurch.Close()
		for rowsPurch.Next() {
			var sid int64
			var total float64
			if err := rowsPurch.Scan(&sid, &total); err != nil {
				continue
			}
			purchasedBySupplier[sid] += total
		}
	}

	// Debts per entity
	debts, _ := s.ListDebts()
	debtByOrder := map[int64]float64{}
	paidByOrder := map[int64]float64{}
	for _, d := range debts {
		if d.EntityType == "order" || d.EntityType == "sale" {
			debtByOrder[d.EntityID] += d.DebtUAH
			paidByOrder[d.EntityID] += d.PaidUAH
		}
	}

	var report CounterpartyReport
	for _, cp := range counterparties {
		row := CounterpartyReportRow{
			CounterpartyID:   cp.ID,
			CounterpartyName: cp.Name,
			IsCustomer:       cp.IsCustomer,
			IsSupplier:       cp.IsSupplier,
		}
		// Match sales by name
		row.SalesUAH = salesByName[cp.Name]
		// Match purchases by supplier_id link
		if cp.SupplierID != nil {
			row.PurchasedUAH = purchasedBySupplier[*cp.SupplierID]
		}
		// Debts: sum from orders matching customer_name
		var totalDebt, totalPaid float64
		orderRows, err4 := s.db.Query(`SELECT id FROM customer_orders WHERE customer_name = $1`, cp.Name)
		if err4 == nil {
			defer orderRows.Close()
			for orderRows.Next() {
				var oid int64
				if err := orderRows.Scan(&oid); err != nil {
					continue
				}
				totalDebt += debtByOrder[oid]
				totalPaid += paidByOrder[oid]
			}
		}
		row.DebtUAH = totalDebt
		row.PaidUAH = totalPaid

		report.Rows = append(report.Rows, row)
		report.TotalSales += row.SalesUAH
		report.TotalPurchased += row.PurchasedUAH
		report.TotalPaid += row.PaidUAH
		report.TotalDebt += row.DebtUAH
	}
	if report.Rows == nil {
		report.Rows = []CounterpartyReportRow{}
	}
	return report, nil
}

func (s *Store) listCounterpartiesLocked() ([]Counterparty, error) {
	rows, err := s.db.Query(`
		SELECT id, name, phone, email, comment, is_customer, is_supplier,
		       customer_id, supplier_id, created_at
		FROM counterparties ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Counterparty
	for rows.Next() {
		var cp Counterparty
		if err := rows.Scan(
			&cp.ID, &cp.Name, &cp.Phone, &cp.Email, &cp.Comment,
			&cp.IsCustomer, &cp.IsSupplier, &cp.CustomerID, &cp.SupplierID, &cp.CreatedAt,
		); err != nil {
			continue
		}
		out = append(out, cp)
	}
	return out, nil
}

func (s *Store) UpdateSupplierOrderSupplier(supplierOrderID int64, supplierID int64) error {
	if supplierID <= 0 {
		return errors.New("supplier id is required")
	}
	if s.db != nil {
		// verify supplier exists
		var exists bool
		if err := s.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM suppliers WHERE id = $1)`, supplierID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errors.New("supplier not found")
		}
		result, err := s.db.Exec(`
			UPDATE supplier_orders SET supplier_id = $2, updated_at = NOW() WHERE id = $1
		`, supplierOrderID, supplierID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return errors.New("supplier order not found")
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.supplierOrders {
		if s.supplierOrders[i].ID == supplierOrderID {
			s.supplierOrders[i].SupplierID = supplierID
			s.supplierOrders[i].UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("supplier order not found")
}