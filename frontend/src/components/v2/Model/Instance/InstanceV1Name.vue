<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && 'normal-link'"
  >
    <InstanceV1EngineIcon
      v-if="icon && iconPosition === 'prefix'"
      :instance="instance"
    />

    <span>{{ instanceV1Name(instance) }}</span>

    <InstanceV1EngineIcon
      v-if="icon && iconPosition === 'suffix'"
      :instance="instance"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import InstanceV1EngineIcon from "./InstanceV1EngineIcon.vue";
import { instanceV1Name, instanceV1Slug } from "@/utils";
import { Instance } from "@/types/proto/v1/instance_service";

const props = withDefaults(
  defineProps<{
    instance: Instance;
    tag?: string;
    link?: boolean;
    icon?: boolean;
    iconPosition?: "prefix" | "suffix";
  }>(),
  {
    tag: "span",
    link: true,
    icon: true,
    iconPosition: "prefix",
  }
);

const bindings = computed(() => {
  if (props.link) {
    return {
      to: `/instance/${instanceV1Slug(props.instance)}`,
      activeClass: "",
      exactActiveClass: "",
      onClick: (e: MouseEvent) => {
        e.stopPropagation();
      },
    };
  }
  return {};
});
</script>