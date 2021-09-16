<template>
  <div class="flex" :class="direction">
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
          pr-8
          pl-4
          w-full
          border-2 border-gray-200
          rounded
          appearance-none
          focus:outline-none focus:border-primary-500 focus:ring-0
        "
        :class="selected ? '' : 'text-gray-500'"
        :placeholder="placeholder"
        @change="onChange"
      >
        <option value="">選択してください</option>
        <template v-for="(v, i) in options">
          <option
            :key="`option-${i}`"
            :value="v.value"
            :selected="v.value === selected"
          >
            {{ v.text }}
          </option>
        </template>
      </select>
    </div>
  </div>
</template>
<script lang="ts">
import Vue, { PropType } from 'vue'

export type Option = {
  text: string
  value: unknown
}

export default Vue.extend({
  model: {
    prop: 'selected',
    event: 'change',
  },
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
    options: {
      type: Array as PropType<Option[]>,
      required: true,
      default: () => [],
    },
    selected: {
      type: [String, Number],
      default: undefined,
    },
  },
  computed: {
    direction() {
      return this.labelDirection.search(/^col/) >= 0
        ? 'flex-col items-start'
        : 'flex-row items-center'
    },
  },
  updated() {
    this.$emit('change', this.selected)
  },
  methods: {
    onChange(event: Event): void {
      const { target } = event
      if (target instanceof HTMLSelectElement) {
        this.$emit('change', target.value)
      }
    },
  },
})
</script>
