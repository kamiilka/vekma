package app

const (
	PermProductsRead        = "products:read"
	PermProductsWrite       = "products:write"
	PermPurchasePricesRead  = "purchase_prices:read"
	PermStockWrite          = "stock:write"
	PermSalesRead           = "sales:read"
	PermSalesWrite          = "sales:write"
	PermAnalyticsRead       = "analytics:read"
	PermFinancialRead       = "financial:read"
	PermAuditRead           = "audit:read"
	PermUsersRead           = "users:read"
	PermUsersWrite          = "users:write"
	PermRolesRead           = "roles:read"
	PermRolesWrite          = "roles:write"
	PermOrdersRead          = "orders:read"
	PermOrdersWrite         = "orders:write"
	PermReserveRead         = "reserve:read"
	PermReserveWrite        = "reserve:write"
	PermPaymentsRead        = "payments:read"
	PermPaymentsWrite       = "payments:write"
	PermDebtsRead           = "debts:read"
	PermCRMRead             = "crm:read"
	PermCRMWrite            = "crm:write"
	PermCashboxesRead       = "cashboxes:read"
	PermCashboxesWrite      = "cashboxes:write"
	PermCashOpsRead         = "cashops:read"
	PermCashOpsWrite        = "cashops:write"
	PermCashShiftsRead      = "cash_shifts:read"
	PermCashShiftsWrite     = "cash_shifts:write"
	PermExchangeRatesRead   = "exchange_rates:read"
	PermExchangeRatesWrite  = "exchange_rates:write"
	PermSuppliersRead       = "suppliers:read"
	PermSuppliersWrite      = "suppliers:write"
	PermSupplierOrdersRead  = "supplier_orders:read"
	PermSupplierOrdersWrite = "supplier_orders:write"
	PermPurchasesRead       = "purchases:read"
	PermPurchasesWrite      = "purchases:write"
	PermWarehousesRead      = "warehouses:read"
	PermWarehousesWrite     = "warehouses:write"
	PermTransfersRead       = "transfers:read"
	PermTransfersWrite      = "transfers:write"
	PermInventoryRead       = "inventory:read"
	PermInventoryWrite      = "inventory:write"
	PermNotificationsRead   = "notifications:read"
	PermNotificationsWrite  = "notifications:write"
	PermJobsRead            = "jobs:read"
	PermJobsWrite           = "jobs:write"
	PermDocumentsRead       = "documents:read"
	PermDocumentsWrite      = "documents:write"
	PermDocumentsPrint      = "documents:print"
	PermDocTemplatesRead    = "doc_templates:read"
	PermDocTemplatesWrite   = "doc_templates:write"
)

func AllPermissions() []string {
	return []string{
		PermProductsRead,
		PermProductsWrite,
		PermPurchasePricesRead,
		PermStockWrite,
		PermSalesRead,
		PermSalesWrite,
		PermAnalyticsRead,
		PermFinancialRead,
		PermAuditRead,
		PermUsersRead,
		PermUsersWrite,
		PermRolesRead,
		PermRolesWrite,
		PermOrdersRead,
		PermOrdersWrite,
		PermReserveRead,
		PermReserveWrite,
		PermPaymentsRead,
		PermPaymentsWrite,
		PermDebtsRead,
		PermCRMRead,
		PermCRMWrite,
		PermCashboxesRead,
		PermCashboxesWrite,
		PermCashOpsRead,
		PermCashOpsWrite,
		PermCashShiftsRead,
		PermCashShiftsWrite,
		PermExchangeRatesRead,
		PermExchangeRatesWrite,
		PermSuppliersRead,
		PermSuppliersWrite,
		PermSupplierOrdersRead,
		PermSupplierOrdersWrite,
		PermPurchasesRead,
		PermPurchasesWrite,
		PermWarehousesRead,
		PermWarehousesWrite,
		PermTransfersRead,
		PermTransfersWrite,
		PermInventoryRead,
		PermInventoryWrite,
		PermNotificationsRead,
		PermNotificationsWrite,
		PermJobsRead,
		PermJobsWrite,
		PermDocumentsRead,
		PermDocumentsWrite,
		PermDocumentsPrint,
		PermDocTemplatesRead,
		PermDocTemplatesWrite,
	}
}

func defaultRolePermissions() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"admin": {
			PermProductsRead:        {},
			PermProductsWrite:       {},
			PermPurchasePricesRead:  {},
			PermStockWrite:          {},
			PermSalesRead:           {},
			PermSalesWrite:          {},
			PermAnalyticsRead:       {},
			PermFinancialRead:       {},
			PermAuditRead:           {},
			PermUsersRead:           {},
			PermUsersWrite:          {},
			PermRolesRead:           {},
			PermRolesWrite:          {},
			PermOrdersRead:          {},
			PermOrdersWrite:         {},
			PermReserveRead:         {},
			PermReserveWrite:        {},
			PermPaymentsRead:        {},
			PermPaymentsWrite:       {},
			PermDebtsRead:           {},
			PermCRMRead:             {},
			PermCRMWrite:            {},
			PermCashboxesRead:       {},
			PermCashboxesWrite:      {},
			PermCashOpsRead:         {},
			PermCashOpsWrite:        {},
			PermCashShiftsRead:      {},
			PermCashShiftsWrite:     {},
			PermExchangeRatesRead:   {},
			PermExchangeRatesWrite:  {},
			PermSuppliersRead:       {},
			PermSuppliersWrite:      {},
			PermSupplierOrdersRead:  {},
			PermSupplierOrdersWrite: {},
			PermPurchasesRead:       {},
			PermPurchasesWrite:      {},
			PermWarehousesRead:      {},
			PermWarehousesWrite:     {},
			PermTransfersRead:       {},
			PermTransfersWrite:      {},
			PermInventoryRead:       {},
			PermInventoryWrite:      {},
			PermNotificationsRead:   {},
			PermNotificationsWrite:  {},
			PermJobsRead:            {},
			PermJobsWrite:           {},
			PermDocumentsRead:       {},
			PermDocumentsWrite:      {},
			PermDocumentsPrint:      {},
			PermDocTemplatesRead:    {},
			PermDocTemplatesWrite:   {},
		},
		"manager": {
			PermProductsRead:        {},
			PermProductsWrite:       {},
			PermPurchasePricesRead:  {},
			PermStockWrite:          {},
			PermSalesRead:           {},
			PermSalesWrite:          {},
			PermAnalyticsRead:       {},
			PermFinancialRead:       {},
			PermAuditRead:           {},
			PermOrdersRead:          {},
			PermOrdersWrite:         {},
			PermReserveRead:         {},
			PermReserveWrite:        {},
			PermPaymentsRead:        {},
			PermPaymentsWrite:       {},
			PermDebtsRead:           {},
			PermCRMRead:             {},
			PermCRMWrite:            {},
			PermCashboxesRead:       {},
			PermCashboxesWrite:      {},
			PermCashOpsRead:         {},
			PermCashOpsWrite:        {},
			PermCashShiftsRead:      {},
			PermCashShiftsWrite:     {},
			PermExchangeRatesRead:   {},
			PermExchangeRatesWrite:  {},
			PermSuppliersRead:       {},
			PermSuppliersWrite:      {},
			PermSupplierOrdersRead:  {},
			PermSupplierOrdersWrite: {},
			PermPurchasesRead:       {},
			PermPurchasesWrite:      {},
			PermWarehousesRead:      {},
			PermWarehousesWrite:     {},
			PermTransfersRead:       {},
			PermTransfersWrite:      {},
			PermInventoryRead:       {},
			PermInventoryWrite:      {},
			PermNotificationsRead:   {},
			PermNotificationsWrite:  {},
			PermJobsRead:            {},
			PermJobsWrite:           {},
			PermDocumentsRead:       {},
			PermDocumentsWrite:      {},
			PermDocumentsPrint:      {},
			PermDocTemplatesRead:    {},
			PermDocTemplatesWrite:   {},
		},
		"seller": {
			PermProductsRead:       {},
			PermStockWrite:         {},
			PermSalesRead:          {},
			PermSalesWrite:         {},
			PermAnalyticsRead:      {},
			PermOrdersRead:         {},
			PermOrdersWrite:        {},
			PermReserveRead:        {},
			PermReserveWrite:       {},
			PermPaymentsRead:       {},
			PermPaymentsWrite:      {},
			PermDebtsRead:          {},
			PermCRMRead:            {},
			PermCRMWrite:           {},
			PermCashboxesRead:      {},
			PermCashOpsRead:        {},
			PermCashShiftsRead:     {},
			PermCashShiftsWrite:    {},
			PermExchangeRatesRead:  {},
			PermSuppliersRead:      {},
			PermSupplierOrdersRead: {},
			PermPurchasesRead:      {},
			PermWarehousesRead:     {},
			PermTransfersRead:      {},
			PermTransfersWrite:     {},
			PermInventoryRead:      {},
			PermNotificationsRead:  {},
			PermJobsRead:           {},
			PermDocumentsRead:      {},
			PermDocumentsWrite:     {},
			PermDocTemplatesRead:   {},
		},
		"accountant": {
			PermProductsRead:       {},
			PermPurchasePricesRead: {},
			PermSalesRead:          {},
			PermAnalyticsRead:      {},
			PermFinancialRead:      {},
			PermAuditRead:          {},
			PermOrdersRead:         {},
			PermPaymentsRead:       {},
			PermDebtsRead:          {},
			PermCashboxesRead:      {},
			PermCashOpsRead:        {},
			PermCashShiftsRead:     {},
			PermExchangeRatesRead:  {},
			PermSuppliersRead:      {},
			PermSupplierOrdersRead: {},
			PermPurchasesRead:      {},
			PermWarehousesRead:     {},
			PermDocumentsRead:      {},
			PermDocumentsPrint:     {},
			PermDocTemplatesRead:   {},
			PermNotificationsRead:  {},
		},
		"buyer": {
			PermOrdersRead:  {},
			PermDebtsRead:   {},
			PermPaymentsRead: {},
		},
	}
}
