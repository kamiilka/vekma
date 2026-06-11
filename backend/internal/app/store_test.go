package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type failingSender struct{}

func (f failingSender) Enabled() bool { return true }
func (f failingSender) DefaultRecipient(channel string) string {
	if channel == NotificationChannelEmail {
		return "ops@example.local"
	}
	return "123456"
}
func (f failingSender) Send(channel, recipient, subject, body string) error {
	return errors.New("provider down")
}
func (f failingSender) SendFrom(channel, sender, recipient, subject, body string) error {
	return errors.New("provider down")
}

type successfulReceiptSender struct{}

func (s successfulReceiptSender) Enabled() bool { return true }
func (s successfulReceiptSender) SendSaleReceipt(sale Sale) (ReceiptSendResult, error) {
	return ReceiptSendResult{
		ExternalID:   fmt.Sprintf("ext-%d", sale.ID),
		FiscalNumber: fmt.Sprintf("fiscal-%d", sale.ID),
		QRURL:        fmt.Sprintf("https://qr.local/%d", sale.ID),
	}, nil
}

type failingReceiptSender struct{}

func (f failingReceiptSender) Enabled() bool { return true }
func (f failingReceiptSender) SendSaleReceipt(sale Sale) (ReceiptSendResult, error) {
	return ReceiptSendResult{}, errors.New("checkbox unavailable")
}

func TestCreateSaleReducesStockAndCreatesAnalytics(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:        "Phone Battery",
		SKU:         "BAT-001",
		Stock:       10,
		MinStock:    3,
		RetailPrice: 450,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}

	sale, err := store.CreateSale([]SaleItem{
		{
			ProductID: product.ID,
			Quantity:  2,
			Price:     450,
		},
	})
	if err != nil {
		t.Fatalf("expected sale to be created: %v", err)
	}
	if sale.Total != 900 {
		t.Fatalf("expected total 900, got %v", sale.Total)
	}

	products, err := store.ListProducts()
	if err != nil {
		t.Fatalf("expected products to be listed: %v", err)
	}
	if products[0].Stock != 8 {
		t.Fatalf("expected stock 8, got %d", products[0].Stock)
	}

	saleID := sale.ID
	receipts, err := store.ListReceipts(&saleID, nil)
	if err != nil {
		t.Fatalf("expected receipts to be listed: %v", err)
	}
	if len(receipts) != 1 {
		t.Fatalf("expected one auto-created receipt, got %d", len(receipts))
	}
	if receipts[0].Status != ReceiptStatusPending {
		t.Fatalf("expected pending receipt status, got %s", receipts[0].Status)
	}
	if receipts[0].Provider != ReceiptProviderCheckbox {
		t.Fatalf("expected checkbox receipt provider, got %s", receipts[0].Provider)
	}

	summary, err := store.AnalyticsSummary()
	if err != nil {
		t.Fatalf("expected summary to be calculated: %v", err)
	}
	if summary.Revenue != 900 {
		t.Fatalf("expected revenue 900, got %v", summary.Revenue)
	}
	if summary.SalesCount != 1 {
		t.Fatalf("expected sales count 1, got %d", summary.SalesCount)
	}
}

func TestRetryReceiptFlow(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:        "Receipt Product",
		SKU:         "RCP-1",
		Stock:       4,
		RetailPrice: 120,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}

	sale, err := store.CreateSale([]SaleItem{
		{
			ProductID: product.ID,
			Quantity:  1,
			Price:     product.RetailPrice,
		},
	})
	if err != nil {
		t.Fatalf("expected sale to be created: %v", err)
	}

	receipt, err := store.ReceiptBySaleID(sale.ID)
	if err != nil {
		t.Fatalf("expected receipt for sale: %v", err)
	}

	if _, err := store.RetryReceipt(receipt.ID); !errors.Is(err, ErrReceiptSenderNotConfigured) {
		t.Fatalf("expected sender not configured error, got %v", err)
	}

	store.SetReceiptSender(failingReceiptSender{})
	failedReceipt, err := store.RetryReceipt(receipt.ID)
	if err == nil {
		t.Fatalf("expected retry error from failing sender")
	}
	if failedReceipt.Status != ReceiptStatusFailed {
		t.Fatalf("expected failed receipt status, got %s", failedReceipt.Status)
	}
	if failedReceipt.ErrorMessage == "" {
		t.Fatalf("expected failed receipt error message")
	}

	store.SetReceiptSender(successfulReceiptSender{})
	sentReceipt, err := store.RetryReceipt(receipt.ID)
	if err != nil {
		t.Fatalf("expected successful retry, got %v", err)
	}
	if sentReceipt.Status != ReceiptStatusSent {
		t.Fatalf("expected sent receipt status, got %s", sentReceipt.Status)
	}
	if sentReceipt.ExternalID == "" || sentReceipt.FiscalNumber == "" || sentReceipt.QRURL == "" {
		t.Fatalf("expected external fields from sender, got %+v", sentReceipt)
	}
}

func TestCreateSaleAutoSendsReceiptWhenSenderConfigured(t *testing.T) {
	store := NewStore()
	store.SetReceiptSender(successfulReceiptSender{})

	product, err := store.CreateProduct(Product{
		Name:        "Auto Send Product",
		SKU:         "AUTO-RCP-1",
		Stock:       2,
		RetailPrice: 200,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}

	sale, err := store.CreateSale([]SaleItem{
		{
			ProductID: product.ID,
			Quantity:  1,
			Price:     product.RetailPrice,
		},
	})
	if err != nil {
		t.Fatalf("expected sale to be created: %v", err)
	}

	receipt, err := store.ReceiptBySaleID(sale.ID)
	if err != nil {
		t.Fatalf("expected receipt for sale: %v", err)
	}
	if receipt.Status != ReceiptStatusSent {
		t.Fatalf("expected sent status after auto-send, got %s", receipt.Status)
	}
	if receipt.ExternalID == "" || receipt.FiscalNumber == "" || receipt.QRURL == "" {
		t.Fatalf("expected auto-send fields in receipt, got %+v", receipt)
	}
}

func TestRetryReceiptsBulkRespectsStatusAndLimit(t *testing.T) {
	store := NewStore()

	product, err := store.CreateProduct(Product{
		Name:        "Bulk Retry Product",
		SKU:         "BULK-RCP-1",
		Stock:       10,
		RetailPrice: 150,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}

	for range 3 {
		if _, err := store.CreateSale([]SaleItem{
			{
				ProductID: product.ID,
				Quantity:  1,
				Price:     product.RetailPrice,
			},
		}); err != nil {
			t.Fatalf("expected sale to be created: %v", err)
		}
	}

	pending := ReceiptStatusPending
	pendingReceipts, err := store.ListReceipts(nil, &pending)
	if err != nil {
		t.Fatalf("expected list pending receipts: %v", err)
	}
	if len(pendingReceipts) != 3 {
		t.Fatalf("expected three pending receipts, got %d", len(pendingReceipts))
	}

	store.SetReceiptSender(failingReceiptSender{})
	firstResult, err := store.RetryReceiptsBulk(2, &pending)
	if err != nil {
		t.Fatalf("expected bulk retry result without fatal error: %v", err)
	}
	if firstResult.Attempted != 2 || firstResult.Failed != 2 || firstResult.Succeeded != 0 {
		t.Fatalf("unexpected first bulk result: %+v", firstResult)
	}

	failed := ReceiptStatusFailed
	failedReceipts, err := store.ListReceipts(nil, &failed)
	if err != nil {
		t.Fatalf("expected list failed receipts: %v", err)
	}
	if len(failedReceipts) != 2 {
		t.Fatalf("expected two failed receipts after first bulk, got %d", len(failedReceipts))
	}

	store.SetReceiptSender(successfulReceiptSender{})
	secondResult, err := store.RetryReceiptsBulk(10, &failed)
	if err != nil {
		t.Fatalf("expected successful bulk retry for failed receipts: %v", err)
	}
	if secondResult.Attempted != 2 || secondResult.Succeeded != 2 || secondResult.Failed != 0 {
		t.Fatalf("unexpected second bulk result: %+v", secondResult)
	}
	for _, item := range secondResult.Items {
		if item.Status != ReceiptStatusSent {
			t.Fatalf("expected sent status after successful bulk retry, got %s", item.Status)
		}
	}

	pendingReceipts, err = store.ListReceipts(nil, &pending)
	if err != nil {
		t.Fatalf("expected list pending receipts after bulk retries: %v", err)
	}
	if len(pendingReceipts) != 1 {
		t.Fatalf("expected one pending receipt left, got %d", len(pendingReceipts))
	}
}

func TestProfitabilityReport(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:          "Profit Product",
		SKU:           "PRF-1",
		Stock:         10,
		PurchasePrice: 60,
		RetailPrice:   100,
		Currency:      "UAH",
	})
	if err != nil {
		t.Fatalf("expected create product: %v", err)
	}
	_, err = store.CreateSale([]SaleItem{
		{
			ProductID: product.ID,
			Quantity:  2,
			Price:     100,
		},
	})
	if err != nil {
		t.Fatalf("expected create sale: %v", err)
	}

	report, err := store.ProfitabilityReport()
	if err != nil {
		t.Fatalf("expected profitability report: %v", err)
	}
	if len(report.Items) != 1 {
		t.Fatalf("expected one profitability item, got %d", len(report.Items))
	}
	if report.TotalRevenue < 199.99 || report.TotalRevenue > 200.01 {
		t.Fatalf("expected total revenue 200, got %.2f", report.TotalRevenue)
	}
	if report.TotalCost < 119.99 || report.TotalCost > 120.01 {
		t.Fatalf("expected total cost 120, got %.2f", report.TotalCost)
	}
	if report.TotalProfit < 79.99 || report.TotalProfit > 80.01 {
		t.Fatalf("expected total profit 80, got %.2f", report.TotalProfit)
	}
}

func TestCustomersAndRemindersFlow(t *testing.T) {
	store := NewStore()
	customer, err := store.CreateCustomer(Customer{
		Name:    "Test Customer",
		Phone:   "+380501234567",
		Email:   "test@example.com",
		Comment: "vip",
	})
	if err != nil {
		t.Fatalf("expected create customer: %v", err)
	}

	customers, err := store.ListCustomers("test")
	if err != nil {
		t.Fatalf("expected list customers: %v", err)
	}
	if len(customers) != 1 || customers[0].ID != customer.ID {
		t.Fatalf("expected one customer from search")
	}

	reminder, err := store.CreateCustomerReminder(CustomerReminder{
		CustomerID: customer.ID,
		Text:       "Call back",
	})
	if err != nil {
		t.Fatalf("expected create reminder: %v", err)
	}
	if reminder.Status != CustomerReminderStatusPending {
		t.Fatalf("expected pending reminder status, got %s", reminder.Status)
	}

	reminders, err := store.ListCustomerReminders(&customer.ID, nil, false)
	if err != nil {
		t.Fatalf("expected list reminders: %v", err)
	}
	if len(reminders) != 1 {
		t.Fatalf("expected one reminder, got %d", len(reminders))
	}

	completed, err := store.CompleteCustomerReminder(reminder.ID)
	if err != nil {
		t.Fatalf("expected complete reminder: %v", err)
	}
	if completed.Status != CustomerReminderStatusDone {
		t.Fatalf("expected done status after completion, got %s", completed.Status)
	}
}

func TestServiceOrdersFlow(t *testing.T) {
	store := NewStore()
	customer, err := store.CreateCustomer(Customer{Name: "Repair Customer"})
	if err != nil {
		t.Fatalf("expected create customer: %v", err)
	}
	product, err := store.CreateProduct(Product{
		Name:        "Broken Phone",
		SKU:         "REPAIR-1",
		Stock:       1,
		RetailPrice: 1000,
		Currency:    "UAH",
	})
	if err != nil {
		t.Fatalf("expected create product: %v", err)
	}

	order, err := store.CreateServiceOrder(ServiceOrder{
		CustomerID:  customer.ID,
		ProductID:   &product.ID,
		Title:       "Display replacement",
		Description: "Need to replace cracked screen",
		Technician:  "Ivan",
		LaborMin:    40,
		Price:       1200,
		Currency:    "UAH",
	})
	if err != nil {
		t.Fatalf("expected create service order: %v", err)
	}
	if order.Status != ServiceOrderStatusNew {
		t.Fatalf("expected new status, got %s", order.Status)
	}
	if order.Technician != "Ivan" || order.LaborMin != 40 {
		t.Fatalf("expected technician and labor fields to be stored")
	}

	updated, err := store.UpdateServiceOrderDetails(order.ID, ServiceOrder{
		ProductID:   &product.ID,
		Title:       "Display + battery replacement",
		Description: "Need to replace cracked screen and battery",
		Technician:  "Petro",
		LaborMin:    75,
		Price:       1500,
		Currency:    "UAH",
	})
	if err != nil {
		t.Fatalf("expected update service order details: %v", err)
	}
	if updated.Technician != "Petro" || updated.LaborMin != 75 || updated.Price != 1500 {
		t.Fatalf("unexpected updated service order details: %+v", updated)
	}

	partProduct, err := store.CreateProduct(Product{
		Name:        "Screen module",
		SKU:         "PART-1",
		Stock:       5,
		RetailPrice: 600,
		Currency:    "UAH",
	})
	if err != nil {
		t.Fatalf("expected create part product: %v", err)
	}
	orderWithPart, part, err := store.AddServiceOrderPart(order.ID, ServiceOrderPart{
		ProductID: partProduct.ID,
		Quantity:  2,
		Price:     450,
	})
	if err != nil {
		t.Fatalf("expected add service order part: %v", err)
	}
	if part.Total != 900 {
		t.Fatalf("expected part total 900, got %.2f", part.Total)
	}
	if orderWithPart.PartsTotal != 900 || len(orderWithPart.Parts) != 1 {
		t.Fatalf("expected order parts totals to be updated: %+v", orderWithPart)
	}
	payment, err := store.CreatePayment(Payment{
		ServiceOrderID: &order.ID,
		Amount:         1000,
		Currency:       "UAH",
		Method:         PaymentMethodCash,
		Note:           "prepayment",
	})
	if err != nil {
		t.Fatalf("expected create payment for service order: %v", err)
	}
	if payment.ServiceOrderID == nil || *payment.ServiceOrderID != order.ID {
		t.Fatalf("expected serviceOrderId in payment response: %+v", payment)
	}
	products, err := store.ListProductsFiltered("PART-1", true)
	if err != nil {
		t.Fatalf("expected list products after part consume: %v", err)
	}
	if len(products) != 1 || products[0].Stock != 3 {
		t.Fatalf("expected part stock to be decreased to 3, got %+v", products)
	}

	items, err := store.ListServiceOrders(&customer.ID, nil)
	if err != nil {
		t.Fatalf("expected list service orders: %v", err)
	}
	if len(items) != 1 || items[0].ID != order.ID {
		t.Fatalf("expected one service order in list")
	}
	if items[0].Paid < 999.99 || items[0].Paid > 1000.01 {
		t.Fatalf("expected paid about 1000, got %.2f", items[0].Paid)
	}
	if items[0].Debt < 1399.99 || items[0].Debt > 1400.01 {
		t.Fatalf("expected debt about 1400, got %.2f", items[0].Debt)
	}

	inProgress, err := store.UpdateServiceOrderStatus(order.ID, ServiceOrderStatusInProgress)
	if err != nil {
		t.Fatalf("expected move to in_progress: %v", err)
	}
	if inProgress.Status != ServiceOrderStatusInProgress {
		t.Fatalf("expected in_progress status, got %s", inProgress.Status)
	}

	done, err := store.UpdateServiceOrderStatus(order.ID, ServiceOrderStatusDone)
	if err != nil {
		t.Fatalf("expected move to done: %v", err)
	}
	if done.Status != ServiceOrderStatusDone || done.CompletedAt == nil {
		t.Fatalf("expected done status with completedAt, got %+v", done)
	}
	if _, _, err := store.AddServiceOrderPart(order.ID, ServiceOrderPart{
		ProductID: partProduct.ID,
		Quantity:  1,
		Price:     100,
	}); err == nil {
		t.Fatalf("expected add part to closed order to fail")
	}
	if _, err := store.CreatePayment(Payment{
		ServiceOrderID: &order.ID,
		Amount:         2000,
		Currency:       "UAH",
		Method:         PaymentMethodCash,
		Note:           "overpay attempt",
	}); err == nil {
		t.Fatalf("expected overpayment to fail")
	}
	foundAct, err := store.ServiceOrderActDocument(order.ID)
	if err != nil {
		t.Fatalf("expected auto-created act after done status: %v", err)
	}
	if foundAct.Type != DocumentTypeAct || foundAct.SourceServiceOrderID == nil || *foundAct.SourceServiceOrderID != order.ID {
		t.Fatalf("expected act linked to service order, got %+v", foundAct)
	}
	if foundAct.Status != DocumentStatusPosted {
		t.Fatalf("expected auto-created act to be posted, got %s", foundAct.Status)
	}
	if _, err := store.CreateServiceOrderActDocument(order.ID, "duplicate"); err == nil {
		t.Fatalf("expected duplicate act creation to fail")
	}
	if _, err := store.UpdateServiceOrderStatus(order.ID, ServiceOrderStatusInProgress); err == nil {
		t.Fatalf("expected reopen to fail while act is active")
	}
	if _, err := store.CancelServiceOrderActDocument(order.ID); err != nil {
		t.Fatalf("expected cancel service order act: %v", err)
	}
	reopened, err := store.UpdateServiceOrderStatus(order.ID, ServiceOrderStatusInProgress)
	if err != nil {
		t.Fatalf("expected reopen after act cancel: %v", err)
	}
	if reopened.Status != ServiceOrderStatusInProgress {
		t.Fatalf("expected reopened status in_progress, got %s", reopened.Status)
	}
	if reopened.CompletedAt != nil {
		t.Fatalf("expected completedAt to be cleared on reopen")
	}

	doneAgain, err := store.UpdateServiceOrderStatus(order.ID, ServiceOrderStatusDone)
	if err != nil {
		t.Fatalf("expected move to done again after reopen: %v", err)
	}
	if doneAgain.Status != ServiceOrderStatusDone || doneAgain.CompletedAt == nil {
		t.Fatalf("expected done status with completedAt on second finish, got %+v", doneAgain)
	}
	secondAct, err := store.ServiceOrderActDocument(order.ID)
	if err != nil {
		t.Fatalf("expected auto-created second act after second done: %v", err)
	}
	if secondAct.ID == foundAct.ID {
		t.Fatalf("expected new act id after reopening and finishing again")
	}
	if secondAct.Status != DocumentStatusPosted {
		t.Fatalf("expected second auto-created act to be posted, got %s", secondAct.Status)
	}
	if _, err := store.CancelServiceOrderActDocument(order.ID); err != nil {
		t.Fatalf("expected cancel second service order act: %v", err)
	}
	reopenedAgain, err := store.UpdateServiceOrderStatus(order.ID, ServiceOrderStatusInProgress)
	if err != nil {
		t.Fatalf("expected reopen again after second act cancel: %v", err)
	}
	if reopenedAgain.Status != ServiceOrderStatusInProgress {
		t.Fatalf("expected reopened-again status in_progress, got %s", reopenedAgain.Status)
	}

	cancelled, err := store.UpdateServiceOrderStatus(order.ID, ServiceOrderStatusCancelled)
	if err != nil {
		t.Fatalf("expected cancel after reopen: %v", err)
	}
	if cancelled.Status != ServiceOrderStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelled.Status)
	}
}

func TestCreateMovementFailsOnUnknownProduct(t *testing.T) {
	store := NewStore()

	_, err := store.CreateStockMovement(StockMovement{
		ProductID: 999,
		Type:      "incoming",
		Quantity:  4,
	}, 0)
	if err != ErrProductNotFound {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestImportExportProductsCSV(t *testing.T) {
	store := NewStore()

	csvInput := strings.Join([]string{
		"name,sku,stock,purchasePrice,retailPrice,currency",
		"Keyboard,KB-1,10,200,350,UAH",
		"Mouse,MS-1,8,120,220,UAH",
	}, "\n")
	importRes, err := store.ImportProductsCSV(csvInput, false)
	if err != nil {
		t.Fatalf("expected csv import: %v", err)
	}
	if importRes.Imported != 2 || importRes.Updated != 0 || importRes.Skipped != 0 {
		t.Fatalf("unexpected import result: %+v", importRes)
	}

	updateInput := strings.Join([]string{
		"name,sku,stock,purchasePrice,retailPrice,currency",
		"Keyboard Updated,KB-1,15,250,390,UAH",
	}, "\n")
	updateRes, err := store.ImportProductsCSV(updateInput, true)
	if err != nil {
		t.Fatalf("expected csv update import: %v", err)
	}
	if updateRes.Updated != 1 {
		t.Fatalf("expected one updated row, got %+v", updateRes)
	}

	products, err := store.ListProductsFiltered("KB-1", true)
	if err != nil {
		t.Fatalf("expected products list: %v", err)
	}
	if len(products) != 1 || products[0].Stock != 15 || products[0].RetailPrice != 390 {
		t.Fatalf("expected updated product values, got %+v", products)
	}

	exported, err := store.ExportProductsCSV(true)
	if err != nil {
		t.Fatalf("expected csv export: %v", err)
	}
	if !strings.Contains(exported, "name,code,sku") {
		t.Fatalf("expected csv header in export")
	}
	if !strings.Contains(exported, "KB-1") || !strings.Contains(exported, "MS-1") {
		t.Fatalf("expected exported products in csv output")
	}
}

func TestImportExportProductsXLSX(t *testing.T) {
	store := NewStore()
	_, err := store.CreateProduct(Product{
		Name:        "XLSX Product",
		SKU:         "XLSX-1",
		Stock:       4,
		RetailPrice: 200,
		Currency:    "UAH",
	})
	if err != nil {
		t.Fatalf("expected create product for xlsx export: %v", err)
	}
	content, err := store.ExportProductsXLSX(true)
	if err != nil {
		t.Fatalf("expected xlsx export: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("expected non-empty xlsx export content")
	}

	importStore := NewStore()
	result, err := importStore.ImportProductsXLSX(content, false)
	if err != nil {
		t.Fatalf("expected xlsx import: %v", err)
	}
	if result.Imported == 0 {
		t.Fatalf("expected imported rows from xlsx, got %+v", result)
	}
}

func TestBulkUpdateProductPrices(t *testing.T) {
	store := NewStore()
	_, err := store.CreateProduct(Product{
		Name:        "Bulk A",
		SKU:         "BLK-A",
		Category:    "Phones",
		RetailPrice: 100,
	})
	if err != nil {
		t.Fatalf("expected create product A: %v", err)
	}
	_, err = store.CreateProduct(Product{
		Name:        "Bulk B",
		SKU:         "BLK-B",
		Category:    "Phones",
		RetailPrice: 200,
	})
	if err != nil {
		t.Fatalf("expected create product B: %v", err)
	}

	result, err := store.BulkUpdateProductPrices(ProductPriceBulkUpdateRequest{
		Mode:       "percent",
		Value:      10,
		PriceField: "retailPrice",
		Category:   "Phones",
	})
	if err != nil {
		t.Fatalf("expected bulk update prices: %v", err)
	}
	if result.Updated != 2 {
		t.Fatalf("expected 2 updated products, got %d", result.Updated)
	}

	products, err := store.ListProductsFiltered("BLK-", true)
	if err != nil {
		t.Fatalf("expected products list after bulk update: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected two products, got %d", len(products))
	}
	priceBySKU := map[string]float64{}
	for _, product := range products {
		priceBySKU[product.SKU] = product.RetailPrice
	}
	if priceBySKU["BLK-A"] < 109.99 || priceBySKU["BLK-A"] > 110.01 {
		t.Fatalf("expected BLK-A price 110, got %.2f", priceBySKU["BLK-A"])
	}
	if priceBySKU["BLK-B"] < 219.99 || priceBySKU["BLK-B"] > 220.01 {
		t.Fatalf("expected BLK-B price 220, got %.2f", priceBySKU["BLK-B"])
	}
}

func TestBulkUpdateProductPricesWithRoundingAndBarcodeGeneration(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:        "Rounded Product",
		SKU:         "ROUND-1",
		RetailPrice: 101,
	})
	if err != nil {
		t.Fatalf("expected create product: %v", err)
	}

	result, err := store.BulkUpdateProductPrices(ProductPriceBulkUpdateRequest{
		Mode:       "percent",
		Value:      10,
		PriceField: "retailPrice",
		RoundMode:  "nearest",
		RoundTo:    5,
		Search:     "ROUND-1",
	})
	if err != nil {
		t.Fatalf("expected rounded bulk update: %v", err)
	}
	if result.Updated != 1 {
		t.Fatalf("expected one updated product, got %d", result.Updated)
	}

	products, err := store.ListProductsFiltered("ROUND-1", true)
	if err != nil {
		t.Fatalf("expected products list: %v", err)
	}
	if len(products) != 1 || products[0].RetailPrice != 110 {
		t.Fatalf("expected retail price rounded to 110, got %+v", products)
	}

	updated, err := store.GenerateProductBarcode(product.ID)
	if err != nil {
		t.Fatalf("expected barcode generation: %v", err)
	}
	if len(updated.Barcode) != 13 {
		t.Fatalf("expected EAN-13 barcode, got %q", updated.Barcode)
	}
}

func TestMergeDuplicateProducts(t *testing.T) {
	store := NewStore()
	target, err := store.CreateProduct(Product{
		Name:        "Merge Target",
		SKU:         "MRG-T",
		Stock:       5,
		RetailPrice: 100,
	})
	if err != nil {
		t.Fatalf("expected create target product: %v", err)
	}
	source, err := store.CreateProduct(Product{
		Name:        "Merge Source",
		SKU:         "MRG-S",
		Stock:       3,
		RetailPrice: 100,
	})
	if err != nil {
		t.Fatalf("expected create source product: %v", err)
	}

	_, err = store.CreateSale([]SaleItem{
		{ProductID: source.ID, Quantity: 1, Price: 100},
	})
	if err != nil {
		t.Fatalf("expected create sale for source product: %v", err)
	}

	result, err := store.MergeDuplicateProducts(target.ID, []int64{source.ID})
	if err != nil {
		t.Fatalf("expected merge duplicate products: %v", err)
	}
	if result.TargetProductID != target.ID {
		t.Fatalf("expected target product id %d, got %d", target.ID, result.TargetProductID)
	}
	if len(result.MergedProductIDs) != 1 || result.MergedProductIDs[0] != source.ID {
		t.Fatalf("expected merged source id %d, got %+v", source.ID, result.MergedProductIDs)
	}

	products, err := store.ListProductsFiltered("", true)
	if err != nil {
		t.Fatalf("expected list products after merge: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected one product after merge, got %d", len(products))
	}
	if products[0].ID != target.ID {
		t.Fatalf("expected remaining target product id %d, got %d", target.ID, products[0].ID)
	}
	if products[0].Stock != 7 {
		t.Fatalf("expected merged stock 7, got %d", products[0].Stock)
	}

	sales, err := store.ListSales()
	if err != nil {
		t.Fatalf("expected list sales after merge: %v", err)
	}
	if len(sales) != 1 || len(sales[0].Items) != 1 {
		t.Fatalf("expected one sale item after merge")
	}
	if sales[0].Items[0].ProductID != target.ID {
		t.Fatalf("expected sale item remapped to target product")
	}
}

func TestOrderReservationBlocksOtherSales(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "SSD",
		SKU:   "SSD-001",
		Stock: 10,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}

	_, err = store.CreateOrder(
		"Client A",
		[]SaleItem{
			{
				ProductID: product.ID,
				Quantity:  8,
				Price:     100,
			},
		},
		true,
		nil,
		"UAH",
		nil,
	)
	if err != nil {
		t.Fatalf("expected order to be created: %v", err)
	}

	_, err = store.CreateSale([]SaleItem{
		{
			ProductID: product.ID,
			Quantity:  3,
			Price:     100,
		},
	})
	if err != ErrInsufficientStock {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestCancelOrderReleasesReservations(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Battery",
		SKU:   "BAT-100",
		Stock: 5,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}

	order, err := store.CreateOrder(
		"Client B",
		[]SaleItem{
			{
				ProductID: product.ID,
				Quantity:  4,
				Price:     50,
			},
		},
		true,
		nil,
		"UAH",
		nil,
	)
	if err != nil {
		t.Fatalf("expected order to be created: %v", err)
	}

	if err := store.UpdateOrderStatus(order.ID, OrderStatusCancelled); err != nil {
		t.Fatalf("expected order status update: %v", err)
	}

	_, err = store.CreateSale([]SaleItem{
		{
			ProductID: product.ID,
			Quantity:  4,
			Price:     50,
		},
	})
	if err != nil {
		t.Fatalf("expected sale to succeed after releasing reservation: %v", err)
	}
}

func TestReservationExpiryJobReleasesExpiredReservations(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Expiring Battery",
		SKU:   "EXP-BAT",
		Stock: 5,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}
	expiresAt := time.Now().UTC().Add(-time.Hour)

	_, err = store.CreateOrder(
		"Reservation Client",
		[]SaleItem{{ProductID: product.ID, Quantity: 5, Price: 10}},
		true,
		&expiresAt,
		"UAH",
		nil,
	)
	if err != nil {
		t.Fatalf("expected order to be created: %v", err)
	}
	if _, err := store.CreateSale([]SaleItem{{ProductID: product.ID, Quantity: 1, Price: 10}}); err != ErrInsufficientStock {
		t.Fatalf("expected reservation to block sale before expiry job, got %v", err)
	}

	job, err := store.EnqueueReservationExpiryJob(time.Now().UTC())
	if err != nil {
		t.Fatalf("expected enqueue reservation expiry job: %v", err)
	}
	processed, err := store.RunDueBackgroundJobs()
	if err != nil {
		t.Fatalf("expected run due jobs: %v", err)
	}
	if len(processed) != 1 || processed[0].ID != job.ID || processed[0].Status != BackgroundJobStatusCompleted {
		t.Fatalf("expected completed reservation expiry job, got %+v", processed)
	}

	reservations, err := store.ListReservations(ReservationStatusExpired)
	if err != nil {
		t.Fatalf("expected list expired reservations: %v", err)
	}
	if len(reservations) != 1 {
		t.Fatalf("expected one expired reservation, got %d", len(reservations))
	}
	if _, err := store.CreateSale([]SaleItem{{ProductID: product.ID, Quantity: 1, Price: 10}}); err != nil {
		t.Fatalf("expected sale after expired reservation release: %v", err)
	}
}

func TestPartialPaymentCreatesDebtAndThenClosesIt(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Router",
		SKU:   "RT-1",
		Stock: 4,
	})
	if err != nil {
		t.Fatalf("expected product to be created: %v", err)
	}

	order, err := store.CreateOrder(
		"Client Debt",
		[]SaleItem{
			{
				ProductID: product.ID,
				Quantity:  2,
				Price:     100,
			},
		},
		false,
		nil,
		"UAH",
		nil,
	)
	if err != nil {
		t.Fatalf("expected order to be created: %v", err)
	}

	_, err = store.CreatePayment(Payment{
		OrderID:  &order.ID,
		Amount:   80,
		Currency: "UAH",
		Method:   "cash",
	})
	if err != nil {
		t.Fatalf("expected payment to be created: %v", err)
	}

	debts, err := store.ListDebts()
	if err != nil {
		t.Fatalf("expected debts to be listed: %v", err)
	}
	if len(debts) != 1 {
		t.Fatalf("expected one debt, got %d", len(debts))
	}
	if debts[0].Debt != 120 {
		t.Fatalf("expected debt 120, got %v", debts[0].Debt)
	}

	_, err = store.CreatePayment(Payment{
		OrderID:  &order.ID,
		Amount:   120,
		Currency: "UAH",
		Method:   "card",
	})
	if err != nil {
		t.Fatalf("expected second payment to be created: %v", err)
	}

	debts, err = store.ListDebts()
	if err != nil {
		t.Fatalf("expected debts to be listed: %v", err)
	}
	if len(debts) != 0 {
		t.Fatalf("expected no debts after full payment, got %d", len(debts))
	}
}

func TestPaymentCreatesCashOperationAndUpdatesBalance(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Switch",
		SKU:   "SW-1",
		Stock: 3,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	order, err := store.CreateOrder(
		"Cash Client",
		[]SaleItem{
			{
				ProductID: product.ID,
				Quantity:  1,
				Price:     200,
			},
		},
		false,
		nil,
		"UAH",
		nil,
	)
	if err != nil {
		t.Fatalf("expected order create: %v", err)
	}

	payment, err := store.CreatePayment(Payment{
		OrderID:  &order.ID,
		Amount:   50,
		Currency: "UAH",
		Method:   PaymentMethodCash,
	})
	if err != nil {
		t.Fatalf("expected payment create: %v", err)
	}
	if payment.CashboxID == 0 {
		t.Fatalf("expected cashbox id to be resolved")
	}

	cashboxes, err := store.ListCashboxes()
	if err != nil {
		t.Fatalf("expected cashboxes list: %v", err)
	}
	found := false
	for _, cashbox := range cashboxes {
		if cashbox.ID == payment.CashboxID {
			found = true
			if cashbox.Balance != 50 {
				t.Fatalf("expected cashbox balance 50, got %v", cashbox.Balance)
			}
		}
	}
	if !found {
		t.Fatalf("expected cashbox for payment")
	}

	operations, err := store.ListCashOperations(&payment.CashboxID)
	if err != nil {
		t.Fatalf("expected cash operations list: %v", err)
	}
	if len(operations) == 0 {
		t.Fatalf("expected at least one cash operation")
	}
}

func TestOverdueDebtAndPaymentHistory(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Overdue Product",
		SKU:   "OVD-1",
		Stock: 5,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	pastDue := time.Now().UTC().Add(-48 * time.Hour)
	order, err := store.CreateOrder(
		"Overdue Client",
		[]SaleItem{
			{
				ProductID: product.ID,
				Quantity:  2,
				Price:     100,
			},
		},
		false,
		nil,
		"USD",
		&pastDue,
	)
	if err != nil {
		t.Fatalf("expected order create: %v", err)
	}

	_, err = store.CreatePayment(Payment{
		OrderID:  &order.ID,
		Amount:   50,
		Currency: "UAH",
		Method:   PaymentMethodCard,
	})
	if err != nil {
		t.Fatalf("expected payment create: %v", err)
	}

	overdue, err := store.ListOverdueDebts(time.Now().UTC())
	if err != nil {
		t.Fatalf("expected overdue debts: %v", err)
	}
	if len(overdue) == 0 || !overdue[0].IsOverdue {
		t.Fatalf("expected overdue debt entry")
	}

	history, err := store.DebtPaymentHistory("order", order.ID)
	if err != nil {
		t.Fatalf("expected payment history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected one payment history row, got %d", len(history))
	}
	if history[0].RemainingDebt <= 0 {
		t.Fatalf("expected positive remaining debt")
	}
}

func TestOverdueReminderBackgroundJobCreatesNotifications(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Reminder Product",
		SKU:   "REM-1",
		Stock: 3,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	due := time.Now().UTC().Add(-24 * time.Hour)
	_, err = store.CreateOrder(
		"Reminder Client",
		[]SaleItem{
			{
				ProductID: product.ID,
				Quantity:  1,
				Price:     100,
			},
		},
		false,
		nil,
		"UAH",
		&due,
	)
	if err != nil {
		t.Fatalf("expected order create: %v", err)
	}

	_, err = store.EnqueueOverdueReminderJob(time.Now().UTC())
	if err != nil {
		t.Fatalf("expected enqueue job: %v", err)
	}

	processed, err := store.RunDueBackgroundJobs()
	if err != nil {
		t.Fatalf("expected run jobs: %v", err)
	}
	if len(processed) == 0 {
		t.Fatalf("expected processed jobs")
	}

	notifications, err := store.ListNotifications(20)
	if err != nil {
		t.Fatalf("expected notifications list: %v", err)
	}
	if len(notifications) == 0 {
		t.Fatalf("expected generated notifications")
	}
}

func TestOverdueReminderJobRetryBackoffOnFailure(t *testing.T) {
	store := NewStore()
	store.SetNotificationSender(failingSender{})

	product, err := store.CreateProduct(Product{
		Name:  "Retry Product",
		SKU:   "RETRY-1",
		Stock: 2,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	due := time.Now().UTC().Add(-24 * time.Hour)
	_, err = store.CreateOrder(
		"Retry Client",
		[]SaleItem{{ProductID: product.ID, Quantity: 1, Price: 100}},
		false,
		nil,
		"UAH",
		&due,
	)
	if err != nil {
		t.Fatalf("expected order create: %v", err)
	}

	_, err = store.EnqueueOverdueReminderJob(time.Now().UTC())
	if err != nil {
		t.Fatalf("expected enqueue job: %v", err)
	}

	processed, err := store.RunDueBackgroundJobs()
	if err != nil {
		t.Fatalf("expected run jobs: %v", err)
	}
	if len(processed) != 1 {
		t.Fatalf("expected one processed job, got %d", len(processed))
	}
	if processed[0].Status != BackgroundJobStatusFailed {
		t.Fatalf("expected failed job status, got %s", processed[0].Status)
	}
	if processed[0].NextRetryAt == nil {
		t.Fatalf("expected nextRetryAt for failed job")
	}
}

func TestReceiptRetryBackgroundJobBackoffAndRecovery(t *testing.T) {
	store := NewStore()
	store.SetReceiptSender(failingReceiptSender{})

	product, err := store.CreateProduct(Product{
		Name:        "Receipt Job Product",
		SKU:         "RJOB-1",
		Stock:       3,
		RetailPrice: 100,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	sale, err := store.CreateSale([]SaleItem{
		{
			ProductID: product.ID,
			Quantity:  1,
			Price:     100,
		},
	})
	if err != nil {
		t.Fatalf("expected sale create: %v", err)
	}

	initialReceipt, err := store.ReceiptBySaleID(sale.ID)
	if err != nil {
		t.Fatalf("expected receipt by sale: %v", err)
	}
	if initialReceipt.Status != ReceiptStatusFailed {
		t.Fatalf("expected failed receipt after sender error, got %s", initialReceipt.Status)
	}

	job, err := store.EnqueueReceiptRetryJob(ReceiptStatusFailed, 10)
	if err != nil {
		t.Fatalf("expected enqueue receipt retry job: %v", err)
	}
	if job.JobType != BackgroundJobTypeReceiptRetries {
		t.Fatalf("expected receipt retry job type, got %s", job.JobType)
	}

	processed, err := store.RunDueBackgroundJobs()
	if err != nil {
		t.Fatalf("expected run due jobs: %v", err)
	}
	if len(processed) == 0 {
		t.Fatalf("expected at least one processed job")
	}
	var failedJob *BackgroundJob
	for i := range processed {
		if processed[i].JobType == BackgroundJobTypeReceiptRetries {
			failedJob = &processed[i]
			break
		}
	}
	if failedJob == nil {
		t.Fatalf("expected processed receipt retry job")
	}
	if failedJob.Status != BackgroundJobStatusFailed {
		t.Fatalf("expected failed status for receipt retry job, got %s", failedJob.Status)
	}
	if failedJob.NextRetryAt == nil {
		t.Fatalf("expected nextRetryAt for failed receipt retry job")
	}

	store.SetReceiptSender(successfulReceiptSender{})
	store.mu.Lock()
	for index := range store.backgroundJobs {
		if store.backgroundJobs[index].ID == failedJob.ID {
			retryAt := time.Now().UTC().Add(-time.Minute)
			store.backgroundJobs[index].Status = BackgroundJobStatusFailed
			store.backgroundJobs[index].NextRetryAt = &retryAt
		}
	}
	store.mu.Unlock()

	processed, err = store.RunDueBackgroundJobs()
	if err != nil {
		t.Fatalf("expected run due jobs after recovery: %v", err)
	}
	var completedJob *BackgroundJob
	for i := range processed {
		if processed[i].ID == failedJob.ID {
			completedJob = &processed[i]
			break
		}
	}
	if completedJob == nil {
		t.Fatalf("expected recovered receipt retry job in processed list")
	}
	if completedJob.Status != BackgroundJobStatusCompleted {
		t.Fatalf("expected completed status after recovery, got %s", completedJob.Status)
	}

	finalReceipt, err := store.ReceiptBySaleID(sale.ID)
	if err != nil {
		t.Fatalf("expected receipt by sale after recovery: %v", err)
	}
	if finalReceipt.Status != ReceiptStatusSent {
		t.Fatalf("expected sent receipt status after recovery, got %s", finalReceipt.Status)
	}
}

func TestSupplierOrderPurchaseFlowUpdatesStockAndStatus(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Supply Product",
		SKU:   "SUP-1",
		Stock: 1,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	supplier, err := store.CreateSupplier(Supplier{
		Name: "Main Supplier",
	})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}

	order, err := store.CreateSupplierOrder(
		supplier.ID,
		[]PurchaseItem{
			{
				ProductID: product.ID,
				Quantity:  5,
				Price:     40,
			},
		},
		"UAH",
	)
	if err != nil {
		t.Fatalf("expected supplier order create: %v", err)
	}

	_, err = store.CreatePurchase(Purchase{
		SupplierID:      supplier.ID,
		SupplierOrderID: &order.ID,
		Currency:        "UAH",
		Items: []PurchaseItem{
			{
				ProductID: product.ID,
				Quantity:  2,
				Price:     40,
			},
		},
		Note: "first receipt",
	})
	if err != nil {
		t.Fatalf("expected first purchase create: %v", err)
	}

	orders, err := store.ListSupplierOrders()
	if err != nil {
		t.Fatalf("expected supplier orders list: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected one supplier order")
	}
	if orders[0].Status != SupplierOrderStatusPartiallyReceived {
		t.Fatalf("expected partially_received status, got %s", orders[0].Status)
	}

	_, err = store.CreatePurchase(Purchase{
		SupplierID:      supplier.ID,
		SupplierOrderID: &order.ID,
		Currency:        "UAH",
		Items: []PurchaseItem{
			{
				ProductID: product.ID,
				Quantity:  3,
				Price:     40,
			},
		},
		Note: "second receipt",
	})
	if err != nil {
		t.Fatalf("expected second purchase create: %v", err)
	}

	orders, err = store.ListSupplierOrders()
	if err != nil {
		t.Fatalf("expected supplier orders list: %v", err)
	}
	if orders[0].Status != SupplierOrderStatusReceived {
		t.Fatalf("expected received status, got %s", orders[0].Status)
	}

	products, err := store.ListProducts()
	if err != nil {
		t.Fatalf("expected products list: %v", err)
	}
	if products[0].Stock != 6 {
		t.Fatalf("expected stock 6 after receipts, got %d", products[0].Stock)
	}
}

func TestImportPurchaseCSVCreatesPurchaseAndUpdatesStock(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "CSV Purchase Product",
		SKU:   "CSV-PUR-1",
		Stock: 1,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}
	supplier, err := store.CreateSupplier(Supplier{Name: "CSV Supplier"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}

	result, err := store.ImportPurchaseCSV(Purchase{
		SupplierID: supplier.ID,
		Currency:   "UAH",
		Note:       "csv receipt",
	}, fmt.Sprintf("productId,quantity,price\n%d,4,25\nbad,1,10", product.ID))
	if err != nil {
		t.Fatalf("expected purchase csv import: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 1 || len(result.Errors) != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	products, err := store.ListProducts()
	if err != nil {
		t.Fatalf("expected list products: %v", err)
	}
	if products[0].Stock != 5 {
		t.Fatalf("expected stock 5 after purchase csv import, got %d", products[0].Stock)
	}
}

func TestImportDocumentCSVCreatesDraftDocument(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "CSV Document Product",
		SKU:   "CSV-DOC-1",
		Stock: 3,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}
	warehouseID := int64(1)

	result, err := store.ImportDocumentCSV(Document{
		Type:        DocumentTypeInvoice,
		WarehouseID: &warehouseID,
		Currency:    "UAH",
		Note:        "csv document",
	}, fmt.Sprintf("productId,quantity,price\n%d,2,50\nbad,1,10", product.ID))
	if err != nil {
		t.Fatalf("expected document csv import: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 1 || len(result.Document.Items) != 1 {
		t.Fatalf("unexpected document import result: %+v", result)
	}
	if result.Document.Status != DocumentStatusDraft {
		t.Fatalf("expected draft document, got %s", result.Document.Status)
	}
}

func TestServiceCatalogCRUD(t *testing.T) {
	store := NewStore()
	category, err := store.CreateServiceCategory(ServiceCategory{Name: "Repairs"})
	if err != nil {
		t.Fatalf("expected service category create: %v", err)
	}
	service, err := store.CreateService(Service{
		CategoryID:  category.ID,
		Name:        "Diagnostics",
		Price:       300,
		Currency:    "UAH",
		DurationMin: 30,
	})
	if err != nil {
		t.Fatalf("expected service create: %v", err)
	}
	services, err := store.ListServices()
	if err != nil {
		t.Fatalf("expected list services: %v", err)
	}
	if len(services) != 1 || services[0].ID != service.ID {
		t.Fatalf("expected created service in list, got %+v", services)
	}
}

func TestQuickMessageHistoryAndCategoryAnalytics(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:          "Analytics Phone",
		SKU:           "AN-PH-1",
		Category:      "Phones",
		Stock:         3,
		PurchasePrice: 100,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}
	if _, err := store.CreateSale([]SaleItem{{ProductID: product.ID, Quantity: 1, Price: 150}}); err != nil {
		t.Fatalf("expected sale create: %v", err)
	}
	notification, err := store.SendQuickMessage(QuickMessageRequest{
		Channel:   NotificationChannelSMS,
		Recipient: "+380000000000",
		Body:      "Test",
	})
	if err != nil {
		t.Fatalf("expected quick sms message: %v", err)
	}
	if notification.Status != NotificationStatusSentStub {
		t.Fatalf("expected stub status, got %s", notification.Status)
	}
	store.AddAuditLog("tester", "product.update", "product", "new value")
	history, err := store.ChangeHistory(10)
	if err != nil {
		t.Fatalf("expected change history: %v", err)
	}
	if len(history) == 0 || history[0].NewValue == "" {
		t.Fatalf("expected history entries with new value, got %+v", history)
	}
	categories, err := store.CategoryAnalytics()
	if err != nil {
		t.Fatalf("expected category analytics: %v", err)
	}
	if len(categories) != 1 || categories[0].Category != "Phones" || categories[0].Quantity != 1 {
		t.Fatalf("unexpected category analytics: %+v", categories)
	}
}

func TestSupplierOrderOverReceiptIsRejected(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Over Receipt Product",
		SKU:   "ORP-1",
		Stock: 0,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	supplier, err := store.CreateSupplier(Supplier{Name: "Over Supplier"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}

	order, err := store.CreateSupplierOrder(
		supplier.ID,
		[]PurchaseItem{{ProductID: product.ID, Quantity: 2, Price: 10}},
		"UAH",
	)
	if err != nil {
		t.Fatalf("expected supplier order create: %v", err)
	}

	_, err = store.CreatePurchase(Purchase{
		SupplierID:      supplier.ID,
		SupplierOrderID: &order.ID,
		Currency:        "UAH",
		Items:           []PurchaseItem{{ProductID: product.ID, Quantity: 2, Price: 10}},
	})
	if err != nil {
		t.Fatalf("expected initial purchase create: %v", err)
	}

	_, err = store.CreatePurchase(Purchase{
		SupplierID:      supplier.ID,
		SupplierOrderID: &order.ID,
		Currency:        "UAH",
		Items:           []PurchaseItem{{ProductID: product.ID, Quantity: 1, Price: 10}},
	})
	if err == nil {
		t.Fatalf("expected over-receipt to be rejected")
	}
}

func TestReceiveSupplierOrderByLinesAndPending(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Receive Lines Product",
		SKU:   "RLP-1",
		Stock: 0,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	supplier, err := store.CreateSupplier(Supplier{Name: "Receive Supplier"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}

	order, err := store.CreateSupplierOrder(
		supplier.ID,
		[]PurchaseItem{{ProductID: product.ID, Quantity: 5, Price: 25}},
		"UAH",
	)
	if err != nil {
		t.Fatalf("expected supplier order create: %v", err)
	}

	_, err = store.ReceiveSupplierOrderByLines(
		order.ID,
		"",
		[]SupplierOrderReceiveLine{
			{
				ProductID: product.ID,
				Quantity:  2,
			},
		},
		"partial receive",
	)
	if err != nil {
		t.Fatalf("expected receive by lines success: %v", err)
	}

	pending, err := store.ListSupplierOrdersPending()
	if err != nil {
		t.Fatalf("expected pending list success: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected one pending summary, got %d", len(pending))
	}
	if pending[0].PendingItems[0].Pending != 3 {
		t.Fatalf("expected pending quantity 3, got %d", pending[0].PendingItems[0].Pending)
	}
}

func TestCannotManuallySetSupplierOrderReceivedWhenPendingExists(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Manual Status Product",
		SKU:   "MSP-1",
		Stock: 0,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}
	supplier, err := store.CreateSupplier(Supplier{Name: "Manual Status Supplier"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}

	order, err := store.CreateSupplierOrder(
		supplier.ID,
		[]PurchaseItem{{ProductID: product.ID, Quantity: 5, Price: 10}},
		"UAH",
	)
	if err != nil {
		t.Fatalf("expected supplier order create: %v", err)
	}

	err = store.UpdateSupplierOrderStatus(order.ID, SupplierOrderStatusReceived)
	if err == nil {
		t.Fatalf("expected status update to be rejected when pending exists")
	}

	_, err = store.CreatePurchase(Purchase{
		SupplierID:      supplier.ID,
		SupplierOrderID: &order.ID,
		Currency:        "UAH",
		Items:           []PurchaseItem{{ProductID: product.ID, Quantity: 5, Price: 10}},
	})
	if err != nil {
		t.Fatalf("expected full purchase create: %v", err)
	}

	err = store.UpdateSupplierOrderStatus(order.ID, SupplierOrderStatusReceived)
	if err != nil {
		t.Fatalf("expected status update after full receive: %v", err)
	}

	err = store.UpdateSupplierOrderStatus(order.ID, SupplierOrderStatusDraft)
	if err == nil {
		t.Fatalf("expected downgrade from received to draft to be rejected")
	}
}

func TestWarehouseTransferAndInventoryAdjustments(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "WH Product",
		SKU:   "WH-1",
		Stock: 10,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	targetWarehouse, err := store.CreateWarehouse(Warehouse{
		Name: "Secondary Warehouse",
	})
	if err != nil {
		t.Fatalf("expected warehouse create: %v", err)
	}

	_, err = store.CreateStockTransfer(StockTransfer{
		FromWarehouseID: 1,
		ToWarehouseID:   targetWarehouse.ID,
		Items: []StockTransferItem{
			{ProductID: product.ID, Quantity: 4},
		},
	})
	if err != nil {
		t.Fatalf("expected transfer create: %v", err)
	}

	stocks, err := store.ListWarehouseStocks(&targetWarehouse.ID, &product.ID)
	if err != nil {
		t.Fatalf("expected warehouse stocks list: %v", err)
	}
	if len(stocks) != 1 || stocks[0].Quantity != 4 {
		t.Fatalf("expected target warehouse qty 4")
	}

	inventory, err := store.CreateInventory(Inventory{
		WarehouseID: targetWarehouse.ID,
		Items: []InventoryItem{
			{
				ProductID:      product.ID,
				ActualQuantity: 3,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected inventory create: %v", err)
	}

	applied, err := store.ApplyInventory(inventory.ID)
	if err != nil {
		t.Fatalf("expected inventory apply: %v", err)
	}
	if applied.Status != InventoryStatusApplied {
		t.Fatalf("expected inventory status applied, got %s", applied.Status)
	}

	products, err := store.ListProducts()
	if err != nil {
		t.Fatalf("expected products list: %v", err)
	}
	if products[0].Stock != 9 {
		t.Fatalf("expected total stock 9 after inventory adjustment, got %d", products[0].Stock)
	}
}

func TestWarehouseZonesCellsAndCellStocks(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Cell Product",
		SKU:   "CELL-1",
		Stock: 5,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	warehouse, err := store.CreateWarehouse(Warehouse{Name: "Zoned Warehouse"})
	if err != nil {
		t.Fatalf("expected warehouse create: %v", err)
	}

	zone, err := store.CreateWarehouseZone(WarehouseZone{
		WarehouseID: warehouse.ID,
		Name:        "A",
	})
	if err != nil {
		t.Fatalf("expected zone create: %v", err)
	}

	_, err = store.CreateWarehouseCell(WarehouseCell{
		WarehouseID: warehouse.ID,
		ZoneID:      zone.ID,
		Code:        "A-01",
	})
	if err != nil {
		t.Fatalf("expected cell create: %v", err)
	}

	_, err = store.CreateStockTransfer(StockTransfer{
		FromWarehouseID: 1,
		ToWarehouseID:   warehouse.ID,
		Items: []StockTransferItem{
			{ProductID: product.ID, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("expected transfer create: %v", err)
	}

	cellStocks, err := store.ListCellStocks(nil, &product.ID)
	if err != nil {
		t.Fatalf("expected cell stocks list: %v", err)
	}
	if len(cellStocks) == 0 {
		t.Fatalf("expected at least one cell stock record")
	}

	zones, err := store.ListWarehouseZones(&warehouse.ID)
	if err != nil {
		t.Fatalf("expected zones list: %v", err)
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
	defaultCells, err := store.ListWarehouseCells(&defaultZoneID)
	if err != nil {
		t.Fatalf("expected default cells list: %v", err)
	}
	var mainCellID int64
	for _, cell := range defaultCells {
		if cell.Code == "MAIN" {
			mainCellID = cell.ID
			break
		}
	}
	if mainCellID == 0 {
		t.Fatalf("expected main cell")
	}
	customCells, err := store.ListWarehouseCells(&zone.ID)
	if err != nil {
		t.Fatalf("expected custom cells list: %v", err)
	}
	if len(customCells) == 0 {
		t.Fatalf("expected custom cell")
	}
	customCellID := customCells[0].ID

	_, err = store.CreateStockTransfer(StockTransfer{
		FromWarehouseID: warehouse.ID,
		ToWarehouseID:   warehouse.ID,
		Items: []StockTransferItem{
			{
				ProductID:  product.ID,
				Quantity:   1,
				FromCellID: &mainCellID,
				ToCellID:   &customCellID,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected in-warehouse cell transfer: %v", err)
	}

	_, err = store.CreateCellTransfer(
		mainCellID,
		customCellID,
		[]StockTransferItem{
			{
				ProductID: product.ID,
				Quantity:  1,
			},
		},
		"cell endpoint style",
	)
	if err != nil {
		t.Fatalf("expected create cell transfer: %v", err)
	}

	fifoTransfer, err := store.CreateCellTransferFIFO(
		mainCellID,
		[]CellTransferFIFOItem{
			{
				ProductID: product.ID,
				Quantity:  1,
			},
		},
		"fifo cell transfer",
	)
	if err != nil {
		t.Fatalf("expected create fifo cell transfer: %v", err)
	}
	if len(fifoTransfer.Items) != 1 {
		t.Fatalf("expected one resolved fifo item, got %d", len(fifoTransfer.Items))
	}
	if fifoTransfer.Items[0].FromCellID == nil || *fifoTransfer.Items[0].FromCellID != customCellID {
		t.Fatalf("expected fifo source cell custom")
	}
	if fifoTransfer.Items[0].ToCellID == nil || *fifoTransfer.Items[0].ToCellID != mainCellID {
		t.Fatalf("expected fifo target main cell")
	}
}

func TestDocumentPostingUpdatesStockAndCash(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Doc Product",
		SKU:   "DOC-1",
		Stock: 1,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	warehouseID := int64(1)
	invoice, err := store.CreateDocument(Document{
		Type:        DocumentTypeInvoice,
		WarehouseID: &warehouseID,
		Currency:    "UAH",
		Items: []DocumentItem{
			{
				ProductID: product.ID,
				Quantity:  2,
				Price:     150,
			},
		},
		Note: "invoice posting test",
	})
	if err != nil {
		t.Fatalf("expected create invoice document: %v", err)
	}
	if invoice.Status != DocumentStatusDraft {
		t.Fatalf("expected draft status, got %s", invoice.Status)
	}

	posted, err := store.PostDocument(invoice.ID)
	if err != nil {
		t.Fatalf("expected post invoice document: %v", err)
	}
	if posted.Status != DocumentStatusPosted {
		t.Fatalf("expected posted status, got %s", posted.Status)
	}

	products, err := store.ListProducts()
	if err != nil {
		t.Fatalf("expected products list: %v", err)
	}
	if products[0].Stock != 3 {
		t.Fatalf("expected stock 3 after posted invoice, got %d", products[0].Stock)
	}

	cashboxID := int64(1)
	cashDoc, err := store.CreateDocument(Document{
		Type:      DocumentTypeCashInOrder,
		CashboxID: &cashboxID,
		Currency:  "UAH",
		Total:     250,
		Note:      "cash in doc",
	})
	if err != nil {
		t.Fatalf("expected create cash document: %v", err)
	}

	_, err = store.PostDocument(cashDoc.ID)
	if err != nil {
		t.Fatalf("expected post cash document: %v", err)
	}
	cashboxes, err := store.ListCashboxes()
	if err != nil {
		t.Fatalf("expected cashboxes list: %v", err)
	}
	if cashboxes[0].Balance != 250 {
		t.Fatalf("expected balance 250 after cash document, got %.2f", cashboxes[0].Balance)
	}
}

func TestDocumentTemplatesAndPDFRender(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "PDF Product",
		SKU:   "PDF-1",
		Stock: 1,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}

	_, err = store.UpsertDocumentTemplate(DocumentTemplate{
		Code:     DocumentTypeInvoice,
		Name:     "Invoice Custom",
		Body:     "Invoice {{number}} total {{total}} {{currency}}",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("expected upsert document template: %v", err)
	}

	warehouseID := int64(1)
	doc, err := store.CreateDocument(Document{
		Type:        DocumentTypeInvoice,
		WarehouseID: &warehouseID,
		Currency:    "UAH",
		Items: []DocumentItem{
			{
				ProductID: product.ID,
				Quantity:  1,
				Price:     120,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected create document: %v", err)
	}

	pdf, err := store.RenderDocumentPDF(doc.ID)
	if err != nil {
		t.Fatalf("expected render pdf: %v", err)
	}
	if len(pdf) < 4 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("expected PDF content prefix")
	}
}

func TestDocumentTemplateValidationAndPreview(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:  "Preview Product",
		SKU:   "PRV-1",
		Stock: 1,
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}
	warehouseID := int64(1)
	doc, err := store.CreateDocument(Document{
		Type:        DocumentTypeInvoice,
		WarehouseID: &warehouseID,
		Currency:    "UAH",
		Items: []DocumentItem{
			{
				ProductID: product.ID,
				Quantity:  1,
				Price:     100,
			},
		},
		Note: "validation test",
	})
	if err != nil {
		t.Fatalf("expected create document: %v", err)
	}

	validation, err := store.ValidateDocumentTemplate(
		DocumentTypeInvoice,
		"Invoice {{number}} total {{total}} {{currency}} {{items}}",
		true,
	)
	if err != nil {
		t.Fatalf("expected template validation: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("expected template to be valid")
	}
	if len(validation.Unknown) != 0 {
		t.Fatalf("expected no unknown placeholders")
	}
	if len(validation.MissingRequired) != 0 {
		t.Fatalf("expected no missing required placeholders")
	}

	invalid, err := store.ValidateDocumentTemplate(
		DocumentTypeInvoice,
		"Invoice {{number}} {{unknown_field}}",
		false,
	)
	if err != nil {
		t.Fatalf("expected invalid template validation response, got error: %v", err)
	}
	if invalid.Valid {
		t.Fatalf("expected template to be invalid")
	}
	if len(invalid.Unknown) != 1 || invalid.Unknown[0] != "unknown_field" {
		t.Fatalf("expected unknown placeholder unknown_field")
	}

	preview, err := store.PreviewDocumentTemplate(
		DocumentTypeInvoice,
		"Invoice {{number}} total {{total}} {{currency}} {{items}}",
		&doc.ID,
		true,
	)
	if err != nil {
		t.Fatalf("expected template preview: %v", err)
	}
	if preview.Content == "" {
		t.Fatalf("expected non-empty preview content")
	}
	if !strings.Contains(preview.Content, doc.Number) {
		t.Fatalf("expected preview to contain document number")
	}

	strictInvalid, err := store.ValidateDocumentTemplate(
		DocumentTypeInvoice,
		"Invoice {{number}}",
		true,
	)
	if err != nil {
		t.Fatalf("expected strict validation response, got error: %v", err)
	}
	if strictInvalid.Valid {
		t.Fatalf("expected strict validation to fail on missing required placeholders")
	}
	if len(strictInvalid.MissingRequired) == 0 {
		t.Fatalf("expected strict validation missing required placeholders")
	}
}

func TestReturnDocumentsAvailabilityAndLimits(t *testing.T) {
	store := NewStore()

	customerProduct, err := store.CreateProduct(Product{
		Name:  "Return Customer Product",
		SKU:   "RET-CUST-1",
		Stock: 10,
	})
	if err != nil {
		t.Fatalf("expected customer product create: %v", err)
	}
	sale, err := store.CreateSale([]SaleItem{
		{
			ProductID: customerProduct.ID,
			Quantity:  5,
			Price:     100,
		},
	})
	if err != nil {
		t.Fatalf("expected create sale: %v", err)
	}

	customerAvailability, err := store.CustomerReturnAvailability(sale.ID)
	if err != nil {
		t.Fatalf("expected customer return availability: %v", err)
	}
	if len(customerAvailability.Items) != 1 {
		t.Fatalf("expected one customer availability item, got %d", len(customerAvailability.Items))
	}
	if customerAvailability.Items[0].AvailableQty != 5 {
		t.Fatalf("expected available qty 5, got %d", customerAvailability.Items[0].AvailableQty)
	}

	warehouseID := int64(1)
	customerReturnDoc, err := store.CreateReturnFromCustomerDocument(
		sale.ID,
		warehouseID,
		"UAH",
		[]DocumentItem{
			{ProductID: customerProduct.ID, Quantity: 2},
		},
		"customer return",
	)
	if err != nil {
		t.Fatalf("expected create customer return document: %v", err)
	}
	if customerReturnDoc.SourceSaleID == nil || *customerReturnDoc.SourceSaleID != sale.ID {
		t.Fatalf("expected source sale id in return document")
	}

	customerAvailability, err = store.CustomerReturnAvailability(sale.ID)
	if err != nil {
		t.Fatalf("expected customer return availability after draft: %v", err)
	}
	if customerAvailability.Items[0].AvailableQty != 3 {
		t.Fatalf("expected available qty 3 after draft return, got %d", customerAvailability.Items[0].AvailableQty)
	}

	if _, err := store.CreateReturnFromCustomerDocument(
		sale.ID,
		warehouseID,
		"UAH",
		[]DocumentItem{
			{ProductID: customerProduct.ID, Quantity: 4},
		},
		"overflow customer return",
	); err == nil {
		t.Fatalf("expected overflow customer return validation error")
	}

	if _, err := store.PostDocument(customerReturnDoc.ID); err != nil {
		t.Fatalf("expected post customer return document: %v", err)
	}
	products, err := store.ListProducts()
	if err != nil {
		t.Fatalf("expected products list: %v", err)
	}
	if products[0].Stock != 7 {
		t.Fatalf("expected stock 7 after sale and customer return post, got %d", products[0].Stock)
	}

	supplierProduct, err := store.CreateProduct(Product{
		Name:  "Return Supplier Product",
		SKU:   "RET-SUP-1",
		Stock: 0,
	})
	if err != nil {
		t.Fatalf("expected supplier product create: %v", err)
	}
	supplier, err := store.CreateSupplier(Supplier{Name: "Return Supplier"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}
	purchase, err := store.CreatePurchase(Purchase{
		SupplierID: supplier.ID,
		Currency:   "UAH",
		Items: []PurchaseItem{
			{ProductID: supplierProduct.ID, Quantity: 6, Price: 40},
		},
	})
	if err != nil {
		t.Fatalf("expected create purchase: %v", err)
	}

	supplierAvailability, err := store.SupplierReturnAvailability(purchase.ID)
	if err != nil {
		t.Fatalf("expected supplier return availability: %v", err)
	}
	if len(supplierAvailability.Items) != 1 {
		t.Fatalf("expected one supplier availability item, got %d", len(supplierAvailability.Items))
	}
	if supplierAvailability.Items[0].AvailableQty != 6 {
		t.Fatalf("expected supplier available qty 6, got %d", supplierAvailability.Items[0].AvailableQty)
	}

	supplierReturnDoc, err := store.CreateReturnToSupplierDocument(
		purchase.ID,
		warehouseID,
		"UAH",
		[]DocumentItem{
			{ProductID: supplierProduct.ID, Quantity: 2},
		},
		"supplier return",
	)
	if err != nil {
		t.Fatalf("expected create supplier return document: %v", err)
	}
	if supplierReturnDoc.SourcePurchaseID == nil || *supplierReturnDoc.SourcePurchaseID != purchase.ID {
		t.Fatalf("expected source purchase id in supplier return document")
	}
	if _, err := store.PostDocument(supplierReturnDoc.ID); err != nil {
		t.Fatalf("expected post supplier return document: %v", err)
	}

	supplierAvailability, err = store.SupplierReturnAvailability(purchase.ID)
	if err != nil {
		t.Fatalf("expected supplier return availability after post: %v", err)
	}
	if supplierAvailability.Items[0].AvailableQty != 4 {
		t.Fatalf("expected supplier available qty 4 after posted return, got %d", supplierAvailability.Items[0].AvailableQty)
	}
	if _, err := store.CreateReturnToSupplierDocument(
		purchase.ID,
		warehouseID,
		"UAH",
		[]DocumentItem{
			{ProductID: supplierProduct.ID, Quantity: 5},
		},
		"overflow supplier return",
	); err == nil {
		t.Fatalf("expected overflow supplier return validation error")
	}
}

func TestPurchaseRecommendationsAndOrderCreation(t *testing.T) {
	store := NewStore()
	product, err := store.CreateProduct(Product{
		Name:          "Rec Product",
		SKU:           "REC-1",
		Stock:         2,
		MinStock:      5,
		PurchasePrice: 20,
		Supplier:      "Rec Supplier",
	})
	if err != nil {
		t.Fatalf("expected product create: %v", err)
	}
	supplier, err := store.CreateSupplier(Supplier{Name: "Rec Supplier"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}

	recommendations, err := store.ListPurchaseRecommendations(20)
	if err != nil {
		t.Fatalf("expected recommendations list: %v", err)
	}
	if len(recommendations) == 0 {
		t.Fatalf("expected at least one recommendation")
	}
	if recommendations[0].ProductID != product.ID {
		t.Fatalf("expected recommendation for test product")
	}
	if recommendations[0].RecommendedQty <= 0 {
		t.Fatalf("expected positive recommended qty")
	}

	order, err := store.CreateSupplierOrderFromRecommendations(
		supplier.ID,
		"UAH",
		[]PurchaseRecommendationOrderLine{
			{
				ProductID: product.ID,
				Quantity:  recommendations[0].RecommendedQty,
			},
		},
	)
	if err != nil {
		t.Fatalf("expected create supplier order from recommendations: %v", err)
	}
	if len(order.Items) != 1 {
		t.Fatalf("expected one order item")
	}
	if order.Items[0].Price != 20 {
		t.Fatalf("expected fallback price from purchase price")
	}
}

func TestPurchaseRecommendationsGroupedAndBulkCreate(t *testing.T) {
	store := NewStore()
	supplierA, err := store.CreateSupplier(Supplier{Name: "Group Supplier A"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}
	supplierB, err := store.CreateSupplier(Supplier{Name: "Group Supplier B"})
	if err != nil {
		t.Fatalf("expected supplier create: %v", err)
	}

	productA, err := store.CreateProduct(Product{
		Name:          "Group Product A",
		SKU:           "GPA-1",
		Stock:         0,
		MinStock:      5,
		PurchasePrice: 11,
		Supplier:      supplierA.Name,
	})
	if err != nil {
		t.Fatalf("expected product A create: %v", err)
	}
	productB, err := store.CreateProduct(Product{
		Name:          "Group Product B",
		SKU:           "GPB-1",
		Stock:         1,
		MinStock:      6,
		PurchasePrice: 13,
		Supplier:      supplierB.Name,
	})
	if err != nil {
		t.Fatalf("expected product B create: %v", err)
	}

	groups, err := store.ListPurchaseRecommendationsGrouped(20)
	if err != nil {
		t.Fatalf("expected grouped recommendations: %v", err)
	}
	if len(groups) < 2 {
		t.Fatalf("expected at least two groups, got %d", len(groups))
	}

	orders, err := store.CreateSupplierOrdersBulkFromRecommendations(
		[]PurchaseRecommendationCreateOrderRequest{
			{
				SupplierID: supplierA.ID,
				Currency:   "UAH",
				Items: []PurchaseRecommendationOrderLine{
					{ProductID: productA.ID, Quantity: 2},
				},
			},
			{
				SupplierID: supplierB.ID,
				Currency:   "UAH",
				Items: []PurchaseRecommendationOrderLine{
					{ProductID: productB.ID, Quantity: 3},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("expected bulk create from recommendations: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected two created orders, got %d", len(orders))
	}
}

func TestCashShiftOpenCloseFlow(t *testing.T) {
	store := NewStore()
	cashboxID := int64(1)
	_, err := store.CreateCashOperation(CashOperation{
		CashboxID:   cashboxID,
		Type:        CashOperationTypeIncoming,
		Amount:      300,
		Method:      PaymentMethodCash,
		Description: "seed",
	})
	if err != nil {
		t.Fatalf("expected seed cash operation: %v", err)
	}

	opened, err := store.OpenCashShift(cashboxID, "admin", "day shift")
	if err != nil {
		t.Fatalf("expected open shift: %v", err)
	}
	if opened.Status != CashShiftStatusOpen {
		t.Fatalf("expected open status, got %s", opened.Status)
	}
	if opened.OpeningBalance != 300 {
		t.Fatalf("expected opening balance 300, got %.2f", opened.OpeningBalance)
	}

	_, err = store.OpenCashShift(cashboxID, "admin", "duplicate")
	if err == nil {
		t.Fatalf("expected duplicate open shift to fail")
	}

	_, err = store.CreateCashOperation(CashOperation{
		CashboxID:   cashboxID,
		Type:        CashOperationTypeOutgoing,
		Amount:      50,
		Method:      PaymentMethodCash,
		Description: "expense during shift",
	})
	if err != nil {
		t.Fatalf("expected operation during shift: %v", err)
	}

	closed, err := store.CloseCashShift(opened.ID, "admin", "close shift")
	if err != nil {
		t.Fatalf("expected close shift: %v", err)
	}
	if closed.Status != CashShiftStatusClosed {
		t.Fatalf("expected closed status, got %s", closed.Status)
	}
	if closed.ClosingBalance != 250 {
		t.Fatalf("expected closing balance 250, got %.2f", closed.ClosingBalance)
	}

	shifts, err := store.ListCashShifts(&cashboxID, CashShiftStatusClosed)
	if err != nil {
		t.Fatalf("expected list shifts: %v", err)
	}
	if len(shifts) == 0 {
		t.Fatalf("expected at least one closed shift")
	}
}
