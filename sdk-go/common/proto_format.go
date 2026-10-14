package common

import (
	"fmt"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
	moneypb "google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/protobuf/proto"
)

var defaultFormatProtoOptions = FormatProtoOptions{
	locale:  language.AmericanEnglish,
	printer: message.NewPrinter(language.AmericanEnglish),
}

type FormatProtoOptions struct {
	locale  language.Tag
	printer *message.Printer
}

func FormatProto(msg proto.Message, opts ...message.Option) (string, error) {
	return defaultFormatProtoOptions.Format(msg, opts...)
}

func FormatMoney(amount *moneypb.Money, opts ...message.Option) (string, error) {
	return defaultFormatProtoOptions.FormatMoney(amount, opts...)
}

func (f FormatProtoOptions) Format(msg proto.Message, opts ...message.Option) (string, error) {
	switch msg := msg.(type) {
	case *moneypb.Money:
		return FormatMoney(msg, opts...)
	}
	return "", fmt.Errorf("format not found for proto: %s", msg.ProtoReflect().Descriptor().FullName())
}

func (f FormatProtoOptions) FormatMoney(amount *moneypb.Money, opts ...message.Option) (string, error) {
	curr, err := currency.ParseISO(amount.GetCurrencyCode())
	if err != nil {
		return "", err
	}

	totalAmount := float64(amount.GetUnits()) + (float64(amount.GetNanos()) / 1e9)
	scale, _ := currency.Cash.Rounding(curr)
	formatter := number.Decimal(totalAmount, number.Scale(scale))

	printer := f.printer
	if printer == nil {
		printer = message.NewPrinter(f.locale, opts...)
	}

	return printer.Sprintf("%v%v", currency.Symbol(curr), formatter), nil
}
