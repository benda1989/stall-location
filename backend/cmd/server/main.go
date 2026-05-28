package main

import (
	"log"

	"github.com/gkk/stall-location/backend/internal/api"
	"github.com/gkk/stall-location/backend/internal/config"
	"github.com/gkk/stall-location/backend/internal/db"
)

func main() {
	cfg := config.Load()
	conn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := db.AutoMigrate(conn); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if cfg.SeedDemoData {
		if err := db.SeedDemoData(conn); err != nil {
			log.Fatalf("seed demo data: %v", err)
		}
	}
	r := api.NewRouter(conn, cfg)
	log.Printf("backend listening on %s", cfg.Addr)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
