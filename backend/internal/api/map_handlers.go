package api

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gkk/stall-location/backend/internal/models"
)

type stallMapItem struct {
	Shop             models.Shop         `json:"shop"`
	StallSession     models.StallSession `json:"stall_session"`
	Products         []models.Product    `json:"products"`
	DistanceMeters   *int                `json:"distance_meters,omitempty"`
	WalkMinutes      *int                `json:"walk_minutes,omitempty"`
	DisplayStatus    string              `json:"display_status"`
	EntryMode        string              `json:"entry_mode"`
	LocationAccuracy int                 `json:"location_accuracy"`
	LastOnlineAt     time.Time           `json:"last_online_at"`
}

type mapBounds struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLng float64 `json:"min_lng"`
	MaxLng float64 `json:"max_lng"`
}

func (s *Server) NearbyStalls(c *gin.Context) {
	s.expireSessions()
	lat, lng, hasUserLocation := parseLatLng(c)
	limit := parseLimit(c.Query("limit"), 50)
	searchText := strings.TrimSpace(c.Query("q"))
	includeRecent := c.Query("include_recent") == "1" || searchText != ""
	bounds, hasBounds := parseMapBounds(c)
	categories := parseCategoryFilter(c)
	zoom := strings.TrimSpace(c.Query("zoom"))

	var sessions []models.StallSession
	now := time.Now()
	query := s.DB.Preload("Shop").
		Joins("JOIN shops ON shops.id = stall_sessions.shop_id").
		Where("shops.status = ?", models.ShopStatusActive)
	if hasBounds {
		query = query.Where("stall_sessions.lat BETWEEN ? AND ? AND stall_sessions.lng BETWEEN ? AND ?", bounds.MinLat, bounds.MaxLat, bounds.MinLng, bounds.MaxLng)
	}
	if len(categories) > 0 {
		query = query.Where("shops.category IN ?", categories)
	}
	if searchText != "" {
		like := "%" + searchText + "%"
		query = query.Where(
			"(shops.name LIKE ? OR shops.category LIKE ? OR shops.announcement LIKE ? OR stall_sessions.address LIKE ?)",
			like,
			like,
			like,
			like,
		)
	}
	if includeRecent {
		cutoff := now.Add(-72 * time.Hour)
		query = query.Where(
			"(stall_sessions.status = ? AND stall_sessions.expected_end_at > ?) OR stall_sessions.updated_at >= ? OR stall_sessions.started_at >= ? OR stall_sessions.ended_at >= ?",
			models.StallStatusActive,
			now,
			cutoff,
			cutoff,
			cutoff,
		).Order("CASE WHEN stall_sessions.status = 'active' AND stall_sessions.expected_end_at > CURRENT_TIMESTAMP THEN 0 ELSE 1 END ASC, stall_sessions.updated_at DESC, stall_sessions.started_at DESC").
			Limit(limit * 4)
	} else {
		query = query.Where("stall_sessions.status = ? AND stall_sessions.expected_end_at > ?", models.StallStatusActive, now).
			Order("stall_sessions.started_at DESC").
			Limit(limit)
	}
	err := query.Find(&sessions).Error
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	if includeRecent {
		sessions = uniqueLatestSessionsByShop(sessions, limit)
	}

	items := make([]stallMapItem, 0, len(sessions))
	for _, session := range sessions {
		item := s.stallItem(session, "nearby", lat, lng, hasUserLocation)
		items = append(items, item)
	}
	if hasUserLocation {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].DistanceMeters == nil || items[j].DistanceMeters == nil {
				return items[i].DistanceMeters != nil
			}
			return *items[i].DistanceMeters < *items[j].DistanceMeters
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"entry_mode": "nearby",
		"load_mode":  mapLoadMode(hasBounds),
		"bounds":     boundsPayload(bounds, hasBounds),
		"zoom":       zoom,
		"q":          searchText,
		"categories": categories,
		"stalls":     items,
		"count":      len(items),
		"notice":     mapNotice(includeRecent, hasBounds),
	})
}

func (s *Server) ShopMapState(c *gin.Context) {
	shop, err := s.findShopByCode(c.Param("shopCode"))
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	lat, lng, hasUserLocation := parseLatLng(c)
	session, err := s.activeSession(shop.ID)
	products := s.hotProducts(shop.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"entry_mode":     "focused",
			"shop":           shop,
			"stall_session":  nil,
			"products":       products,
			"display_status": "unavailable",
			"notice":         "该摊主当前未出摊；不会展示历史实时位置，也不会推荐其他摊主。",
		})
		return
	}
	item := s.stallItem(session, "focused", lat, lng, hasUserLocation)
	c.JSON(http.StatusOK, gin.H{
		"entry_mode":      "focused",
		"shop":            shop,
		"stall_session":   session,
		"products":        products,
		"distance_meters": item.DistanceMeters,
		"walk_minutes":    item.WalkMinutes,
		"display_status":  item.DisplayStatus,
		"last_online_at":  item.LastOnlineAt,
		"notice":          "单摊导航模式只显示该摊主当前位置；位置由摊主主动更新，距离仅供参考。",
	})
}

func (s *Server) OrderLocation(c *gin.Context) {
	var order models.Order
	if err := s.DB.Preload("Shop").Where("order_no = ?", c.Param("orderNo")).First(&order).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var snapshot map[string]any
	if len(order.LocationSnapshot) > 0 {
		_ = json.Unmarshal(order.LocationSnapshot, &snapshot)
	}
	var current any
	if session, err := s.activeSession(order.ShopID); err == nil {
		current = session
	}
	c.JSON(http.StatusOK, gin.H{
		"order_no":          order.OrderNo,
		"shop":              order.Shop,
		"location_snapshot": snapshot,
		"current_location":  current,
		"notice":            "取货位置优先以下单时位置快照为准；当前公开位置可能已变化。",
	})
}

func (s *Server) stallItem(session models.StallSession, mode string, userLat float64, userLng float64, hasUserLocation bool) stallMapItem {
	active := session.Status == models.StallStatusActive && session.ExpectedEndAt.After(time.Now())
	displayStatus := "recent"
	if active {
		displayStatus = "active"
	}
	item := stallMapItem{
		Shop:             session.Shop,
		StallSession:     session,
		Products:         s.hotProducts(session.ShopID),
		DisplayStatus:    displayStatus,
		EntryMode:        mode,
		LocationAccuracy: session.LocationAccuracy,
		LastOnlineAt:     sessionLastOnlineAt(session),
	}
	if hasUserLocation {
		distance := int(math.Round(haversineMeters(userLat, userLng, session.Lat, session.Lng)))
		walk := int(math.Max(1, math.Round(float64(distance)/80.0)))
		item.DistanceMeters = &distance
		item.WalkMinutes = &walk
	}
	return item
}

func uniqueLatestSessionsByShop(sessions []models.StallSession, limit int) []models.StallSession {
	seen := make(map[uint]bool, len(sessions))
	result := make([]models.StallSession, 0, limit)
	for _, session := range sessions {
		if seen[session.ShopID] {
			continue
		}
		seen[session.ShopID] = true
		result = append(result, session)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func sessionLastOnlineAt(session models.StallSession) time.Time {
	if session.EndedAt != nil && !session.EndedAt.IsZero() {
		return *session.EndedAt
	}
	if !session.UpdatedAt.IsZero() {
		return session.UpdatedAt
	}
	return session.StartedAt
}

func mapNotice(includeRecent bool, hasBounds bool) string {
	if includeRecent {
		if hasBounds {
			return "已按当前地图视野加载；搜索时会展示最近 3 天在线过的摊位。"
		}
		return "搜索时会展示最近 3 天在线过的摊位；位置以摊主最后更新为准。"
	}
	if hasBounds {
		return "已按当前地图视野加载营业摊位；移动或缩放地图会自动刷新。"
	}
	return "位置由摊主主动更新，距离仅供参考"
}

func (s *Server) hotProducts(shopID uint) []models.Product {
	var products []models.Product
	_ = s.DB.Where("shop_id = ? AND status = ?", shopID, models.ProductStatusOnSale).
		Order("sort_order ASC, id ASC").Limit(3).Find(&products).Error
	return products
}

func parseLatLng(c *gin.Context) (float64, float64, bool) {
	lat, latErr := strconv.ParseFloat(c.Query("lat"), 64)
	lng, lngErr := strconv.ParseFloat(c.Query("lng"), 64)
	if latErr != nil || lngErr != nil || lat == 0 || lng == 0 {
		return 0, 0, false
	}
	return lat, lng, true
}

func parseLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 100 {
		return 100
	}
	return value
}

func parseMapBounds(c *gin.Context) (mapBounds, bool) {
	raw := []string{c.Query("min_lat"), c.Query("max_lat"), c.Query("min_lng"), c.Query("max_lng")}
	hasAny := false
	for _, value := range raw {
		if strings.TrimSpace(value) != "" {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return mapBounds{}, false
	}
	minLat, err1 := strconv.ParseFloat(raw[0], 64)
	maxLat, err2 := strconv.ParseFloat(raw[1], 64)
	minLng, err3 := strconv.ParseFloat(raw[2], 64)
	maxLng, err4 := strconv.ParseFloat(raw[3], 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return mapBounds{}, false
	}
	if minLat > maxLat {
		minLat, maxLat = maxLat, minLat
	}
	if minLng > maxLng {
		minLng, maxLng = maxLng, minLng
	}
	if minLat < -90 || maxLat > 90 || minLng < -180 || maxLng > 180 || minLat == maxLat || minLng == maxLng {
		return mapBounds{}, false
	}
	return mapBounds{MinLat: minLat, MaxLat: maxLat, MinLng: minLng, MaxLng: maxLng}, true
}

func parseCategoryFilter(c *gin.Context) []string {
	rawValues := append([]string{}, c.QueryArray("category")...)
	if aliases := c.Query("categories"); aliases != "" {
		rawValues = append(rawValues, aliases)
	}
	seen := make(map[string]bool)
	categories := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		for _, item := range strings.Split(raw, ",") {
			category := strings.TrimSpace(item)
			if category == "" || category == "全部" || seen[category] {
				continue
			}
			seen[category] = true
			categories = append(categories, category)
		}
	}
	sort.Strings(categories)
	return categories
}

func mapLoadMode(hasBounds bool) string {
	if hasBounds {
		return "bounds"
	}
	return "nearby"
}

func boundsPayload(bounds mapBounds, ok bool) any {
	if !ok {
		return nil
	}
	return bounds
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	rLat1 := toRad(lat1)
	rLat2 := toRad(lat2)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
