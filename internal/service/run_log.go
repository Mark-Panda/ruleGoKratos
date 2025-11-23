package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"

	"google.golang.org/protobuf/types/known/structpb"
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
		var ruleChainMap map[string]interface{}
		if item.RuleChainInfo != "" {
			_ = json.Unmarshal([]byte(item.RuleChainInfo), &ruleChainMap)
		}
		ruleChainStruct, _ := structpb.NewStruct(ruleChainMap)

		var metadataMap map[string]interface{}
		if item.Metadata != "" {
			_ = json.Unmarshal([]byte(item.Metadata), &metadataMap)
		}
		metadataStruct, _ := structpb.NewStruct(metadataMap)

		var logsList []map[string]interface{}
		if item.NodeLog != "" {
			_ = json.Unmarshal([]byte(item.NodeLog), &logsList)
		}
		logsStructs := make([]*structpb.Struct, 0, len(logsList))
		for _, logItem := range logsList {
			logStruct, _ := structpb.NewStruct(logItem)
			logsStructs = append(logsStructs, logStruct)
		}

		items = append(items, &v1.RunLogItem{
			Id:        strconv.FormatInt(item.ID, 10),
			RuleChain: ruleChainStruct,
			Metadata:  metadataStruct,
			StartTs:   item.StartTs,
			EndTs:     item.EndTs,
			Logs:      logsStructs,
		})
	}

	return &v1.ListRunLogsReply{
		Items: items,
		Page:  int32(page),
		Size:  int32(size),
		Total: int32(total),
	}, nil
}
