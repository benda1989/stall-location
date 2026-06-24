package service

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

type clock func() time.Time

func serviceNow(c clock) time.Time {
	if c != nil {
		return c()
	}
	return time.Now()
}

func withTx(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return validation("数据库连接未初始化")
	}
	return db.Transaction(fn)
}

func randomCode(prefix string, n int) string {
	if n <= 0 {
		n = 6
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
	}
	code := strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "=")
	if len(code) > n {
		code = code[:n]
	}
	return prefix + code
}

func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0
	rad := func(v float64) float64 { return v * math.Pi / 180 }
	dLat := rad(lat2 - lat1)
	dLng := rad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func ptrTime(t time.Time) *time.Time { return &t }

func ptrUint(v uint) *uint { return &v }
