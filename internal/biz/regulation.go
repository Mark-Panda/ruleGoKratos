package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/builtin/processor"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/endpoint"
	"github.com/rulego/rulego/engine"
	"github.com/rulego/rulego/node_pool"
)

type RegulationRepo interface {
	CreateRegulation(ctx context.Context, regulation *entity.Regulation) error
	UpdateRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error
	DeleteRegulation(ctx context.Context, where map[string]interface{}) error
	FindOneRegulation(ctx context.Context, where map[string]interface{}) (*entity.Regulation, error)
	FindListRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.Regulation, int64, error)
	FindAllRegulation(ctx context.Context, where map[string]interface{}) ([]entity.Regulation, error)
}

// RegulationUsecase is a Regulation usecase.
type RegulationUsecase struct {
	repo       RegulationRepo
	runLogRepo RunLogRepo
	log        *log.Helper
	ruleEngine *rulego.RuleGo
	ruleConfig *types.Config
}

// NewRegulationUsecase new a Regulation usecase.
func NewRegulationUsecase(repo RegulationRepo, runLogRepo RunLogRepo, logger log.Logger, ruleEngine *rulego.RuleGo, ruleConfig *types.Config) *RegulationUsecase {
	return &RegulationUsecase{repo: repo, runLogRepo: runLogRepo, log: log.NewHelper(logger), ruleEngine: ruleEngine, ruleConfig: ruleConfig}
}

// 辅助函数：将任意对象转换为 *structpb.Struct
func toStructPb(v interface{}) (*structpb.Struct, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}

// 辅助函数：将任意对象转换为 *structpb.ListValue
func toListValuePb(v interface{}) (*structpb.ListValue, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var l []interface{}
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	return structpb.NewList(l)
}

func (s *RegulationUsecase) GetComponents(ctx context.Context) (*v1.GetComponentsReply, error) {
	componentRegistry := engine.NewCustomComponentRegistry(engine.Registry, new(engine.RuleComponentRegistry))
	ruleConfig := rulego.NewConfig(types.WithDefaultPool(),
		// types.WithLogger(logger.Logger),
		types.WithComponentsRegistry(componentRegistry),
		types.WithNodePool(node_pool.DefaultNodePool))

	nodePool, _ := node_pool.DefaultNodePool.GetAllDef()

	// endpoint组件
	endpointsList, err := toListValuePb(endpoint.Registry.GetComponentForms().Values())
	if err != nil {
		return nil, err
	}

	// 节点组件
	nodesList, err := toListValuePb(ruleConfig.ComponentsRegistry.GetComponentForms().Values())
	if err != nil {
		return nil, err
	}

	// 组件配置内置选项
	builtinsStruct, err := toStructPb(map[string]interface{}{
		// functions节点组件
		"functions": map[string]interface{}{
			//函数名选项
			"functionName": action.Functions.Names(),
		},
		//endpoints内置路由选项
		"endpoints": map[string]interface{}{
			//in 处理器列表
			"inProcessors": processor.InBuiltins.Names(),
			//in 处理器列表
			"outProcessors": processor.OutBuiltins.Names(),
		},
		//共享节点池
		"nodePool": nodePool,
	})
	if err != nil {
		return nil, err
	}

	return &v1.GetComponentsReply{
		Endpoints: endpointsList,
		Nodes:     nodesList,
		Builtins:  builtinsStruct,
	}, nil
}

func (s *RegulationUsecase) GetRegulationsList(ctx context.Context, in *v1.GetRegulationsListReq) (*v1.GetRegulationsListReply, error) {
	res := &v1.GetRegulationsListReply{}
	var page = 1
	var size = 20
	currentStr := in.Page
	if i, err := strconv.Atoi(currentStr); err == nil {
		page = i
	}
	pageSizeStr := in.Size
	if i, err := strconv.Atoi(pageSizeStr); err == nil {
		size = i
	}
	var root *bool
	var disabled *bool
	if i, err := strconv.ParseBool(in.Root); err == nil {
		root = &i
	}
	if i, err := strconv.ParseBool(in.Disabled); err == nil {
		disabled = &i
	}
	query := map[string]interface{}{}
	if root != nil {
		query["root"] = root
	}
	if disabled != nil {
		query["disabled"] = disabled
	}
	if in.Keywords != "" {
		query["name"] = in.Keywords
	}
	regulations, total, err := s.repo.FindListRegulation(ctx, query, page, size)
	if err != nil {
		return nil, err
	}
	res.Page = int32(page)
	res.Size = int32(size)
	res.Total = int32(total)
	for _, regulation := range regulations {
		var ruleChainLi types.RuleChain
		err := json.Unmarshal([]byte(regulation.RuleConfig), &ruleChainLi)
		if err != nil {
			return nil, err
		}
		structPb, err := toStructPb(ruleChainLi)
		if err != nil {
			return nil, err
		}
		res.Items = append(res.Items, structPb)
	}
	return res, nil
}

func (s *RegulationUsecase) ExecuteRuleChain(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainReply, error) {
	res := &v1.ExecuteRuleChainReply{}
	engine, find := s.ruleEngine.Get(in.Id)
	if !find {
		return nil, errors.New("rule chain not found")
	}
	data, err := json.Marshal(in.Data)
	if err != nil {
		return nil, err
	}
	msg := types.RuleMsg{
		Id:       in.MsgId,
		Data:     types.NewSharedData(string(data)),
		Type:     in.MsgType,
		DataType: types.JSON,
	}
	engine.OnMsg(msg, s.addWithOnRuleChainCompleted(ctx))
	return res, nil
}

func (s *RegulationUsecase) addWithOnRuleChainCompleted(ctx context.Context) types.RuleContextOption {
	return types.WithOnRuleChainCompleted(func(ctn types.RuleContext, snapshot types.RuleChainRunSnapshot) {
		nodelogs, _ := json.Marshal(snapshot.Logs)
		additionalInfo, _ := json.Marshal(snapshot.AdditionalInfo)
		ruleChainInfo, _ := json.Marshal(snapshot.RuleChain)
		metadata, _ := json.Marshal(snapshot.Metadata)
		t := time.Now()
		s.runLogRepo.CreateRunLog(ctx, &entity.RunLog{
			RunID:          snapshot.Id,
			ChainID:        snapshot.RuleChain.RuleChain.ID,
			ChainName:      snapshot.RuleChain.RuleChain.Name,
			NodeLog:        string(nodelogs),
			AdditionalInfo: string(additionalInfo),
			RuleChainInfo:  string(ruleChainInfo),
			Metadata:       string(metadata),
			StartTs:        snapshot.StartTs,
			EndTs:          snapshot.EndTs,
			CreatedAt:      &t,
			UpdatedAt:      &t,
		})
	})
}

func (s *RegulationUsecase) ExecuteRuleChainSync(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainSyncReply, error) {
	engine, find := s.ruleEngine.Get(in.Id)
	if !find {
		return nil, errors.New("rule chain not found")
	}
	var err error
	data, err := json.Marshal(in.Data)
	if err != nil {
		return nil, err
	}
	msg := types.RuleMsg{
		Id:       in.MsgId,
		Data:     types.NewSharedData(string(data)),
		Type:     in.MsgType,
		DataType: types.JSON,
	}
	var result string
	engine.OnMsgAndWait(msg, types.WithOnEnd(func(ctn types.RuleContext, msg types.RuleMsg, err1 error, relationType string) {
		if err1 != nil {
			err = err1
			return
		}
		result = msg.GetData()
	}))
	if err != nil {
		return nil, err
	}
	structPb, err := toStructPb(result)
	if err != nil {
		return nil, err
	}
	return &v1.ExecuteRuleChainSyncReply{
		Data: structPb,
	}, nil
}

func (s *RegulationUsecase) DeployRuleChain(ctx context.Context, in *v1.DeployRuleChainReq) (*v1.DeployRuleChainReply, error) {
	res := &v1.DeployRuleChainReply{}
	chainId := in.Id
	if in.Type == "start" {
		// 部署
		err := s.deployRuleChain(ctx, chainId)
		if err != nil {
			return nil, err
		}
	} else if in.Type == "stop" {
		// 下线
		err := s.stopRuleChain(ctx, chainId)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("invalid type")
	}
	return res, nil
}

// 部署上线
func (s *RegulationUsecase) deployRuleChain(ctx context.Context, chainId string) error {
	var def []byte
	var err error
	regulation, err := s.repo.FindOneRegulation(ctx, map[string]interface{}{
		"rule_chain_id": chainId,
	})
	if err != nil {
		return err
	}

	var ruleChain types.RuleChain
	err = json.Unmarshal([]byte(regulation.RuleConfig), &ruleChain)
	if err != nil {
		return err
	}
	if def, err = json.Marshal(ruleChain); err != nil {
		return err
	} else {
		ruleEngine, ok := s.ruleEngine.Get(chainId)
		if ok {
			err = ruleEngine.ReloadSelf(def)
		} else {
			_, err = s.ruleEngine.New(chainId, def, rulego.WithConfig(*s.ruleConfig))
		}
		if err != nil {
			// 下线
			ruleChain.RuleChain.Disabled = true
			ruleChainJson, err := json.Marshal(ruleChain)
			if err != nil {
				return err
			}
			s.repo.UpdateRegulation(ctx, map[string]interface{}{
				"rule_chain_id": chainId,
			}, map[string]interface{}{
				"disabled":    true,
				"rule_config": string(ruleChainJson),
			})
			return err
		}
		// 上线
		ruleChain.RuleChain.Disabled = false
		ruleChainJson, err := json.Marshal(ruleChain)
		if err != nil {
			return err
		}
		s.repo.UpdateRegulation(ctx, map[string]interface{}{
			"rule_chain_id": chainId,
		}, map[string]interface{}{
			"disabled":    false,
			"rule_config": string(ruleChainJson),
		})
	}
	return nil
}

// 下线
func (s *RegulationUsecase) stopRuleChain(ctx context.Context, chainId string) error {
	var err error
	regulation, err := s.repo.FindOneRegulation(ctx, map[string]interface{}{
		"rule_chain_id": chainId,
	})
	if err != nil {
		return err
	}
	var ruleChain types.RuleChain
	err = json.Unmarshal([]byte(regulation.RuleConfig), &ruleChain)
	if err != nil {
		return err
	}
	s.ruleEngine.Del(chainId)
	// 下线
	ruleChain.RuleChain.Disabled = true
	ruleChainJson, err := json.Marshal(ruleChain)
	if err != nil {
		return err
	}
	err = s.repo.UpdateRegulation(ctx, map[string]interface{}{
		"rule_chain_id": chainId,
	}, map[string]interface{}{
		"disabled":    true,
		"rule_config": string(ruleChainJson),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *RegulationUsecase) UpsertRuleChain(ctx context.Context, in *v1.UpsertRuleChainReq) (*v1.UpsertRuleChainReply, error) {
	var ruleChain types.RuleChain
	if in.RuleChain != nil {
		b, err := in.RuleChain.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &ruleChain.RuleChain); err != nil {
			return nil, err
		}
	}
	if in.Metadata != nil {
		b, err := in.Metadata.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &ruleChain.Metadata); err != nil {
			return nil, err
		}
	}
	// Ensure ID matches
	ruleChain.RuleChain.ID = in.Id

	ruleChainJson, err := json.Marshal(ruleChain)
	if err != nil {
		return nil, err
	}

	regulation, err := s.repo.FindOneRegulation(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	})

	if err == nil && regulation != nil {
		// Update
		err = s.repo.UpdateRegulation(ctx, map[string]interface{}{
			"rule_chain_id": in.Id,
		}, map[string]interface{}{
			"rule_config": string(ruleChainJson),
			"name":        ruleChain.RuleChain.Name,
			"root":        ruleChain.RuleChain.Root,
			"disabled":    ruleChain.RuleChain.Disabled,
		})
	} else {
		// Create
		t := time.Now()
		reg := &entity.Regulation{
			RuleChainID: in.Id,
			RuleConfig:  string(ruleChainJson),
			Name:        ruleChain.RuleChain.Name,
			Root:        ruleChain.RuleChain.Root,
			Disabled:    ruleChain.RuleChain.Disabled,
			CreatedAt:   &t,
			UpdatedAt:   &t,
		}
		err = s.repo.CreateRegulation(ctx, reg)
	}

	if err != nil {
		return nil, err
	}

	return &v1.UpsertRuleChainReply{}, nil
}

func (s *RegulationUsecase) UpdateRuleChainBaseInfo(ctx context.Context, in *v1.UpdateRuleChainBaseInfoReq) (*v1.UpdateRuleChainBaseInfoReply, error) {
	regulation, err := s.repo.FindOneRegulation(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	})
	if err != nil {
		return nil, err
	}

	var ruleChain types.RuleChain
	err = json.Unmarshal([]byte(regulation.RuleConfig), &ruleChain)
	if err != nil {
		return nil, err
	}

	// Update fields
	if in.Name != "" {
		ruleChain.RuleChain.Name = in.Name
	}
	ruleChain.RuleChain.Root = in.Root
	if in.AdditionalInfo != nil {
		additionalInfoBytes, err := in.AdditionalInfo.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var additionalInfo map[string]interface{}
		err = json.Unmarshal(additionalInfoBytes, &additionalInfo)
		if err != nil {
			return nil, err
		}
		ruleChain.RuleChain.AdditionalInfo = additionalInfo
	}

	// Marshal back to JSON
	ruleChainJson, err := json.Marshal(ruleChain)
	if err != nil {
		return nil, err
	}

	// Update DB
	err = s.repo.UpdateRegulation(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	}, map[string]interface{}{
		"rule_config": string(ruleChainJson),
	})
	if err != nil {
		return nil, err
	}

	return &v1.UpdateRuleChainBaseInfoReply{}, nil
}
