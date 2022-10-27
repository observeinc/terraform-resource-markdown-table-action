package terraform

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func TestParserResourceAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		providers  *tfjson.ProviderSchemas
		config     string
		resource   string
		attributes []string
		want       map[string]interface{}
	}{
		{
			name: "simple",
			providers: &tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"registry.terraform.io/test/test": {
						ResourceSchemas: map[string]*tfjson.Schema{
							"test_resource": {
								Block: &tfjson.SchemaBlock{
									Attributes: map[string]*tfjson.SchemaAttribute{
										"foo": {
											AttributeType: cty.String,
										},
									},
								},
							},
						},
					},
				},
			},
			config: `
resource "test_resource" "test" {
	foo = "bar"
}

terraform {
	required_providers {
		test = {
			source = "test/test"
		}
	}
}
`,
			resource: "test_resource.test",
			want: map[string]interface{}{
				"foo": "bar",
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser, err := NewParser(tc.providers)
			if err != nil {
				t.Fatal(err)
			}

			dir, err := os.MkdirTemp("", "test")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })

			if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tc.config), 0644); err != nil {
				t.Fatal(err)
			}

			parser.LoadModule(dir)

			rp := strings.SplitN(tc.resource, ".", 2)
			if len(rp) != 2 {
				t.Fatalf("invalid resource name: %s", tc.resource)
			}

			got, err := parser.ResourceAttributes(rp[0], rp[1], []string{"foo"})
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
