<template>
  <button
    class="py-2.5 px-6 border rounded"
    :class="colorType"
    :type="type"
    @click="$emit('click')"
  >
    <slot />
  </button>
</template>
<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  props: {
    color: {
      type: String,
      default: 'plain',
    },
    type: {
      type: String,
      default: 'button',
    },
  },
  computed: {
    colorType() {
      if (this.color === 'primary') {
        return ['bg-primary-500', 'text-white']
      } else {
        return ['bg-white', 'border-primary-500', 'text-primary-500']
      }
    },
  },
})
</script>
