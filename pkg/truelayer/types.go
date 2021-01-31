package truelayer

import "time"

type Account struct {
	UpdateTimestamp time.Time     `json:"update_timestamp"`
	AccountID       string        `json:"account_id"`
	AccountType     string        `json:"account_type"`
	DisplayName     string        `json:"display_name"`
	Currency        string        `json:"currency"`
	AccountNumber   AccountNumber `json:"account_number"`
	Provider        Provider      `json:"provider"`
}

type AccountNumber struct {
	Iban     string `json:"iban"`
	Number   string `json:"number"`
	SortCode string `json:"sort_code"`
	SwiftBic string `json:"swift_bic"`
}

type Provider struct {
	ProviderID string `json:"provider_id"`
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
