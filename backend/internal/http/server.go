package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"erp-backend/internal/app"
)

type Server struct {
	store  *app.Store
	tokens *app.TokenManager
	logger *slog.Logger
}

func NewServer(store *app.Store, tokens *app.TokenManager, logger *slog.Logger) *Server {
	return &Server{
		store:  store,
		tokens: tokens,
		logger: logger,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(corsMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)
		// Public supplier deficit portal — authenticated by static token in query string
		r.Get("/public/supplier-deficit", s.handleSupplierDeficitPortal)

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			r.Get("/auth/me", s.handleMe)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products", s.handleListProducts)
			r.With(s.requirePermission(app.PermProductsWrite)).Post("/products", s.handleCreateProduct)
			r.With(s.requirePermission(app.PermProductsWrite)).Put("/products/{id}", s.handleUpdateProduct)
			r.With(s.requirePermission(app.PermProductsWrite)).Post("/products/{id}/archive", s.handleArchiveProduct)
			r.With(s.requirePermission(app.PermProductsWrite)).Post("/products/{id}/generate-barcode", s.handleGenerateProductBarcode)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products/{id}/lifecycle", s.handleGetProductLifecycle)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products/{id}/label.pdf", s.handleProductLabelPDF)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products/labels.pdf", s.handleProductLabelPDF)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products/{id}/qr.svg", s.handleProductQRImage)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products/duplicates", s.handleProductDuplicates)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products/export-csv", s.handleExportProductsCSV)
			r.With(s.requirePermission(app.PermProductsWrite)).Post("/products/import-csv", s.handleImportProductsCSV)
			r.With(s.requirePermission(app.PermProductsRead)).Get("/products/export-xlsx", s.handleExportProductsXLSX)
			r.With(s.requirePermission(app.PermProductsWrite)).Post("/products/import-xlsx", s.handleImportProductsXLSX)
			r.With(s.requirePermission(app.PermProductsWrite)).Post("/products/prices/bulk", s.handleBulkUpdateProductPrices)
			r.With(s.requirePermission(app.PermProductsWrite)).Post("/products/merge-duplicates", s.handleMergeProducts)
			r.With(s.requirePermission(app.PermTransfersRead)).Get("/stock/movements", s.handleListStockMovements)
			r.With(s.requirePermission(app.PermStockWrite)).Post("/stock/movements", s.handleCreateMovement)
			r.With(s.requirePermission(app.PermSalesWrite)).Post("/sales", s.handleCreateSale)
			r.With(s.requirePermission(app.PermSalesWrite)).Post("/sales/{id}/send-receipt", s.handleSendReceiptForSale)
			r.With(s.requirePermission(app.PermSalesRead)).Get("/sales", s.handleListSales)
			r.With(s.requirePermission(app.PermSalesRead)).Get("/receipts", s.handleListReceipts)
			r.With(s.requirePermission(app.PermSalesRead)).Get("/receipts/{id}", s.handleGetReceipt)
			r.With(s.requirePermission(app.PermSalesWrite)).Post("/receipts/{id}/retry", s.handleRetryReceipt)
			r.With(s.requirePermission(app.PermSalesWrite)).Post("/receipts/retry-bulk", s.handleRetryReceiptsBulk)
			r.With(s.requirePermission(app.PermOrdersWrite)).Post("/orders", s.handleCreateOrder)
			r.With(s.requirePermission(app.PermOrdersRead)).Get("/orders", s.handleListOrders)
			r.With(s.requirePermission(app.PermOrdersWrite)).Put("/orders/{id}/status", s.handleUpdateOrderStatus)
			r.With(s.requirePermission(app.PermOrdersWrite)).Put("/orders/{id}", s.handleUpdateOrder)
			r.With(s.requirePermission(app.PermOrdersRead)).Get("/orders/{id}/chain", s.handleGetOrderChain)
			r.With(s.requirePermission(app.PermOrdersRead)).Get("/documents/registry", s.handleSearchDocuments)
			r.With(s.requirePermission(app.PermOrdersRead)).Get("/journal", s.handleJournal)
			r.With(s.requirePermission(app.PermCRMRead)).Get("/counterparties", s.handleListCounterparties)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/counterparties", s.handleCreateCounterparty)
			r.With(s.requirePermission(app.PermCRMWrite)).Put("/counterparties/{id}", s.handleUpdateCounterparty)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/customers", s.handleCreateCustomer)
			r.With(s.requirePermission(app.PermCRMRead)).Get("/customers", s.handleListCustomers)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/customer-reminders", s.handleCreateCustomerReminder)
			r.With(s.requirePermission(app.PermCRMRead)).Get("/customer-reminders", s.handleListCustomerReminders)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/customer-reminders/{id}/complete", s.handleCompleteCustomerReminder)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/service-orders", s.handleCreateServiceOrder)
			r.With(s.requirePermission(app.PermCRMRead)).Get("/service-orders", s.handleListServiceOrders)
			r.With(s.requirePermission(app.PermCRMWrite)).Put("/service-orders/{id}", s.handleUpdateServiceOrder)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/service-orders/{id}/parts", s.handleAddServiceOrderPart)
			r.With(s.requirePermission(app.PermDocumentsWrite)).Post("/service-orders/{id}/act", s.handleCreateServiceOrderActDocument)
			r.With(s.requirePermission(app.PermDocumentsRead)).Get("/service-orders/{id}/act/pdf", s.handleRenderServiceOrderActPDF)
			r.With(s.requirePermission(app.PermDocumentsWrite)).Post("/service-orders/{id}/act/cancel", s.handleCancelServiceOrderActDocument)
			r.With(s.requirePermission(app.PermCRMWrite)).Put("/service-orders/{id}/status", s.handleUpdateServiceOrderStatus)

				// Attachments (universal: service_order, customer_order, purchase, sale)
				r.With(s.requirePermission(app.PermCRMRead)).Get("/attachments", s.handleListAttachments)
				r.With(s.requirePermission(app.PermCRMWrite)).Post("/attachments", s.handleUploadAttachment)
				r.With(s.requirePermission(app.PermCRMRead)).Get("/attachments/{id}/download", s.handleDownloadAttachment)
				r.With(s.requirePermission(app.PermCRMWrite)).Delete("/attachments/{id}", s.handleDeleteAttachment)
			r.With(s.requirePermission(app.PermCRMRead)).Get("/service-categories", s.handleListServiceCategories)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/service-categories", s.handleCreateServiceCategory)
			r.With(s.requirePermission(app.PermCRMRead)).Get("/services", s.handleListServices)
			r.With(s.requirePermission(app.PermCRMWrite)).Post("/services", s.handleCreateService)
			r.With(s.requirePermission(app.PermReserveRead)).Get("/reservations", s.handleListReservations)
			r.With(s.requirePermission(app.PermWarehousesWrite)).Post("/warehouses", s.handleCreateWarehouse)
			r.With(s.requirePermission(app.PermWarehousesRead)).Get("/warehouses", s.handleListWarehouses)
			r.With(s.requirePermission(app.PermWarehousesRead)).Get("/warehouses/stocks", s.handleListWarehouseStocks)
			r.With(s.requirePermission(app.PermWarehousesWrite)).Post("/warehouse-zones", s.handleCreateWarehouseZone)
			r.With(s.requirePermission(app.PermWarehousesRead)).Get("/warehouse-zones", s.handleListWarehouseZones)
			r.With(s.requirePermission(app.PermWarehousesWrite)).Post("/warehouse-cells", s.handleCreateWarehouseCell)
			r.With(s.requirePermission(app.PermWarehousesRead)).Get("/warehouse-cells", s.handleListWarehouseCells)
			r.With(s.requirePermission(app.PermWarehousesRead)).Get("/warehouse-cells/stocks", s.handleListCellStocks)
			r.With(s.requirePermission(app.PermTransfersWrite)).Post("/transfers/cell", s.handleCreateCellTransfer)
			r.With(s.requirePermission(app.PermTransfersWrite)).Post("/transfers/cell/fifo", s.handleCreateCellTransferFIFO)
			r.With(s.requirePermission(app.PermTransfersWrite)).Post("/transfers", s.handleCreateTransfer)
			r.With(s.requirePermission(app.PermInventoryWrite)).Post("/inventories", s.handleCreateInventory)
			r.With(s.requirePermission(app.PermInventoryRead)).Get("/inventories", s.handleListInventories)
			r.With(s.requirePermission(app.PermInventoryWrite)).Post("/inventories/{id}/apply", s.handleApplyInventory)
			r.With(s.requirePermission(app.PermDocumentsWrite)).Post("/documents", s.handleCreateDocument)
			r.With(s.requirePermission(app.PermDocumentsWrite)).Post("/documents/import-csv", s.handleImportDocumentCSV)
			r.With(s.requirePermission(app.PermDocumentsRead)).Get("/documents/returns/customer/{saleId}/available", s.handleCustomerReturnAvailability)
			r.With(s.requirePermission(app.PermDocumentsWrite)).Post("/documents/returns/customer", s.handleCreateCustomerReturnDocument)
			r.With(s.requirePermission(app.PermDocumentsRead)).Get("/documents/returns/supplier/{purchaseId}/available", s.handleSupplierReturnAvailability)
			r.With(s.requirePermission(app.PermDocumentsWrite)).Post("/documents/returns/supplier", s.handleCreateSupplierReturnDocument)
			r.With(s.requirePermission(app.PermDocumentsRead)).Get("/documents", s.handleListDocuments)
			r.With(s.requirePermission(app.PermDocumentsWrite)).Post("/documents/{id}/post", s.handlePostDocument)
			r.With(s.requirePermission(app.PermDocumentsRead)).Get("/documents/{id}/pdf", s.handleRenderDocumentPDF)
			r.With(s.requirePermission(app.PermDocTemplatesRead)).Get("/document-templates", s.handleListDocumentTemplates)
			r.With(s.requirePermission(app.PermDocTemplatesWrite)).Put("/document-templates", s.handleUpsertDocumentTemplate)
			r.With(s.requirePermission(app.PermDocTemplatesRead)).Get("/document-templates/placeholders", s.handleDocumentTemplatePlaceholders)
			r.With(s.requirePermission(app.PermDocTemplatesWrite)).Post("/document-templates/validate", s.handleValidateDocumentTemplate)
			r.With(s.requirePermission(app.PermDocTemplatesWrite)).Post("/document-templates/preview", s.handlePreviewDocumentTemplate)
			r.With(s.requirePermission(app.PermSuppliersWrite)).Post("/suppliers", s.handleCreateSupplier)
			r.With(s.requirePermission(app.PermSuppliersRead)).Get("/suppliers", s.handleListSuppliers)
			r.With(s.requirePermission(app.PermSupplierOrdersWrite)).Post("/supplier-orders", s.handleCreateSupplierOrder)
			r.With(s.requirePermission(app.PermSupplierOrdersRead)).Get("/supplier-orders", s.handleListSupplierOrders)
			r.With(s.requirePermission(app.PermSupplierOrdersRead)).Get("/supplier-orders/pending", s.handleSupplierOrdersPending)
			r.With(s.requirePermission(app.PermSupplierOrdersRead)).Get("/supplier-orders/recommendations", s.handleSupplierOrderRecommendations)
			r.With(s.requirePermission(app.PermSupplierOrdersRead)).Get("/supplier-orders/recommendations/grouped", s.handleSupplierOrderRecommendationsGrouped)
			r.With(s.requirePermission(app.PermSupplierOrdersWrite)).Post("/supplier-orders/recommendations/create-order", s.handleCreateSupplierOrderFromRecommendations)
			r.With(s.requirePermission(app.PermSupplierOrdersWrite)).Post("/supplier-orders/recommendations/create-orders-bulk", s.handleCreateSupplierOrdersBulkFromRecommendations)
			r.With(s.requirePermission(app.PermPurchasesWrite)).Post("/supplier-orders/{id}/receive", s.handleReceiveSupplierOrderByLines)
			r.With(s.requirePermission(app.PermSupplierOrdersWrite)).Put("/supplier-orders/{id}/status", s.handleUpdateSupplierOrderStatus)
			r.With(s.requirePermission(app.PermSupplierOrdersWrite)).Put("/supplier-orders/{id}", s.handleUpdateSupplierOrder)
			r.With(s.requirePermission(app.PermPurchasesWrite)).Post("/purchases", s.handleCreatePurchase)
			r.With(s.requirePermission(app.PermPurchasesWrite)).Post("/purchases/import-csv", s.handleImportPurchaseCSV)
			r.With(s.requirePermission(app.PermPurchasesRead)).Get("/purchases", s.handleListPurchases)
			r.With(s.requirePermission(app.PermPaymentsWrite)).Post("/payments", s.handleCreatePayment)
			r.With(s.requirePermission(app.PermPaymentsRead)).Get("/payments", s.handleListPayments)
			r.With(s.requirePermission(app.PermDebtsRead)).Get("/debts", s.handleListDebts)
			r.With(s.requirePermission(app.PermDebtsRead)).Get("/debts/overdue", s.handleListOverdueDebts)
			r.With(s.requirePermission(app.PermDebtsRead)).Get("/debts/history", s.handleDebtPaymentHistory)
			r.With(s.requirePermission(app.PermNotificationsRead)).Get("/notification-templates", s.handleListNotificationTemplates)
			r.With(s.requirePermission(app.PermNotificationsWrite)).Put("/notification-templates", s.handleUpsertNotificationTemplate)
			r.With(s.requirePermission(app.PermNotificationsRead)).Get("/notifications", s.handleListNotifications)
			r.With(s.requirePermission(app.PermNotificationsWrite)).Post("/notifications/quick-message", s.handleQuickMessage)
			r.With(s.requirePermission(app.PermNotificationsRead)).Get("/settings/notification", s.handleGetNotificationConfig)
			r.With(s.requirePermission(app.PermNotificationsWrite)).Put("/settings/notification", s.handleSaveNotificationConfig)
			r.With(s.requirePermission(app.PermJobsWrite)).Post("/jobs/overdue-reminders", s.handleEnqueueOverdueReminderJob)
			r.With(s.requirePermission(app.PermJobsWrite)).Post("/jobs/reservation-expiry", s.handleEnqueueReservationExpiryJob)
			r.With(s.requirePermission(app.PermJobsWrite)).Post("/jobs/receipts-retry", s.handleEnqueueReceiptRetryJob)
			r.With(s.requirePermission(app.PermJobsWrite)).Post("/jobs/run", s.handleRunBackgroundJobs)
			r.With(s.requirePermission(app.PermJobsRead)).Get("/jobs", s.handleListBackgroundJobs)
			r.With(s.requirePermission(app.PermCashboxesWrite)).Post("/cashboxes", s.handleCreateCashbox)
			r.With(s.requirePermission(app.PermCashboxesRead)).Get("/cashboxes", s.handleListCashboxes)
			r.With(s.requirePermission(app.PermCashOpsWrite)).Post("/cash-operations", s.handleCreateCashOperation)
			r.With(s.requirePermission(app.PermCashOpsRead)).Get("/cash-operations", s.handleListCashOperations)
			r.With(s.requirePermission(app.PermCashShiftsWrite)).Post("/cash-shifts/open", s.handleOpenCashShift)
			r.With(s.requirePermission(app.PermCashShiftsWrite)).Post("/cash-shifts/{id}/close", s.handleCloseCashShift)
			r.With(s.requirePermission(app.PermCashShiftsRead)).Get("/cash-shifts", s.handleListCashShifts)
			r.With(s.requirePermission(app.PermExchangeRatesRead)).Get("/exchange-rates", s.handleListExchangeRates)
			r.With(s.requirePermission(app.PermExchangeRatesWrite)).Put("/exchange-rates", s.handleUpsertExchangeRate)
			r.With(s.requirePermission(app.PermAnalyticsRead)).Get("/analytics/summary", s.handleSummary)
			r.With(s.requirePermission(app.PermAnalyticsRead)).Get("/analytics/profitability", s.handleProfitability)
			r.With(s.requirePermission(app.PermAnalyticsRead)).Get("/analytics/categories", s.handleCategoryAnalytics)
			r.With(s.requirePermission(app.PermAnalyticsRead)).Get("/analytics/sales-grouped", s.handleSalesGrouped)
			r.With(s.requirePermission(app.PermSuppliersRead)).Get("/reports/suppliers", s.handleSupplierReport)
			r.With(s.requirePermission(app.PermCRMRead)).Get("/reports/counterparties", s.handleCounterpartyReport)
			r.With(s.requirePermission(app.PermDocumentsRead)).Get("/documents/export-vat-csv", s.handleExportVATCsv)
			r.With(s.requirePermission(app.PermDocumentsRead)).Get("/documents/export-vat-xlsx", s.handleExportVATXlsx)
			r.With(s.requirePermission(app.PermAuditRead)).Get("/audit/logs", s.handleAuditLogs)
			r.With(s.requirePermission(app.PermAuditRead)).Get("/history/changes", s.handleChangeHistory)
			r.With(s.requirePermission(app.PermUsersRead)).Get("/users", s.handleListUsers)
			r.With(s.requirePermission(app.PermUsersWrite)).Post("/users", s.handleCreateUser)
			r.With(s.requirePermission(app.PermRolesRead)).Get("/roles", s.handleListRoles)
			r.With(s.requirePermission(app.PermRolesWrite)).Put("/roles/{role}/permissions", s.handleUpdateRolePermissions)
		})
	})

	return r
}
