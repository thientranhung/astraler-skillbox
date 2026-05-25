package app

// Compile-time checks: concrete repository types must satisfy the service interfaces
// used by ProjectService. If any method signature drifts, this file fails to compile.

import (
	"github.com/astraler/skillbox/core-go/internal/repositories"
	"github.com/astraler/skillbox/core-go/internal/services"
)

var (
	_ services.ProjectRepo         = (*repositories.ProjectRepo)(nil)
	_ services.ProjectProviderRepo = (*repositories.ProjectProviderRepo)(nil)
	_ services.ProjectWarningRepo  = (*repositories.WarningRepo)(nil)
	_ services.ProjectInstallRepo  = (*repositories.InstallRepo)(nil)
)
