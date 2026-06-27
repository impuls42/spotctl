package nodepools

import (
	"reflect"
	"testing"
)

func TestParseKVCommaSeparated(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    map[string]string
		wantErr bool
	}{
		{name: "empty", in: "", want: map[string]string{}},
		{name: "whitespace only", in: "   ", want: map[string]string{}},
		{name: "single", in: "a=1", want: map[string]string{"a": "1"}},
		{name: "multiple", in: "a=1,b=2", want: map[string]string{"a": "1", "b": "2"}},
		{name: "trims spaces", in: " a = 1 , b = 2 ", want: map[string]string{"a": "1", "b": "2"}},
		{name: "value contains equals", in: "a=1=2", want: map[string]string{"a": "1=2"}},
		{name: "empty value allowed", in: "a=", want: map[string]string{"a": ""}},
		{name: "missing equals", in: "a", wantErr: true},
		{name: "empty key", in: "=1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKVCommaSeparated(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil (result=%v)", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.in, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseKVCommaSeparated(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
