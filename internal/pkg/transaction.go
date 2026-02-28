package pkg

import "time"

type TransactionType string

const (
	INFLOW  TransactionType = "INFLOW"
	OUTFLOW TransactionType = "OUTFLOW"
)

type RawTransaction struct {
	Reference     string `json:"Transaction Reference"`
	FromAccountNo string `json:"From Account No"`
	FromBank      string `json:"From Bank"`
	ToAccountNo   string `json:"To Account No"`
	SessionID     string `json:"Session ID"`
	Date          string `json:"Transaction Date"`
	Amount        string `json:"Amount"`
	Type          string `json:"Type,omitempty"`
	StatementID   string `json:"Statement IDs,omitempty"`
	ResponseCode  string `json:"Transaction Response,omitempty"`
	Currency      string `json:"Currency,omitempty"`
	Wallet        string `json:"Wallet Name,omitempty"`
}

type CanonicalTransaction struct {
	Reference   string
	FromAccount string
	ToAccount   string
	FromBank    string
	SessionID   string
	Timestamp   time.Time
	AmountMinor int64
	Currency    string
	Type        TransactionType
}

func Normalize(rawTrx RawTransaction) (CanonicalTransaction, error) {
	return CanonicalTransaction{}, nil
}
