<template>
  <div class="flex flex-auto" :class="direction">
    <div class="flex-shrink-0 mr-2">
      <label class="text-gray-500 font-bold text-right" :for="id">
        {{ label }}
      </label>
    </div>
    <div class="w-full">
      <select
        :id="id"
        class="
          block
          py-1.5
          px-4
          w-full
          border-2 border-gray-200
          rounded
          placeholder-gray-500
          appearance-none
          focus:ring-primary-200
        "
        :placeholder="placeholder"
      >
        <option value="">選択してください</option>
        <template v-for="(v, i) in value">
          <option :key="`option-${i}`" :value="v.value">{{ v.text }}</option>
        </template>
      </select>
    </div>
  </div>
</template>
<script lang="ts">
import Vue, { PropType } from 'vue'

export type Value = {
  text: string
  value: unknown
}

export default Vue.extend({
  props: {
    id: {
      type: String,
      required: true,
    },
    label: {
      type: String,
      required: true,
    },
    labelDirection: {
      type: String,
      default: 'column',
    },
    placeholder: {
      type: String,
      default: '',
    },
    value: {
      type: Array as PropType<Value[]>,
      required: true,
      default: () => [],
    },
  },
  computed: {
    direction() {
      return this.labelDirection.search(/^col/) >= 0
        ? 'flex-col items-start'
        : 'flex-row items-center'
    },
  },
})
</script>
