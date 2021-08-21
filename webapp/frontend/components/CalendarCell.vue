<template>
  <div
    class="border transition duration-500 ease hover:bg-primary-100"
    :class="cursorType"
  >
    <div
      class="
        flex flex-col
        h-full
        xl:min-h-40
        lg:min-h-32
        md:min-h-32
        sm:min-h-full
        xl:w-40
        lg:w-30
        md:w-30
        sm:w-full
        px-2
        py-2
        overflow-hidden
      "
    >
      <slot />
    </div>
  </div>
</template>
<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  props: {
    cursor: {
      type: String,
      default: 'default',
    },
  },
  computed: {
    cursorType(): string {
      return `cursor-${this.cursor}`
    },
  },
})
</script>
