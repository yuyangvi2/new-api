package claudemessages

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func claudeToolChoiceToOpenAI(toolChoice any) any {
	choice, err := common.Any2Type[dto.ClaudeToolChoice](toolChoice)
	if err != nil {
		if str, ok := toolChoice.(string); ok {
			return str
		}
		common.SysError(fmt.Sprintf("claudeToolChoiceToOpenAI: unexpected tool_choice format: %v, error: %v", toolChoice, err))
		return toolChoice
	}
	switch choice.Type {
	case "none":
		return "none"
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "tool":
		if strings.TrimSpace(choice.Name) == "" {
			return "required"
		}
		return map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": choice.Name,
			},
		}
	default:
		return toolChoice
	}
}
