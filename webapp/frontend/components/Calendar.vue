<template>
  <div class="grid grid-cols-calendar">
    <CalendarHead class=""></CalendarHead>
    <CalendarHead class="">月</CalendarHead>
    <CalendarHead class="">火</CalendarHead>
    <CalendarHead class="">水</CalendarHead>
    <CalendarHead class="">木</CalendarHead>
    <CalendarHead class="">金</CalendarHead>

    <template v-for="i in periodCount">
      <div
        :key="`calendar-row-${i}`"
        class="flex flex-col items-center justify-center h-full border"
        :class="`col-start-1 row-start-${i + 1}`"
      >
        <span class="inline-block font-bold">{{ i }}</span>
      </div>
    </template>

    <div
      class="col-start-2 col-end-7 row-start-2"
      :class="`row-end-${periodCount + 2}`"
    >
      <div class="grid grid-cols-5" :class="`grid-rows-${periodCount}`">
        <slot />
      </div>
    </div>
  </div>
</template>
<script lang="ts">
import Vue from 'vue'
import CalendarHead from '~/components/CalendarHead.vue'

export default Vue.extend({
  components: { CalendarHead },
  props: {
    periodCount: {
      type: Number,
      default: 6,
    },
  },
})
</script>
