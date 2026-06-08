package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"erp-backend/internal/app"
)

type successfulHTTPReceiptSender struct{}

func (s successfulHTTPReceiptSender) Enabled() bool { return true }
func (s successfulHTTPReceiptSender) SendSaleReceipt(sale app.Sale) (app.ReceiptSendResult, error) {
	return app.ReceiptSendResult{
		ExternalID:   fmt.Sprintf("http-ext-%d", sale.ID),
		FiscalNumber: fmt.Sprintf("http-fiscal-%d", sale.ID),
		QRURL:        fmt.Sprintf("https://qr.local/%d", sale.ID),
	}, nil
}

type failingHTTPReceiptSender struct{}

func (f failingHTTPReceiptSender) Enabled() bool { return true }
func (f failingHTTPReceiptSender) SendSaleReceipt(sale app.Sale) (app.ReceiptSendResult, error) {
	return app.ReceiptSendResult{}, fmt.Errorf("temporary checkbox failure for sale %d", sale.ID)
}

func loginToken(t *testing.T, server *Server, username, password string) string {
	t.Helper()

	loginBody := map[string]string{
		"username": username,
		"password": password,
	}
	rawLogin, _ := json.Marshal(loginBody)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(rawLogin))
	loginRec := httptest.NewRecorder()
	server.Router().ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", loginRec.Code)
	}

	var loginRes map[string]string
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginRes); err != nil {
		t.Fatalf("cannot parse login response: %v", err)
	}

	return loginRes["token"]
}

func TestLoginAndProtectedFlow(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":          "Laptop SSD",
		"sku":           "SSD-100",
		"purchasePrice": 1300.0,
		"retailPrice":   1699.0,
		"stock":         7,
		"minStock":      2,
	}
	rawProduct, _ := json.Marshal(productBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createRec.Code, createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listRec.Code)
	}
}

func TestProductsCSVImportExportEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	importBody := map[string]interface{}{
		"csv":            "name,sku,stock,purchasePrice,retailPrice,currency\nDesk,DESK-1,3,1000,1500,UAH",
		"updateExisting": false,
	}
	rawImport, _ := json.Marshal(importBody)
	importReq := httptest.NewRequest(http.MethodPost, "/api/v1/products/import-csv", bytes.NewReader(rawImport))
	importReq.Header.Set("Authorization", "Bearer "+token)
	importRec := httptest.NewRecorder()
	server.Router().ServeHTTP(importRec, importReq)
	if importRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for import csv, got %d; body=%s", importRec.Code, importRec.Body.String())
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/v1/products/export-csv?includeArchived=true", nil)
	exportReq.Header.Set("Authorization", "Bearer "+token)
	exportRec := httptest.NewRecorder()
	server.Router().ServeHTTP(exportRec, exportReq)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for export csv, got %d; body=%s", exportRec.Code, exportRec.Body.String())
	}
	if contentType := exportRec.Header().Get("Content-Type"); contentType == "" || !strings.HasPrefix(contentType, "text/csv") {
		t.Fatalf("expected text/csv content type, got %s", contentType)
	}
	if !bytes.Contains(exportRec.Body.Bytes(), []byte("DESK-1")) {
		t.Fatalf("expected exported csv to contain imported sku")
	}
}

func TestProductsXLSXImportExportEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	createBody := map[string]interface{}{
		"name":  "XLSX API Product",
		"sku":   "XLSX-API-1",
		"stock": 2,
	}
	rawCreate, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawCreate))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createRec.Code, createRec.Body.String())
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/v1/products/export-xlsx?includeArchived=true", nil)
	exportReq.Header.Set("Authorization", "Bearer "+token)
	exportRec := httptest.NewRecorder()
	server.Router().ServeHTTP(exportRec, exportReq)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for export xlsx, got %d; body=%s", exportRec.Code, exportRec.Body.String())
	}
	xlsxBytes := exportRec.Body.Bytes()
	if len(xlsxBytes) == 0 {
		t.Fatalf("expected non-empty xlsx export")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "products.xlsx")
	if err != nil {
		t.Fatalf("cannot create multipart file: %v", err)
	}
	if _, err := part.Write(xlsxBytes); err != nil {
		t.Fatalf("cannot write xlsx content: %v", err)
	}
	if err := writer.WriteField("updateExisting", "true"); err != nil {
		t.Fatalf("cannot write updateExisting field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cannot close multipart writer: %v", err)
	}

	importReq := httptest.NewRequest(http.MethodPost, "/api/v1/products/import-xlsx", body)
	importReq.Header.Set("Authorization", "Bearer "+token)
	importReq.Header.Set("Content-Type", writer.FormDataContentType())
	importRec := httptest.NewRecorder()
	server.Router().ServeHTTP(importRec, importReq)
	if importRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for import xlsx, got %d; body=%s", importRec.Code, importRec.Body.String())
	}
}

func TestProductsBulkPriceUpdateEndpoint(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	createBody := map[string]interface{}{
		"name":        "Bulk Endpoint Product",
		"sku":         "BEP-1",
		"retailPrice": 100,
		"stock":       1,
	}
	rawCreate, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawCreate))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createRec.Code, createRec.Body.String())
	}

	bulkBody := map[string]interface{}{
		"mode":       "percent",
		"value":      10,
		"priceField": "retailPrice",
		"search":     "BEP-1",
	}
	rawBulk, _ := json.Marshal(bulkBody)
	bulkReq := httptest.NewRequest(http.MethodPost, "/api/v1/products/prices/bulk", bytes.NewReader(rawBulk))
	bulkReq.Header.Set("Authorization", "Bearer "+token)
	bulkRec := httptest.NewRecorder()
	server.Router().ServeHTTP(bulkRec, bulkReq)
	if bulkRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for bulk prices update, got %d; body=%s", bulkRec.Code, bulkRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/products?search=BEP-1&includeArchived=true", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for products list, got %d; body=%s", listRec.Code, listRec.Body.String())
	}
	var products []app.Product
	if err := json.Unmarshal(listRec.Body.Bytes(), &products); err != nil {
		t.Fatalf("cannot parse products list: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected one product, got %d", len(products))
	}
	if products[0].RetailPrice < 109.99 || products[0].RetailPrice > 110.01 {
		t.Fatalf("expected retail price 110 after bulk update, got %.2f", products[0].RetailPrice)
	}
}

func TestProductsMergeDuplicatesEndpoint(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	createProduct := func(name, sku string, stock int) app.Product {
		t.Helper()
		createBody := map[string]interface{}{
			"name":  name,
			"sku":   sku,
			"stock": stock,
		}
		rawCreate, _ := json.Marshal(createBody)
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawCreate))
		createReq.Header.Set("Authorization", "Bearer "+token)
		createRec := httptest.NewRecorder()
		server.Router().ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d; body=%s", createRec.Code, createRec.Body.String())
		}
		var product app.Product
		if err := json.Unmarshal(createRec.Body.Bytes(), &product); err != nil {
			t.Fatalf("cannot parse product: %v", err)
		}
		return product
	}

	target := createProduct("Merge Target", "MRGE-T", 5)
	source := createProduct("Merge Source", "MRGE-S", 3)

	mergeBody := map[string]interface{}{
		"targetProductId": target.ID,
		"sourceProductIds": []int64{
			source.ID,
		},
	}
	rawMerge, _ := json.Marshal(mergeBody)
	mergeReq := httptest.NewRequest(http.MethodPost, "/api/v1/products/merge-duplicates", bytes.NewReader(rawMerge))
	mergeReq.Header.Set("Authorization", "Bearer "+token)
	mergeRec := httptest.NewRecorder()
	server.Router().ServeHTTP(mergeRec, mergeReq)
	if mergeRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for merge duplicates, got %d; body=%s", mergeRec.Code, mergeRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/products?includeArchived=true", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for products list, got %d; body=%s", listRec.Code, listRec.Body.String())
	}
	var products []app.Product
	if err := json.Unmarshal(listRec.Body.Bytes(), &products); err != nil {
		t.Fatalf("cannot parse products list: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected one product after merge, got %d", len(products))
	}
	if products[0].ID != target.ID {
		t.Fatalf("expected remaining target product id %d, got %d", target.ID, products[0].ID)
	}
	if products[0].Stock != 8 {
		t.Fatalf("expected merged stock 8, got %d", products[0].Stock)
	}
}

func TestAnalyticsProfitabilityEndpoint(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	createProductBody := map[string]interface{}{
		"name":          "Profit Endpoint Product",
		"sku":           "PEP-1",
		"stock":         10,
		"purchasePrice": 50,
		"retailPrice":   90,
		"currency":      "UAH",
	}
	rawProduct, _ := json.Marshal(createProductBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createRec.Code, createRec.Body.String())
	}
	var product app.Product
	if err := json.Unmarshal(createRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product response: %v", err)
	}

	saleBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  2,
				"price":     90,
			},
		},
	}
	rawSale, _ := json.Marshal(saleBody)
	saleReq := httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(rawSale))
	saleReq.Header.Set("Authorization", "Bearer "+token)
	saleRec := httptest.NewRecorder()
	server.Router().ServeHTTP(saleRec, saleReq)
	if saleRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for sale, got %d; body=%s", saleRec.Code, saleRec.Body.String())
	}

	profitReq := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/profitability", nil)
	profitReq.Header.Set("Authorization", "Bearer "+token)
	profitRec := httptest.NewRecorder()
	server.Router().ServeHTTP(profitRec, profitReq)
	if profitRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for profitability endpoint, got %d; body=%s", profitRec.Code, profitRec.Body.String())
	}
	var report app.ProfitabilityReport
	if err := json.Unmarshal(profitRec.Body.Bytes(), &report); err != nil {
		t.Fatalf("cannot parse profitability response: %v", err)
	}
	if len(report.Items) == 0 {
		t.Fatalf("expected non-empty profitability report items")
	}
}

func TestCRMCustomersAndRemindersEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	customerBody := map[string]interface{}{
		"name":    "CRM User",
		"phone":   "+380500000001",
		"email":   "crm@example.com",
		"comment": "lead",
	}
	rawCustomer, _ := json.Marshal(customerBody)
	createCustomerReq := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewReader(rawCustomer))
	createCustomerReq.Header.Set("Authorization", "Bearer "+token)
	createCustomerRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createCustomerRec, createCustomerReq)
	if createCustomerRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for customer create, got %d; body=%s", createCustomerRec.Code, createCustomerRec.Body.String())
	}
	var customer app.Customer
	if err := json.Unmarshal(createCustomerRec.Body.Bytes(), &customer); err != nil {
		t.Fatalf("cannot parse customer response: %v", err)
	}

	listCustomersReq := httptest.NewRequest(http.MethodGet, "/api/v1/customers?search=crm", nil)
	listCustomersReq.Header.Set("Authorization", "Bearer "+token)
	listCustomersRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listCustomersRec, listCustomersReq)
	if listCustomersRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for list customers, got %d; body=%s", listCustomersRec.Code, listCustomersRec.Body.String())
	}

	reminderBody := map[string]interface{}{
		"customerId": customer.ID,
		"text":       "Call and offer discount",
	}
	rawReminder, _ := json.Marshal(reminderBody)
	createReminderReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer-reminders", bytes.NewReader(rawReminder))
	createReminderReq.Header.Set("Authorization", "Bearer "+token)
	createReminderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createReminderRec, createReminderReq)
	if createReminderRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for reminder create, got %d; body=%s", createReminderRec.Code, createReminderRec.Body.String())
	}
	var reminder app.CustomerReminder
	if err := json.Unmarshal(createReminderRec.Body.Bytes(), &reminder); err != nil {
		t.Fatalf("cannot parse reminder response: %v", err)
	}

	listRemindersReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/customer-reminders?customerId=%d&status=pending", customer.ID), nil)
	listRemindersReq.Header.Set("Authorization", "Bearer "+token)
	listRemindersRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRemindersRec, listRemindersReq)
	if listRemindersRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for list reminders, got %d; body=%s", listRemindersRec.Code, listRemindersRec.Body.String())
	}

	completeReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/customer-reminders/%d/complete", reminder.ID), bytes.NewReader([]byte(`{}`)))
	completeReq.Header.Set("Authorization", "Bearer "+token)
	completeRec := httptest.NewRecorder()
	server.Router().ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for reminder complete, got %d; body=%s", completeRec.Code, completeRec.Body.String())
	}
}

func TestServiceOrdersEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	customerBody := map[string]interface{}{
		"name":  "Service Client",
		"phone": "+380500000777",
	}
	rawCustomer, _ := json.Marshal(customerBody)
	createCustomerReq := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewReader(rawCustomer))
	createCustomerReq.Header.Set("Authorization", "Bearer "+token)
	createCustomerRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createCustomerRec, createCustomerReq)
	if createCustomerRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for customer create, got %d; body=%s", createCustomerRec.Code, createCustomerRec.Body.String())
	}
	var customer app.Customer
	if err := json.Unmarshal(createCustomerRec.Body.Bytes(), &customer); err != nil {
		t.Fatalf("cannot parse customer response: %v", err)
	}

	serviceOrderBody := map[string]interface{}{
		"customerId":  customer.ID,
		"title":       "Battery replacement",
		"description": "Replace old battery",
		"technician":  "Master 1",
		"laborMin":    30,
		"price":       800,
		"currency":    "UAH",
	}
	rawServiceOrder, _ := json.Marshal(serviceOrderBody)
	createOrderReq := httptest.NewRequest(http.MethodPost, "/api/v1/service-orders", bytes.NewReader(rawServiceOrder))
	createOrderReq.Header.Set("Authorization", "Bearer "+token)
	createOrderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createOrderRec, createOrderReq)
	if createOrderRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for service order create, got %d; body=%s", createOrderRec.Code, createOrderRec.Body.String())
	}
	var created app.ServiceOrder
	if err := json.Unmarshal(createOrderRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("cannot parse service order response: %v", err)
	}
	if created.Technician != "Master 1" || created.LaborMin != 30 {
		t.Fatalf("expected technician/labor fields in create response, got %+v", created)
	}

	partProductBody := map[string]interface{}{
		"name":        "Service Part",
		"sku":         "SRV-PART-1",
		"stock":       4,
		"retailPrice": 300,
		"currency":    "UAH",
	}
	rawPartProduct, _ := json.Marshal(partProductBody)
	createPartReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawPartProduct))
	createPartReq.Header.Set("Authorization", "Bearer "+token)
	createPartRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createPartRec, createPartReq)
	if createPartRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for part product create, got %d; body=%s", createPartRec.Code, createPartRec.Body.String())
	}
	var partProduct app.Product
	if err := json.Unmarshal(createPartRec.Body.Bytes(), &partProduct); err != nil {
		t.Fatalf("cannot parse part product response: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/service-orders?customerId=%d&status=new", customer.ID), nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for service order list, got %d; body=%s", listRec.Code, listRec.Body.String())
	}

	updateOrderBody := map[string]interface{}{
		"productId":   partProduct.ID,
		"title":       "Battery + diagnostics",
		"description": "Replace battery and run diagnostics",
		"technician":  "Master 2",
		"laborMin":    55,
		"price":       950,
		"currency":    "UAH",
	}
	rawUpdateOrder, _ := json.Marshal(updateOrderBody)
	updateOrderReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/service-orders/%d", created.ID), bytes.NewReader(rawUpdateOrder))
	updateOrderReq.Header.Set("Authorization", "Bearer "+token)
	updateOrderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(updateOrderRec, updateOrderReq)
	if updateOrderRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for service order details update, got %d; body=%s", updateOrderRec.Code, updateOrderRec.Body.String())
	}

	addPartBody := map[string]interface{}{
		"productId": partProduct.ID,
		"quantity":  2,
		"price":     250,
	}
	rawPart, _ := json.Marshal(addPartBody)
	addPartReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/service-orders/%d/parts", created.ID), bytes.NewReader(rawPart))
	addPartReq.Header.Set("Authorization", "Bearer "+token)
	addPartRec := httptest.NewRecorder()
	server.Router().ServeHTTP(addPartRec, addPartReq)
	if addPartRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for service order part add, got %d; body=%s", addPartRec.Code, addPartRec.Body.String())
	}

	paymentBody := map[string]interface{}{
		"serviceOrderId": created.ID,
		"cashboxId":      1,
		"amount":         400.0,
		"currency":       "UAH",
		"method":         "cash",
		"note":           "service prepayment",
	}
	rawPayment, _ := json.Marshal(paymentBody)
	createPaymentReq := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(rawPayment))
	createPaymentReq.Header.Set("Authorization", "Bearer "+token)
	createPaymentRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createPaymentRec, createPaymentReq)
	if createPaymentRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for service order payment, got %d; body=%s", createPaymentRec.Code, createPaymentRec.Body.String())
	}

	servicePaymentsReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/payments?serviceOrderId=%d", created.ID), nil)
	servicePaymentsReq.Header.Set("Authorization", "Bearer "+token)
	servicePaymentsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(servicePaymentsRec, servicePaymentsReq)
	if servicePaymentsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for service order payments list, got %d; body=%s", servicePaymentsRec.Code, servicePaymentsRec.Body.String())
	}

	listAfterPaymentReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/service-orders?customerId=%d", customer.ID), nil)
	listAfterPaymentReq.Header.Set("Authorization", "Bearer "+token)
	listAfterPaymentRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listAfterPaymentRec, listAfterPaymentReq)
	if listAfterPaymentRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for service order list after payment, got %d; body=%s", listAfterPaymentRec.Code, listAfterPaymentRec.Body.String())
	}
	var afterPayment []app.ServiceOrder
	if err := json.Unmarshal(listAfterPaymentRec.Body.Bytes(), &afterPayment); err != nil {
		t.Fatalf("cannot parse service orders after payment: %v", err)
	}
	if len(afterPayment) == 0 || afterPayment[0].Paid <= 0 {
		t.Fatalf("expected paid amount in service order after payment")
	}

	actBody := map[string]interface{}{
		"note":     "Service is done",
		"autoPost": true,
	}
	rawAct, _ := json.Marshal(actBody)
	createActReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/service-orders/%d/act", created.ID), bytes.NewReader(rawAct))
	createActReq.Header.Set("Authorization", "Bearer "+token)
	createActRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createActRec, createActReq)
	if createActRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for act create before done status, got %d; body=%s", createActRec.Code, createActRec.Body.String())
	}

	updateBody := map[string]interface{}{"status": "in_progress"}
	rawUpdate, _ := json.Marshal(updateBody)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/service-orders/%d/status", created.ID), bytes.NewReader(rawUpdate))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateRec := httptest.NewRecorder()
	server.Router().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for service order status update, got %d; body=%s", updateRec.Code, updateRec.Body.String())
	}

	doneBody := map[string]interface{}{"status": "done"}
	rawDone, _ := json.Marshal(doneBody)
	doneReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/service-orders/%d/status", created.ID), bytes.NewReader(rawDone))
	doneReq.Header.Set("Authorization", "Bearer "+token)
	doneRec := httptest.NewRecorder()
	server.Router().ServeHTTP(doneRec, doneReq)
	if doneRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for service order done status update, got %d; body=%s", doneRec.Code, doneRec.Body.String())
	}

	createActAfterDoneReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/service-orders/%d/act", created.ID), bytes.NewReader(rawAct))
	createActAfterDoneReq.Header.Set("Authorization", "Bearer "+token)
	createActAfterDoneRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createActAfterDoneRec, createActAfterDoneReq)
	if createActAfterDoneRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for duplicate act after auto-create, got %d; body=%s", createActAfterDoneRec.Code, createActAfterDoneRec.Body.String())
	}

	pdfReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/service-orders/%d/act/pdf", created.ID), nil)
	pdfReq.Header.Set("Authorization", "Bearer "+token)
	pdfRec := httptest.NewRecorder()
	server.Router().ServeHTTP(pdfRec, pdfReq)
	if pdfRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for service order act pdf, got %d; body=%s", pdfRec.Code, pdfRec.Body.String())
	}
	if got := pdfRec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("expected pdf content-type, got %s", got)
	}

	reopenBody := map[string]interface{}{"status": "in_progress"}
	rawReopen, _ := json.Marshal(reopenBody)
	reopenReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/service-orders/%d/status", created.ID), bytes.NewReader(rawReopen))
	reopenReq.Header.Set("Authorization", "Bearer "+token)
	reopenRec := httptest.NewRecorder()
	server.Router().ServeHTTP(reopenRec, reopenReq)
	if reopenRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for reopen with active act, got %d; body=%s", reopenRec.Code, reopenRec.Body.String())
	}

	cancelActReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/service-orders/%d/act/cancel", created.ID), bytes.NewReader([]byte(`{}`)))
	cancelActReq.Header.Set("Authorization", "Bearer "+token)
	cancelActRec := httptest.NewRecorder()
	server.Router().ServeHTTP(cancelActRec, cancelActReq)
	if cancelActRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for cancel act, got %d; body=%s", cancelActRec.Code, cancelActRec.Body.String())
	}

	reopenAfterCancelReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/service-orders/%d/status", created.ID), bytes.NewReader(rawReopen))
	reopenAfterCancelReq.Header.Set("Authorization", "Bearer "+token)
	reopenAfterCancelRec := httptest.NewRecorder()
	server.Router().ServeHTTP(reopenAfterCancelRec, reopenAfterCancelReq)
	if reopenAfterCancelRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for reopen after act cancel, got %d; body=%s", reopenAfterCancelRec.Code, reopenAfterCancelRec.Body.String())
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs?limit=100", nil)
	auditReq.Header.Set("Authorization", "Bearer "+token)
	auditRec := httptest.NewRecorder()
	server.Router().ServeHTTP(auditRec, auditReq)
	if auditRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for audit logs list, got %d; body=%s", auditRec.Code, auditRec.Body.String())
	}
	var logs []app.AuditLog
	if err := json.Unmarshal(auditRec.Body.Bytes(), &logs); err != nil {
		t.Fatalf("cannot parse audit logs response: %v", err)
	}
	hasReopen := false
	hasAutoAct := false
	for _, entry := range logs {
		if entry.Action == "service_order.reopen" {
			hasReopen = true
		}
		if entry.Action == "service_order.auto_act.create" {
			hasAutoAct = true
		}
	}
	if !hasReopen {
		t.Fatalf("expected reopen audit action in logs")
	}
	if !hasAutoAct {
		t.Fatalf("expected auto act create audit action in logs")
	}
}

func TestSellerCannotCreateProduct(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "seller", "seller123")
	productBody := map[string]interface{}{
		"name": "Blocked Product",
		"sku":  "BLK-1",
	}
	rawProduct, _ := json.Marshal(productBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", createRec.Code)
	}
}

func TestAdminCanReadAuditLogs(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "admin", "admin123")
	productBody := map[string]interface{}{
		"name": "Logged Product",
		"sku":  "LOG-1",
	}
	rawProduct, _ := json.Marshal(productBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createRec.Code)
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs?limit=10", nil)
	auditReq.Header.Set("Authorization", "Bearer "+token)
	auditRec := httptest.NewRecorder()
	server.Router().ServeHTTP(auditRec, auditReq)
	if auditRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", auditRec.Code)
	}

	var logs []app.AuditLog
	if err := json.Unmarshal(auditRec.Body.Bytes(), &logs); err != nil {
		t.Fatalf("cannot parse logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected audit logs to be present")
	}
}

func TestAdminCanCreateUserAndListUsers(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "admin", "admin123")
	createBody := map[string]string{
		"username": "manager",
		"password": "manager123",
		"role":     "seller",
	}
	rawCreate, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(rawCreate))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createRec.Code, createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listRec.Code)
	}
}

func TestSellerCannotReadUsers(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "seller", "seller123")
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", listRec.Code)
	}
}

func TestOrderReservationAndCancellationFlow(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":  "Reserved Product",
		"sku":   "RSV-1",
		"stock": 10,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createProductRec.Code)
	}

	var createdProduct app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &createdProduct); err != nil {
		t.Fatalf("cannot parse product response: %v", err)
	}

	orderBody := map[string]interface{}{
		"customerName": "Customer A",
		"reserve":      true,
		"items": []map[string]interface{}{
			{
				"productId": createdProduct.ID,
				"quantity":  8,
				"price":     100,
			},
		},
	}
	rawOrder, _ := json.Marshal(orderBody)
	createOrderReq := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(rawOrder))
	createOrderReq.Header.Set("Authorization", "Bearer "+token)
	createOrderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createOrderRec, createOrderReq)
	if createOrderRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createOrderRec.Code, createOrderRec.Body.String())
	}

	saleBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"productId": createdProduct.ID,
				"quantity":  3,
				"price":     100,
			},
		},
	}
	rawSale, _ := json.Marshal(saleBody)
	createSaleReq := httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(rawSale))
	createSaleReq.Header.Set("Authorization", "Bearer "+token)
	createSaleRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createSaleRec, createSaleReq)
	if createSaleRec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", createSaleRec.Code)
	}

	var createdOrder app.CustomerOrder
	if err := json.Unmarshal(createOrderRec.Body.Bytes(), &createdOrder); err != nil {
		t.Fatalf("cannot parse order response: %v", err)
	}

	updateBody := map[string]string{"status": app.OrderStatusCancelled}
	rawUpdate, _ := json.Marshal(updateBody)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/orders/%d/status", createdOrder.ID), bytes.NewReader(rawUpdate))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateRec := httptest.NewRecorder()
	server.Router().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", updateRec.Code, updateRec.Body.String())
	}
}

func TestPaymentAndDebtsFlow(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":  "Debt Product",
		"sku":   "DEBT-1",
		"stock": 5,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createProductRec.Code)
	}

	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product response: %v", err)
	}

	orderBody := map[string]interface{}{
		"customerName": "Customer Debt",
		"reserve":      false,
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  2,
				"price":     100,
			},
		},
	}
	rawOrder, _ := json.Marshal(orderBody)
	createOrderReq := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(rawOrder))
	createOrderReq.Header.Set("Authorization", "Bearer "+token)
	createOrderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createOrderRec, createOrderReq)
	if createOrderRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createOrderRec.Code)
	}

	var order app.CustomerOrder
	if err := json.Unmarshal(createOrderRec.Body.Bytes(), &order); err != nil {
		t.Fatalf("cannot parse order response: %v", err)
	}

	paymentBody := map[string]interface{}{
		"orderId": order.ID,
		"amount":  50,
		"method":  "cash",
	}
	rawPayment, _ := json.Marshal(paymentBody)
	createPaymentReq := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(rawPayment))
	createPaymentReq.Header.Set("Authorization", "Bearer "+token)
	createPaymentRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createPaymentRec, createPaymentReq)
	if createPaymentRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createPaymentRec.Code, createPaymentRec.Body.String())
	}

	debtsReq := httptest.NewRequest(http.MethodGet, "/api/v1/debts", nil)
	debtsReq.Header.Set("Authorization", "Bearer "+token)
	debtsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(debtsRec, debtsReq)
	if debtsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", debtsRec.Code)
	}

	var debts []app.DebtSummary
	if err := json.Unmarshal(debtsRec.Body.Bytes(), &debts); err != nil {
		t.Fatalf("cannot parse debts response: %v", err)
	}
	if len(debts) == 0 {
		t.Fatalf("expected at least one debt item")
	}
}

func TestCashboxAndCashOperationFlow(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "admin", "admin123")

	cashboxesReq := httptest.NewRequest(http.MethodGet, "/api/v1/cashboxes", nil)
	cashboxesReq.Header.Set("Authorization", "Bearer "+token)
	cashboxesRec := httptest.NewRecorder()
	server.Router().ServeHTTP(cashboxesRec, cashboxesReq)
	if cashboxesRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", cashboxesRec.Code)
	}

	var cashboxes []app.Cashbox
	if err := json.Unmarshal(cashboxesRec.Body.Bytes(), &cashboxes); err != nil {
		t.Fatalf("cannot parse cashboxes response: %v", err)
	}
	if len(cashboxes) == 0 {
		t.Fatalf("expected default cashboxes")
	}

	operationBody := map[string]interface{}{
		"cashboxId":   cashboxes[0].ID,
		"type":        app.CashOperationTypeIncoming,
		"amount":      150,
		"method":      app.PaymentMethodCash,
		"description": "Manual cash in",
	}
	rawOperation, _ := json.Marshal(operationBody)
	createOperationReq := httptest.NewRequest(http.MethodPost, "/api/v1/cash-operations", bytes.NewReader(rawOperation))
	createOperationReq.Header.Set("Authorization", "Bearer "+token)
	createOperationRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createOperationRec, createOperationReq)
	if createOperationRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createOperationRec.Code, createOperationRec.Body.String())
	}
}

func TestOverdueDebtsAndHistoryEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())

	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":  "Overdue API Product",
		"sku":   "OD-1",
		"stock": 5,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createProductRec.Code)
	}

	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product response: %v", err)
	}

	dueDate := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	orderBody := map[string]interface{}{
		"customerName": "Overdue API Client",
		"reserve":      false,
		"currency":     "USD",
		"dueDate":      dueDate,
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
				"price":     100,
			},
		},
	}
	rawOrder, _ := json.Marshal(orderBody)
	createOrderReq := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(rawOrder))
	createOrderReq.Header.Set("Authorization", "Bearer "+token)
	createOrderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createOrderRec, createOrderReq)
	if createOrderRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createOrderRec.Code)
	}

	var order app.CustomerOrder
	if err := json.Unmarshal(createOrderRec.Body.Bytes(), &order); err != nil {
		t.Fatalf("cannot parse order response: %v", err)
	}

	paymentBody := map[string]interface{}{
		"orderId":  order.ID,
		"amount":   20,
		"currency": "UAH",
		"method":   "card",
	}
	rawPayment, _ := json.Marshal(paymentBody)
	createPaymentReq := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(rawPayment))
	createPaymentReq.Header.Set("Authorization", "Bearer "+token)
	createPaymentRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createPaymentRec, createPaymentReq)
	if createPaymentRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createPaymentRec.Code)
	}

	overdueReq := httptest.NewRequest(http.MethodGet, "/api/v1/debts/overdue", nil)
	overdueReq.Header.Set("Authorization", "Bearer "+token)
	overdueRec := httptest.NewRecorder()
	server.Router().ServeHTTP(overdueRec, overdueReq)
	if overdueRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", overdueRec.Code)
	}

	historyReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/debts/history?entityType=order&entityId=%d", order.ID), nil)
	historyReq.Header.Set("Authorization", "Bearer "+token)
	historyRec := httptest.NewRecorder()
	server.Router().ServeHTTP(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", historyRec.Code)
	}
}

func TestBackgroundJobsAndNotificationsEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":  "Job Product",
		"sku":   "JOB-1",
		"stock": 3,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createProductRec.Code)
	}

	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product: %v", err)
	}

	dueDate := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	orderBody := map[string]interface{}{
		"customerName": "Job Client",
		"reserve":      false,
		"dueDate":      dueDate,
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
				"price":     100,
			},
		},
	}
	rawOrder, _ := json.Marshal(orderBody)
	createOrderReq := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(rawOrder))
	createOrderReq.Header.Set("Authorization", "Bearer "+token)
	createOrderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createOrderRec, createOrderReq)
	if createOrderRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createOrderRec.Code)
	}

	enqueueReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/overdue-reminders", bytes.NewReader([]byte(`{}`)))
	enqueueReq.Header.Set("Authorization", "Bearer "+token)
	enqueueRec := httptest.NewRecorder()
	server.Router().ServeHTTP(enqueueRec, enqueueReq)
	if enqueueRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", enqueueRec.Code)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/run", bytes.NewReader([]byte(`{}`)))
	runReq.Header.Set("Authorization", "Bearer "+token)
	runRec := httptest.NewRecorder()
	server.Router().ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", runRec.Code)
	}

	notificationsReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=10", nil)
	notificationsReq.Header.Set("Authorization", "Bearer "+token)
	notificationsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(notificationsRec, notificationsReq)
	if notificationsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", notificationsRec.Code)
	}
}

func TestSuppliersSupplierOrdersAndPurchasesEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":     "Supplier API Product",
		"sku":      "SAP-1",
		"stock":    2,
		"minStock": 5,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createProductRec.Code)
	}

	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product: %v", err)
	}

	supplierBody := map[string]interface{}{
		"name":    "API Supplier",
		"contact": "John",
	}
	rawSupplier, _ := json.Marshal(supplierBody)
	createSupplierReq := httptest.NewRequest(http.MethodPost, "/api/v1/suppliers", bytes.NewReader(rawSupplier))
	createSupplierReq.Header.Set("Authorization", "Bearer "+token)
	createSupplierRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createSupplierRec, createSupplierReq)
	if createSupplierRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createSupplierRec.Code, createSupplierRec.Body.String())
	}

	var supplier app.Supplier
	if err := json.Unmarshal(createSupplierRec.Body.Bytes(), &supplier); err != nil {
		t.Fatalf("cannot parse supplier: %v", err)
	}

	orderBody := map[string]interface{}{
		"supplierId": supplier.ID,
		"currency":   "UAH",
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  4,
				"price":     50,
			},
		},
	}
	rawOrder, _ := json.Marshal(orderBody)
	createOrderReq := httptest.NewRequest(http.MethodPost, "/api/v1/supplier-orders", bytes.NewReader(rawOrder))
	createOrderReq.Header.Set("Authorization", "Bearer "+token)
	createOrderRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createOrderRec, createOrderReq)
	if createOrderRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createOrderRec.Code, createOrderRec.Body.String())
	}

	var supplierOrder app.SupplierOrder
	if err := json.Unmarshal(createOrderRec.Body.Bytes(), &supplierOrder); err != nil {
		t.Fatalf("cannot parse supplier order: %v", err)
	}

	forceReceivedBody := map[string]interface{}{
		"status": app.SupplierOrderStatusReceived,
	}
	rawForceReceived, _ := json.Marshal(forceReceivedBody)
	forceReceivedReq := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/supplier-orders/%d/status", supplierOrder.ID),
		bytes.NewReader(rawForceReceived),
	)
	forceReceivedReq.Header.Set("Authorization", "Bearer "+token)
	forceReceivedRec := httptest.NewRecorder()
	server.Router().ServeHTTP(forceReceivedRec, forceReceivedReq)
	if forceReceivedRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for premature received status, got %d", forceReceivedRec.Code)
	}

	pendingReq := httptest.NewRequest(http.MethodGet, "/api/v1/supplier-orders/pending", nil)
	pendingReq.Header.Set("Authorization", "Bearer "+token)
	pendingRec := httptest.NewRecorder()
	server.Router().ServeHTTP(pendingRec, pendingReq)
	if pendingRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", pendingRec.Code)
	}

	recommendationsReq := httptest.NewRequest(http.MethodGet, "/api/v1/supplier-orders/recommendations?limit=10", nil)
	recommendationsReq.Header.Set("Authorization", "Bearer "+token)
	recommendationsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(recommendationsRec, recommendationsReq)
	if recommendationsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", recommendationsRec.Code, recommendationsRec.Body.String())
	}
	var recommendations []app.PurchaseRecommendation
	if err := json.Unmarshal(recommendationsRec.Body.Bytes(), &recommendations); err != nil {
		t.Fatalf("cannot parse recommendations: %v", err)
	}
	if len(recommendations) == 0 {
		t.Fatalf("expected non-empty recommendations")
	}

	createFromRecommendationsBody := map[string]interface{}{
		"supplierId": supplier.ID,
		"currency":   "UAH",
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
			},
		},
	}
	rawCreateFromRecommendations, _ := json.Marshal(createFromRecommendationsBody)
	createFromRecommendationsReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/supplier-orders/recommendations/create-order",
		bytes.NewReader(rawCreateFromRecommendations),
	)
	createFromRecommendationsReq.Header.Set("Authorization", "Bearer "+token)
	createFromRecommendationsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createFromRecommendationsRec, createFromRecommendationsReq)
	if createFromRecommendationsRec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status 201, got %d; body=%s",
			createFromRecommendationsRec.Code,
			createFromRecommendationsRec.Body.String(),
		)
	}

	groupedRecommendationsReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/supplier-orders/recommendations/grouped?limit=10",
		nil,
	)
	groupedRecommendationsReq.Header.Set("Authorization", "Bearer "+token)
	groupedRecommendationsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(groupedRecommendationsRec, groupedRecommendationsReq)
	if groupedRecommendationsRec.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d; body=%s",
			groupedRecommendationsRec.Code,
			groupedRecommendationsRec.Body.String(),
		)
	}

	bulkCreateBody := map[string]interface{}{
		"orders": []map[string]interface{}{
			{
				"supplierId": supplier.ID,
				"currency":   "UAH",
				"items": []map[string]interface{}{
					{
						"productId": product.ID,
						"quantity":  1,
					},
				},
			},
		},
	}
	rawBulkCreate, _ := json.Marshal(bulkCreateBody)
	bulkCreateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/supplier-orders/recommendations/create-orders-bulk",
		bytes.NewReader(rawBulkCreate),
	)
	bulkCreateReq.Header.Set("Authorization", "Bearer "+token)
	bulkCreateRec := httptest.NewRecorder()
	server.Router().ServeHTTP(bulkCreateRec, bulkCreateReq)
	if bulkCreateRec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status 201, got %d; body=%s",
			bulkCreateRec.Code,
			bulkCreateRec.Body.String(),
		)
	}

	receiveBody := map[string]interface{}{
		"lines": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  4,
			},
		},
		"note": "full receipt by lines",
	}
	rawReceive, _ := json.Marshal(receiveBody)
	receiveReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/supplier-orders/%d/receive", supplierOrder.ID),
		bytes.NewReader(rawReceive),
	)
	receiveReq.Header.Set("Authorization", "Bearer "+token)
	receiveRec := httptest.NewRecorder()
	server.Router().ServeHTTP(receiveRec, receiveReq)
	if receiveRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", receiveRec.Code, receiveRec.Body.String())
	}

	overReceiveBody := map[string]interface{}{
		"lines": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
			},
		},
	}
	rawOverReceive, _ := json.Marshal(overReceiveBody)
	overReceiveReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/supplier-orders/%d/receive", supplierOrder.ID),
		bytes.NewReader(rawOverReceive),
	)
	overReceiveReq.Header.Set("Authorization", "Bearer "+token)
	overReceiveRec := httptest.NewRecorder()
	server.Router().ServeHTTP(overReceiveRec, overReceiveReq)
	if overReceiveRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for over-receipt, got %d", overReceiveRec.Code)
	}

	listOrdersReq := httptest.NewRequest(http.MethodGet, "/api/v1/supplier-orders", nil)
	listOrdersReq.Header.Set("Authorization", "Bearer "+token)
	listOrdersRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listOrdersRec, listOrdersReq)
	if listOrdersRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listOrdersRec.Code)
	}

	var orders []app.SupplierOrder
	if err := json.Unmarshal(listOrdersRec.Body.Bytes(), &orders); err != nil {
		t.Fatalf("cannot parse supplier orders list: %v", err)
	}
	if len(orders) == 0 || orders[0].Status != app.SupplierOrderStatusReceived {
		t.Fatalf("expected supplier order status received")
	}

	downgradeBody := map[string]interface{}{
		"status": app.SupplierOrderStatusDraft,
	}
	rawDowngrade, _ := json.Marshal(downgradeBody)
	downgradeReq := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/supplier-orders/%d/status", supplierOrder.ID),
		bytes.NewReader(rawDowngrade),
	)
	downgradeReq.Header.Set("Authorization", "Bearer "+token)
	downgradeRec := httptest.NewRecorder()
	server.Router().ServeHTTP(downgradeRec, downgradeReq)
	if downgradeRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for downgrade transition, got %d", downgradeRec.Code)
	}
}

func TestWarehouseTransferAndInventoryEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":  "WH API Product",
		"sku":   "WHA-1",
		"stock": 8,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createProductRec.Code)
	}
	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product: %v", err)
	}

	warehouseBody := map[string]interface{}{"name": "WH-2"}
	rawWarehouse, _ := json.Marshal(warehouseBody)
	createWarehouseReq := httptest.NewRequest(http.MethodPost, "/api/v1/warehouses", bytes.NewReader(rawWarehouse))
	createWarehouseReq.Header.Set("Authorization", "Bearer "+token)
	createWarehouseRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createWarehouseRec, createWarehouseReq)
	if createWarehouseRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createWarehouseRec.Code)
	}
	var warehouse app.Warehouse
	if err := json.Unmarshal(createWarehouseRec.Body.Bytes(), &warehouse); err != nil {
		t.Fatalf("cannot parse warehouse: %v", err)
	}

	transferBody := map[string]interface{}{
		"fromWarehouseId": 1,
		"toWarehouseId":   warehouse.ID,
		"items": []map[string]interface{}{
			{"productId": product.ID, "quantity": 3},
		},
	}
	rawTransfer, _ := json.Marshal(transferBody)
	createTransferReq := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(rawTransfer))
	createTransferReq.Header.Set("Authorization", "Bearer "+token)
	createTransferRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createTransferRec, createTransferReq)
	if createTransferRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createTransferRec.Code, createTransferRec.Body.String())
	}

	inventoryBody := map[string]interface{}{
		"warehouseId": warehouse.ID,
		"items": []map[string]interface{}{
			{"productId": product.ID, "actualQuantity": 2},
		},
	}
	rawInventory, _ := json.Marshal(inventoryBody)
	createInventoryReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventories", bytes.NewReader(rawInventory))
	createInventoryReq.Header.Set("Authorization", "Bearer "+token)
	createInventoryRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createInventoryRec, createInventoryReq)
	if createInventoryRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createInventoryRec.Code, createInventoryRec.Body.String())
	}
	var inventory app.Inventory
	if err := json.Unmarshal(createInventoryRec.Body.Bytes(), &inventory); err != nil {
		t.Fatalf("cannot parse inventory: %v", err)
	}

	applyReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/inventories/%d/apply", inventory.ID), bytes.NewReader([]byte(`{}`)))
	applyReq.Header.Set("Authorization", "Bearer "+token)
	applyRec := httptest.NewRecorder()
	server.Router().ServeHTTP(applyRec, applyReq)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", applyRec.Code, applyRec.Body.String())
	}

	movementsReq := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/stock/movements?productId=%d&warehouseId=%d", product.ID, warehouse.ID),
		nil,
	)
	movementsReq.Header.Set("Authorization", "Bearer "+token)
	movementsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(movementsRec, movementsReq)
	if movementsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for stock movements, got %d; body=%s", movementsRec.Code, movementsRec.Body.String())
	}
	var movements []app.StockMovement
	if err := json.Unmarshal(movementsRec.Body.Bytes(), &movements); err != nil {
		t.Fatalf("cannot parse stock movements: %v", err)
	}
	if len(movements) == 0 {
		t.Fatalf("expected product flow movements")
	}
}

func TestSalesListEndpoint(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":  "Sales List Product",
		"sku":   "SLS-1",
		"stock": 5,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createProductRec.Code, createProductRec.Body.String())
	}
	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product: %v", err)
	}

	createSaleBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  2,
				"price":     100,
			},
		},
	}
	rawSale, _ := json.Marshal(createSaleBody)
	createSaleReq := httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(rawSale))
	createSaleReq.Header.Set("Authorization", "Bearer "+token)
	createSaleRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createSaleRec, createSaleReq)
	if createSaleRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createSaleRec.Code, createSaleRec.Body.String())
	}

	listSalesReq := httptest.NewRequest(http.MethodGet, "/api/v1/sales", nil)
	listSalesReq.Header.Set("Authorization", "Bearer "+token)
	listSalesRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listSalesRec, listSalesReq)
	if listSalesRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", listSalesRec.Code, listSalesRec.Body.String())
	}
	var sales []app.Sale
	if err := json.Unmarshal(listSalesRec.Body.Bytes(), &sales); err != nil {
		t.Fatalf("cannot parse sales list: %v", err)
	}
	if len(sales) == 0 {
		t.Fatalf("expected non-empty sales list")
	}
}

func TestReceiptsEndpoints(t *testing.T) {
	store := app.NewStore()
	store.SetReceiptSender(failingHTTPReceiptSender{})
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{
		"name":  "Receipt Endpoint Product",
		"sku":   "REP-1",
		"stock": 3,
	}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createProductRec.Code, createProductRec.Body.String())
	}
	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product: %v", err)
	}

	createSaleBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
				"price":     100,
			},
		},
	}
	rawSale, _ := json.Marshal(createSaleBody)
	createSaleReq := httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(rawSale))
	createSaleReq.Header.Set("Authorization", "Bearer "+token)
	createSaleRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createSaleRec, createSaleReq)
	if createSaleRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createSaleRec.Code, createSaleRec.Body.String())
	}
	var sale app.Sale
	if err := json.Unmarshal(createSaleRec.Body.Bytes(), &sale); err != nil {
		t.Fatalf("cannot parse sale: %v", err)
	}

	listReceiptsReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/receipts?saleId=%d", sale.ID), nil)
	listReceiptsReq.Header.Set("Authorization", "Bearer "+token)
	listReceiptsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listReceiptsRec, listReceiptsReq)
	if listReceiptsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", listReceiptsRec.Code, listReceiptsRec.Body.String())
	}
	var receipts []app.Receipt
	if err := json.Unmarshal(listReceiptsRec.Body.Bytes(), &receipts); err != nil {
		t.Fatalf("cannot parse receipts list: %v", err)
	}
	if len(receipts) != 1 {
		t.Fatalf("expected one receipt, got %d", len(receipts))
	}
	if receipts[0].Status != app.ReceiptStatusFailed {
		t.Fatalf("expected failed receipt status after auto-attempt, got %s", receipts[0].Status)
	}
	if receipts[0].ErrorMessage == "" {
		t.Fatalf("expected error message on failed receipt")
	}

	getReceiptReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/receipts/%d", receipts[0].ID), nil)
	getReceiptReq.Header.Set("Authorization", "Bearer "+token)
	getReceiptRec := httptest.NewRecorder()
	server.Router().ServeHTTP(getReceiptRec, getReceiptReq)
	if getReceiptRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", getReceiptRec.Code, getReceiptRec.Body.String())
	}

	store.SetReceiptSender(successfulHTTPReceiptSender{})

	retryReceiptReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/receipts/%d/retry", receipts[0].ID),
		bytes.NewReader([]byte(`{}`)),
	)
	retryReceiptReq.Header.Set("Authorization", "Bearer "+token)
	retryReceiptRec := httptest.NewRecorder()
	server.Router().ServeHTTP(retryReceiptRec, retryReceiptReq)
	if retryReceiptRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on retry, got %d; body=%s", retryReceiptRec.Code, retryReceiptRec.Body.String())
	}
	var retried app.Receipt
	if err := json.Unmarshal(retryReceiptRec.Body.Bytes(), &retried); err != nil {
		t.Fatalf("cannot parse retried receipt: %v", err)
	}
	if retried.Status != app.ReceiptStatusSent {
		t.Fatalf("expected sent status after retry, got %s", retried.Status)
	}
	if retried.ExternalID == "" || retried.FiscalNumber == "" || retried.QRURL == "" {
		t.Fatalf("expected sender payload in receipt after retry")
	}

	store.SetReceiptSender(failingHTTPReceiptSender{})
	createSecondSaleReq := httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(rawSale))
	createSecondSaleReq.Header.Set("Authorization", "Bearer "+token)
	createSecondSaleRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createSecondSaleRec, createSecondSaleReq)
	if createSecondSaleRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for second sale, got %d; body=%s", createSecondSaleRec.Code, createSecondSaleRec.Body.String())
	}

	store.SetReceiptSender(successfulHTTPReceiptSender{})
	bulkRetryReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/receipts/retry-bulk",
		bytes.NewReader([]byte(`{"status":"failed","limit":10}`)),
	)
	bulkRetryReq.Header.Set("Authorization", "Bearer "+token)
	bulkRetryRec := httptest.NewRecorder()
	server.Router().ServeHTTP(bulkRetryRec, bulkRetryReq)
	if bulkRetryRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for bulk retry, got %d; body=%s", bulkRetryRec.Code, bulkRetryRec.Body.String())
	}
	var bulkResult app.ReceiptBulkRetryResult
	if err := json.Unmarshal(bulkRetryRec.Body.Bytes(), &bulkResult); err != nil {
		t.Fatalf("cannot parse bulk retry response: %v", err)
	}
	if bulkResult.Attempted == 0 {
		t.Fatalf("expected bulk retry to process failed receipts")
	}
	if bulkResult.Succeeded == 0 {
		t.Fatalf("expected bulk retry to succeed for at least one receipt")
	}
	for _, item := range bulkResult.Items {
		if item.Status != app.ReceiptStatusSent {
			t.Fatalf("expected sent status in bulk retry items, got %s", item.Status)
		}
	}

	store.SetReceiptSender(failingHTTPReceiptSender{})
	createThirdSaleReq := httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(rawSale))
	createThirdSaleReq.Header.Set("Authorization", "Bearer "+token)
	createThirdSaleRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createThirdSaleRec, createThirdSaleReq)
	if createThirdSaleRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for third sale, got %d; body=%s", createThirdSaleRec.Code, createThirdSaleRec.Body.String())
	}
	var thirdSale app.Sale
	if err := json.Unmarshal(createThirdSaleRec.Body.Bytes(), &thirdSale); err != nil {
		t.Fatalf("cannot parse third sale response: %v", err)
	}

	enqueueReceiptJobReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs/receipts-retry",
		bytes.NewReader([]byte(`{"status":"failed","limit":10}`)),
	)
	enqueueReceiptJobReq.Header.Set("Authorization", "Bearer "+token)
	enqueueReceiptJobRec := httptest.NewRecorder()
	server.Router().ServeHTTP(enqueueReceiptJobRec, enqueueReceiptJobReq)
	if enqueueReceiptJobRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for receipt retry job enqueue, got %d; body=%s", enqueueReceiptJobRec.Code, enqueueReceiptJobRec.Body.String())
	}
	var queuedJob app.BackgroundJob
	if err := json.Unmarshal(enqueueReceiptJobRec.Body.Bytes(), &queuedJob); err != nil {
		t.Fatalf("cannot parse queued receipt retry job: %v", err)
	}
	if queuedJob.JobType != app.BackgroundJobTypeReceiptRetries {
		t.Fatalf("expected receipt retry job type, got %s", queuedJob.JobType)
	}

	store.SetReceiptSender(successfulHTTPReceiptSender{})
	runJobsReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/run", bytes.NewReader([]byte(`{}`)))
	runJobsReq.Header.Set("Authorization", "Bearer "+token)
	runJobsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(runJobsRec, runJobsReq)
	if runJobsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for jobs run, got %d; body=%s", runJobsRec.Code, runJobsRec.Body.String())
	}

	thirdSaleReceiptsReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/receipts?saleId=%d", thirdSale.ID), nil)
	thirdSaleReceiptsReq.Header.Set("Authorization", "Bearer "+token)
	thirdSaleReceiptsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(thirdSaleReceiptsRec, thirdSaleReceiptsReq)
	if thirdSaleReceiptsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for third sale receipts, got %d; body=%s", thirdSaleReceiptsRec.Code, thirdSaleReceiptsRec.Body.String())
	}
	var thirdSaleReceipts []app.Receipt
	if err := json.Unmarshal(thirdSaleReceiptsRec.Body.Bytes(), &thirdSaleReceipts); err != nil {
		t.Fatalf("cannot parse third sale receipts: %v", err)
	}
	if len(thirdSaleReceipts) == 0 {
		t.Fatalf("expected third sale receipt to exist")
	}
	if thirdSaleReceipts[0].Status != app.ReceiptStatusSent {
		t.Fatalf("expected sent status after background retry job, got %s", thirdSaleReceipts[0].Status)
	}
}

func TestWarehouseZonesCellsAndCellStocksEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{"name": "Cell API Product", "sku": "CAP-1", "stock": 4}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createProductRec.Code)
	}
	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product: %v", err)
	}

	createWarehouseReq := httptest.NewRequest(http.MethodPost, "/api/v1/warehouses", bytes.NewReader([]byte(`{"name":"Cell WH"}`)))
	createWarehouseReq.Header.Set("Authorization", "Bearer "+token)
	createWarehouseRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createWarehouseRec, createWarehouseReq)
	if createWarehouseRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createWarehouseRec.Code)
	}
	var warehouse app.Warehouse
	if err := json.Unmarshal(createWarehouseRec.Body.Bytes(), &warehouse); err != nil {
		t.Fatalf("cannot parse warehouse: %v", err)
	}

	createZoneReq := httptest.NewRequest(http.MethodPost, "/api/v1/warehouse-zones", bytes.NewReader([]byte(fmt.Sprintf(`{"warehouseId":%d,"name":"B"}`, warehouse.ID))))
	createZoneReq.Header.Set("Authorization", "Bearer "+token)
	createZoneRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createZoneRec, createZoneReq)
	if createZoneRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createZoneRec.Code, createZoneRec.Body.String())
	}
	var zone app.WarehouseZone
	if err := json.Unmarshal(createZoneRec.Body.Bytes(), &zone); err != nil {
		t.Fatalf("cannot parse zone: %v", err)
	}

	createCellReq := httptest.NewRequest(http.MethodPost, "/api/v1/warehouse-cells", bytes.NewReader([]byte(fmt.Sprintf(`{"warehouseId":%d,"zoneId":%d,"code":"B-01"}`, warehouse.ID, zone.ID))))
	createCellReq.Header.Set("Authorization", "Bearer "+token)
	createCellRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createCellRec, createCellReq)
	if createCellRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createCellRec.Code)
	}
	var customCell app.WarehouseCell
	if err := json.Unmarshal(createCellRec.Body.Bytes(), &customCell); err != nil {
		t.Fatalf("cannot parse cell: %v", err)
	}

	transferBody := map[string]interface{}{
		"fromWarehouseId": 1,
		"toWarehouseId":   warehouse.ID,
		"items":           []map[string]interface{}{{"productId": product.ID, "quantity": 2}},
	}
	rawTransfer, _ := json.Marshal(transferBody)
	createTransferReq := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(rawTransfer))
	createTransferReq.Header.Set("Authorization", "Bearer "+token)
	createTransferRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createTransferRec, createTransferReq)
	if createTransferRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createTransferRec.Code)
	}

	listZonesReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/warehouse-zones?warehouseId=%d", warehouse.ID), nil)
	listZonesReq.Header.Set("Authorization", "Bearer "+token)
	listZonesRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listZonesRec, listZonesReq)
	if listZonesRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listZonesRec.Code)
	}
	var zones []app.WarehouseZone
	if err := json.Unmarshal(listZonesRec.Body.Bytes(), &zones); err != nil {
		t.Fatalf("cannot parse zones: %v", err)
	}
	var defaultZoneID int64
	for _, zoneItem := range zones {
		if zoneItem.Name == "DEFAULT" {
			defaultZoneID = zoneItem.ID
			break
		}
	}
	if defaultZoneID == 0 {
		t.Fatalf("expected default zone")
	}

	listCellsReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/warehouse-cells?zoneId=%d", defaultZoneID), nil)
	listCellsReq.Header.Set("Authorization", "Bearer "+token)
	listCellsRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listCellsRec, listCellsReq)
	if listCellsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listCellsRec.Code)
	}
	var defaultCells []app.WarehouseCell
	if err := json.Unmarshal(listCellsRec.Body.Bytes(), &defaultCells); err != nil {
		t.Fatalf("cannot parse cells: %v", err)
	}
	var mainCellID int64
	for _, cellItem := range defaultCells {
		if cellItem.Code == "MAIN" {
			mainCellID = cellItem.ID
			break
		}
	}
	if mainCellID == 0 {
		t.Fatalf("expected main cell")
	}

	cellTransferBody := map[string]interface{}{
		"fromWarehouseId": warehouse.ID,
		"toWarehouseId":   warehouse.ID,
		"items": []map[string]interface{}{
			{
				"productId":  product.ID,
				"quantity":   1,
				"fromCellId": mainCellID,
				"toCellId":   customCell.ID,
			},
		},
	}
	rawCellTransfer, _ := json.Marshal(cellTransferBody)
	cellTransferReq := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(rawCellTransfer))
	cellTransferReq.Header.Set("Authorization", "Bearer "+token)
	cellTransferRec := httptest.NewRecorder()
	server.Router().ServeHTTP(cellTransferRec, cellTransferReq)
	if cellTransferRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for cell transfer, got %d; body=%s", cellTransferRec.Code, cellTransferRec.Body.String())
	}

	cellEndpointBody := map[string]interface{}{
		"fromCellId": customCell.ID,
		"toCellId":   mainCellID,
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
			},
		},
	}
	rawCellEndpoint, _ := json.Marshal(cellEndpointBody)
	cellEndpointReq := httptest.NewRequest(http.MethodPost, "/api/v1/transfers/cell", bytes.NewReader(rawCellEndpoint))
	cellEndpointReq.Header.Set("Authorization", "Bearer "+token)
	cellEndpointRec := httptest.NewRecorder()
	server.Router().ServeHTTP(cellEndpointRec, cellEndpointReq)
	if cellEndpointRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for /transfers/cell, got %d; body=%s", cellEndpointRec.Code, cellEndpointRec.Body.String())
	}

	fifoBody := map[string]interface{}{
		"toCellId": customCell.ID,
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
			},
		},
	}
	rawFIFO, _ := json.Marshal(fifoBody)
	fifoReq := httptest.NewRequest(http.MethodPost, "/api/v1/transfers/cell/fifo", bytes.NewReader(rawFIFO))
	fifoReq.Header.Set("Authorization", "Bearer "+token)
	fifoRec := httptest.NewRecorder()
	server.Router().ServeHTTP(fifoRec, fifoReq)
	if fifoRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for /transfers/cell/fifo, got %d; body=%s", fifoRec.Code, fifoRec.Body.String())
	}
	var fifoTransfer app.StockTransfer
	if err := json.Unmarshal(fifoRec.Body.Bytes(), &fifoTransfer); err != nil {
		t.Fatalf("cannot parse fifo transfer: %v", err)
	}
	if len(fifoTransfer.Items) == 0 || fifoTransfer.Items[0].FromCellID == nil {
		t.Fatalf("expected resolved fifo source cell")
	}

	cellStocksReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/warehouse-cells/stocks?productId=%d", product.ID), nil)
	cellStocksReq.Header.Set("Authorization", "Bearer "+token)
	cellStocksRec := httptest.NewRecorder()
	server.Router().ServeHTTP(cellStocksRec, cellStocksReq)
	if cellStocksRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", cellStocksRec.Code)
	}
}

func TestDocumentsTemplatesAndPDFEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	productBody := map[string]interface{}{"name": "Doc API Product", "sku": "DAP-1", "stock": 1}
	rawProduct, _ := json.Marshal(productBody)
	createProductReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(rawProduct))
	createProductReq.Header.Set("Authorization", "Bearer "+token)
	createProductRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createProductRec, createProductReq)
	if createProductRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createProductRec.Code, createProductRec.Body.String())
	}
	var product app.Product
	if err := json.Unmarshal(createProductRec.Body.Bytes(), &product); err != nil {
		t.Fatalf("cannot parse product: %v", err)
	}

	createDocBody := map[string]interface{}{
		"type":        app.DocumentTypeInvoice,
		"warehouseId": 1,
		"currency":    "UAH",
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
				"price":     200,
			},
		},
		"note": "doc endpoint test",
	}
	rawDoc, _ := json.Marshal(createDocBody)
	createDocReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents", bytes.NewReader(rawDoc))
	createDocReq.Header.Set("Authorization", "Bearer "+token)
	createDocRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createDocRec, createDocReq)
	if createDocRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", createDocRec.Code, createDocRec.Body.String())
	}
	var document app.Document
	if err := json.Unmarshal(createDocRec.Body.Bytes(), &document); err != nil {
		t.Fatalf("cannot parse document: %v", err)
	}

	postReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/documents/%d/post", document.ID), bytes.NewReader([]byte(`{}`)))
	postReq.Header.Set("Authorization", "Bearer "+token)
	postRec := httptest.NewRecorder()
	server.Router().ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", postRec.Code, postRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/documents?type=invoice&status=posted", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", listRec.Code, listRec.Body.String())
	}

	sale, err := store.CreateSale([]app.SaleItem{
		{
			ProductID: product.ID,
			Quantity:  1,
			Price:     200,
		},
	})
	if err != nil {
		t.Fatalf("expected create sale for return flow: %v", err)
	}

	customerAvailabilityReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/documents/returns/customer/%d/available", sale.ID), nil)
	customerAvailabilityReq.Header.Set("Authorization", "Bearer "+token)
	customerAvailabilityRec := httptest.NewRecorder()
	server.Router().ServeHTTP(customerAvailabilityRec, customerAvailabilityReq)
	if customerAvailabilityRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for customer return availability, got %d; body=%s", customerAvailabilityRec.Code, customerAvailabilityRec.Body.String())
	}

	createCustomerReturnBody := map[string]interface{}{
		"saleId":      sale.ID,
		"warehouseId": 1,
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
			},
		},
		"note": "customer return via endpoint",
	}
	rawCreateCustomerReturn, _ := json.Marshal(createCustomerReturnBody)
	createCustomerReturnReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents/returns/customer", bytes.NewReader(rawCreateCustomerReturn))
	createCustomerReturnReq.Header.Set("Authorization", "Bearer "+token)
	createCustomerReturnRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createCustomerReturnRec, createCustomerReturnReq)
	if createCustomerReturnRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for customer return create, got %d; body=%s", createCustomerReturnRec.Code, createCustomerReturnRec.Body.String())
	}
	var customerReturnDoc app.Document
	if err := json.Unmarshal(createCustomerReturnRec.Body.Bytes(), &customerReturnDoc); err != nil {
		t.Fatalf("cannot parse customer return document: %v", err)
	}
	if customerReturnDoc.SourceSaleID == nil || *customerReturnDoc.SourceSaleID != sale.ID {
		t.Fatalf("expected source sale id in customer return doc")
	}

	supplier, err := store.CreateSupplier(app.Supplier{Name: "Return API Supplier"})
	if err != nil {
		t.Fatalf("expected supplier create for return flow: %v", err)
	}
	purchase, err := store.CreatePurchase(app.Purchase{
		SupplierID: supplier.ID,
		Currency:   "UAH",
		Items: []app.PurchaseItem{
			{
				ProductID: product.ID,
				Quantity:  1,
				Price:     200,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected purchase create for return flow: %v", err)
	}

	supplierAvailabilityReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/documents/returns/supplier/%d/available", purchase.ID), nil)
	supplierAvailabilityReq.Header.Set("Authorization", "Bearer "+token)
	supplierAvailabilityRec := httptest.NewRecorder()
	server.Router().ServeHTTP(supplierAvailabilityRec, supplierAvailabilityReq)
	if supplierAvailabilityRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for supplier return availability, got %d; body=%s", supplierAvailabilityRec.Code, supplierAvailabilityRec.Body.String())
	}

	createSupplierReturnBody := map[string]interface{}{
		"purchaseId":  purchase.ID,
		"warehouseId": 1,
		"items": []map[string]interface{}{
			{
				"productId": product.ID,
				"quantity":  1,
			},
		},
		"note": "supplier return via endpoint",
	}
	rawCreateSupplierReturn, _ := json.Marshal(createSupplierReturnBody)
	createSupplierReturnReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents/returns/supplier", bytes.NewReader(rawCreateSupplierReturn))
	createSupplierReturnReq.Header.Set("Authorization", "Bearer "+token)
	createSupplierReturnRec := httptest.NewRecorder()
	server.Router().ServeHTTP(createSupplierReturnRec, createSupplierReturnReq)
	if createSupplierReturnRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for supplier return create, got %d; body=%s", createSupplierReturnRec.Code, createSupplierReturnRec.Body.String())
	}
	var supplierReturnDoc app.Document
	if err := json.Unmarshal(createSupplierReturnRec.Body.Bytes(), &supplierReturnDoc); err != nil {
		t.Fatalf("cannot parse supplier return document: %v", err)
	}
	if supplierReturnDoc.SourcePurchaseID == nil || *supplierReturnDoc.SourcePurchaseID != purchase.ID {
		t.Fatalf("expected source purchase id in supplier return doc")
	}

	templatesReq := httptest.NewRequest(http.MethodGet, "/api/v1/document-templates", nil)
	templatesReq.Header.Set("Authorization", "Bearer "+token)
	templatesRec := httptest.NewRecorder()
	server.Router().ServeHTTP(templatesRec, templatesReq)
	if templatesRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", templatesRec.Code, templatesRec.Body.String())
	}

	updateTemplateBody := map[string]interface{}{
		"code":     app.DocumentTypeInvoice,
		"name":     "Invoice API Template",
		"body":     "Invoice {{number}} total {{total}}",
		"isActive": true,
	}
	rawTemplate, _ := json.Marshal(updateTemplateBody)
	updateTemplateReq := httptest.NewRequest(http.MethodPut, "/api/v1/document-templates", bytes.NewReader(rawTemplate))
	updateTemplateReq.Header.Set("Authorization", "Bearer "+token)
	updateTemplateRec := httptest.NewRecorder()
	server.Router().ServeHTTP(updateTemplateRec, updateTemplateReq)
	if updateTemplateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", updateTemplateRec.Code, updateTemplateRec.Body.String())
	}

	placeholdersReq := httptest.NewRequest(http.MethodGet, "/api/v1/document-templates/placeholders", nil)
	placeholdersReq.Header.Set("Authorization", "Bearer "+token)
	placeholdersRec := httptest.NewRecorder()
	server.Router().ServeHTTP(placeholdersRec, placeholdersReq)
	if placeholdersRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", placeholdersRec.Code, placeholdersRec.Body.String())
	}

	validateTemplateBody := map[string]interface{}{
		"code":   app.DocumentTypeInvoice,
		"body":   "Invoice {{number}} total {{total}}",
		"strict": true,
	}
	rawValidateTemplate, _ := json.Marshal(validateTemplateBody)
	validateReq := httptest.NewRequest(http.MethodPost, "/api/v1/document-templates/validate", bytes.NewReader(rawValidateTemplate))
	validateReq.Header.Set("Authorization", "Bearer "+token)
	validateRec := httptest.NewRecorder()
	server.Router().ServeHTTP(validateRec, validateReq)
	if validateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", validateRec.Code, validateRec.Body.String())
	}

	strictFailBody := map[string]interface{}{
		"code":   app.DocumentTypeInvoice,
		"body":   "Invoice {{number}}",
		"strict": true,
	}
	rawStrictFail, _ := json.Marshal(strictFailBody)
	strictFailReq := httptest.NewRequest(http.MethodPost, "/api/v1/document-templates/preview", bytes.NewReader(rawStrictFail))
	strictFailReq.Header.Set("Authorization", "Bearer "+token)
	strictFailRec := httptest.NewRecorder()
	server.Router().ServeHTTP(strictFailRec, strictFailReq)
	if strictFailRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d; body=%s", strictFailRec.Code, strictFailRec.Body.String())
	}

	previewTemplateBody := map[string]interface{}{
		"code":       app.DocumentTypeInvoice,
		"body":       "Invoice {{number}} total {{total}} {{currency}} {{items}}",
		"documentId": document.ID,
		"strict":     true,
	}
	rawPreviewTemplate, _ := json.Marshal(previewTemplateBody)
	previewReq := httptest.NewRequest(http.MethodPost, "/api/v1/document-templates/preview", bytes.NewReader(rawPreviewTemplate))
	previewReq.Header.Set("Authorization", "Bearer "+token)
	previewRec := httptest.NewRecorder()
	server.Router().ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", previewRec.Code, previewRec.Body.String())
	}

	pdfReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/documents/%d/pdf", document.ID), nil)
	pdfReq.Header.Set("Authorization", "Bearer "+token)
	pdfRec := httptest.NewRecorder()
	server.Router().ServeHTTP(pdfRec, pdfReq)
	if pdfRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", pdfRec.Code, pdfRec.Body.String())
	}
	if contentType := pdfRec.Header().Get("Content-Type"); contentType != "application/pdf" {
		t.Fatalf("expected pdf content type, got %s", contentType)
	}
	if !bytes.HasPrefix(pdfRec.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("expected PDF output prefix")
	}
}

func TestCashShiftEndpoints(t *testing.T) {
	store := app.NewStore()
	tokens := app.NewTokenManager("test-secret")
	server := NewServer(store, tokens, slog.Default())
	token := loginToken(t, server, "admin", "admin123")

	seedCashReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/cash-operations",
		bytes.NewReader([]byte(`{"cashboxId":1,"type":"incoming","amount":200,"method":"cash","description":"seed"}`)),
	)
	seedCashReq.Header.Set("Authorization", "Bearer "+token)
	seedCashRec := httptest.NewRecorder()
	server.Router().ServeHTTP(seedCashRec, seedCashReq)
	if seedCashRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", seedCashRec.Code, seedCashRec.Body.String())
	}

	openReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/cash-shifts/open",
		bytes.NewReader([]byte(`{"cashboxId":1,"note":"open shift"}`)),
	)
	openReq.Header.Set("Authorization", "Bearer "+token)
	openRec := httptest.NewRecorder()
	server.Router().ServeHTTP(openRec, openReq)
	if openRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body=%s", openRec.Code, openRec.Body.String())
	}
	var shift app.CashShift
	if err := json.Unmarshal(openRec.Body.Bytes(), &shift); err != nil {
		t.Fatalf("cannot parse shift: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/cash-shifts?cashboxId=1&status=open", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	server.Router().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", listRec.Code, listRec.Body.String())
	}

	closeReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/cash-shifts/%d/close", shift.ID),
		bytes.NewReader([]byte(`{"note":"close shift"}`)),
	)
	closeReq.Header.Set("Authorization", "Bearer "+token)
	closeRec := httptest.NewRecorder()
	server.Router().ServeHTTP(closeRec, closeReq)
	if closeRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body=%s", closeRec.Code, closeRec.Body.String())
	}
}
