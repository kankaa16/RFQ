package models

type Customer struct {
	BaseModel `bson:",inline"`

	CompanyName  string `bson:"company_name" json:"company_name"`
	CustomerCode string `bson:"customer_code" json:"customer_code"`

	Industry     string `bson:"industry" json:"industry"`
	CustomerType string `bson:"customer_type" json:"customer_type"`

	ContactPerson string `bson:"contact_person" json:"contact_person"`
	Email         string `bson:"email" json:"email"`
	Phone         string `bson:"phone" json:"phone"`

	PaymentTerms string `bson:"payment_terms" json:"payment_terms"`
	GSTIN        string `bson:"gstin,omitempty" json:"gstin,omitempty"`

	PreferredCurrency string `bson:"preferred_currency" json:"preferred_currency"`

	IsActive bool `bson:"is_active" json:"is_active"`
}

const (
	CustomerTypeOEM   = "oem"
	CustomerTypeTier1 = "tier_1"
	CustomerTypeOther = "other"
)
