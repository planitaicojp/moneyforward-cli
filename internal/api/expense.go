package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/planitaicojp/moneyforward-cli/internal/model"
)

const (
	expenseBaseURL   = "https://expense.moneyforward.com/api/external/v1"
	expenseBaseURLV2 = "https://expense.moneyforward.com/api/external/v2"
)

type ExpenseService struct {
	client *Client
	base   string
	baseV2 string
}

func NewExpenseService(client *Client, base, baseV2 string) *ExpenseService {
	return &ExpenseService{client: client, base: base, baseV2: baseV2}
}

func NewExpenseServiceDefault(client *Client) *ExpenseService {
	return NewExpenseService(client, expenseBaseURL, expenseBaseURLV2)
}

// expenseListResponse is a generic helper for Expense API list responses.
type expenseListResponse[T any] struct {
	Items []T
	Next  *string
	Prev  *string
}

// decodeExpenseList decodes a list response where the array is under the given key.
func decodeExpenseList[T any](data []byte, key string) (*expenseListResponse[T], error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var items []T
	if itemsRaw, ok := raw[key]; ok {
		if err := json.Unmarshal(itemsRaw, &items); err != nil {
			return nil, fmt.Errorf("decoding %s: %w", key, err)
		}
	}
	resp := &expenseListResponse[T]{Items: items}
	if nextRaw, ok := raw["next"]; ok {
		var next *string
		if err := json.Unmarshal(nextRaw, &next); err == nil {
			resp.Next = next
		}
	}
	if prevRaw, ok := raw["prev"]; ok {
		var prev *string
		if err := json.Unmarshal(prevRaw, &prev); err == nil {
			resp.Prev = prev
		}
	}
	return resp, nil
}

// doExpenseList performs a GET request and decodes the list response.
func doExpenseList[T any](s *ExpenseService, url, key string) ([]T, bool, error) {
	body, err := s.client.DoRaw(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := decodeExpenseList[T](body, key)
	if err != nil {
		return nil, false, err
	}
	hasNext := resp.Next != nil && *resp.Next != ""
	return resp.Items, hasNext, nil
}

func (s *ExpenseService) ListOffices(page int) ([]model.ExpenseOffice, bool, error) {
	u := fmt.Sprintf("%s/offices?page=%d", s.base, page)
	items, hasNext, err := doExpenseList[model.ExpenseOffice](s, u, "offices")
	if err != nil {
		return nil, false, fmt.Errorf("listing offices: %w", err)
	}
	return items, hasNext, nil
}

// --- Depts ---
func (s *ExpenseService) ListDepts(officeID string, page int, keyword string) ([]model.Dept, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/depts?page=%d", s.base, url.PathEscape(officeID), page)
	if keyword != "" {
		u += "&search_keyword=" + url.QueryEscape(keyword)
	}
	items, hasNext, err := doExpenseList[model.Dept](s, u, "depts")
	if err != nil {
		return nil, false, fmt.Errorf("listing depts: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetDept(officeID, id string) (*model.Dept, error) {
	var dept model.Dept
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/depts/%s", s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, &dept)
	if err != nil {
		return nil, fmt.Errorf("getting dept: %w", err)
	}
	return &dept, nil
}

// --- Projects ---
func (s *ExpenseService) ListProjects(officeID string, page int, keyword string) ([]model.ExpenseProject, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/projects?page=%d", s.base, url.PathEscape(officeID), page)
	if keyword != "" {
		u += "&search_keyword=" + url.QueryEscape(keyword)
	}
	items, hasNext, err := doExpenseList[model.ExpenseProject](s, u, "projects")
	if err != nil {
		return nil, false, fmt.Errorf("listing projects: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetProject(officeID, id string) (*model.ExpenseProject, error) {
	var project model.ExpenseProject
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/projects/%s", s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, &project)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}
	return &project, nil
}

// --- ExItems (経費科目) ---
func (s *ExpenseService) ListExItems(officeID string, page int, keyword string) ([]model.ExItem, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/ex_items?page=%d", s.base, url.PathEscape(officeID), page)
	if keyword != "" {
		u += "&search_keyword=" + url.QueryEscape(keyword)
	}
	items, hasNext, err := doExpenseList[model.ExItem](s, u, "ex_items")
	if err != nil {
		return nil, false, fmt.Errorf("listing ex_items: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetExItem(officeID, id string) (*model.ExItem, error) {
	var item model.ExItem
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/ex_items/%s", s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, &item)
	if err != nil {
		return nil, fmt.Errorf("getting ex_item: %w", err)
	}
	return &item, nil
}

// --- Excises (税区分) ---
func (s *ExpenseService) ListExcises(officeID string, page int) ([]model.ExpenseExcise, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/excises?page=%d", s.base, url.PathEscape(officeID), page)
	items, hasNext, err := doExpenseList[model.ExpenseExcise](s, u, "excises")
	if err != nil {
		return nil, false, fmt.Errorf("listing excises: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetExcise(officeID, id string) (*model.ExpenseExcise, error) {
	var excise model.ExpenseExcise
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/excises/%s", s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, &excise)
	if err != nil {
		return nil, fmt.Errorf("getting excise: %w", err)
	}
	return &excise, nil
}

// --- Positions (役職) ---
func (s *ExpenseService) ListPositions(officeID string, page int) ([]model.Position, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/positions?page=%d", s.base, url.PathEscape(officeID), page)
	items, hasNext, err := doExpenseList[model.Position](s, u, "positions")
	if err != nil {
		return nil, false, fmt.Errorf("listing positions: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetPosition(officeID, id string) (*model.Position, error) {
	var pos model.Position
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/positions/%s", s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, &pos)
	if err != nil {
		return nil, fmt.Errorf("getting position: %w", err)
	}
	return &pos, nil
}

// --- Members (v2) ---
func (s *ExpenseService) ListMembersV2(officeID string, page int, onlyActive bool) ([]model.OfficeMemberV2, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/office_members?page=%d&only_active=%t",
		s.baseV2, url.PathEscape(officeID), page, onlyActive)
	items, hasNext, err := doExpenseList[model.OfficeMemberV2](s, u, "office_members")
	if err != nil {
		return nil, false, fmt.Errorf("listing members: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetMemberV2(officeID, id string) (*model.OfficeMemberV2, error) {
	var member model.OfficeMemberV2
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/office_members/%s",
		s.baseV2, url.PathEscape(officeID), url.PathEscape(id)), nil, &member)
	if err != nil {
		return nil, fmt.Errorf("getting member: %w", err)
	}
	return &member, nil
}

func (s *ExpenseService) GetMe(officeID string) (*model.OfficeMemberV2, error) {
	var me model.OfficeMemberV2
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/me",
		s.baseV2, url.PathEscape(officeID)), nil, &me)
	if err != nil {
		return nil, fmt.Errorf("getting me: %w", err)
	}
	return &me, nil
}

// --- Transactions (v1) ---

func (s *ExpenseService) ListMyTransactions(officeID string, page int) ([]model.ExTransaction, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/me/ex_transactions?page=%d",
		s.base, url.PathEscape(officeID), page)
	items, hasNext, err := doExpenseList[model.ExTransaction](s, u, "ex_transactions")
	if err != nil {
		return nil, false, fmt.Errorf("listing my transactions: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetMyTransaction(officeID, id string) (*model.ExTransaction, error) {
	var tx model.ExTransaction
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/me/ex_transactions/%s",
		s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, &tx)
	if err != nil {
		return nil, fmt.Errorf("getting my transaction: %w", err)
	}
	return &tx, nil
}

func (s *ExpenseService) ListOrgTransactions(officeID string, page int) ([]model.ExTransaction, bool, error) {
	u := fmt.Sprintf("%s/offices/%s/ex_transactions?page=%d",
		s.base, url.PathEscape(officeID), page)
	items, hasNext, err := doExpenseList[model.ExTransaction](s, u, "ex_transactions")
	if err != nil {
		return nil, false, fmt.Errorf("listing org transactions: %w", err)
	}
	return items, hasNext, nil
}

func (s *ExpenseService) GetOrgTransaction(officeID, id string) (*model.ExTransaction, error) {
	var tx model.ExTransaction
	err := s.client.DoJSON(http.MethodGet, fmt.Sprintf("%s/offices/%s/ex_transactions/%s",
		s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, &tx)
	if err != nil {
		return nil, fmt.Errorf("getting org transaction: %w", err)
	}
	return &tx, nil
}

func (s *ExpenseService) CreateMyTransaction(officeID string, input model.ExTransactionCreateInput) (*model.ExTransaction, error) {
	var tx model.ExTransaction
	err := s.client.DoJSON(http.MethodPost, fmt.Sprintf("%s/offices/%s/me/ex_transactions",
		s.base, url.PathEscape(officeID)), input, &tx)
	if err != nil {
		return nil, fmt.Errorf("creating transaction: %w", err)
	}
	return &tx, nil
}

func (s *ExpenseService) UpdateMyTransaction(officeID, id string, input model.ExTransactionUpdateInput) (*model.ExTransaction, error) {
	var tx model.ExTransaction
	err := s.client.DoJSON(http.MethodPut, fmt.Sprintf("%s/offices/%s/me/ex_transactions/%s",
		s.base, url.PathEscape(officeID), url.PathEscape(id)), input, &tx)
	if err != nil {
		return nil, fmt.Errorf("updating my transaction: %w", err)
	}
	return &tx, nil
}

func (s *ExpenseService) DeleteMyTransaction(officeID, id string) error {
	return s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/offices/%s/me/ex_transactions/%s",
		s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, nil)
}

func (s *ExpenseService) UpdateOrgTransaction(officeID, id string, input model.ExTransactionUpdateInput) (*model.ExTransaction, error) {
	var tx model.ExTransaction
	err := s.client.DoJSON(http.MethodPut, fmt.Sprintf("%s/offices/%s/ex_transactions/%s",
		s.base, url.PathEscape(officeID), url.PathEscape(id)), input, &tx)
	if err != nil {
		return nil, fmt.Errorf("updating org transaction: %w", err)
	}
	return &tx, nil
}

func (s *ExpenseService) DeleteOrgTransaction(officeID, id string) error {
	return s.client.DoJSON(http.MethodDelete, fmt.Sprintf("%s/offices/%s/ex_transactions/%s",
		s.base, url.PathEscape(officeID), url.PathEscape(id)), nil, nil)
}
