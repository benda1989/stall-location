package main

import (
	"gkk/orm"
	"gkk/tool/grace"
	"log"

	"github.com/gkk/stall-location/backend/internal/bootstrap"
	"github.com/gkk/stall-location/backend/internal/conf"
	"github.com/gkk/stall-location/backend/internal/model"
)

func main() {
	conf.Init()
	model.Init()

	if conf.C.SeedDemoData {
		if err := bootstrap.PrepareLegacySchema(orm.DB); err != nil {
			log.Fatalf("prepare legacy schema: %v", err)
		}
		if err := bootstrap.SeedDemoData(orm.DB); err != nil {
			log.Fatalf("seed gkk demo data: %v", err)
		}
	}
	grace.Init(bootstrap.Register)
	grace.Run()
}
