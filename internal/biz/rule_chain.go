package biz

import (
	"context"
	"encoding/json"
	"errors"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/builtin/processor"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/endpoint"
	"github.com/rulego/rulego/engine"
	"github.com/rulego/rulego/node_pool"
	"google.golang.org/protobuf/types/known/structpb"
)

type RuleChainRepo interface {
	CreateRuleChain(ctx context.Context, ruleChain *entity.RuleChain) error
	UpdateRuleChain(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error
	DeleteRuleChain(ctx context.Context, where map[string]interface{}) error
	FindOneRuleChain(ctx context.Context, where map[string]interface{}) (*entity.RuleChain, error)
	FindListRuleChain(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.RuleChain, int64, error)
	FindAllRuleChain(ctx context.Context, where map[string]interface{}) ([]entity.RuleChain, error)
}

type RuleChainUsecase struct {
	ruleChainRepo RuleChainRepo
	log           *log.Helper
	runLogRepo    RunLogRepo
	ruleEngine    *rulego.RuleGo
	ruleConfig    *types.Config
}

func NewRuleChainUsecase(ruleChainRepo RuleChainRepo, runLogRepo RunLogRepo, logger log.Logger, ruleEngine *rulego.RuleGo, ruleConfig *types.Config) *RuleChainUsecase {
	return &RuleChainUsecase{ruleChainRepo: ruleChainRepo, runLogRepo: runLogRepo, log: log.NewHelper(logger), ruleEngine: ruleEngine, ruleConfig: ruleConfig}
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

func (s *RuleChainUsecase) GetComponents(ctx context.Context) (*v1.GetComponentsReply, error) {
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

func (s *RuleChainUsecase) GetRegulationsList(ctx context.Context, in *v1.GetRegulationsListReq) (*v1.GetRegulationsListReply, error) {
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
	regulations, total, err := s.ruleChainRepo.FindListRuleChain(ctx, query, page, size)
	if err != nil {
		return nil, err
	}
	res.Page = int32(page)
	res.Size = int32(size)
	res.Total = int32(total)
	for _, regulation := range regulations {
		ruleChainLi, err := s.RuleChainDBToRuleChain(&regulation)
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

func (s *RuleChainUsecase) GetRuleChain(ctx context.Context, in *v1.GetRuleChainReq) (*v1.GetRuleChainReply, error) {
	ruleChainDB, err := s.ruleChainRepo.FindOneRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	})
	if err != nil {
		return nil, err
	}

	ruleChain, err := s.RuleChainDBToRuleChain(ruleChainDB)
	if err != nil {
		return nil, err
	}

	ruleChainPb, err := toStructPb(ruleChain.RuleChain)
	if err != nil {
		return nil, err
	}

	metadataPb, err := toStructPb(ruleChain.Metadata)
	if err != nil {
		return nil, err
	}

	return &v1.GetRuleChainReply{
		RuleChain: ruleChainPb,
		Metadata:  metadataPb,
	}, nil
}

// RuleChainDBToRuleChain 将数据库中的规则链转换为RuleChain
func (s *RuleChainUsecase) RuleChainDBToRuleChain(ruleChainDB *entity.RuleChain) (*types.RuleChain, error) {
	var ruleChain types.RuleChain
	ruleChainInfo := types.RuleChainBaseInfo{
		ID:        ruleChainDB.RuleChainID,
		Name:      ruleChainDB.Name,
		DebugMode: ruleChainDB.DebugMode,
		Root:      ruleChainDB.Root,
		Disabled:  ruleChainDB.Disabled,
	}
	if ruleChainDB.AdditionalInfo != nil {
		additionalInfo := map[string]interface{}{}
		json.Unmarshal([]byte(*ruleChainDB.AdditionalInfo), &additionalInfo)
		ruleChainInfo.AdditionalInfo = additionalInfo
	}
	if ruleChainDB.Configuration != nil {
		configuration := map[string]interface{}{}
		json.Unmarshal([]byte(*ruleChainDB.Configuration), &configuration)
		ruleChainInfo.Configuration = configuration
	}
	if ruleChainDB.Metadata != nil {
		ruleChainMetadata := types.RuleMetadata{}
		json.Unmarshal([]byte(*ruleChainDB.Metadata), &ruleChainMetadata)
		ruleChain.Metadata = ruleChainMetadata
	}
	ruleChain.RuleChain = ruleChainInfo
	return &ruleChain, nil
}

func (s *RuleChainUsecase) ExecuteRuleChain(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainReply, error) {
	res := &v1.ExecuteRuleChainReply{}
	engine, find := s.ruleEngine.Get(in.Id)
	if !find {
		return nil, errors.New("规则链未部署")
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

func (s *RuleChainUsecase) addWithOnRuleChainCompleted(ctt context.Context) types.RuleContextOption {
	return types.WithOnRuleChainCompleted(func(ctn types.RuleContext, snapshot types.RuleChainRunSnapshot) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		nodelogs, _ := json.Marshal(snapshot.Logs)
		additionalInfo, _ := json.Marshal(snapshot.AdditionalInfo)
		ruleChainInfo, _ := json.Marshal(snapshot.RuleChain)
		metadata, _ := json.Marshal(snapshot.Metadata)
		t := time.Now()
		err := s.runLogRepo.CreateRunLog(ctx, &entity.RunLog{
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
		if err != nil {
			s.log.Errorf(`create run log failed, error: %v`, err)
		}
	})
}

func (s *RuleChainUsecase) ExecuteRuleChainSync(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainSyncReply, error) {
	engine, find := s.ruleEngine.Get(in.Id)
	if !find {
		return nil, errors.New("规则链未部署")
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
	if result == "" {
		return nil, errors.New("result is empty")
	}
	structPb, err := toStructPb(result)
	if err != nil {
		return nil, err
	}
	return &v1.ExecuteRuleChainSyncReply{
		Data: structPb,
	}, nil
}

func (s *RuleChainUsecase) DeployRuleChain(ctx context.Context, in *v1.DeployRuleChainReq) (*v1.DeployRuleChainReply, error) {
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
func (s *RuleChainUsecase) deployRuleChain(ctx context.Context, chainId string) error {
	var def []byte
	var err error
	regulation, err := s.ruleChainRepo.FindOneRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": chainId,
	})
	if err != nil {
		return err
	}
	ruleChain, err := s.RuleChainDBToRuleChain(regulation)
	if err != nil {
		return err
	}
	// 必须设置为false，否则会报错 给规则引擎的时候如果这个值是true 会报the rule chain has been disabled  规则引擎按这个字段标识是否加载
	ruleChain.RuleChain.Disabled = false
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
			s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
				"rule_chain_id": chainId,
			}, map[string]interface{}{
				"disabled": true,
			})
			return err
		}
		// 上线
		s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
			"rule_chain_id": chainId,
		}, map[string]interface{}{
			"disabled": false,
		})
	}
	return nil
}

// 下线
func (s *RuleChainUsecase) stopRuleChain(ctx context.Context, chainId string) error {
	var err error
	regulation, err := s.ruleChainRepo.FindOneRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": chainId,
	})
	if err != nil {
		return err
	}
	ruleChain, err := s.RuleChainDBToRuleChain(regulation)
	if err != nil {
		return err
	}
	s.ruleEngine.Del(chainId)
	// 下线
	ruleChain.RuleChain.Disabled = true
	err = s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": chainId,
	}, map[string]interface{}{
		"disabled": true,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *RuleChainUsecase) UpsertRuleChain(ctx context.Context, in *v1.UpsertRuleChainReq) (*v1.UpsertRuleChainReply, error) {
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

	ruleChainInfo, err := s.ruleChainRepo.FindOneRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	})

	if err == nil && ruleChainInfo != nil {
		// Update
		updateData := map[string]interface{}{
			"name":       ruleChain.RuleChain.Name,
			"root":       ruleChain.RuleChain.Root,
			"disabled":   ruleChain.RuleChain.Disabled,
			"debug_mode": ruleChain.RuleChain.DebugMode,
		}
		configuration, err := json.Marshal(ruleChain.RuleChain.Configuration)
		if err != nil {
			return nil, err
		}
		updateData["configuration"] = string(configuration)
		metadata, err := json.Marshal(ruleChain.Metadata)
		if err != nil {
			return nil, err
		}
		updateData["metadata"] = string(metadata)
		additionalInfo, err := json.Marshal(ruleChain.RuleChain.AdditionalInfo)
		if err != nil {
			return nil, err
		}
		updateData["additional_info"] = string(additionalInfo)
		err = s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
			"rule_chain_id": in.Id,
		}, updateData)
	} else {
		// Create
		t := time.Now()
		reg := &entity.RuleChain{
			RuleChainID: in.Id,
			UserName:    "admin",
			Name:        ruleChain.RuleChain.Name,
			Root:        ruleChain.RuleChain.Root,
			Disabled:    ruleChain.RuleChain.Disabled,
			DebugMode:   ruleChain.RuleChain.DebugMode,
			RuleVersion: 0,
			CreatedAt:   &t,
			UpdatedAt:   &t,
		}
		configuration, err := json.Marshal(ruleChain.RuleChain.Configuration)
		if err != nil {
			return nil, err
		}
		configurationStr := string(configuration)
		reg.Configuration = &configurationStr
		metadata, err := json.Marshal(ruleChain.Metadata)
		if err != nil {
			return nil, err
		}
		metadataStr := string(metadata)
		reg.Metadata = &metadataStr
		additionalInfo, err := json.Marshal(ruleChain.RuleChain.AdditionalInfo)
		if err != nil {
			return nil, err
		}
		additionalInfoStr := string(additionalInfo)
		reg.AdditionalInfo = &additionalInfoStr
		err = s.ruleChainRepo.CreateRuleChain(ctx, reg)
	}

	if err != nil {
		return nil, err
	}
	return &v1.UpsertRuleChainReply{}, nil
}

// asJSONMap 将 interface{} 规范为 map，便于合并 configuration。
func asJSONMap(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func cloneJSONMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		cp := make(map[string]interface{}, len(m))
		for k, v := range m {
			cp[k] = v
		}
		return cp
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

// mergeFlowgramJSON 深度合并 flowgram 下的 io / editor / skill，避免 PATCH 时误删子字段。
func mergeFlowgramJSON(existing interface{}, patch interface{}) map[string]interface{} {
	out := cloneJSONMap(asJSONMap(existing))
	pm := asJSONMap(patch)
	for k, v := range pm {
		switch k {
		case "io", "editor", "skill":
			sub := cloneJSONMap(asJSONMap(out[k]))
			for sk, sv := range asJSONMap(v) {
				sub[sk] = sv
			}
			out[k] = sub
		default:
			out[k] = v
		}
	}
	return out
}

// mergeRuleChainConfigurationPatch 将请求中的 configuration 与库中已有合并。
func mergeRuleChainConfigurationPatch(existing map[string]interface{}, patch map[string]interface{}) map[string]interface{} {
	base := cloneJSONMap(existing)
	for k, v := range patch {
		if k == "flowgram" {
			base[k] = mergeFlowgramJSON(base["flowgram"], v)
			continue
		}
		if k == "devpilot" {
			// 兼容旧客户端仍提交 devpilot：合并进 flowgram 并不再保留 devpilot
			base["flowgram"] = mergeFlowgramJSON(base["flowgram"], v)
			delete(base, "devpilot")
			continue
		}
		base[k] = v
	}
	if _, ok := base["flowgram"]; ok {
		delete(base, "devpilot")
	}
	return base
}

func (s *RuleChainUsecase) UpdateRuleChainBaseInfo(ctx context.Context, in *v1.UpdateRuleChainBaseInfoReq) (*v1.UpdateRuleChainBaseInfoReply, error) {
	ruleChainInfo, err := s.ruleChainRepo.FindOneRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	})
	if err != nil {
		return nil, err
	}
	t := time.Now()
	if ruleChainInfo == nil {
		// 创建
		err := s.ruleChainRepo.CreateRuleChain(ctx, &entity.RuleChain{
			RuleChainID: in.Id,
			UserName:    "admin",
			Name:        in.Name,
			Root:        in.Root,
			Disabled:    in.Disabled,
			DebugMode:   in.DebugMode,
			RuleVersion: 0,
			CreatedAt:   &t,
			UpdatedAt:   &t,
		})
		if err != nil {
			return nil, err
		}
		return &v1.UpdateRuleChainBaseInfoReply{}, nil
	}

	ruleChain, err := s.RuleChainDBToRuleChain(ruleChainInfo)
	if err != nil {
		return nil, err
	}
	updateData := map[string]interface{}{
		"name":       in.Name,
		"root":       in.Root,
		"disabled":   in.Disabled,
		"debug_mode": in.DebugMode,
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
		updateData["additional_info"] = string(additionalInfoBytes)
	}
	if in.Configuration != nil {
		patchBytes, err := in.Configuration.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var patch map[string]interface{}
		if err := json.Unmarshal(patchBytes, &patch); err != nil {
			return nil, err
		}
		merged := mergeRuleChainConfigurationPatch(ruleChain.RuleChain.Configuration, patch)
		mergedBytes, err := json.Marshal(merged)
		if err != nil {
			return nil, err
		}
		updateData["configuration"] = string(mergedBytes)
	}
	// Update DB
	err = s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	}, updateData)
	if err != nil {
		return nil, err
	}

	return &v1.UpdateRuleChainBaseInfoReply{}, nil
}

func (s *RuleChainUsecase) DeleteRuleChain(ctx context.Context, in *v1.DeleteRuleChainReq) (*v1.DeleteRuleChainReply, error) {
	// 先从引擎中删除
	s.ruleEngine.Del(in.Id)
	// 再从数据库删除
	err := s.ruleChainRepo.DeleteRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	})
	if err != nil {
		return nil, err
	}
	return &v1.DeleteRuleChainReply{}, nil
}
