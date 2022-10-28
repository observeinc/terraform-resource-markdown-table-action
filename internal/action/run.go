package action

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/observeinc/terraform-resource-markdown-table-action/internal/terraform"
)

const (
	BeforeComment = `<!-- BEGIN_TF_RESOURCE_TABLES -->`
	AfterComment  = `<!-- END_TF_RESOURCE_TABLES -->`
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

	var buffer bytes.Buffer
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

		if err := WriteMarkdown(*resourceType, rows, &buffer); err != nil {
			return fmt.Errorf("failed to write markdown: %w", err)
		}
	}

	file, err := os.OpenFile(inputs.OutputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}

	defer file.Close()

	existing, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate output file: %w", err)
	}

	newline := []byte("\n")
	start, end, ok := commentIndexes(existing)
	if !ok {
		file.Write(existing)
		file.Write([]byte(BeforeComment))
		file.Write(newline)
		file.Write(buffer.Bytes())
		file.Write(newline)
		file.Write([]byte(AfterComment))
		file.Write(newline)
		return nil
	}

	file.Write(existing[:start])
	file.Write(newline)
	file.Write(buffer.Bytes())
	file.Write(newline)
	file.Write(existing[end:])
	return nil
}

func commentIndexes(b []byte) (int, int, bool) {
	start := bytes.Index(b, []byte(BeforeComment))
	if start == -1 {
		return 0, 0, false
	}

	end := bytes.Index(b, []byte(AfterComment))
	if end == -1 {
		return 0, 0, false
	}

	if end < start {
		return 0, 0, false
	}

	return start, end, true
}
