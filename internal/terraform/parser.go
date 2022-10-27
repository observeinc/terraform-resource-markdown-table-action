package terraform

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/schema"
	"github.com/zclconf/go-cty/cty"
)

type Parser struct {
	hcl          *hclparse.Parser
	module       *tfconfig.Module
	moduleSchema *hcl.BodySchema
	providers    map[tfaddr.Provider]*schema.ProviderSchema
}

func NewParser() *Parser {
	return &Parser{
		hcl:       hclparse.NewParser(),
		providers: make(map[tfaddr.Provider]*schema.ProviderSchema),
	}
}

func (p *Parser) LoadModule(dir string) error {
	return p.loadModuleFromFilesystem(tfconfig.NewOsFs(), dir)
}

func (p *Parser) loadModuleFromFilesystem(fs tfconfig.FS, dir string) error {
	module, diags := tfconfig.LoadModuleFromFilesystem(fs, dir)
	if diags.HasErrors() {
		return diags.Err()
	}

	p.module = module

	bs, err := schema.CoreModuleSchemaForVersion(schema.LatestAvailableVersion)
	if err != nil {
		return err
	}

	p.moduleSchema = bs.ToHCLSchema()

	return nil
}

func (p *Parser) File(filename string) (*hcl.File, hcl.Diagnostics) {
	if filepath.Ext(filename) == ".json" {
		return p.hcl.ParseJSONFile(filename)
	} else {
		return p.hcl.ParseHCLFile(filename)
	}
}

func (p *Parser) SetProviderSchema(addr tfaddr.Provider, provider *schema.ProviderSchema) {
	p.providers[addr] = provider
}

func (p *Parser) ProviderSchema(addr tfaddr.Provider) *schema.ProviderSchema {
	return p.providers[addr]
}

func (p *Parser) RequiredProviderSource(name string) tfaddr.Provider {
	rp := p.module.RequiredProviders[name]
	return tfaddr.MustParseProviderSource(rp.Source)
}

func (p *Parser) ResourceAttributes(resource *tfconfig.Resource, attributes []string) (map[string]interface{}, error) {
	block, diags := p.ResourceBlock(resource)
	if diags.HasErrors() {
		return nil, diags
	}

	source := p.RequiredProviderSource(resource.Provider.Name)

	ps := p.ProviderSchema(source)
	rs := ps.Resources[resource.Type].ToHCLSchema()

	content, _, diags := block.Body.PartialContent(rs)
	if diags.HasErrors() {
		return nil, diags
	}

	result := make(map[string]interface{}, len(attributes))
	for _, attr := range attributes {
		expr, ok := content.Attributes[attr]
		if !ok {
			return nil, fmt.Errorf("attribute %q not found for resource %s", attr, resource.MapKey())
		}

		value, diags := expr.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to evaluate attribute %q for resource %s: %w", attr, resource.MapKey(), diags)
		}

		if !value.Type().IsPrimitiveType() {
			return nil, fmt.Errorf("attribute %q for resource %s is not a primitive type", attr, resource.MapKey())
		}

		if !value.IsKnown() {
			return nil, fmt.Errorf("attribute %q for resource %s is not known", attr, resource.MapKey())
		}

		switch value.Type() {
		case cty.String:
			result[attr] = value.AsString()
		case cty.Number:
			f, _ := value.AsBigFloat().Float64()
			result[attr] = f
		case cty.Bool:
			result[attr] = value.True()
		default:
			panic("unexpected primitive type") // should never happen
		}
	}

	return result, nil
}

func (p *Parser) ResourceBlock(resource *tfconfig.Resource) (*hcl.Block, hcl.Diagnostics) {
	file, diags := p.File(resource.Pos.Filename)
	if diags.HasErrors() {
		return nil, diags
	}

	content, _, diags := file.Body.PartialContent(p.moduleSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	for _, block := range content.Blocks.OfType("resource") {
		if block.Labels[0] == resource.Type && block.Labels[1] == resource.Name {
			return block, nil
		}
	}

	return nil, hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "resource block not found",
			Detail:   fmt.Sprintf("resource %s.%s not found", resource.Type, resource.Name),
		},
	}
}
