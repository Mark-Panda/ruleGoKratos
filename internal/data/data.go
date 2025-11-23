package data

import (
	"fmt"
	"ruleGoKratos/internal/conf"
	_ "ruleGoKratos/internal/data/components"
	"ruleGoKratos/internal/data/dao"

	_ "github.com/rulego/rulego-components-ai/ai/action"
	_ "github.com/rulego/rulego-components-ai/ai/endpoint"

	_ "github.com/rulego/rulego-components/endpoint/redis"
	_ "github.com/rulego/rulego-components/external/redis"
	_ "github.com/rulego/rulego-components/filter"
	_ "github.com/rulego/rulego-components/transform"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/engine"
	"github.com/rulego/rulego/node_pool"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewDB, NewData, NewRuleEngine, NewComponentUseRuleRepo, NewComponentRegulationRepo, NewMdWorkflowRepo, NewRunLogRepo, NewRegulationRepo)

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

const (
	LogModeSilent = `silent`
	LogModeError  = `error`
	LogModeWarn   = `warn`
	LogModeInfo   = `info`
)

func LogModeString2GormLogLevel(mode string) gormLogger.LogLevel {
	switch mode {
	case LogModeSilent:
		return gormLogger.Silent
	case LogModeError:
		return gormLogger.Error
	case LogModeWarn:
		return gormLogger.Warn
	case LogModeInfo:
		return gormLogger.Info
	default:
		return gormLogger.Warn
	}
}

func NewDB(config *conf.Data) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s timezone=%s", config.Postgres.Host, config.Postgres.User, config.Postgres.Password, config.Postgres.Dbname, config.Postgres.Port, config.Postgres.Sslmode, config.Postgres.Timezone)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         gormLogger.Default.LogMode(LogModeString2GormLogLevel("info")),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
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
	componentRegistry := engine.NewCustomComponentRegistry(engine.Registry, new(engine.RuleComponentRegistry))
	ruleConfig := rulego.NewConfig(types.WithDefaultPool(),
		// types.WithLogger(logger.Logger),
		types.WithComponentsRegistry(componentRegistry),
		types.WithNodePool(node_pool.DefaultNodePool))
	// ruleConfig.Logger.Printf("init %s data", username)
	pool := rulego.NewRuleGo()
	// 加载所有规则链
	for _, regulation := range regulations {
		// 如果规则链已经存在，重新加载规则配置
		if ruleEngine, ok := pool.Get(regulation.RuleChainID); ok {
			fmt.Println("重新加载规则配置", regulation.RuleChainID)
			err = ruleEngine.ReloadSelf([]byte(regulation.RuleConfig))
		} else {
			fmt.Println("加载规则链", regulation.RuleChainID)
			_, err = pool.New(regulation.RuleChainID, []byte(regulation.RuleConfig), rulego.WithConfig(ruleConfig))
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
