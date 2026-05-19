package taskengine

import (
	"context"
	"testing"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/pkg/matcher"
)

func TestMDCTScorer_BasicScoring(t *testing.T) {
	weights := config.MDCTConfig{
		W1:             1000,
		W2:             10,
		W3:             10,
		W4:             100,
		AllowAiArbiter: false,
	}

	scorer := NewMDCTScorer(weights)
	// mock aiService is not strictly needed for basic scoring where diff > 5% or AiArbiter=false

	record := map[string]interface{}{
		"age":   25,
		"name":  "Alice",
		"email": "alice@test.com",
	}

	// Rule 1: High priority
	rule1 := parsedRule{
		model: &model.SysMatchRule{
			BaseModel: model.BaseModel{ID: 1},
			Priority: 2,
			Name:     "High Priority Rule",
		},
		mRule: matcher.MatchRule{
			Field:    "age",
			Operator: "greater_than",
			Value:    20,
		},
	}

	// Rule 2: Low priority but more complex
	rule2 := parsedRule{
		model: &model.SysMatchRule{
			BaseModel: model.BaseModel{ID: 2},
			Priority: 1,
			Name:     "Low Priority Rule",
		},
		mRule: matcher.MatchRule{
			And: []matcher.MatchRule{
				{Field: "age", Operator: "greater_than", Value: 18},
				{Field: "name", Operator: "equals", Value: "Alice"},
			},
		},
	}

	matchedRules := []parsedRule{rule1, rule2}

	ctx := context.Background()
	scored := scorer.EvaluateAndSort(ctx, record, matchedRules)

	if len(scored) != 2 {
		t.Fatalf("Expected 2 scored rules, got %d", len(scored))
	}

	if scored[0].ParsedRule.model.ID != 1 {
		t.Errorf("Expected rule 1 to win due to W1 (priority), but got rule %d", scored[0].ParsedRule.model.ID)
	}

	// Rule 1 PriorityScore = 2 * 1000 = 2000
	// W2 = 1.2 * 10 * 10 = 120
	// W3 = 3/3 * 100 * 10 = 1000
	// Total roughly ~3120
	if scored[0].FinalScore < 2000 {
		t.Errorf("Expected Rule 1 score > 2000, got %v", scored[0].FinalScore)
	}

	// 验证置信度
	if scored[0].Confidence <= scored[1].Confidence {
		t.Errorf("Expected rule 1 confidence > rule 2 confidence")
	}
}
