// 多维共识打标算法计算模块
package taskengine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/pkg/logger"
	"TagMatrix/internal/pkg/matcher"
	"TagMatrix/internal/service/aiengine"
)

// MDCTScorer 多维共识打标算法计算器
type MDCTScorer struct {
	Weights   config.MDCTConfig
	AiService *aiengine.AIEngineService
}

// ScoredRule 包装经过打分后的规则
type ScoredRule struct {
	ParsedRule          parsedRule
	BaseScore           float64
	FinalScore          float64
	Confidence          float64
	IsAiIntervened      bool
	AiArbitrationReason string
}

// NewMDCTScorer 创建打分器
func NewMDCTScorer(weights config.MDCTConfig) *MDCTScorer {
	return &MDCTScorer{
		Weights:   weights,
		AiService: aiengine.NewAIEngineService(),
	}
}

// EvaluateAndSort 对命中的规则集合进行完整 MDCT 打分、AI 仲裁与归一化排序
func (s *MDCTScorer) EvaluateAndSort(ctx context.Context, record map[string]interface{}, matchedRules []parsedRule) []ScoredRule {
	if len(matchedRules) == 0 {
		return nil
	}

	var scoredRules []ScoredRule
	for _, pr := range matchedRules {
		baseScore := s.RankScore(record, pr)
		scoredRules = append(scoredRules, ScoredRule{
			ParsedRule: pr,
			BaseScore:  baseScore,
			FinalScore: baseScore,
		})
	}

	// 按基础分降序排序
	sort.SliceStable(scoredRules, func(i, j int) bool {
		if scoredRules[i].BaseScore != scoredRules[j].BaseScore {
			return scoredRules[i].BaseScore > scoredRules[j].BaseScore
		}
		// 兜底：ID大的优先
		return scoredRules[i].ParsedRule.model.ID > scoredRules[j].ParsedRule.model.ID
	})

	// 只有一个命中规则，直接归一化100%并返回
	if len(scoredRules) == 1 {
		scoredRules[0].Confidence = 100.0
		return scoredRules
	}

	// 检查前两名是否分数相近 (差值 < 5%)，并且允许 AI 介入
	r1 := &scoredRules[0]
	r2 := &scoredRules[1]

	if s.Weights.AllowAiArbiter && r1.BaseScore > 0 {
		diffRatio := (r1.BaseScore - r2.BaseScore) / r1.BaseScore
		if diffRatio < 0.05 {
			// 触发 AI 语义裁决 (W4)
			winnerIndex, reason, err := s.AskAIToArbitrate(ctx, record, r1.ParsedRule, r2.ParsedRule)
			if err != nil {
				logger.Error(fmt.Sprintf("[MDCT] AI Arbitration failed: %v", err))
			} else {
				// 注入 W4 得分
				w4Score := float64(s.Weights.W4)
				switch winnerIndex {
				case 0:
					r1.FinalScore += w4Score
					r1.IsAiIntervened = true
					r1.AiArbitrationReason = reason
				case 1:
					r2.FinalScore += w4Score
					r2.IsAiIntervened = true
					r2.AiArbitrationReason = reason
				}

				// 重新按最终总分排序
				sort.SliceStable(scoredRules, func(i, j int) bool {
					if scoredRules[i].FinalScore != scoredRules[j].FinalScore {
						return scoredRules[i].FinalScore > scoredRules[j].FinalScore
					}
					return scoredRules[i].ParsedRule.model.ID > scoredRules[j].ParsedRule.model.ID
				})
			}
		}
	}

	// 得分归一化与置信度百分比计算
	var totalFinalScore float64 = 0
	for _, sr := range scoredRules {
		totalFinalScore += sr.FinalScore
	}

	for i := range scoredRules {
		if totalFinalScore > 0 {
			scoredRules[i].Confidence = (scoredRules[i].FinalScore / totalFinalScore) * 100.0
		} else {
			scoredRules[i].Confidence = 0
		}
	}

	return scoredRules
}

// AskAIToArbitrate 请求大模型进行最终裁决
// 返回值: 获胜索引 (0代表r1, 1代表r2), 理由, 错误
func (s *MDCTScorer) AskAIToArbitrate(ctx context.Context, record map[string]interface{}, r1, r2 parsedRule) (int, string, error) {
	recordJSON, _ := json.Marshal(record)

	prompt := fmt.Sprintf(`你是一个数据风控专家。现有一条数据：%s。
它同时符合【%s】(选项A) 和【%s】(选项B) 的打标规则，且置信度一致。
基于商业常识，哪个标签更适合作为它的主标签？
请务必以如下格式回复：
裁决结果：A（或B）
判定理由：你的简短理由...`,
		string(recordJSON), r1.model.Name, r2.model.Name)

	resp, err := s.AiService.ChatWithAI(ctx, prompt)
	if err != nil {
		return -1, "", err
	}

	// 解析大模型返回的结果
	respUpper := strings.ToUpper(resp)
	lines := strings.Split(strings.ReplaceAll(resp, "\r\n", "\n"), "\n")

	winner := -1
	reason := resp

	// 简单解析逻辑：寻找 "裁决结果："
	for _, line := range lines {
		if strings.Contains(strings.ToUpper(line), "裁决结果") {
			if strings.Contains(strings.ToUpper(line), "A") {
				winner = 0
			} else if strings.Contains(strings.ToUpper(line), "B") {
				winner = 1
			}
		} else if strings.Contains(line, "判定理由") {
			parts := strings.SplitN(line, "：", 2)
			if len(parts) == 2 {
				reason = strings.TrimSpace(parts[1])
			} else {
				parts = strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					reason = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Fallback 解析
	if winner == -1 {
		if strings.Contains(respUpper, "选项A") || strings.HasPrefix(strings.TrimSpace(respUpper), "A") {
			winner = 0
		} else if strings.Contains(respUpper, "选项B") || strings.HasPrefix(strings.TrimSpace(respUpper), "B") {
			winner = 1
		}
	}

	if winner == -1 {
		return -1, "", fmt.Errorf("AI response unclear: %s", resp)
	}

	return winner, reason, nil
}

// RankScore 计算不含 AI 裁决的基础总分 (W1+W2+W3)
func (s *MDCTScorer) RankScore(record map[string]interface{}, rule parsedRule) float64 {
	// W1 优先级打分
	w1Score := s.PriorityScore(rule.model)

	// W2 逻辑深度打分
	w2Score := s.ComplexityScore(rule.mRule)

	// W3 数据完整度打分
	w3Score := s.CompletenessScore(record)

	return float64(s.Weights.W1)*w1Score +
		float64(s.Weights.W2)*w2Score +
		float64(s.Weights.W3)*w3Score
}

// PriorityScore (W1) 计算规则的优先级得分
func (s *MDCTScorer) PriorityScore(rule *model.SysMatchRule) float64 {
	// 直接返回规则的 Priority
	return float64(rule.Priority)
}

// ComplexityScore (W2) 计算规则逻辑深度
// 基础分：每多一层 AND/OR 节点 +10
// 算子系数：精确=1.5，范围/正则=1.2，模糊=1.0
func (s *MDCTScorer) ComplexityScore(rule matcher.MatchRule) float64 {
	return s.calculateRuleComplexity(rule)
}

func (s *MDCTScorer) calculateRuleComplexity(rule matcher.MatchRule) float64 {
	var score float64 = 0

	// 处理分支节点
	if len(rule.And) > 0 {
		score += 10.0
		for _, sub := range rule.And {
			score += s.calculateRuleComplexity(sub)
		}
	} else if len(rule.Or) > 0 {
		score += 10.0
		for _, sub := range rule.Or {
			score += s.calculateRuleComplexity(sub)
		}
	} else {
		// 叶子节点 (条件节点)
		if rule.Field != "" && rule.Operator != "" {
			// 根据操作符给予不同的确信度基础分 (假设默认基础分10，乘算子系数)
			base := 10.0
			switch rule.Operator {
			case "equals", "not_equals", "is_null", "is_not_null":
				score += base * 1.5 // 精确匹配
			case "greater_than", "less_than", "greater_than_or_equal", "less_than_or_equal", "regex", "cidr", "in", "not_in":
				score += base * 1.2 // 范围正则
			case "contains", "not_contains", "starts_with", "ends_with", "like", "list_contains":
				score += base * 1.0 // 模糊匹配
			default:
				score += base
			}
		}
	}

	return score
}

// CompletenessScore (W3) 计算数据置信度
// 考察非 null 或空字符串的字段占比
func (s *MDCTScorer) CompletenessScore(record map[string]interface{}) float64 {
	if len(record) == 0 {
		return 0
	}

	validFields := 0
	totalFields := len(record)

	for _, val := range record {
		if val == nil {
			continue
		}

		// 检查空字符串
		if str, ok := val.(string); ok {
			if str == "" {
				continue
			}
		}

		validFields++
	}

	// 完整度得分可以设为 0 ~ 100
	return float64(validFields) / float64(totalFields) * 100.0
}
