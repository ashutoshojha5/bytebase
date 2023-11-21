package taskrun

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	v1pb "github.com/ashutoshojha5/bytebase/proto/generated-go/v1"

	"github.com/ashutoshojha5/bytebase/backend/component/activity"
	"github.com/ashutoshojha5/bytebase/backend/component/config"
	"github.com/ashutoshojha5/bytebase/backend/component/dbfactory"
	"github.com/ashutoshojha5/bytebase/backend/component/state"
	enterprise "github.com/ashutoshojha5/bytebase/backend/enterprise/api"

	api "github.com/ashutoshojha5/bytebase/backend/legacyapi"
	"github.com/ashutoshojha5/bytebase/backend/plugin/db"
	"github.com/ashutoshojha5/bytebase/backend/store"
	"github.com/ashutoshojha5/bytebase/backend/store/model"
)

// NewDataUpdateExecutor creates a data update (DML) task executor.
func NewDataUpdateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterprise.LicenseService, stateCfg *state.State, profile config.Profile) Executor {
	return &DataUpdateExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		license:         license,
		stateCfg:        stateCfg,
		profile:         profile,
	}
}

// DataUpdateExecutor is the data update (DML) task executor.
type DataUpdateExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	license         enterprise.LicenseService
	stateCfg        *state.State
	profile         config.Profile
}

// RunOnce will run the data update (DML) task executor once.
func (exec *DataUpdateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *api.TaskRunResultPayload, err error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database data update payload")
	}

	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, err
	}
	version := model.Version{Version: payload.SchemaVersion}
	return runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.activityManager, exec.license, exec.stateCfg, exec.profile, task, taskRunUID, db.Data, statement, version, &payload.SheetID)
}
