<template>
  <div>
    <div class="mb-6">
      <AnnouncementCard
        v-for="announcement in announcements"
        :key="announcement.id"
        :announcement="announcement"
        @open="$emit('open', $event, announcement)"
        @close="$emit('close', $event)"
      />
    </div>
    <div class="flex justify-center">
      <Pagination
        :prev-disabled="!Boolean(parsedLink.prev)"
        :next-disabled="!Boolean(parsedLink.next)"
        @goPrev="$emit('movePage', parsedLink.prev)"
        @goNext="$emit('movePage', parsedLink.next)"
      />
    </div>
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
      parsedLink: { prev: '', next: '' },
    }
  },
  created() {
    if (this.link) {
      this.parsedLink = parseLinkHeader(this.link)
    }
  },
})
</script>
