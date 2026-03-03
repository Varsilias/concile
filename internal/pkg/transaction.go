package pkg

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
	OutflowAmount string `json:"Transaction Amount"`
	Type          string `json:"Type"`
	StatementID   string `json:"Statement IDs"`
	ResponseCode  string `json:"Transaction Response"`
	Currency      string `json:"Currency"`
	Wallet        string `json:"Wallet Name"`
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

// OUTFLOW SCHEMA EXAMPLE DATA
// {
// 	"Session ID":"414002479498026811172690138178",
// 	"Statement IDS":"31356989-31356993",
// 	"Transaction Amount":"24000",
// 	"Transaction Date":"2022-04-02 01:42:46",
// 	"Transaction Reference":"v1-zpay-959dc989-8ca4-4531-8ee3-76df032627b1",
// 	"Transaction Response":"00","
// 	Type":"OUTFLOW"
// }

// INFLOW SCHEMA EXAMPLE DATA
// {
// 	"Amount":"3,464,883.32",
// 	"From Account No":"4243914279",
// 	"From Bank":"Access Bank",
// 	"Session ID":"941848771823095733170191134194",
// 	"To Account No":"5773153770",
// 	"Transaction Date":"2022-02-17 06:40:12",
// 	"Transaction Reference":"Zpay-20220217064012271",
// 	"Type":"INFLOW",
// 	"Wallet Name":"Zpay"
// }

func Normalize(rawTrx RawTransaction) (CanonicalTransaction, error) {
	var nilTrx CanonicalTransaction
	trim := strings.TrimSpace
	// validate required fields
	if trim(rawTrx.Reference) == "" || trim(rawTrx.SessionID) == "" || trim(rawTrx.Date) == "" || trim(rawTrx.Type) == "" {
		return nilTrx, fmt.Errorf("invalid canonical trx structure")
	}
	// inflow validation goes with more checks due to the structure of the schema
	if trim(rawTrx.Type) == string(INFLOW) && (trim(rawTrx.Amount) == "" || trim(rawTrx.FromAccountNo) == "" || trim(rawTrx.FromBank) == "" || trim(rawTrx.ToAccountNo) == "") {
		return nilTrx, fmt.Errorf("Amount cannot be empty, if transaction type is INFLOW")
	}

	if trim(rawTrx.Type) == string(OUTFLOW) && trim(rawTrx.OutflowAmount) == "" {
		return nilTrx, fmt.Errorf("Transaction Amount cannot be empty, if transaction type is OUTFLOW")

	}
	parsedDate, err := time.ParseInLocation("2006-01-02 15:04:05", trim(rawTrx.Date), time.UTC)
	if err != nil {
		return nilTrx, err
	}

	if parsedDate.IsZero() {
		return nilTrx, fmt.Errorf("invalid transaction date format")
	}

	var amountMinor float64
	// since we take inflow or outflow type of jsonl file, only one of the amount can be available at any time
	if trim(rawTrx.Amount) != "" {
		iAmount, err := strconv.ParseFloat(strings.ReplaceAll(trim(rawTrx.Amount), ",", ""), 64)
		if err != nil {
			return nilTrx, err
		}
		amountMinor = iAmount * 100 // converting to Kobo
	} else if trim(rawTrx.OutflowAmount) != "" {
		oAmount, err := strconv.ParseFloat(strings.ReplaceAll(trim(rawTrx.OutflowAmount), ",", ""), 64)
		if err != nil {
			return nilTrx, err
		}
		amountMinor = oAmount * 100 // converting to Kobo

	}
	return CanonicalTransaction{
		Reference:   trim(rawTrx.Reference),
		FromAccount: trim(rawTrx.FromAccountNo),
		ToAccount:   trim(rawTrx.ToAccountNo),
		FromBank:    trim(rawTrx.FromBank),
		SessionID:   trim(rawTrx.SessionID),
		Timestamp:   parsedDate,
		AmountMinor: int64(amountMinor),
		Currency:    rawTrx.Currency,
		Type:        TransactionType(rawTrx.Type),
	}, nil
}
