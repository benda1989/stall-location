package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gkk/stall-location/backend/internal/config"
	"github.com/gkk/stall-location/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	}

	switch cfg.DatabaseDriver {
	case "postgres", "postgresql":
		return gorm.Open(postgres.Open(cfg.DatabaseDSN), gormCfg)
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.DatabaseDSN), 0o755); err != nil && filepath.Dir(cfg.DatabaseDSN) != "." {
			return nil, err
		}
		return gorm.Open(sqlite.Open(cfg.DatabaseDSN), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.DatabaseDriver)
	}
}

func AutoMigrate(conn *gorm.DB) error {
	return conn.AutoMigrate(
		&models.User{},
		&models.SystemRole{},
		&models.SystemMenu{},
		&models.SystemUserRole{},
		&models.SystemRoleMenu{},
		&models.Merchant{},
		&models.Shop{},
		&models.Product{},
		&models.StallSession{},
		&models.Order{},
		&models.OrderItem{},
		&models.ShopQRCode{},
		&models.MerchantApplication{},
		&models.SMSCode{},
		&models.Feedback{},
	)
}
