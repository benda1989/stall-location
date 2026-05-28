package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Addr            string
	DatabaseDriver  string
	DatabaseDSN     string
	FrontendURL     string
	BaseURL         string
	TokenSecret     string
	SeedDemoData    bool
	GinMode         string
	WeChatAppID     string
	WeChatAppSecret string
	WeChatTicket    string
	SMS             SMSConfig
}

type SMSConfig struct {
	Provider        string
	AccessKeyID     string
	AccessKeySecret string
	TemplateCode    string
	SignName        string
	RegionID        string
}

func Load() Config {
	loadDotEnv(".env", "backend/.env")
	cfg := Config{
		Addr:            env("APP_ADDR", ":8080"),
		DatabaseDriver:  strings.ToLower(env("APP_DATABASE_DRIVER", "sqlite")),
		DatabaseDSN:     env("APP_DATABASE_DSN", "data/stall.db"),
		FrontendURL:     env("APP_FRONTEND_URL", "http://localhost:5173"),
		BaseURL:         env("APP_BASE_URL", "http://localhost:5173"),
		TokenSecret:     env("APP_TOKEN_SECRET", "dev-secret-change-me"),
		GinMode:         env("GIN_MODE", "debug"),
		WeChatAppID:     env("WECHAT_APP_ID", ""),
		WeChatAppSecret: env("WECHAT_APP_SECRET", ""),
		WeChatTicket:    env("WECHAT_JSAPI_TICKET", ""),
		SMS: SMSConfig{
			Provider:        strings.ToLower(firstEnv("APP_SMS_PROVIDER", "SMS_PROVIDER")),
			AccessKeyID:     firstEnv("ALIYUN_SMS_ACCESS_KEY_ID", "ALIYUN_SMS_KEY", "SMS_KEY"),
			AccessKeySecret: firstEnv("ALIYUN_SMS_ACCESS_KEY_SECRET", "ALIYUN_SMS_SECRET", "SMS_SECRET"),
			TemplateCode:    firstEnv("ALIYUN_SMS_TEMPLATE_CODE", "ALIYUN_SMS_CODE", "SMS_CODE"),
			SignName:        firstEnv("ALIYUN_SMS_SIGN_NAME", "ALIYUN_SMS_OWNER", "SMS_OWNER"),
			RegionID:        env("ALIYUN_SMS_REGION_ID", "cn-hangzhou"),
		},
	}
	if cfg.SMS.Provider == "" && cfg.SMS.AccessKeyID != "" && cfg.SMS.AccessKeySecret != "" && cfg.SMS.TemplateCode != "" && cfg.SMS.SignName != "" {
		cfg.SMS.Provider = "aliyun"
	}
	cfg.SeedDemoData = strings.ToLower(env("APP_SEED_DEMO", "true")) != "false"
	return cfg
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func loadDotEnv(paths ...string) {
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		func() {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				key, value, ok := strings.Cut(line, "=")
				if !ok {
					continue
				}
				key = strings.TrimSpace(key)
				if key == "" || os.Getenv(key) != "" {
					continue
				}
				value = strings.TrimSpace(value)
				value = strings.Trim(value, "\"'")
				_ = os.Setenv(key, value)
			}
		}()
	}
}
