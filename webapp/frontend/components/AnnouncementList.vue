<template>
  <div>
    <div class="mb-2">
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

type Link = {
  prev: String
  next: String
}

type AnnouncementListData = {
  parsedLink: Link
}

function parseLinkHeader(linkHeader: String): Link {
  const parsedLink = { prev: '', next: '' }
  const linkData = linkHeader.split(',')
  for (const link of linkData) {
    const linkInfo = /<([^>]+)>;\s+rel="([^"]+)"/gi.exec(link)
    if (linkInfo && (linkInfo[2] === 'prev' || linkInfo[2] === 'next')) {
      const path = '/' + linkInfo[1].split('/').splice(3, 2).join('/')
      parsedLink[linkInfo[2]] = path
    }
  }
  return parsedLink
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
    } as PropOptions<Array<Announcement>>,
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
