<template>
  <div class="mt-2 flex-grow-0">
    <Accordion @open="$emit('open', $event)" @close="$emit('close', $event)">
      <template #header>
        <p class="text-2xl text-primary-500 font-bold flex items-center">
          {{ announcement.title }}
        </p>
        <div class="text-neutral-300 text-sm mb-4">
          {{ announcement.createdAt }}
        </div>
      </template>
      <template #default>
        <p class="text-black text-base mb-4">
          {{ announcement.message || '' }}
        </p>
      </template>
    </Accordion>
  </div>
</template>

<script lang="ts">
import Vue, { PropOptions } from 'vue'
import { Announcement } from '~/types/courses'
import Accordion from '~/components/common/Accordion.vue'

export default Vue.extend({
  name: 'Announcement',
  components: {
    Accordion,
  },
  props: {
    announcement: {
      type: Object,
      required: true,
    } as PropOptions<Announcement>,
  },
  data() {
    return {}
  },
})
</script>
