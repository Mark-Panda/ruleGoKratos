package service

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ v1.TaskBoardServiceServer = (*TaskBoardService)(nil)
var _ v1.ServiceManagementServiceServer = (*TaskBoardService)(nil)

// TaskBoardService 任务看板服务
type TaskBoardService struct {
	v1.UnimplementedTaskBoardServiceServer
	v1.UnimplementedServiceManagementServiceServer
	taskUsecase    *biz.TaskBoardUsecase
	serviceUsecase *biz.ServiceManagementUsecase
	log            *log.Helper
}

// NewTaskBoardService 创建任务看板服务实例
func NewTaskBoardService(taskUsecase *biz.TaskBoardUsecase, serviceUsecase *biz.ServiceManagementUsecase, logger log.Logger) *TaskBoardService {
	return &TaskBoardService{
		taskUsecase:    taskUsecase,
		serviceUsecase: serviceUsecase,
		log:            log.NewHelper(logger),
	}
}

// ------------------------------ 任务看板接口 ------------------------------

// CreateTask 创建任务
func (s *TaskBoardService) CreateTask(ctx context.Context, req *v1.CreateTaskReq) (*v1.CreateTaskReply, error) {
	task, err := s.taskUsecase.CreateTask(ctx, req.Name, req.Priority, int32(req.Type), req.HandlerUserId, req.Description)
	if err != nil {
		return nil, err
	}
	return &v1.CreateTaskReply{
		Task: convertTaskToPB(task),
	}, nil
}

// GetTask 获取任务详情
func (s *TaskBoardService) GetTask(ctx context.Context, req *v1.GetTaskReq) (*v1.GetTaskReply, error) {
	task, err := s.taskUsecase.GetTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return &v1.GetTaskReply{}, nil
	}
	return &v1.GetTaskReply{
		Task: convertTaskToPB(task),
	}, nil
}

// ListTasks 查询任务列表
func (s *TaskBoardService) ListTasks(ctx context.Context, req *v1.ListTasksReq) (*v1.ListTasksReply, error) {
	tasks, total, err := s.taskUsecase.ListTasks(ctx, int32(req.Status), int32(req.Type), req.HandlerUserId, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	pbTasks := make([]*v1.TaskBoard, len(tasks))
	for i, task := range tasks {
		pbTasks[i] = convertTaskToPB(task)
	}
	return &v1.ListTasksReply{
		Tasks: pbTasks,
		Total: total,
	}, nil
}

// UpdateTask 更新任务
func (s *TaskBoardService) UpdateTask(ctx context.Context, req *v1.UpdateTaskReq) (*v1.UpdateTaskReply, error) {
	var (
		name          *string
		priority      *int32
		status        *int32
		handlerUserID *string
		description   *string
	)
	if req.Name != "" {
		name = &req.Name
	}
	if req.Priority > 0 {
		priority = &req.Priority
	}
	if req.Status != v1.TaskStatus_TASK_STATUS_UNSPECIFIED {
		s := int32(req.Status)
		status = &s
	}
	if req.HandlerUserId != "" {
		handlerUserID = &req.HandlerUserId
	}
	if req.Description != "" {
		description = &req.Description
	}
	task, err := s.taskUsecase.UpdateTask(ctx, req.Id, name, priority, status, handlerUserID, description)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateTaskReply{
		Task: convertTaskToPB(task),
	}, nil
}

// DeleteTask 删除任务
func (s *TaskBoardService) DeleteTask(ctx context.Context, req *v1.DeleteTaskReq) (*v1.DeleteTaskReply, error) {
	if err := s.taskUsecase.DeleteTask(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteTaskReply{
		Success: true,
	}, nil
}

// ------------------------------ 服务管理接口 ------------------------------

// CreateService 创建服务
func (s *TaskBoardService) CreateService(ctx context.Context, req *v1.CreateServiceReq) (*v1.CreateServiceReply, error) {
	// 创建接口兼容按名称保存语义：同名即更新，不存在则新建。
	service, err := s.serviceUsecase.SaveServiceByName(ctx, req.Name, int32(req.Status), req.VolcLogServiceId, req.GitRepoUrl, req.Description)
	if err != nil {
		return nil, err
	}
	return &v1.CreateServiceReply{
		Service: convertServiceToPB(service),
	}, nil
}

// GetService 获取服务详情
func (s *TaskBoardService) GetService(ctx context.Context, req *v1.GetServiceReq) (*v1.GetServiceReply, error) {
	service, err := s.serviceUsecase.GetService(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return &v1.GetServiceReply{}, nil
	}
	return &v1.GetServiceReply{
		Service: convertServiceToPB(service),
	}, nil
}

// ListServices 查询服务列表
func (s *TaskBoardService) ListServices(ctx context.Context, req *v1.ListServicesReq) (*v1.ListServicesReply, error) {
	services, total, err := s.serviceUsecase.ListServices(ctx, int32(req.Status), req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	pbServices := make([]*v1.ServiceManagement, len(services))
	for i, service := range services {
		pbServices[i] = convertServiceToPB(service)
	}
	return &v1.ListServicesReply{
		Services: pbServices,
		Total:    total,
	}, nil
}

// UpdateService 更新服务
func (s *TaskBoardService) UpdateService(ctx context.Context, req *v1.UpdateServiceReq) (*v1.UpdateServiceReply, error) {
	var (
		name             *string
		status           *int32
		volcLogServiceID *string
		gitRepoURL       *string
		description      *string
	)
	if req.Name != "" {
		name = &req.Name
	}
	if req.Status != v1.ServiceStatus_SERVICE_STATUS_UNSPECIFIED {
		s := int32(req.Status)
		status = &s
	}
	if req.VolcLogServiceId != "" {
		volcLogServiceID = &req.VolcLogServiceId
	}
	if req.GitRepoUrl != "" {
		gitRepoURL = &req.GitRepoUrl
	}
	if req.Description != "" {
		description = &req.Description
	}
	service, err := s.serviceUsecase.UpdateService(ctx, req.Id, name, status, volcLogServiceID, gitRepoURL, description)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateServiceReply{
		Service: convertServiceToPB(service),
	}, nil
}

// DeleteService 删除服务
func (s *TaskBoardService) DeleteService(ctx context.Context, req *v1.DeleteServiceReq) (*v1.DeleteServiceReply, error) {
	if err := s.serviceUsecase.DeleteService(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteServiceReply{
		Success: true,
	}, nil
}

// ------------------------------ 转换函数 ------------------------------

// convertTaskToPB 把entity.TaskBoard转换为PB结构
func convertTaskToPB(task *entity.TaskBoard) *v1.TaskBoard {
	if task == nil {
		return nil
	}
	pbTask := &v1.TaskBoard{
		Id:            task.ID,
		Name:          task.Name,
		Priority:      task.Priority,
		Status:        v1.TaskStatus(task.Status),
		Type:          v1.TaskType(task.Type),
		CreatedAt:     timestamppb.New(task.CreatedAt),
		UpdatedAt:     timestamppb.New(task.UpdatedAt),
		HandlerUserId: task.HandlerUserID,
		Description:   task.Description,
	}
	if task.DeletedAt != nil {
		pbTask.DeletedAt = timestamppb.New(*task.DeletedAt)
	}
	return pbTask
}

// convertServiceToPB 把entity.ServiceManagement转换为PB结构
func convertServiceToPB(service *entity.ServiceManagement) *v1.ServiceManagement {
	if service == nil {
		return nil
	}
	pbService := &v1.ServiceManagement{
		Id:               service.ID,
		Name:             service.Name,
		Status:           v1.ServiceStatus(service.Status),
		VolcLogServiceId: service.VolcLogServiceID,
		GitRepoUrl:       service.GitRepoURL,
		CreatedAt:        timestamppb.New(service.CreatedAt),
		UpdatedAt:        timestamppb.New(service.UpdatedAt),
		Description:      service.Description,
	}
	if service.DeletedAt != nil {
		pbService.DeletedAt = timestamppb.New(*service.DeletedAt)
	}
	return pbService
}
