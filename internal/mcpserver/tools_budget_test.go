package mcpserver_test

import (
	"encoding/json"
	"testing"
)

// Budget for the compact tools/list result. Two starter tools are far below
// it; a new tool with a long description or an accidental outputSchema on the
// text-only tools will cross it. Raise it only on purpose, together with the
// golden file.
const budgetBytes = 2048

func TestToolsListBudget(t *testing.T) {
	tools := listTools(t)
	raw, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("marshal tools: %v", err)
	}
	t.Logf("%d tools, tools/list = %d bytes (~%d tokens), budget %d", len(tools), len(raw), len(raw)/4, budgetBytes)
	if len(raw) > budgetBytes {
		t.Errorf("tools/list is %d bytes, over the budget of %d; shorten new descriptions/schemas or raise the budget on purpose", len(raw), budgetBytes)
	}
}
