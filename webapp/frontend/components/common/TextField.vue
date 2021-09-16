<template>
  <div>
    <div class="flex" :class="wrapperClass">
      <div class="flex-shrink-0 mr-2" :class="labelClass">
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
            text-gray-800
            leading-tight
            focus:outline-none focus:border-primary-500 focus:ring-0
          "
          :type="type"
          :placeholder="placeholder"
          :value="value"
          :autocomplete="autocomplete"
          :min="min"
          :max="max"
          :required="required"
          @input="$emit('input', $event.target.value)"
        />
      </div>
    </div>
    <template v-if="invalid">
      <span class="font-bold text-sm text-red-500"
        ><fa-icon icon="exclamation-triangle" class="mr-0.5" />
        {{ invalidText }}
      </span>
    </template>
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
    labelDirection: {
      type: String,
      default: 'horizontal',
    },
    placeholder: {
      type: String,
      default: '',
    },
    value: {
      type: String,
      default: '',
    },
    autocomplete: {
      type: String,
      default: 'on',
    },
    required: {
      type: Boolean,
      default: false,
    },
    invalid: {
      type: Boolean,
      default: false,
    },
    invalidText: {
      type: String,
      default: '',
    },
    min: {
      type: [String, Number],
      default: null,
    },
    max: {
      type: [String, Number],
      default: null,
    },
  },
  computed: {
    wrapperClass(): string[] {
      if (this.labelDirection === 'vertical') {
        return ['flex-col']
      } else {
        return ['items-center']
      }
    },
    labelClass(): string[] {
      if (this.labelDirection === 'vertical') {
        return []
      } else {
        return ['w-1/6']
      }
    },
  },
})
</script>

<style>
input:placeholder-shown {
  text-overflow: ellipsis;
}
</style>
