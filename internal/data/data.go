package data

import (
	"encoding/json"
	"fmt"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/conf"
	rulegodatacomp "ruleGoKratos/internal/data/components"
	"ruleGoKratos/internal/data/dao"
	"time"

	_ "github.com/rulego/rulego-components-ci/ci/action"
	_ "github.com/rulego/rulego-components/endpoint/redis"
	_ "github.com/rulego/rulego-components/external/redis"
	_ "github.com/rulego/rulego/components/action"
	_ "github.com/rulego/rulego/components/common"
	// v0.35.2：lua 组件迁至 filter/lua、transform/lua 子包（根路径不再含包）
	_ "github.com/rulego/rulego-components/filter/lua"
	_ "github.com/rulego/rulego-components/transform/lua"

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
var ProviderSet = wire.NewSet(
	NewDB,
	NewData,
	NewRuleConfig,
	NewRuleEngine,
	NewComponentUseRuleRepo,
	NewComponentRegulationRepo,
	NewMdWorkflowRepo,
	NewRunLogRepo,
	NewRuleChainRepo,
	NewScheduledTaskRepo,
	NewTaskBoardRepo,
	NewServiceManagementRepo,
)

var DBClient *gorm.DB

// Data .
type Data struct {
	// TODO wrapped database client
	db *gorm.DB
}

// DB 返回 GORM 客户端（供 Playground 等组件注入）。
func (d *Data) DB() *gorm.DB {
	return d.db
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

func NewRuleConfig() *types.Config {
	componentRegistry := engine.NewCustomComponentRegistry(engine.Registry, new(engine.RuleComponentRegistry))
	config := rulego.NewConfig(types.WithDefaultPool(),
		// types.WithLogger(logger.Logger),
		types.WithComponentsRegistry(componentRegistry),
		types.WithNodePool(node_pool.DefaultNodePool))
	return &config
}

// 初始化规则引擎
func NewRuleEngine(c *conf.Data, ruleConfig *types.Config, agentUc *biz.AgentUsecase, taskBoardUc *biz.TaskBoardUsecase, serviceMgmtUc *biz.ServiceManagementUsecase) (*rulego.RuleGo, error) {
	agentUc.SetManagedLLMResolver(NewManagedLLMResolver())
	agentUc.SetManagedAgentLoader(NewManagedAgentHarnessLoader())
	agentUc.SetMcpConfigAdmin(NewMcpConfigAdmin())
	agentUc.SetMcpToolProvider(NewDatabaseMcpToolProvider())
	WireRuleGoAgent(agentUc)
	rulegodatacomp.SetTaskUsecase(taskBoardUc)
	rulegodatacomp.SetServiceUsecase(serviceMgmtUc)
	// 获取所有的规则链信息
	var ruleChainList []RuleChain
	if err := DBClient.Table("rule_chain").Where("disabled = ?", false).Find(&ruleChainList).Error; err != nil {
		return nil, err
	}
	pool := rulego.NewRuleGo()
	// 延迟注入规则引擎实例到 TaskBoardUsecase
	taskBoardUc.SetRuleEngine(pool, ruleConfig)
	if memStore := agentUc.GetMemoryStore(); memStore != nil {
		taskBoardUc.SetMemoryStore(memStore)
	}
	// 加载所有规则链
	for _, item := range ruleChainList {
		// 更新规则配置状态为禁用
		ruleChain, err := ruleChainDBToRuleChain(&item)
		if err != nil {
			return nil, err
		}
		ruleChainJson, err := json.Marshal(ruleChain)
		if err != nil {
			return nil, err
		}
		// 如果规则链已经存在，重新加载规则配置
		if ruleEngine, ok := pool.Get(item.RuleChainID); ok {
			fmt.Println("重新加载规则配置", item.RuleChainID)
			err = ruleEngine.ReloadSelf(ruleChainJson)
		} else {
			fmt.Println("加载规则链", item.RuleChainID)
			_, err = pool.New(item.RuleChainID, ruleChainJson, rulego.WithConfig(*ruleConfig))
			if err != nil {
				return nil, err
			}
			DBClient.Table("rule_chain").Where("id = ?", item.ID).Updates(map[string]interface{}{
				"disabled": false,
			})
		}
		if err != nil {
			DBClient.Table("rule_chain").Where("id = ?", item.ID).Updates(map[string]interface{}{
				"disabled": true,
			})
		}
	}
	return pool, nil
}

// RuleChainDBToRuleChain 将数据库中的规则链转换为RuleChain
func ruleChainDBToRuleChain(ruleChainDB *RuleChain) (*types.RuleChain, error) {
	var ruleChain types.RuleChain
	ruleChainInfo := types.RuleChainBaseInfo{
		ID:        ruleChainDB.RuleChainID,
		Name:      ruleChainDB.Name,
		DebugMode: ruleChainDB.DebugMode,
		Root:      ruleChainDB.Root,
		Disabled:  ruleChainDB.Disabled,
	}
	additionalInfo := map[string]interface{}{}
	json.Unmarshal([]byte(ruleChainDB.AdditionalInfo), &additionalInfo)
	ruleChainInfo.AdditionalInfo = additionalInfo
	configuration := map[string]interface{}{}
	json.Unmarshal([]byte(ruleChainDB.Configuration), &configuration)
	ruleChainInfo.Configuration = configuration
	ruleChainMetadata := types.RuleMetadata{}
	json.Unmarshal([]byte(ruleChainDB.Metadata), &ruleChainMetadata)
	ruleChain.RuleChain = ruleChainInfo
	ruleChain.Metadata = ruleChainMetadata
	return &ruleChain, nil
}

type RuleChain struct {
	ID             int64      `json:"id"`
	UserName       string     `json:"userName"`
	Root           bool       `json:"root"`
	Disabled       bool       `json:"disabled"`
	DebugMode      bool       `json:"debugMode"`
	Name           string     `json:"name"`
	RuleChainID    string     `json:"ruleChainId"`
	RuleVersion    int        `json:"ruleVersion"`
	Configuration  string     `json:"configuration"`
	Metadata       string     `json:"metadata"`
	AdditionalInfo string     `json:"additionalInfo"`
	CreatedAt      *time.Time `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt"`
}
