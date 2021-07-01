<template>
  <div class="py-10 px-8 bg-white shadow-lg">
    <slot />
  </div>
</template>

<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  name: 'Card',
})
</script>
