package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

// RuleGoService is a rulego service.
type RunLogService struct {
	v1.UnimplementedRunLogServer
	rlu *biz.RunLogUsecase
}

// NewRunLogService new a runlog service.
func NewRunLogService(rlu *biz.RunLogUsecase) *RunLogService {
	return &RunLogService{rlu: rlu}
}

func (s *RunLogService) ListRunLogs(ctx context.Context, req *v1.ListRunLogsRequest) (*v1.ListRunLogsReply, error) {
	page := int(req.Page)
	size := int(req.Size)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	params := ""
	if req.StartTime != "" && req.EndTime != "" {
		params = fmt.Sprintf(`created_at >= '%s' and created_at <= '%s'`, req.StartTime, req.EndTime)
	}
	list, total, err := s.rlu.List(ctx, page, size, params)
	if err != nil {
		return nil, err
	}

	items := make([]*v1.RunLogItem, 0, len(list))
	for _, item := range list {
		items = append(items, toRunLogItem(&item))
	}

	return &v1.ListRunLogsReply{
		Items: items,
		Page:  int32(page),
		Size:  int32(size),
		Total: int32(total),
	}, nil
}

func (s *RunLogService) GetRunLogByMsgId(ctx context.Context, req *v1.GetRunLogByMsgIdReq) (*v1.RunLogItem, error) {
	item, err := s.rlu.FindOne(ctx, req.MsgId)
	if err != nil {
		// 前端轮询用 404 表示「尚未落库」，与 WorkflowExecuteSection 注释一致；勿将 RecordNotFound 透出为 500。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "run log not found for msgId=%s", req.MsgId)
		}
		return nil, err
	}
	return toRunLogItem(item), nil
}

func emptyStructPB() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]interface{}{})
	return s
}

func mapToStructPB(m map[string]interface{}) *structpb.Struct {
	if m == nil {
		return emptyStructPB()
	}
	s, err := structpb.NewStruct(m)
	if err != nil || s == nil {
		return emptyStructPB()
	}
	return s
}

func toRunLogItem(item *entity.RunLog) *v1.RunLogItem {
	if item == nil {
		return nil
	}
	ruleChainStruct := emptyStructPB()
	if item.RuleChainInfo != "" {
		var ruleChainMap map[string]interface{}
		if json.Unmarshal([]byte(item.RuleChainInfo), &ruleChainMap) == nil && ruleChainMap != nil {
			ruleChainStruct = mapToStructPB(ruleChainMap)
		}
	}

	metadataStruct := emptyStructPB()
	if item.Metadata != "" {
		var metadataMap map[string]interface{}
		if json.Unmarshal([]byte(item.Metadata), &metadataMap) == nil && metadataMap != nil {
			metadataStruct = mapToStructPB(metadataMap)
		}
	}

	var logsList []map[string]interface{}
	if item.NodeLog != "" {
		_ = json.Unmarshal([]byte(item.NodeLog), &logsList)
	}
	logsStructs := make([]*structpb.Struct, 0, len(logsList))
	for _, logItem := range logsList {
		if logItem == nil {
			continue
		}
		logStruct, err := structpb.NewStruct(logItem)
		if err != nil || logStruct == nil {
			continue
		}
		logsStructs = append(logsStructs, logStruct)
	}

	return &v1.RunLogItem{
		Id:        strconv.FormatInt(item.ID, 10),
		RuleChain: ruleChainStruct,
		Metadata:  metadataStruct,
		StartTs:   item.StartTs,
		EndTs:     item.EndTs,
		Logs:      logsStructs,
	}
}
