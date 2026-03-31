package model

type Office struct {
	Name       string `json:"name"`
	Zip        string `json:"zip"`
	Prefecture string `json:"prefecture"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	Tel        string `json:"tel"`
	Fax        string `json:"fax"`
}

type Partner struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	NameKana    string              `json:"name_kana,omitempty"`
	NameSuffix  string              `json:"name_suffix,omitempty"`
	Code        string              `json:"code,omitempty"`
	Memo        string              `json:"memo,omitempty"`
	Departments []PartnerDepartment `json:"departments"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

type PartnerDepartment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Zip         string `json:"zip,omitempty"`
	Tel         string `json:"tel,omitempty"`
	Prefecture  string `json:"prefecture,omitempty"`
	Address1    string `json:"address1,omitempty"`
	Address2    string `json:"address2,omitempty"`
	PersonName  string `json:"person_name,omitempty"`
	PersonTitle string `json:"person_title,omitempty"`
	Email       string `json:"email,omitempty"`
	CCEmails    string `json:"cc_emails,omitempty"`
}

type CreatePartnerParams struct {
	Name       string `json:"name"`
	NameKana   string `json:"name_kana,omitempty"`
	NameSuffix string `json:"name_suffix,omitempty"`
	Code       string `json:"code,omitempty"`
	Memo       string `json:"memo,omitempty"`
}

type UpdatePartnerParams struct {
	Name       *string `json:"name,omitempty"`
	NameKana   *string `json:"name_kana,omitempty"`
	NameSuffix *string `json:"name_suffix,omitempty"`
	Code       *string `json:"code,omitempty"`
	Memo       *string `json:"memo,omitempty"`
}

// --- Enums ---

type PaymentStatus string

const (
	PaymentStatusUnsettled PaymentStatus = "unsettled"
	PaymentStatusSettled   PaymentStatus = "settled"
)

type ExciseType string

const (
	ExciseTenPercent                 ExciseType = "ten_percent"
	ExciseEightPercent               ExciseType = "eight_percent"
	ExciseEightPercentReducedTaxRate ExciseType = "eight_percent_as_reduced_tax_rate"
	ExciseFivePercent                ExciseType = "five_percent"
	ExciseUntaxable                  ExciseType = "untaxable"
	ExciseTaxExemption               ExciseType = "tax_exemption"
	ExciseNonTaxable                 ExciseType = "non_taxable"
)

// --- Item ---

type Item struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  *int   `json:"price,omitempty"`
	Quantity               *int   `json:"quantity,omitempty"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise,omitempty"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

type CreateItemParams struct {
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  *int   `json:"price,omitempty"`
	Quantity               *int   `json:"quantity,omitempty"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise,omitempty"`
}

type UpdateItemParams struct {
	Name                   *string `json:"name,omitempty"`
	Code                   *string `json:"code,omitempty"`
	Detail                 *string `json:"detail,omitempty"`
	Unit                   *string `json:"unit,omitempty"`
	Price                  *int    `json:"price,omitempty"`
	Quantity               *int    `json:"quantity,omitempty"`
	IsDeductWithholdingTax *bool   `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 *string `json:"excise,omitempty"`
}

// --- Billing ---

type Billing struct {
	ID               string        `json:"id"`
	PDFURL           string        `json:"pdf_url,omitempty"`
	OperatorID       string        `json:"operator_id,omitempty"`
	DepartmentID     string        `json:"department_id,omitempty"`
	PartnerID        string        `json:"partner_id,omitempty"`
	PartnerName      string        `json:"partner_name,omitempty"`
	PartnerDetail    string        `json:"partner_detail,omitempty"`
	MemberID         string        `json:"member_id,omitempty"`
	MemberName       string        `json:"member_name,omitempty"`
	Title            string        `json:"title,omitempty"`
	Memo             string        `json:"memo,omitempty"`
	PaymentCondition string        `json:"payment_condition,omitempty"`
	BillingNumber    string        `json:"billing_number,omitempty"`
	BillingDate      string        `json:"billing_date,omitempty"`
	DueDate          string        `json:"due_date,omitempty"`
	SalesDate        string        `json:"sales_date,omitempty"`
	PaymentStatus    PaymentStatus `json:"payment_status"`
	Subtotal         *int          `json:"subtotal,omitempty"`
	TotalPrice       *int          `json:"total_price,omitempty"`
	Tax              *int          `json:"tax,omitempty"`
	Items            []BillingItem `json:"items"`
	CreatedAt        string        `json:"created_at"`
	UpdatedAt        string        `json:"updated_at"`
}

type BillingItem struct {
	ID                     string `json:"id,omitempty"`
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  int    `json:"price"`
	Quantity               int    `json:"quantity"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise,omitempty"`
}

// InvoiceTemplateLine is a line item for billing/quote creation (Invoice Act compliant).
type InvoiceTemplateLine struct {
	Name                   string `json:"name" yaml:"name"`
	Code                   string `json:"code,omitempty" yaml:"code,omitempty"`
	Detail                 string `json:"detail,omitempty" yaml:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty" yaml:"unit,omitempty"`
	Price                  int    `json:"price" yaml:"price"`
	Quantity               int    `json:"quantity" yaml:"quantity"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty" yaml:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise" yaml:"excise"`
}

// CreateBillingParams is sent to POST /invoice_template_billings (Invoice Act).
type CreateBillingParams struct {
	DepartmentID     string                `json:"department_id"`
	BillingDate      string                `json:"billing_date"`
	Title            string                `json:"title,omitempty"`
	Memo             string                `json:"memo,omitempty"`
	PaymentCondition string                `json:"payment_condition,omitempty"`
	DueDate          string                `json:"due_date,omitempty"`
	SalesDate        string                `json:"sales_date,omitempty"`
	BillingNumber    string                `json:"billing_number,omitempty"`
	Items            []InvoiceTemplateLine `json:"items,omitempty"`
}

// UpdateBillingParams is wrapped as {"billing": {...}} for PATCH /billings/{id}.
type UpdateBillingParams struct {
	Title            *string               `json:"title,omitempty"`
	Memo             *string               `json:"memo,omitempty"`
	PaymentCondition *string               `json:"payment_condition,omitempty"`
	BillingDate      *string               `json:"billing_date,omitempty"`
	DueDate          *string               `json:"due_date,omitempty"`
	SalesDate        *string               `json:"sales_date,omitempty"`
	Items            []InvoiceTemplateLine  `json:"items,omitempty"`
}

// --- Quote ---

type QuoteStatus string

const (
	QuoteStatusDraft     QuoteStatus = "draft"
	QuoteStatusSent      QuoteStatus = "sent"
	QuoteStatusAccepted  QuoteStatus = "accepted"
	QuoteStatusRejected  QuoteStatus = "rejected"
	QuoteStatusCancelled QuoteStatus = "cancelled"
)

type Quote struct {
	ID            string      `json:"id"`
	PDFURL        string      `json:"pdf_url,omitempty"`
	OperatorID    string      `json:"operator_id,omitempty"`
	DepartmentID  string      `json:"department_id,omitempty"`
	PartnerID     string      `json:"partner_id,omitempty"`
	PartnerName   string      `json:"partner_name,omitempty"`
	PartnerDetail string      `json:"partner_detail,omitempty"`
	Title         string      `json:"title,omitempty"`
	Memo          string      `json:"memo,omitempty"`
	QuoteNumber   string      `json:"quote_number,omitempty"`
	QuoteDate     string      `json:"quote_date,omitempty"`
	ExpiredDate   string      `json:"expired_date,omitempty"`
	Status        QuoteStatus `json:"status"`
	Subtotal      *int        `json:"subtotal,omitempty"`
	TotalPrice    *int        `json:"total_price,omitempty"`
	Tax           *int        `json:"tax,omitempty"`
	Items         []QuoteItem `json:"items"`
	CreatedAt     string      `json:"created_at"`
	UpdatedAt     string      `json:"updated_at"`
}

type QuoteItem struct {
	ID                     string `json:"id,omitempty"`
	Name                   string `json:"name"`
	Code                   string `json:"code,omitempty"`
	Detail                 string `json:"detail,omitempty"`
	Unit                   string `json:"unit,omitempty"`
	Price                  int    `json:"price"`
	Quantity               int    `json:"quantity"`
	IsDeductWithholdingTax *bool  `json:"is_deduct_withholding_tax,omitempty"`
	Excise                 string `json:"excise,omitempty"`
}

// CreateQuoteParams is sent to POST /quotes (direct, no wrapping).
type CreateQuoteParams struct {
	DepartmentID string                `json:"department_id"`
	QuoteDate    string                `json:"quote_date"`
	ExpiredDate  string                `json:"expired_date"`
	Title        string                `json:"title,omitempty"`
	Memo         string                `json:"memo,omitempty"`
	Items        []InvoiceTemplateLine `json:"items,omitempty"`
}

// UpdateQuoteParams is sent to PATCH /quotes/{id} (direct, no wrapping).
type UpdateQuoteParams struct {
	Title       *string               `json:"title,omitempty"`
	Memo        *string               `json:"memo,omitempty"`
	QuoteDate   *string               `json:"quote_date,omitempty"`
	ExpiredDate *string               `json:"expired_date,omitempty"`
	Items       []InvoiceTemplateLine  `json:"items,omitempty"`
}

// --- Sent History ---

type SentHistory struct {
	ID         string `json:"id"`
	Type       string `json:"type,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	Operator   string `json:"operator,omitempty"`
	SentAt     string `json:"sent_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}
