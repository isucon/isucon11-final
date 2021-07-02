<template>
  <nav
    class="
      flex
      items-center
      justify-between
      flex-wrap
      bg-primary-500
      p-6
      shadow-lg
    "
  >
    <div class="flex items-center flex-no-shrink text-white mr-6">
      <span class="text-xl tracking-tight">ISUCHOLAR</span>
    </div>

    <div class="w-full flex-grow lg:flex lg:items-center lg:w-auto text-white">
      <div class="text-sm lg:flex-grow"></div>
      <div :class="state">
        <NuxtLink to="announce">お知らせ</NuxtLink>
        <span>学籍番号: xxxxxx</span>
      </div>
    </div>
  </nav>
</template>
<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  computed: {
    state() {
      return this.$cookies.get('session') ? 'block' : 'hidden'
    },
  },
})
</script>
