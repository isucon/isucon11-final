<template>
  <div>
    <div class="w-8/12">
      <Card>
        <div class="flex-1 flex-col">
          <div class="flex flex-row items-center mb-4">
            <h1 class="text-2xl font-bold mr-4">お知らせ一覧</h1>
            <div
              class="border border-gray-400 pl-1 pr-1 mr-4 cursor-pointer"
              :class="unreadFilterClasses"
              @click="toggleUnreadFilter"
            >
              <span class="text-sm">未読</span>
              <span
                class="bg-primary-800 text-white font-bold text-sm pl-1 pr-1"
                >{{ numOfUnreads }}</span
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
          <AnnouncementList
            :announcements="announcements"
            :link="link"
            @movePage="paginate"
            @open="openAnnouncement"
            @close="closeAnnouncement"
          />
        </div>
      </Card>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Announcement, AnnouncementResponse } from '~/types/courses'
import Card from '~/components/common/Card.vue'
import TextField from '~/components/common/TextField.vue'
import AnnouncementList from '~/components/AnnouncementList.vue'

type AsyncAnnounceData = {
  innerAnnouncements: Array<Announcement>
  numOfUnreads: number
  link: string
}

type AnnounceListData = AsyncAnnounceData & {
  courseName: string
  announcements: Array<Announcement>
  showUnreads: boolean
}

export default Vue.extend({
  key(route) {
    return route.fullPath
  },
  components: { Card, TextField, AnnouncementList },
  middleware: 'is_loggedin',
  async asyncData({ $axios, query }): Promise<AsyncAnnounceData> {
    const path = query.path ? (query.path as string) : '/api/announcements'
    const response = await $axios.get(path)
    const announcements: Array<AnnouncementResponse> = response.data
    const link = response.headers.link
    const result = announcements.map(
      (item: AnnouncementResponse): Announcement => {
        return {
          id: item.id,
          courseId: item.courseId,
          courseName: item.courseName,
          title: item.title,
          unread: item.unread,
          createdAt: new Date(item.createdAt * 1000).toLocaleString(),
        }
      }
    )
    const count = announcements.filter((item) => {
      return item.unread
    }).length
    return {
      innerAnnouncements: result,
      numOfUnreads: count,
      link,
    }
  },
  data(): AnnounceListData {
    return {
      innerAnnouncements: [],
      announcements: [],
      courseName: '',
      numOfUnreads: 0,
      showUnreads: false,
      link: '',
    }
  },
  computed: {
    unreadFilterClasses(): Array<String> {
      return this.showUnreads
        ? ['bg-primary-500', 'text-white']
        : ['bg-white', 'text-black']
    },
  },
  watchQuery: ['path'],
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
      const target = this.innerAnnouncements.find(
        (item) => item.id === announcement.id
      )
      if (target) {
        target.message = announcementDetail.message
      }
      if (announcement.unread) {
        this.numOfUnreads = this.numOfUnreads - 1
        announcement.unread = false
      }
      event.done()
    },
    closeAnnouncement(event: { done: () => undefined }) {
      event.done()
    },
    filterAnnouncements() {
      this.announcements = this.innerAnnouncements.filter((item) => {
        return item.courseName.indexOf(this.courseName) === 0
      })
      if (this.showUnreads) {
        this.announcements = this.announcements.filter((item) => {
          return item.unread
        })
      }
    },
    toggleUnreadFilter() {
      this.showUnreads = !this.showUnreads
      this.filterAnnouncements()
    },
    paginate(path: string) {
      this.$router.push(`/announce?path=${encodeURIComponent(path)}`)
    },
  },
})
</script>
