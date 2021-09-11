<template>
  <div class="flex flex-col items-start">
    <div class="flex-shrink-0 mr-2">
      <label class="text-gray-500 font-bold text-right"> {{ label }} </label>
    </div>
    <div class="w-full">
      <span class="px-4">
        {{ value }}
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  props: {
    label: {
      type: String,
      required: true,
    },
    value: {
      type: String,
      required: true,
    },
  },
})
</script>
