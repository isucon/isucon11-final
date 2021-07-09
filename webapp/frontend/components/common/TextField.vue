<template>
  <div class="flex flex-auto items-center">
    <div class="flex-shrink-0 mr-2">
      <label class="text-gray-500 font-bold text-right" :for="id">
        {{ label }}
      </label>
    </div>
    <div class="w-full">
      <input
        :id="id"
        class="
          w-full
          bg-white
          appearance-none
          border-2 border-gray-200
          rounded
          py-2
          px-4
          text-gray-700
          leading-tight
          focus:outline-none focus:bg-white focus:border-purple-500
        "
        :type="type"
        :placeholder="placeholder"
        :value="defaultValue"
        @input="$emit('input', $event.target.value)"
      />
    </div>
  </div>
</template>
<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  props: {
    id: {
      type: String,
      required: true,
    },
    type: {
      type: String,
      default: 'text',
    },
    label: {
      type: String,
      required: true,
    },
    placeholder: {
      type: String,
      default: '',
    },
    defaultValue: {
      type: String,
      default: '',
    },
  },
})
</script>
