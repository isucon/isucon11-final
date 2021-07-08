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
          >&#9660; すべて開く</span
        >
        <span v-show="isOpen" class="text-primary-500">&#9650; 閉じる</span>
      </a>
    </div>
  </Card>
</template>

<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  name: 'Accordion',
  data(): { isOpen: boolean } {
    return {
      isOpen: false,
    }
  },
  methods: {
    toggle() {
      if (this.isOpen) {
        this.$emit('close')
      } else {
        this.$emit('open')
      }
      this.isOpen = !this.isOpen
    },
  },
})
</script>
