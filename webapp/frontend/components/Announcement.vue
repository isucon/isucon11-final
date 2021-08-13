<template>
  <div class="mt-2 flex-grow-0">
    <Accordion @open="$emit('open', $event)" @close="$emit('close', $event)">
      <template #header>
        <div class="text-2xl text-primary-500 flex items-center">
          <span class="font-light mr-2">
            {{ announcement.courseName }}
          </span>
          <span class="font-bold">
            {{ announcement.title }}
          </span>
        </div>
        <div class="text-gray-500 mb-4">
          <fa-icon class="mr-1" icon="clock" size="sm" />
          <span class="text-sm">
            {{ announcement.createdAt }}
          </span>
        </div>
      </template>
      <template #default>
        <p class="text-black text-base break-all mb-4">
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
