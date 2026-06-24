package conf

import (
	"fmt"
	"strings"

	gkk "gkk"
)

var C CustomConfig

const (
	DefaultMaxMerchantProducts  = 50
	DefaultMaxCustomerFavorites = 50
)

// CustomConfig maps config.yaml:custom. Keep project knobs here and leave the
// top-level gkk config namespace for framework settings.
type CustomConfig struct {
	BaseURL              string    `yaml:"share_url" json:"base_url"`
	SeedDemoData         bool      `yaml:"seed_demo_data" json:"seed_demo_data"`
	CustomerAuth         string    `yaml:"customer_auth" json:"customer_auth"`
	Frontend             string    `yaml:"frontend" json:"frontend"`
	MaxMerchantProducts  int       `yaml:"max_merchant_products" json:"max_merchant_products"`
	MaxCustomerFavorites int       `yaml:"max_customer_favorites" json:"max_customer_favorites"`
	SMS                  SMSConfig `yaml:"sms" json:"sms"`
}

type SMSConfig struct {
	Provider        string `yaml:"provider" json:"provider"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret" json:"access_key_secret"`
	TemplateCode    string `yaml:"template_code" json:"template_code"`
	SignName        string `yaml:"sign_name" json:"sign_name"`
}

func MaxMerchantProducts() int {
	if C.MaxMerchantProducts <= 0 {
		return DefaultMaxMerchantProducts
	}
	return C.MaxMerchantProducts
}

func MaxCustomerFavorites() int {
	if C.MaxCustomerFavorites <= 0 {
		return DefaultMaxCustomerFavorites
	}
	return C.MaxCustomerFavorites
}

func ShareURL(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(C.BaseURL), "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	return fmt.Sprintf("%s/share/%s", base, code)
}

func Init() {
	gkk.Init("config.yaml", &C)
}
