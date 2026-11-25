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
	ruleTransition = "BR-03"
)

// transitionError is the field error of ErrInvalidTransition: the claim's status does not allow the decision (BR-03).
var transitionError = FieldError{Field: "status", Rule: ruleTransition, Message: "Only a submitted claim can be approved or rejected."}

// Limits of the decision comment (BR-04), counted in characters after trimming.
const (
	// MinReasonLength and MaxCommentLength bound a reject or return reason. An approve comment has only the upper bound.
	MinReasonLength  = 10
	MaxCommentLength = 500
)

// checkRules checks item count, amounts, total, and when firstSubmittedAt is set, the expense date window.
// The window runs from 90 days before the first submission through today's Bangkok calendar date (BR-07).
func checkRules(in Input, now, firstSubmittedAt time.Time) error {
	var fields []FieldError

	if n := len(in.Items); n < MinItems || n > MaxItems {
		fields = append(fields, FieldError{Field: "items", Rule: ruleItemLimits, Message: "A claim needs 1 to 20 line items."})
	}

	var today, earliest time.Time
	if !firstSubmittedAt.IsZero() {
		today = calendarDate(now.In(bangkok))
		earliest = calendarDate(firstSubmittedAt.In(bangkok)).AddDate(0, 0, -MaxExpenseAgeDays)
	}

	var total money.Satang

	for i := range in.Items {
		total += in.Items[i].Amount
		fields = append(fields, checkItem(i, &in.Items[i], today, earliest)...)
	}

	if total > MaxTotal {
		fields = append(fields, FieldError{Field: "total", Rule: ruleTotal, Message: "Claim total cannot be more than 50,000.00 THB."})
	}

	if len(fields) > 0 {
		return ValidationError{Fields: fields}
	}

	return nil
}

// calendarDate compares calendar dates independently of the timestamp's time and zone.
func calendarDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// checkItem checks the amount (BR-05, BR-12) and the date when the expense window applies (BR-07).
func checkItem(i int, item *LineItemInput, today, earliest time.Time) []FieldError {
	var fields []FieldError

	switch {
	case item.Amount < MinAmount || item.Amount > MaxAmount:
		fields = append(fields, FieldError{Field: itemField(i, "amount"), Rule: ruleItemLimits, Message: "Amount must be between 0.01 and 20,000.00 THB."})
	case item.Category == db.ExpenseCategoryMEALS && item.Amount > MaxMealsAmount:
		fields = append(fields, FieldError{Field: itemField(i, "amount"), Rule: ruleMeals, Message: "A Meals item cannot be more than 1,500.00 THB."})
	}

	if today.IsZero() {
		return fields
	}

	date := calendarDate(item.ExpenseDate)

	switch {
	case date.After(today):
		fields = append(fields, FieldError{Field: itemField(i, "expenseDate"), Rule: ruleDate, Message: "Expense date cannot be in the future."})
	case date.Before(earliest):
		fields = append(fields, FieldError{Field: itemField(i, "expenseDate"), Rule: ruleDate, Message: "Expense date cannot be more than 90 days before the claim was first submitted."})
	}

	return fields
}

// itemField names a field of line item i the way the problem's errors do, such as items[1].amount.
func itemField(i int, name string) string {
	return fmt.Sprintf("items[%d].%s", i, name)
}

// checkComment runs BR-04 on a decision comment: a reject or return reason is required with 10 to 500 characters, an
// approve comment is optional with at most 500. It counts characters, not bytes, after trimming, and returns a ValidationError on the
// comment field, or nil.
func checkComment(action Action, comment string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(comment))
	needsReason := action == ActionReject || action == ActionReturn

	var message string

	switch {
	case length > MaxCommentLength:
		message = "Comment cannot be more than 500 characters."
	case needsReason && length == 0:
		message = fmt.Sprintf("Enter a reason (%d to %d characters).", MinReasonLength, MaxCommentLength)
	case needsReason && length < MinReasonLength:
		message = fmt.Sprintf("Reason must be at least %d characters.", MinReasonLength)
	default:
		return nil
	}

	return ValidationError{Fields: []FieldError{{Field: "comment", Rule: ruleComment, Message: message}}}
}
