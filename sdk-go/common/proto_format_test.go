package common

import (
	"testing"

	"google.golang.org/genproto/googleapis/type/money"
)

func TestProtoFormat_Money(t *testing.T) {
	amount := new(money.Money{
		CurrencyCode: "EUR",
		Units:        83583,
		Nanos:        430000000,
	})

	str, err := FormatMoney(amount)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("amount: %s", str)
}
