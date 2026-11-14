package claim

import "time"

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

// checkRules is the seam for the field rules BR-05, BR-06, BR-07, and BR-12 (SDS §3.3, step 3). Create calls it after
// the BR-15 check and before the first write, and returns what it returns.
func checkRules(_ Input, _ time.Time) error {
	return nil
}
