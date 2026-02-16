package agentic

import (
	"context"
	"fmt"
	"time"
)

func (s *AgenticSkill) handleCreateTaskPlan(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	goal, _ := args["goal"].(string)
	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}

	steps := []string{}
	if s, ok := args["steps"].([]interface{}); ok {
		for _, step := range s {
			if str, ok := step.(string); ok {
				steps = append(steps, str)
			}
		}
	}

	return map[string]interface{}{
		"goal":       goal,
		"steps":      steps,
		"created_at": time.Now().Format(time.RFC3339),
		"status":     "planning",
	}, nil
}

func (s *AgenticSkill) handleReflectOnTask(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	task, _ := args["task"].(string)
	currentState, _ := args["current_state"].(string)

	actions := []string{}
	if a, ok := args["actions_taken"].([]interface{}); ok {
		for _, action := range a {
			if str, ok := action.(string); ok {
				actions = append(actions, str)
			}
		}
	}

	return map[string]interface{}{
		"task":          task,
		"actions_taken": len(actions),
		"current_state": currentState,
		"suggestions": []string{
			"Consider verifying the current state",
			"Check if all prerequisites are met",
			"Document progress so far",
		},
	}, nil
}
