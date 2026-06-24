package service

import (
	"strings"

	"gkk/expect"
	"gkk/model/user"
	"gkk/orm"

	"github.com/gkk/stall-location/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplicationService struct {
	DB  *gorm.DB
	Now clock
}

type ReviewApplicationRequest struct {
	ReviewReason string `json:"review_reason" validate:"required"`
}

func (s *ApplicationService) Approve(operator *user.Info, id uint, reason string) error {
	return withTx(s.DB, func(tx *gorm.DB) error {
		app := orm.First[Application](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id))
		if app.Id == 0 {
			return notFound("入驻申请不存在")
		}
		if app.Status == ApplicationApproved {
			return conflict(expect.CodeAuditAlreadyApproved, "申请已审核通过")
		}
		merchant, err := s.ensureMerchant(tx, app)
		if err != nil {
			return err
		}
		if _, err := ensureMerchantSharePoster(tx, merchant); err != nil {
			return err
		}
		now := serviceNow(s.Now)
		app.Status = ApplicationApproved
		app.ReviewReason = strings.TrimSpace(reason)
		app.ReviewerID = &operator.Id
		app.ReviewedAt = &now
		app.MerchantID = &merchant.Id
		fields := []string{"status", "review_reason", "reviewer_id", "reviewed_at", "merchant_id"}
		if err := expect.NDM(tx.Model(&Application{}).Where("id = ?", app.Id).Select(fields).UpdateColumns(app), "审核通过失败"); err != nil {
			return err
		}
		patch := model.User{
			MerchantID: &merchant.Id,
			PageMode:   IdentityMerchant,
			Info: user.Info{
				Simple:   user.Simple{Username: merchant.Phone, Avatar: merchant.AvatarURL},
				Phone:    merchant.Phone,
				Nickname: merchant.DisplayName,
			},
		}
		if err := expect.NDM(
			tx.Model(&model.User{}).Where("id = ?", app.UserID).Select("merchant_id", "page_mode", "username", "phone", "nickname", "avatar").Updates(patch),
			"绑定商户身份失败",
		); err != nil {
			return err
		}
		return nil
	})
}

func (s *ApplicationService) Reject(operator *user.Info, id uint, reason string) error {
	return withTx(s.DB, func(tx *gorm.DB) error {
		app := orm.First[Application](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id))
		if app.Id == 0 {
			return notFound("入驻申请不存在")
		}
		if app.Status == ApplicationApproved {
			return conflict(expect.CodeAuditAlreadyApproved, "已通过的申请不能改为其他状态")
		}
		now := serviceNow(s.Now)
		app.Status = ApplicationRejected
		app.ReviewReason = reason
		app.ReviewerID = &operator.Id
		app.ReviewedAt = &now
		fields := []string{"status", "review_reason", "reviewer_id", "reviewed_at"}
		if err := expect.NDM(tx.Model(&Application{}).Where("id = ?", app.Id).Select(fields).UpdateColumns(app), "更新审核状态失败"); err != nil {
			return err
		}
		if err := updateUserPageMode(tx, app.UserID, IdentityApplication, nil); err != nil {
			return err
		}
		return nil
	})
}

func updateUserPageMode(tx *gorm.DB, userID uint, pageMode string, merchantID *uint) error {
	patch := model.User{PageMode: pageMode, MerchantID: merchantID}
	fields := []string{"page_mode"}
	if merchantID != nil {
		fields = append(fields, "merchant_id")
	}
	return expect.NDM(tx.Model(&model.User{}).Where("id = ?", userID).Select(fields).Updates(patch), "更新用户页面状态失败")
}

func (s *ApplicationService) ensureMerchant(tx *gorm.DB, app Application) (Merchant, error) {
	var merchant Merchant
	tx.Where("user_id = ?", app.UserID).First(&merchant)
	if merchant.Id == 0 {
		merchant.UserID = app.UserID
		merchant.Status = StatusActive
		merchant.VerifyStatus = VerifyVerified
		merchant.DisplayName = app.MerchantName
		merchant.Phone = app.ContactPhone
		merchant.ContactPhone = app.ContactPhone
		merchant.Category = app.Category
		merchant.AvatarURL = app.PhotoURL
		if db := tx.Create(&merchant); db.Error != nil {
			return Merchant{}, expect.NDM(db, "创建商户失败")
		}
	}
	return merchant, nil
}
