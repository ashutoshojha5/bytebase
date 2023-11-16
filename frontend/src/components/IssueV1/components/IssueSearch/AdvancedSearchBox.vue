<template>
  <div class="relative" :class="customClass">
    <NInput
      ref="inputRef"
      :value="state.searchText"
      :clearable="!!state.searchText"
      :placeholder="$t('issue.advanced-search.self')"
      style="width: 100%"
      @update:value="onUpdate($event)"
      @blur="onClear"
      @keyup="onKeydown"
      @click="onKeydown"
    >
      <template #prefix>
        <heroicons-outline:search class="h-4 w-4 text-control-placeholder" />
      </template>
    </NInput>
    <div
      v-if="state.showSearchScopes"
      class="absolute z-50 pt-2 mt-0.5 top-full w-full divide-y divide-block-border bg-white shadow-md"
    >
      <div
        v-for="item in searchScopes"
        :key="item.id"
        class="flex gap-x-1 px-3 py-1 items-center cursor-pointer hover:bg-gray-100"
        @mousedown.prevent.stop="onScopeSelect(item.id)"
      >
        <div class="space-x-1 text-sm">
          <span class="text-accent">{{ item.id }}:</span>
          <span class="text-control-light">{{ item.description }}</span>
        </div>
      </div>
    </div>
    <div
      v-if="state.currentScope"
      class="absolute z-50 top-full w-full bg-white shadow-md"
    >
      <div class="px-3 pt-2 pb-1 text-sm text-control font-semibold">
        {{ searchKeyword }}
      </div>
      <div class="max-h-60 overflow-y-auto divide-block-border">
        <div
          v-for="option in searchOptions"
          :key="option.id"
          class="flex gap-x-2 px-3 py-1 items-center cursor-pointer hover:bg-gray-100"
          @mousedown.prevent.stop="onOptionSelect(option.id)"
        >
          <component :is="option.label" class="text-control text-sm" />
          <span class="text-control-light text-sm">{{ option.id }}</span>
        </div>
        <div
          v-if="searchOptions.length === 0"
          class="px-3 py-1 text-control text-sm"
        >
          N/A
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { debounce, orderBy } from "lodash-es";
import { NInput } from "naive-ui";
import {
  reactive,
  computed,
  h,
  watch,
  VNode,
  onMounted,
  ref,
  nextTick,
} from "vue";
import { useI18n } from "vue-i18n";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import GitIcon from "@/components/GitIcon.vue";
import YouTag from "@/components/misc/YouTag.vue";
import {
  InstanceV1Name,
  InstanceV1EngineIcon,
  DatabaseV1Name,
} from "@/components/v2";
import {
  useProjectV1ListByCurrentUser,
  useInstanceV1List,
  useSearchDatabaseV1List,
  useUserStore,
  useDatabaseV1Store,
  useCurrentUserV1,
} from "@/store";
import {
  projectNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { SYSTEM_BOT_EMAIL } from "@/types";
import { Workflow } from "@/types/proto/v1/project_service";
import {
  projectV1Name,
  environmentV1Name,
  extractProjectResourceName,
  extractInstanceResourceName,
  SearchParams,
  SearchScopeId,
  buildSearchParamsBySearchText,
} from "@/utils";

const props = withDefaults(
  defineProps<{
    customClass?: string;
    params?: SearchParams;
    autofocus?: boolean;
  }>(),
  {
    customClass: "",
    params: undefined,
    autofocus: false,
  }
);

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();

interface LocalState {
  searchText: string;
  showSearchScopes: boolean;
  currentScope?: SearchScopeId;
}

interface SearchOption {
  id: string;
  label: VNode;
}

interface SearchScope {
  id: SearchScopeId;
  title: string;
  description: string;
  options: SearchOption[];
}

const buildSearchTextByParams = (params: SearchParams | undefined): string => {
  const prefix = (params?.scopes ?? [])
    .map((scope) => `${scope.id}:${scope.value}`)
    .join(" ");
  const query = params?.query ?? "";
  if (!prefix && !query) {
    return "";
  }
  return `${prefix} ${query}`;
};

const state = reactive<LocalState>({
  searchText: buildSearchTextByParams(props.params),
  showSearchScopes: props.autofocus,
});
const me = useCurrentUserV1();
const inputRef = ref<InstanceType<typeof NInput>>();
const userStore = useUserStore();
const databaseV1Store = useDatabaseV1Store();

watch(
  () => state.showSearchScopes,
  (show) => {
    if (show) state.currentScope = undefined;
  }
);

watch(
  () => props.params,
  (params) => {
    state.searchText = buildSearchTextByParams(params);
  }
);

const { projectList } = useProjectV1ListByCurrentUser();
const { instanceList } = useInstanceV1List(false /* !showDeleted */);
const { databaseList } = useSearchDatabaseV1List({
  parent: "instances/-",
});

const principalSearchOptions = computed(() => {
  const sortedUsers = orderBy(
    userStore.activeUserList,
    (user) => (user.name === me.value.name ? -1 : 1),
    "asc"
  );
  return sortedUsers.map((user) => {
    const children = [
      h(BBAvatar, { size: "TINY", username: user.title }),
      h("span", {}, user.title),
    ];
    if (user.name === me.value.name) {
      children.push(h(YouTag));
    }
    return {
      id: user.email,
      label: h("div", { class: "flex items-center gap-x-1" }, children),
    };
  });
});

// fullScopes provides full search scopes and options.
// we need this as the source of truth.
const fullScopes = computed((): SearchScope[] => {
  const scopes: SearchScope[] = [
    {
      id: "project",
      title: t("issue.advanced-search.scope.project.title"),
      description: t("issue.advanced-search.scope.project.description"),
      options: projectList.value.map((proj) => {
        const children: VNode[] = [
          h("span", { innerHTML: projectV1Name(proj) }),
        ];
        if (proj.workflow === Workflow.VCS) {
          children.push(h(GitIcon, { class: "w-5" }));
        }
        return {
          id: extractProjectResourceName(proj.name),
          label: h("div", { class: "flex items-center gap-x-2" }, children),
        };
      }),
    },
    {
      id: "approval",
      title: t("issue.advanced-search.scope.approval.title"),
      description: t("issue.advanced-search.scope.approval.description"),
      options: [
        {
          id: "pending",
          label: h(
            "span",
            {},
            t("issue.advanced-search.scope.approval.value.pending")
          ),
        },
        {
          id: "approved",
          label: h(
            "span",
            {},
            t("issue.advanced-search.scope.approval.value.approved")
          ),
        },
      ],
    },
    {
      id: "creator",
      title: t("issue.advanced-search.scope.creator.title"),
      description: t("issue.advanced-search.scope.creator.description"),
      options: principalSearchOptions.value,
    },
    {
      id: "assignee",
      title: t("issue.advanced-search.scope.assignee.title"),
      description: t("issue.advanced-search.scope.assignee.description"),
      options: principalSearchOptions.value,
    },
    {
      id: "approver",
      title: t("issue.advanced-search.scope.approver.title"),
      description: t("issue.advanced-search.scope.approver.description"),
      options: principalSearchOptions.value.filter(
        (o) => o.id !== SYSTEM_BOT_EMAIL
      ),
    },
    {
      id: "subscriber",
      title: t("issue.advanced-search.scope.subscriber.title"),
      description: t("issue.advanced-search.scope.subscriber.description"),
      options: principalSearchOptions.value,
    },
    {
      id: "type",
      title: t("issue.advanced-search.scope.type.title"),
      description: t("issue.advanced-search.scope.type.description"),
      options: [
        {
          id: "DDL",
          label: h("span", { innerHTML: "Data Definition Language" }),
        },
        {
          id: "DML",
          label: h("span", { innerHTML: "Data Manipulation Language" }),
        },
      ],
    },
    {
      id: "instance",
      title: t("issue.advanced-search.scope.instance.title"),
      description: t("issue.advanced-search.scope.instance.description"),
      options: instanceList.value.map((ins) => {
        return {
          id: extractInstanceResourceName(ins.name),
          label: h("div", { class: "flex items-center gap-x-1" }, [
            h(InstanceV1Name, { instance: ins, link: false, tooltip: false }),
            h("span", {
              innerHTML: `(${environmentV1Name(ins.environmentEntity)})`,
            }),
          ]),
        };
      }),
    },
    {
      id: "database",
      title: t("issue.advanced-search.scope.database.title"),
      description: t("issue.advanced-search.scope.database.description"),
      options: databaseList.value.map((db) => {
        return {
          id: `${db.databaseName}-${db.uid}`,
          label: h("div", { class: "flex items-center gap-x-1" }, [
            h(InstanceV1EngineIcon, { instance: db.instanceEntity }),
            h(DatabaseV1Name, { database: db, link: false }),
            h("span", {
              innerHTML: `(${environmentV1Name(
                db.effectiveEnvironmentEntity
              )})`,
            }),
          ]),
        };
      }),
    },
  ];
  return scopes;
});

// filteredScopes will filter search options by chosen scope.
// For example, if users select a specific project, we should only allow them select instances related with this project.
const filteredScopes = computed((): SearchScope[] => {
  const params = buildSearchParamsBySearchText(state.searchText);
  const existedScope = new Map<SearchScopeId, string>(
    params.scopes.map((scope) => [scope.id, scope.value])
  );

  const clone = fullScopes.value.map((scope) => ({
    ...scope,
    options: scope.options.map((option) => ({
      ...option,
    })),
  }));
  const index = clone.findIndex((scope) => scope.id === "database");
  clone[index].options = clone[index].options.filter((option) => {
    if (!existedScope.has("project") && !existedScope.has("instance")) {
      return true;
    }

    const uid = option.id.split("-").slice(-1)[0];
    const db = databaseV1Store.getDatabaseByUID(uid);
    const project = db.project;
    const instance = db.instance;

    if (project === `${projectNamePrefix}${existedScope.get("project")}`) {
      return true;
    }
    if (instance === `${instanceNamePrefix}${existedScope.get("instance")}`) {
      return true;
    }

    return false;
  });

  return clone;
});

// searchScopes will hide chosen search scope.
// For example, if uses already select the instance, we should NOT show the instance scope in the dropdown.
const searchScopes = computed((): SearchScope[] => {
  const params = buildSearchParamsBySearchText(state.searchText);
  const existedScope = new Set<SearchScopeId>(
    params.scopes.map((scope) => scope.id)
  );

  return filteredScopes.value.filter((scope) => {
    if (existedScope.has(scope.id)) {
      return false;
    }
    return true;
  });
});

const searchOptions = computed((): SearchOption[] => {
  const item = filteredScopes.value.find(
    (item) => item.id === state.currentScope
  );
  return item?.options ?? [];
});

const searchKeyword = computed(() => {
  const scope = filteredScopes.value.find(
    (item) => item.id === state.currentScope
  );
  return scope?.title ?? "";
});

const onScopeSelect = (scopeId: SearchScopeId) => {
  state.showSearchScopes = false;
  state.currentScope = scopeId;
};

const onOptionSelect = (scopeValue: string) => {
  const scopeId = state.currentScope;
  if (!scopeId) {
    return;
  }
  const params = buildSearchParamsBySearchText(state.searchText);
  const index = params.scopes.findIndex((s) => s.id === scopeId);
  if (index < 0) {
    params.scopes.push({
      id: scopeId,
      value: scopeValue,
    });
  } else {
    params.scopes[index] = {
      id: scopeId,
      value: scopeValue,
    };
  }
  state.searchText = buildSearchTextByParams(params);
  debouncedUpdate();
  onClear();
};

const onClear = (immediate = false) => {
  const clear = () => {
    state.showSearchScopes = false;
    state.currentScope = undefined;
  };
  if (immediate) {
    clear();
  } else {
    nextTick(clear);
  }
};

const debouncedUpdate = debounce(() => {
  emit("update:params", buildSearchParamsBySearchText(state.searchText));
}, 500);

const onUpdate = (value: string) => {
  state.searchText = value;
  debouncedUpdate();
};

onMounted(() => {
  if (props.autofocus) {
    inputRef.value?.inputElRef?.focus();
  }
});

const onKeydown = () => {
  if (!inputRef.value || !inputRef.value.inputElRef) {
    return;
  }
  if (!state.searchText) {
    state.showSearchScopes = true;
    return;
  }

  const start = inputRef.value.inputElRef.selectionStart ?? -1;
  const end = inputRef.value.inputElRef.selectionEnd ?? -1;
  if (start !== end) {
    onClear();
    return;
  }

  // Try to find the active section the cursor in.
  // For example, the searchText is (the | is the current cursor):
  // example 1:
  // project:xxx insta|nce:yyy custom-search-text
  // then the active section is instance:yyy, we should show the instances selector
  //
  // example 2:
  // project:xxx| instance:yyy custom-search-text
  // then the active section is project:xxx, we should show the projects selector
  //
  // example 3:
  // project:xxx instance:yyy |
  // we should NOT show any selector.
  const sections = state.searchText.split(" ");
  let i = 0;
  let len = 0;
  while (i < sections.length) {
    len += sections[i].length;
    if (i < sections.length - 1) {
      len += 1; // this is the length for empty space " " between sections.
    }
    if (len > start) {
      break;
    }
    i++;
  }

  if (i >= sections.length) {
    onClear(true /* immediate */);
    state.showSearchScopes = true;
    return;
  }

  const currentScope = sections[i].split(":")[0] as SearchScopeId;
  const existed =
    fullScopes.value.findIndex((item) => item.id === currentScope) >= 0;
  if (!existed) {
    onClear(true /* immediate */);
    state.showSearchScopes = true;
    return;
  }

  state.showSearchScopes = false;
  state.currentScope = currentScope;
};
</script>