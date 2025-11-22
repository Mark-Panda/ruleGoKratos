package data

import (
	"fmt"
	"ruleGoKratos/internal/conf"
	"ruleGoKratos/internal/data/dao"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/rulego/rulego"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewDB, NewData, NewRuleGoRepo, NewRuleEngine)

var DBClient *gorm.DB

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
	DBClient = db
	dao.Init(db)
	return db, nil
}

// 初始化规则引擎
func NewRuleEngine(c *conf.Data) (*rulego.RuleGo, error) {
	var err error
	// 获取所有的规则链信息
	var regulations []Regulation
	if err := DBClient.Table("regulation").Where("disabled = ?", false).Find(&regulations).Error; err != nil {
		return nil, err
	}
	pool := rulego.NewRuleGo()
	// 加载所有规则链
	for _, regulation := range regulations {
		// 如果规则链已经存在，重新加载规则配置
		if ruleEngine, ok := pool.Get(regulation.RuleChainID); ok {
			fmt.Println("重新加载规则配置", regulation.RuleChainID)
			err = ruleEngine.ReloadSelf([]byte(regulation.RuleConfig))
		} else {
			fmt.Println("加载规则链", regulation.RuleChainID)
			_, err = pool.New(regulation.RuleChainID, []byte(regulation.RuleConfig))
		}
		if err != nil {
			// 更新规则配置状态为禁用
			DBClient.Table("regulation").Where("id = ?", regulation.ID).Update("disabled", true)
		}
	}
	return pool, nil
}

type Regulation struct {
	ID          int64  `gorm:"column:id"`
	UserName    string `gorm:"column:user_name"`
	Root        bool   `gorm:"column:root"`
	Disabled    bool   `gorm:"column:disabled"`
	Name        string `gorm:"column:name"`
	RuleChainID string `gorm:"column:rule_chain_id"`
	RuleVersion int    `gorm:"column:rule_version"`
	RuleConfig  string `gorm:"column:rule_config"`
}
