import { TemplateType } from "@/plugins";
import {
  IssueCreate,
  IssueType,
  MigrationType,
  MigrationContext,
} from "@/types";
import {
  findProject,
  BuildNewIssueContext,
  VALIDATE_ONLY_SQL,
  findDatabaseListByQuery,
} from "../common";
import { IssueCreateHelper } from "./helper";

export const maybeBuildTenantDeployIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate | undefined> => {
  const { route } = context;
  if (route.query.mode !== "tenant") {
    return undefined;
  }

  const project = await findProject(context);
  const issueType = route.query.template as IssueType;
  const isMigrate =
    issueType === "bb.issue.database.schema.update" ||
    issueType === "bb.issue.database.data.update";
  if (project.tenantMode === "TENANT" && isMigrate) {
    // Only to build tenant issue when:
    // 1. Project is tenant mode.
    // 2. Is schema update or data update (no to establish baseline).
    return buildNewTenantSchemaUpdateIssue(context);
  }
  return undefined;
};

const buildNewTenantSchemaUpdateIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate> => {
  const { route } = context;
  const helper = new IssueCreateHelper(context);

  await helper.prepare();
  // tenant issue is generated by specifying databaseName
  const templateType = route.query.template as TemplateType;
  let migrationType: MigrationType = "MIGRATE";
  if (templateType === "bb.issue.database.data.update") {
    migrationType = "DATA";
  }

  const databaseList = findDatabaseListByQuery(context);
  if (databaseList.length !== 0) {
    helper.issueCreate!.createContext = {
      detailList: databaseList.map((db) => {
        return {
          migrationType: migrationType,
          databaseId: db.id,
          statement: VALIDATE_ONLY_SQL,
        };
      }),
    };
  } else {
    helper.issueCreate!.createContext = {
      detailList: [
        {
          migrationType: migrationType,
          statement: VALIDATE_ONLY_SQL,
        },
      ],
    };
  }
  await helper.validate();

  // we are setting SQL statement to "" because showing /* YOUR_SQL_HERE */
  // is not that friendly to users
  // setting it to empty can provide a placeholder to user, along with an
  // exclamation mark indicating "No SQL statement"
  const createContext = helper.issueCreate!.createContext as MigrationContext;
  for (const detail of createContext.detailList) {
    detail.statement = "";
  }

  return helper.generate();
};
