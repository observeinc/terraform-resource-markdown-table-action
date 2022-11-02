package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
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
				"foo": &UnknownAttributeValue{},
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

			if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(UnknownAttributeValue{}, "Expr")); diff != "" {
				t.Errorf("unexpected attributes -want +got:\n%s", diff)
			}
		})
	}
}

func TestParser_ResourcesOfType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       string
		resourceType string
		want         []*tfconfig.Resource
	}{
		{
			name: "one",
			config: `
resource "test_resource_a" "test" {}
`,
			resourceType: "test_resource_a",
			want: []*tfconfig.Resource{
				{
					Type: "test_resource_a",
					Name: "test",
				},
			},
		},
		{
			name:         "empty",
			resourceType: "test_resource_a",
			want:         []*tfconfig.Resource{},
		},
		{
			name: "ignore other types",
			config: `
resource "test_resource_a" "test" {}
resource "test_resource_b" "test" {}
`,
			resourceType: "test_resource_a",
			want: []*tfconfig.Resource{
				{
					Type: "test_resource_a",
					Name: "test",
				},
			},
		},
		{
			name: "sort resources by name",
			config: `
resource "test_resource_a" "c" {}
resource "test_resource_a" "b" {}
resource "test_resource_a" "a" {}
`,
			resourceType: "test_resource_a",
			want: []*tfconfig.Resource{
				{
					Type: "test_resource_a",
					Name: "a",
				},
				{
					Type: "test_resource_a",
					Name: "b",
				},
				{
					Type: "test_resource_a",
					Name: "c",
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser, err := NewParser(&tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"registry.terraform.io/test/test": {
						ResourceSchemas: map[string]*tfjson.Schema{
							"test_resource_a": {
								Block: &tfjson.SchemaBlock{
									Attributes: map[string]*tfjson.SchemaAttribute{
										"foo": {
											AttributeType: cty.Bool,
										},
									},
								},
							},
							"test_resource_b": {
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
			})

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

			got := parser.ResourcesOfType(tc.resourceType)

			opts := cmp.Options{
				cmpopts.IgnoreFields(tfconfig.Resource{}, "Provider", "Pos", "Mode"),
			}

			if diff := cmp.Diff(got, tc.want, opts); diff != "" {
				t.Errorf("unexpected resources -want +got:\n%s", diff)
			}
		})
	}
}
