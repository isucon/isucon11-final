<template>
  <div>
    <Header />
    <Nuxt class="flex justify-center" />
  </div>
</template>
<script lang="ts">
import Vue from 'vue'
import Header from '~/components/common/Header.vue'

export default Vue.extend({
  components: { Header },
})
</script>
