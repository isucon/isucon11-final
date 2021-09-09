<template>
  <div class="w-full border px-4 py-2 rounded" :class="[wrapperColor]">
    <div class="font-bold mb-2">
      <fa-icon :icon="iconType" :class="[iconColor]" />
      <slot name="title" />
    </div>
    <div>
      <p class="text-sm text-gray-800"><slot name="message" /></p>
    </div>
  </div>
</template>
<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  props: {
    type: {
      type: String,
      default: 'info', // error, warn, info
    },
  },
  computed: {
    wrapperColor(): string[] {
      switch (this.type) {
        case 'error':
          return ['bg-red-100', 'border-red-500']
        case 'warn':
          return ['bg-yellow-100', 'border-yellow-500']
        case 'info':
        default:
          return ['bg-blue-100', 'border-blue-500']
      }
    },
    iconType(): string {
      switch (this.type) {
        case 'error':
        case 'warn':
          return 'exclamation-triangle'
        case 'info':
        default:
          return 'info-circle'
      }
    },
    iconColor(): string {
      switch (this.type) {
        case 'error':
          return 'text-red-700'
        case 'warn':
          return 'text-yellow-600'
        case 'info':
        default:
          return 'text-blue-700'
      }
    },
  },
})
</script>
