<template>
  <div
    class="
      border
      transition
      cursor-pointer
      duration-500
      ease
      hover:bg-primary-100
    "
  >
    <div
      class="
        flex flex-col
        xl:h-40
        lg:h-30
        md:h-30
        sm:h-full
        xl:w-40
        lg:w-30
        md:w-30
        sm:w-full
        px-2
        py-2
        overflow-hidden
      "
    >
      <slot />
    </div>
  </div>
</template>
<script lang="ts">
import Vue from 'vue'

export default Vue.extend({})
</script>
