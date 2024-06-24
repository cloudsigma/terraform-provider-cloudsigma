//go:build tools

package tools

import (
	// changelog generation (git-chglog)
	_ "github.com/git-chglog/git-chglog/cmd/git-chglog"
	// source code linting (golangci-lint)
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	// documentation generation and validation (tfplugindocs)
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
