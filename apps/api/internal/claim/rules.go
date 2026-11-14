package claim

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// FieldError is one broken rule on one field: the field path, such as items[1].amount, the rule ID, such as BR-05, and the message to show.
type FieldError struct {
	Field   string
	Rule    string
	Message string
}

// ValidationError carries every broken rule of an input, one entry per field (NFR-08).
type ValidationError struct {
	Fields []FieldError
}

func (e ValidationError) Error() string {
	return "claim: validation failed"
}

// Limits of the claim rules (BR-05, BR-06, BR-07, BR-12). They are not in the contract schema, which checks only shape.
const (
	// MinItems and MaxItems bound the line items of a claim (BR-05).
	MinItems = 1
	MaxItems = 20
	// MinAmount and MaxAmount bound each line item amount, 0.01 to 20,000.00 THB (BR-05).
	MinAmount money.Satang = 1
	MaxAmount money.Satang = 2_000_000
	// MaxTotal bounds the total of a claim, 50,000.00 THB (BR-06).
	MaxTotal money.Satang = 5_000_000
	// MaxMealsAmount bounds a Meals item, 1,500.00 THB (BR-12).
	MaxMealsAmount money.Satang = 150_000
	// MaxExpenseAgeDays is how many days before the first submission an expense date may fall (BR-07).
	MaxExpenseAgeDays = 90
)

// Rule IDs of the field errors.
const (
	ruleItemLimits = "BR-05"
	ruleTotal      = "BR-06"
	ruleDate       = "BR-07"
	ruleMeals      = "BR-12"
	ruleComment    = "BR-04"
)

// Limits of the decision comment (BR-04), counted in characters after trimming.
const (
	// MinReasonLength and MaxCommentLength bound a reject reason. An approve comment has only the upper bound.
	MinReasonLength  = 5
	MaxCommentLength = 500
)

// ShapeMessages are the messages of the 400 errors that name a field of ClaimInput (contract §1.4), keyed by the field
// path with every array index written as []. The schema checks shape only: types, required fields, string lengths,
// the money format, the category enum, and the date format.
var ShapeMessages = map[string]string{
	"title":               "Enter a title (up to 120 characters).",
	"purpose":             "Enter a purpose (up to 1,000 characters).",
	"items":               "A claim needs 1 to 20 line items.",
	"items[]":             "Enter the line item.",
	"items[].category":    "Choose a category.",
	"items[].description": "Enter a description (up to 200 characters).",
	"items[].expenseDate": "Enter a date as YYYY-MM-DD.",
	"items[].amount":      "Enter an amount with 2 decimals, e.g. 1250.00.",
	"items[].receiptRef":  "Enter the receipt reference (up to 50 characters).",
}

// checkRules runs the field rules BR-05, BR-06, BR-07, and BR-12 (SDS §3.3, step 3) and returns a ValidationError with
// one entry per broken field, or nil. Create calls it after the BR-15 check and before the first write. Dates are
// calendar dates in Asia/Bangkok: an expense date may not be after today and may not be more than 90 days before it,
// where today is the Bangkok date of now (the first submission is today).
func checkRules(in Input, now time.Time) error {
	var fields []FieldError

	if n := len(in.Items); n < MinItems || n > MaxItems {
		fields = append(fields, FieldError{Field: "items", Rule: ruleItemLimits, Message: "A claim needs 1 to 20 line items."})
	}

	nowInBangkok := now.In(bangkok)
	today := time.Date(nowInBangkok.Year(), nowInBangkok.Month(), nowInBangkok.Day(), 0, 0, 0, 0, time.UTC)
	oldest := today.AddDate(0, 0, -MaxExpenseAgeDays)

	var total money.Satang

	for i, item := range in.Items {
		total += item.Amount

		switch {
		case item.Amount < MinAmount || item.Amount > MaxAmount:
			fields = append(fields, FieldError{Field: itemField(i, "amount"), Rule: ruleItemLimits, Message: "Amount must be between 0.01 and 20,000.00 THB."})
		case item.Category == db.ExpenseCategoryMEALS && item.Amount > MaxMealsAmount:
			fields = append(fields, FieldError{Field: itemField(i, "amount"), Rule: ruleMeals, Message: "A Meals item cannot be more than 1,500.00 THB."})
		}

		date := time.Date(item.ExpenseDate.Year(), item.ExpenseDate.Month(), item.ExpenseDate.Day(), 0, 0, 0, 0, time.UTC)

		switch {
		case date.After(today):
			fields = append(fields, FieldError{Field: itemField(i, "expenseDate"), Rule: ruleDate, Message: "Expense date cannot be in the future."})
		case date.Before(oldest):
			fields = append(fields, FieldError{Field: itemField(i, "expenseDate"), Rule: ruleDate, Message: "Expense date cannot be more than 90 days before the claim was first submitted."})
		}
	}

	if total > MaxTotal {
		fields = append(fields, FieldError{Field: "total", Rule: ruleTotal, Message: "Claim total cannot be more than 50,000.00 THB."})
	}

	if len(fields) > 0 {
		return ValidationError{Fields: fields}
	}

	return nil
}

// itemField names a field of line item i the way the problem's errors do, such as items[1].amount.
func itemField(i int, name string) string {
	return fmt.Sprintf("items[%d].%s", i, name)
}

// checkComment runs BR-04 on a decision comment: a reject reason is required with 5 to 500 characters, an approve comment
// is optional with at most 500. It counts characters, not bytes, after trimming, and returns a ValidationError on the
// comment field, or nil.
func checkComment(action, comment string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(comment))

	var message string

	switch {
	case length > MaxCommentLength:
		message = "Comment cannot be more than 500 characters."
	case action == ActionReject && length == 0:
		message = "Enter a reason (5 to 500 characters)."
	case action == ActionReject && length < MinReasonLength:
		message = "Reason must be at least 5 characters."
	default:
		return nil
	}

	return ValidationError{Fields: []FieldError{{Field: "comment", Rule: ruleComment, Message: message}}}
}
