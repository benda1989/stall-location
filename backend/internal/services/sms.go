package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type SMSConfig struct {
	Provider        string
	AccessKeyID     string
	AccessKeySecret string
	TemplateCode    string
	SignName        string
	RegionID        string
}

func (cfg SMSConfig) Enabled() bool {
	return cfg.Provider == "aliyun" && cfg.AccessKeyID != "" && cfg.AccessKeySecret != "" && cfg.TemplateCode != "" && cfg.SignName != ""
}

func SendVerificationCode(ctx context.Context, cfg SMSConfig, phone string, code string) error {
	if !cfg.Enabled() {
		return nil
	}
	if cfg.Provider != "aliyun" {
		return fmt.Errorf("unsupported sms provider: %s", cfg.Provider)
	}
	return sendAliyunVerificationCode(ctx, cfg, phone, code)
}

type aliyunSMSResponse struct {
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestID string `json:"RequestId"`
	BizID     string `json:"BizId"`
}

func sendAliyunVerificationCode(ctx context.Context, cfg SMSConfig, phone string, code string) error {
	templateParam, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return err
	}
	query := map[string]string{
		"PhoneNumbers":  phone,
		"SignName":      cfg.SignName,
		"TemplateCode":  cfg.TemplateCode,
		"TemplateParam": string(templateParam),
	}
	if cfg.RegionID != "" {
		query["RegionId"] = cfg.RegionID
	}

	const endpoint = "dysmsapi.aliyuncs.com"
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	nonce := RandomCode("", 16)
	payloadHash := sha256Hex(nil)
	canonicalQuery := canonicalQueryString(query)
	signedHeaders := "host;x-acs-action;x-acs-content-sha256;x-acs-date;x-acs-signature-nonce;x-acs-version"
	canonicalHeaders := strings.Join([]string{
		"host:" + endpoint,
		"x-acs-action:SendSms",
		"x-acs-content-sha256:" + payloadHash,
		"x-acs-date:" + now,
		"x-acs-signature-nonce:" + nonce,
		"x-acs-version:2017-05-25",
		"",
	}, "\n")
	canonicalRequest := strings.Join([]string{
		http.MethodPost,
		"/",
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	stringToSign := "ACS3-HMAC-SHA256\n" + sha256Hex([]byte(canonicalRequest))
	signature := hmacSHA256Hex([]byte(cfg.AccessKeySecret), []byte(stringToSign))
	authorization := "ACS3-HMAC-SHA256 Credential=" + cfg.AccessKeyID + ",SignedHeaders=" + signedHeaders + ",Signature=" + signature

	reqURL := "https://" + endpoint + "/?" + canonicalQuery
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Host", endpoint)
	req.Header.Set("X-Acs-Action", "SendSms")
	req.Header.Set("X-Acs-Content-Sha256", payloadHash)
	req.Header.Set("X-Acs-Date", now)
	req.Header.Set("X-Acs-Signature-Nonce", nonce)
	req.Header.Set("X-Acs-Version", "2017-05-25")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send aliyun sms: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("send aliyun sms: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out aliyunSMSResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decode aliyun sms response: %w", err)
	}
	if out.Code != "OK" {
		if out.Message == "" {
			out.Message = "unknown aliyun sms error"
		}
		return errors.New(out.Message)
	}
	return nil
}

func canonicalQueryString(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, aliyunPercentEncode(key)+"="+aliyunPercentEncode(values[key]))
	}
	return strings.Join(pairs, "&")
}

func aliyunPercentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256Hex(secret []byte, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
