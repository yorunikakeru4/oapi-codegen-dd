package gen

import (
	"testing"
)

func TestReport_TreeData_Item_Validate_NoRecursion(t *testing.T) {
	// Create a recursive tree structure
	item := Report_TreeData_Item{
		Value: "root",
		Children: []Report_TreeData_Item{
			{
				Value: "child1",
				Children: []Report_TreeData_Item{
					{Value: "grandchild1"},
				},
			},
			{
				Value: "child2",
			},
		},
	}

	// Should not cause infinite recursion
	err := item.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReport_TreeData_Item_Validate_Constraint(t *testing.T) {
	// Empty value should fail validation (minLength: 1)
	item := Report_TreeData_Item{
		Value: "",
	}

	err := item.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty value")
	}

	errStr := err.Error()
	if !contains(errStr, "Value") {
		t.Errorf("expected error to mention 'Value', got: %s", errStr)
	}
}

func TestReport_TreeData_Item_Validate_NestedConstraint(t *testing.T) {
	// Nested child with empty value should fail
	item := Report_TreeData_Item{
		Value: "root",
		Children: []Report_TreeData_Item{
			{Value: ""}, // invalid
		},
	}

	err := item.Validate()
	if err == nil {
		t.Fatal("expected validation error for nested empty value")
	}

	errStr := err.Error()
	// Should show path to the nested error
	if !contains(errStr, "Children[0]") {
		t.Errorf("expected error to mention 'Children[0]', got: %s", errStr)
	}
}

func TestReport_TreeData_Item_Validate_DeeplyNested(t *testing.T) {
	// 3 levels deep - error at grandchild level
	item := Report_TreeData_Item{
		Value: "root",
		Children: []Report_TreeData_Item{
			{
				Value: "child",
				Children: []Report_TreeData_Item{
					{Value: ""}, // invalid grandchild
				},
			},
		},
	}

	err := item.Validate()
	if err == nil {
		t.Fatal("expected validation error for deeply nested empty value")
	}

	errStr := err.Error()
	// Should show full path to the nested error
	if !contains(errStr, "Children[0]") {
		t.Errorf("expected error to mention 'Children[0]', got: %s", errStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestReport_ReportData_Item_Validate_NoRecursion(t *testing.T) {
	// Create a recursive component structure
	item := Report_ReportData_Item{
		Name: ptr("root"),
		Components: []Report_ReportData_Item{
			{
				Name: ptr("component1"),
				Components: []Report_ReportData_Item{
					{Name: ptr("nested")},
				},
			},
		},
	}

	// Should not cause infinite recursion
	err := item.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func ptr(s string) *string {
	return &s
}

