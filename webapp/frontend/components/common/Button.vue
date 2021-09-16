<template>
  <button
    class="
      border
      rounded
      disabled:bg-gray-400
      disabled:text-white
      disabled:border-gray-400
      disabled:cursor-default
      hover:opacity-90 hover:shadow-md
      transition-opacity transition-shadow
      ease-out
      duration-500
    "
    :class="[colorType, sizeType, wClass]"
    :type="type"
    :disabled="disabled"
    @click="$emit('click')"
  >
    <slot />
  </button>
</template>

<script lang="ts">
import Vue, { PropType } from 'vue'

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
    size: {
      type: String,
      default: 'default',
    },
    wClass: {
      type: String as PropType<`w-${number}`>,
      default: 'w-auto',
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
    sizeType() {
      if (this.size === 'mini') {
        return ['py-1', 'px-3']
      } else {
        return ['py-2', 'px-6']
      }
    },
  },
})
</script>
