<template>
  <div>
    <div
      class="py-10 px-8 bg-white shadow-lg w-192 max-w-full mt-8 mb-8 rounded"
    >
      <template v-if="!course">
        <div>now loading</div>
      </template>
      <div v-else class="flex flex-1 flex-col gap-4">
        <h1 class="text-2xl font-bold text-gray-800">
          {{ course.name }}
        </h1>
        <div>
          <h2 class="text-lg font-bold text-gray-800">講義の概要と目的</h2>
          <div class="text-gray-800">
            {{ course.description }}
          </div>
        </div>
        <div class="grid grid-cols-course w-full gap-1 text-sm text-gray-800">
          <h2 class="font-bold">科目コード</h2>
          <div>
            {{ course.code }}
          </div>
          <h2 class="font-bold">科目種別</h2>
          <div>
            {{ formatType(course.type) }}
          </div>
          <h2 class="font-bold">科目の状態</h2>
          <div>
            {{ formatStatus(course.status) }}
          </div>
          <h2 class="font-bold">教員</h2>
          <div>
            {{ course.teacher }}
          </div>
          <h2 class="font-bold">単位数</h2>
          <div>
            {{ course.credit }}
          </div>
        </div>
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
import { formatStatus, formatType } from '~/helpers/course_helper'
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
      const apiPath = params.apipath ?? `/api/announcements`
      const [course, classes, announcementResult] = await Promise.all([
        $axios.get<Course>(`/api/courses/${params.id}`),
        $axios.get<ClassInfo[]>(`/api/courses/${params.id}/classes`),
        $axios.get<GetAnnouncementResponse>(apiPath, {
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
    paginate(apipath: string, query: URLSearchParams) {
      // course_id はURLのパスから取得するのでAPIから返ってきたパラメータはページング先のURLに引き継がないように消す
      this.$router.push({
        query: urlSearchParamsToObject(query, ['course_id']),
        params: { apipath },
      })
    },
    formatStatus,
    formatType,
  },
})
</script>
