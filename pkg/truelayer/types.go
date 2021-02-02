package truelayer

import (
	"context"
	"time"
)

type Account struct {
	UpdateTimestamp time.Time     `json:"update_timestamp"`
	AccountID       string        `json:"account_id"`
	AccountType     string        `json:"account_type"`
	DisplayName     string        `json:"display_name"`
	Currency        string        `json:"currency"`
	AccountNumber   AccountNumber `json:"account_number"`
	Provider        Provider      `json:"provider"`

	client *Client
}

func (a *Account) Transactions(ctx context.Context) ([]Transaction, error) {
	return a.client.Transactions(ctx, a.AccountID)
}

func (a *Account) Balance(ctx context.Context) (*Balance, error) {
	return a.client.Balance(ctx, a.AccountID)
}

type AccountNumber struct {
	Iban     string `json:"iban"`
	Number   string `json:"number"`
	SortCode string `json:"sort_code"`
	SwiftBic string `json:"swift_bic"`
}

type Transaction struct {
	TransactionID             string         `json:"transaction_id"`
	Timestamp                 string         `json:"timestamp"`
	Description               string         `json:"description"`
	Amount                    float64        `json:"amount"`
	Currency                  string         `json:"currency"`
	TransactionType           string         `json:"transaction_type"`
	TransactionCategory       string         `json:"transaction_category"`
	TransactionClassification []string       `json:"transaction_classification"`
	MerchantName              string         `json:"merchant_name"`
	RunningBalance            RunningBalance `json:"running_balance"`
	Meta                      Meta           `json:"meta"`
}

type RunningBalance struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Meta struct {
	BankTransactionID           string `json:"bank_transaction_id"`
	ProviderTransactionCategory string `json:"provider_transaction_category"`
}

type Balance struct {
	Currency        string    `json:"currency"`
	Available       float64   `json:"available"`
	Current         float64   `json:"current"`
	Overdraft       float64   `json:"overdraft"`
	UpdateTimestamp time.Time `json:"update_timestamp"`
}

type Metadata struct {
	ClientID               string    `json:"client_id"`
	CredentialsID          string    `json:"credentials_id"`
	ConsentStatus          string    `json:"consent_status"`
	ConsentStatusUpdatedAt time.Time `json:"consent_status_updated_at"`
	ConsentCreatedAt       time.Time `json:"consent_created_at"`
	ConsentExpiresAt       time.Time `json:"consent_expires_at"`
	Provider               Provider  `json:"provider"`
	Scopes                 []string  `json:"scopes"`
	PrivacyPolicy          string    `json:"privacy_policy"`
}

type Provider struct {
	DisplayName string `json:"display_name"`
	LogoURI     string `json:"logo_uri"`
	ProviderID  string `json:"provider_id"`
}
