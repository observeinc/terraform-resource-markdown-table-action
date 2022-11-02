package action

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/observeinc/terraform-resource-markdown-table-action/internal/terraform"
	"github.com/olekukonko/tablewriter"
)

type ResourceRow struct {
	Name       string
	Position   tfconfig.SourcePos
	Attributes map[string]interface{}
}

func WriteMarkdown(dir string, resource TerraformResourceType, rows []*ResourceRow, headerLevel int, writer io.Writer) error {
	if _, err := writer.Write([]byte(fmt.Sprintf("%s %s\n\n", strings.Repeat("#", headerLevel), resource.Name))); err != nil {
		return err
	}

	table := tablewriter.NewWriter(writer)

	table.SetHeader(tableHeaders(resource.Attributes))

	table.SetAutoFormatHeaders(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetAutoWrapText(false)

	for _, row := range rows {
		r, err := tableRow(dir, resource, row)
		if err != nil {
			return err
		}

		table.Append(r)
	}

	table.Render()
	return nil
}

func tableRow(dir string, resource TerraformResourceType, data *ResourceRow) ([]string, error) {
	filename, err := filepath.Rel(dir, data.Position.Filename)
	if err != nil {
		return nil, err
	}

	row := []string{fmt.Sprintf("[`%s`](%s#L%d)", data.Name, filename, data.Position.Line)}

	for _, key := range resource.Attributes {
		value := data.Attributes[key]
		row = append(row, ValueToMarkdown(value))
	}

	return row, nil
}

func tableHeaders(attributes []string) []string {
	headers := []string{"**Name**"}

	for _, attribute := range attributes {
		headers = append(headers, fmt.Sprintf("`%s`", attribute))
	}

	return headers
}

func ValueToMarkdown(value interface{}) string {
	if value == nil {
		return ""
	}

	if _, ok := value.(*terraform.UnknownAttributeValue); ok {
		return "_unknown_"
	}

	return fmt.Sprintf("%v", value)
}
