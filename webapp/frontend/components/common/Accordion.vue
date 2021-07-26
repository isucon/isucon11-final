<template>
  <Card>
    <div class="flex flex-col justify-between leading-normal">
      <a
        href="#"
        class="p-4 block bg-white no-underline text-black"
        @click.prevent="toggle"
      >
        <slot name="header" />
        <div v-show="isOpen" class="p-2"><slot /></div>
        <span v-show="!isOpen" class="text-primary-500"
          >&#9660; 詳細を見る</span
        >
        <span v-show="isOpen" class="text-primary-500">&#9650; 閉じる</span>
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
