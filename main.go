package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/olekukonko/tablewriter"
	"github.com/sethvargo/go-githubactions"
	"gopkg.in/yaml.v3"
)

type Resource struct {
	Name       string   `yaml:"name"`
	Attributes []string `yaml:"attributes"`
}

func main() {
	workingDirectory := githubactions.GetInput("working-directory")

	resources, err := getResourceInput()
	if err != nil {
		githubactions.Fatalf(err.Error())
	}

	if len(resources) == 0 {
		githubactions.Fatalf("No resources found")
	}

	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		githubactions.Fatalf("terraform not found in PATH")
	}

	tf, err := tfexec.NewTerraform(workingDirectory, tfPath)
	if err != nil {
		githubactions.Fatalf(err.Error())
	}

	schemas, err := tf.ProvidersSchema(context.TODO())
	if err != nil {
		githubactions.Fatalf("failed to get provider schemas: %v", err)
	}

	_ = schemas

	module, diags := tfconfig.LoadModule(workingDirectory)
	if diags.HasErrors() {
		githubactions.Fatalf("failed to load module: %v", diags.Err())
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

			relPath, err := filepath.Rel(workingDirectory, r.Pos.Filename)
			if err != nil {
				githubactions.Fatalf("failed to get relative path: %v", err)
			}

			row := []string{fmt.Sprintf("[`%s`](%s#L%d)", r.Name, relPath, r.Pos.Line)}
			for range resource.Attributes {
				// TODO: get attribute value
				row = append(row, "-")
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
