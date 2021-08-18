<template>
  <div>
    <div class="w-8/12">
      <Card>
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
      </Card>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  Course,
  Announcement,
  AnnouncementResponse,
  GetAnnouncementResponse,
  ClassInfo,
} from '~/types/courses'
import { notify } from '~/helpers/notification_helper'
import Card from '~/components/common/Card.vue'
import AnnouncementList from '~/components/AnnouncementList.vue'
import ClassInfoCard from '~/components/ClassInfo.vue'

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
    Card,
    AnnouncementList,
    ClassInfoCard,
  },
  middleware: 'is_loggedin',
  async asyncData({ params, query, $axios }): Promise<CourseData> {
    const path = query.path
      ? (query.path as string)
      : `/api/announcements?course_id=${params.id}`
    const [course, classes, announcementResult] = await Promise.all([
      $axios.$get(`/api/syllabus/${params.id}`),
      $axios.$get(`/api/courses/${params.id}/classes`),
      $axios.get(path),
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
  watchQuery: ['path'],
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
    paginate(path: string) {
      this.$router.push(
        `/courses/${this.course.id}?path=${encodeURIComponent(path)}`
      )
    },
    submissionComplete(classIdx: number) {
      this.classes[classIdx].submitted = true
    },
  },
})
</script>
