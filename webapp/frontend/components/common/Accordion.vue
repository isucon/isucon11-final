<template>
  <Card>
    <div class="flex flex-col justify-between leading-normal">
      <a
        href="#"
        class="block bg-white no-underline text-black"
        @click.prevent="toggle"
      >
        <slot name="header" />
        <div v-show="isOpen" class="p-2"><slot /></div>
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
