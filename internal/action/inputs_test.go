package action

import (
	"reflect"
	"testing"
)

func TestResourcesInput_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Resources
		wantErr bool
	}{
		{
			name:  "empty",
			input: "",
			want:  Resources{},
		},
		{
			name:  "resource",
			input: "[{name: foo, attributes: [bar]}]",
			want: Resources{
				{
					Name:       "foo",
					Attributes: []string{"bar"},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := ResourcesInput(tc.input)
			got, err := r.Parse()
			if (err != nil) != tc.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse() got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResources_Validate(t *testing.T) {
	tests := []struct {
		name      string
		resources Resources
		valid     bool
	}{
		{
			name:      "empty resources",
			resources: Resources{},
			valid:     false,
		},
		{
			name: "empty attributes",
			resources: Resources{
				{
					Name:       "foo",
					Attributes: []string{},
				},
			},
			valid: false,
		},
		{
			name: "valid",
			resources: Resources{
				{
					Name:       "foo",
					Attributes: []string{"bar"},
				},
			},
			valid: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.resources.Validate()

			if valid := err != nil; valid != tc.valid {
				t.Errorf("Validate(), got %v, want %v", valid, tc.valid)
				return
			}

		})
	}
}
