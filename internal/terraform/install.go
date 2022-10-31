package terraform

import (
	"context"

	"github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
)

func EnsureInstalled(ctx context.Context, constraints version.Constraints) (string, error) {
	installer := install.NewInstaller()

	return installer.Ensure(ctx, []src.Source{
		&fs.AnyVersion{
			Product: &product.Terraform,
		},
		&releases.LatestVersion{
			Product:     product.Terraform,
			Constraints: constraints,
		},
	})
}
