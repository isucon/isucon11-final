<template>
  <div class="border" :class="cursorType">
    <div
      class="
        flex flex-col
        h-full
        xl:min-h-40
        lg:min-h-32
        md:min-h-32
        sm:min-h-full
        w-full
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
