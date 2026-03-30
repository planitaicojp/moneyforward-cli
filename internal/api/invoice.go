package api

import (
	"fmt"
	"net/http"

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
	u := fmt.Sprintf("%s/partners?%s", s.base, params.QueryString())
	if query != "" {
		u += "&q=" + query
	}
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
	u := fmt.Sprintf("%s/items?%s", s.base, params.QueryString())
	if query != "" {
		u += "&q=" + query
	}
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
