package action

import (
	"fmt"
	"io"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/olekukonko/tablewriter"
)

type ResourceRow struct {
	Name       string
	Position   tfconfig.SourcePos
	Attributes map[string]interface{}
}

func WriteMarkdown(resource TerraformResourceType, rows []*ResourceRow, writer io.Writer) error {
	if _, err := writer.Write([]byte(fmt.Sprintf("## %s\n\n", resource.Name))); err != nil {
		return err
	}

	table := tablewriter.NewWriter(writer)

	table.SetHeader(tableHeaders(resource.Attributes))

	table.SetAutoFormatHeaders(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetAutoWrapText(false)

	for _, row := range rows {
		table.Append(tableRow(resource, row))
	}

	table.Render()
	return nil
}

func tableRow(resource TerraformResourceType, data *ResourceRow) []string {
	row := []string{fmt.Sprintf("[`%s`](%s#L%d)", data.Name, data.Position.Filename, data.Position.Line)}

	for _, key := range resource.Attributes {
		value := data.Attributes[key]

		if value == nil {
			row = append(row, "_unknown_")
			continue
		}

		row = append(row, fmt.Sprintf("%v", value))
	}

	return row
}

func tableHeaders(attributes []string) []string {
	headers := []string{"**Name**"}

	for _, attribute := range attributes {
		headers = append(headers, fmt.Sprintf("`%s`", attribute))
	}

	return headers
}
