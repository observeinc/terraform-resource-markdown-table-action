package main

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/sethvargo/go-githubactions"
	"gopkg.in/yaml.v3"
)

type Resource struct {
	Name       string   `yaml:"name"`
	Attributes []string `yaml:"attributes"`
}

func main() {
	resources, err := getResourceInput()
	if err != nil {
		githubactions.Fatalf(err.Error())
	}

	if len(resources) == 0 {
		githubactions.Fatalf("No resources found")
	}

	for _, resource := range resources {
		fmt.Printf("## %s\n", resource.Name)

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader(tableHeaders(resource.Attributes))
		table.SetAutoFormatHeaders(false)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
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
