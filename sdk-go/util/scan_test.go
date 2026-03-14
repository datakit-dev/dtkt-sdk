package util

import "testing"

func TestScanValue(t *testing.T) {
	tests := []struct {
		name    string
		typ     any
		str     string
		want    any
		wantErr bool
	}{
		{"int", 0, "42", 42, false},
		{"string", "", "hello", "hello", false},
		{"bool", false, "true", true, false},
		{"invalid type", 0.0, "3.14", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ScanValue(tt.typ, tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScanValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ScanValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
