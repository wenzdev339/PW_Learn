package dbtest

import (
	"backend/internal/config"
	"backend/internal/db"

	"gorm.io/gorm"
)

// Connect opens a connection to the test database and ensures its schema is
// up to date. Call once per package in TestMain and reuse the returned
// *gorm.DB across that package's tests.
func Connect() *gorm.DB {
	cfg := config.TestConfig()
	gdb, err := db.Connect(cfg)
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(gdb); err != nil {
		panic(err)
	}
	return gdb
}
