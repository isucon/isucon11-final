<template>
  <div>
    <Accordion @open="$emit('open', $event)" @close="$emit('close', $event)">
      <template #header>
        <div class="text-xl text-primary-500 flex items-center justify-between">
          <span>
            {{ announcement.title }}
          </span>
          <span
            v-if="announcement.unread"
            class="ml-2 py-1 px-2 text-xs text-white rounded bg-primary-500"
            >未読</span
          >
        </div>
        <div class="flex items-center">
          <fa-icon class="text-gray-500" :icon="['far', 'clock']" size="sm" />
          <span class="text-gray-500 text-sm mr-2">
            {{ announcement.createdAt }}
          </span>
        </div>
      </template>
      <template #default>
        <p class="text-black text-base break-all">
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
