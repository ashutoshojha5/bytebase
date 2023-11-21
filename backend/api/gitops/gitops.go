// Package gitops is the package for GitOps APIs.
package gitops

import (
	v1pb "github.com/ashutoshojha5/bytebase/backend/api/v1"
	"github.com/ashutoshojha5/bytebase/backend/component/activity"
	"github.com/ashutoshojha5/bytebase/backend/component/dbfactory"
	"github.com/ashutoshojha5/bytebase/backend/component/state"
	enterprise "github.com/ashutoshojha5/bytebase/backend/enterprise/api"
	"github.com/ashutoshojha5/bytebase/backend/store"
)

// Service is the API endpoint for handling GitOps requests.
type Service struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	stateCfg        *state.State
	licenseService  enterprise.LicenseService
	rolloutService  *v1pb.RolloutService
	issueService    *v1pb.IssueService
}

// NewService creates a GitOps service.
func NewService(
	store *store.Store,
	dbFactory *dbfactory.DBFactory,
	activityManager *activity.Manager,
	stateCfg *state.State,
	licenseService enterprise.LicenseService,
	rolloutService *v1pb.RolloutService,
	issueService *v1pb.IssueService,
) *Service {
	return &Service{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		stateCfg:        stateCfg,
		licenseService:  licenseService,
		rolloutService:  rolloutService,
		issueService:    issueService,
	}
}
