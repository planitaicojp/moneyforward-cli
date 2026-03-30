package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

const invoiceBaseURL = "https://invoice.moneyforward.com/api/v3"

type InvoiceService struct {
	client *Client
	base   string
}

func NewInvoiceService(client *Client, base string) *InvoiceService {
	return &InvoiceService{client: client, base: base}
}

func NewInvoiceServiceDefault(client *Client) *InvoiceService {
	return NewInvoiceService(client, invoiceBaseURL)
}

// buildListQuery builds url.Values with pagination and optional search query.
// Callers can add more params before calling Encode().
func buildListQuery(params pagination.Params, query string) url.Values {
	v := make(url.Values)
	v.Set("page", strconv.Itoa(params.Page))
	v.Set("per_page", strconv.Itoa(params.PerPage))
	if query != "" {
		v.Set("q", query)
	}
	return v
}

// listResponse is a generic wrapper for paginated API responses.
type listResponse[T any] struct {
	Data       []T               `json:"data"`
	Pagination pagination.Result `json:"pagination"`
}

// departmentsResponse wraps the departments list endpoint response.
type departmentsResponse struct {
	Data []model.PartnerDepartment `json:"data"`
}

func (s *InvoiceService) GetOffice() (*model.Office, error) {
	var office model.Office
	err := s.client.DoJSON(http.MethodGet, s.base+"/office", nil, &office)
	if err != nil {
		return nil, fmt.Errorf("getting office: %w", err)
	}
	return &office, nil
}

func (s *InvoiceService) ListPartners(params pagination.Params, query string) ([]model.Partner, *pagination.Result, error) {
	u := s.base + "/partners?" + buildListQuery(params, query).Encode()
	var resp listResponse[model.Partner]
	if err := s.client.DoJSON(http.MethodGet, u, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing partners: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}

func (s *InvoiceService) GetPartner(id string) (*model.Partner, error) {
	var partner model.Partner
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/partners/%s", s.base, id), nil, &partner)
	if err != nil {
		return nil, fmt.Errorf("getting partner: %w", err)
	}
	return &partner, nil
}

func (s *InvoiceService) CreatePartner(params model.CreatePartnerParams) (*model.Partner, error) {
	var partner model.Partner
	err := s.client.DoJSON(http.MethodPost, s.base+"/partners", params, &partner)
	if err != nil {
		return nil, fmt.Errorf("creating partner: %w", err)
	}
	return &partner, nil
}

func (s *InvoiceService) UpdatePartner(id string, params model.UpdatePartnerParams) (*model.Partner, error) {
	var partner model.Partner
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/partners/%s", s.base, id), params, &partner)
	if err != nil {
		return nil, fmt.Errorf("updating partner: %w", err)
	}
	return &partner, nil
}

func (s *InvoiceService) DeletePartner(id string) error {
	err := s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/partners/%s", s.base, id), nil, nil)
	if err != nil {
		return fmt.Errorf("deleting partner: %w", err)
	}
	return nil
}

func (s *InvoiceService) ListPartnerDepartments(partnerID string) ([]model.PartnerDepartment, error) {
	var resp departmentsResponse
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/partners/%s/departments", s.base, partnerID), nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("listing partner departments: %w", err)
	}
	return resp.Data, nil
}

// --- Items ---

func (s *InvoiceService) ListItems(params pagination.Params, query string) ([]model.Item, *pagination.Result, error) {
	u := s.base + "/items?" + buildListQuery(params, query).Encode()
	var resp listResponse[model.Item]
	if err := s.client.DoJSON(http.MethodGet, u, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing items: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}

func (s *InvoiceService) GetItem(id string) (*model.Item, error) {
	var item model.Item
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/items/%s", s.base, id), nil, &item)
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}
	return &item, nil
}

func (s *InvoiceService) CreateItem(params model.CreateItemParams) (*model.Item, error) {
	var item model.Item
	err := s.client.DoJSON(http.MethodPost, s.base+"/items", params, &item)
	if err != nil {
		return nil, fmt.Errorf("creating item: %w", err)
	}
	return &item, nil
}

func (s *InvoiceService) UpdateItem(id string, params model.UpdateItemParams) (*model.Item, error) {
	var item model.Item
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/items/%s", s.base, id), params, &item)
	if err != nil {
		return nil, fmt.Errorf("updating item: %w", err)
	}
	return &item, nil
}

func (s *InvoiceService) DeleteItem(id string) error {
	err := s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/items/%s", s.base, id), nil, nil)
	if err != nil {
		return fmt.Errorf("deleting item: %w", err)
	}
	return nil
}

// --- Billings ---

// BillingListOptions holds filter parameters for listing billings.
type BillingListOptions struct {
	pagination.Params
	PartnerID     string
	PaymentStatus string
	From          string
	To            string
	Query         string
}

func (o BillingListOptions) queryString() string {
	v := buildListQuery(o.Params, o.Query)
	if o.PartnerID != "" {
		v.Set("partner_id", o.PartnerID)
	}
	if o.PaymentStatus != "" {
		v.Set("payment_status", o.PaymentStatus)
	}
	if o.From != "" {
		v.Set("from", o.From)
	}
	if o.To != "" {
		v.Set("to", o.To)
	}
	return v.Encode()
}

func (s *InvoiceService) ListBillings(opts BillingListOptions) ([]model.Billing, *pagination.Result, error) {
	u := fmt.Sprintf("%s/billings?%s", s.base, opts.queryString())
	var resp listResponse[model.Billing]
	if err := s.client.DoJSON(http.MethodGet, u, nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("listing billings: %w", err)
	}
	return resp.Data, &resp.Pagination, nil
}

func (s *InvoiceService) GetBilling(id string) (*model.Billing, error) {
	var billing model.Billing
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/billings/%s", s.base, id), nil, &billing)
	if err != nil {
		return nil, fmt.Errorf("getting billing: %w", err)
	}
	return &billing, nil
}

// CreateBilling uses POST /invoice_template_billings (Invoice Act compliant).
// Body is sent directly (no wrapping).
func (s *InvoiceService) CreateBilling(params model.CreateBillingParams) (*model.Billing, error) {
	var billing model.Billing
	err := s.client.DoJSON(http.MethodPost, s.base+"/invoice_template_billings", params, &billing)
	if err != nil {
		return nil, fmt.Errorf("creating billing: %w", err)
	}
	return &billing, nil
}

// UpdateBilling uses PATCH /billings/{id} with body wrapped as {"billing": {...}}.
func (s *InvoiceService) UpdateBilling(id string, params model.UpdateBillingParams) (*model.Billing, error) {
	wrapped := map[string]any{"billing": params}
	var billing model.Billing
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/billings/%s", s.base, id), wrapped, &billing)
	if err != nil {
		return nil, fmt.Errorf("updating billing: %w", err)
	}
	return &billing, nil
}

func (s *InvoiceService) DeleteBilling(id string) error {
	err := s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/billings/%s", s.base, id), nil, nil)
	if err != nil {
		return fmt.Errorf("deleting billing: %w", err)
	}
	return nil
}

// SetPaymentStatus uses PATCH /billings/{id} with {"billing": {"payment_status": status}}.
func (s *InvoiceService) SetPaymentStatus(id string, status model.PaymentStatus) (*model.Billing, error) {
	wrapped := map[string]any{"billing": map[string]any{"payment_status": status}}
	var billing model.Billing
	err := s.client.DoJSON(http.MethodPatch, fmt.Sprintf("%s/billings/%s", s.base, id), wrapped, &billing)
	if err != nil {
		return nil, fmt.Errorf("setting payment status: %w", err)
	}
	return &billing, nil
}

// GetBillingPDF returns the PDF URL for a billing.
func (s *InvoiceService) GetBillingPDF(id string) (string, error) {
	billing, err := s.GetBilling(id)
	if err != nil {
		return "", err
	}
	return billing.PDFURL, nil
}

// DownloadPDF fetches the PDF at the given URL.
// Uses a plain HTTP client (no auth) since PDF URLs are typically signed/external.
func (s *InvoiceService) DownloadPDF(pdfURL string) (*http.Response, error) {
	httpClient := &http.Client{Timeout: s.client.http.Timeout}
	resp, err := httpClient.Get(pdfURL)
	if err != nil {
		return nil, fmt.Errorf("downloading PDF: %w", err)
	}
	return resp, nil
}
