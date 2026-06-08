package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"

	"erp-backend/internal/app"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createProductRequest struct {
	Name               string  `json:"name"`
	Code               string  `json:"code"`
	SKU                string  `json:"sku"`
	Article            string  `json:"article"`
	Barcode            string  `json:"barcode"`
	SerialNumber       string  `json:"serialNumber"`
	Category           string  `json:"category"`
	Brand              string  `json:"brand"`
	Supplier           string  `json:"supplier"`
	PurchasePrice      float64 `json:"purchasePrice"`
	RetailPrice        float64 `json:"retailPrice"`
	WholesalePrice     float64 `json:"wholesalePrice"`
	Currency           string  `json:"currency"`
	VATPercent         float64 `json:"vatPercent"`
	Stock              int     `json:"stock"`
	MinStock           int     `json:"minStock"`
	WarehousePosition  string  `json:"warehousePosition"`
	Comments           string  `json:"comments"`
	InitialWarehouseID int64   `json:"initialWarehouseId"`
}

type movementRequest struct {
	ProductID   int64  `json:"productId"`
	WarehouseID int64  `json:"warehouseId"`
	Type        string `json:"type"`
	Quantity    int    `json:"quantity"`
	Note        string `json:"note"`
}

type createSaleRequest struct {
	OrderID  *int64         `json:"orderId,omitempty"`
	Items    []app.SaleItem `json:"items"`
	Currency string         `json:"currency,omitempty"`
}

type retryReceiptsBulkRequest struct {
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

type archiveProductRequest struct {
	Archived bool `json:"archived"`
}

type importProductsCSVRequest struct {
	CSV            string `json:"csv"`
	UpdateExisting bool   `json:"updateExisting"`
}

type bulkUpdateProductPricesRequest struct {
	Mode            string  `json:"mode"`
	Value           float64 `json:"value"`
	PriceField      string  `json:"priceField"`
	RoundMode       string  `json:"roundMode,omitempty"`
	RoundTo         float64 `json:"roundTo,omitempty"`
	Search          string  `json:"search,omitempty"`
	Category        string  `json:"category,omitempty"`
	Brand           string  `json:"brand,omitempty"`
	Supplier        string  `json:"supplier,omitempty"`
	IncludeArchived bool    `json:"includeArchived"`
}

type mergeProductsRequest struct {
	TargetProductID  int64   `json:"targetProductId"`
	SourceProductIDs []int64 `json:"sourceProductIds"`
}

type createOrderRequest struct {
	CustomerName string         `json:"customerName"`
	Items        []app.SaleItem `json:"items"`
	Reserve      bool           `json:"reserve"`
	ReserveUntil string         `json:"reserveUntil,omitempty"`
	DueDate      string         `json:"dueDate,omitempty"`
	Currency     string         `json:"currency,omitempty"`
}

type createCustomerRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Comment string `json:"comment"`
}

type createCustomerReminderRequest struct {
	CustomerID int64  `json:"customerId"`
	Text       string `json:"text"`
	DueAt      string `json:"dueAt,omitempty"`
}

type createServiceOrderRequest struct {
	CustomerID  int64   `json:"customerId"`
	ProductID   *int64  `json:"productId,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Technician  string  `json:"technician"`
	LaborMin    int     `json:"laborMin"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency,omitempty"`
}

type updateServiceOrderStatusRequest struct {
	Status string `json:"status"`
}

type updateServiceOrderRequest struct {
	ProductID   *int64  `json:"productId,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Technician  string  `json:"technician"`
	LaborMin    int     `json:"laborMin"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency,omitempty"`
}

type addServiceOrderPartRequest struct {
	ProductID int64   `json:"productId"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type createServiceOrderActRequest struct {
	Note     string `json:"note"`
	AutoPost bool   `json:"autoPost"`
}

type createServiceCategoryRequest struct {
	Name string `json:"name"`
}

type createServiceRequest struct {
	CategoryID  int64   `json:"categoryId"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency,omitempty"`
	DurationMin int     `json:"durationMin"`
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

type updateOrderRequest struct {
	CustomerName string         `json:"customerName"`
	Currency     string         `json:"currency"`
	DueDate      string         `json:"dueDate"`
	Items        []app.SaleItem `json:"items"`
}

type updateSupplierOrderRequest struct {
	CustomerOrderID *int64 `json:"customerOrderId"`
	SupplierID      *int64 `json:"supplierId"`
}

type createPaymentRequest struct {
	OrderID         *int64  `json:"orderId,omitempty"`
	SaleID          *int64  `json:"saleId,omitempty"`
	ServiceOrderID  *int64  `json:"serviceOrderId,omitempty"`
	SupplierOrderID *int64  `json:"supplierOrderId,omitempty"`
	PurchaseID      *int64  `json:"purchaseId,omitempty"`
	CashboxID       int64   `json:"cashboxId,omitempty"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency,omitempty"`
	Method          string  `json:"method"`
	Note            string  `json:"note"`
}

type createCashboxRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Currency string `json:"currency"`
}

type createCashOperationRequest struct {
	CashboxID   int64   `json:"cashboxId"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Method      string  `json:"method"`
	Description string  `json:"description"`
}

type openCashShiftRequest struct {
	CashboxID int64  `json:"cashboxId"`
	Note      string `json:"note"`
}

type closeCashShiftRequest struct {
	Note string `json:"note"`
}

type upsertExchangeRateRequest struct {
	Currency  string  `json:"currency"`
	RateToUAH float64 `json:"rateToUah"`
}

type debtHistoryRequest struct {
	EntityType string `json:"entityType"`
	EntityID   int64  `json:"entityId"`
}

type upsertNotificationTemplateRequest struct {
	Code     string `json:"code"`
	Channel  string `json:"channel"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	IsActive bool   `json:"isActive"`
}

type quickMessageRequest struct {
	Channel    string `json:"channel"`
	Recipient  string `json:"recipient"`
	Sender     string `json:"sender,omitempty"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   int64  `json:"entityId,omitempty"`
}

type enqueueBackgroundJobRequest struct {
	AsOf string `json:"asOf,omitempty"`
}

type enqueueReceiptRetryJobRequest struct {
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type createSupplierRequest struct {
	Name     string `json:"name"`
	Contact  string `json:"contact"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Comments string `json:"comments"`
}

type createSupplierOrderRequest struct {
	SupplierID int64              `json:"supplierId"`
	Currency   string             `json:"currency,omitempty"`
	Items      []app.PurchaseItem `json:"items"`
}

type createSupplierOrderFromRecommendationsRequest struct {
	SupplierID int64                                 `json:"supplierId"`
	Currency   string                                `json:"currency,omitempty"`
	Items      []app.PurchaseRecommendationOrderLine `json:"items"`
}

type createSupplierOrdersBulkFromRecommendationsRequest struct {
	Orders []app.PurchaseRecommendationCreateOrderRequest `json:"orders"`
}

type updateSupplierOrderStatusRequest struct {
	Status string `json:"status"`
}

type createPurchaseRequest struct {
	SupplierID      int64              `json:"supplierId"`
	SupplierOrderID *int64             `json:"supplierOrderId,omitempty"`
	Currency        string             `json:"currency,omitempty"`
	Items           []app.PurchaseItem `json:"items"`
	Note            string             `json:"note"`
}

type importPurchasesCSVRequest struct {
	SupplierID      int64  `json:"supplierId"`
	SupplierOrderID *int64 `json:"supplierOrderId,omitempty"`
	Currency        string `json:"currency,omitempty"`
	CSV             string `json:"csv"`
	Note            string `json:"note"`
}

type receiveSupplierOrderRequest struct {
	Currency string                         `json:"currency,omitempty"`
	Lines    []app.SupplierOrderReceiveLine `json:"lines"`
	Note     string                         `json:"note"`
}

type createWarehouseRequest struct {
	Name         string `json:"name"`
	IsVirtual    bool   `json:"isVirtual"`
	LocationType string `json:"locationType"`
}

type createWarehouseZoneRequest struct {
	WarehouseID int64  `json:"warehouseId"`
	Name        string `json:"name"`
}

type createWarehouseCellRequest struct {
	WarehouseID int64  `json:"warehouseId"`
	ZoneID      int64  `json:"zoneId"`
	Code        string `json:"code"`
}

type createTransferRequest struct {
	FromWarehouseID int64                   `json:"fromWarehouseId"`
	ToWarehouseID   int64                   `json:"toWarehouseId"`
	Items           []app.StockTransferItem `json:"items"`
	Note            string                  `json:"note"`
}

type createCellTransferRequest struct {
	FromCellID int64                   `json:"fromCellId"`
	ToCellID   int64                   `json:"toCellId"`
	Items      []app.StockTransferItem `json:"items"`
	Note       string                  `json:"note"`
}

type createCellTransferFIFORequest struct {
	ToCellID int64                      `json:"toCellId"`
	Items    []app.CellTransferFIFOItem `json:"items"`
	Note     string                     `json:"note"`
}

type createInventoryRequest struct {
	WarehouseID int64               `json:"warehouseId"`
	Items       []app.InventoryItem `json:"items"`
	Note        string              `json:"note"`
}

type createDocumentRequest struct {
	Type        string             `json:"type"`
	WarehouseID *int64             `json:"warehouseId,omitempty"`
	CashboxID   *int64             `json:"cashboxId,omitempty"`
	Currency    string             `json:"currency,omitempty"`
	Total       float64            `json:"total"`
	Items       []app.DocumentItem `json:"items"`
	Note        string             `json:"note"`
}

type importDocumentCSVRequest struct {
	Type        string  `json:"type"`
	WarehouseID *int64  `json:"warehouseId,omitempty"`
	CashboxID   *int64  `json:"cashboxId,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	Total       float64 `json:"total"`
	CSV         string  `json:"csv"`
	Note        string  `json:"note"`
}

type createCustomerReturnDocumentRequest struct {
	SaleID      int64              `json:"saleId"`
	WarehouseID int64              `json:"warehouseId"`
	Currency    string             `json:"currency,omitempty"`
	Items       []app.DocumentItem `json:"items"`
	Note        string             `json:"note"`
}

type createSupplierReturnDocumentRequest struct {
	PurchaseID  int64              `json:"purchaseId"`
	WarehouseID int64              `json:"warehouseId"`
	Currency    string             `json:"currency,omitempty"`
	Items       []app.DocumentItem `json:"items"`
	Note        string             `json:"note"`
}

type upsertDocumentTemplateRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Body     string `json:"body"`
	IsActive bool   `json:"isActive"`
}

type previewDocumentTemplateRequest struct {
	Code       string `json:"code"`
	Body       string `json:"body"`
	DocumentID *int64 `json:"documentId,omitempty"`
	Strict     bool   `json:"strict"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	user, ok := s.store.ValidateUser(req.Username, req.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "невірний логін або пароль")
		return
	}

	token, err := s.tokens.NewToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка авторизації")
		return
	}
	s.store.AddAuditLog(
		user.Username,
		"auth.login",
		"session",
		"user logged in",
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"token": token,
		"role":  user.Role,
		"user":  user.Username,
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	info, err := s.store.SessionInfo(userFromContext(r.Context()), roleFromContext(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження сесії")
		return
	}
	type sessionResponse struct {
		app.UserSessionInfo
		Features map[string]bool `json:"features"`
	}
	writeJSON(w, http.StatusOK, sessionResponse{
		UserSessionInfo: info,
		Features: map[string]bool{
			"checkboxEnabled": s.store.IsReceiptSenderEnabled(),
		},
	})
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	search := r.URL.Query().Get("search")

	products, err := s.store.ListProductsFiltered(search, includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження товарів")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.SKU == "" {
		req.SKU = fmt.Sprintf("SKU-%d", time.Now().UnixNano())
	}

	// When a specific warehouse is chosen we pass Stock=0 to CreateProduct so it
	// doesn't seed the default warehouse.  The CreateInventory/ApplyInventory call
	// below will place the correct quantity in the chosen warehouse instead.
	initialStock := req.Stock
	stockForDefault := req.Stock
	if req.InitialWarehouseID > 0 {
		stockForDefault = 0
	}

	product, err := s.store.CreateProduct(app.Product{
		Name:              req.Name,
		Code:              req.Code,
		SKU:               req.SKU,
		Article:           req.Article,
		Barcode:           req.Barcode,
		SerialNumber:      req.SerialNumber,
		Category:          req.Category,
		Brand:             req.Brand,
		Supplier:          req.Supplier,
		PurchasePrice:     req.PurchasePrice,
		RetailPrice:       req.RetailPrice,
		WholesalePrice:    req.WholesalePrice,
		Currency:          req.Currency,
		VATPercent:        req.VATPercent,
		Stock:             stockForDefault,
		MinStock:          req.MinStock,
		WarehousePosition: req.WarehousePosition,
		Comments:          req.Comments,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка створення товару")
		return
	}

	// If an initial warehouse is specified and stock > 0, place stock there via inventory
	if req.InitialWarehouseID > 0 && initialStock > 0 {
		inv, invErr := s.store.CreateInventory(app.Inventory{
			WarehouseID: req.InitialWarehouseID,
			Items: []app.InventoryItem{
				{ProductID: product.ID, ActualQuantity: initialStock},
			},
			Note: "Initial stock on product creation",
		})
		if invErr == nil {
			_, _ = s.store.ApplyInventory(inv.ID)
			product.Stock = initialStock
		}
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.create",
		"product",
		fmt.Sprintf("product_id=%d sku=%s", product.ID, product.SKU),
	)

	writeJSON(w, http.StatusCreated, product)
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID товару")
		return
	}

	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.Name == "" || req.SKU == "" {
		writeError(w, http.StatusBadRequest, "name and sku are required")
		return
	}

	updated, err := s.store.UpdateProduct(id, app.Product{
		Name:              req.Name,
		Code:              req.Code,
		SKU:               req.SKU,
		Article:           req.Article,
		Barcode:           req.Barcode,
		SerialNumber:      req.SerialNumber,
		Category:          req.Category,
		Brand:             req.Brand,
		Supplier:          req.Supplier,
		PurchasePrice:     req.PurchasePrice,
		RetailPrice:       req.RetailPrice,
		WholesalePrice:    req.WholesalePrice,
		Currency:          req.Currency,
		VATPercent:        req.VATPercent,
		Stock:             req.Stock,
		MinStock:          req.MinStock,
		WarehousePosition: req.WarehousePosition,
		Comments:          req.Comments,
	})
	if err != nil {
		if errors.Is(err, app.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.update",
		"product",
		fmt.Sprintf("product_id=%d sku=%s", updated.ID, updated.SKU),
	)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleArchiveProduct(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID товару")
		return
	}

	var req archiveProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if err := s.store.ArchiveProduct(id, req.Archived); err != nil {
		if errors.Is(err, app.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.archive",
		"product",
		fmt.Sprintf("product_id=%d archived=%t", id, req.Archived),
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGenerateProductBarcode(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID товару")
		return
	}

	product, err := s.store.GenerateProductBarcode(id)
	if err != nil {
		if errors.Is(err, app.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.barcode.generate",
		"product",
		fmt.Sprintf("product_id=%d barcode=%s", product.ID, product.Barcode),
	)
	writeJSON(w, http.StatusOK, product)
}

// handleProductLabelPDF generates a PDF price-tag label for one or more products.
// Query params: ids=1,2,3  format=small|large (default small)
func (s *Server) handleProductLabelPDF(w http.ResponseWriter, r *http.Request) {
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		writeError(w, http.StatusBadRequest, "ids query param required")
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "small"
	}

	var productIDs []int64
	for _, part := range strings.Split(idsParam, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid product id: "+part)
			return
		}
		productIDs = append(productIDs, id)
	}
	if len(productIDs) == 0 {
		writeError(w, http.StatusBadRequest, "не вказано жодного товару")
		return
	}

	products := make([]app.Product, 0, len(productIDs))
	for _, pid := range productIDs {
		p, err := s.store.GetProduct(pid)
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("product %d not found", pid))
			return
		}
		products = append(products, p)
	}

	pdf := buildLabelsPDF(products, format)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=labels.pdf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

// handleProductQRImage returns a simple SVG QR-like placeholder for a product barcode.
// Real QR generation can be swapped in by adding a library; this produces a valid
// SVG that embeds the barcode text as a machine-readable data URI.
func (s *Server) handleProductQRImage(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID товару")
		return
	}

	product, err := s.store.GetProduct(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	code := product.Barcode
	if code == "" {
		code = product.SKU
	}

	svg := buildBarcodeSVG(code, product.Name)
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}



func (s *Server) handleProductDuplicates(w http.ResponseWriter, r *http.Request) {
	duplicates, err := s.store.FindProductDuplicates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка пошуку дублікатів")
		return
	}
	writeJSON(w, http.StatusOK, duplicates)
}

func (s *Server) handleExportProductsCSV(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	content, err := s.store.ExportProductsCSV(includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка експорту товарів у CSV")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=products.csv")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (s *Server) handleImportProductsCSV(w http.ResponseWriter, r *http.Request) {
	var req importProductsCSVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	result, err := s.store.ImportProductsCSV(req.CSV, req.UpdateExisting)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.import_csv",
		"product",
		fmt.Sprintf("imported=%d updated=%d skipped=%d", result.Imported, result.Updated, result.Skipped),
	)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleExportProductsXLSX(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	content, err := s.store.ExportProductsXLSX(includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка експорту товарів у XLSX")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=products.xlsx")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleImportProductsXLSX(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	updateExisting := strings.EqualFold(strings.TrimSpace(r.FormValue("updateExisting")), "true") || r.FormValue("updateExisting") == "1"
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read uploaded file")
		return
	}
	result, err := s.store.ImportProductsXLSX(content, updateExisting)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.import_xlsx",
		"product",
		fmt.Sprintf("imported=%d updated=%d skipped=%d", result.Imported, result.Updated, result.Skipped),
	)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleBulkUpdateProductPrices(w http.ResponseWriter, r *http.Request) {
	var req bulkUpdateProductPricesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	result, err := s.store.BulkUpdateProductPrices(app.ProductPriceBulkUpdateRequest{
		Mode:            req.Mode,
		Value:           req.Value,
		PriceField:      req.PriceField,
		RoundMode:       req.RoundMode,
		RoundTo:         req.RoundTo,
		Search:          req.Search,
		Category:        req.Category,
		Brand:           req.Brand,
		Supplier:        req.Supplier,
		IncludeArchived: req.IncludeArchived,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.bulk_update_prices",
		"product",
		fmt.Sprintf("updated=%d mode=%s price_field=%s value=%.4f", result.Updated, req.Mode, req.PriceField, req.Value),
	)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMergeProducts(w http.ResponseWriter, r *http.Request) {
	var req mergeProductsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	result, err := s.store.MergeDuplicateProducts(req.TargetProductID, req.SourceProductIDs)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"product.merge_duplicates",
		"product",
		fmt.Sprintf("target=%d merged=%v", result.TargetProductID, result.MergedProductIDs),
	)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCreateMovement(w http.ResponseWriter, r *http.Request) {
	var req movementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be greater than zero")
		return
	}

	// Normalize frontend type names to internal type names
	movType := req.Type
	switch req.Type {
	case "receipt":
		movType = "incoming"
	case "writeoff":
		movType = "write_off"
	case "return_to_supplier":
		movType = "return_to_supplier"
	case "adjustment":
		// adjustment sets the exact quantity — handled via inventory
		warehouseID := req.WarehouseID
		if warehouseID <= 0 {
			var wErr error
			warehouseID, wErr = s.store.DefaultWarehouseID()
			if wErr != nil {
				writeError(w, http.StatusInternalServerError, "cannot resolve default warehouse")
				return
			}
		}
		inv, invErr := s.store.CreateInventory(app.Inventory{
			WarehouseID: warehouseID,
			Items: []app.InventoryItem{
				{ProductID: req.ProductID, ActualQuantity: req.Quantity},
			},
			Note: req.Note,
		})
		if invErr != nil {
			writeError(w, http.StatusBadRequest, invErr.Error())
			return
		}
		result, applyErr := s.store.ApplyInventory(inv.ID)
		if applyErr != nil {
			writeError(w, http.StatusBadRequest, applyErr.Error())
			return
		}
		s.store.AddAuditLog(
			userFromContext(r.Context()),
			"stock.movement.adjustment",
			"stock_movement",
			fmt.Sprintf("product_id=%d qty=%d warehouse_id=%d", req.ProductID, req.Quantity, req.WarehouseID),
		)
		writeJSON(w, http.StatusCreated, result)
		return
	}

	movement, err := s.store.CreateStockMovement(app.StockMovement{
		ProductID: req.ProductID,
		Type:      movType,
		Quantity:  req.Quantity,
		Note:      req.Note,
	}, req.WarehouseID)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, app.ErrInsufficientStock):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"stock.movement.create",
		"stock_movement",
		fmt.Sprintf("product_id=%d qty=%d type=%s", req.ProductID, req.Quantity, req.Type),
	)

	writeJSON(w, http.StatusCreated, movement)
}

func (s *Server) handleListStockMovements(w http.ResponseWriter, r *http.Request) {
	var productID *int64
	if raw := r.URL.Query().Get("productId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid productId")
			return
		}
		productID = &value
	}
	var warehouseID *int64
	if raw := r.URL.Query().Get("warehouseId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid warehouseId")
			return
		}
		if !s.hasWarehouseAccess(r, value) {
			writeError(w, http.StatusForbidden, "warehouse access denied")
			return
		}
		warehouseID = &value
	}
	movements, err := s.store.ListStockMovements(productID, warehouseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load stock movements")
		return
	}
	if warehouseID == nil {
		movements = filterStockMovementsByAccess(movements, func(movement app.StockMovement) bool {
			fromAllowed := movement.FromWarehouseID == nil || s.hasWarehouseAccess(r, *movement.FromWarehouseID)
			toAllowed := movement.ToWarehouseID == nil || s.hasWarehouseAccess(r, *movement.ToWarehouseID)
			return fromAllowed && toAllowed
		})
	}
	writeJSON(w, http.StatusOK, movements)
}

func (s *Server) handleCreateSale(w http.ResponseWriter, r *http.Request) {
	var req createSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	sale, err := s.store.CreateSaleFromOrder(req.Items, req.OrderID, req.Currency)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, app.ErrInsufficientStock):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"sale.create",
		"sale",
		fmt.Sprintf("sale_id=%d order_id=%v items=%d total=%.2f", sale.ID, req.OrderID, len(sale.Items), sale.Total),
	)
	if receipt, receiptErr := s.store.ReceiptBySaleID(sale.ID); receiptErr == nil {
		s.store.AddAuditLog(
			userFromContext(r.Context()),
			"receipt.auto.create",
			"receipt",
			fmt.Sprintf("receipt_id=%d sale_id=%d status=%s", receipt.ID, sale.ID, receipt.Status),
		)
	}

	writeJSON(w, http.StatusCreated, sale)
}

func (s *Server) handleListSales(w http.ResponseWriter, r *http.Request) {
	sales, err := s.store.ListSales()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження продажів")
		return
	}
	writeJSON(w, http.StatusOK, sales)
}

func (s *Server) handleListReceipts(w http.ResponseWriter, r *http.Request) {
	var saleID *int64
	if rawSaleID := r.URL.Query().Get("saleId"); rawSaleID != "" {
		parsed, err := strconv.ParseInt(rawSaleID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid saleId")
			return
		}
		saleID = &parsed
	}

	var status *string
	if rawStatus := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status"))); rawStatus != "" {
		switch rawStatus {
		case app.ReceiptStatusPending, app.ReceiptStatusSent, app.ReceiptStatusFailed:
			status = &rawStatus
		default:
			writeError(w, http.StatusBadRequest, "невірний статус")
			return
		}
	}

	receipts, err := s.store.ListReceipts(saleID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження чеків")
		return
	}
	writeJSON(w, http.StatusOK, receipts)
}

func (s *Server) handleGetReceipt(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid receipt id")
		return
	}

	receipt, err := s.store.ReceiptByID(id)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrReceiptNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "cannot load receipt")
		}
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (s *Server) handleRetryReceipt(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid receipt id")
		return
	}

	receipt, err := s.store.RetryReceipt(id)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrReceiptNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, app.ErrReceiptSenderNotConfigured):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}

	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"receipt.retry",
		"receipt",
		fmt.Sprintf("receipt_id=%d sale_id=%d status=%s", receipt.ID, receipt.SaleID, receipt.Status),
	)
	writeJSON(w, http.StatusOK, receipt)
}

// handleSendReceiptForSale creates (or fetches) the fiscal receipt for a sale
// and triggers an immediate send attempt to the Checkbox provider.
func (s *Server) handleSendReceiptForSale(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	saleID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || saleID <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID продажу")
		return
	}

	sale, err := s.store.GetSale(saleID)
	if err != nil {
		if errors.Is(err, app.ErrSaleNotFound) {
			writeError(w, http.StatusNotFound, "продаж не знайдено")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ensure receipt record exists, then send
	receipt, err := s.store.EnsureAndSendReceipt(sale)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrReceiptSenderNotConfigured):
			writeError(w, http.StatusBadRequest, "Checkbox не налаштовано")
		default:
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}

	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"receipt.send",
		"sale",
		fmt.Sprintf("sale_id=%d receipt_id=%d status=%s", sale.ID, receipt.ID, receipt.Status),
	)
	writeJSON(w, http.StatusOK, receipt)
}


func (s *Server) handleRetryReceiptsBulk(w http.ResponseWriter, r *http.Request) {
	req := retryReceiptsBulkRequest{
		Limit: 20,
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 200 {
		req.Limit = 200
	}

	var status *string
	rawStatus := strings.TrimSpace(strings.ToLower(req.Status))
	if rawStatus != "" {
		switch rawStatus {
		case app.ReceiptStatusPending, app.ReceiptStatusFailed:
			status = &rawStatus
		default:
			writeError(w, http.StatusBadRequest, "невірний статус")
			return
		}
	}

	result, err := s.store.RetryReceiptsBulk(req.Limit, status)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrReceiptSenderNotConfigured):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"receipt.retry.bulk",
		"receipt",
		fmt.Sprintf(
			"attempted=%d succeeded=%d failed=%d limit=%d status=%s",
			result.Attempted,
			result.Succeeded,
			result.Failed,
			req.Limit,
			rawStatus,
		),
	)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.CustomerName == "" {
		writeError(w, http.StatusBadRequest, "customer name is required")
		return
	}

	var reserveUntil *time.Time
	if req.ReserveUntil != "" {
		parsed, err := time.Parse(time.RFC3339, req.ReserveUntil)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid reserveUntil format")
			return
		}
		reserveUntil = &parsed
	}
	var dueDate *time.Time
	if req.DueDate != "" {
		var parsed time.Time
		var err error
		// Accept both full RFC3339 ("2026-06-03T00:00:00Z") and plain date ("2026-06-03")
		parsed, err = time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", req.DueDate)
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueDate format")
			return
		}
		dueDate = &parsed
	}

	order, err := s.store.CreateOrder(req.CustomerName, req.Items, req.Reserve, reserveUntil, req.Currency, dueDate)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, app.ErrInsufficientStock):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"order.create",
		"customer_order",
		fmt.Sprintf("order_id=%d items=%d reserve=%t", order.ID, len(order.Items), req.Reserve),
	)
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req createCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	customer, err := s.store.CreateCustomer(app.Customer{
		Name:    req.Name,
		Phone:   req.Phone,
		Email:   req.Email,
		Comment: req.Comment,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"customer.create",
		"customer",
		fmt.Sprintf("customer_id=%d", customer.ID),
	)
	writeJSON(w, http.StatusCreated, customer)
}

func (s *Server) handleListCustomers(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	customers, err := s.store.ListCustomers(search)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження клієнтів")
		return
	}
	writeJSON(w, http.StatusOK, customers)
}

func (s *Server) handleCreateCustomerReminder(w http.ResponseWriter, r *http.Request) {
	var req createCustomerReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	var dueAt *time.Time
	if strings.TrimSpace(req.DueAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueAt format")
			return
		}
		dueAt = &parsed
	}
	reminder, err := s.store.CreateCustomerReminder(app.CustomerReminder{
		CustomerID: req.CustomerID,
		Text:       req.Text,
		DueAt:      dueAt,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"customer_reminder.create",
		"customer_reminder",
		fmt.Sprintf("reminder_id=%d customer_id=%d", reminder.ID, reminder.CustomerID),
	)
	writeJSON(w, http.StatusCreated, reminder)
}

func (s *Server) handleCompleteCustomerReminder(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid reminder id")
		return
	}
	reminder, err := s.store.CompleteCustomerReminder(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"customer_reminder.complete",
		"customer_reminder",
		fmt.Sprintf("reminder_id=%d", reminder.ID),
	)
	writeJSON(w, http.StatusOK, reminder)
}

func (s *Server) handleListCustomerReminders(w http.ResponseWriter, r *http.Request) {
	var customerID *int64
	if rawCustomerID := strings.TrimSpace(r.URL.Query().Get("customerId")); rawCustomerID != "" {
		parsed, err := strconv.ParseInt(rawCustomerID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid customerId")
			return
		}
		customerID = &parsed
	}
	var status *string
	if rawStatus := strings.TrimSpace(r.URL.Query().Get("status")); rawStatus != "" {
		status = &rawStatus
	}
	overdueOnly := r.URL.Query().Get("overdueOnly") == "true"
	items, err := s.store.ListCustomerReminders(customerID, status, overdueOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load customer reminders")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateServiceOrder(w http.ResponseWriter, r *http.Request) {
	var req createServiceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	order, err := s.store.CreateServiceOrder(app.ServiceOrder{
		CustomerID:  req.CustomerID,
		ProductID:   req.ProductID,
		Title:       req.Title,
		Description: req.Description,
		Technician:  req.Technician,
		LaborMin:    req.LaborMin,
		Price:       req.Price,
		Currency:    req.Currency,
	})
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"service_order.create",
		"service_order",
		fmt.Sprintf("service_order_id=%d customer_id=%d", order.ID, order.CustomerID),
	)
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleListServiceOrders(w http.ResponseWriter, r *http.Request) {
	var customerID *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("customerId")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid customerId")
			return
		}
		customerID = &parsed
	}
	var status *string
	if rawStatus := strings.TrimSpace(r.URL.Query().Get("status")); rawStatus != "" {
		status = &rawStatus
	}
	items, err := s.store.ListServiceOrders(customerID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження сервісних замовлень")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleUpdateServiceOrderStatus(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID сервісного замовлення")
		return
	}
	var req updateServiceOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	before, err := s.store.ServiceOrderByID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := s.store.UpdateServiceOrderStatus(id, req.Status)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"service_order.status.update",
		"service_order",
		fmt.Sprintf("service_order_id=%d status=%s", item.ID, item.Status),
	)
	if before.Status == app.ServiceOrderStatusDone && item.Status == app.ServiceOrderStatusInProgress {
		s.store.AddAuditLog(
			userFromContext(r.Context()),
			"service_order.reopen",
			"service_order",
			fmt.Sprintf("service_order_id=%d from=%s to=%s", item.ID, before.Status, item.Status),
		)
	}
	if item.Status == app.ServiceOrderStatusDone {
		if actDoc, actErr := s.store.ServiceOrderActDocument(id); actErr == nil {
			s.store.AddAuditLog(
				userFromContext(r.Context()),
				"service_order.auto_act.create",
				"document",
				fmt.Sprintf("service_order_id=%d document_id=%d status=%s", item.ID, actDoc.ID, actDoc.Status),
			)
		}
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleUpdateServiceOrder(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID сервісного замовлення")
		return
	}
	var req updateServiceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	item, err := s.store.UpdateServiceOrderDetails(id, app.ServiceOrder{
		ProductID:   req.ProductID,
		Title:       req.Title,
		Description: req.Description,
		Technician:  req.Technician,
		LaborMin:    req.LaborMin,
		Price:       req.Price,
		Currency:    req.Currency,
	})
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"service_order.update",
		"service_order",
		fmt.Sprintf("service_order_id=%d", item.ID),
	)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleAddServiceOrderPart(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID сервісного замовлення")
		return
	}
	var req addServiceOrderPartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	order, part, err := s.store.AddServiceOrderPart(id, app.ServiceOrderPart{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Price:     req.Price,
	})
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, app.ErrInsufficientStock):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"service_order.part.add",
		"service_order",
		fmt.Sprintf("service_order_id=%d part_id=%d product_id=%d qty=%d", order.ID, part.ID, part.ProductID, part.Quantity),
	)
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleCreateServiceOrderActDocument(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	serviceOrderID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || serviceOrderID <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID сервісного замовлення")
		return
	}
	var req createServiceOrderActRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	document, err := s.store.CreateServiceOrderActDocument(serviceOrderID, req.Note)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	if req.AutoPost {
		document, err = s.store.PostDocument(document.ID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document.service_order_act.create",
		"document",
		fmt.Sprintf("document_id=%d source_service_order_id=%d", document.ID, serviceOrderID),
	)
	writeJSON(w, http.StatusCreated, document)
}

func (s *Server) handleRenderServiceOrderActPDF(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	serviceOrderID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || serviceOrderID <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID сервісного замовлення")
		return
	}
	document, err := s.store.ServiceOrderActDocument(serviceOrderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	content, err := s.store.RenderDocumentPDF(document.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=service-order-%d-act-%d.pdf", serviceOrderID, document.ID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleCancelServiceOrderActDocument(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	serviceOrderID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || serviceOrderID <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID сервісного замовлення")
		return
	}
	document, err := s.store.CancelServiceOrderActDocument(serviceOrderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document.service_order_act.cancel",
		"document",
		fmt.Sprintf("document_id=%d source_service_order_id=%d", document.ID, serviceOrderID),
	)
	writeJSON(w, http.StatusOK, document)
}

func (s *Server) handleCreateServiceCategory(w http.ResponseWriter, r *http.Request) {
	var req createServiceCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	category, err := s.store.CreateServiceCategory(app.ServiceCategory{Name: req.Name})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (s *Server) handleListServiceCategories(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListServiceCategories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load service categories")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var req createServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	service, err := s.store.CreateService(app.Service{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Price:       req.Price,
		Currency:    req.Currency,
		DurationMin: req.DurationMin,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, service)
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListServices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load services")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.store.ListOrders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження замовлень")
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID замовлення")
		return
	}

	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	if err := s.store.UpdateOrderStatus(id, req.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"order.status.update",
		"customer_order",
		fmt.Sprintf("order_id=%d status=%s", id, req.Status),
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID замовлення")
		return
	}

	var req updateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.CustomerName == "" {
		writeError(w, http.StatusBadRequest, "customer name is required")
		return
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", req.DueDate)
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueDate format")
			return
		}
		dueDate = &parsed
	}

	currency := req.Currency
	if currency == "" {
		currency = "UAH"
	}

	order, err := s.store.UpdateOrder(id, req.CustomerName, currency, dueDate, req.Items)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"order.update",
		"customer_order",
		fmt.Sprintf("order_id=%d items=%d", id, len(req.Items)),
	)
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handleUpdateSupplierOrder(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID замовлення постачальника")
		return
	}

	var req updateSupplierOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	if req.SupplierID != nil {
		if err := s.store.UpdateSupplierOrderSupplier(id, *req.SupplierID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.CustomerOrderID != nil {
		if err := s.store.UpdateSupplierOrderCustomerLink(id, req.CustomerOrderID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"supplier_order.update",
		"supplier_order",
		fmt.Sprintf("supplier_order_id=%d customer_order_id=%v supplier_id=%v", id, req.CustomerOrderID, req.SupplierID),
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListReservations(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	reservations, err := s.store.ListReservations(status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load reservations")
		return
	}
	writeJSON(w, http.StatusOK, reservations)
}

func (s *Server) handleCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req createWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	warehouse, err := s.store.CreateWarehouse(app.Warehouse{
		Name:         req.Name,
		IsVirtual:    req.IsVirtual,
		LocationType: req.LocationType,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"warehouse.create",
		"warehouse",
		fmt.Sprintf("warehouse_id=%d name=%s", warehouse.ID, warehouse.Name),
	)
	writeJSON(w, http.StatusCreated, warehouse)
}

func (s *Server) handleListWarehouses(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListWarehouses()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження складів")
		return
	}
	items = filterWarehousesByAccess(items, func(warehouseID int64) bool {
		return s.hasWarehouseAccess(r, warehouseID)
	})
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListWarehouseStocks(w http.ResponseWriter, r *http.Request) {
	var warehouseID *int64
	if raw := r.URL.Query().Get("warehouseId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid warehouseId")
			return
		}
		warehouseID = &value
	}
	if warehouseID != nil && !s.hasWarehouseAccess(r, *warehouseID) {
		writeError(w, http.StatusForbidden, "warehouse access denied")
		return
	}
	var productID *int64
	if raw := r.URL.Query().Get("productId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid productId")
			return
		}
		productID = &value
	}
	items, err := s.store.ListWarehouseStocks(warehouseID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load warehouse stocks")
		return
	}
	items = filterWarehouseStocksByAccess(items, func(item app.WarehouseStock) bool {
		return s.hasWarehouseAccess(r, item.WarehouseID)
	})
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateWarehouseZone(w http.ResponseWriter, r *http.Request) {
	var req createWarehouseZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if !s.hasWarehouseAccess(r, req.WarehouseID) {
		writeError(w, http.StatusForbidden, "warehouse access denied")
		return
	}
	item, err := s.store.CreateWarehouseZone(app.WarehouseZone{
		WarehouseID: req.WarehouseID,
		Name:        req.Name,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleListWarehouseZones(w http.ResponseWriter, r *http.Request) {
	var warehouseID *int64
	if raw := r.URL.Query().Get("warehouseId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid warehouseId")
			return
		}
		warehouseID = &value
	}
	items, err := s.store.ListWarehouseZones(warehouseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load warehouse zones")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateWarehouseCell(w http.ResponseWriter, r *http.Request) {
	var req createWarehouseCellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if !s.hasWarehouseAccess(r, req.WarehouseID) {
		writeError(w, http.StatusForbidden, "warehouse access denied")
		return
	}
	item, err := s.store.CreateWarehouseCell(app.WarehouseCell{
		WarehouseID: req.WarehouseID,
		ZoneID:      req.ZoneID,
		Code:        req.Code,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleListWarehouseCells(w http.ResponseWriter, r *http.Request) {
	var zoneID *int64
	if raw := r.URL.Query().Get("zoneId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid zoneId")
			return
		}
		zoneID = &value
	}
	items, err := s.store.ListWarehouseCells(zoneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load warehouse cells")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListCellStocks(w http.ResponseWriter, r *http.Request) {
	var cellID *int64
	if raw := r.URL.Query().Get("cellId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid cellId")
			return
		}
		cellID = &value
	}
	var productID *int64
	if raw := r.URL.Query().Get("productId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid productId")
			return
		}
		productID = &value
	}
	items, err := s.store.ListCellStocks(cellID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load cell stocks")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	var req createTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if !s.hasWarehouseAccess(r, req.FromWarehouseID) || !s.hasWarehouseAccess(r, req.ToWarehouseID) {
		writeError(w, http.StatusForbidden, "warehouse access denied")
		return
	}
	transfer, err := s.store.CreateStockTransfer(app.StockTransfer{
		FromWarehouseID: req.FromWarehouseID,
		ToWarehouseID:   req.ToWarehouseID,
		Items:           req.Items,
		Note:            req.Note,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"transfer.create",
		"stock_transfer",
		fmt.Sprintf("transfer_id=%d from=%d to=%d", transfer.ID, transfer.FromWarehouseID, transfer.ToWarehouseID),
	)
	writeJSON(w, http.StatusCreated, transfer)
}

func (s *Server) handleCreateCellTransfer(w http.ResponseWriter, r *http.Request) {
	var req createCellTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	transfer, err := s.store.CreateCellTransfer(req.FromCellID, req.ToCellID, req.Items, req.Note)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"transfer.cell.create",
		"stock_transfer",
		fmt.Sprintf("transfer_id=%d from_cell=%d to_cell=%d", transfer.ID, req.FromCellID, req.ToCellID),
	)
	writeJSON(w, http.StatusCreated, transfer)
}

func (s *Server) handleCreateCellTransferFIFO(w http.ResponseWriter, r *http.Request) {
	var req createCellTransferFIFORequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	transfer, err := s.store.CreateCellTransferFIFO(req.ToCellID, req.Items, req.Note)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"transfer.cell.fifo.create",
		"stock_transfer",
		fmt.Sprintf("transfer_id=%d to_cell=%d", transfer.ID, req.ToCellID),
	)
	writeJSON(w, http.StatusCreated, transfer)
}

func (s *Server) handleCreateInventory(w http.ResponseWriter, r *http.Request) {
	var req createInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if !s.hasWarehouseAccess(r, req.WarehouseID) {
		writeError(w, http.StatusForbidden, "warehouse access denied")
		return
	}
	inventory, err := s.store.CreateInventory(app.Inventory{
		WarehouseID: req.WarehouseID,
		Items:       req.Items,
		Note:        req.Note,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"inventory.create",
		"inventory",
		fmt.Sprintf("inventory_id=%d warehouse_id=%d", inventory.ID, inventory.WarehouseID),
	)
	writeJSON(w, http.StatusCreated, inventory)
}

func (s *Server) handleListInventories(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListInventories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load inventories")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleApplyInventory(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid inventory id")
		return
	}
	inventory, err := s.store.ApplyInventory(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"inventory.apply",
		"inventory",
		fmt.Sprintf("inventory_id=%d", inventory.ID),
	)
	writeJSON(w, http.StatusOK, inventory)
}

func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	var req createDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.WarehouseID != nil && !s.hasWarehouseAccess(r, *req.WarehouseID) {
		writeError(w, http.StatusForbidden, "warehouse access denied")
		return
	}
	if req.CashboxID != nil && !s.hasCashboxAccess(r, *req.CashboxID) {
		writeError(w, http.StatusForbidden, "cashbox access denied")
		return
	}

	document, err := s.store.CreateDocument(app.Document{
		Type:        req.Type,
		WarehouseID: req.WarehouseID,
		CashboxID:   req.CashboxID,
		Currency:    req.Currency,
		Total:       req.Total,
		Items:       req.Items,
		Note:        req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document.create",
		"document",
		fmt.Sprintf("document_id=%d type=%s number=%s", document.ID, document.Type, document.Number),
	)
	writeJSON(w, http.StatusCreated, document)
}

func (s *Server) handleImportDocumentCSV(w http.ResponseWriter, r *http.Request) {
	var req importDocumentCSVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.WarehouseID != nil && !s.hasWarehouseAccess(r, *req.WarehouseID) {
		writeError(w, http.StatusForbidden, "warehouse access denied")
		return
	}
	if req.CashboxID != nil && !s.hasCashboxAccess(r, *req.CashboxID) {
		writeError(w, http.StatusForbidden, "cashbox access denied")
		return
	}
	result, err := s.store.ImportDocumentCSV(app.Document{
		Type:        req.Type,
		WarehouseID: req.WarehouseID,
		CashboxID:   req.CashboxID,
		Currency:    req.Currency,
		Total:       req.Total,
		Note:        req.Note,
	}, req.CSV)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document.import_csv",
		"document",
		fmt.Sprintf("document_id=%d imported=%d skipped=%d", result.Document.ID, result.Imported, result.Skipped),
	)
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleCustomerReturnAvailability(w http.ResponseWriter, r *http.Request) {
	rawSaleID := chi.URLParam(r, "saleId")
	saleID, err := strconv.ParseInt(rawSaleID, 10, 64)
	if err != nil || saleID <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID продажу")
		return
	}
	availability, err := s.store.CustomerReturnAvailability(saleID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, availability)
}

func (s *Server) handleCreateCustomerReturnDocument(w http.ResponseWriter, r *http.Request) {
	var req createCustomerReturnDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	document, err := s.store.CreateReturnFromCustomerDocument(
		req.SaleID,
		req.WarehouseID,
		req.Currency,
		req.Items,
		req.Note,
	)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document.return_from_customer.create",
		"document",
		fmt.Sprintf("document_id=%d source_sale_id=%d", document.ID, req.SaleID),
	)
	writeJSON(w, http.StatusCreated, document)
}

func (s *Server) handleSupplierReturnAvailability(w http.ResponseWriter, r *http.Request) {
	rawPurchaseID := chi.URLParam(r, "purchaseId")
	purchaseID, err := strconv.ParseInt(rawPurchaseID, 10, 64)
	if err != nil || purchaseID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid purchase id")
		return
	}
	availability, err := s.store.SupplierReturnAvailability(purchaseID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, availability)
}

func (s *Server) handleCreateSupplierReturnDocument(w http.ResponseWriter, r *http.Request) {
	var req createSupplierReturnDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	document, err := s.store.CreateReturnToSupplierDocument(
		req.PurchaseID,
		req.WarehouseID,
		req.Currency,
		req.Items,
		req.Note,
	)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document.return_to_supplier.create",
		"document",
		fmt.Sprintf("document_id=%d source_purchase_id=%d", document.ID, req.PurchaseID),
	)
	writeJSON(w, http.StatusCreated, document)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	var docType *string
	var status *string
	if value := strings.TrimSpace(r.URL.Query().Get("type")); value != "" {
		docType = &value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("status")); value != "" {
		status = &value
	}
	documents, err := s.store.ListDocuments(docType, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження документів")
		return
	}
	writeJSON(w, http.StatusOK, documents)
}

func (s *Server) handlePostDocument(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}
	document, err := s.store.PostDocument(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document.post",
		"document",
		fmt.Sprintf("document_id=%d number=%s", document.ID, document.Number),
	)
	writeJSON(w, http.StatusOK, document)
}

func (s *Server) handleRenderDocumentPDF(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}
	content, err := s.store.RenderDocumentPDF(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=document-%d.pdf", id))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleListDocumentTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.store.ListDocumentTemplates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load document templates")
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) handleUpsertDocumentTemplate(w http.ResponseWriter, r *http.Request) {
	var req upsertDocumentTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if _, err := s.store.ValidateDocumentTemplate(req.Code, req.Body, true); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	template, err := s.store.UpsertDocumentTemplate(app.DocumentTemplate{
		Code:     req.Code,
		Name:     req.Name,
		Body:     req.Body,
		IsActive: req.IsActive,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"document_template.upsert",
		"document_template",
		fmt.Sprintf("template_id=%d code=%s", template.ID, template.Code),
	)
	writeJSON(w, http.StatusOK, template)
}

func (s *Server) handleValidateDocumentTemplate(w http.ResponseWriter, r *http.Request) {
	var req previewDocumentTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	validation, err := s.store.ValidateDocumentTemplate(req.Code, req.Body, req.Strict)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, validation)
}

func (s *Server) handlePreviewDocumentTemplate(w http.ResponseWriter, r *http.Request) {
	var req previewDocumentTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	preview, err := s.store.PreviewDocumentTemplate(req.Code, req.Body, req.DocumentID, req.Strict)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleDocumentTemplatePlaceholders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.DocumentTemplatePlaceholders())
}

func (s *Server) handleCreateSupplier(w http.ResponseWriter, r *http.Request) {
	var req createSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	supplier, err := s.store.CreateSupplier(app.Supplier{
		Name:     req.Name,
		Contact:  req.Contact,
		Phone:    req.Phone,
		Email:    req.Email,
		Comments: req.Comments,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"supplier.create",
		"supplier",
		fmt.Sprintf("supplier_id=%d name=%s", supplier.ID, supplier.Name),
	)
	writeJSON(w, http.StatusCreated, supplier)
}

func (s *Server) handleListSuppliers(w http.ResponseWriter, r *http.Request) {
	suppliers, err := s.store.ListSuppliers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження постачальників")
		return
	}
	writeJSON(w, http.StatusOK, suppliers)
}

func (s *Server) handleCreateSupplierOrder(w http.ResponseWriter, r *http.Request) {
	var req createSupplierOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	order, err := s.store.CreateSupplierOrder(req.SupplierID, req.Items, req.Currency)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"supplier_order.create",
		"supplier_order",
		fmt.Sprintf("supplier_order_id=%d supplier_id=%d items=%d", order.ID, order.SupplierID, len(order.Items)),
	)
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleListSupplierOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.store.ListSupplierOrders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження замовлень постачальника")
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) handleReceiveSupplierOrderByLines(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid supplier order id")
		return
	}

	var req receiveSupplierOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	purchase, err := s.store.ReceiveSupplierOrderByLines(id, req.Currency, req.Lines, req.Note)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"supplier_order.receive",
		"supplier_order",
		fmt.Sprintf("supplier_order_id=%d purchase_id=%d lines=%d", id, purchase.ID, len(req.Lines)),
	)
	writeJSON(w, http.StatusCreated, purchase)
}

func (s *Server) handleSupplierOrdersPending(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSupplierOrdersPending()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load pending supplier orders")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleSupplierOrderRecommendations(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "невірне значення ліміту")
			return
		}
		limit = parsed
	}
	items, err := s.store.ListPurchaseRecommendations(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load recommendations")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateSupplierOrderFromRecommendations(w http.ResponseWriter, r *http.Request) {
	var req createSupplierOrderFromRecommendationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	order, err := s.store.CreateSupplierOrderFromRecommendations(req.SupplierID, req.Currency, req.Items)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"supplier_order.recommendation.create",
		"supplier_order",
		fmt.Sprintf("supplier_order_id=%d supplier_id=%d items=%d", order.ID, order.SupplierID, len(order.Items)),
	)
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleSupplierOrderRecommendationsGrouped(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "невірне значення ліміту")
			return
		}
		limit = parsed
	}
	items, err := s.store.ListPurchaseRecommendationsGrouped(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load grouped recommendations")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateSupplierOrdersBulkFromRecommendations(w http.ResponseWriter, r *http.Request) {
	var req createSupplierOrdersBulkFromRecommendationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	orders, err := s.store.CreateSupplierOrdersBulkFromRecommendations(req.Orders)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"supplier_order.recommendation.bulk_create",
		"supplier_order",
		fmt.Sprintf("created_orders=%d", len(orders)),
	)
	writeJSON(w, http.StatusCreated, orders)
}

func (s *Server) handleUpdateSupplierOrderStatus(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid supplier order id")
		return
	}

	var req updateSupplierOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	if err := s.store.UpdateSupplierOrderStatus(id, req.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"supplier_order.status.update",
		"supplier_order",
		fmt.Sprintf("supplier_order_id=%d status=%s", id, req.Status),
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreatePurchase(w http.ResponseWriter, r *http.Request) {
	var req createPurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	purchase, err := s.store.CreatePurchase(app.Purchase{
		SupplierID:      req.SupplierID,
		SupplierOrderID: req.SupplierOrderID,
		Currency:        req.Currency,
		Items:           req.Items,
		Note:            req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"purchase.create",
		"purchase",
		fmt.Sprintf("purchase_id=%d supplier_id=%d items=%d", purchase.ID, purchase.SupplierID, len(purchase.Items)),
	)
	writeJSON(w, http.StatusCreated, purchase)
}

func (s *Server) handleImportPurchaseCSV(w http.ResponseWriter, r *http.Request) {
	var req importPurchasesCSVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	result, err := s.store.ImportPurchaseCSV(app.Purchase{
		SupplierID:      req.SupplierID,
		SupplierOrderID: req.SupplierOrderID,
		Currency:        req.Currency,
		Note:            req.Note,
	}, req.CSV)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"purchase.import_csv",
		"purchase",
		fmt.Sprintf("purchase_id=%d imported=%d skipped=%d", result.Purchase.ID, result.Imported, result.Skipped),
	)
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleListPurchases(w http.ResponseWriter, r *http.Request) {
	var supplierOrderID *int64
	if rawOrderID := r.URL.Query().Get("supplierOrderId"); rawOrderID != "" {
		parsed, err := strconv.ParseInt(rawOrderID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid supplierOrderId")
			return
		}
		supplierOrderID = &parsed
	}

	purchases, err := s.store.ListPurchases(supplierOrderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження закупівель")
		return
	}
	writeJSON(w, http.StatusOK, purchases)
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	var req createPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.CashboxID > 0 && !s.hasCashboxAccess(r, req.CashboxID) {
		writeError(w, http.StatusForbidden, "cashbox access denied")
		return
	}
	payment, err := s.store.CreatePayment(app.Payment{
		OrderID:         req.OrderID,
		SaleID:          req.SaleID,
		ServiceOrderID:  req.ServiceOrderID,
		SupplierOrderID: req.SupplierOrderID,
		PurchaseID:      req.PurchaseID,
		CashboxID:       req.CashboxID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Method:          req.Method,
		Note:            req.Note,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"payment.create",
		"payment",
		fmt.Sprintf("payment_id=%d amount=%.2f", payment.ID, payment.Amount),
	)
	writeJSON(w, http.StatusCreated, payment)
}

func (s *Server) handleListPayments(w http.ResponseWriter, r *http.Request) {
	var orderID *int64
	var saleID *int64
	var serviceOrderID *int64
	var supplierOrderID *int64

	if rawOrderID := r.URL.Query().Get("orderId"); rawOrderID != "" {
		parsed, err := strconv.ParseInt(rawOrderID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid orderId")
			return
		}
		orderID = &parsed
	}

	if rawSaleID := r.URL.Query().Get("saleId"); rawSaleID != "" {
		parsed, err := strconv.ParseInt(rawSaleID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid saleId")
			return
		}
		saleID = &parsed
	}
	if rawServiceOrderID := r.URL.Query().Get("serviceOrderId"); rawServiceOrderID != "" {
		parsed, err := strconv.ParseInt(rawServiceOrderID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid serviceOrderId")
			return
		}
		serviceOrderID = &parsed
	}
	if rawSupplierOrderID := r.URL.Query().Get("supplierOrderId"); rawSupplierOrderID != "" {
		parsed, err := strconv.ParseInt(rawSupplierOrderID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid supplierOrderId")
			return
		}
		supplierOrderID = &parsed
	}

	payments, err := s.store.ListPayments(orderID, saleID, serviceOrderID, supplierOrderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження платежів")
		return
	}
	writeJSON(w, http.StatusOK, payments)
}

func (s *Server) handleListDebts(w http.ResponseWriter, r *http.Request) {
	debts, err := s.store.ListDebts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження боргів")
		return
	}
	writeJSON(w, http.StatusOK, debts)
}

func (s *Server) handleListOverdueDebts(w http.ResponseWriter, r *http.Request) {
	asOf := time.Now().UTC()
	if rawAsOf := r.URL.Query().Get("asOf"); rawAsOf != "" {
		parsed, err := time.Parse(time.RFC3339, rawAsOf)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid asOf format")
			return
		}
		asOf = parsed
	}

	debts, err := s.store.ListOverdueDebts(asOf)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load overdue debts")
		return
	}
	writeJSON(w, http.StatusOK, debts)
}

func (s *Server) handleDebtPaymentHistory(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entityType")
	rawEntityID := r.URL.Query().Get("entityId")
	if entityType == "" || rawEntityID == "" {
		writeError(w, http.StatusBadRequest, "entityType and entityId are required")
		return
	}
	entityID, err := strconv.ParseInt(rawEntityID, 10, 64)
	if err != nil || entityID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid entityId")
		return
	}

	history, err := s.store.DebtPaymentHistory(entityType, entityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (s *Server) handleListNotificationTemplates(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	templates, err := s.store.ListNotificationTemplates(code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load notification templates")
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) handleUpsertNotificationTemplate(w http.ResponseWriter, r *http.Request) {
	var req upsertNotificationTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	template, err := s.store.UpsertNotificationTemplate(app.NotificationTemplate{
		Code:     req.Code,
		Channel:  req.Channel,
		Subject:  req.Subject,
		Body:     req.Body,
		IsActive: req.IsActive,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"notification_template.upsert",
		"notification_template",
		fmt.Sprintf("template_id=%d code=%s channel=%s", template.ID, template.Code, template.Channel),
	)
	writeJSON(w, http.StatusOK, template)
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "невірне значення ліміту")
			return
		}
		limit = parsed
	}
	notifications, err := s.store.ListNotifications(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження повідомлень")
		return
	}
	writeJSON(w, http.StatusOK, notifications)
}

func (s *Server) handleQuickMessage(w http.ResponseWriter, r *http.Request) {
	var req quickMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	notification, err := s.store.SendQuickMessage(app.QuickMessageRequest{
		Channel:    req.Channel,
		Recipient:  req.Recipient,
		Sender:     req.Sender,
		Subject:    req.Subject,
		Body:       req.Body,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(userFromContext(r.Context()), "notification.quick_message", "notification", fmt.Sprintf("notification_id=%d channel=%s", notification.ID, notification.Channel))
	writeJSON(w, http.StatusCreated, notification)
}

func (s *Server) handleGetNotificationConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.store.GetNotificationConfig()
	// Mask passwords before sending to frontend
	if cfg.SMTPPass != "" {
		cfg.SMTPPass = "••••••••"
	}
	if cfg.ViberToken != "" {
		cfg.ViberToken = "••••••••"
	}
	if cfg.TelegramToken != "" {
		cfg.TelegramToken = "••••••••"
	}
	if cfg.SMSGatewayToken != "" {
		cfg.SMSGatewayToken = "••••••••"
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleSaveNotificationConfig(w http.ResponseWriter, r *http.Request) {
	var cfg app.NotificationConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	// If a masked value is sent back, keep the existing one
	existing := s.store.GetNotificationConfig()
	if cfg.SMTPPass == "••••••••" {
		cfg.SMTPPass = existing.SMTPPass
	}
	if cfg.ViberToken == "••••••••" {
		cfg.ViberToken = existing.ViberToken
	}
	if cfg.TelegramToken == "••••••••" {
		cfg.TelegramToken = existing.TelegramToken
	}
	if cfg.SMSGatewayToken == "••••••••" {
		cfg.SMSGatewayToken = existing.SMSGatewayToken
	}
	if err := s.store.SaveNotificationConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "помилка збереження налаштувань")
		return
	}
	s.store.AddAuditLog(userFromContext(r.Context()), "settings.notification.save", "settings", "notification config updated")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleEnqueueOverdueReminderJob(w http.ResponseWriter, r *http.Request) {
	var req enqueueBackgroundJobRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	asOf := time.Now().UTC()
	if req.AsOf != "" {
		parsed, err := time.Parse(time.RFC3339, req.AsOf)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid asOf format")
			return
		}
		asOf = parsed
	}

	job, err := s.store.EnqueueOverdueReminderJob(asOf)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка постановки задачі в чергу")
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"background_job.enqueue",
		"background_job",
		fmt.Sprintf("job_id=%d type=%s", job.ID, job.JobType),
	)
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleEnqueueReservationExpiryJob(w http.ResponseWriter, r *http.Request) {
	var req enqueueBackgroundJobRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	asOf := time.Now().UTC()
	if req.AsOf != "" {
		parsed, err := time.Parse(time.RFC3339, req.AsOf)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid asOf format")
			return
		}
		asOf = parsed
	}

	job, err := s.store.EnqueueReservationExpiryJob(asOf)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot enqueue reservation expiry job")
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"background_job.enqueue.reservation_expiry",
		"background_job",
		fmt.Sprintf("job_id=%d type=%s", job.ID, job.JobType),
	)
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleEnqueueReceiptRetryJob(w http.ResponseWriter, r *http.Request) {
	req := enqueueReceiptRetryJobRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	job, err := s.store.EnqueueReceiptRetryJob(req.Status, req.Limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"background_job.enqueue.receipt_retry",
		"background_job",
		fmt.Sprintf("job_id=%d type=%s payload=%s", job.ID, job.JobType, job.Payload),
	)
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleRunBackgroundJobs(w http.ResponseWriter, r *http.Request) {
	processed, err := s.store.RunDueBackgroundJobs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot run background jobs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"processed": processed,
		"count":     len(processed),
	})
}

func (s *Server) handleListBackgroundJobs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "невірне значення ліміту")
			return
		}
		limit = parsed
	}
	jobs, err := s.store.ListBackgroundJobs(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження фонових задач")
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleCreateCashbox(w http.ResponseWriter, r *http.Request) {
	var req createCashboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	cashbox, err := s.store.CreateCashbox(app.Cashbox{
		Name:     req.Name,
		Type:     req.Type,
		Currency: req.Currency,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"cashbox.create",
		"cashbox",
		fmt.Sprintf("cashbox_id=%d name=%s", cashbox.ID, cashbox.Name),
	)
	writeJSON(w, http.StatusCreated, cashbox)
}

func (s *Server) handleListCashboxes(w http.ResponseWriter, r *http.Request) {
	cashboxes, err := s.store.ListCashboxes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження кас")
		return
	}
	cashboxes = filterCashboxesByAccess(cashboxes, func(cashboxID int64) bool {
		return s.hasCashboxAccess(r, cashboxID)
	})
	writeJSON(w, http.StatusOK, cashboxes)
}

func (s *Server) handleCreateCashOperation(w http.ResponseWriter, r *http.Request) {
	var req createCashOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if !s.hasCashboxAccess(r, req.CashboxID) {
		writeError(w, http.StatusForbidden, "cashbox access denied")
		return
	}

	operation, err := s.store.CreateCashOperation(app.CashOperation{
		CashboxID:   req.CashboxID,
		Type:        req.Type,
		Amount:      req.Amount,
		Method:      req.Method,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"cash.operation.create",
		"cash_operation",
		fmt.Sprintf("cash_operation_id=%d cashbox_id=%d amount=%.2f", operation.ID, operation.CashboxID, operation.Amount),
	)
	writeJSON(w, http.StatusCreated, operation)
}

func (s *Server) handleListCashOperations(w http.ResponseWriter, r *http.Request) {
	var cashboxID *int64
	if rawCashboxID := r.URL.Query().Get("cashboxId"); rawCashboxID != "" {
		parsed, err := strconv.ParseInt(rawCashboxID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid cashboxId")
			return
		}
		cashboxID = &parsed
	}
	if cashboxID != nil && !s.hasCashboxAccess(r, *cashboxID) {
		writeError(w, http.StatusForbidden, "cashbox access denied")
		return
	}

	operations, err := s.store.ListCashOperations(cashboxID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження касових операцій")
		return
	}
	writeJSON(w, http.StatusOK, operations)
}

func (s *Server) handleOpenCashShift(w http.ResponseWriter, r *http.Request) {
	var req openCashShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if !s.hasCashboxAccess(r, req.CashboxID) {
		writeError(w, http.StatusForbidden, "cashbox access denied")
		return
	}
	shift, err := s.store.OpenCashShift(req.CashboxID, userFromContext(r.Context()), req.Note)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"cash_shift.open",
		"cash_shift",
		fmt.Sprintf("shift_id=%d cashbox_id=%d", shift.ID, shift.CashboxID),
	)
	writeJSON(w, http.StatusCreated, shift)
}

func (s *Server) handleCloseCashShift(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	shiftID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || shiftID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid shift id")
		return
	}
	var req closeCashShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	shift, err := s.store.CloseCashShift(shiftID, userFromContext(r.Context()), req.Note)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"cash_shift.close",
		"cash_shift",
		fmt.Sprintf("shift_id=%d cashbox_id=%d", shift.ID, shift.CashboxID),
	)
	writeJSON(w, http.StatusOK, shift)
}

func (s *Server) handleListCashShifts(w http.ResponseWriter, r *http.Request) {
	var cashboxID *int64
	if rawCashboxID := strings.TrimSpace(r.URL.Query().Get("cashboxId")); rawCashboxID != "" {
		parsed, err := strconv.ParseInt(rawCashboxID, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid cashboxId")
			return
		}
		cashboxID = &parsed
	}
	if cashboxID != nil && !s.hasCashboxAccess(r, *cashboxID) {
		writeError(w, http.StatusForbidden, "cashbox access denied")
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	shifts, err := s.store.ListCashShifts(cashboxID, status)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, shifts)
}

func (s *Server) handleListExchangeRates(w http.ResponseWriter, r *http.Request) {
	rates, err := s.store.ListExchangeRates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження курсів валют")
		return
	}
	writeJSON(w, http.StatusOK, rates)
}

func (s *Server) handleUpsertExchangeRate(w http.ResponseWriter, r *http.Request) {
	var req upsertExchangeRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	rate, err := s.store.UpsertExchangeRate(req.Currency, req.RateToUAH)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"exchange_rate.upsert",
		"exchange_rate",
		fmt.Sprintf("currency=%s rate_to_uah=%.4f", rate.Currency, rate.RateToUAH),
	)
	writeJSON(w, http.StatusOK, rate)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.store.AnalyticsSummary()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot build summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleProfitability(w http.ResponseWriter, r *http.Request) {
	report, err := s.store.ProfitabilityReport()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot build profitability report")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleCategoryAnalytics(w http.ResponseWriter, r *http.Request) {
	report, err := s.store.CategoryAnalytics()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot build category analytics")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleChangeHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "невірне значення ліміту")
			return
		}
		limit = parsed
	}
	history, err := s.store.ChangeHistory(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot load change history")
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "невірне значення ліміту")
			return
		}
		limit = parsed
	}

	logs, err := s.store.ListAuditLogs(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження журналу аудиту")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження користувачів")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if req.Username == "" || req.Password == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "username, password and role are required")
		return
	}

	user, err := s.store.CreateUser(app.User{
		Username: req.Username,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"user.create",
		"user",
		fmt.Sprintf("username=%s role=%s", user.Username, user.Role),
	)

	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.store.ListRolePermissions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка завантаження ролей")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"roles":       roles,
		"permissions": app.AllPermissions(),
	})
}

func (s *Server) handleUpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if role == "" {
		writeError(w, http.StatusBadRequest, "role is required")
		return
	}

	var req updateRolePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}

	if err := s.store.UpdateRolePermissions(role, req.Permissions); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"role.permissions.update",
		"role",
		fmt.Sprintf("role=%s permissions=%d", role, len(req.Permissions)),
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func filterWarehousesByAccess(items []app.Warehouse, allowed func(int64) bool) []app.Warehouse {
	result := make([]app.Warehouse, 0, len(items))
	for _, item := range items {
		if allowed(item.ID) {
			result = append(result, item)
		}
	}
	return result
}

func filterWarehouseStocksByAccess(
	items []app.WarehouseStock,
	allowed func(app.WarehouseStock) bool,
) []app.WarehouseStock {
	result := make([]app.WarehouseStock, 0, len(items))
	for _, item := range items {
		if allowed(item) {
			result = append(result, item)
		}
	}
	return result
}

func filterCashboxesByAccess(items []app.Cashbox, allowed func(int64) bool) []app.Cashbox {
	result := make([]app.Cashbox, 0, len(items))
	for _, item := range items {
		if allowed(item.ID) {
			result = append(result, item)
		}
	}
	return result
}

func filterStockMovementsByAccess(
	items []app.StockMovement,
	allowed func(app.StockMovement) bool,
) []app.StockMovement {
	result := make([]app.StockMovement, 0, len(items))
	for _, item := range items {
		if allowed(item) {
			result = append(result, item)
		}
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error": msg,
	})
}

// buildLabelsPDF generates a minimal raw PDF containing price-tag labels.
// Each label shows: product name, SKU/barcode, and retail price.
// format: "small" = 58x40mm (2 per row), "large" = 100x60mm (1 per row)
func buildLabelsPDF(products []app.Product, format string) []byte {
	var lines []string
	lines = append(lines, "Цінники / Price Labels")
	lines = append(lines, fmt.Sprintf("Формат: %s | Кількість: %d", format, len(products)))
	lines = append(lines, "")
	for i, p := range products {
		barcode := p.Barcode
		if barcode == "" {
			barcode = p.SKU
		}
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, p.Name))
		lines = append(lines, fmt.Sprintf("   Код: %s", barcode))
		lines = append(lines, fmt.Sprintf("   Ціна: %.2f %s", p.RetailPrice, p.Currency))
		lines = append(lines, "")
	}

	body := strings.Join(lines, "\n")
	pdfContent := fmt.Sprintf(
		"%%PDF-1.1\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n"+
			"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n"+
			"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 595 842]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n"+
			"4 0 obj<</Length %d>>stream\nBT /F1 10 Tf 40 800 Td 14 TL (%s) Tj ET\nendstream\nendobj\n"+
			"5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n"+
			"xref\n0 6\n0000000000 65535 f \ntrailer<</Root 1 0 R/Size 6>>\nstartxref\n0\n%%%%EOF\n",
		len(body)+32, sanitizeLabelText(body),
	)
	return []byte(pdfContent)
}

// buildBarcodeSVG generates an SVG visual barcode representation using simple bars.
func buildBarcodeSVG(code, name string) string {
	// Generate a deterministic bar pattern from the code characters
	barWidth := 2
	barHeight := 60
	quietZone := 10
	x := quietZone

	var bars strings.Builder
	for i, ch := range code {
		w := barWidth
		if i%3 == 0 {
			w = barWidth + 1
		}
		fill := "#000000"
		if int(ch)%2 == 0 {
			fill = "#ffffff"
		}
		bars.WriteString(fmt.Sprintf(`<rect x="%d" y="10" width="%d" height="%d" fill="%s"/>`, x, w, barHeight, fill))
		x += w + 1
	}
	totalWidth := x + quietZone

	safeName := strings.ReplaceAll(name, "<", "&lt;")
	safeName = strings.ReplaceAll(safeName, ">", "&gt;")
	safeCode := strings.ReplaceAll(code, "<", "&lt;")

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="100" viewBox="0 0 %d 100">
  <rect width="%d" height="100" fill="white"/>
  %s
  <text x="%d" y="85" font-family="monospace" font-size="9" text-anchor="middle" fill="#000">%s</text>
  <text x="%d" y="97" font-family="sans-serif" font-size="7" text-anchor="middle" fill="#555">%s</text>
</svg>`, totalWidth, totalWidth, totalWidth, bars.String(),
		totalWidth/2, safeCode,
		totalWidth/2, safeName)
}

func sanitizeLabelText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 {
			if r == '(' || r == ')' || r == '\\' {
				b.WriteRune('\\')
			}
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

type salesGroupedRow struct {
	Period   string  `json:"period"`
	SalesQty int     `json:"salesQty"`
	Revenue  float64 `json:"revenue"`
	Profit   float64 `json:"profit"`
}

func (s *Server) handleSalesGrouped(w http.ResponseWriter, r *http.Request) {
	groupBy := r.URL.Query().Get("groupBy") // "day" | "month"
	if groupBy == "" {
		groupBy = "month"
	}
	rows, err := s.store.SalesGrouped(groupBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// handleExportVATCsv exports all posted documents with VAT breakdown as CSV.
func (s *Server) handleExportVATCsv(w http.ResponseWriter, r *http.Request) {
	posted := "posted"
	docs, err := s.store.ListDocuments(nil, &posted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var buf strings.Builder
	buf.WriteString("Номер,Тип,Статус,Дата,Позицій,Сума без ПДВ,ПДВ,Сума з ПДВ,Валюта\n")
	for _, doc := range docs {
		var subtotal, vatTotal float64
		for _, item := range doc.Items {
			line := float64(item.Quantity) * item.Price
			subtotal += line
			vatTotal += line * item.VATPercent / 100
		}
		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d,%.2f,%.2f,%.2f,%s\n",
			doc.Number, doc.Type, doc.Status,
			doc.CreatedAt.Format("2006-01-02"),
			len(doc.Items),
			subtotal, vatTotal, subtotal+vatTotal,
			doc.Currency,
		))
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="vat-export.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("\xef\xbb\xbf")) // UTF-8 BOM for Excel
	_, _ = w.Write([]byte(buf.String()))
}

// handleSupplierDeficitPortal returns a public (no auth) JSON list of products
// below minimum stock, grouped by supplier. The token in the URL is a static
// read-only key set via the SUPPLIER_PORTAL_TOKEN env variable.
func (s *Server) handleSupplierDeficitPortal(w http.ResponseWriter, r *http.Request) {
	expected := strings.TrimSpace(os.Getenv("SUPPLIER_PORTAL_TOKEN"))
	if expected == "" {
		writeError(w, http.StatusServiceUnavailable, "портал постачальника не налаштовано (SUPPLIER_PORTAL_TOKEN не задано)")
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		token = chi.URLParam(r, "token")
	}
	if token != expected {
		writeError(w, http.StatusUnauthorized, "недійсний токен")
		return
	}

	recs, err := s.store.ListPurchaseRecommendationsGrouped(200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"generatedAt":  time.Now().UTC(),
		"deficitGroups": recs,
	})
}

// handleExportVATXlsx exports all posted documents with full VAT breakdown as XLSX.
func (s *Server) handleExportVATXlsx(w http.ResponseWriter, r *http.Request) {
	posted := "posted"
	docs, err := s.store.ListDocuments(nil, &posted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	f := excelize.NewFile()
	sheet := "ПДВ"
	f.SetSheetName("Sheet1", sheet)

	// Header row
	headers := []string{
		"Номер", "Тип", "Статус", "Дата", "Позицій",
		"Сума без ПДВ", "ПДВ", "Сума з ПДВ", "Валюта",
	}
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2D8A58"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
	}
	f.SetColWidth(sheet, "A", "I", 18)

	// Data rows
	numStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4, // #,##0.00
	})
	for row, doc := range docs {
		var subtotal, vatTotal float64
		for _, item := range doc.Items {
			line := float64(item.Quantity) * item.Price
			subtotal += line
			vatTotal += line * item.VATPercent / 100
		}
		values := []interface{}{
			doc.Number,
			doc.Type,
			doc.Status,
			doc.CreatedAt.Format("2006-01-02"),
			len(doc.Items),
			subtotal,
			vatTotal,
			subtotal + vatTotal,
			doc.Currency,
		}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row+2)
			f.SetCellValue(sheet, cell, v)
			if col >= 5 && col <= 7 {
				f.SetCellStyle(sheet, cell, cell, numStyle)
			}
		}
	}

	// Totals row
	totalsRow := len(docs) + 2
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1A3A2E"}, Pattern: 1},
		NumFmt: 4,
	})
	f.SetCellValue(sheet, fmt.Sprintf("A%d", totalsRow), "РАЗОМ")
	for _, col := range []int{6, 7, 8} {
		colName, _ := excelize.ColumnNumberToName(col)
		cell := fmt.Sprintf("%s%d", colName, totalsRow)
		startCell := fmt.Sprintf("%s2", colName)
		endCell := fmt.Sprintf("%s%d", colName, totalsRow-1)
		f.SetCellFormula(sheet, cell, fmt.Sprintf("SUM(%s:%s)", startCell, endCell))
		f.SetCellStyle(sheet, cell, cell, totalStyle)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка генерації XLSX")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="vat-export.xlsx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// ── Order chain ───────────────────────────────────────────────────────────

func (s *Server) handleGetOrderChain(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID замовлення")
		return
	}
	chain, err := s.store.GetOrderChain(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chain)
}

// ── Counterparties ────────────────────────────────────────────────────────

type createCounterpartyRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Comment    string `json:"comment"`
	IsCustomer bool   `json:"isCustomer"`
	IsSupplier bool   `json:"isSupplier"`
}

type updateCounterpartyRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Comment    string `json:"comment"`
	IsCustomer bool   `json:"isCustomer"`
	IsSupplier bool   `json:"isSupplier"`
}

func (s *Server) handleListCounterparties(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListCounterparties()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateCounterparty(w http.ResponseWriter, r *http.Request) {
	var req createCounterpartyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	if !req.IsCustomer && !req.IsSupplier {
		writeError(w, http.StatusBadRequest, "оберіть хоча б одну роль: покупець або постачальник")
		return
	}
	cp, err := s.store.CreateCounterparty(app.Counterparty{
		Name:       req.Name,
		Phone:      req.Phone,
		Email:      req.Email,
		Comment:    req.Comment,
		IsCustomer: req.IsCustomer,
		IsSupplier: req.IsSupplier,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"counterparty.create",
		"counterparty",
		fmt.Sprintf("id=%d name=%s", cp.ID, cp.Name),
	)
	writeJSON(w, http.StatusCreated, cp)
}

func (s *Server) handleUpdateCounterparty(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID контрагента")
		return
	}
	var req updateCounterpartyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "невірний формат запиту")
		return
	}
	cp, err := s.store.UpdateCounterparty(id, app.Counterparty{
		Name:       req.Name,
		Phone:      req.Phone,
		Email:      req.Email,
		Comment:    req.Comment,
		IsCustomer: req.IsCustomer,
		IsSupplier: req.IsSupplier,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"counterparty.update",
		"counterparty",
		fmt.Sprintf("id=%d name=%s", cp.ID, cp.Name),
	)
	writeJSON(w, http.StatusOK, cp)
}

// ── Document Registry ─────────────────────────────────────────────────────

func (s *Server) handleSearchDocuments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Optional type filter: ?types=customer_order,sale,purchase
	var docTypes []string
	if rawTypes := r.URL.Query().Get("types"); rawTypes != "" {
		for _, t := range strings.Split(rawTypes, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				docTypes = append(docTypes, t)
			}
		}
	}

	results, err := s.store.SearchDocuments(query, docTypes, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// ── Operation Journal ─────────────────────────────────────────────────────

func (s *Server) handleJournal(w http.ResponseWriter, r *http.Request) {
	q      := r.URL.Query()
	search   := q.Get("search")
	user     := q.Get("user")
	entity   := q.Get("entity")
	dateFrom := q.Get("from")
	dateTo   := q.Get("to")
	limit    := 200
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}

	logs, err := s.store.FilteredListAuditLogs(search, user, entity, dateFrom, dateTo, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// handleGetProductLifecycle returns the full lifecycle history of a product.
func (s *Server) handleGetProductLifecycle(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID товару")
		return
	}
	lifecycle, err := s.store.GetProductLifecycle(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, lifecycle)
}

// --- Attachments ---

const attachmentMaxMemory = 10 << 20 // 10 MB

// handleListAttachments GET /attachments?entityType=service_order&entityId=42
func (s *Server) handleListAttachments(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entityType")
	rawID := r.URL.Query().Get("entityId")
	entityID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || entityID <= 0 || entityType == "" {
		writeError(w, http.StatusBadRequest, "entityType та entityId обов'язкові")
		return
	}
	items, err := s.store.ListAttachments(entityType, entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleUploadAttachment POST /attachments (multipart/form-data)
func (s *Server) handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(attachmentMaxMemory); err != nil {
		writeError(w, http.StatusBadRequest, "невірний multipart запит")
		return
	}
	entityType := r.FormValue("entityType")
	rawID := r.FormValue("entityId")
	entityID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || entityID <= 0 || entityType == "" {
		writeError(w, http.StatusBadRequest, "entityType та entityId обов'язкові")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "файл обов'язковий")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "не вдалося прочитати файл")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	item, err := s.store.CreateAttachment(app.CreateAttachmentInput{
		EntityType: entityType,
		EntityID:   entityID,
		FileName:   header.Filename,
		MimeType:   mimeType,
		Data:       data,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.store.AddAuditLog(
		userFromContext(r.Context()),
		"attachment.upload",
		entityType,
		fmt.Sprintf("id=%d entity_id=%d file=%s size=%d", item.ID, entityID, header.Filename, len(data)),
	)
	writeJSON(w, http.StatusCreated, item)
}

// handleDownloadAttachment GET /attachments/{id}/download
func (s *Server) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID")
		return
	}
	a, err := s.store.GetAttachmentData(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", a.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.FileName))
	w.Header().Set("Content-Length", strconv.Itoa(len(a.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.Data)
}

// handleDeleteAttachment DELETE /attachments/{id}
func (s *Server) handleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "невірний ID")
		return
	}
	if err := s.store.DeleteAttachment(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	s.store.AddAuditLog(userFromContext(r.Context()), "attachment.delete", "attachment", fmt.Sprintf("id=%d", id))
	w.WriteHeader(http.StatusNoContent)
}

// ── Report handlers ───────────────────────────────────────────────────────────

// handleSupplierReport GET /reports/suppliers
func (s *Server) handleSupplierReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.store.SupplierReport()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка формування звіту по постачальниках")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// handleCounterpartyReport GET /reports/counterparties
func (s *Server) handleCounterpartyReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.store.CounterpartyReport()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "помилка формування звіту по контрагентах")
		return
	}
	writeJSON(w, http.StatusOK, report)
}