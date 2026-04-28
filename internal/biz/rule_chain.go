package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/conf"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/builtin/processor"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/endpoint"
	"github.com/rulego/rulego/engine"
	"github.com/rulego/rulego/node_pool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// RuleChainSkillAgentRunner 是规则链 Skill 同步生成所需的最小 Agent 依赖。
type RuleChainSkillAgentRunner interface {
	ExecuteHarnessSync(ctx context.Context, req HarnessRequest) (string, error)
}

type RuleChainUsecase struct {
	ruleChainRepo RuleChainRepo
	log           *log.Helper
	runLogRepo    RunLogRepo
	ruleEngine    *rulego.RuleGo
	ruleConfig    *types.Config
	skillAgent    RuleChainSkillAgentRunner
	skillRoot     string
}

const ruleChainSkillCreatorName = "skill-creator-0.1.0"

// userIDContextKey 和 projectPathContextKey 与 auth middleware 保持一致
const (
	userIDContextKey      = "x-user-id"
	projectPathContextKey = "x-project-path"
)

func extractUserIDFromContext(ctx context.Context) string {
	if v := ctx.Value(userIDContextKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func NewRuleChainUsecase(ruleChainRepo RuleChainRepo, runLogRepo RunLogRepo, logger log.Logger, ruleEngine *rulego.RuleGo, ruleConfig *types.Config, skillAgent RuleChainSkillAgentRunner, config *conf.Bootstrap) *RuleChainUsecase {
	return &RuleChainUsecase{
		ruleChainRepo: ruleChainRepo,
		runLogRepo:    runLogRepo,
		log:           log.NewHelper(logger),
		ruleEngine:    ruleEngine,
		ruleConfig:    ruleConfig,
		skillAgent:    skillAgent,
		skillRoot:     resolveRuleChainSkillRoot(config),
	}
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

func (s *RuleChainUsecase) GetRuleChainSkillStatus(ctx context.Context, in *v1.GetRuleChainSkillStatusReq) (*v1.GetRuleChainSkillStatusReply, error) {
	_, ruleChain, err := s.loadRootRuleChainForSkill(ctx, in.GetId())
	if err != nil {
		return nil, err
	}

	currentSignature := s.computeRuleChainSkillSignature(ruleChain)
	meta := ParseRuleChainSkillMeta(asJSONMap(ruleChain.RuleChain.Configuration))
	currentStatus, err := ResolveRuleChainSkillStatus(s.effectiveRuleChainSkillRoot(), meta, currentSignature)
	if err != nil {
		return nil, err
	}

	return &v1.GetRuleChainSkillStatusReply{
		Status:                    string(currentStatus),
		DirName:                   normalizeRuleChainSkillDirName(meta.DirName),
		EntryFile:                 normalizeRuleChainSkillEntryFile(meta.EntryFile),
		Signature:                 currentSignature,
		GeneratedAt:               strings.TrimSpace(meta.GeneratedAt),
		GeneratedByManagedAgentId: meta.GeneratedByManagedAgentID,
		LastError:                 strings.TrimSpace(meta.LastError),
	}, nil
}

func (s *RuleChainUsecase) GenerateRuleChainSkill(ctx context.Context, in *v1.GenerateRuleChainSkillReq) (*v1.GenerateRuleChainSkillReply, error) {
	if in.GetManagedAgentId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "managedAgentId 必须大于 0")
	}
	if s.skillAgent == nil {
		return nil, status.Error(codes.FailedPrecondition, "规则链 Skill Agent 未配置")
	}

	ruleChainDB, ruleChain, err := s.loadRootRuleChainForSkill(ctx, in.GetId())
	if err != nil {
		return nil, err
	}

	currentSignature := s.computeRuleChainSkillSignature(ruleChain)
	existingMeta := ParseRuleChainSkillMeta(asJSONMap(ruleChain.RuleChain.Configuration))
	dirName, err := s.chooseRuleChainSkillDirName(ctx, ruleChainDB, ruleChain, existingMeta)
	if err != nil {
		return nil, err
	}
	promptInput := BuildRuleChainSkillPromptInput(ruleChain, dirName)
	promptInput.SkillRoot = s.effectiveRuleChainSkillRoot()
	prompt := BuildRuleChainSkillGenerationPrompt(promptInput)
	entryFile := normalizeRuleChainSkillEntryFile(existingMeta.EntryFile)

	successMeta := RuleChainSkillMeta{
		DirName:                   dirName,
		EntryFile:                 entryFile,
		Signature:                 currentSignature,
		LastGenerated:             currentSignature,
		GeneratedByManagedAgentID: in.GetManagedAgentId(),
	}

	harnessOutput, err := s.skillAgent.ExecuteHarnessSync(ctx, HarnessRequest{
		Input: prompt,
		ToolOptions: &HarnessToolOptions{
			EnableSkillTool: true,
			// 规则链技能生成固定依赖 skill-creator，避免被托管 Agent 白名单误拦截。
			SkillAllowlist: []string{ruleChainSkillCreatorName},
		},
		ManagedAgentID: in.GetManagedAgentId(),
		UserID:        extractUserIDFromContext(ctx),
		ProjectPath:   dirName,
	})
	if err != nil {
		_ = s.persistRuleChainSkillFailure(ctx, ruleChainDB, ruleChain, existingMeta, dirName, entryFile, currentSignature, err.Error())
		return nil, err
	}

	content, err := ReadRuleChainSkillFile(s.effectiveRuleChainSkillRoot(), dirName, entryFile)
	if err != nil {
		// 兼容 skill-creator 仅返回内容未实际落盘的场景：从最终输出提取 markdown 并回写目标文件。
		fallbackContent := ExtractRuleChainSkillMarkdownFromHarnessOutput(harnessOutput)
		if strings.TrimSpace(fallbackContent) != "" {
			fallbackContent = NormalizeGeneratedRuleChainSkillContent(fallbackContent, promptInput)
			if writeErr := WriteRuleChainSkillFile(s.effectiveRuleChainSkillRoot(), dirName, entryFile, fallbackContent); writeErr == nil {
				content, err = ReadRuleChainSkillFile(s.effectiveRuleChainSkillRoot(), dirName, entryFile)
			}
		}
	}
	if err != nil {
		_ = s.persistRuleChainSkillFailure(ctx, ruleChainDB, ruleChain, existingMeta, dirName, entryFile, currentSignature, fmt.Sprintf("生成后未找到有效的 %s: %v", entryFile, err))
		return nil, status.Errorf(codes.FailedPrecondition, "生成后未找到有效的 %s: %v", entryFile, err)
	}
	normalizedContent := NormalizeGeneratedRuleChainSkillContent(content, promptInput)
	if normalizedContent != strings.TrimSpace(content) {
		if writeErr := WriteRuleChainSkillFile(s.effectiveRuleChainSkillRoot(), dirName, entryFile, normalizedContent); writeErr == nil {
			content = normalizedContent
		}
	}
	if err := ValidateGeneratedRuleChainSkillContent(content, promptInput); err != nil {
		_ = s.persistRuleChainSkillFailure(ctx, ruleChainDB, ruleChain, existingMeta, dirName, entryFile, currentSignature, err.Error())
		return nil, status.Errorf(codes.FailedPrecondition, "%s", err.Error())
	}

	readyMeta := successMeta
	readyMeta.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	readyMeta.LastError = ""
	currentStatus, err := ResolveRuleChainSkillStatus(s.effectiveRuleChainSkillRoot(), readyMeta, currentSignature)
	if err != nil {
		return nil, err
	}
	readyMeta.Status = string(currentStatus)
	if readyMeta.Status != string(RuleChainSkillStatusReady) {
		return nil, status.Errorf(codes.FailedPrecondition, "规则链 Skill 未达到 ready 状态: %s", readyMeta.Status)
	}

	if err := s.persistRuleChainSkillMeta(ctx, ruleChainDB, ruleChain, readyMeta); err != nil {
		return nil, err
	}

	return &v1.GenerateRuleChainSkillReply{
		Status:  readyMeta.Status,
		DirName: readyMeta.DirName,
	}, nil
}

func resolveRuleChainSkillRoot(config *conf.Bootstrap) string {
	if envRoot := strings.TrimSpace(os.Getenv("RULE_CHAIN_SKILL_DIR")); envRoot != "" {
		return envRoot
	}
	if config != nil && config.Agent != nil && config.Agent.Skill != nil {
		if dir := strings.TrimSpace(config.Agent.Skill.GetDir()); dir != "" {
			return dir
		}
		if dirs := ParseCommaSeparated(config.Agent.Skill.GetDirs()); len(dirs) > 0 {
			return dirs[0]
		}
	}
	return "/workflow/skills"
}

func (s *RuleChainUsecase) effectiveRuleChainSkillRoot() string {
	return effectiveRuleChainSkillRootPath(s.skillRoot)
}

func (s *RuleChainUsecase) loadRootRuleChainForSkill(ctx context.Context, id string) (*entity.RuleChain, *types.RuleChain, error) {
	ruleChainDB, err := s.ruleChainRepo.FindOneRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": id,
	})
	if err != nil {
		return nil, nil, err
	}
	ruleChain, err := s.RuleChainDBToRuleChain(ruleChainDB)
	if err != nil {
		return nil, nil, err
	}
	if !ruleChain.RuleChain.Root {
		return nil, nil, status.Error(codes.InvalidArgument, "仅主规则链支持 Skill 状态查询与生成")
	}
	return ruleChainDB, ruleChain, nil
}

func (s *RuleChainUsecase) computeRuleChainSkillSignature(ruleChain *types.RuleChain) string {
	input := BuildRuleChainSkillPromptInput(ruleChain, "")
	return BuildRuleChainSkillSignature(
		input.Description,
		input.RequestMetadataParams,
		input.RequestBodyParams,
		input.ResponseBodyParams,
	)
}

func (s *RuleChainUsecase) chooseRuleChainSkillDirName(ctx context.Context, ruleChainDB *entity.RuleChain, ruleChain *types.RuleChain, meta RuleChainSkillMeta) (string, error) {
	baseDir := BuildRuleChainSkillBaseDirName(ruleChain)
	explicitDir := normalizeRuleChainSkillDirName(meta.DirName)
	candidateDir := explicitDir
	if candidateDir == "" {
		candidateDir = BuildRuleChainSkillUniqueDirName(ruleChain)
	}
	all, err := s.ruleChainRepo.FindAllRuleChain(ctx, map[string]interface{}{})
	if err != nil {
		return "", err
	}
	for _, item := range all {
		if item.RuleChainID == ruleChainDB.RuleChainID {
			continue
		}
		other, convErr := s.RuleChainDBToRuleChain(&item)
		if convErr != nil {
			return "", convErr
		}
		otherMeta := ParseRuleChainSkillMeta(asJSONMap(other.RuleChain.Configuration))
		if normalizeRuleChainSkillDirName(otherMeta.DirName) == candidateDir {
			return BuildRuleChainSkillConflictDirName(baseDir, ruleChain.RuleChain.ID), nil
		}
	}
	exists, err := RuleChainSkillDirExists(s.effectiveRuleChainSkillRoot(), candidateDir)
	if err != nil {
		return "", err
	}
	if exists && explicitDir == "" {
		return BuildRuleChainSkillConflictDirName(baseDir, ruleChain.RuleChain.ID), nil
	}
	return candidateDir, nil
}

func (s *RuleChainUsecase) persistRuleChainSkillMeta(ctx context.Context, ruleChainDB *entity.RuleChain, ruleChain *types.RuleChain, meta RuleChainSkillMeta) error {
	configuration := mergeRuleChainConfigurationPatch(ruleChain.RuleChain.Configuration, BuildRuleChainSkillMetaPatch(meta))
	configurationBytes, err := json.Marshal(configuration)
	if err != nil {
		return err
	}
	if err := s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": ruleChainDB.RuleChainID,
	}, map[string]interface{}{
		"configuration": string(configurationBytes),
	}); err != nil {
		return err
	}
	ruleChain.RuleChain.Configuration = configuration
	return nil
}

func (s *RuleChainUsecase) persistRuleChainSkillFailure(ctx context.Context, ruleChainDB *entity.RuleChain, ruleChain *types.RuleChain, existingMeta RuleChainSkillMeta, dirName string, entryFile string, currentSignature string, lastError string) error {
	failureMeta := RuleChainSkillMeta{
		DirName:                   dirName,
		EntryFile:                 entryFile,
		Signature:                 strings.TrimSpace(existingMeta.Signature),
		LastGenerated:             strings.TrimSpace(existingMeta.LastGenerated),
		Status:                    strings.TrimSpace(existingMeta.Status),
		GeneratedAt:               strings.TrimSpace(existingMeta.GeneratedAt),
		GeneratedByManagedAgentID: existingMeta.GeneratedByManagedAgentID,
		LastError:                 strings.TrimSpace(lastError),
	}
	failureStatus, err := ResolveRuleChainSkillStatus(s.effectiveRuleChainSkillRoot(), failureMeta, currentSignature)
	if err != nil {
		return err
	}
	failureMeta.Status = string(failureStatus)
	return s.persistRuleChainSkillMeta(ctx, ruleChainDB, ruleChain, failureMeta)
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
	data, err := json.Marshal(structToMap(in.Data))
	if err != nil {
		return nil, err
	}
	metadata, err := buildRuleMsgMetadata(in.Metadata)
	if err != nil {
		return nil, err
	}
	msg := types.RuleMsg{
		Id:       in.MsgId,
		Data:     types.NewSharedData(string(data)),
		Type:     in.MsgType,
		DataType: types.JSON,
		Metadata: metadata,
	}
	engine.OnMsg(msg, s.addWithOnRuleChainCompleted(ctx))
	return res, nil
}

func (s *RuleChainUsecase) GetScheduledTaskRuleChain(ctx context.Context, id string) (*entity.RuleChain, error) {
	if s == nil || s.ruleChainRepo == nil {
		return nil, errors.New("规则链仓储未配置")
	}
	return s.ruleChainRepo.FindOneRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": id,
	})
}

func (s *RuleChainUsecase) IsRuleChainLoaded(id string) bool {
	if s == nil || s.ruleEngine == nil {
		return false
	}
	_, ok := s.ruleEngine.Get(id)
	return ok
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
	data, err := json.Marshal(structToMap(in.Data))
	if err != nil {
		return nil, err
	}
	metadata, err := buildRuleMsgMetadata(in.Metadata)
	if err != nil {
		return nil, err
	}
	msg := types.RuleMsg{
		Id:       in.MsgId,
		Data:     types.NewSharedData(string(data)),
		Type:     in.MsgType,
		DataType: types.JSON,
		Metadata: metadata,
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
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultMap); err == nil && resultMap != nil {
		structPb, err := structpb.NewStruct(resultMap)
		if err != nil {
			return nil, err
		}
		return &v1.ExecuteRuleChainSyncReply{
			Data: structPb,
		}, nil
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
		if err == nil {
			// 同步刷新内存引擎：若该规则链已部署，保存后立即 ReloadSelf，
			// 防止「只改 DB、引擎仍持旧定义」导致执行日志无法生成。
			if reloadErr := s.reloadEngineIfDeployed(ctx, in.Id, &ruleChain); reloadErr != nil {
				return nil, reloadErr
			}
		}
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
	if m, ok := v.(types.Configuration); ok {
		return cloneJSONMap(map[string]interface{}(m))
	}
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
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

// BuildRuleChainSyncExecutePayload 统一构建同步执行入口的 metadata/data 载荷。
func BuildRuleChainSyncExecutePayload(metadata, data map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"metadata": cloneJSONMap(metadata),
		"data":     cloneJSONMap(data),
	}, nil
}

func structToMap(st *structpb.Struct) map[string]interface{} {
	if st == nil {
		return map[string]interface{}{}
	}
	return st.AsMap()
}

func buildRuleMsgMetadata(st *structpb.Struct) (*types.Metadata, error) {
	values := structToMap(st)
	if len(values) == 0 {
		return nil, nil
	}
	metadata := make(map[string]string, len(values))
	for key, value := range values {
		metadata[key] = stringifyRuleMsgMetadataValue(value)
	}
	return types.BuildMetadata(metadata), nil
}

func stringifyRuleMsgMetadataValue(v interface{}) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int8:
		return strconv.FormatInt(int64(typed), 10)
	case int16:
		return strconv.FormatInt(int64(typed), 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	default:
		b, err := json.Marshal(v)
		if err == nil {
			return string(b)
		}
		return fmt.Sprint(v)
	}
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

// reloadEngineIfDeployed 若规则链已在内存引擎中，则用新定义热重载，保持 DB 与引擎同步。
// 仅在链已在引擎池中（即已部署）时才执行，避免把下线的链意外上线。
// 热重载失败时：同步将 DB 中该链标记为 disabled=true（与 deployRuleChain 失败逻辑一致），
// 并将错误透传给前端，使用户明确感知。
func (s *RuleChainUsecase) reloadEngineIfDeployed(ctx context.Context, chainId string, ruleChain *types.RuleChain) error {
	existingEngine, ok := s.ruleEngine.Get(chainId)
	if !ok {
		return nil
	}
	clone := *ruleChain
	clone.RuleChain.Disabled = false
	def, err := json.Marshal(&clone)
	if err != nil {
		// marshal 出错一般是程序 bug，同样下线以保证一致性
		_ = s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
			"rule_chain_id": chainId,
		}, map[string]interface{}{
			"disabled": true,
		})
		return fmt.Errorf("规则链热重载失败（marshal）: %w", err)
	}
	if err = existingEngine.ReloadSelf(def); err != nil {
		// 重载失败：引擎仍持旧定义，将 DB 标记为下线，保持 DB 与引擎状态一致
		_ = s.ruleChainRepo.UpdateRuleChain(ctx, map[string]interface{}{
			"rule_chain_id": chainId,
		}, map[string]interface{}{
			"disabled": true,
		})
		return fmt.Errorf("规则链热重载失败: %w", err)
	}
	return nil
}

func (s *RuleChainUsecase) DeleteRuleChain(ctx context.Context, in *v1.DeleteRuleChainReq) (*v1.DeleteRuleChainReply, error) {
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
	meta := ParseRuleChainSkillMeta(asJSONMap(ruleChain.RuleChain.Configuration))
	pendingSkillDir, err := PrepareDeleteRuleChainSkillDir(s.effectiveRuleChainSkillRoot(), meta)
	if err != nil {
		return nil, err
	}
	// 先从数据库删除；失败时恢复暂存的 skill 目录，避免半成功状态。
	err = s.ruleChainRepo.DeleteRuleChain(ctx, map[string]interface{}{
		"rule_chain_id": in.Id,
	})
	if err != nil {
		if restoreErr := pendingSkillDir.Restore(); restoreErr != nil {
			return nil, fmt.Errorf("删除规则链失败，且恢复 skill 目录失败: deleteErr=%v restoreErr=%v", err, restoreErr)
		}
		return nil, err
	}
	// 数据库删除成功后，再从引擎移除并清理暂存目录。
	s.ruleEngine.Del(in.Id)
	if err := pendingSkillDir.Finalize(); err != nil {
		s.log.Warnf("best-effort cleanup recycled skill dir failed for ruleChain=%s: %v", in.Id, err)
	}
	return &v1.DeleteRuleChainReply{}, nil
}
