package biz

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose" // 更新版本后应该可以正常导入
	"github.com/cloudwego/eino/schema"
)

type TodoAddParams struct {
	Content   string `json:"content"`
	StartedAt int64  `json:"started_at"`
	Deadline  int64  `json:"deadline"`
}

func (uc *AgentUsecase) RuleChainTestEino(ctx context.Context, userMessage string) error {
	// Create configuration
	config := &duckduckgo.Config{
		MaxResults: 3, // Limit to return 20 results
		Region:     duckduckgo.RegionWT,
		Timeout:    10 * time.Second,
	}

	// Create search client
	searchTool, err := duckduckgo.NewTextSearchTool(ctx, config)
	if err != nil {
		log.Fatalf("NewTextSearchTool of duckduckgo failed, err=%v", err)
	}
	// 初始化 tools
	todoTools := []tool.BaseTool{
		getAddTodoTool(), // NewTool 构建
		// updateTool,       // InferTool 构建
		// &ListTodoTool{},  // 实现Tool接口
		searchTool, // 官方封装的工具
	}
	// 创建并配置 ChatModel
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:  "gpt-4",
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}
	// 获取工具信息并绑定到 ChatModel
	toolInfos := make([]*schema.ToolInfo, 0, len(todoTools))
	for _, tool := range todoTools {
		info, err := tool.Info(ctx)
		if err != nil {
			log.Fatal(err)
		}
		toolInfos = append(toolInfos, info)
	}
	err = chatModel.BindTools(toolInfos)
	if err != nil {
		log.Fatal(err)
	}

	// 创建 tools 节点
	todoToolsNode, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
		Tools: todoTools,
	})
	if err != nil {
		log.Fatal(err)
	}
	// 构建完整的处理链
	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.
		AppendChatModel(chatModel, compose.WithNodeName("chat_model")).
		AppendToolsNode(todoToolsNode, compose.WithNodeName("tools"))

	// 编译并运行 chain
	agent, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 运行示例
	resp, err := agent.Invoke(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "添加一个学习 Eino 的 TODO，同时搜索一下 cloudwego/eino 的仓库地址",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 输出结果
	for _, msg := range resp {
		fmt.Println(msg.Content)
	}
	return nil
}

// 处理函数
func AddTodoFunc(_ context.Context, params *TodoAddParams) (string, error) {
	// Mock处理逻辑
	return `{"msg": "add todo success"}`, nil
}

func getAddTodoTool() tool.InvokableTool {
	// 工具信息
	info := &schema.ToolInfo{
		Name: "add_todo",
		Desc: "Add a todo item",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"content": {
				Desc:     "The content of the todo item",
				Type:     schema.String,
				Required: true,
			},
			"started_at": {
				Desc: "The started time of the todo item, in unix timestamp",
				Type: schema.Integer,
			},
			"deadline": {
				Desc: "The deadline of the todo item, in unix timestamp",
				Type: schema.Integer,
			},
		}),
	}

	// 使用NewTool创建工具
	return utils.NewTool(info, AddTodoFunc)
}
