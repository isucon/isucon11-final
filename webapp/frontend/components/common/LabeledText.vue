<template>
  <div class="flex flex-col">
    <div>
      <label class="text-gray-500 font-bold text-right"> {{ label }} </label>
    </div>
    <span class="mt-0.5 ml-3">
      {{ value }}
    </span>
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
