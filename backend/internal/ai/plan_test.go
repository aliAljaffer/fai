package ai

import "testing"

func TestParseTaskPlan(t *testing.T) {
	plan, err := parseTaskPlan("Here is the plan:\n```json\n" +
		`{"title":"Launch the site","subtasks":["Write copy","  Write copy  ","Build page","","Review"]}` +
		"\n```")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Title != "Launch the site" {
		t.Errorf("title = %q", plan.Title)
	}
	if len(plan.Subtasks) != 3 {
		t.Fatalf("subtasks = %#v", plan.Subtasks)
	}
	if plan.Subtasks[0] != "Write copy" || plan.Subtasks[2] != "Review" {
		t.Errorf("subtasks = %#v", plan.Subtasks)
	}
}

func TestParseTaskPlanRejectsEmpty(t *testing.T) {
	if _, err := parseTaskPlan(`{"title":"","subtasks":["x"]}`); err == nil {
		t.Error("expected an error for an empty title")
	}
	if _, err := parseTaskPlan(`{"title":"x","subtasks":[]}`); err == nil {
		t.Error("expected an error for an empty subtask list")
	}
}
