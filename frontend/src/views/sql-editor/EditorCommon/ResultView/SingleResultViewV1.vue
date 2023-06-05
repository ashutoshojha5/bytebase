<template>
  <template v-if="viewMode === 'RESULT'">
    <div
      class="w-full shrink-0 flex flex-row justify-between items-center mb-2 overflow-x-auto"
    >
      <div class="flex flex-row justify-start items-center mr-2 shrink-0">
        <NInput
          v-if="showSearchFeature"
          v-model:value="state.search"
          class="!max-w-[8rem] sm:!max-w-xs"
          type="text"
          :placeholder="t('sql-editor.search-results')"
        >
          <template #prefix>
            <heroicons-outline:search class="h-5 w-5 text-gray-300" />
          </template>
        </NInput>
        <span class="ml-2 whitespace-nowrap text-sm text-gray-500">{{
          `${data.length} ${t("sql-editor.rows", data.length)}`
        }}</span>
        <span
          v-if="data.length === RESULT_ROWS_LIMIT"
          class="ml-2 whitespace-nowrap text-sm text-gray-500"
        >
          <span>-</span>
          <span class="ml-2">{{ $t("sql-editor.rows-upper-limit") }}</span>
        </span>
      </div>
      <div class="flex justify-between items-center gap-x-3">
        <NPagination
          v-if="showPagination"
          :item-count="table.getCoreRowModel().rows.length"
          :page="table.getState().pagination.pageIndex + 1"
          :page-size="table.getState().pagination.pageSize"
          :show-quick-jumper="true"
          :show-size-picker="true"
          :page-sizes="[20, 50, 100]"
          @update-page="handleChangePage"
          @update-page-size="(ps) => table.setPageSize(ps)"
        />
        <NButton
          v-if="showVisualizeButton"
          text
          type="primary"
          @click="visualizeExplain"
        >
          {{ $t("sql-editor.visualize-explain") }}
        </NButton>
        <!-- In enterprise plan, we don't allow export data in SQL editor. -->
        <NDropdown
          v-if="showExportButton"
          trigger="hover"
          :options="exportDropdownOptions"
          @select="handleExportBtnClick"
        >
          <NButton>
            <template #icon>
              <heroicons-outline:download class="h-5 w-5" />
            </template>
            {{ t("common.export") }}
          </NButton>
        </NDropdown>
      </div>
    </div>

    <div class="flex-1 w-full flex flex-col overflow-y-auto">
      <DataTable
        ref="dataTable"
        :table="table"
        :columns="columns"
        :data="data"
        :sensitive="sensitive"
        :keyword="state.search"
      />
    </div>
  </template>
  <template v-else-if="viewMode === 'AFFECTED-ROWS'">
    <div
      class="text-md font-normal flex items-center gap-x-1"
      :class="[
        dark ? 'text-[var(--color-matrix-green-hover)]' : 'text-control-light',
      ]"
    >
      <span>{{ extractSQLRowValue(result.rows[0].values[0]) }}</span>
      <span>rows affected</span>
    </div>
  </template>
  <template v-else-if="viewMode === 'EMPTY'">
    <EmptyView />
  </template>
  <template v-else-if="viewMode === 'ERROR'">
    <ErrorView :error="result.error" />
  </template>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { NPagination } from "naive-ui";
import { useI18n } from "vue-i18n";
import { debouncedRef } from "@vueuse/core";
import {
  ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { isEmpty } from "lodash-es";
import { unparse } from "papaparse";
import dayjs from "dayjs";

import {
  createExplainToken,
  extractSQLRowValue,
  instanceV1HasStructuredQueryResult,
} from "@/utils";
import {
  useInstanceV1Store,
  useTabStore,
  RESULT_ROWS_LIMIT,
  featureToRef,
  useCurrentUserIamPolicy,
  pushNotification,
  useDatabaseV1Store,
} from "@/store";
import DataTable from "./DataTable";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView.vue";
import { useSQLResultViewContext } from "./context";
import { Engine } from "@/types/proto/v1/common";
import { QueryResult } from "@/types/proto/v1/sql_service";
import { SQLResultSetV1 } from "@/types";

type LocalState = {
  search: string;
};
type ViewMode = "RESULT" | "EMPTY" | "AFFECTED-ROWS" | "ERROR";

const PAGE_SIZES = [20, 50, 100];
const DEFAULT_PAGE_SIZE = 50;

const props = defineProps<{
  result: QueryResult;
}>();

const state = reactive<LocalState>({
  search: "",
});

const { dark } = useSQLResultViewContext();

const { t } = useI18n();
const tabStore = useTabStore();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const dataTable = ref<InstanceType<typeof DataTable>>();

const viewMode = computed((): ViewMode => {
  const { result } = props;
  if (result.error) {
    return "ERROR";
  }
  const columnNames = result.columnNames;
  if (columnNames?.length === 0) {
    return "EMPTY";
  }
  if (columnNames?.length === 1 && columnNames[0] === "Affected Rows") {
    return "AFFECTED-ROWS";
  }
  return "RESULT";
});

const showSearchFeature = computed(() => {
  const instance = instanceStore.getInstanceByUID(
    tabStore.currentTab.connection.instanceId
  );
  return instanceV1HasStructuredQueryResult(instance);
});

// show export button only when the subscription is not enterprise.
// In enterprise plan, all users need to fill in the export data request issue.
const showExportButton = computed(() => {
  return !featureToRef("bb.feature.custom-role").value;
});

const allowToExportData = computed(() => {
  const database = databaseStore.getDatabaseByUID(
    tabStore.currentTab.connection.databaseId
  );
  return useCurrentUserIamPolicy().allowToExportDatabaseV1(database);
});

// use a debounced value to improve performance when typing rapidly
const keyword = debouncedRef(
  computed(() => state.search),
  200
);

const columns = computed(() => {
  const columns = props.result.columnNames;
  return columns.map<ColumnDef<string[]>>((col, index) => ({
    id: `${col}@${index}`,
    accessorFn: (item) => item[index],
    header: col,
  }));
});

const convertedData = computed(() => {
  const rows = props.result.rows;
  return rows.map((row) => {
    return row.values.map((value) => extractSQLRowValue(value));
  });
});

const data = computed(() => {
  const data = convertedData.value;
  const search = keyword.value.trim().toLowerCase();
  let temp = data;
  if (search) {
    temp = data.filter((item) => {
      return item.some((col) => String(col).toLowerCase().includes(search));
    });
  }
  return temp;
});

const sensitive = computed(() => {
  return props.result.masked;
});

const table = useVueTable<string[]>({
  get data() {
    return data.value;
  },
  get columns() {
    return columns.value;
  },
  getCoreRowModel: getCoreRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
});

table.setPageSize(DEFAULT_PAGE_SIZE);

const exportDropdownOptions = computed(() => [
  {
    label: t("sql-editor.download-as-csv"),
    key: "csv",
    disabled: props.result === null || isEmpty(props.result),
  },
  {
    label: t("sql-editor.download-as-json"),
    key: "json",
    disabled: props.result === null || isEmpty(props.result),
  },
]);

const handleExportBtnClick = (format: "csv" | "json") => {
  if (!allowToExportData.value) {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: "You don't have permission to export data.",
    });
    return;
  }

  let rawText = "";

  if (format === "csv") {
    const csvFields = columns.value.map((col) => col.header as string);
    const csvData = data.value.map((d) => {
      const temp: any[] = [];
      for (const k in d) {
        temp.push(d[k]);
      }
      return temp;
    });

    rawText = unparse({
      fields: csvFields,
      data: csvData,
    });
  } else {
    const objects = [];
    for (const item of data.value) {
      const object = {} as any;
      for (let i = 0; i < columns.value.length; i++) {
        object[columns.value[i].header as string] = item[i];
      }
      objects.push(object);
    }
    rawText = JSON.stringify(objects);
  }

  const encodedUri = encodeURI(`data:text/${format};charset=utf-8,${rawText}`);
  const formattedDateString = dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss");
  // Example filename: `mysheet-2022-03-23T09-54-21.json`
  const filename = `${tabStore.currentTab.name}-${formattedDateString}`;
  const link = document.createElement("a");

  link.download = `${filename}.${format}`;
  link.href = encodedUri;
  link.click();
};

const showVisualizeButton = computed((): boolean => {
  const instance = instanceStore.getInstanceByUID(
    tabStore.currentTab.connection.instanceId
  );
  const databaseType = instance.engine;
  const { executeParams } = tabStore.currentTab;
  return databaseType === Engine.POSTGRES && !!executeParams?.option?.explain;
});

const visualizeExplain = () => {
  try {
    const { executeParams, sqlResultSet } = tabStore.currentTab;
    if (!executeParams || !sqlResultSet) return;

    const statement = executeParams.query || "";
    if (!statement) return;

    const explain = explainFromSQLResultSetV1(sqlResultSet);
    if (!explain) return;

    const token = createExplainToken(statement, explain);

    window.open(`/explain-visualizer.html?token=${token}`, "_blank");
  } catch {
    // nothing
  }
};

const showPagination = computed(() => data.value.length > PAGE_SIZES[0]);

const handleChangePage = (page: number) => {
  table.setPageIndex(page - 1);
  dataTable.value?.scrollTo(0, 0);
};

const explainFromSQLResultSetV1 = (resultSet: SQLResultSetV1 | undefined) => {
  if (!resultSet) return "";
  const lines = resultSet.results[0].rows.map((row) =>
    row.values.map((value) => String(extractSQLRowValue(value)))
  );
  const explain = lines.map((line) => line[0]).join("\n");
  return explain;
};
</script>