<template>
  <div class="grid grid-cols-calendar justify-items-stretch items-center">
    <CalendarHead class=""></CalendarHead>
    <CalendarHead class="">月</CalendarHead>
    <CalendarHead class="">火</CalendarHead>
    <CalendarHead class="">水</CalendarHead>
    <CalendarHead class="">木</CalendarHead>
    <CalendarHead class="">金</CalendarHead>

    <template v-for="i in 5">
      <div
        :key="`calendar-row-${i}`"
        class="flex flex-col items-center justify-center h-full border"
      >
        <span class="inline-block font-bold">{{ i }}</span>
      </div>
    </template>

    <div
      class="
        grid grid-cols-5 grid-rows-5
        col-start-2 col-end-7
        row-start-2 row-end-7
      "
    >
      <slot />
    </div>
  </div>
</template>
<script lang="ts">
import Vue from 'vue'
import CalendarHead from '~/components/CalendarHead.vue'

export default Vue.extend({
  components: { CalendarHead },
})
</script>
