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

func TestInvoiceService_ListQuotes(t *testing.T) {
	subtotal, total := 3000, 3300
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page = %q, want %q", q.Get("page"), "1")
		}
		if q.Get("per_page") != "25" {
			t.Errorf("per_page = %q, want %q", q.Get("per_page"), "25")
		}
		if q.Get("partner_id") != "p1" {
			t.Errorf("partner_id = %q, want %q", q.Get("partner_id"), "p1")
		}
		if q.Get("status") != "draft" {
			t.Errorf("status = %q, want %q", q.Get("status"), "draft")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.Quote{
				{ID: "q1", Title: "Quote 1", Status: model.QuoteStatusDraft, Subtotal: &subtotal, TotalPrice: &total, CreatedAt: "2024-01-01", UpdatedAt: "2024-01-01"},
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

	opts := api.QuoteListOptions{
		Params:    pagination.Params{Page: 1, PerPage: 25},
		PartnerID: "p1",
		Status:    "draft",
	}
	quotes, pag, err := svc.ListQuotes(opts)
	if err != nil {
		t.Fatalf("ListQuotes() error: %v", err)
	}
	if len(quotes) != 1 {
		t.Errorf("len(quotes) = %d, want 1", len(quotes))
	}
	if quotes[0].ID != "q1" {
		t.Errorf("quotes[0].ID = %q, want %q", quotes[0].ID, "q1")
	}
	if quotes[0].Status != model.QuoteStatusDraft {
		t.Errorf("quotes[0].Status = %q, want %q", quotes[0].Status, model.QuoteStatusDraft)
	}
	if pag.TotalCount != 1 {
		t.Errorf("pag.TotalCount = %d, want 1", pag.TotalCount)
	}
}

func TestInvoiceService_GetQuote(t *testing.T) {
	subtotal, total := 5000, 5500
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:          "q1",
			Title:       "Test Quote",
			Status:      model.QuoteStatusSent,
			PDFURL:      "https://example.com/q1.pdf",
			QuoteDate:   "2024-06-01",
			ExpiredDate: "2024-07-01",
			Subtotal:    &subtotal,
			TotalPrice:  &total,
			CreatedAt:   "2024-01-01",
			UpdatedAt:   "2024-01-01",
		})
	})

	quote, err := svc.GetQuote("q1")
	if err != nil {
		t.Fatalf("GetQuote() error: %v", err)
	}
	if quote.ID != "q1" {
		t.Errorf("quote.ID = %q, want %q", quote.ID, "q1")
	}
	if quote.Title != "Test Quote" {
		t.Errorf("quote.Title = %q, want %q", quote.Title, "Test Quote")
	}
	if quote.PDFURL != "https://example.com/q1.pdf" {
		t.Errorf("quote.PDFURL = %q, want %q", quote.PDFURL, "https://example.com/q1.pdf")
	}
	if quote.Status != model.QuoteStatusSent {
		t.Errorf("quote.Status = %q, want %q", quote.Status, model.QuoteStatusSent)
	}
}

func TestInvoiceService_CreateQuote(t *testing.T) {
	subtotal, total := 10000, 11000
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body model.CreateQuoteParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.DepartmentID != "d1" {
			t.Errorf("body.DepartmentID = %q, want %q", body.DepartmentID, "d1")
		}
		if body.QuoteDate != "2024-06-01" {
			t.Errorf("body.QuoteDate = %q, want %q", body.QuoteDate, "2024-06-01")
		}
		if body.ExpiredDate != "2024-07-01" {
			t.Errorf("body.ExpiredDate = %q, want %q", body.ExpiredDate, "2024-07-01")
		}
		if len(body.Items) != 1 {
			t.Fatalf("len(body.Items) = %d, want 1", len(body.Items))
		}
		if body.Items[0].Name != "Consulting" {
			t.Errorf("body.Items[0].Name = %q, want %q", body.Items[0].Name, "Consulting")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:           "q-new",
			DepartmentID: body.DepartmentID,
			QuoteDate:    body.QuoteDate,
			ExpiredDate:  body.ExpiredDate,
			Status:       model.QuoteStatusDraft,
			Subtotal:     &subtotal,
			TotalPrice:   &total,
			CreatedAt:    "2024-06-01",
			UpdatedAt:    "2024-06-01",
		})
	})

	params := model.CreateQuoteParams{
		DepartmentID: "d1",
		QuoteDate:    "2024-06-01",
		ExpiredDate:  "2024-07-01",
		Items: []model.InvoiceTemplateLine{
			{Name: "Consulting", Price: 10000, Quantity: 1, Excise: "ten_percent"},
		},
	}
	quote, err := svc.CreateQuote(params)
	if err != nil {
		t.Fatalf("CreateQuote() error: %v", err)
	}
	if quote.ID != "q-new" {
		t.Errorf("quote.ID = %q, want %q", quote.ID, "q-new")
	}
	if quote.Status != model.QuoteStatusDraft {
		t.Errorf("quote.Status = %q, want %q", quote.Status, model.QuoteStatusDraft)
	}
}

func TestInvoiceService_UpdateQuote(t *testing.T) {
	newTitle := "Updated Quote"
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body model.UpdateQuoteParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Title == nil || *body.Title != "Updated Quote" {
			t.Errorf("body.Title = %v, want pointer to %q", body.Title, "Updated Quote")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:        "q1",
			Title:     *body.Title,
			Status:    model.QuoteStatusDraft,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-06-01",
		})
	})

	params := model.UpdateQuoteParams{Title: &newTitle}
	quote, err := svc.UpdateQuote("q1", params)
	if err != nil {
		t.Fatalf("UpdateQuote() error: %v", err)
	}
	if quote.Title != "Updated Quote" {
		t.Errorf("quote.Title = %q, want %q", quote.Title, "Updated Quote")
	}
}

func TestInvoiceService_DeleteQuote(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := svc.DeleteQuote("q1")
	if err != nil {
		t.Fatalf("DeleteQuote() error: %v", err)
	}
}

func TestInvoiceService_SetQuoteStatus(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body["status"] != "sent" {
			t.Errorf("body[status] = %v, want %q", body["status"], "sent")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:        "q1",
			Status:    model.QuoteStatusSent,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-06-01",
		})
	})

	quote, err := svc.SetQuoteStatus("q1", model.QuoteStatusSent)
	if err != nil {
		t.Fatalf("SetQuoteStatus() error: %v", err)
	}
	if quote.Status != model.QuoteStatusSent {
		t.Errorf("quote.Status = %q, want %q", quote.Status, model.QuoteStatusSent)
	}
}

func TestInvoiceService_ConvertQuoteToBilling(t *testing.T) {
	subtotal, total := 10000, 11000
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1/convert_to_billing" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("body should be empty, got %v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b-converted",
			PaymentStatus: model.PaymentStatusUnsettled,
			Subtotal:      &subtotal,
			TotalPrice:    &total,
			CreatedAt:     "2024-06-01",
			UpdatedAt:     "2024-06-01",
		})
	})

	billing, err := svc.ConvertQuoteToBilling("q1")
	if err != nil {
		t.Fatalf("ConvertQuoteToBilling() error: %v", err)
	}
	if billing.ID != "b-converted" {
		t.Errorf("billing.ID = %q, want %q", billing.ID, "b-converted")
	}
	if billing.PaymentStatus != model.PaymentStatusUnsettled {
		t.Errorf("billing.PaymentStatus = %q, want %q", billing.PaymentStatus, model.PaymentStatusUnsettled)
	}
}

func TestInvoiceService_GetQuotePDF(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/quotes/q1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Quote{
			ID:        "q1",
			PDFURL:    "https://example.com/q1.pdf",
			Status:    model.QuoteStatusDraft,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-01-01",
		})
	})

	pdfURL, err := svc.GetQuotePDF("q1")
	if err != nil {
		t.Fatalf("GetQuotePDF() error: %v", err)
	}
	if pdfURL != "https://example.com/q1.pdf" {
		t.Errorf("pdfURL = %q, want %q", pdfURL, "https://example.com/q1.pdf")
	}
}

func TestInvoiceService_ListSentHistories(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/sent_histories" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page = %q, want %q", q.Get("page"), "1")
		}
		if q.Get("per_page") != "25" {
			t.Errorf("per_page = %q, want %q", q.Get("per_page"), "25")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.SentHistory{
				{ID: "sh1", Type: "billing", DocumentID: "b1", Operator: "user@example.com", SentAt: "2024-06-01T10:00:00Z", CreatedAt: "2024-06-01", UpdatedAt: "2024-06-01"},
				{ID: "sh2", Type: "quote", DocumentID: "q1", Operator: "user@example.com", SentAt: "2024-06-02T10:00:00Z", CreatedAt: "2024-06-02", UpdatedAt: "2024-06-02"},
			},
			"pagination": map[string]int{
				"total_count":  2,
				"total_pages":  1,
				"current_page": 1,
				"per_page":     25,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	params := pagination.Params{Page: 1, PerPage: 25}
	histories, pag, err := svc.ListSentHistories(params)
	if err != nil {
		t.Fatalf("ListSentHistories() error: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("len(histories) = %d, want 2", len(histories))
	}
	if histories[0].ID != "sh1" {
		t.Errorf("histories[0].ID = %q, want %q", histories[0].ID, "sh1")
	}
	if histories[0].Type != "billing" {
		t.Errorf("histories[0].Type = %q, want %q", histories[0].Type, "billing")
	}
	if histories[1].DocumentID != "q1" {
		t.Errorf("histories[1].DocumentID = %q, want %q", histories[1].DocumentID, "q1")
	}
	if pag.TotalCount != 2 {
		t.Errorf("pag.TotalCount = %d, want 2", pag.TotalCount)
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

func TestInvoiceService_ListItems(t *testing.T) {
	price1, price2 := 1000, 2000
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/items" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page = %q, want %q", q.Get("page"), "1")
		}
		if q.Get("per_page") != "25" {
			t.Errorf("per_page = %q, want %q", q.Get("per_page"), "25")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.Item{
				{ID: "i1", Name: "Item 1", Price: &price1, CreatedAt: "2024-01-01", UpdatedAt: "2024-01-01"},
				{ID: "i2", Name: "Item 2", Price: &price2, CreatedAt: "2024-01-02", UpdatedAt: "2024-01-02"},
			},
			"pagination": map[string]int{
				"total_count":  2,
				"total_pages":  1,
				"current_page": 1,
				"per_page":     25,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	params := pagination.Params{Page: 1, PerPage: 25}
	items, pag, err := svc.ListItems(params, "")
	if err != nil {
		t.Fatalf("ListItems() error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "i1" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "i1")
	}
	if items[0].Name != "Item 1" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "Item 1")
	}
	if pag.TotalCount != 2 {
		t.Errorf("pag.TotalCount = %d, want 2", pag.TotalCount)
	}
	if pag.CurrentPage != 1 {
		t.Errorf("pag.CurrentPage = %d, want 1", pag.CurrentPage)
	}
}

func TestInvoiceService_ListItems_WithQuery(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("q") != "widget" {
			t.Errorf("q = %q, want %q", q.Get("q"), "widget")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		price := 500
		resp := map[string]interface{}{
			"data": []model.Item{
				{ID: "i1", Name: "Widget", Price: &price, CreatedAt: "2024-01-01", UpdatedAt: "2024-01-01"},
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
	items, _, err := svc.ListItems(params, "widget")
	if err != nil {
		t.Fatalf("ListItems() with query error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
	if items[0].Name != "Widget" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "Widget")
	}
}

func TestInvoiceService_GetItem(t *testing.T) {
	price := 1500
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/items/i1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Item{
			ID:        "i1",
			Name:      "Test Item",
			Unit:      "pcs",
			Price:     &price,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-01-01",
		})
	})

	item, err := svc.GetItem("i1")
	if err != nil {
		t.Fatalf("GetItem() error: %v", err)
	}
	if item.ID != "i1" {
		t.Errorf("item.ID = %q, want %q", item.ID, "i1")
	}
	if item.Name != "Test Item" {
		t.Errorf("item.Name = %q, want %q", item.Name, "Test Item")
	}
	if item.Unit != "pcs" {
		t.Errorf("item.Unit = %q, want %q", item.Unit, "pcs")
	}
	if item.Price == nil || *item.Price != 1500 {
		t.Errorf("item.Price = %v, want pointer to 1500", item.Price)
	}
}

func TestInvoiceService_CreateItem(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/items" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body model.CreateItemParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Name != "New Item" {
			t.Errorf("body.Name = %q, want %q", body.Name, "New Item")
		}
		if body.Unit != "kg" {
			t.Errorf("body.Unit = %q, want %q", body.Unit, "kg")
		}

		price := 3000
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model.Item{
			ID:        "i-new",
			Name:      body.Name,
			Unit:      body.Unit,
			Price:     &price,
			CreatedAt: "2024-06-01",
			UpdatedAt: "2024-06-01",
		})
	})

	price := 3000
	params := model.CreateItemParams{Name: "New Item", Unit: "kg", Price: &price}
	item, err := svc.CreateItem(params)
	if err != nil {
		t.Fatalf("CreateItem() error: %v", err)
	}
	if item.ID != "i-new" {
		t.Errorf("item.ID = %q, want %q", item.ID, "i-new")
	}
	if item.Name != "New Item" {
		t.Errorf("item.Name = %q, want %q", item.Name, "New Item")
	}
}

func TestInvoiceService_UpdateItem(t *testing.T) {
	newName := "Updated Item"
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/items/i1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body model.UpdateItemParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Name == nil || *body.Name != "Updated Item" {
			t.Errorf("body.Name = %v, want pointer to %q", body.Name, "Updated Item")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Item{
			ID:        "i1",
			Name:      *body.Name,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-06-01",
		})
	})

	params := model.UpdateItemParams{Name: &newName}
	item, err := svc.UpdateItem("i1", params)
	if err != nil {
		t.Fatalf("UpdateItem() error: %v", err)
	}
	if item.Name != "Updated Item" {
		t.Errorf("item.Name = %q, want %q", item.Name, "Updated Item")
	}
}

func TestInvoiceService_DeleteItem(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/items/i1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := svc.DeleteItem("i1")
	if err != nil {
		t.Fatalf("DeleteItem() error: %v", err)
	}
}

func TestInvoiceService_ListBillings(t *testing.T) {
	subtotal, total := 1000, 1100
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/billings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("page = %q, want %q", q.Get("page"), "1")
		}
		if q.Get("per_page") != "25" {
			t.Errorf("per_page = %q, want %q", q.Get("per_page"), "25")
		}
		if q.Get("partner_id") != "p1" {
			t.Errorf("partner_id = %q, want %q", q.Get("partner_id"), "p1")
		}
		if q.Get("payment_status") != "unsettled" {
			t.Errorf("payment_status = %q, want %q", q.Get("payment_status"), "unsettled")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []model.Billing{
				{ID: "b1", Title: "Billing 1", PaymentStatus: model.PaymentStatusUnsettled, Subtotal: &subtotal, TotalPrice: &total, CreatedAt: "2024-01-01", UpdatedAt: "2024-01-01"},
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

	opts := api.BillingListOptions{
		Params:        pagination.Params{Page: 1, PerPage: 25},
		PartnerID:     "p1",
		PaymentStatus: "unsettled",
	}
	billings, pag, err := svc.ListBillings(opts)
	if err != nil {
		t.Fatalf("ListBillings() error: %v", err)
	}
	if len(billings) != 1 {
		t.Errorf("len(billings) = %d, want 1", len(billings))
	}
	if billings[0].ID != "b1" {
		t.Errorf("billings[0].ID = %q, want %q", billings[0].ID, "b1")
	}
	if billings[0].PaymentStatus != model.PaymentStatusUnsettled {
		t.Errorf("billings[0].PaymentStatus = %q, want %q", billings[0].PaymentStatus, model.PaymentStatusUnsettled)
	}
	if pag.TotalCount != 1 {
		t.Errorf("pag.TotalCount = %d, want 1", pag.TotalCount)
	}
}

func TestInvoiceService_GetBilling(t *testing.T) {
	subtotal, total := 2000, 2200
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/billings/b1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b1",
			Title:         "Test Billing",
			PaymentStatus: model.PaymentStatusUnsettled,
			PDFURL:        "https://example.com/b1.pdf",
			Subtotal:      &subtotal,
			TotalPrice:    &total,
			CreatedAt:     "2024-01-01",
			UpdatedAt:     "2024-01-01",
		})
	})

	billing, err := svc.GetBilling("b1")
	if err != nil {
		t.Fatalf("GetBilling() error: %v", err)
	}
	if billing.ID != "b1" {
		t.Errorf("billing.ID = %q, want %q", billing.ID, "b1")
	}
	if billing.Title != "Test Billing" {
		t.Errorf("billing.Title = %q, want %q", billing.Title, "Test Billing")
	}
	if billing.PDFURL != "https://example.com/b1.pdf" {
		t.Errorf("billing.PDFURL = %q, want %q", billing.PDFURL, "https://example.com/b1.pdf")
	}
}

func TestInvoiceService_CreateBilling(t *testing.T) {
	subtotal, total := 5000, 5500
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		// Critical: must POST to /invoice_template_billings, NOT /billings
		if r.URL.Path != "/api/v3/invoice_template_billings" {
			t.Errorf("unexpected path: %s (want /api/v3/invoice_template_billings)", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		// Critical: body must be direct (not wrapped in {"billing": ...})
		var body model.CreateBillingParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.DepartmentID != "dept1" {
			t.Errorf("body.DepartmentID = %q, want %q", body.DepartmentID, "dept1")
		}
		if body.BillingDate != "2024-06-01" {
			t.Errorf("body.BillingDate = %q, want %q", body.BillingDate, "2024-06-01")
		}
		if body.Title != "Test Invoice" {
			t.Errorf("body.Title = %q, want %q", body.Title, "Test Invoice")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b-new",
			Title:         body.Title,
			PaymentStatus: model.PaymentStatusUnsettled,
			Subtotal:      &subtotal,
			TotalPrice:    &total,
			CreatedAt:     "2024-06-01",
			UpdatedAt:     "2024-06-01",
		})
	})

	params := model.CreateBillingParams{
		DepartmentID: "dept1",
		BillingDate:  "2024-06-01",
		Title:        "Test Invoice",
	}
	billing, err := svc.CreateBilling(params)
	if err != nil {
		t.Fatalf("CreateBilling() error: %v", err)
	}
	if billing.ID != "b-new" {
		t.Errorf("billing.ID = %q, want %q", billing.ID, "b-new")
	}
	if billing.Title != "Test Invoice" {
		t.Errorf("billing.Title = %q, want %q", billing.Title, "Test Invoice")
	}
}

func TestInvoiceService_UpdateBilling(t *testing.T) {
	newTitle := "Updated Billing"
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/billings/b1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Critical: body must be wrapped as {"billing": {...}}
		var rawBody map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		inner, ok := rawBody["billing"]
		if !ok {
			t.Errorf("expected body to have 'billing' key, got: %v", rawBody)
		}
		var params model.UpdateBillingParams
		if err := json.Unmarshal(inner, &params); err != nil {
			t.Errorf("unmarshaling billing inner: %v", err)
		}
		if params.Title == nil || *params.Title != "Updated Billing" {
			t.Errorf("params.Title = %v, want pointer to %q", params.Title, "Updated Billing")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b1",
			Title:         *params.Title,
			PaymentStatus: model.PaymentStatusUnsettled,
			CreatedAt:     "2024-01-01",
			UpdatedAt:     "2024-06-01",
		})
	})

	params := model.UpdateBillingParams{Title: &newTitle}
	billing, err := svc.UpdateBilling("b1", params)
	if err != nil {
		t.Fatalf("UpdateBilling() error: %v", err)
	}
	if billing.Title != "Updated Billing" {
		t.Errorf("billing.Title = %q, want %q", billing.Title, "Updated Billing")
	}
}

func TestInvoiceService_UpdateBilling_WithItems(t *testing.T) {
	newTitle := "With Items"
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify wrapping includes items
		var rawBody map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		inner, ok := rawBody["billing"]
		if !ok {
			t.Fatal("expected body to have 'billing' key")
		}
		var parsed struct {
			Title *string                  `json:"title"`
			Items []model.InvoiceTemplateLine `json:"items"`
		}
		if err := json.Unmarshal(inner, &parsed); err != nil {
			t.Fatalf("unmarshaling billing inner: %v", err)
		}
		if parsed.Title == nil || *parsed.Title != "With Items" {
			t.Errorf("Title = %v, want %q", parsed.Title, "With Items")
		}
		if len(parsed.Items) != 1 {
			t.Fatalf("Items len = %d, want 1", len(parsed.Items))
		}
		if parsed.Items[0].Name != "Line 1" {
			t.Errorf("Items[0].Name = %q, want %q", parsed.Items[0].Name, "Line 1")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b1",
			Title:         *parsed.Title,
			PaymentStatus: model.PaymentStatusUnsettled,
			CreatedAt:     "2024-01-01",
			UpdatedAt:     "2024-06-01",
		})
	})

	params := model.UpdateBillingParams{
		Title: &newTitle,
		Items: []model.InvoiceTemplateLine{
			{Name: "Line 1", Price: 1000, Quantity: 1, Excise: "ten_percent"},
		},
	}
	billing, err := svc.UpdateBilling("b1", params)
	if err != nil {
		t.Fatalf("UpdateBilling() with items error: %v", err)
	}
	if billing.Title != "With Items" {
		t.Errorf("billing.Title = %q, want %q", billing.Title, "With Items")
	}
}

func TestInvoiceService_DeleteBilling(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/billings/b1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := svc.DeleteBilling("b1")
	if err != nil {
		t.Fatalf("DeleteBilling() error: %v", err)
	}
}

func TestInvoiceService_SetPaymentStatus(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/billings/b1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Critical: body must be wrapped as {"billing": {"payment_status": "settled"}}
		var rawBody map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		inner, ok := rawBody["billing"]
		if !ok {
			t.Errorf("expected body to have 'billing' key, got: %v", rawBody)
		}
		var innerMap map[string]json.RawMessage
		if err := json.Unmarshal(inner, &innerMap); err != nil {
			t.Errorf("unmarshaling billing inner: %v", err)
		}
		statusRaw, ok := innerMap["payment_status"]
		if !ok {
			t.Errorf("expected inner to have 'payment_status' key")
		}
		var status string
		if err := json.Unmarshal(statusRaw, &status); err != nil {
			t.Errorf("unmarshaling payment_status: %v", err)
		}
		if status != "settled" {
			t.Errorf("payment_status = %q, want %q", status, "settled")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b1",
			Title:         "Some Billing",
			PaymentStatus: model.PaymentStatusSettled,
			CreatedAt:     "2024-01-01",
			UpdatedAt:     "2024-06-01",
		})
	})

	billing, err := svc.SetPaymentStatus("b1", model.PaymentStatusSettled)
	if err != nil {
		t.Fatalf("SetPaymentStatus() error: %v", err)
	}
	if billing.PaymentStatus != model.PaymentStatusSettled {
		t.Errorf("billing.PaymentStatus = %q, want %q", billing.PaymentStatus, model.PaymentStatusSettled)
	}
}

func TestInvoiceService_GetBillingPDF(t *testing.T) {
	svc, _ := newTestInvoiceService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/billings/b1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.Billing{
			ID:            "b1",
			Title:         "PDF Billing",
			PaymentStatus: model.PaymentStatusUnsettled,
			PDFURL:        "https://example.com/invoice/b1.pdf",
			CreatedAt:     "2024-01-01",
			UpdatedAt:     "2024-01-01",
		})
	})

	pdfURL, err := svc.GetBillingPDF("b1")
	if err != nil {
		t.Fatalf("GetBillingPDF() error: %v", err)
	}
	if pdfURL != "https://example.com/invoice/b1.pdf" {
		t.Errorf("pdfURL = %q, want %q", pdfURL, "https://example.com/invoice/b1.pdf")
	}
}
