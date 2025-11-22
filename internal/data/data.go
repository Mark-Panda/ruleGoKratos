package data

import (
	"fmt"
	"ruleGoKratos/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewDB, NewData, NewRuleGoRepo)

// Data .
type Data struct {
	// TODO wrapped database client
	db *gorm.DB
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	db, err := NewDB(c)
	if err != nil {
		return nil, nil, err
	}
	return &Data{db: db}, cleanup, nil
}

func NewDB(config *conf.Data) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s timezone=%s", config.Postgres.Host, config.Postgres.User, config.Postgres.Password, config.Postgres.Dbname, config.Postgres.Port, config.Postgres.Sslmode, config.Postgres.Timezone)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
