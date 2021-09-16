<template>
  <div>
    <Accordion @open="$emit('open', $event)" @close="$emit('close', $event)">
      <template #header>
        <div class="text-xl text-gray-800 flex items-center justify-between">
          <span>
            {{ announcement.title }}
          </span>
          <span
            v-if="announcement.unread"
            class="ml-2 py-1 px-2 text-xs text-white rounded bg-primary-500"
            >未読</span
          >
        </div>
      </template>
      <template #default>
        <template v-if="!announcement.hasError">
          <p class="text-gray-800 text-base break-all">
            {{ announcement.message || '' }}
          </p>
        </template>
        <template v-else>
          <p class="text-base text-gray-800 break-all my-2">
            お知らせ詳細の取得に失敗しました。
          </p>
        </template>
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
