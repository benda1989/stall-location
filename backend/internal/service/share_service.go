package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"gkk/config"
	"gkk/expect"
	uploadali "gkk/handler/upload/ali"
	"gkk/handler/wx"
	wxservice "gkk/handler/wx/service"
	"gkk/orm"

	"github.com/gkk/stall-location/backend/internal/conf"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

const (
	merchantQRCodeChannelH5   = "h5"
	merchantQRCodeChannelMini = "mini_program"
)

// ensureMerchantSharePoster is an explicit business step for creation or
// approval flows. Plain profile reads must not call it.
func ensureMerchantSharePoster(db *gorm.DB, merchant Merchant) (Merchant, error) {
	merchant, err := ensureMerchantShareCode(db, merchant)
	if err != nil {
		return Merchant{}, err
	}
	shareURL := conf.ShareURL(merchant.ShareCode)
	merchant.ShareURL = shareURL
	desiredChannel := merchantShareQRCodeChannel()
	if strings.TrimSpace(merchant.SharePosterURL) != "" &&
		strings.TrimSpace(merchant.ShareQRCodeChannel) == desiredChannel {
		return merchant, nil
	}
	image, channel, err := buildMerchantShareQRCode(merchant.ShareCode, shareURL)
	if err != nil {
		return Merchant{}, err
	}
	patch := Merchant{
		SharePosterURL:     image,
		ShareQRCodeChannel: channel,
	}
	db = db.Model(&Merchant{}).
		Where("id = ?", merchant.Id).
		Select("share_poster_url", "share_qrcode_channel").
		Updates(patch)
	if err := expect.NDM(db, "更新商户分享图失败"); err != nil {
		return Merchant{}, err
	}
	merchant.SharePosterURL = patch.SharePosterURL
	merchant.ShareQRCodeChannel = patch.ShareQRCodeChannel
	return merchant, nil
}

func ensureMerchantShareCode(db *gorm.DB, merchant Merchant) (Merchant, error) {
	if strings.TrimSpace(merchant.ShareCode) != "" {
		return merchant, nil
	}
	var lastErr error
	for range 3 {
		next := nextShareCode()
		result := db.Model(&Merchant{}).
			Where("id = ? AND (share_code IS NULL OR share_code = '')", merchant.Id).
			UpdateColumn("share_code", next)
		if result.Error != nil {
			lastErr = result.Error
			continue
		}
		if result.RowsAffected > 0 {
			merchant.ShareCode = next
			return merchant, nil
		}
		merchant = orm.First[Merchant](db.Where("id = ?", merchant.Id))
		if strings.TrimSpace(merchant.ShareCode) != "" {
			return merchant, nil
		}
	}
	if lastErr != nil {
		return Merchant{}, expect.Wrap(lastErr, http.StatusInternalServerError, expect.CodeCommonInternalError, "生成商户分享码失败")
	}
	return Merchant{}, expect.New(http.StatusInternalServerError, expect.CodeCommonInternalError, "生成商户分享码失败")
}

func nextShareCode() string {
	return randomCode("", 6)
}

func buildMerchantShareQRCode(scene, fallbackURL string) (string, string, error) {
	if !wx.MiniReady() {
		data, err := qrcode.Encode(fallbackURL, qrcode.Medium, 256)
		if err != nil {
			return "", "", err
		}
		return fixedShareImage(scene, merchantQRCodeChannelH5, data), merchantQRCodeChannelH5, nil
	}
	mini := wxservice.NewOauth(config.C.Auth.Mini.Appid, config.C.Auth.Mini.Secret)
	data, err := mini.GetWxaCodeUnlimit(wxservice.WxaCodeUnlimitRequest{
		Scene:      scene,
		Page:       "pages/customer/index",
		EnvVersion: "release",
	})
	if err != nil {
		return "", "", expect.Wrap(err, http.StatusBadGateway, expect.CodeWxLoginUnavailable, "生成小程序码失败")
	}
	return fixedShareImage(scene, merchantQRCodeChannelMini, data), merchantQRCodeChannelMini, nil
}

func fixedShareImage(scene, channel string, data []byte) string {
	name := fmt.Sprintf("user/nearby/share/%s-%s%s", scene, channel, shareImageExt(data))
	if url := uploadali.UploadBuffer(name, bytes.NewReader(data)); url != "" {
		return url
	}
	mime := "image/png"
	if shareImageExt(data) == ".jpg" {
		mime = "image/jpeg"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func merchantShareQRCodeChannel() string {
	if wx.MiniReady() {
		return merchantQRCodeChannelMini
	}
	return merchantQRCodeChannelH5
}

func shareImageExt(data []byte) string {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return ".jpg"
	}
	if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return ".png"
	}
	return ".png"
}
