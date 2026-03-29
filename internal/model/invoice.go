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
