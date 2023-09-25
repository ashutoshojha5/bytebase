<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    :close-on-esc="true"
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <NDrawerContent
      :title="$t('schema-designer.merge-branch')"
      :closable="true"
    >
      <div
        class="space-y-3 w-[calc(100vw-24rem)] min-w-[64rem] max-w-[calc(100vw-8rem)] h-full overflow-x-auto"
      >
        <div class="w-full flex flex-row justify-end items-center gap-2">
          <NButton @click="() => handleSaveDraft()">{{
            $t("schema-designer.save-draft")
          }}</NButton>
          <NButton type="primary" @click="handleMergeBranch">
            {{ $t("common.merge") }}
          </NButton>
        </div>
        <div class="w-full flex flex-row justify-end items-center gap-2">
          <NCheckbox v-model:checked="state.deleteBranchAfterMerged">
            {{ $t("schema-designer.delete-branch-after-merged") }}
          </NCheckbox>
        </div>
        <div class="w-full pr-12 pt-4 pb-6 grid grid-cols-3">
          <div class="flex flex-row justify-end">
            <NInput
              class="!w-4/5 text-center"
              readonly
              :value="targetBranch.title"
              size="large"
            />
          </div>
          <div class="flex flex-row justify-center">
            <MoveLeft :size="40" stroke-width="1" />
          </div>
          <div class="flex flex-row justify-start">
            <NInput
              class="!w-4/5 text-center"
              readonly
              :value="sourceBranch.title"
              size="large"
            />
          </div>
        </div>
        <div class="w-full grid grid-cols-2">
          <div class="col-span-1">
            <span>{{ $t("schema-designer.diff-editor.latest-schema") }}</span>
          </div>
          <div class="col-span-1">
            <span>{{ $t("schema-designer.diff-editor.editing-schema") }}</span>
          </div>
        </div>
        <div class="w-full h-[calc(100%-14rem)] border">
          <DiffEditor
            v-if="state.initialized"
            class="h-full"
            :original="targetBranch.schema"
            :value="state.editingSchema"
            @change="state.editingSchema = $event"
          />
        </div>
      </div>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { MoveLeft } from "lucide-vue-next";
import {
  NButton,
  NDrawer,
  NDrawerContent,
  NInput,
  NCheckbox,
  useDialog,
} from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import DiffEditor from "@/components/MonacoEditor/DiffEditor.vue";
import { pushNotification, useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import {
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";

interface LocalState {
  editingSchema: string;
  initialized: boolean;
  deleteBranchAfterMerged: boolean;
}

const props = defineProps<{
  sourceBranchName: string;
  targetBranchName: string;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "merged", branchName: string): void;
}>();

const state = reactive<LocalState>({
  editingSchema: "",
  initialized: false,
  deleteBranchAfterMerged: true,
});
const { t } = useI18n();
const dialog = useDialog();
const sheetStore = useSheetV1Store();
const schemaDesignStore = useSchemaDesignStore();

const sourceBranch = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(props.sourceBranchName || "");
});

const targetBranch = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(props.targetBranchName || "");
});

onMounted(async () => {
  // Fetching the latest source branch.
  await schemaDesignStore.fetchSchemaDesignByName(props.targetBranchName);
  state.editingSchema = sourceBranch.value.schema;
  state.initialized = true;
});

const handleSaveDraft = async (ignoreNotify?: boolean) => {
  const updateMask = ["schema", "baseline_sheet_name"];
  const [projectName] = getProjectAndSchemaDesignSheetId(
    sourceBranch.value.name
  );
  // Create a baseline sheet for the schema design.
  const baselineSheet = await sheetStore.createSheet(
    `projects/${projectName}`,
    {
      title: `baseline schema of ${targetBranch.value.title}`,
      database: targetBranch.value.baselineDatabase,
      content: new TextEncoder().encode(targetBranch.value.schema),
      visibility: Sheet_Visibility.VISIBILITY_PROJECT,
      source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
      type: Sheet_Type.TYPE_SQL,
    }
  );

  // Update the schema design draft first.
  await schemaDesignStore.updateSchemaDesign(
    SchemaDesign.fromPartial({
      name: sourceBranch.value.name,
      engine: sourceBranch.value.engine,
      schema: state.editingSchema,
      baselineSheetName: baselineSheet.name,
    }),
    updateMask
  );

  if (!ignoreNotify) {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
    emit("dismiss");
  }
};

const handleMergeBranch = async () => {
  await handleSaveDraft(true);

  try {
    await schemaDesignStore.mergeSchemaDesign({
      name: sourceBranch.value.name,
      targetName: targetBranch.value.name,
    });
  } catch (error: any) {
    // If there is conflict, we need to show the conflict and let user resolve it.
    if (error.code === Status.FAILED_PRECONDITION) {
      dialog.create({
        negativeText: t("schema-designer.save-draft"),
        positiveText: t("schema-designer.diff-editor.resolve"),
        title: t("schema-designer.diff-editor.auto-merge-failed"),
        content: t("schema-designer.diff-editor.need-to-resolve-conflicts"),
        autoFocus: true,
        closable: true,
        maskClosable: true,
        closeOnEsc: true,
        onNegativeClick: () => {
          // Do nothing.
        },
        onPositiveClick: async () => {
          // Fetching the latest target branch.
          await schemaDesignStore.fetchSchemaDesignByName(
            props.targetBranchName
          );
        },
      });
    } else {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Request error occurred`,
        description: error.details,
      });
    }
    return;
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.merge-to-main-successfully"),
  });

  emit("merged", props.targetBranchName);
  if (state.deleteBranchAfterMerged) {
    await schemaDesignStore.deleteSchemaDesign(props.sourceBranchName);
  }
};
</script>