<template>
  <div class="flex flex-row items-center">
    <div
      class="p-2 mr-3"
      :class="prevClasses"
      @click="prevDisabled ? null : $emit('goPrev')"
    >
      <fa-icon class="mr-2" icon="chevron-left" size="lg" />
      <span class="text-base"> Prev </span>
    </div>
    <div
      class="p-2 ml-3"
      :class="nextClasses"
      @click="nextDisabled ? null : $emit('goNext')"
    >
      <span class="text-base"> Next </span>
      <fa-icon class="ml-2" icon="chevron-right" size="lg" />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  name: 'Pagination',
  props: {
    prevDisabled: {
      type: Boolean,
      required: false,
      default() {
        return false
      },
    },
    nextDisabled: {
      type: Boolean,
      required: false,
      default() {
        return false
      },
    },
  },
  computed: {
    prevClasses(): string[] {
      return this.getClasses(this.prevDisabled)
    },
    nextClasses(): string[] {
      return this.getClasses(this.nextDisabled)
    },
  },
  methods: {
    getClasses(isDisabled: boolean): string[] {
      if (isDisabled) {
        return ['text-gray-500']
      } else {
        return ['cursor-pointer', 'text-gray-800']
      }
    },
  },
})
</script>
