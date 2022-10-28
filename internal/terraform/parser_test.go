package terraform

import (
	"os"
	"path/filepath"
	"reflect"
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
		wantErr    bool
	}{
		{
			name: "string attribute",
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
		{
			name: "number attribute",
			providers: &tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"registry.terraform.io/test/test": {
						ResourceSchemas: map[string]*tfjson.Schema{
							"test_resource": {
								Block: &tfjson.SchemaBlock{
									Attributes: map[string]*tfjson.SchemaAttribute{
										"foo": {
											AttributeType: cty.Number,
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
	foo = 2
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
				"foo": float64(2),
			},
		},
		{
			name: "bool attribute",
			providers: &tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"registry.terraform.io/test/test": {
						ResourceSchemas: map[string]*tfjson.Schema{
							"test_resource": {
								Block: &tfjson.SchemaBlock{
									Attributes: map[string]*tfjson.SchemaAttribute{
										"foo": {
											AttributeType: cty.Bool,
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
	foo = true
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
				"foo": true,
			},
		},
		{
			name: "unevaluable attribute",
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
variable "foo" {}

resource "test_resource" "test" {
	foo = var.foo
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
				"foo": nil,
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

			if err := parser.LoadModule(dir); err != nil {
				t.Fatal(err)
			}

			got, err := parser.ResourceAttributes(parser.module.ManagedResources[tc.resource], []string{"foo"})
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
