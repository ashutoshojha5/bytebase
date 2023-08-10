<template>
  <div class="flex items-center flex-wrap gap-2 flex-1">
    <PlanCheckBadge
      v-for="group in planCheckRunsGroupByType"
      :key="group.type"
      :type="group.type"
      :plan-check-run-list="group.list"
      @click="$emit('select-type', group.type)"
    />
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { computed } from "vue";
import {
  PlanCheckRun,
  PlanCheckRun_Type,
  Task,
} from "@/types/proto/v1/rollout_service";
import { groupBy } from "@/utils/collections";
import PlanCheckBadge from "./PlanCheckBadge.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  selectedType?: PlanCheckRun_Type;
  task: Task;
}>();

defineEmits<{
  (event: "select-type", type: PlanCheckRun_Type): void;
}>();

const planCheckRunsGroupByType = computed(() => {
  const groups = groupBy(props.planCheckRunList, (checkRun) => checkRun.type);
  const list = Array.from(groups.entries()).map(([type, list]) => ({
    type,
    list,
  }));
  // Sort by pre-defined orders
  // If an item's order is not defined, put it behind
  return orderBy(
    list,
    [(group) => PlanCheckTypeOrderDict.get(group.type) ?? 99999],
    "asc"
  );
});

const PlanCheckTypeOrderList: PlanCheckRun_Type[] = [
  PlanCheckRun_Type.DATABASE_PITR_MYSQL,
  PlanCheckRun_Type.DATABASE_GHOST_SYNC,
  PlanCheckRun_Type.DATABASE_STATEMENT_COMPATIBILITY,
  PlanCheckRun_Type.DATABASE_STATEMENT_TYPE,
  PlanCheckRun_Type.DATABASE_CONNECT,
  PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE,
];
const PlanCheckTypeOrderDict = new Map<PlanCheckRun_Type, number>(
  PlanCheckTypeOrderList.map((type, order) => [type, order])
);

// import { computed, defineComponent, PropType, reactive, watch } from "vue";
// import { groupBy, maxBy } from "lodash-es";
// import { useI18n } from "vue-i18n";

// import { HiddenCheckTypes } from "@/utils";

// interface LocalState {
//   selectedTaskCheckType: TaskCheckType | undefined;
// }

// export default defineComponent({
//   name: "TaskCheckBadgeBar",
//   components: { TaskSpinner },
//   props: {
//     taskCheckRunList: {
//       required: true,
//       type: Object as PropType<TaskCheckRun[]>,
//     },
//     allowSelection: {
//       default: true,
//       type: Boolean,
//     },
//     stickySelection: {
//       default: false,
//       type: Boolean,
//     },
//     selectedTaskCheckType: {
//       type: String as PropType<TaskCheckType>,
//       default: undefined,
//     },
//   },
//   emits: ["select-task-check-type"],
//   setup(props, { emit }) {
//     const { t } = useI18n();
//     const state = reactive<LocalState>({
//       selectedTaskCheckType: props.selectedTaskCheckType,
//     });

//     watch(
//       () => props.selectedTaskCheckType,
//       (curNew, _) => {
//         state.selectedTaskCheckType = curNew;
//       }
//     );

//     const buttonStyle = (checkRun: TaskCheckRun): string => {
//       let bgColor = "";
//       let bgHoverColor = "";
//       let textColor = "";
//       switch (checkRun.status) {
//         case "RUNNING":
//           bgColor = "bg-blue-100";
//           bgHoverColor = "bg-blue-300";
//           textColor = "text-blue-800";
//           break;
//         case "FAILED":
//           bgColor = "bg-red-100";
//           bgHoverColor = "bg-red-300";
//           textColor = "text-red-800";
//           break;
//         case "CANCELED":
//           bgColor = "bg-yellow-100";
//           bgHoverColor = "bg-yellow-300";
//           textColor = "text-yellow-800";
//           break;
//         case "DONE":
//           switch (taskCheckStatus(checkRun)) {
//             case "SUCCESS":
//               bgColor = "bg-gray-100";
//               bgHoverColor = "bg-gray-300";
//               textColor = "text-gray-800";
//               break;
//             case "WARN":
//               bgColor = "bg-yellow-100";
//               bgHoverColor = "bg-yellow-300";
//               textColor = "text-yellow-800";
//               break;
//             case "ERROR":
//               bgColor = "bg-red-100";
//               bgHoverColor = "bg-red-300";
//               textColor = "text-red-800";
//               break;
//           }
//           break;
//       }

//       const styleList: string[] = [textColor];
//       if (props.allowSelection) {
//         styleList.push("cursor-pointer", `hover:${bgHoverColor}`);
//         if (
//           props.stickySelection &&
//           checkRun.type == state.selectedTaskCheckType
//         ) {
//           styleList.push(bgHoverColor);
//         } else {
//           styleList.push(bgColor);
//         }
//       } else {
//         styleList.push(bgColor);
//         styleList.push("cursor-default");
//       }

//       return styleList.join(" ");
//     };

//     // For a particular check type, only returns the most recent one
//     const filteredTaskCheckRunList = computed((): TaskCheckRun[] => {
//       const groupByType = groupBy(
//         props.taskCheckRunList.filter((run) => !HiddenCheckTypes.has(run.type)),
//         (run) => run.type
//       );
//       /*
//         `groupByType` looks like: {
//           "bb.task-check.database.statement.compatibility": [run1, run2, ...],
//           "bb.task-check.database.statement.syntax": [run1, run2, ...],
//           ...
//         }
//         `result` is an array of the most recent TaskCheckRun in each group
//       */
//       const result = Object.keys(groupByType).map((type) => {
//         const groupList = groupByType[type];
//         const mostRecentInGroup = maxBy(groupList, (run) => run.updatedTs)!;
//         return mostRecentInGroup;
//       });

//       return result.sort((a: TaskCheckRun, b: TaskCheckRun) => {
//         const taskCheckRunTypeOrder = (type: TaskCheckType) => {
//           const has = TaskCheckTypeOrderDict.has(type);
//           console.assert(has, `Missing TaskCheckType order of "${type}"`);
//           if (has) {
//             return TaskCheckTypeOrderDict.get(type)!;
//           }
//           // Fallback, types not defined in the dictionary will go to the tail.
//           return FAKE_MAX_TASK_CHECK_TYPE_ORDER;
//         };

//         return taskCheckRunTypeOrder(a.type) - taskCheckRunTypeOrder(b.type);
//       });
//     });

//     // Returns the most severe status
//     const taskCheckStatus = (taskCheckRun: TaskCheckRun): TaskCheckStatus => {
//       let value: TaskCheckStatus = "SUCCESS";
//       for (const result of taskCheckRun.result.resultList ?? []) {
//         if (result.status == "ERROR") {
//           return "ERROR";
//         }
//         if (result.status == "WARN") {
//           value = "WARN";
//         }
//       }
//       return value;
//     };

//     const name = (taskCheckRun: TaskCheckRun): string => {
//       const { type } = taskCheckRun;
//       const has = TaskCheckTypeNameDict.has(type);
//       console.assert(has, `Missing TaskCheckType name of "${type}"`);
//       if (has) {
//         const key = TaskCheckTypeNameDict.get(type)!;
//         return t(key);
//       }
//       return type;
//     };

//     const selectTaskCheckType = (type: TaskCheckType) => {
//       emit("select-task-check-type", type);
//     };

//     return {
//       state,
//       buttonStyle,
//       filteredTaskCheckRunList,
//       taskCheckStatus,
//       name,
//       selectTaskCheckType,
//     };
//   },
// });

// // Defines the order of TaskCheckType
// const TaskCheckTypeOrderList: TaskCheckType[] = [
//   "bb.task-check.pitr.mysql",
//   "bb.task-check.database.ghost.sync",
//   "bb.task-check.database.statement.compatibility",
//   "bb.task-check.database.statement.syntax",
//   "bb.task-check.database.statement.type",
//   "bb.task-check.database.connect",
//   "bb.task-check.database.statement.advise",
//   "bb.task-check.issue.lgtm",
//   "bb.task-check.database.statement.affected-rows.report",
//   "bb.task-check.database.statement.type.report",
// ];
// const TaskCheckTypeOrderDict = new Map<TaskCheckType, number>(
//   TaskCheckTypeOrderList.map((type, index) => [type, index])
// );
// const FAKE_MAX_TASK_CHECK_TYPE_ORDER = 100;
// TaskCheckTypeOrderDict.set(
//   "bb.task-check.database.statement.fake-advise",
//   FAKE_MAX_TASK_CHECK_TYPE_ORDER
// );

// // Defines the mapping from TaskCheckType to an i18n resource keypath
// const TaskCheckTypeNameDict = new Map<TaskCheckType, string>([
//   ["bb.task-check.database.statement.fake-advise", "task.check-type.fake"],
//   ["bb.task-check.database.statement.syntax", "task.check-type.syntax"],
//   [
//     "bb.task-check.database.statement.compatibility",
//     "task.check-type.compatibility",
//   ],
//   ["bb.task-check.database.statement.advise", "task.check-type.sql-review"],
//   ["bb.task-check.database.statement.type", "task.check-type.statement-type"],
//   ["bb.task-check.database.connect", "task.check-type.connection"],
//   ["bb.task-check.database.ghost.sync", "task.check-type.ghost-sync"],
//   ["bb.task-check.issue.lgtm", "task.check-type.lgtm"],
//   ["bb.task-check.pitr.mysql", "task.check-type.pitr"],
//   [
//     "bb.task-check.database.statement.affected-rows.report",
//     "task.check-type.affected-rows",
//   ],
//   ["bb.task-check.database.statement.type.report", "task.check-type.sql-type"],
// ]);
</script>