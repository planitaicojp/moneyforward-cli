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

func (s *InvoiceService) ListPartners(params pagination.Params) ([]model.Partner, *pagination.Result, error) {
	url := fmt.Sprintf("%s/partners?%s", s.base, params.QueryString())
	var resp listResponse[model.Partner]
	if err := s.client.DoJSON(http.MethodGet, url, nil, &resp); err != nil {
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
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/partners/%s", s.base, id), nil)
	if err != nil {
		return fmt.Errorf("creating delete request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("deleting partner: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("deleting partner: HTTP %d", resp.StatusCode)
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
