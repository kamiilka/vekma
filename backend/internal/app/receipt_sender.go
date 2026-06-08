package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ReceiptSendResult struct {
	ExternalID   string
	FiscalNumber string
	QRURL        string
}

type ReceiptSender interface {
	Enabled() bool
	SendSaleReceipt(sale Sale) (ReceiptSendResult, error)
}

type CheckboxReceiptSender struct {
	apiURL     string
	apiToken   string
	httpClient *http.Client
}

func NewCheckboxReceiptSenderFromEnv() *CheckboxReceiptSender {
	apiURL := strings.TrimSpace(os.Getenv("CHECKBOX_API_URL"))
	apiURL = strings.TrimRight(apiURL, "/")
	return &CheckboxReceiptSender{
		apiURL:   apiURL,
		apiToken: strings.TrimSpace(os.Getenv("CHECKBOX_API_TOKEN")),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *CheckboxReceiptSender) Enabled() bool {
	return c != nil && c.apiURL != "" && c.apiToken != ""
}

func (c *CheckboxReceiptSender) SendSaleReceipt(sale Sale) (ReceiptSendResult, error) {
	if c == nil {
		return ReceiptSendResult{}, errors.New("receipt sender is nil")
	}
	if !c.Enabled() {
		return ReceiptSendResult{}, errors.New("receipt sender is not configured")
	}

	payload := map[string]any{
		"saleId":   sale.ID,
		"currency": sale.Currency,
		"total":    sale.Total,
		"items":    sale.Items,
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return ReceiptSendResult{}, err
	}

	req, err := http.NewRequest(http.MethodPost, c.apiURL+"/receipts", bytes.NewReader(rawPayload))
	if err != nil {
		return ReceiptSendResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ReceiptSendResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return ReceiptSendResult{}, fmt.Errorf(
			"checkbox request failed with status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	type checkboxReceiptResponse struct {
		ID           string `json:"id"`
		ExternalID   string `json:"externalId"`
		FiscalCode   string `json:"fiscalCode"`
		FiscalNumber string `json:"fiscalNumber"`
		QRURL        string `json:"qrUrl"`
		QRCode       string `json:"qrCode"`
	}
	var result checkboxReceiptResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil && !errors.Is(err, io.EOF) {
		return ReceiptSendResult{}, err
	}

	return ReceiptSendResult{
		ExternalID:   firstNonEmpty(result.ExternalID, result.ID),
		FiscalNumber: firstNonEmpty(result.FiscalNumber, result.FiscalCode),
		QRURL:        firstNonEmpty(result.QRURL, result.QRCode),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
