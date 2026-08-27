package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/blaiseva001-cloud/backend/internal/config"
	"github.com/blaiseva001-cloud/backend/internal/models"
)

func ConnectPostgres(cfg *config.Config) *gorm.DB {
	g, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		log.Fatal("[postgres] ", err)
	}
	sqlDB, _ := g.DB()
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	log.Println("[postgres] connected")
	return g
}

func Migrate(g *gorm.DB) {
	if err := g.AutoMigrate(&models.User{}, &models.Site{}, &models.Link{}, &models.PageView{}, &models.LinkClick{}); err != nil {
		log.Fatal("[migrate] ", err)
	}
	log.Println("[migrate] schema ready")
}
