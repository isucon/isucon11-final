<template>
  <div class="bg-white">
    <nav class="flex flex-col sm:flex-row">
      <button
        v-for="(tab, index) in tabs"
        :key="tab.id"
        class="p-4 block hover:text-primary-500 focus:outline-none border-b-4"
        :class="tabClasses(index)"
        @click="activate(index)"
      >
        {{ tab.label }}
      </button>
    </nav>
    <slot :name="activeSlotName" />
  </div>
</template>

<script lang="ts">
import Vue, { PropOptions } from 'vue'

type Tab = {
  id: string
  label: string
}

export default Vue.extend({
  props: {
    tabs: {
      type: Array,
      required: true,
    } as PropOptions<Tab[]>,
  },
  data() {
    return {
      activeTabIndex: 0,
    }
  },
  computed: {
    activeSlotName(): string {
      return this.tabs[this.activeTabIndex].id
    },
  },
  methods: {
    tabClasses(index: number): string[] {
      return index === this.activeTabIndex
        ? ['text-primary-500', 'font-bold', 'border-primary-500']
        : ['text-gray-800', 'border-gray-600']
    },
    activate(index: number) {
      this.activeTabIndex = index
    },
  },
})
</script>
