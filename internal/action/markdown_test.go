package action

import (
	"testing"

	"github.com/observeinc/terraform-resource-markdown-table-action/internal/terraform"
)

func TestValueToMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{
			name:  "string",
			value: "foo",
			want:  "foo",
		},
		{
			name:  "int",
			value: 1,
			want:  "1",
		},
		{
			name:  "float",
			value: 1.1,
			want:  "1.1",
		},
		{
			name:  "bool",
			value: true,
			want:  "true",
		},
		{
			name:  "unknown",
			value: &terraform.UnknownAttributeValue{},
			want:  "_unknown_",
		},
	}
	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := ValueToMarkdown(tc.value); got != tc.want {
				t.Errorf("ValueToMarkdown() = %v, want %v", got, tc.want)
			}
		})
	}
}
