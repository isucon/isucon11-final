<template>
  <div>
    <Card>
      <div class="flex-1 flex-col">
        <div class="flex flex-row items-center mb-4">
          <h1 class="text-2xl font-bold mr-4">お知らせ一覧</h1>
          <div
            class="border border-gray-400 pl-1 pr-1 mr-4"
            @click="filterUnreadAnnouncements"
          >
            <span class="text-primary-500 text-sm">未読</span>
            <span class="bg-primary-500 text-white font-bold text-sm pl-1 pr-1"
              >4</span
            >
          </div>
          <TextField
            id="input-course-name"
            v-model="courseName"
            class=""
            label=""
            type="text"
            placeholder="科目名で絞り込み"
            @input="filterAnnouncements"
          />
        </div>
        <Announcement
          v-for="announcement in announcements"
          :key="announcement.id"
          :announcement="announcement"
          @open="openAnnouncement($event, announcement)"
          @close="closeAnnouncement($event)"
        />
      </div>
    </Card>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Announcement } from '~/types/courses'

type AsyncAnnounceData = {
  innerAnnouncements: Array<Announcement>
}

type AnnounceListData = AsyncAnnounceData & {
  courseName: string
  announcements: Array<Announcement>
}

type AnnouncementResponse = {
  id: string
  courseName: string
  title: string
  createdAt: number
}

export default Vue.extend({
  async asyncData({ $axios }): Promise<AsyncAnnounceData> {
    const announcements: Array<AnnouncementResponse> = await $axios.$get(
      '/api/announcements'
    )
    const result = announcements.map(
      (item: AnnouncementResponse): Announcement => {
        return {
          id: item.id,
          courseName: item.courseName,
          title: item.title,
          createdAt: new Date(item.createdAt).toLocaleString(),
        }
      }
    )
    return {
      innerAnnouncements: result,
    }
  },
  data(): AnnounceListData {
    return {
      innerAnnouncements: [],
      announcements: [],
      courseName: '',
    }
  },
  created() {
    this.announcements = this.innerAnnouncements
  },
  methods: {
    async openAnnouncement(
      event: { done: () => undefined },
      announcement: Announcement
    ) {
      const announcementDetail: Announcement = await this.$axios.$get(
        `/api/announcements/${announcement.id}`
      )
      this.innerAnnouncements.filter(
        (item) => item.id === announcement.id
      )[0].message = announcementDetail.message
      event.done()
    },
    closeAnnouncement(event: { done: () => undefined }) {
      event.done()
    },
    filterAnnouncements() {
      this.announcements = this.innerAnnouncements.filter((item) => {
        return item.courseName.indexOf(this.courseName) === 0
      })
    },
    async filterUnreadAnnouncements() {
      // まだAPI側でunread fieldが実装されていないのでひとまずコメントアウト
      // this.announcements = this.innerAnnouncements.filter((item) => {
      //   return item.unread
      // })
    },
  },
})
</script>
