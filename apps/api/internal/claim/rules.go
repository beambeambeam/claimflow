package claim

import "time"

// checkRules is the seam for the field rules BR-05, BR-06, BR-07, and BR-12 (SDS §3.3, step 3). Create calls it after
// the BR-15 check and before the first write, and returns what it returns. F-02/02 fills it in with ValidationError;
// until then every input passes, and the database CHECK constraints are the only limit.
func checkRules(_ Input, _ time.Time) error {
	return nil
}
