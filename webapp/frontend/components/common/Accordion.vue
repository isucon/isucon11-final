<template>
  <Card>
    <div class="flex flex-col justify-between leading-normal">
      <a
        href="#"
        class="block bg-white no-underline text-gray-800"
        @click.prevent="toggle"
      >
        <slot name="header" />
        <transition
          enter-active-class="duration-300 ease-out"
          leave-active-class="duration-300 ease-in"
          enter-to-class="max-h-24 overflow-hidden"
          leave-class="max-h-24 overflow-hidden"
          enter-class="max-h-0 overflow-hidden"
          leave-to-class="max-h-0 overflow-hidden"
        >
          <div v-if="isOpen" class="p-2"><slot /></div>
        </transition>
        <div class="mt-4 text-primary-500">
          {{ isOpen ? '閉じる' : '詳細を見る' }}
        </div>
      </a>
    </div>
  </Card>
</template>

<script lang="ts">
import Vue from 'vue'
import Card from '~/components/common/Card.vue'

export default Vue.extend({
  name: 'Accordion',
  components: {
    Card,
  },
  data(): { isOpen: boolean } {
    return {
      isOpen: false,
    }
  },
  methods: {
    toggle() {
      if (this.isOpen) {
        this.$emit('close', { done: () => (this.isOpen = !this.isOpen) })
      } else {
        this.$emit('open', { done: () => (this.isOpen = !this.isOpen) })
      }
    },
  },
})
</script>
