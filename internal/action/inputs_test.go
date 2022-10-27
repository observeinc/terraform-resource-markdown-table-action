package action

import (
	"reflect"
	"testing"
)

func TestResourcesInput_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    TerraformResources
		wantErr bool
	}{
		{
			name:  "empty",
			input: "",
			want:  TerraformResources{},
		},
		{
			name:  "resource",
			input: "[{name: foo, attributes: [bar]}]",
			want: TerraformResources{
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
	t.Parallel()

	tests := []struct {
		name      string
		resources TerraformResources
		valid     bool
	}{
		{
			name:      "empty resources",
			resources: TerraformResources{},
			valid:     false,
		},
		{
			name: "empty attributes",
			resources: TerraformResources{
				{
					Name:       "foo",
					Attributes: []string{},
				},
			},
			valid: false,
		},
		{
			name: "valid",
			resources: TerraformResources{
				{
					Name:       "foo",
					Attributes: []string{"bar"},
				},
			},
			valid: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.resources.Validate()

			if valid := err == nil; valid != tc.valid {
				t.Errorf("Validate(), got %v, want %v", valid, tc.valid)
				return
			}

		})
	}
}
