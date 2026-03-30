package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

func newTestInvoiceService(t *testing.T, handler http.HandlerFunc) (*api.InvoiceService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := api.NewWithToken("test-token", "1.0.0", false)
	svc := api.NewInvoiceService(client, srv.URL+"/api/v3")
	return svc, srv
}

func TestInvoiceService_GetOffice(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/office" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Office{
			Name:    "Test Office",
			Zip:     "100-0001",
			Address1: "Tokyo",
		})
	})

	office, err := svc.GetOffice()
	if err != nil {
		t.Fatalf("GetOffice() error: %v", err)
	}
	if office.Name != "Test Office" {
		t.Errorf("office.Name = %q, want %q", office.Name, "Test Office")
	}
	if office.Zip != "100-0001" {
		t.Errorf("office.Zip = %q, want %q", office.Zip, "100-0001")
	}
}

func TestInvoiceService_ListPartners(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" {
			t.Errorf("page = %q, want %q", q.Get("page"), "2")
		}
		if q.Get("per_page") != "10" {
			t.Errorf("per_page = %q, want %q", q.Get("per_page"), "10")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.Partner{
				{ID: "p1", Name: "Partner 1", CreatedAt: "2024-01-01", UpdatedAt: "2024-01-01"},
				{ID: "p2", Name: "Partner 2", CreatedAt: "2024-01-02", UpdatedAt: "2024-01-02"},
			},
			"pagination": map[string]int{
				"total_count":  2,
				"total_pages":  1,
				"current_page": 2,
				"per_page":     10,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	params := pagination.Params{Page: 2, PerPage: 10}
	partners, pag, err := svc.ListPartners(params, "")
	if err != nil {
		t.Fatalf("ListPartners() error: %v", err)
	}
	if len(partners) != 2 {
		t.Errorf("len(partners) = %d, want 2", len(partners))
	}
	if partners[0].ID != "p1" {
		t.Errorf("partners[0].ID = %q, want %q", partners[0].ID, "p1")
	}
	if pag.TotalCount != 2 {
		t.Errorf("pag.TotalCount = %d, want 2", pag.TotalCount)
	}
	if pag.TotalPages != 1 {
		t.Errorf("pag.TotalPages = %d, want 1", pag.TotalPages)
	}
	if pag.CurrentPage != 2 {
		t.Errorf("pag.CurrentPage = %d, want 2", pag.CurrentPage)
	}
	if pag.PerPage != 10 {
		t.Errorf("pag.PerPage = %d, want 10", pag.PerPage)
	}
}

func TestInvoiceService_ListPartners_WithQuery(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("q") != "acme" {
			t.Errorf("q = %q, want %q", q.Get("q"), "acme")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.Partner{
				{ID: "p1", Name: "Acme Corp", CreatedAt: "2024-01-01", UpdatedAt: "2024-01-01"},
			},
			"pagination": map[string]int{
				"total_count":  1,
				"total_pages":  1,
				"current_page": 1,
				"per_page":     25,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	params := pagination.Params{Page: 1, PerPage: 25}
	partners, _, err := svc.ListPartners(params, "acme")
	if err != nil {
		t.Fatalf("ListPartners() with query error: %v", err)
	}
	if len(partners) != 1 {
		t.Errorf("len(partners) = %d, want 1", len(partners))
	}
	if partners[0].Name != "Acme Corp" {
		t.Errorf("partners[0].Name = %q, want %q", partners[0].Name, "Acme Corp")
	}
}

func TestInvoiceService_GetPartner(t *testing.T) {
	dept := model.PartnerDepartment{
		ID:   "d1",
		Name: "Sales",
		Tel:  "03-1234-5678",
	}
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners/p1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Partner{
			ID:          "p1",
			Name:        "Test Partner",
			Departments: []model.PartnerDepartment{dept},
			CreatedAt:   "2024-01-01",
			UpdatedAt:   "2024-01-01",
		})
	})

	partner, err := svc.GetPartner("p1")
	if err != nil {
		t.Fatalf("GetPartner() error: %v", err)
	}
	if partner.ID != "p1" {
		t.Errorf("partner.ID = %q, want %q", partner.ID, "p1")
	}
	if partner.Name != "Test Partner" {
		t.Errorf("partner.Name = %q, want %q", partner.Name, "Test Partner")
	}
	if len(partner.Departments) != 1 {
		t.Fatalf("len(partner.Departments) = %d, want 1", len(partner.Departments))
	}
	if partner.Departments[0].ID != "d1" {
		t.Errorf("department.ID = %q, want %q", partner.Departments[0].ID, "d1")
	}
	if partner.Departments[0].Name != "Sales" {
		t.Errorf("department.Name = %q, want %q", partner.Departments[0].Name, "Sales")
	}
}

func TestInvoiceService_CreatePartner(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body model.CreatePartnerParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Name != "New Partner" {
			t.Errorf("body.Name = %q, want %q", body.Name, "New Partner")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model.Partner{
			ID:        "p-new",
			Name:      body.Name,
			CreatedAt: "2024-06-01",
			UpdatedAt: "2024-06-01",
		})
	})

	params := model.CreatePartnerParams{Name: "New Partner"}
	partner, err := svc.CreatePartner(params)
	if err != nil {
		t.Fatalf("CreatePartner() error: %v", err)
	}
	if partner.ID != "p-new" {
		t.Errorf("partner.ID = %q, want %q", partner.ID, "p-new")
	}
	if partner.Name != "New Partner" {
		t.Errorf("partner.Name = %q, want %q", partner.Name, "New Partner")
	}
}

func TestInvoiceService_UpdatePartner(t *testing.T) {
	newName := "Updated Partner"
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners/p1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body model.UpdatePartnerParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Name == nil || *body.Name != "Updated Partner" {
			t.Errorf("body.Name = %v, want pointer to %q", body.Name, "Updated Partner")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Partner{
			ID:        "p1",
			Name:      *body.Name,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-06-01",
		})
	})

	params := model.UpdatePartnerParams{Name: &newName}
	partner, err := svc.UpdatePartner("p1", params)
	if err != nil {
		t.Fatalf("UpdatePartner() error: %v", err)
	}
	if partner.Name != "Updated Partner" {
		t.Errorf("partner.Name = %q, want %q", partner.Name, "Updated Partner")
	}
}

func TestInvoiceService_DeletePartner(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners/p1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := svc.DeletePartner("p1")
	if err != nil {
		t.Fatalf("DeletePartner() error: %v", err)
	}
}

func TestInvoiceService_ListPartnerDepartments(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/partners/p1/departments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.PartnerDepartment{
				{ID: "d1", Name: "Sales", Tel: "03-1111-2222"},
				{ID: "d2", Name: "Engineering"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	depts, err := svc.ListPartnerDepartments("p1")
	if err != nil {
		t.Fatalf("ListPartnerDepartments() error: %v", err)
	}
	if len(depts) != 2 {
		t.Fatalf("len(depts) = %d, want 2", len(depts))
	}
	if depts[0].ID != "d1" {
		t.Errorf("depts[0].ID = %q, want %q", depts[0].ID, "d1")
	}
	if depts[0].Name != "Sales" {
		t.Errorf("depts[0].Name = %q, want %q", depts[0].Name, "Sales")
	}
	if depts[1].ID != "d2" {
		t.Errorf("depts[1].ID = %q, want %q", depts[1].ID, "d2")
	}
}
