package domain

import (
	"time"
)

// PaymentTransaction represents an accredited incoming payment transaction (e.g. from bank transfer or virtual wallet).
type PaymentTransaction struct {
	ID                  string    `json:"id"`
	Amount              float64   `json:"amount"`
	Currency            string    `json:"currency"`
	DateApproved        time.Time `json:"date_approved"`
	OperationType       string    `json:"operation_type"`
	PayerName           string    `json:"payer_name"`
	PayerIdentification string    `json:"payer_identification"`
	Status              string    `json:"status"`
}

// ReconciliationStatus represents the outcome of an automated transfer reconciliation attempt.
type ReconciliationStatus string

const (
	ReconciliationStatusMatched                  ReconciliationStatus = "MATCHED"
	ReconciliationStatusNotFoundYet              ReconciliationStatus = "NOT_FOUND_YET"
	ReconciliationStatusAmbiguousMultipleMatches ReconciliationStatus = "AMBIGUOUS_MULTIPLE_MATCHES"
	ReconciliationStatusAmountMismatch           ReconciliationStatus = "AMOUNT_MISMATCH"
	ReconciliationStatusAlreadyUsed              ReconciliationStatus = "ALREADY_USED"
)

// ReconciliationResult provides the deterministic evaluation of matching an order against incoming payments.
type ReconciliationResult struct {
	Status             ReconciliationStatus `json:"status"`
	OrderID            uint                 `json:"order_id"`
	MatchedTransaction *PaymentTransaction  `json:"matched_transaction,omitempty"`
	CandidateCount     int                  `json:"candidate_count"`
	Message            string               `json:"message"`
}
