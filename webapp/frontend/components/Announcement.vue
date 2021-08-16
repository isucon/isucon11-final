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
        <div class="mb-4">
          <fa-icon
            class="mr-1 text-gray-500"
            :icon="['far', 'clock']"
            size="sm"
          />
          <span class="text-gray-500 text-sm mr-2">
            {{ announcement.createdAt }}
          </span>
          <template v-if="announcement.unread">
            <span
              class="
                pt-1
                pl-2
                pr-2
                pb-1
                text-xs text-white
                rounded-sm
                bg-primary-500
              "
              >未読</span
            >
          </template>
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
