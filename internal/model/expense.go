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

// OfficeMemberV2 is an employee from the Expense v2 API.
type OfficeMemberV2 struct {
	ID                 string `json:"id"`
	LoginID            string `json:"login_id,omitempty"`
	IdentificationCode string `json:"identification_code,omitempty"`
	Number             string `json:"number,omitempty"`
	Name               string `json:"name"`
	IsActive           bool   `json:"is_active"`
	IsExUser           bool   `json:"is_ex_user"`
	IsExAuthorizer     bool   `json:"is_ex_authorizer"`
	IsExAdministrator  bool   `json:"is_ex_administrator"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// ExTransaction is an expense transaction.
type ExTransaction struct {
	ID                        string  `json:"id"`
	Number                    int     `json:"number"`
	Remark                    string  `json:"remark"`
	RecognizedAt              string  `json:"recognized_at"`
	Value                     float64 `json:"value"`
	Memo                      string  `json:"memo,omitempty"`
	ReportNumber              string  `json:"report_number,omitempty"`
	JPYRate                   float64 `json:"jpyrate,omitempty"`
	Currency                  string  `json:"currency,omitempty"`
	UseCustomJPYRate          bool    `json:"use_custom_jpy_rate,omitempty"`
	AutomaticStatus           string  `json:"automatic_status,omitempty"`
	OfficeMemberID            string  `json:"office_member_id"`
	ExItemID                  string  `json:"ex_item_id,omitempty"`
	DrExciseID                string  `json:"dr_excise_id,omitempty"`
	DeptID                    string  `json:"dept_id,omitempty"`
	ProjectID                 string  `json:"project_id,omitempty"`
	CrItemID                  string  `json:"cr_item_id,omitempty"`
	CrSubItemID               string  `json:"cr_sub_item_id,omitempty"`
	InvoiceRegistrationNumber string  `json:"invoice_registration_number,omitempty"`
	InvoiceKind               int     `json:"invoice_kind,omitempty"`
	ExciseValue               int     `json:"excise_value,omitempty"`
	CreatedAt                 string  `json:"created_at"`
	UpdatedAt                 string  `json:"updated_at"`
}

// ExTransactionCreateInput is the request body for creating an expense transaction.
// Required fields: Remark, RecognizedAt, Value, ExItemID.
type ExTransactionCreateInput struct {
	Remark       string  `json:"remark"`
	RecognizedAt string  `json:"recognized_at"`
	Value        float64 `json:"value"`
	ExItemID     string  `json:"ex_item_id"`
	Memo         string  `json:"memo,omitempty"`
	ReportNumber string  `json:"report_number,omitempty"`
	DrExciseID   string  `json:"dr_excise_id,omitempty"`
	DeptID       string  `json:"dept_id,omitempty"`
	ProjectID    string  `json:"project_id,omitempty"`
	CrItemID     string  `json:"cr_item_id,omitempty"`
	CrSubItemID  string  `json:"cr_sub_item_id,omitempty"`
	JPYRate      float64 `json:"jpyrate,omitempty"`
	Currency     string  `json:"currency,omitempty"`
}

// ExTransactionUpdateInput is the request body for updating an expense transaction.
// Pointer types allow distinguishing between "not provided" and "set to zero value".
type ExTransactionUpdateInput struct {
	Remark       *string  `json:"remark,omitempty"`
	RecognizedAt *string  `json:"recognized_at,omitempty"`
	Value        *float64 `json:"value,omitempty"`
	ExItemID     *string  `json:"ex_item_id,omitempty"`
	Memo         *string  `json:"memo,omitempty"`
	ReportNumber *string  `json:"report_number,omitempty"`
	DrExciseID   *string  `json:"dr_excise_id,omitempty"`
	DeptID       *string  `json:"dept_id,omitempty"`
	ProjectID    *string  `json:"project_id,omitempty"`
	CrItemID     *string  `json:"cr_item_id,omitempty"`
	CrSubItemID  *string  `json:"cr_sub_item_id,omitempty"`
	JPYRate      *float64 `json:"jpyrate,omitempty"`
	Currency     *string  `json:"currency,omitempty"`
}
