package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/storyvows/backend/integrations"
)

type WhatsAppService struct {
	cfg    *integrations.Secrets
	client *http.Client
}

func NewWhatsAppService(cfg *integrations.Secrets) *WhatsAppService {
	return &WhatsAppService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *WhatsAppService) Send(ctx context.Context, phoneNumber, message string) error {
	if phoneNumber == "" {
		return errors.New("phone_number is required")
	}
	if message == "" {
		return errors.New("message is required")
	}

	if s.cfg.TwilioAccountSID != "" && s.cfg.TwilioAuthToken != "" && s.cfg.TwilioWhatsAppFrom != "" {
		return s.sendViaTwilio(ctx, phoneNumber, message)
	}

	if s.cfg.WhatsAppPhoneNumberID != "" && s.cfg.WhatsAppAccessToken != "" {
		return s.sendViaMeta(ctx, phoneNumber, message)
	}

	return errors.New("whatsapp configuration is missing")
}

func (s *WhatsAppService) SendTwilio(ctx context.Context, phoneNumber, message string) error {
	if phoneNumber == "" {
		return errors.New("phone_number is required")
	}
	if message == "" {
		return errors.New("message is required")
	}
	if s.cfg.TwilioAccountSID == "" || s.cfg.TwilioAuthToken == "" || s.cfg.TwilioWhatsAppFrom == "" {
		return errors.New("twilio whatsapp configuration is missing")
	}
	return s.sendViaTwilio(ctx, phoneNumber, message)
}

func (s *WhatsAppService) SendTemplate(ctx context.Context, phoneNumber, contentSid string, contentVariables map[string]string) error {
	if phoneNumber == "" {
		return errors.New("phone_number is required")
	}
	if contentSid == "" {
		return errors.New("content_sid is required")
	}
	if s.cfg.TwilioAccountSID == "" || s.cfg.TwilioAuthToken == "" || s.cfg.TwilioWhatsAppFrom == "" {
		return errors.New("twilio whatsapp configuration is missing")
	}
	return s.sendViaTwilioTemplate(ctx, phoneNumber, contentSid, contentVariables)
}

func (s *WhatsAppService) sendViaMeta(ctx context.Context, phoneNumber, message string) error {
	endpoint := fmt.Sprintf("%s/%s/messages", strings.TrimRight(s.cfg.WhatsAppAPIURL, "/"), s.cfg.WhatsAppPhoneNumberID)
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                normalizePhoneNumber(phoneNumber),
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal whatsapp payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create whatsapp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.WhatsAppAccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send whatsapp message via meta: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("meta whatsapp error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *WhatsAppService) sendViaTwilio(ctx context.Context, phoneNumber, message string) error {
	form := url.Values{}
	form.Set("To", normalizeWhatsAppNumber(phoneNumber))
	form.Set("From", normalizeWhatsAppNumber(s.cfg.TwilioWhatsAppFrom))
	form.Set("Body", message)

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.cfg.TwilioAccountSID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create whatsapp request: %w", err)
	}
	req.SetBasicAuth(s.cfg.TwilioAccountSID, s.cfg.TwilioAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send whatsapp message via twilio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio whatsapp error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *WhatsAppService) sendViaTwilioTemplate(ctx context.Context, phoneNumber, contentSid string, contentVariables map[string]string) error {
	form := url.Values{}
	form.Set("To", normalizeWhatsAppNumber(phoneNumber))
	form.Set("From", normalizeWhatsAppNumber(s.cfg.TwilioWhatsAppFrom))
	form.Set("ContentSid", contentSid)
	if contentVariables != nil {
		varsJSON, err := json.Marshal(contentVariables)
		if err != nil {
			return fmt.Errorf("marshal content variables: %w", err)
		}
		form.Set("ContentVariables", string(varsJSON))
	}

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.cfg.TwilioAccountSID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create whatsapp template request: %w", err)
	}
	req.SetBasicAuth(s.cfg.TwilioAccountSID, s.cfg.TwilioAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send whatsapp template via twilio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio whatsapp template error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func normalizeWhatsAppNumber(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "whatsapp:") {
		value = "whatsapp:" + value
	}
	return value
}

func normalizePhoneNumber(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.TrimPrefix(value, "whatsapp:")
	return value
}
