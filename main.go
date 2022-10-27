package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/earlydecoder"
	"github.com/hashicorp/terraform-schema/schema"
	"github.com/observeinc/terraform-resource-markdown-table-action/internal/action"
	"github.com/observeinc/terraform-resource-markdown-table-action/internal/terraform"
	"github.com/olekukonko/tablewriter"
	"github.com/sethvargo/go-githubactions"
)

func main() {
	inputs := action.Inputs{
		WorkingDirectory: githubactions.GetInput("working_directory"),
		OutputFile:       githubactions.GetInput("output_file"),
		Resources:        action.ResourcesInput(githubactions.GetInput("resources")),
	}

	resources, err := inputs.Resources.Parse()
	if err != nil {
		githubactions.Fatalf("failed to parse resources: %v", err)
	}

	if err := resources.Validate(); err != nil {
		githubactions.Fatalf("failed to validate resources: %v", err)
	}

	tfPath, err := terraform.EnsureInstalled(context.Background(), version.MustConstraints(version.NewConstraint(">= 1")))
	if err != nil {
		githubactions.Fatalf("failed to ensure terraform is installed: %v", err)
	}

	tf, err := tfexec.NewTerraform(inputs.WorkingDirectory, tfPath)
	if err != nil {
		githubactions.Fatalf(err.Error())
	}

	if err := tf.Init(context.Background()); err != nil {
		githubactions.Fatalf("failed to terraform init: %v", err)
	}

	module, diags := tfconfig.LoadModule(inputs.WorkingDirectory)
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

			relPath, err := filepath.Rel(inputs.WorkingDirectory, r.Pos.Filename)
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
