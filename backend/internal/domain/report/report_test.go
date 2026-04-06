package report

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// IsValidStatus
// ---------------------------------------------------------------------------

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"new is valid", "new", true},
		{"in_review is valid", "in_review", true},
		{"resolved is valid", "resolved", true},
		{"rejected is valid", "rejected", true},
		{"empty string is invalid", "", false},
		{"unknown is invalid", "unknown", false},
		{"uppercase NEW is invalid", "NEW", false},
		{"mixed case New is invalid", "New", false},
		{"extra whitespace is invalid", " new ", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidStatus(tc.input)
			if got != tc.expect {
				t.Errorf("IsValidStatus(%q) = %v, want %v", tc.input, got, tc.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsValidPriority
// ---------------------------------------------------------------------------

func TestIsValidPriority(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"Высокий is valid", "Высокий", true},
		{"Средний is valid", "Средний", true},
		{"Низкий is valid", "Низкий", true},
		{"Не задан is valid", "Не задан", true},
		{"empty string is invalid", "", false},
		{"unknown is invalid", "unknown", false},
		{"lowercase высокий is invalid", "высокий", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidPriority(tc.input)
			if got != tc.expect {
				t.Errorf("IsValidPriority(%q) = %v, want %v", tc.input, got, tc.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsValidInfluence
// ---------------------------------------------------------------------------

func TestIsValidInfluence(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"Крит/блокер lowercase б is valid", "Крит/блокер", true},
		{"Крит/Блокер uppercase Б is valid", "Крит/Блокер", true},
		{"Высокий is valid", "Высокий", true},
		{"Средний is valid", "Средний", true},
		{"Низкий is valid", "Низкий", true},
		{"Не баг а фича is valid", "Не баг а фича", true},
		{"Не задано is valid", "Не задано", true},
		{"empty string is invalid", "", false},
		{"unknown is invalid", "unknown", false},
		{"wrong case крит/блокер is invalid", "крит/блокер", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidInfluence(tc.input)
			if got != tc.expect {
				t.Errorf("IsValidInfluence(%q) = %v, want %v", tc.input, got, tc.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ApplyListFilter
// ---------------------------------------------------------------------------

// helper to create a *string
func strPtr(s string) *string { return &s }

// helper to create a *time.Time
func timePtr(t time.Time) *time.Time { return &t }

// base time for deterministic tests
var baseTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func makeReports() []Report {
	return []Report{
		{
			ID: "1", ReporterName: "Alice", Description: "login page crash",
			Status: "new", Influence: "Высокий", Priority: "Высокий",
			CreatedAt: baseTime, UpdatedAt: baseTime.Add(5 * time.Hour),
		},
		{
			ID: "2", ReporterName: "Bob", Description: "dashboard slow",
			Status: "in_review", Influence: "Средний", Priority: "Средний",
			CreatedAt: baseTime.Add(1 * time.Hour), UpdatedAt: baseTime.Add(3 * time.Hour),
		},
		{
			ID: "3", ReporterName: "Alice", Description: "signup error",
			Status: "new", Influence: "Низкий", Priority: "Низкий",
			CreatedAt: baseTime.Add(2 * time.Hour), UpdatedAt: baseTime.Add(4 * time.Hour),
		},
		{
			ID: "4", ReporterName: "Charlie", Description: "Payment timeout",
			Status: "resolved", Influence: "Крит/блокер", Priority: "Высокий",
			CreatedAt: baseTime.Add(3 * time.Hour), UpdatedAt: baseTime.Add(1 * time.Hour),
		},
		{
			ID: "5", ReporterName: "Diana", Description: "Minor typo",
			Status: "rejected", Influence: "Не баг а фича", Priority: "Не задан",
			CreatedAt: baseTime.Add(4 * time.Hour), UpdatedAt: baseTime.Add(2 * time.Hour),
		},
	}
}

func TestApplyListFilter_EmptyList(t *testing.T) {
	result, total, err := ApplyListFilter(nil, ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestApplyListFilter_FilterByStatus(t *testing.T) {
	reports := makeReports()
	result, total, err := ApplyListFilter(reports, ListFilter{
		Status: strPtr("new"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2", len(result))
	}
	for _, r := range result {
		if r.Status != "new" {
			t.Errorf("got status %q, want %q", r.Status, "new")
		}
	}
}

func TestApplyListFilter_FilterByReporterName(t *testing.T) {
	reports := makeReports()
	result, total, err := ApplyListFilter(reports, ListFilter{
		ReporterName: strPtr("Alice"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	for _, r := range result {
		if r.ReporterName != "Alice" {
			t.Errorf("got reporter %q, want %q", r.ReporterName, "Alice")
		}
	}
}

func TestApplyListFilter_FilterByReporterName_NoMatch(t *testing.T) {
	reports := makeReports()
	result, total, err := ApplyListFilter(reports, ListFilter{
		ReporterName: strPtr("NoSuchUser"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestApplyListFilter_FilterByQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantIDs   []string
		wantTotal int
	}{
		{
			name:      "substring match in description",
			query:     "crash",
			wantIDs:   []string{"1"},
			wantTotal: 1,
		},
		{
			name:      "case insensitive match",
			query:     "CRASH",
			wantIDs:   []string{"1"},
			wantTotal: 1,
		},
		{
			name:      "match in reporter name",
			query:     "charlie",
			wantIDs:   []string{"4"},
			wantTotal: 1,
		},
		{
			name:      "partial match across fields",
			query:     "alice",
			wantIDs:   []string{"1", "3"},
			wantTotal: 2,
		},
		{
			name:      "empty query returns all",
			query:     "   ",
			wantTotal: 5,
		},
		{
			name:      "no match",
			query:     "zzzzzzz",
			wantTotal: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, total, err := ApplyListFilter(makeReports(), ListFilter{
				Query: strPtr(tc.query),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tc.wantTotal {
				t.Errorf("total = %d, want %d", total, tc.wantTotal)
			}
			if tc.wantIDs != nil {
				if len(result) != len(tc.wantIDs) {
					t.Fatalf("len(result) = %d, want %d", len(result), len(tc.wantIDs))
				}
				for i, id := range tc.wantIDs {
					if result[i].ID != id {
						t.Errorf("result[%d].ID = %q, want %q", i, result[i].ID, id)
					}
				}
			}
		})
	}
}

func TestApplyListFilter_FilterByCreatedAtRange(t *testing.T) {
	reports := makeReports()

	from := baseTime.Add(1 * time.Hour)
	to := baseTime.Add(3 * time.Hour)

	t.Run("from and to", func(t *testing.T) {
		result, total, err := ApplyListFilter(reports, ListFilter{
			CreatedAt: &TimeRange{From: timePtr(from), To: timePtr(to)},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// IDs 2 (1h), 3 (2h), 4 (3h) match
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
		if len(result) != 3 {
			t.Errorf("len(result) = %d, want 3", len(result))
		}
	})

	t.Run("only from", func(t *testing.T) {
		result, total, err := ApplyListFilter(reports, ListFilter{
			CreatedAt: &TimeRange{From: timePtr(baseTime.Add(3 * time.Hour))},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// IDs 4 (3h) and 5 (4h)
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
		_ = result
	})

	t.Run("only to", func(t *testing.T) {
		_, total, err := ApplyListFilter(reports, ListFilter{
			CreatedAt: &TimeRange{To: timePtr(baseTime.Add(1 * time.Hour))},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// IDs 1 (0h) and 2 (1h)
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
	})
}

func TestApplyListFilter_SortByCreatedAtASC(t *testing.T) {
	reports := makeReports()
	result, _, err := ApplyListFilter(reports, ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(result); i++ {
		if result[i].CreatedAt.Before(result[i-1].CreatedAt) {
			t.Errorf("result[%d].CreatedAt (%v) < result[%d].CreatedAt (%v): not sorted ASC",
				i, result[i].CreatedAt, i-1, result[i-1].CreatedAt)
		}
	}
}

func TestApplyListFilter_SortByUpdatedAtDESC(t *testing.T) {
	reports := makeReports()
	result, _, err := ApplyListFilter(reports, ListFilter{
		SortBy:   "updated_at",
		SortDesc: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(result); i++ {
		if result[i].UpdatedAt.After(result[i-1].UpdatedAt) {
			t.Errorf("result[%d].UpdatedAt (%v) > result[%d].UpdatedAt (%v): not sorted DESC",
				i, result[i].UpdatedAt, i-1, result[i-1].UpdatedAt)
		}
	}
}

func TestApplyListFilter_SortByCreatedAtDESC(t *testing.T) {
	reports := makeReports()
	result, _, err := ApplyListFilter(reports, ListFilter{
		SortBy:   "created_at",
		SortDesc: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(result); i++ {
		if result[i].CreatedAt.After(result[i-1].CreatedAt) {
			t.Errorf("result[%d].CreatedAt (%v) > result[%d].CreatedAt (%v): not sorted DESC",
				i, result[i].CreatedAt, i-1, result[i-1].CreatedAt)
		}
	}
}

func TestApplyListFilter_Pagination(t *testing.T) {
	reports := makeReports() // 5 items

	t.Run("limit 2 offset 0", func(t *testing.T) {
		result, total, err := ApplyListFilter(reports, ListFilter{
			Limit:  2,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d, want 5", total)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
	})

	t.Run("limit 2 offset 3", func(t *testing.T) {
		result, total, err := ApplyListFilter(reports, ListFilter{
			Limit:  2,
			Offset: 3,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d, want 5", total)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
	})

	t.Run("limit 2 offset 4 returns last item", func(t *testing.T) {
		result, total, err := ApplyListFilter(reports, ListFilter{
			Limit:  2,
			Offset: 4,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d, want 5", total)
		}
		if len(result) != 1 {
			t.Errorf("len(result) = %d, want 1", len(result))
		}
	})
}

func TestApplyListFilter_OffsetBeyondLength(t *testing.T) {
	reports := makeReports()
	result, total, err := ApplyListFilter(reports, ListFilter{
		Limit:  10,
		Offset: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestApplyListFilter_DefaultLimitAndNegativeOffset(t *testing.T) {
	reports := makeReports()

	t.Run("zero limit defaults to 20", func(t *testing.T) {
		result, total, err := ApplyListFilter(reports, ListFilter{
			Limit: 0,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d, want 5", total)
		}
		// all 5 returned since 5 < default 20
		if len(result) != 5 {
			t.Errorf("len(result) = %d, want 5", len(result))
		}
	})

	t.Run("negative offset treated as 0", func(t *testing.T) {
		result, _, err := ApplyListFilter(reports, ListFilter{
			Limit:  2,
			Offset: -5,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
		// Should start from the beginning (same as offset 0)
		if result[0].ID != "1" {
			t.Errorf("result[0].ID = %q, want %q", result[0].ID, "1")
		}
	})

	t.Run("limit over 100 defaults to 20", func(t *testing.T) {
		result, _, err := ApplyListFilter(reports, ListFilter{
			Limit: 200,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// all 5 returned since 5 < clamped 20
		if len(result) != 5 {
			t.Errorf("len(result) = %d, want 5", len(result))
		}
	})
}

func TestApplyListFilter_CombinedFilters(t *testing.T) {
	reports := makeReports()

	// Filter: status=new, reporter=Alice, sort by created_at DESC, limit 1
	result, total, err := ApplyListFilter(reports, ListFilter{
		Status:       strPtr("new"),
		ReporterName: strPtr("Alice"),
		SortBy:       "created_at",
		SortDesc:     true,
		Limit:        1,
		Offset:       0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Alice has 2 reports with status "new": ID 1 and 3
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	// DESC by created_at → ID 3 first (created at baseTime+2h)
	if result[0].ID != "3" {
		t.Errorf("result[0].ID = %q, want %q", result[0].ID, "3")
	}
}

func TestApplyListFilter_CombinedFiltersWithQueryAndDateRange(t *testing.T) {
	reports := makeReports()

	from := baseTime
	to := baseTime.Add(2 * time.Hour)

	result, total, err := ApplyListFilter(reports, ListFilter{
		Query:     strPtr("alice"),
		CreatedAt: &TimeRange{From: timePtr(from), To: timePtr(to)},
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Alice's reports: ID 1 (0h) and ID 3 (2h) — both in [0h, 2h]
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2", len(result))
	}
}

func TestApplyListFilter_PaginationPreservesOrder(t *testing.T) {
	reports := makeReports()

	// Get page 1
	page1, _, err := ApplyListFilter(reports, ListFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Get page 2
	page2, _, err := ApplyListFilter(reports, ListFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Get page 3
	page3, _, err := ApplyListFilter(reports, ListFilter{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := append(append(page1, page2...), page3...)
	if len(all) != 5 {
		t.Fatalf("total across pages = %d, want 5", len(all))
	}

	// Verify ASC order by created_at across pages
	for i := 1; i < len(all); i++ {
		if all[i].CreatedAt.Before(all[i-1].CreatedAt) {
			t.Errorf("page boundary break: item %d (%v) before item %d (%v)",
				i, all[i].CreatedAt, i-1, all[i-1].CreatedAt)
		}
	}
}
