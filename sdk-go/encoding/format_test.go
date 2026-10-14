package encoding_test

import (
	"mime"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
)

func TestFormat(t *testing.T) {
	for _, format := range encoding.Formats() {
		for _, ext := range format.AltExts() {
			if format.MimeType() != mime.TypeByExtension(ext) {
				t.Fatalf("expected %s ext %s to be registerd", format, ext)
			}
		}

		exts, err := mime.ExtensionsByType(format.MimeType())
		if err != nil {
			t.Fatal(err)
		}

		for _, ext := range exts {
			if format.MimeType() != mime.TypeByExtension(ext) {
				t.Fatalf("expected %s mime type to be: %s, got: %s", format, format.MimeType(), mime.TypeByExtension(format.Ext()))
			}
		}
	}
}
