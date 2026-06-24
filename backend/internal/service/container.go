package service

import (
	"gkk/tool/cron"

	"gorm.io/gorm"
)

// Container owns the concrete business services that share one DB.
type Container struct {
	DB          *gorm.DB
	Stall       *StallService
	Preorder    *PreorderService
	Application *ApplicationService
	Feedback    *FeedbackService
}

// NewContainer centralizes concrete service construction for route wiring.
func NewContainer(db *gorm.DB) *Container {
	container := &Container{
		DB:          db,
		Stall:       &StallService{DB: db},
		Preorder:    &PreorderService{DB: db},
		Application: &ApplicationService{DB: db},
		Feedback:    &FeedbackService{DB: db},
	}
	container.Stall.ExpireSessions()
	cron.Run("@every 1m", container.Stall.ExpireSessions)
	return container
}
