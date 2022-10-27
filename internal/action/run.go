package action

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/observeinc/terraform-resource-markdown-table-action/internal/terraform"
)

func Run(ctx context.Context, inputs Inputs) error {
	resourceTypes, err := inputs.ResourceTypes.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse resources: %w", err)
	}

	if err := resourceTypes.Validate(); err != nil {
		return fmt.Errorf("failed to validate resource types input: %w", err)
	}

	tfPath, err := terraform.EnsureInstalled(ctx, version.MustConstraints(version.NewConstraint(">= 1")))
	if err != nil {
		return fmt.Errorf("failed to ensure terraform is installed: %w", err)
	}

	tf, err := tfexec.NewTerraform(inputs.WorkingDirectory, tfPath)
	if err != nil {
		return fmt.Errorf("failed to create terraform exec: %w", err)
	}

	if err := tf.Init(ctx); err != nil {
		return fmt.Errorf("failed to terraform init: %w", err)
	}

	schemas, err := tf.ProvidersSchema(ctx)
	if err != nil {
		return fmt.Errorf("failed to get provider schemas: %w", err)
	}

	parser, err := terraform.NewParser(schemas)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	if err := parser.LoadModule(inputs.WorkingDirectory); err != nil {
		return fmt.Errorf("failed to load module: %w", err)
	}

	for _, resourceType := range resourceTypes {
		rows := []*ResourceRow{}
		for _, resource := range parser.ResourcesOfType(resourceType.Name) {
			attrs, err := parser.ResourceAttributes(resource.Type, resource.Name, resourceType.Attributes)
			if err != nil {
				return fmt.Errorf("failed to parse resource attributes for %s: %w", resource.MapKey(), err)
			}

			row := &ResourceRow{
				Name:       resource.Name,
				Position:   resource.Pos,
				Attributes: attrs,
			}

			rows = append(rows, row)
		}

		if err := WriteMarkdown(*resourceType, rows, os.Stdout); err != nil {
			return fmt.Errorf("failed to write markdown: %w", err)
		}
	}

	return nil
}
