package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
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

	parser, err := terraform.NewParser(schemasJson)
	if err != nil {
		githubactions.Fatalf("failed to create parser: %v", err)
	}

	if err := parser.LoadModule(inputs.WorkingDirectory); err != nil {
		githubactions.Fatalf("failed to load module: %v", err)
	}

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

			relPath, err := filepath.Rel(inputs.WorkingDirectory, r.Pos.Filename)
			if err != nil {
				githubactions.Fatalf("failed to get relative path: %v", err)
			}

			row := []string{fmt.Sprintf("[`%s`](%s#L%d)", r.Name, relPath, r.Pos.Line)}

			attrs, err := parser.ResourceAttributes(r, resource.Attributes)
			if err != nil {
				githubactions.Fatalf("failed to get resource attributes: %v", err)
			}

			for _, attr := range resource.Attributes {
				row = append(row, fmt.Sprintf("%v", attrs[attr]))
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
