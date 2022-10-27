package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/earlydecoder"
	"github.com/hashicorp/terraform-schema/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/sethvargo/go-githubactions"
	"gopkg.in/yaml.v3"
)

type Resource struct {
	Name       string   `yaml:"name"`
	Attributes []string `yaml:"attributes"`
}

func main() {
	workingDirectory := githubactions.GetInput("working_directory")

	resources, err := getResourceInput()
	if err != nil {
		githubactions.Fatalf(err.Error())
	}

	if len(resources) == 0 {
		githubactions.Fatalf("No resources found")
	}

	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		githubactions.Fatalf(err.Error())
	}

	tf, err := tfexec.NewTerraform(workingDirectory, tfPath)
	if err != nil {
		githubactions.Fatalf(err.Error())
	}

	module, diags := tfconfig.LoadModule(workingDirectory)
	if diags.HasErrors() {
		githubactions.Fatalf("failed to load module: %v", diags.Err())
	}

	schemasJson, err := tf.ProvidersSchema(context.TODO())
	if err != nil {
		githubactions.Fatalf("failed to get provider schemas: %v", err)
	}

	core, err := schema.CoreModuleSchemaForConstraint(version.MustConstraints(version.NewConstraint(">= 1")))
	if err != nil {
		githubactions.Fatalf("failed to get core module schema: %v", err)
	}

	merger := schema.NewSchemaMerger(core)
	providerSchemas := localProviderSchemas{}

	for rawAddr, s := range schemasJson.Schemas {
		addr := tfaddr.MustParseProviderSource(rawAddr)
		providerSchema := schema.ProviderSchemaFromJson(s, addr)
		providerSchemas[addr] = providerSchema
	}

	moduleMeta, mDiags := earlydecoder.LoadModule(module.Path, map[string]*hcl.File{})
	if mDiags.HasErrors() {
		githubactions.Fatalf("failed to load module: %v", mDiags.Error())
	}

	merger.SetSchemaReader(providerSchemas)
	bs, err := merger.SchemaForModule(moduleMeta)
	if err != nil {
		githubactions.Fatalf("failed to get merged schema: %v", err)
	}

	bodySchema := bs.ToHCLSchema()

	parser := hclparse.NewParser()

	for _, resource := range resources {
		fmt.Printf("## %s\n", resource.Name)

		table := tablewriter.NewWriter(os.Stdout)

		table.SetHeader(tableHeaders(resource.Attributes))

		table.SetAutoFormatHeaders(false)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")

		for _, r := range module.ManagedResources {
			if r.Type != resource.Name {
				continue
			}

			var file *hcl.File
			var diags hcl.Diagnostics

			switch filename := r.Pos.Filename; filepath.Ext(filename) {
			case ".tf":
				file, diags = parser.ParseHCLFile(filename)
			case ".json":
				file, diags = parser.ParseJSONFile(filename)
			}

			if diags.HasErrors() {
				githubactions.Fatalf("failed to parse file: %v", diags.Error())
			}

			content, _, diags := file.Body.PartialContent(bodySchema)
			if diags.HasErrors() {
				githubactions.Fatalf("failed to parse resource: %v", diags.Error())
			}

			relPath, err := filepath.Rel(workingDirectory, r.Pos.Filename)
			if err != nil {
				githubactions.Fatalf("failed to get relative path: %v", err)
			}

			var resourceBlock *hcl.Block
			for _, block := range content.Blocks {
				if block.Type != "resource" || block.Labels[0] != resource.Name || block.Labels[1] != r.Name {
					continue
				}

				resourceBlock = block
			}

			if resourceBlock == nil {
				githubactions.Fatalf("failed to find resource block")
			}

			row := []string{fmt.Sprintf("[`%s`](%s#L%d)", r.Name, relPath, r.Pos.Line)}
			for _, attr := range resource.Attributes {
				providerSchema := providerSchemas[tfaddr.MustParseProviderSource(module.RequiredProviders[r.Provider.Name].Source)]

				content, _, diags := resourceBlock.Body.PartialContent(providerSchema.Resources[r.Type].ToHCLSchema())
				if diags.HasErrors() {
					githubactions.Fatalf("failed to get attribute value: %v", diags.Error())
				}

				val, diags := content.Attributes[attr].Expr.Value(&hcl.EvalContext{})
				if diags.HasErrors() {
					githubactions.Fatalf("failed to get attribute value: %v", diags.Error())
				}

				// TODO: handle non-string types
				row = append(row, val.AsString())
			}

			table.Append(row)
		}

		table.Render()
		fmt.Println()
	}
}

func getResourceInput() ([]Resource, error) {
	input := githubactions.GetInput("resources")
	var resources []Resource

	err := yaml.Unmarshal([]byte(input), &resources)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resources: %w", err)
	}

	return resources, nil
}

func tableHeaders(attributes []string) []string {
	headers := []string{"**Name**"}

	for _, attribute := range attributes {
		headers = append(headers, fmt.Sprintf("`%s`", attribute))
	}

	return headers
}

type localProviderSchemas map[tfaddr.Provider]*schema.ProviderSchema

func (l localProviderSchemas) ProviderSchema(_ string, addr tfaddr.Provider, _ version.Constraints) (*schema.ProviderSchema, error) {
	s, ok := l[addr]
	if !ok {
		return nil, fmt.Errorf("no schema found for %s", addr)
	}

	return s, nil
}
