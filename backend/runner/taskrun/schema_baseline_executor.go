package taskrun

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/ashutoshojha5/bytebase/backend/common/log"
	"github.com/ashutoshojha5/bytebase/backend/component/activity"
	"github.com/ashutoshojha5/bytebase/backend/component/config"
	"github.com/ashutoshojha5/bytebase/backend/component/dbfactory"
	"github.com/ashutoshojha5/bytebase/backend/component/state"
	enterprise "github.com/ashutoshojha5/bytebase/backend/enterprise/api"
	api "github.com/ashutoshojha5/bytebase/backend/legacyapi"
	"github.com/ashutoshojha5/bytebase/backend/plugin/db"
	"github.com/ashutoshojha5/bytebase/backend/runner/schemasync"
	"github.com/ashutoshojha5/bytebase/backend/store"
	"github.com/ashutoshojha5/bytebase/backend/store/model"
	v1pb "github.com/ashutoshojha5/bytebase/proto/generated-go/v1"
)

// NewSchemaBaselineExecutor creates a schema baseline task executor.
func NewSchemaBaselineExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &SchemaBaselineExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		license:         license,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// SchemaBaselineExecutor is the schema baseline task executor.
type SchemaBaselineExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	license         enterprise.LicenseService
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run the schema update (DDL) task executor once.
func (exec *SchemaBaselineExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *api.TaskRunResultPayload, error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

	payload := &api.TaskDatabaseSchemaBaselinePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema baseline payload")
	}

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}

	version := model.Version{Version: payload.SchemaVersion}
	terminated, result, err := runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.activityManager, exec.license, exec.stateCfg, exec.profile, task, taskRunUID, db.Baseline, "" /* statement */, version, nil /* sheetID */)
	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	return terminated, result, err
}
