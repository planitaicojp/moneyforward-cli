package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
)

func newTestExpenseService(t *testing.T, handler http.HandlerFunc) *api.ExpenseService {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := api.NewWithToken("test-token", "1.0.0", false)
	return api.NewExpenseService(client, srv.URL+"/api/external/v1", srv.URL+"/api/external/v2")
}

func TestExpenseService_ListOffices(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"offices": []model.ExpenseOffice{
				{ID: "o1", Name: "Test Office", OfficeTypeID: 2},
			},
			"next": nil,
			"prev": nil,
		})
	})

	offices, hasNext, err := svc.ListOffices(1)
	if err != nil {
		t.Fatalf("ListOffices() error: %v", err)
	}
	if len(offices) != 1 || offices[0].ID != "o1" {
		t.Errorf("unexpected offices: %+v", offices)
	}
	if hasNext {
		t.Error("hasNext = true, want false")
	}
}

func TestExpenseService_ListOffices_Pagination(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		if page == "" || page == "1" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"offices": []model.ExpenseOffice{{ID: "o1", Name: "Office 1"}},
				"next":    "/api/external/v1/offices?page=2",
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"offices": []model.ExpenseOffice{{ID: "o2", Name: "Office 2"}},
				"next":    nil,
			})
		}
	})

	_, hasNext1, err := svc.ListOffices(1)
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if !hasNext1 {
		t.Error("page 1: hasNext = false, want true")
	}

	_, hasNext2, err := svc.ListOffices(2)
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if hasNext2 {
		t.Error("page 2: hasNext = true, want false")
	}
}

func TestExpenseService_ListDepts(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/off1/depts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if kw := r.URL.Query().Get("search_keyword"); kw != "sales" {
			t.Errorf("unexpected keyword: %s", kw)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"depts": []model.Dept{
				{ID: "d1", Name: "Sales", IsActive: true},
			},
			"next": nil,
		})
	})

	depts, hasNext, err := svc.ListDepts("off1", 1, "sales")
	if err != nil {
		t.Fatalf("ListDepts() error: %v", err)
	}
	if len(depts) != 1 || depts[0].ID != "d1" {
		t.Errorf("unexpected depts: %+v", depts)
	}
	if hasNext {
		t.Error("hasNext = true, want false")
	}
}

func TestExpenseService_GetDept(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/off1/depts/d1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(model.Dept{ID: "d1", Name: "Sales", IsActive: true})
	})

	dept, err := svc.GetDept("off1", "d1")
	if err != nil {
		t.Fatalf("GetDept() error: %v", err)
	}
	if dept.ID != "d1" || dept.Name != "Sales" {
		t.Errorf("unexpected dept: %+v", dept)
	}
}

func TestExpenseService_ListExItems(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/off1/ex_items" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ex_items": []model.ExItem{
				{ID: "ei1", Name: "Travel", IsActive: true},
			},
			"next": nil,
		})
	})

	items, hasNext, err := svc.ListExItems("off1", 1, "")
	if err != nil {
		t.Fatalf("ListExItems() error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "ei1" {
		t.Errorf("unexpected ex_items: %+v", items)
	}
	if hasNext {
		t.Error("hasNext = true, want false")
	}
}

func TestExpenseService_ListExcises(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/off1/excises" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"excises": []model.ExpenseExcise{
				{ID: "ex1", LongName: "10%", Rate: 0.1},
			},
			"next": nil,
		})
	})

	excises, hasNext, err := svc.ListExcises("off1", 1)
	if err != nil {
		t.Fatalf("ListExcises() error: %v", err)
	}
	if len(excises) != 1 || excises[0].ID != "ex1" {
		t.Errorf("unexpected excises: %+v", excises)
	}
	if hasNext {
		t.Error("hasNext = true, want false")
	}
}

func TestExpenseService_ListPositions(t *testing.T) {
	svc := newTestExpenseService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/v1/offices/off1/positions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"positions": []model.Position{
				{ID: "p1", Name: "Manager", IsAuthorizer: true, Priority: 1},
			},
			"next": nil,
		})
	})

	positions, hasNext, err := svc.ListPositions("off1", 1)
	if err != nil {
		t.Fatalf("ListPositions() error: %v", err)
	}
	if len(positions) != 1 || positions[0].ID != "p1" {
		t.Errorf("unexpected positions: %+v", positions)
	}
	if hasNext {
		t.Error("hasNext = true, want false")
	}
}
