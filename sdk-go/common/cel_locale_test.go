package common

import (
	"testing"

	"golang.org/x/text/language"
	moneypb "google.golang.org/genproto/googleapis/type/money"
)

func TestCELLocal(t *testing.T) {
	celLocale := NewCELLocal(
		WithCELLocaleTag(language.AmericanEnglish),
	)

	amount := new(moneypb.Money{
		CurrencyCode: "USD",
		Units:        83583,
		Nanos:        430000000,
	})

	str := celLocale.Sprintf("%s", amount)
	exp := "$83,583.43"

	if str != exp {
		t.Fatalf("expected: %s, got: %s", exp, str)
	}
}
