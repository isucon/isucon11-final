<template>
  <transition
    enter-active-class="transition-opacity duration-200"
    leave-active-class="transition-opacity duration-200"
    enter-class="opacity-0"
    leave-to-class="opacity-0"
  >
    <div
      v-show="isShown"
      class="
        fixed
        z-10
        inset-0
        h-screen
        w-screen
        bg-gray-500 bg-opacity-75
        flex
        items-center
        justify-center
      "
      aria-labelledby="modal-title"
      role="dialog"
      aria-modal="true"
      @click="$emit('close')"
    >
      <div class="m-auto" @click.stop>
        <slot />
      </div>
    </div>
  </transition>
</template>
<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  props: {
    isShown: {
      type: Boolean,
      default: false,
    },
  },
  computed: {
    state() {
      return this.isShown ? 'block' : 'hidden'
    },
  },
  watch: {
    isShown(newVal, oldVal) {
      if (newVal === oldVal) {
        return
      }
      if (newVal) {
        document.body.style.overflow = 'hidden'
        const getScrollbarWidth = () => {
          const element = document.createElement('div')
          element.style.visibility = 'hidden'
          element.style.overflow = 'scroll'
          document.body.appendChild(element)
          const scrollbarWidth = element.offsetWidth - element.clientWidth
          document.body.removeChild(element)
          return scrollbarWidth
        }
        document.body.style.paddingRight = `${getScrollbarWidth()}px`
      }
      if (!newVal) {
        document.body.style.overflow = 'auto'
        document.body.style.paddingRight = '0'
      }
    },
  },
})
</script>
