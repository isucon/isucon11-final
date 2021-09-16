<template>
  <div>
    <div
      class="py-10 px-8 bg-white shadow-lg mt-8 mb-8 rounded w-192 max-w-full"
    >
      <div class="flex-1 flex-col">
        <div class="flex flex-row items-center mb-4">
          <h1 class="text-2xl font-bold text-gray-800 mr-4">お知らせ一覧</h1>
          <span class="bg-primary-500 text-white text-sm py-1 px-2 rounded-sm"
            ><fa-icon icon="bell" class="mr-0.5" /><span>{{
              numOfUnreads
            }}</span></span
          >
          <TextField
            id="input-course-name"
            v-model="courseName"
            class="flex-auto"
            label=""
            type="text"
            placeholder="科目名で絞り込み"
            @input="filterAnnouncements"
          />
        </div>
        <template v-if="!hasError">
          <AnnouncementList
            :announcements="announcements"
            :link="link"
            @movePage="paginate"
            @open="openAnnouncement"
            @close="closeAnnouncement"
          />
        </template>
        <template v-else>
          <InlineNotification type="error" class="my-4">
            <template #title>APIエラーがあります</template>
            <template #message>お知らせ一覧の取得に失敗しました。</template>
          </InlineNotification>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'
import { Announcement, GetAnnouncementResponse } from '~/types/courses'
import { notify } from '~/helpers/notification_helper'
import TextField from '~/components/common/TextField.vue'
import AnnouncementList from '~/components/AnnouncementList.vue'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'
import InlineNotification from '~/components/common/InlineNotification.vue'

type AsyncAnnounceData = {
  innerAnnouncements: Announcement[]
  numOfUnreads: number
  link: string
}

type AnnounceListData = AsyncAnnounceData & {
  courseName: string
  announcements: Announcement[]
  hasError: boolean
}

export default Vue.extend({
  key(route) {
    return route.fullPath
  },
  components: { InlineNotification, TextField, AnnouncementList },
  middleware: 'is_student',
  async asyncData(ctx: Context) {
    const { $axios, query, params } = ctx

    try {
      const apiPath = params.apipath ?? '/api/announcements'
      const response = await $axios.get(apiPath, { params: query })
      const responseBody: GetAnnouncementResponse = response.data
      const link = response.headers.link
      const announcements: Announcement[] = Object.values(
        responseBody.announcements ?? []
      ).map((item) => {
        return {
          id: item.id,
          courseId: item.courseId,
          courseName: item.courseName,
          title: item.title,
          unread: item.unread,
          hasError: false,
        }
      })
      const count = responseBody.unreadCount
      return {
        innerAnnouncements: announcements,
        numOfUnreads: count,
        link,
        hasError: false,
      }
    } catch (e) {
      notify('お知らせ一覧の取得に失敗しました')
    }

    return {
      innerAnnouncements: [],
      numOfUnreads: 0,
      link: '',
      hasError: true,
    }
  },
  data(): AnnounceListData {
    return {
      innerAnnouncements: [],
      announcements: [],
      courseName: '',
      numOfUnreads: 0,
      link: '',
      hasError: false,
    }
  },
  head: {
    title: 'ISUCHOLAR - お知らせ',
  },
  watchQuery: true,
  created() {
    this.announcements = this.innerAnnouncements
  },
  methods: {
    async openAnnouncement(
      event: { done: () => undefined },
      announcement: Announcement
    ) {
      try {
        const res = await this.$axios.get<Announcement>(
          `/api/announcements/${announcement.id}`
        )
        const announcementDetail = res.data
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
      } catch (e) {
        const target = this.innerAnnouncements.find(
          (item) => item.id === announcement.id
        )
        if (target) {
          target.hasError = true
        }
        event.done()
        notify('お知らせ詳細の取得に失敗しました')
      }
    },
    closeAnnouncement(event: { done: () => undefined }) {
      event.done()
    },
    filterAnnouncements() {
      this.announcements = this.innerAnnouncements.filter((item) => {
        return item.courseName.indexOf(this.courseName) === 0
      })
    },
    paginate(apipath: string, query: URLSearchParams) {
      this.$router.push({
        name: 'announce',
        query: urlSearchParamsToObject(query),
        params: { apipath },
      })
    },
  },
})
</script>
