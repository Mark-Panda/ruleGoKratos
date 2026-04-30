package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"ruleGoKratos/internal/biz/entity"
)

// TaskBoardRepo 任务看板仓库接口
type TaskBoardRepo interface {
	Create(ctx context.Context, task *entity.TaskBoard) error
	GetByID(ctx context.Context, id int64) (*entity.TaskBoard, error)
	List(ctx context.Context, status, typ int32, handlerUserID string, page, pageSize int32) ([]*entity.TaskBoard, int64, error)
	Update(ctx context.Context, task *entity.TaskBoard) error
	Delete(ctx context.Context, id int64) error
	ListByParentID(ctx context.Context, parentID int64, page, pageSize int32) ([]*entity.TaskBoard, int64, error)
}

// TaskBoardUsecase 任务看板业务逻辑
type TaskBoardUsecase struct {
	repo        TaskBoardRepo
	ruleEngine  *rulego.RuleGo
	ruleConfig  *types.Config
	memoryStore MemoryStore
	runLogRepo  RunLogRepo
	log         *log.Helper
}

// NewTaskBoardUsecase 创建任务看板业务逻辑实例
func NewTaskBoardUsecase(repo TaskBoardRepo, runLogRepo RunLogRepo, logger log.Logger) *TaskBoardUsecase {
	return &TaskBoardUsecase{
		repo:       repo,
		runLogRepo: runLogRepo,
		log:        log.NewHelper(logger),
	}
}

// SetRuleEngine 设置规则引擎实例（延迟注入，在 NewRuleEngine 中调用）
func (uc *TaskBoardUsecase) SetRuleEngine(engine *rulego.RuleGo, config *types.Config) {
	uc.ruleEngine = engine
	uc.ruleConfig = config
}

// SetMemoryStore 设置记忆存储（延迟注入，从 AgentUsecase 获取）
func (uc *TaskBoardUsecase) SetMemoryStore(store MemoryStore) {
	uc.memoryStore = store
}

// CreateTask 创建任务
func (uc *TaskBoardUsecase) CreateTask(ctx context.Context, name string, priority int32, typ int32, handlerUserID, description, ruleChainID string) (*entity.TaskBoard, error) {
	// 校验关联的规则链是否存在（非空时）
	if ruleChainID != "" && uc.ruleEngine != nil {
		if _, loaded := uc.ruleEngine.Get(ruleChainID); !loaded {
			return nil, errors.New("关联的规则链不存在或未部署")
		}
	}
	task := &entity.TaskBoard{
		Name:          name,
		Priority:      priority,
		Status:        entity.TaskStatusPending,
		Type:          typ,
		HandlerUserID: handlerUserID,
		Description:   description,
		RuleChainID:   ruleChainID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if priority == 0 {
		task.Priority = 99
	}
	if err := uc.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// GetTask 获取任务详情
func (uc *TaskBoardUsecase) GetTask(ctx context.Context, id int64) (*entity.TaskBoard, error) {
	return uc.repo.GetByID(ctx, id)
}

// ListTasks 查询任务列表
func (uc *TaskBoardUsecase) ListTasks(ctx context.Context, status, typ int32, handlerUserID string, page, pageSize int32) ([]*entity.TaskBoard, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return uc.repo.List(ctx, status, typ, handlerUserID, page, pageSize)
}

// UpdateTask 更新任务
func (uc *TaskBoardUsecase) UpdateTask(ctx context.Context, id int64, name *string, priority *int32, status *int32, handlerUserID *string, description *string, ruleChainID *string, lastRunID *string) (*entity.TaskBoard, error) {
	task, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("任务不存在")
	}
	// 仅待处理状态允许修改规则链关联
	if ruleChainID != nil && task.Status != entity.TaskStatusPending {
		return nil, errors.New("仅待处理状态的任务可以修改关联的规则链")
	}
	// 校验关联的规则链是否存在（非空时）
	if ruleChainID != nil && *ruleChainID != "" {
		if uc.ruleEngine != nil {
			if _, loaded := uc.ruleEngine.Get(*ruleChainID); !loaded {
				return nil, errors.New("关联的规则链不存在或未部署")
			}
		}
	}
	if name != nil {
		task.Name = *name
	}
	if priority != nil {
		task.Priority = *priority
	}
	if status != nil {
		task.Status = *status
	}
	if handlerUserID != nil {
		task.HandlerUserID = *handlerUserID
	}
	if description != nil {
		task.Description = *description
	}
	// ruleChainID 非nil时更新：空字符串=清除关联，非空=设置新关联
	if ruleChainID != nil {
		task.RuleChainID = *ruleChainID
	}
	if lastRunID != nil {
		task.LastRunID = *lastRunID
	}
	task.UpdatedAt = time.Now()
	if err := uc.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// DeleteTask 删除任务
func (uc *TaskBoardUsecase) DeleteTask(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// ExecuteTaskRuleChain 执行任务关联的规则链
func (uc *TaskBoardUsecase) ExecuteTaskRuleChain(ctx context.Context, id int64) error {
	task, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("任务不存在")
	}
	if task.Status != entity.TaskStatusPending {
		return errors.New("仅待处理状态的任务可以执行")
	}
	if task.RuleChainID == "" {
		return errors.New("任务未关联规则链")
	}
	if uc.ruleEngine == nil {
		return errors.New("规则引擎未配置")
	}
	engine, loaded := uc.ruleEngine.Get(task.RuleChainID)
	if !loaded {
		return errors.New("关联的规则链未部署")
	}

	// 先原子性更新状态为处理中，防止并发重复触发
	processingStatus := int32(entity.TaskStatusProcessing)
	updated, err := uc.UpdateTask(ctx, id, nil, nil, &processingStatus, nil, nil, nil, nil)
	if err != nil {
		return err
	}
	task = updated

	// 构建包含记忆上下文的 metadata
	var parentTask *entity.TaskBoard
	if task.ParentID != nil && *task.ParentID > 0 {
		parentTask, _ = uc.repo.GetByID(ctx, *task.ParentID)
	}
	metadata := uc.buildMemoryMetadata(ctx, task, parentTask)

	// 注入身份元数据，确保规则链内的 Agent 节点能获取正确的身份和记忆上下文
	injectIdentityMetadataFromContext(ctx, metadata)
	// 设置看板的记忆路径，使 Agent 运行时通过此路径读写记忆（而非 session/xxx）
	metadata.PutValue(projectPathContextKey, fmt.Sprintf("task_board/%d", task.ID))
	ensureIdentityMetadataDefaults(metadata)

	// 将父看板的记忆预填充到当前看板的 project memory 中，
	// 使得 Agent 运行时通过 ContextManager.BuildMessages 自然地读到父看板上下文
	uc.inheritParentMemory(ctx, task, parentTask)

	// 构建任务信息 data
	taskData := map[string]interface{}{
		"task_id":          task.ID,
		"task_name":        task.Name,
		"task_description": task.Description,
		"task_type":        task.Type,
		"task_priority":    task.Priority,
		"handler_user_id":  task.HandlerUserID,
	}
	if task.ParentID != nil {
		taskData["parent_id"] = *task.ParentID
	}
	dataBytes, _ := json.Marshal(taskData)

	msg := types.RuleMsg{
		Id:       uuid.NewString(),
		Data:     types.NewSharedData(string(dataBytes)),
		Type:     "TASK_BOARD",
		DataType: types.JSON,
		Metadata: metadata,
	}

	var ctxOpts []types.RuleContextOption
	ctxOpts = append(ctxOpts, uc.withOnRuleChainCompleted(task.ID))
	engine.OnMsg(msg, ctxOpts...)

	return nil
}

// buildMemoryMetadata 构建包含记忆上下文的规则链执行 metadata
func (uc *TaskBoardUsecase) buildMemoryMetadata(ctx context.Context, task *entity.TaskBoard, parentTask *entity.TaskBoard) *types.Metadata {
	metadata := types.NewMetadata()
	metadata.PutValue("task_id", strconv.FormatInt(task.ID, 10))

	// 注入父任务的记忆上下文
	if parentTask != nil && uc.memoryStore != nil {
		parentProjectPath := fmt.Sprintf("task_board/%d", parentTask.ID)
		if projMem, err := uc.memoryStore.GetProjectMemory(ctx, parentProjectPath); err == nil && projMem != nil {
			if ctxStr := projMem.BuildContext(); ctxStr != "" {
				metadata.PutValue("parent_memory_context", ctxStr)
			}
		}
		if parentTask.HandlerUserID != "" {
			if userMem, err := uc.memoryStore.GetUserMemory(ctx, parentTask.HandlerUserID); err == nil && userMem != nil {
				if ctxStr := userMem.BuildContext(); ctxStr != "" {
					metadata.PutValue("parent_user_memory_context", ctxStr)
				}
			}
		}
		metadata.PutValue("parent_task_id", strconv.FormatInt(parentTask.ID, 10))
		metadata.PutValue("parent_task_name", parentTask.Name)
		metadata.PutValue("parent_task_status", strconv.FormatInt(int64(parentTask.Status), 10))
	}

	// 注入当前任务自身的记忆上下文
	if uc.memoryStore != nil {
		currentProjectPath := fmt.Sprintf("task_board/%d", task.ID)
		if projMem, err := uc.memoryStore.GetProjectMemory(ctx, currentProjectPath); err == nil && projMem != nil {
			if ctxStr := projMem.BuildContext(); ctxStr != "" {
				metadata.PutValue("memory_context", ctxStr)
			}
		}
	}

	return metadata
}

// hasMemoryContent 检查 ProjectMemory 是否包含任何内容
func hasMemoryContent(mem *ProjectMemory) bool {
	if mem == nil {
		return false
	}
	return len(mem.Facts.Entries) > 0 || len(mem.Decisions.Entries) > 0 || len(mem.Summaries.Entries) > 0
}

// inheritParentMemory 将父看板的记忆预填充到当前看板的 project memory 中，
// 使 Agent 运行时通过 ContextManager.BuildMessages 自然地读到父看板上下文。
// 仅在当前看板尚无自身记忆时填充，避免覆盖已有内容。
func (uc *TaskBoardUsecase) inheritParentMemory(ctx context.Context, task *entity.TaskBoard, parentTask *entity.TaskBoard) {
	if uc.memoryStore == nil || parentTask == nil {
		return
	}
	currentPath := fmt.Sprintf("task_board/%d", task.ID)
	// 如果当前看板已有自身记忆，不覆盖
	if existing, err := uc.memoryStore.GetProjectMemory(ctx, currentPath); err == nil && hasMemoryContent(existing) {
		return
	}

	parentPath := fmt.Sprintf("task_board/%d", parentTask.ID)
	parentMem, err := uc.memoryStore.GetProjectMemory(ctx, parentPath)
	if err != nil || parentMem == nil {
		return
	}

	for _, entry := range parentMem.Facts.Entries {
		if err := uc.memoryStore.AddProjectFact(ctx, currentPath, entry.Content, "inherited_from_parent:"+parentPath); err != nil {
			uc.log.Warnf("inherit parent fact failed: %v", err)
		}
	}
	for _, entry := range parentMem.Decisions.Entries {
		if err := uc.memoryStore.AddDecision(ctx, currentPath, entry.Content); err != nil {
			uc.log.Warnf("inherit parent decision failed: %v", err)
		}
	}
	for _, entry := range parentMem.Summaries.Entries {
		if err := uc.memoryStore.AddSessionSummary(ctx, currentPath, entry.Content); err != nil {
			uc.log.Warnf("inherit parent summary failed: %v", err)
		}
	}
}

// withOnRuleChainCompleted 规则链执行完成回调，记录运行日志并保存看板记忆
// withOnRuleChainCompleted 规则链执行完成回调，更新状态、记录运行日志、保存看板记忆
func (uc *TaskBoardUsecase) withOnRuleChainCompleted(taskID int64) types.RuleContextOption {
	return types.WithOnRuleChainCompleted(func(ctn types.RuleContext, snapshot types.RuleChainRunSnapshot) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 判断是否有节点执行失败
		hasFailed := false
		for _, log := range snapshot.Logs {
			if log.Err != "" {
				hasFailed = true
				break
			}
		}

		// 根据执行结果更新任务状态，同时记录 last_run_id
		lastRunID := snapshot.Id
		if hasFailed {
			failedStatus := int32(entity.TaskStatusFailed)
			if _, err := uc.UpdateTask(ctx, taskID, nil, nil, &failedStatus, nil, nil, nil, &lastRunID); err != nil {
				uc.log.Errorf("update task status to failed failed, taskID=%d: %v", taskID, err)
			}
		} else {
			completedStatus := int32(entity.TaskStatusCompleted)
			if _, err := uc.UpdateTask(ctx, taskID, nil, nil, &completedStatus, nil, nil, nil, &lastRunID); err != nil {
				uc.log.Errorf("update task status to completed failed, taskID=%d: %v", taskID, err)
			}
		}

		if uc.runLogRepo != nil {
			nodelogs, _ := json.Marshal(snapshot.Logs)
			additionalInfo, _ := json.Marshal(snapshot.AdditionalInfo)
			ruleChainInfo, _ := json.Marshal(snapshot.RuleChain)
			md, _ := json.Marshal(snapshot.Metadata)
			t := time.Now()
			err := uc.runLogRepo.CreateRunLog(ctx, &entity.RunLog{
				RunID:          snapshot.Id,
				ChainID:        snapshot.RuleChain.RuleChain.ID,
				ChainName:      snapshot.RuleChain.RuleChain.Name,
				NodeLog:        string(nodelogs),
				AdditionalInfo: string(additionalInfo),
				RuleChainInfo:  string(ruleChainInfo),
				Metadata:       string(md),
				StartTs:        snapshot.StartTs,
				EndTs:          snapshot.EndTs,
				CreatedAt:      &t,
				UpdatedAt:      &t,
			})
			if err != nil {
				uc.log.Errorf("create run log failed, error: %v", err)
			}
		}

		// 保存看板执行记忆，使后续子看板可以继承父看板的执行上下文
		if uc.memoryStore != nil && taskID > 0 {
			projectPath := fmt.Sprintf("task_board/%d", taskID)
			summary := fmt.Sprintf("规则链[%s]执行完成，链ID=%s，耗时=%.0fms",
				snapshot.RuleChain.RuleChain.Name,
				snapshot.RuleChain.RuleChain.ID,
				float64(snapshot.EndTs-snapshot.StartTs)/1e6)
			if err := uc.memoryStore.AddSessionSummary(ctx, projectPath, summary); err != nil {
				uc.log.Warnf("save task board memory failed, taskID=%d: %v", taskID, err)
			}
		}
	})
}

const defaultChildTaskSuffix = "-子任务"

// CreateChildTask 创建子任务（仅已完成或处理失败的任务可创建）
func (uc *TaskBoardUsecase) CreateChildTask(ctx context.Context, parentID int64, nameSuffix string) (*entity.TaskBoard, error) {
	parent, err := uc.repo.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, errors.New("父任务不存在")
	}
	if parent.Status != entity.TaskStatusCompleted && parent.Status != entity.TaskStatusFailed {
		return nil, errors.New("仅已完成或处理失败的任务可以创建子任务")
	}
	if nameSuffix == "" {
		nameSuffix = defaultChildTaskSuffix
	}

	// 校验继承的规则链是否仍然存在/已部署
	inheritedRuleChainID := parent.RuleChainID
	if inheritedRuleChainID != "" && uc.ruleEngine != nil {
		if _, loaded := uc.ruleEngine.Get(inheritedRuleChainID); !loaded {
			uc.log.Warnf("父任务关联的规则链[%s]已不存在，子任务将不继承该关联", inheritedRuleChainID)
			inheritedRuleChainID = ""
		}
	}

	child := &entity.TaskBoard{
		Name:          parent.Name + nameSuffix,
		Priority:      parent.Priority,
		Status:        entity.TaskStatusPending,
		Type:          parent.Type,
		HandlerUserID: parent.HandlerUserID,
		Description:   parent.Description,
		RuleChainID:   inheritedRuleChainID,
		ParentID:      &parentID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := uc.repo.Create(ctx, child); err != nil {
		return nil, err
	}
	return child, nil
}

// ListChildTasks 查询子任务列表
func (uc *TaskBoardUsecase) ListChildTasks(ctx context.Context, parentID int64, page, pageSize int32) ([]*entity.TaskBoard, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return uc.repo.ListByParentID(ctx, parentID, page, pageSize)
}
