package auth

import "testing"

// A context that is not a gin context, as a direct call passes, has no client: the session gets NULL audit fields.
func TestClientOfPlainContextIsEmpty(t *testing.T) {
	if got := clientOf(t.Context()); got != (Client{}) {
		t.Fatalf("clientOf = %+v, want empty", got)
	}
}
