<template>
  <div class="flex flex-row items-center">
    <div
      class="p-2 mr-6"
      :class="prevClasses"
      @click="$emit('goPrev')"
      @mouseover="overPrev"
      @mouseleave="leavePrev"
    >
      <fa-icon class="mr-2" icon="chevron-left" size="lg" />
      <span class="text-base"> Prev </span>
    </div>
    <div
      class="p-2"
      :class="nextClasses"
      @click="$emit('goNext')"
      @mouseover="overNext"
      @mouseleave="leaveNext"
    >
      <span class="text-base mr-2"> Next </span>
      <fa-icon icon="chevron-right" size="lg" />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'

type paginationData = {
  prevClasses: Array<String>
  nextClasses: Array<String>
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
  data(): paginationData {
    return {
      prevClasses: [],
      nextClasses: [],
    }
  },
  created() {
    this.prevClasses = this.getClasses(this.prevDisabled, false)
    this.nextClasses = this.getClasses(this.nextDisabled, false)
  },
  methods: {
    getClasses(isDisabled: Boolean, hovered: Boolean): Array<String> {
      if (isDisabled) {
        return ['text-gray-500']
      } else {
        return hovered
          ? ['cursor-pointer', 'text-white', 'bg-primary-300', 'rounded']
          : ['cursor-pointer', 'text-black']
      }
    },
    overPrev() {
      this.prevClasses = this.getClasses(this.prevDisabled, true)
    },
    leavePrev() {
      this.prevClasses = this.getClasses(this.prevDisabled, false)
    },
    overNext() {
      this.nextClasses = this.getClasses(this.nextDisabled, true)
    },
    leaveNext() {
      this.nextClasses = this.getClasses(this.nextDisabled, false)
    },
  },
})
</script>
