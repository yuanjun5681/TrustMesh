package assistant

import (
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const systemPromptTemplate = `你是 TrustMesh 项目管理平台的智能助手。你可以帮助用户：
- 搜索知识库中的文档内容
- 查找和查看任务状态及详情
- 查看项目统计数据和项目列表
- 导航到指定页面

规则：
1. 只在用户明确请求相关信息时才调用 tools，不要主动调用。对于打招呼等闲聊直接回复即可，不需要获取数据
2. 使用提供的 tools 来获取信息，然后用简洁的中文回复用户
3. 回复时引用具体的任务名称、文档标题等，方便用户确认
4. 如果用户想导航到某个页面，使用 navigate tool
5. 如果搜索结果为空，友好地告知用户并建议换个关键词
6. 不要编造数据，只使用 tools 返回的真实数据`

func buildSystemPrompt(ctx *ChatContext) string {
	prompt := systemPromptTemplate
	if ctx != nil {
		if ctx.CurrentPage != "" {
			prompt += fmt.Sprintf("\n\n当前用户所在页面：%s", ctx.CurrentPage)
		}
		if ctx.ProjectID != "" {
			prompt += fmt.Sprintf("\n当前项目 ID：%s", ctx.ProjectID)
		}
	}
	return prompt
}

// BuildMessages converts the ChatRequest into OpenAI messages.
func BuildMessages(req *ChatRequest) []openai.ChatCompletionMessage {
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: buildSystemPrompt(req.Context)},
	}
	for _, h := range req.History {
		role := openai.ChatMessageRoleUser
		if h.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		msgs = append(msgs, openai.ChatCompletionMessage{Role: role, Content: h.Content})
	}
	msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: req.Message})
	return msgs
}
