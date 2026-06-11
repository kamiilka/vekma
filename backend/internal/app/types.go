package app

import "time"

type User struct {
	Username string `json:"username"`
	Password string `json:"-"`
	Role     string `json:"role"`
}

type UserAccessScopes struct {
	WarehouseIDs []int64 `json:"warehouseIds"`
	CashboxIDs   []int64 `json:"cashboxIds"`
}

type UserSessionInfo struct {
	User        string           `json:"user"`
	Role        string           `json:"role"`
	Permissions []string         `json:"permissions"`
	Scopes      UserAccessScopes `json:"scopes"`
}

type AuthSession struct {
	Token string `json:"token,omitempty"`
	UserSessionInfo
}

type RolePermissions struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type Product struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Code              string    `json:"code"`
	SKU               string    `json:"sku"`
	Article           string    `json:"article"`
	Barcode           string    `json:"barcode"`
	SerialNumber      string    `json:"serialNumber"`
	Category          string    `json:"category"`
	Brand             string    `json:"brand"`
	Supplier          string    `json:"supplier"`
	PurchasePrice     float64   `json:"purchasePrice"`
	RetailPrice       float64   `json:"retailPrice"`
	WholesalePrice    float64   `json:"wholesalePrice"`
	Currency          string    `json:"currency"`
	VATPercent        float64   `json:"vatPercent"`
	Stock             int       `json:"stock"`
	MinStock          int       `json:"minStock"`
	WarehousePosition string    `json:"warehousePosition"`
	Comments          string    `json:"comments"`
	Archived          bool      `json:"archived"`
	// Supplier name mapping: how the supplier calls this product
	SupplierSKU      string `json:"supplierSku"`
	SupplierNameExt  string `json:"supplierNameExt"`
	CreatedAt         time.Time `json:"createdAt"`
}

type ProductDuplicate struct {
	Field      string   `json:"field"`
	Value      string   `json:"value"`
	ProductIDs []int64  `json:"productIds"`
	SKUs       []string `json:"skus"`
	Names      []string `json:"names"`
}

type ProductCSVImportRowError struct {
	Line  int    `json:"line"`
	Error string `json:"error"`
}

type ProductCSVImportResult struct {
	Imported int                        `json:"imported"`
	Updated  int                        `json:"updated"`
	Skipped  int                        `json:"skipped"`
	Errors   []ProductCSVImportRowError `json:"errors"`
}

type PurchaseCSVImportResult struct {
	Purchase Purchase                   `json:"purchase"`
	Imported int                        `json:"imported"`
	Skipped  int                        `json:"skipped"`
	Errors   []ProductCSVImportRowError `json:"errors"`
}

type DocumentCSVImportResult struct {
	Document Document                   `json:"document"`
	Imported int                        `json:"imported"`
	Skipped  int                        `json:"skipped"`
	Errors   []ProductCSVImportRowError `json:"errors"`
}

type ProductPriceBulkUpdateRequest struct {
	Mode            string  `json:"mode"` // percent | fixed
	Value           float64 `json:"value"`
	PriceField      string  `json:"priceField"`          // retailPrice | purchasePrice | wholesalePrice
	RoundMode       string  `json:"roundMode,omitempty"` // none | nearest | up | down
	RoundTo         float64 `json:"roundTo,omitempty"`
	Search          string  `json:"search,omitempty"`
	Category        string  `json:"category,omitempty"`
	Brand           string  `json:"brand,omitempty"`
	Supplier        string  `json:"supplier,omitempty"`
	IncludeArchived bool    `json:"includeArchived"`
}

type ProductPriceBulkUpdateResult struct {
	Updated int `json:"updated"`
}

type ProductMergeResult struct {
	TargetProductID  int64   `json:"targetProductId"`
	MergedProductIDs []int64 `json:"mergedProductIds"`
}

type StockMovement struct {
	ID              int64     `json:"id"`
	ProductID       int64     `json:"productId"`
	FromWarehouseID *int64    `json:"fromWarehouseId,omitempty"`
	ToWarehouseID   *int64    `json:"toWarehouseId,omitempty"`
	FromCellID      *int64    `json:"fromCellId,omitempty"`
	ToCellID        *int64    `json:"toCellId,omitempty"`
	Type            string    `json:"type"`
	Quantity        int       `json:"quantity"`
	Note            string    `json:"note"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Warehouse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	IsVirtual    bool      `json:"isVirtual"`
	LocationType string    `json:"locationType"` // "warehouse" | "shop"
	CreatedAt    time.Time `json:"createdAt"`
}

type WarehouseStock struct {
	WarehouseID int64     `json:"warehouseId"`
	ProductID   int64     `json:"productId"`
	Quantity    int       `json:"quantity"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type WarehouseZone struct {
	ID          int64     `json:"id"`
	WarehouseID int64     `json:"warehouseId"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"createdAt"`
}

type WarehouseCell struct {
	ID          int64     `json:"id"`
	WarehouseID int64     `json:"warehouseId"`
	ZoneID      int64     `json:"zoneId"`
	Code        string    `json:"code"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CellStock struct {
	CellID    int64     `json:"cellId"`
	ProductID int64     `json:"productId"`
	Quantity  int       `json:"quantity"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type StockTransferItem struct {
	ProductID  int64  `json:"productId"`
	Quantity   int    `json:"quantity"`
	FromCellID *int64 `json:"fromCellId,omitempty"`
	ToCellID   *int64 `json:"toCellId,omitempty"`
}

type CellTransferFIFOItem struct {
	ProductID int64 `json:"productId"`
	Quantity  int   `json:"quantity"`
}

type StockTransfer struct {
	ID              int64               `json:"id"`
	FromWarehouseID int64               `json:"fromWarehouseId"`
	ToWarehouseID   int64               `json:"toWarehouseId"`
	Items           []StockTransferItem `json:"items"`
	Note            string              `json:"note"`
	CreatedAt       time.Time           `json:"createdAt"`
}

const (
	InventoryStatusDraft    = "draft"
	InventoryStatusApplied  = "applied"
	InventoryStatusCanceled = "canceled"
)

type InventoryItem struct {
	ProductID      int64 `json:"productId"`
	SystemQuantity int   `json:"systemQuantity"`
	ActualQuantity int   `json:"actualQuantity"`
	Adjustment     int   `json:"adjustment"`
}

type Inventory struct {
	ID          int64           `json:"id"`
	WarehouseID int64           `json:"warehouseId"`
	Status      string          `json:"status"`
	Items       []InventoryItem `json:"items"`
	Note        string          `json:"note"`
	AppliedAt   *time.Time      `json:"appliedAt,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
}

const (
	DocumentTypeInvoice            = "invoice"
	DocumentTypeAct                = "act"
	DocumentTypeCashInOrder        = "cash_in_order"
	DocumentTypeCashOutOrder       = "cash_out_order"
	DocumentTypeVAT                = "vat"
	DocumentTypeReturnFromCustomer = "return_from_customer"
	DocumentTypeReturnToSupplier   = "return_to_supplier"
)

const (
	DocumentStatusDraft     = "draft"
	DocumentStatusPosted    = "posted"
	DocumentStatusCancelled = "cancelled"
)

type DocumentItem struct {
	ProductID  int64   `json:"productId"`
	Quantity   int     `json:"quantity"`
	Price      float64 `json:"price"`
	VATPercent float64 `json:"vatPercent,omitempty"`
}

type Document struct {
	ID                   int64          `json:"id"`
	Type                 string         `json:"type"`
	Number               string         `json:"number"`
	Status               string         `json:"status"`
	SourceSaleID         *int64         `json:"sourceSaleId,omitempty"`
	SourcePurchaseID     *int64         `json:"sourcePurchaseId,omitempty"`
	SourceServiceOrderID *int64         `json:"sourceServiceOrderId,omitempty"`
	WarehouseID          *int64         `json:"warehouseId,omitempty"`
	CashboxID            *int64         `json:"cashboxId,omitempty"`
	Currency             string         `json:"currency"`
	Total                float64        `json:"total"`
	Items                []DocumentItem `json:"items"`
	Note                 string         `json:"note"`
	PostedAt             *time.Time     `json:"postedAt,omitempty"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

type ReturnAvailabilityItem struct {
	ProductID    int64   `json:"productId"`
	SourceQty    int     `json:"sourceQty"`
	ReturnedQty  int     `json:"returnedQty"`
	AvailableQty int     `json:"availableQty"`
	Price        float64 `json:"price"`
}

type ReturnAvailability struct {
	SourceID int64                    `json:"sourceId"`
	Currency string                   `json:"currency"`
	Items    []ReturnAvailabilityItem `json:"items"`
}

type DocumentTemplate struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TemplatePlaceholder struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DocumentTemplateValidation struct {
	Valid           bool     `json:"valid"`
	Strict          bool     `json:"strict"`
	Used            []string `json:"used"`
	Unknown         []string `json:"unknown"`
	MissingRequired []string `json:"missingRequired"`
	Allowed         []string `json:"allowed"`
	Required        []string `json:"required"`
	TemplateLen     int      `json:"templateLen"`
}

type DocumentTemplatePreview struct {
	Content    string                     `json:"content"`
	Validation DocumentTemplateValidation `json:"validation"`
}

type SaleItem struct {
	ProductID int64   `json:"productId"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type PurchaseItem struct {
	ProductID int64   `json:"productId"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type SupplierOrderReceiveLine struct {
	ProductID int64    `json:"productId"`
	Quantity  int      `json:"quantity"`
	Price     *float64 `json:"price,omitempty"`
}

type SupplierOrderPendingItem struct {
	ProductID int64   `json:"productId"`
	Ordered   int     `json:"ordered"`
	Received  int     `json:"received"`
	Pending   int     `json:"pending"`
	Price     float64 `json:"price"`
}

type Sale struct {
	ID        int64      `json:"id"`
	OrderID   *int64     `json:"orderId,omitempty"`
	Items     []SaleItem `json:"items"`
	Total     float64    `json:"total"`
	Currency  string     `json:"currency"`
	TotalUAH  float64    `json:"totalUah"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
}

const (
	ReceiptProviderCheckbox = "checkbox"
)

const (
	ReceiptStatusPending = "pending"
	ReceiptStatusSent    = "sent"
	ReceiptStatusFailed  = "failed"
)

type Receipt struct {
	ID           int64      `json:"id"`
	SaleID       int64      `json:"saleId"`
	Provider     string     `json:"provider"`
	Status       string     `json:"status"`
	ExternalID   string     `json:"externalId,omitempty"`
	FiscalNumber string     `json:"fiscalNumber,omitempty"`
	QRURL        string     `json:"qrUrl,omitempty"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	Payload      string     `json:"payload,omitempty"`
	SentAt       *time.Time `json:"sentAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type ReceiptBulkRetryResult struct {
	Attempted int       `json:"attempted"`
	Succeeded int       `json:"succeeded"`
	Failed    int       `json:"failed"`
	Items     []Receipt `json:"items"`
}

const (
	// Customer order statuses (замовлення покупця)
	OrderStatusNew          = "new"           // Нове
	OrderStatusInWork       = "in_work"       // В роботі
	OrderStatusOrdered      = "ordered"       // Замовлено у постачальника
	OrderStatusExpected     = "expected"      // Очікується
	OrderStatusArrived      = "arrived"       // Надійшло
	OrderStatusIssued       = "issued"        // Видано
	OrderStatusClosed       = "closed"        // Закрито
	OrderStatusCancelled    = "cancelled"     // Скасовано

	// Legacy aliases kept for backward compatibility
	OrderStatusDraft      = "new"
	OrderStatusConfirmed  = "in_work"
	OrderStatusProcessing = "ordered"
	OrderStatusPaid       = "arrived"
	OrderStatusCompleted  = "closed"
)

const (
	ReservationStatusActive   = "active"
	ReservationStatusReleased = "released"
	ReservationStatusConsumed = "consumed"
	ReservationStatusExpired  = "expired"
)

type CustomerOrder struct {
	ID           int64      `json:"id"`
	CustomerName string     `json:"customerName"`
	Status       string     `json:"status"`
	Currency     string     `json:"currency"`
	Total        float64    `json:"total"`
	Paid         float64    `json:"paid"`
	Debt         float64    `json:"debt"`
	TotalUAH     float64    `json:"totalUah"`
	PaidUAH      float64    `json:"paidUah"`
	DebtUAH      float64    `json:"debtUah"`
	DueDate      *time.Time `json:"dueDate,omitempty"`
	Items        []SaleItem `json:"items"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

const (
	CustomerReminderStatusPending = "pending"
	CustomerReminderStatusDone    = "done"
)

type Customer struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CustomerReminder struct {
	ID          int64      `json:"id"`
	CustomerID  int64      `json:"customerId"`
	Text        string     `json:"text"`
	DueAt       *time.Time `json:"dueAt,omitempty"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

const (
	ServiceOrderStatusNew        = "new"
	ServiceOrderStatusInProgress = "in_progress"
	ServiceOrderStatusDone       = "done"
	ServiceOrderStatusCancelled  = "cancelled"
)

type ServiceOrder struct {
	ID          int64              `json:"id"`
	CustomerID  int64              `json:"customerId"`
	ProductID   *int64             `json:"productId,omitempty"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Technician  string             `json:"technician"`
	LaborMin    int                `json:"laborMin"`
	Status      string             `json:"status"`
	Price       float64            `json:"price"`
	PartsTotal  float64            `json:"partsTotal"`
	Total       float64            `json:"total"`
	Paid        float64            `json:"paid"`
	Debt        float64            `json:"debt"`
	TotalUAH    float64            `json:"totalUah"`
	PaidUAH     float64            `json:"paidUah"`
	DebtUAH     float64            `json:"debtUah"`
	Currency    string             `json:"currency"`
	Parts       []ServiceOrderPart `json:"parts"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	CompletedAt *time.Time         `json:"completedAt,omitempty"`
}

type ServiceOrderPart struct {
	ID             int64     `json:"id"`
	ServiceOrderID int64     `json:"serviceOrderId"`
	ProductID      int64     `json:"productId"`
	Quantity       int       `json:"quantity"`
	Price          float64   `json:"price"`
	Total          float64   `json:"total"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ServiceCategory struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type Service struct {
	ID          int64     `json:"id"`
	CategoryID  int64     `json:"categoryId"`
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	Currency    string    `json:"currency"`
	DurationMin int       `json:"durationMin"`
	CreatedAt   time.Time `json:"createdAt"`
}

const (
	// Supplier order statuses (замовлення постачальнику)
	SupplierOrderStatusDraft             = "draft"       // Створено
	SupplierOrderStatusSent              = "sent"        // Відправлено
	SupplierOrderStatusConfirmed         = "confirmed"   // Підтверджено
	SupplierOrderStatusInTransit         = "in_transit"  // В дорозі
	SupplierOrderStatusReceived          = "received"    // Отримано
	SupplierOrderStatusClosed            = "closed"      // Закрито
	SupplierOrderStatusCancelled         = "cancelled"   // Скасовано

	// Legacy alias
	SupplierOrderStatusPartiallyReceived = "in_transit"
)

type Supplier struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Contact   string    `json:"contact"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Comments  string    `json:"comments"`
	CreatedAt time.Time `json:"createdAt"`
}

type SupplierOrder struct {
	ID              int64          `json:"id"`
	SupplierID      int64          `json:"supplierId"`
	CustomerOrderID *int64         `json:"customerOrderId,omitempty"`
	Status          string         `json:"status"`
	Currency        string         `json:"currency"`
	Total           float64        `json:"total"`
	TotalUAH        float64        `json:"totalUah"`
	Items           []PurchaseItem `json:"items"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type SupplierOrderPendingSummary struct {
	OrderID         int64                      `json:"orderId"`
	SupplierID      int64                      `json:"supplierId"`
	Currency        string                     `json:"currency"`
	Status          string                     `json:"status"`
	PendingItems    []SupplierOrderPendingItem `json:"pendingItems"`
	PendingTotal    float64                    `json:"pendingTotal"`
	PendingTotalUAH float64                    `json:"pendingTotalUah"`
	UpdatedAt       time.Time                  `json:"updatedAt"`
}

type PurchaseRecommendation struct {
	ProductID           int64  `json:"productId"`
	ProductName         string `json:"productName"`
	SKU                 string `json:"sku"`
	SupplierSKU         string `json:"supplierSku"`
	Supplier            string `json:"supplier"`
	SuggestedSupplierID *int64 `json:"suggestedSupplierId,omitempty"`
	CurrentStock        int    `json:"currentStock"`
	Reserved            int    `json:"reserved"`
	Available           int    `json:"available"`
	MinStock            int    `json:"minStock"`
	SoldLast30Days      int    `json:"soldLast30Days"`
	RecommendedQty      int    `json:"recommendedQty"`
}

type PurchaseRecommendationOrderLine struct {
	ProductID int64    `json:"productId"`
	Quantity  int      `json:"quantity"`
	Price     *float64 `json:"price,omitempty"`
}

type PurchaseRecommendationGroup struct {
	SupplierID       *int64                   `json:"supplierId,omitempty"`
	SupplierName     string                   `json:"supplierName"`
	Items            []PurchaseRecommendation `json:"items"`
	TotalRecommended int                      `json:"totalRecommended"`
	ProductsCount    int                      `json:"productsCount"`
}

type PurchaseRecommendationCreateOrderRequest struct {
	SupplierID int64                             `json:"supplierId"`
	Currency   string                            `json:"currency,omitempty"`
	Items      []PurchaseRecommendationOrderLine `json:"items"`
}

type Purchase struct {
	ID              int64          `json:"id"`
	SupplierID      int64          `json:"supplierId"`
	SupplierOrderID *int64         `json:"supplierOrderId,omitempty"`
	Currency        string         `json:"currency"`
	Total           float64        `json:"total"`
	TotalUAH        float64        `json:"totalUah"`
	Items           []PurchaseItem `json:"items"`
	Note            string         `json:"note"`
	CreatedAt       time.Time      `json:"createdAt"`
}

type Reservation struct {
	ID         int64      `json:"id"`
	OrderID    int64      `json:"orderId"`
	ProductID  int64      `json:"productId"`
	Quantity   int        `json:"quantity"`
	Status     string     `json:"status"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	ReleasedAt *time.Time `json:"releasedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type Payment struct {
	SupplierOrderID *int64    `json:"supplierOrderId,omitempty"`
	PurchaseID      *int64    `json:"purchaseId,omitempty"`
	ID             int64     `json:"id"`
	OrderID        *int64    `json:"orderId,omitempty"`
	SaleID         *int64    `json:"saleId,omitempty"`
	ServiceOrderID *int64    `json:"serviceOrderId,omitempty"`
	CashboxID      int64     `json:"cashboxId"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	AmountUAH      float64   `json:"amountUah"`
	Method         string    `json:"method"`
	Note           string    `json:"note"`
	CreatedAt      time.Time `json:"createdAt"`
}

type DebtSummary struct {
	EntityType    string     `json:"entityType"`
	EntityID      int64      `json:"entityId"`
	Currency      string     `json:"currency"`
	Total         float64    `json:"total"`
	Paid          float64    `json:"paid"`
	Debt          float64    `json:"debt"`
	TotalUAH      float64    `json:"totalUah"`
	PaidUAH       float64    `json:"paidUah"`
	DebtUAH       float64    `json:"debtUah"`
	DueDate       *time.Time `json:"dueDate,omitempty"`
	IsOverdue     bool       `json:"isOverdue"`
	OverdueDays   int        `json:"overdueDays"`
	LastPaymentAt *time.Time `json:"lastPaymentAt,omitempty"`
}

type DebtPaymentHistoryEntry struct {
	PaymentID        int64     `json:"paymentId"`
	Amount           float64   `json:"amount"`
	AmountUAH        float64   `json:"amountUah"`
	Currency         string    `json:"currency"`
	Method           string    `json:"method"`
	Note             string    `json:"note"`
	CreatedAt        time.Time `json:"createdAt"`
	RemainingDebt    float64   `json:"remainingDebt"`
	RemainingDebtUAH float64   `json:"remainingDebtUah"`
}

const (
	PaymentMethodCash    = "cash"
	PaymentMethodCard    = "card"
	PaymentMethodBank    = "bank"
	PaymentMethodVirtual = "virtual"
)

const (
	CashOperationTypeIncoming = "incoming"
	CashOperationTypeOutgoing = "outgoing"
)

const (
	CashShiftStatusOpen   = "open"
	CashShiftStatusClosed = "closed"
)

type Cashbox struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Currency  string    `json:"currency"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
}

type CashOperation struct {
	ID          int64     `json:"id"`
	CashboxID   int64     `json:"cashboxId"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	AmountUAH   float64   `json:"amountUah"`
	Method      string    `json:"method"`
	PaymentID   *int64    `json:"paymentId,omitempty"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CashShift struct {
	ID             int64      `json:"id"`
	CashboxID      int64      `json:"cashboxId"`
	Status         string     `json:"status"`
	OpenedBy       string     `json:"openedBy"`
	ClosedBy       string     `json:"closedBy,omitempty"`
	OpeningBalance float64    `json:"openingBalance"`
	ClosingBalance float64    `json:"closingBalance,omitempty"`
	Note           string     `json:"note"`
	OpenedAt       time.Time  `json:"openedAt"`
	ClosedAt       *time.Time `json:"closedAt,omitempty"`
}

type ExchangeRate struct {
	Currency  string    `json:"currency"`
	RateToUAH float64   `json:"rateToUah"`
	UpdatedAt time.Time `json:"updatedAt"`
}

const (
	NotificationChannelEmail    = "email"
	NotificationChannelTelegram = "telegram"
	NotificationChannelSMS      = "sms"
	NotificationChannelViber    = "viber"
)

const (
	NotificationStatusQueued   = "queued"
	NotificationStatusSent     = "sent"
	NotificationStatusSentStub = "sent_stub"
	NotificationStatusFailed   = "failed"
)

const (
	BackgroundJobTypeOverdueReminders  = "overdue_reminders"
	BackgroundJobTypeReceiptRetries    = "receipt_retries"
	BackgroundJobTypeReservationExpiry = "reservation_expiry"
)

const (
	BackgroundJobStatusQueued    = "queued"
	BackgroundJobStatusRunning   = "running"
	BackgroundJobStatusCompleted = "completed"
	BackgroundJobStatusFailed    = "failed"
)

type NotificationTemplate struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Notification struct {
	ID           int64      `json:"id"`
	Channel      string     `json:"channel"`
	Recipient    string     `json:"recipient"`
	Subject      string     `json:"subject"`
	Body         string     `json:"body"`
	EntityType   string     `json:"entityType"`
	EntityID     int64      `json:"entityId"`
	Status       string     `json:"status"`
	Attempts     int        `json:"attempts"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	SentAt       *time.Time `json:"sentAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type QuickMessageRequest struct {
	Channel    string `json:"channel"`
	Recipient  string `json:"recipient"`
	Sender     string `json:"sender,omitempty"` // override "from" address/phone
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   int64  `json:"entityId,omitempty"`
}

type BackgroundJob struct {
	ID           int64      `json:"id"`
	JobType      string     `json:"jobType"`
	Status       string     `json:"status"`
	Attempts     int        `json:"attempts"`
	MaxAttempts  int        `json:"maxAttempts"`
	NextRetryAt  *time.Time `json:"nextRetryAt,omitempty"`
	Payload      string     `json:"payload"`
	Result       string     `json:"result"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type AnalyticsSummary struct {
	ProductCount int     `json:"productCount"`
	LowStock     int     `json:"lowStock"`
	TotalStock   int     `json:"totalStock"`
	SalesCount   int     `json:"salesCount"`
	Revenue      float64 `json:"revenue"`
}

type ProductProfitability struct {
	ProductID    int64   `json:"productId"`
	ProductName  string  `json:"productName"`
	SKU          string  `json:"sku"`
	QuantitySold int     `json:"quantitySold"`
	RevenueUAH   float64 `json:"revenueUah"`
	CostUAH      float64 `json:"costUah"`
	ProfitUAH    float64 `json:"profitUah"`
	MarginPct    float64 `json:"marginPct"`
}

type ProfitabilityReport struct {
	Items        []ProductProfitability `json:"items"`
	TotalRevenue float64                `json:"totalRevenue"`
	TotalCost    float64                `json:"totalCost"`
	TotalProfit  float64                `json:"totalProfit"`
	MarginPct    float64                `json:"marginPct"`
}

type CategoryAnalyticsItem struct {
	Category   string  `json:"category"`
	Quantity   int     `json:"quantity"`
	RevenueUAH float64 `json:"revenueUah"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"createdAt"`
}

type ChangeHistoryEntry struct {
	ID         int64     `json:"id"`
	User       string    `json:"user"`
	Action     string    `json:"action"`
	Entity     string    `json:"entity"`
	OldValue   string    `json:"oldValue"`
	NewValue   string    `json:"newValue"`
	OccurredAt time.Time `json:"occurredAt"`
}

// Counterparty is a unified entity that can be a buyer, a supplier, or both.
type Counterparty struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Phone      string    `json:"phone"`
	Email      string    `json:"email"`
	Comment    string    `json:"comment"`
	IsCustomer bool      `json:"isCustomer"`
	IsSupplier bool      `json:"isSupplier"`
	CustomerID *int64    `json:"customerId,omitempty"`
	SupplierID *int64    `json:"supplierId,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// OrderChainNode is one document in the full chain of a customer order.
type OrderChainNode struct {
	Type     string           `json:"type"`
	ID       int64            `json:"id"`
	Label    string           `json:"label"`
	Status   string           `json:"status,omitempty"`
	Amount   float64          `json:"amount,omitempty"`
	Currency string           `json:"currency,omitempty"`
	Date     time.Time        `json:"date"`
	Children []OrderChainNode `json:"children,omitempty"`
}

type OrderChain struct {
	Root OrderChainNode `json:"root"`
}

// DocumentRegistryItem is one row in the unified document registry search result.
type DocumentRegistryItem struct {
	DocType     string     `json:"docType"`    // customer_order, supplier_order, purchase, sale, service_order, payment_in, payment_out, transfer, inventory, document
	ID          int64      `json:"id"`
	Number      string     `json:"number"`     // human-readable number / title
	Status      string     `json:"status,omitempty"`
	CounterName string     `json:"counterName,omitempty"` // customer / supplier name
	Total       float64    `json:"total,omitempty"`
	Currency    string     `json:"currency,omitempty"`
	ProductHits []string   `json:"productHits,omitempty"` // matching product names
	Note        string     `json:"note,omitempty"`
	Date        time.Time  `json:"date"`
}

// ProductLifecycleEvent represents a single event in a product's lifecycle.
type ProductLifecycleEvent struct {
	EventType     string     `json:"eventType"`     // "created", "purchased", "received", "sold", "returned_from_customer", "returned_to_supplier", "movement"
	EventDate     time.Time  `json:"eventDate"`
	RefID         *int64     `json:"refId,omitempty"`         // ID відповідного запису
	Quantity      *int       `json:"quantity,omitempty"`
	Price         *float64   `json:"price,omitempty"`
	Currency      string     `json:"currency,omitempty"`
	SupplierID    *int64     `json:"supplierId,omitempty"`
	SupplierName  string     `json:"supplierName,omitempty"`
	CustomerName  string     `json:"customerName,omitempty"`
	WarehouseName string     `json:"warehouseName,omitempty"`
	Note          string     `json:"note,omitempty"`
}

// ProductLifecycle is the full lifecycle history for a product.
type ProductLifecycle struct {
	ProductID int64                   `json:"productId"`
	Events    []ProductLifecycleEvent `json:"events"`
}

// Attachment categories for UI grouping
const (
	AttachmentCategoryPhoto     = "photo"
	AttachmentCategoryPDF       = "pdf"
	AttachmentCategoryInvoice   = "invoice"
	AttachmentCategoryAct       = "act"
	AttachmentCategoryReceipt   = "receipt"
	AttachmentCategoryWarranty  = "warranty"
	AttachmentCategoryOther     = "other"
)

// Attachment stores a file linked to any entity (service_order, customer_order, purchase, sale).
type Attachment struct {
	ID         int64     `json:"id"`
	EntityType string    `json:"entityType"` // "service_order" | "customer_order" | "purchase" | "sale"
	EntityID   int64     `json:"entityId"`
	FileName   string    `json:"fileName"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int       `json:"sizeBytes"`
	CreatedAt  time.Time `json:"createdAt"`
	// Data is included only on download, not in list responses
	Data []byte `json:"-"`
}

// AttachmentListItem is a lightweight version without the file data.
type AttachmentListItem struct {
	ID         int64     `json:"id"`
	EntityType string    `json:"entityType"`
	EntityID   int64     `json:"entityId"`
	FileName   string    `json:"fileName"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int       `json:"sizeBytes"`
	CreatedAt  time.Time `json:"createdAt"`
}

type CreateAttachmentInput struct {
	EntityType string
	EntityID   int64
	FileName   string
	MimeType   string
	Data       []byte
}

// ── Report types ──────────────────────────────────────────────────────────────

type SupplierReportRow struct {
	SupplierID   int64   `json:"supplierId"`
	SupplierName string  `json:"supplierName"`
	OrdersCount  int     `json:"ordersCount"`
	PurchasedUAH float64 `json:"purchasedUah"`
	PaidUAH      float64 `json:"paidUah"`
	DebtUAH      float64 `json:"debtUah"`
}

type SupplierReport struct {
	Rows         []SupplierReportRow `json:"rows"`
	TotalPurchased float64           `json:"totalPurchasedUah"`
	TotalPaid      float64           `json:"totalPaidUah"`
	TotalDebt      float64           `json:"totalDebtUah"`
}

type CounterpartyReportRow struct {
	CounterpartyID   int64   `json:"counterpartyId"`
	CounterpartyName string  `json:"counterpartyName"`
	IsCustomer       bool    `json:"isCustomer"`
	IsSupplier       bool    `json:"isSupplier"`
	SalesUAH         float64 `json:"salesUah"`
	PurchasedUAH     float64 `json:"purchasedUah"`
	PaidUAH          float64 `json:"paidUah"`
	DebtUAH          float64 `json:"debtUah"`
}

type CounterpartyReport struct {
	Rows      []CounterpartyReportRow `json:"rows"`
	TotalSales     float64            `json:"totalSalesUah"`
	TotalPurchased float64            `json:"totalPurchasedUah"`
	TotalPaid      float64            `json:"totalPaidUah"`
	TotalDebt      float64            `json:"totalDebtUah"`
}
