package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gkk/stall-location/backend/internal/models"
	"gorm.io/gorm"
)

type systemRoleRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
	MenuIDs     []uint `json:"menu_ids"`
}

type systemUserRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
	Status   string `json:"status"`
	RoleIDs  []uint `json:"role_ids"`
}

type systemMenuRequest struct {
	ParentID   *uint  `json:"parent_id"`
	Code       string `json:"code"`
	Name       string `json:"name" binding:"required"`
	Path       string `json:"path"`
	Icon       string `json:"icon"`
	Type       string `json:"type"`
	Permission string `json:"permission"`
	Status     string `json:"status"`
	SortOrder  int    `json:"sort_order"`
}

type systemUserDTO struct {
	models.User
	Roles []models.SystemRole `json:"roles"`
}

func (s *Server) AdminListSystemRoles(c *gin.Context) {
	var roles []models.SystemRole
	query := s.DB.Order("sort_order asc, id asc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&roles).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	if err := s.attachMenusToRoles(roles); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (s *Server) AdminCreateSystemRole(c *gin.Context) {
	var req systemRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		abort(c, http.StatusBadRequest, errors.New("code is required"))
		return
	}
	role := models.SystemRole{Code: req.Code, Name: req.Name, Description: req.Description, Status: defaultStatus(req.Status), SortOrder: req.SortOrder}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		return replaceRoleMenus(tx, role.ID, req.MenuIDs)
	}); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	roleRows := []models.SystemRole{role}
	_ = s.attachMenusToRoles(roleRows)
	role = roleRows[0]
	c.JSON(http.StatusCreated, gin.H{"role": role})
}

func (s *Server) AdminUpdateSystemRole(c *gin.Context) {
	var role models.SystemRole
	if err := s.DB.First(&role, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var req systemRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"name": req.Name, "description": req.Description, "status": defaultStatus(req.Status), "sort_order": req.SortOrder}
		if strings.TrimSpace(req.Code) != "" {
			updates["code"] = strings.TrimSpace(req.Code)
		}
		if err := tx.Model(&role).Updates(updates).Error; err != nil {
			return err
		}
		return replaceRoleMenus(tx, role.ID, req.MenuIDs)
	}); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	_ = s.DB.First(&role, role.ID).Error
	roleRows := []models.SystemRole{role}
	_ = s.attachMenusToRoles(roleRows)
	role = roleRows[0]
	c.JSON(http.StatusOK, gin.H{"role": role})
}

func (s *Server) AdminListSystemUsers(c *gin.Context) {
	var users []models.User
	query := s.DB.Where("role = ?", models.RoleAdmin).Order("CASE WHEN status = 'disabled' THEN 1 ELSE 0 END, updated_at desc, id asc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&users).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	out, err := s.buildSystemUserDTOs(users)
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

func (s *Server) AdminCreateSystemUser(c *gin.Context) {
	var req systemUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	user := models.User{Role: models.RoleAdmin, Phone: strings.TrimSpace(req.Phone), Nickname: strings.TrimSpace(req.Nickname), Status: defaultStatus(req.Status)}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return replaceUserRoles(tx, user.ID, req.RoleIDs)
	}); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	out, err := s.buildSystemUserDTOs([]models.User{user})
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": out[0]})
}

func (s *Server) AdminUpdateSystemUser(c *gin.Context) {
	var user models.User
	if err := s.DB.Where("role = ?", models.RoleAdmin).First(&user, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var req systemUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"phone": strings.TrimSpace(req.Phone), "nickname": strings.TrimSpace(req.Nickname), "status": defaultStatus(req.Status)}
		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
		return replaceUserRoles(tx, user.ID, req.RoleIDs)
	}); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	_ = s.DB.First(&user, user.ID).Error
	out, err := s.buildSystemUserDTOs([]models.User{user})
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": out[0]})
}

func (s *Server) AdminListSystemMenus(c *gin.Context) {
	var menus []models.SystemMenu
	query := s.DB.Order("sort_order asc, id asc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&menus).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"menus": menus, "tree": buildMenuTree(menus)})
}

func (s *Server) AdminCreateSystemMenu(c *gin.Context) {
	var req systemMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		abort(c, http.StatusBadRequest, errors.New("code is required"))
		return
	}
	menu := models.SystemMenu{ParentID: req.ParentID, Code: req.Code, Name: req.Name, Path: req.Path, Icon: req.Icon, Type: defaultString(req.Type, "menu"), Permission: req.Permission, Status: defaultStatus(req.Status), SortOrder: req.SortOrder}
	if err := s.DB.Create(&menu).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"menu": menu})
}

func (s *Server) AdminUpdateSystemMenu(c *gin.Context) {
	var menu models.SystemMenu
	if err := s.DB.First(&menu, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var req systemMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	updates := map[string]any{"parent_id": req.ParentID, "name": req.Name, "path": req.Path, "icon": req.Icon, "type": defaultString(req.Type, "menu"), "permission": req.Permission, "status": defaultStatus(req.Status), "sort_order": req.SortOrder}
	if strings.TrimSpace(req.Code) != "" {
		updates["code"] = strings.TrimSpace(req.Code)
	}
	if err := s.DB.Model(&menu).Updates(updates).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	_ = s.DB.First(&menu, menu.ID).Error
	c.JSON(http.StatusOK, gin.H{"menu": menu})
}

func (s *Server) attachMenusToRoles(roles []models.SystemRole) error {
	if len(roles) == 0 {
		return nil
	}
	roleIDs := make([]uint, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}
	var links []models.SystemRoleMenu
	if err := s.DB.Where("role_id IN ?", roleIDs).Find(&links).Error; err != nil {
		return err
	}
	menuIDs := make([]uint, 0, len(links))
	roleMenuIDs := map[uint][]uint{}
	for _, link := range links {
		menuIDs = append(menuIDs, link.MenuID)
		roleMenuIDs[link.RoleID] = append(roleMenuIDs[link.RoleID], link.MenuID)
	}
	var menus []models.SystemMenu
	if len(menuIDs) > 0 {
		if err := s.DB.Where("id IN ?", menuIDs).Order("sort_order asc, id asc").Find(&menus).Error; err != nil {
			return err
		}
	}
	menuByID := map[uint]models.SystemMenu{}
	for _, menu := range menus {
		menuByID[menu.ID] = menu
	}
	for i := range roles {
		for _, menuID := range roleMenuIDs[roles[i].ID] {
			if menu, ok := menuByID[menuID]; ok {
				roles[i].Menus = append(roles[i].Menus, menu)
			}
		}
	}
	return nil
}

func (s *Server) buildSystemUserDTOs(users []models.User) ([]systemUserDTO, error) {
	if len(users) == 0 {
		return []systemUserDTO{}, nil
	}
	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	var links []models.SystemUserRole
	if err := s.DB.Where("user_id IN ?", userIDs).Find(&links).Error; err != nil {
		return nil, err
	}
	roleIDs := make([]uint, 0, len(links))
	userRoleIDs := map[uint][]uint{}
	for _, link := range links {
		roleIDs = append(roleIDs, link.RoleID)
		userRoleIDs[link.UserID] = append(userRoleIDs[link.UserID], link.RoleID)
	}
	var roles []models.SystemRole
	if len(roleIDs) > 0 {
		if err := s.DB.Where("id IN ?", roleIDs).Order("sort_order asc, id asc").Find(&roles).Error; err != nil {
			return nil, err
		}
	}
	roleByID := map[uint]models.SystemRole{}
	for _, role := range roles {
		roleByID[role.ID] = role
	}
	out := make([]systemUserDTO, 0, len(users))
	for _, user := range users {
		dto := systemUserDTO{User: user, Roles: []models.SystemRole{}}
		for _, roleID := range userRoleIDs[user.ID] {
			if role, ok := roleByID[roleID]; ok {
				dto.Roles = append(dto.Roles, role)
			}
		}
		out = append(out, dto)
	}
	return out, nil
}

func replaceRoleMenus(tx *gorm.DB, roleID uint, menuIDs []uint) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&models.SystemRoleMenu{}).Error; err != nil {
		return err
	}
	for _, menuID := range uniqueUintIDs(menuIDs) {
		if err := tx.Create(&models.SystemRoleMenu{RoleID: roleID, MenuID: menuID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func replaceUserRoles(tx *gorm.DB, userID uint, roleIDs []uint) error {
	if err := tx.Where("user_id = ?", userID).Delete(&models.SystemUserRole{}).Error; err != nil {
		return err
	}
	for _, roleID := range uniqueUintIDs(roleIDs) {
		if err := tx.Create(&models.SystemUserRole{UserID: userID, RoleID: roleID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func uniqueUintIDs(ids []uint) []uint {
	seen := map[uint]bool{}
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func buildMenuTree(menus []models.SystemMenu) []models.SystemMenu {
	children := map[uint][]models.SystemMenu{}
	roots := []models.SystemMenu{}
	for _, menu := range menus {
		if menu.ParentID != nil && *menu.ParentID != 0 {
			children[*menu.ParentID] = append(children[*menu.ParentID], menu)
			continue
		}
		roots = append(roots, menu)
	}
	var attach func([]models.SystemMenu) []models.SystemMenu
	attach = func(rows []models.SystemMenu) []models.SystemMenu {
		out := make([]models.SystemMenu, 0, len(rows))
		for _, row := range rows {
			row.Children = attach(children[row.ID])
			out = append(out, row)
		}
		return out
	}
	return attach(roots)
}

func defaultStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == models.UserStatusDisabled {
		return models.UserStatusDisabled
	}
	return models.UserStatusActive
}

func parseUintParam(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	return uint(id), err
}
