<template>
  <div>
    <span class="font-semibold">{{ $t("sql-review.filter") }}</span>
    <div class="flex flex-wrap gap-x-3">
      <span>{{ $t("common.database") }}:</span>
      <div
        v-for="stats in engineList"
        :key="stats.engine"
        class="flex items-center"
      >
        <input
          :id="stats.engine"
          type="checkbox"
          :value="stats.engine"
          :checked="isCheckedEngine(stats.engine)"
          class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
          @input="e => emit('toggle-checked-engine', stats.engine, (e.target as HTMLInputElement).checked)"
        />
        <label
          :for="stats.engine"
          class="ml-2 items-center text-sm text-gray-600"
        >
          {{ $t(`sql-review.engine.${stats.engine.toLowerCase()}`) }}
          <span
            class="items-center px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
          >
            {{ stats.count }}
          </span>
        </label>
      </div>
    </div>
    <div class="flex flex-wrap gap-x-3">
      <span>{{ $t("sql-review.level.name") }}:</span>
      <div
        v-for="stats in errorLevelList"
        :key="stats.level"
        class="flex items-center"
      >
        <input
          :id="stats.level"
          type="checkbox"
          :value="stats.level"
          :checked="isCheckedLevel(stats.level)"
          class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
          @input="e =>$emit('toggle-checked-level', stats.level, (e.target as HTMLInputElement).checked)"
        />
        <label
          :for="stats.level"
          class="ml-2 items-center text-sm text-gray-600"
        >
          {{ $t(`sql-review.level.${stats.level.toLowerCase()}`) }}
          <span
            class="items-center px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
          >
            {{ stats.count }}
          </span>
        </label>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import {
  LEVEL_LIST,
  RuleLevel,
  RuleTemplate,
  SchemaRuleEngineType,
} from "@/types";

type EngineTypeStats = {
  engine: SchemaRuleEngineType;
  count: number;
};
type RuleLevelStats = {
  level: RuleLevel;
  count: number;
};

const props = withDefaults(
  defineProps<{
    ruleList: RuleTemplate[];
    isCheckedEngine?: (engine: SchemaRuleEngineType) => boolean;
    isCheckedLevel?: (level: RuleLevel) => boolean;
  }>(),
  {
    isCheckedEngine: () => false,
    isCheckedLevel: () => false,
  }
);

const emit = defineEmits<{
  (
    event: "toggle-checked-engine",
    engine: SchemaRuleEngineType,
    on: boolean
  ): void;
  (event: "toggle-checked-level", level: RuleLevel, on: boolean): void;
}>();

const engineList = computed((): EngineTypeStats[] => {
  const tmp = props.ruleList.reduce((dict, rule) => {
    for (const engine of rule.engineList) {
      if (!dict[engine]) {
        dict[engine] = {
          engine: engine,
          count: 0,
        };
      }
      dict[engine].count += 1;
    }
    return dict;
  }, {} as { [id: string]: EngineTypeStats });
  return Object.values(tmp);
});

const errorLevelList = computed((): RuleLevelStats[] => {
  return LEVEL_LIST.map((level) => ({
    level,
    count: props.ruleList.filter((rule) => rule.level === level).length,
  }));
});
</script>