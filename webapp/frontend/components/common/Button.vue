<template>
  <button
    class="
      py-2
      px-6
      border
      rounded
      disabled:bg-gray-400 disabled:text-white disabled:border-gray-400
    "
    :class="colorType"
    :type="type"
    :disabled="disabled"
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
    disabled: {
      type: Boolean,
      default: false,
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
