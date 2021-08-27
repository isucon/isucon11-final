<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12 mt-8 mb-8 rounded">
      <div class="flex-1 flex-col">
        <h1 class="text-2xl mb-4 font-bold">{{ course.name }}</h1>
        <tabs
          :tabs="[
            { id: 'announcements', label: 'お知らせ' },
            { id: 'classworks', label: '講義情報' },
          ]"
        >
          <template slot="announcements">
            <AnnouncementList
              :announcements="announcements"
              :link="link"
              @movePage="paginate"
              @open="openAnnouncement"
              @close="closeAnnouncement"
            />
          </template>
          <template slot="classworks">
            <ClassInfoCard
              v-for="(cls, index) in classes"
              :key="cls.id"
              :course="course"
              :classinfo="cls"
              @submitted="submissionComplete(index)"
            />
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
  AnnouncementResponse,
  GetAnnouncementResponse,
  ClassInfo,
} from '~/types/courses'
import { notify } from '~/helpers/notification_helper'
import AnnouncementList from '~/components/AnnouncementList.vue'
import ClassInfoCard from '~/components/ClassInfo.vue'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'

type CourseData = {
  course: Course
  announcements: Announcement[]
  classes: ClassInfo[]
  link: string
}

export default Vue.extend({
  key(route) {
    return route.fullPath
  },
  components: {
    AnnouncementList,
    ClassInfoCard,
  },
  middleware: 'is_student',
  async asyncData(ctx: Context): Promise<CourseData> {
    const { params, query, $axios } = ctx
    console.log(params)
    const [course, classes, announcementResult] = await Promise.all([
      $axios.$get(`/api/syllabus/${params.id}`),
      $axios.$get(`/api/courses/${params.id}/classes`),
      $axios.get(`/api/announcements`, {
        params: { ...query, courseId: params.id },
      }),
    ])
    const responseBody: GetAnnouncementResponse = announcementResult.data
    const link = announcementResult.headers.link
    const announcements: Announcement[] = Object.values(
      responseBody.announcements
    ).map((item: AnnouncementResponse) => {
      const announce: Announcement = {
        id: item.id,
        courseId: item.courseId,
        courseName: item.courseName,
        title: item.title,
        unread: item.unread,
        createdAt: new Date(item.createdAt * 1000).toLocaleString(),
      }
      return announce
    })
    return {
      course,
      announcements,
      classes,
      link,
    }
  },
  data(): CourseData | undefined {
    return undefined
  },
  watchQuery: true,
  methods: {
    async openAnnouncement(
      event: { done: () => undefined },
      announcement: Announcement
    ) {
      try {
        const announcementDetail: Announcement = await this.$axios.$get(
          `/api/announcements/${announcement.id}`
        )
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
        notify('お知らせの取得に失敗しました')
      }
    },
    closeAnnouncement(event: { done: () => undefined }) {
      event.done()
    },
    paginate(query: URLSearchParams) {
      this.$router.push({ query: urlSearchParamsToObject(query) })
    },
    submissionComplete(classIdx: number) {
      this.classes[classIdx].submitted = true
    },
  },
})
</script>
