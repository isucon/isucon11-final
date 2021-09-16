<template>
  <div v-if="announcements.length" class="py-4">
    <div class="flex flex-col gap-4">
      <AnnouncementCard
        v-for="announcement in announcements"
        :key="announcement.id"
        :announcement="announcement"
        @open="$emit('open', $event, announcement)"
        @close="$emit('close', $event)"
      />
    </div>
    <div class="flex justify-center mt-4">
      <Pagination
        :prev-disabled="!Boolean(parsedLink.prev)"
        :next-disabled="!Boolean(parsedLink.next)"
        @goPrev="$emit('movePage', parsedLink.prev.path, parsedLink.prev.query)"
        @goNext="$emit('movePage', parsedLink.next.path, parsedLink.next.query)"
      />
    </div>
  </div>
  <div v-else class="py-4 min-h-28">
    <div>お知らせは登録されていません</div>
  </div>
</template>
<script lang="ts">
import Vue, { PropOptions } from 'vue'
import { Announcement } from '~/types/courses'
import AnnouncementCard from '~/components/Announcement.vue'
import Pagination from '~/components/common/Pagination.vue'
import type { Link } from '~/helpers/link_helper'
import { parseLinkHeader } from '~/helpers/link_helper'

type AnnouncementListData = {
  parsedLink: Link
}

export default Vue.extend({
  components: {
    AnnouncementCard,
    Pagination,
  },
  props: {
    announcements: {
      type: Array,
      required: true,
    } as PropOptions<Announcement[]>,
    link: {
      type: String,
      required: false,
      default: '',
    },
  },
  data(): AnnouncementListData {
    return {
      parsedLink: { prev: undefined, next: undefined },
    }
  },
  created() {
    if (this.link) {
      this.parsedLink = parseLinkHeader(this.link)
    }
  },
})
</script>
