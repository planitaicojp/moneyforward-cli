package model

// ExpenseOffice is an office from the Expense API.
// Named ExpenseOffice to avoid collision with Invoice's Office type.
type ExpenseOffice struct {
	ID                 string `json:"id"`
	IdentificationCode string `json:"identification_code,omitempty"`
	OfficeTypeID       int    `json:"office_type_id,omitempty"` // 1:個人, 2:法人
	Name               string `json:"name"`
}

type Dept struct {
	ID              string `json:"id"`
	Code            string `json:"code,omitempty"`
	Name            string `json:"name"`
	DispOrder       int    `json:"disp_order,omitempty"`
	IsActive        bool   `json:"is_active"`
	ParentID        string `json:"parent_id,omitempty"`
	AncestryDepth   int    `json:"ancestry_depth,omitempty"`
	ExJournalDeptID string `json:"ex_journal_dept_id,omitempty"`
	RootID          string `json:"root_id,omitempty"`
}

type ExpenseProject struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	IsActive  bool   `json:"is_active"`
	DispOrder int    `json:"disp_order,omitempty"`
	ParentID  string `json:"parent_id,omitempty"`
	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_to,omitempty"`
}

type ExItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Code            string `json:"code,omitempty"`
	IsActive        bool   `json:"is_active"`
	ItemID          string `json:"item_id,omitempty"`
	SubItemID       string `json:"sub_item_id,omitempty"`
	DefaultExciseID string `json:"default_excise_id,omitempty"`
}

type ExpenseExcise struct {
	ID       string  `json:"id"`
	LongName string  `json:"long_name"`
	Code     string  `json:"code,omitempty"`
	Rate     float64 `json:"rate"`
}

type Position struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsAuthorizer bool   `json:"is_authorizer"`
	Priority     int    `json:"priority"`
}
