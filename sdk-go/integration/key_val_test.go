package integration

import "testing"

func TestJetStreamKeyVal(t *testing.T) {
	key := "foo$bar(Baz"
	key = keyvalKeyRegex.ReplaceAllString(key, "-")
	if key != "foo-bar-Baz" {
		t.Fatalf("expected foo-bar-Baz got %s", key)
	}
}
