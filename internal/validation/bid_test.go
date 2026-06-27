package validation

import "testing"

func TestValidateBidPrice(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "plain", in: "0.5", want: "0.5"},
		{name: "with dollar", in: "$0.5", want: "0.5"},
		{name: "with spaces", in: "  0.005 ", want: "0.005"},
		{name: "multiple of 0.005", in: "0.015", want: "0.015"},
		{name: "trailing zeros trimmed", in: "0.500", want: "0.5"},
		{name: "whole number keeps one decimal", in: "1", want: "1.0"},
		{name: "empty", in: "", wantErr: true},
		{name: "only dollar", in: "$", wantErr: true},
		{name: "zero", in: "0", wantErr: true},
		{name: "negative", in: "-1", wantErr: true},
		{name: "not multiple of 0.005", in: "0.007", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateBidPrice(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %q", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("ValidateBidPrice(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
