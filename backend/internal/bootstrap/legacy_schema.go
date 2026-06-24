package bootstrap

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// PrepareLegacySchema makes old local databases safe for the current gkk_service
// AutoMigrate pass. It only touches columns that would otherwise fail when added
// as NOT NULL to tables that already contain rows.
func PrepareLegacySchema(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := prepareLegacyMerchants(db); err != nil {
		return err
	}
	if err := prepareLegacyMerchantOwnedTables(db); err != nil {
		return err
	}
	return nil
}

func prepareLegacyMerchants(db *gorm.DB) error {
	if !db.Migrator().HasTable("merchants") {
		return nil
	}
	if !db.Migrator().HasColumn("merchants", "share_code") {
		if err := db.Exec(`ALTER TABLE merchants ADD COLUMN share_code varchar(32)`).Error; err != nil {
			return fmt.Errorf("add merchants.share_code: %w", err)
		}
	}
	if !db.Migrator().HasColumn("merchants", "share_poster_url") {
		if err := db.Exec(`ALTER TABLE merchants ADD COLUMN share_poster_url text`).Error; err != nil {
			return fmt.Errorf("add merchants.share_poster_url: %w", err)
		}
	}
	if !db.Migrator().HasColumn("merchants", "share_qrcode_channel") {
		if err := db.Exec(`ALTER TABLE merchants ADD COLUMN share_qrcode_channel varchar(32)`).Error; err != nil {
			return fmt.Errorf("add merchants.share_qrcode_channel: %w", err)
		}
	}
	if db.Migrator().HasColumn("merchants", "share_qr_code_channel") {
		if err := db.Exec(`
			UPDATE merchants
			SET share_qrcode_channel = share_qr_code_channel
			WHERE (share_qrcode_channel IS NULL OR btrim(share_qrcode_channel) = '')
				AND share_qr_code_channel IS NOT NULL
				AND btrim(share_qr_code_channel) <> ''
		`).Error; err != nil {
			return fmt.Errorf("backfill merchants.share_qrcode_channel: %w", err)
		}
	}
	if db.Migrator().HasColumn("merchants", "share_qrcode_url") {
		if err := db.Exec(`
			UPDATE merchants
			SET share_poster_url = share_qrcode_url
			WHERE (share_poster_url IS NULL OR btrim(share_poster_url) = '')
				AND share_qrcode_url IS NOT NULL
				AND btrim(share_qrcode_url) <> ''
		`).Error; err != nil {
			return fmt.Errorf("backfill merchants.share_poster_url from share_qrcode_url: %w", err)
		}
	}
	if db.Migrator().HasTable("merchant_qr_codes") {
		if err := db.Exec(`
			WITH first_codes AS (
				SELECT merchant_id, min(code) AS code
				FROM merchant_qr_codes
				WHERE code IS NOT NULL AND btrim(code) <> ''
				GROUP BY merchant_id
			)
			UPDATE merchants
			SET share_code = first_codes.code
			FROM first_codes
			WHERE first_codes.merchant_id = merchants.id
				AND (merchants.share_code IS NULL OR btrim(merchants.share_code) = '')
		`).Error; err != nil {
			return fmt.Errorf("backfill merchants.share_code from merchant_qr_codes: %w", err)
		}
	}
	if err := db.Exec(`
		UPDATE merchants
		SET share_code = concat('S', id)
		WHERE share_code IS NULL OR btrim(share_code) = ''
	`).Error; err != nil {
		return fmt.Errorf("backfill merchants.share_code: %w", err)
	}
	if db.Migrator().HasColumn("merchants", "display_name") {
		if err := db.Exec(`
			UPDATE merchants
			SET display_name = concat('商户 #', id)
			WHERE display_name IS NULL OR btrim(display_name) = ''
		`).Error; err != nil {
			return fmt.Errorf("backfill merchants.display_name: %w", err)
		}
	}
	return ensureMerchantShareCodeIndex(db)
}

func ensureMerchantShareCodeIndex(db *gorm.DB) error {
	indexName := "idx_merchants_share_code"
	if db.Migrator().HasIndex("merchants", indexName) {
		return nil
	}
	err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_merchants_share_code ON merchants (share_code)`).Error
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create merchants.share_code index: %w", err)
	}
	return nil
}

func prepareLegacyMerchantOwnedTables(db *gorm.DB) error {
	for _, table := range []string{"products", "stall_sessions", "preorders"} {
		if err := prepareLegacyMerchantID(db, table); err != nil {
			return err
		}
	}
	if err := prepareLegacyFavorites(db); err != nil {
		return err
	}
	if err := prepareLegacyApplications(db); err != nil {
		return err
	}
	return nil
}

func prepareLegacyMerchantID(db *gorm.DB, table string) error {
	if !db.Migrator().HasTable(table) {
		return nil
	}
	if db.Migrator().HasColumn(table, "shop_id") {
		if err := dropLegacyNotNull(db, table, "shop_id"); err != nil {
			return err
		}
	}
	if db.Migrator().HasColumn(table, "merchant_id") {
		return nil
	}
	if err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN merchant_id bigint`, table)).Error; err != nil {
		return fmt.Errorf("add %s.merchant_id: %w", table, err)
	}
	if db.Migrator().HasTable("shops") && db.Migrator().HasColumn(table, "shop_id") {
		if err := db.Exec(fmt.Sprintf(`
			UPDATE %s
			SET merchant_id = shops.merchant_id
			FROM shops
			WHERE %s.shop_id = shops.id
				AND (%s.merchant_id IS NULL OR %s.merchant_id = 0)
		`, table, table, table, table)).Error; err != nil {
			return fmt.Errorf("backfill %s.merchant_id from shops: %w", table, err)
		}
	}
	return backfillFirstMerchantID(db, table)
}

func prepareLegacyFavorites(db *gorm.DB) error {
	if !db.Migrator().HasTable("customer_favorites") {
		return nil
	}
	if !db.Migrator().HasColumn("customer_favorites", "user_id") {
		if err := db.Exec(`ALTER TABLE customer_favorites ADD COLUMN user_id bigint`).Error; err != nil {
			return fmt.Errorf("add customer_favorites.user_id: %w", err)
		}
		if db.Migrator().HasColumn("customer_favorites", "user_id") {
			if err := db.Exec(`
				UPDATE customer_favorites
				SET user_id = user_id
				WHERE user_id IS NULL OR user_id = 0
			`).Error; err != nil {
				return fmt.Errorf("backfill customer_favorites.user_id: %w", err)
			}
		}
	}
	if db.Migrator().HasColumn("customer_favorites", "user_id") {
		if err := dropLegacyNotNull(db, "customer_favorites", "user_id"); err != nil {
			return err
		}
	}
	return prepareLegacyMerchantID(db, "customer_favorites")
}

func prepareLegacyApplications(db *gorm.DB) error {
	if !db.Migrator().HasTable("applications") {
		return nil
	}
	if !db.Migrator().HasColumn("applications", "merchant_name") {
		if err := db.Exec(`ALTER TABLE applications ADD COLUMN merchant_name varchar(120)`).Error; err != nil {
			return fmt.Errorf("add applications.merchant_name: %w", err)
		}
	}
	if db.Migrator().HasColumn("applications", "shop_name") {
		if err := db.Exec(`
			UPDATE applications
			SET merchant_name = shop_name
			WHERE merchant_name IS NULL OR btrim(merchant_name) = ''
		`).Error; err != nil {
			return fmt.Errorf("backfill applications.merchant_name: %w", err)
		}
		if err := dropLegacyNotNull(db, "applications", "shop_name"); err != nil {
			return err
		}
	}
	if err := db.Exec(`
		UPDATE applications
		SET merchant_name = concat('商户申请 #', id)
		WHERE merchant_name IS NULL OR btrim(merchant_name) = ''
	`).Error; err != nil {
		return fmt.Errorf("fallback applications.merchant_name: %w", err)
	}
	return nil
}

func dropLegacyNotNull(db *gorm.DB, table, column string) error {
	err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL`, table, column)).Error
	if err != nil {
		return fmt.Errorf("drop %s.%s not null: %w", table, column, err)
	}
	return nil
}

func backfillFirstMerchantID(db *gorm.DB, table string) error {
	if err := db.Exec(fmt.Sprintf(`
		UPDATE %s
		SET merchant_id = (SELECT id FROM merchants ORDER BY id ASC LIMIT 1)
		WHERE (merchant_id IS NULL OR merchant_id = 0)
			AND EXISTS (SELECT 1 FROM merchants)
	`, table)).Error; err != nil {
		return fmt.Errorf("fallback %s.merchant_id: %w", table, err)
	}
	return nil
}
