<template>
  <div class="flex flex-row items-center">
    <div
      class="p-2 mr-6"
      :class="prevClasses"
      @click="prevDisabled ? null : $emit('goPrev')"
    >
      <fa-icon class="mr-2" icon="chevron-left" size="lg" />
      <span class="text-base"> Prev </span>
    </div>
    <div
      class="p-2"
      :class="nextClasses"
      @click="nextDisabled ? null : $emit('goNext')"
    >
      <span class="text-base mr-2"> Next </span>
      <fa-icon icon="chevron-right" size="lg" />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'

type PaginationData = {
  prevClasses: string[]
  nextClasses: string[]
}

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
  data(): PaginationData {
    return {
      prevClasses: [],
      nextClasses: [],
    }
  },
  created() {
    this.prevClasses = this.getClasses(this.prevDisabled)
    this.nextClasses = this.getClasses(this.nextDisabled)
  },
  methods: {
    getClasses(isDisabled: boolean): string[] {
      if (isDisabled) {
        return ['text-gray-500']
      } else {
        return [
          'cursor-pointer',
          'text-black',
          'hover:bg-primary-300',
          'hover:text-white',
          'hover:rounded',
        ]
      }
    },
  },
})
</script>
