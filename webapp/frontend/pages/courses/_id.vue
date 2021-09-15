<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12 mt-8 mb-8 rounded">
      <div class="flex-1 flex-col">
        <h1 class="text-2xl mb-4 font-bold">
          {{ course ? course.name : '講義名（仮）' }}
        </h1>
        <tabs
          :tabs="[
            { id: 'classworks', label: '講義情報' },
            { id: 'announcements', label: 'お知らせ' },
          ]"
        >
          <template v-if="!hasError">
            <template slot="classworks">
              <ClassList :course="course" :classes="classes" />
            </template>
            <template slot="announcements">
              <AnnouncementList
                :announcements="announcements"
                :link="link"
                @movePage="paginate"
                @open="openAnnouncement"
                @close="closeAnnouncement"
              />
            </template>
          </template>
          <template v-else>
            <template slot="classworks">
              <InlineNotification type="error" class="my-4">
                <template #title>APIエラーがあります</template>
                <template #message
                  >お知らせまたは科目概要の取得に失敗しました。</template
                >
              </InlineNotification>
            </template>
            <template slot="announcements">
              <InlineNotification type="error" class="my-4">
                <template #title>APIエラーがあります</template>
                <template #message
                  >お知らせまたは科目概要の取得に失敗しました。</template
                >
              </InlineNotification>
            </template>
          </template>
        </tabs>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'
import {
  Course,
  Announcement,
  GetAnnouncementResponse,
  ClassInfo,
} from '~/types/courses'
import { notify } from '~/helpers/notification_helper'
import AnnouncementList from '~/components/AnnouncementList.vue'
import ClassList from '~/components/ClassList.vue'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'
import InlineNotification from '~/components/common/InlineNotification.vue'

type CourseData = {
  title: string
  course: Course | undefined
  announcements: Announcement[]
  classes: ClassInfo[]
  link: string
  hasError: boolean
}

export default Vue.extend({
  key(route) {
    return route.fullPath
  },
  components: {
    InlineNotification,
    AnnouncementList,
    ClassList,
  },
  middleware: 'is_student',
  async asyncData(ctx: Context): Promise<CourseData> {
    const { params, query, $axios } = ctx

    try {
      const [course, classes, announcementResult] = await Promise.all([
        $axios.get<Course>(`/api/courses/${params.id}`),
        $axios.get<ClassInfo[]>(`/api/courses/${params.id}/classes`),
        $axios.get<GetAnnouncementResponse>(`/api/announcements`, {
          params: { ...query, courseId: params.id },
        }),
      ])
      const responseBody: GetAnnouncementResponse = announcementResult.data
      const link = announcementResult.headers.link
      const announcements: Announcement[] = Object.values(
        responseBody.announcements ?? []
      ).map((item) => {
        return {
          id: item.id,
          courseId: item.courseId,
          courseName: item.courseName,
          title: item.title,
          unread: item.unread,
        }
      })
      return {
        title: `ISUCHOLAR - 講義情報:${course.data.name}`,
        course: course.data,
        announcements,
        classes: classes.data ?? [],
        link,
        hasError: false,
      }
    } catch (e) {
      notify('講義情報の取得に失敗しました')
    }

    return {
      title: '',
      course: undefined,
      announcements: [],
      classes: [],
      link: '',
      hasError: true,
    }
  },
  data(): CourseData {
    return {
      title: '',
      course: undefined,
      announcements: [],
      classes: [],
      link: '',
      hasError: false,
    }
  },
  head(): any {
    return {
      title: this.title,
    }
  },
  watchQuery: true,
  methods: {
    async openAnnouncement(
      event: { done: () => undefined },
      announcement: Announcement
    ) {
      try {
        const res = await this.$axios.get(
          `/api/announcements/${announcement.id}`
        )
        const announcementDetail: Announcement = res.data
        const target = this.announcements.find(
          (item) => item.id === announcement.id
        )
        if (target) {
          target.message = announcementDetail.message
        }
        if (announcement.unread) {
          announcement.unread = false
        }
        event.done()
      } catch (e) {
        const target = this.announcements.find(
          (item) => item.id === announcement.id
        )
        if (target) {
          target.hasError = true
        }
        event.done()
        notify('お知らせの取得に失敗しました')
      }
    },
    closeAnnouncement(event: { done: () => undefined }) {
      event.done()
    },
    paginate(query: URLSearchParams) {
      this.$router.push({ query: urlSearchParamsToObject(query) })
    },
  },
})
</script>
