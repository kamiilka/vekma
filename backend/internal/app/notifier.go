package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type NotificationSender interface {
	Enabled() bool
	DefaultRecipient(channel string) string
	Send(channel, recipient, subject, body string) error
	SendFrom(channel, sender, recipient, subject, body string) error
}

type ProviderNotifier struct {
	smtpAddr string
	smtpUser string
	smtpPass string
	smtpFrom string
	emailTo  string

	telegramToken  string
	telegramChatID string

	// SMS via generic HTTP gateway (e.g. Turbosms, Nexmo, Twilio)
	// POST url with JSON body: {"phone": recipient, "message": body}
	smsGatewayURL   string
	smsGatewayToken string
	smsPhoneTo      string

	// Viber via Viber Bot API
	viberToken    string
	viberRecipient string

	httpClient *http.Client
}

func NewProviderNotifierFromEnv() *ProviderNotifier {
	return &ProviderNotifier{
		smtpAddr:        strings.TrimSpace(os.Getenv("SMTP_ADDR")),
		smtpUser:        strings.TrimSpace(os.Getenv("SMTP_USER")),
		smtpPass:        strings.TrimSpace(os.Getenv("SMTP_PASS")),
		smtpFrom:        strings.TrimSpace(os.Getenv("SMTP_FROM")),
		emailTo:         strings.TrimSpace(os.Getenv("NOTIFY_EMAIL_TO")),
		telegramToken:   strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		telegramChatID:  strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID")),
		smsGatewayURL:   strings.TrimSpace(os.Getenv("SMS_GATEWAY_URL")),
		smsGatewayToken: strings.TrimSpace(os.Getenv("SMS_GATEWAY_TOKEN")),
		smsPhoneTo:      strings.TrimSpace(os.Getenv("SMS_PHONE_TO")),
		viberToken:      strings.TrimSpace(os.Getenv("VIBER_BOT_TOKEN")),
		viberRecipient:  strings.TrimSpace(os.Getenv("VIBER_RECIPIENT_ID")),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewProviderNotifierFromConfig(cfg NotificationConfig) *ProviderNotifier {
	return &ProviderNotifier{
		smtpAddr:        strings.TrimSpace(cfg.SMTPAddr),
		smtpUser:        strings.TrimSpace(cfg.SMTPUser),
		smtpPass:        strings.TrimSpace(cfg.SMTPPass),
		smtpFrom:        strings.TrimSpace(cfg.SMTPFrom),
		emailTo:         "",
		telegramToken:   strings.TrimSpace(cfg.TelegramToken),
		telegramChatID:  strings.TrimSpace(cfg.TelegramChatID),
		smsGatewayURL:   strings.TrimSpace(cfg.SMSGatewayURL),
		smsGatewayToken: strings.TrimSpace(cfg.SMSGatewayToken),
		smsPhoneTo:      strings.TrimSpace(cfg.SMSPhoneTo),
		viberToken:      strings.TrimSpace(cfg.ViberToken),
		viberRecipient:  strings.TrimSpace(cfg.ViberRecipient),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *ProviderNotifier) Enabled() bool {
	return p != nil && (p.smtpAddr != "" || p.telegramToken != "" || p.smsGatewayURL != "" || p.viberToken != "")
}

func (p *ProviderNotifier) DefaultRecipient(channel string) string {
	if p == nil {
		return ""
	}
	switch channel {
	case NotificationChannelEmail:
		return p.emailTo
	case NotificationChannelTelegram:
		return p.telegramChatID
	case NotificationChannelSMS:
		return p.smsPhoneTo
	case NotificationChannelViber:
		return p.viberRecipient
	default:
		return ""
	}
}

func (p *ProviderNotifier) Send(channel, recipient, subject, body string) error {
	return p.SendFrom(channel, "", recipient, subject, body)
}

func (p *ProviderNotifier) SendFrom(channel, sender, recipient, subject, body string) error {
	if p == nil {
		return errors.New("notifier is nil")
	}
	switch channel {
	case NotificationChannelEmail:
		return p.sendEmail(sender, recipient, subject, body)
	case NotificationChannelTelegram:
		return p.sendTelegram(recipient, body)
	case NotificationChannelSMS:
		return p.sendSMS(sender, recipient, body)
	case NotificationChannelViber:
		return p.sendViber(recipient, body)
	default:
		return fmt.Errorf("unsupported notification channel: %s", channel)
	}
}

func (p *ProviderNotifier) sendEmail(fromOverride, recipient, subject, body string) error {
	if p.smtpAddr == "" || p.smtpFrom == "" {
		return errors.New("smtp is not configured")
	}
	if recipient == "" {
		return errors.New("email recipient is required")
	}

	from := p.smtpFrom
	if strings.TrimSpace(fromOverride) != "" {
		from = strings.TrimSpace(fromOverride)
	}

	host := p.smtpAddr
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	var auth smtp.Auth
	if p.smtpUser != "" {
		auth = smtp.PlainAuth("", p.smtpUser, p.smtpPass, host)
	}

	message := bytes.Buffer{}
	message.WriteString(fmt.Sprintf("From: %s\r\n", from))
	message.WriteString(fmt.Sprintf("To: %s\r\n", recipient))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(body)

	return smtp.SendMail(
		p.smtpAddr,
		auth,
		from,
		[]string{recipient},
		message.Bytes(),
	)
}

func (p *ProviderNotifier) sendTelegram(recipient, body string) error {
	if p.telegramToken == "" {
		return errors.New("telegram bot token is not configured")
	}
	chatID := recipient
	if chatID == "" {
		chatID = p.telegramChatID
	}
	if chatID == "" {
		return errors.New("telegram chat id is required")
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", p.telegramToken)
	payload := map[string]string{
		"chat_id": chatID,
		"text":    body,
	}
	raw, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram send failed with status %d", resp.StatusCode)
	}
	return nil
}

func (p *ProviderNotifier) sendSMS(senderOverride, recipient, body string) error {
	if p.smsGatewayURL == "" {
		return errors.New("SMS gateway is not configured (set SMS_GATEWAY_URL)")
	}
	phone := recipient
	if phone == "" {
		phone = p.smsPhoneTo
	}
	if phone == "" {
		return errors.New("SMS recipient phone number is required")
	}

	payload := map[string]string{
		"phone":   phone,
		"message": body,
	}
	if strings.TrimSpace(senderOverride) != "" {
		payload["sender"] = strings.TrimSpace(senderOverride)
	}
	if p.smsGatewayToken != "" {
		payload["token"] = p.smsGatewayToken
	}
	raw, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, p.smsGatewayURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.smsGatewayToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.smsGatewayToken)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SMS gateway returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *ProviderNotifier) sendViber(recipient, body string) error {
	if p.viberToken == "" {
		return errors.New("Viber bot token is not configured (set VIBER_BOT_TOKEN)")
	}
	to := recipient
	if to == "" {
		to = p.viberRecipient
	}
	if to == "" {
		return errors.New("Viber recipient ID is required")
	}

	// Viber REST API: https://chatapi.viber.com/pa/send_message
	payload := map[string]interface{}{
		"receiver": to,
		"min_api_version": 1,
		"type":     "text",
		"text":     body,
	}
	raw, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, "https://chatapi.viber.com/pa/send_message", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Viber-Auth-Token", p.viberToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if status, ok := result["status"].(float64); ok && status != 0 {
		msg, _ := result["status_message"].(string)
		return fmt.Errorf("Viber API error %d: %s", int(status), msg)
	}
	return nil
}
