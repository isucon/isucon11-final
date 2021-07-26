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
              <AnnouncementCard
                v-for="announcement in announcements"
                :key="announcement.id"
                :announcement="announcement"
                @close="closeAnnouncement($event)"
                @open="openAnnouncement($event, announcement)"
              />
            </template>
            <template slot="classworks">
              <ClassInfoCard
                v-for="cls in classes"
                :key="cls.id"
                :course="course"
                :classinfo="cls"
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
  ClassInfo,
} from '~/types/courses'
import Card from '~/components/common/Card.vue'
import AnnouncementCard from '~/components/Announcement.vue'
import ClassInfoCard from '~/components/ClassInfo.vue'

type CourseData = {
  course: Course
  announcements: Array<Announcement>
  classes: Array<ClassInfo>
}

type ClassResponse = {
  id: string
  part: number
  title: string
  description: string
  submissionClosedAt?: number
}

export default Vue.extend({
  components: {
    Card,
    AnnouncementCard,
    ClassInfoCard,
  },
  middleware: 'is_loggedin',
  async asyncData({ params, $axios }): Promise<CourseData> {
    const [course, classResponses, announcementResponses] = await Promise.all([
      $axios.$get(`/api/syllabus/${params.id}`),
      $axios.$get(`/api/courses/${params.id}/classes`),
      $axios.$get(`/api/announcements?course_id=${params.id}`),
    ])
    const classes: Array<ClassInfo> = classResponses.map(
      (item: ClassResponse) => {
        const cls: ClassInfo = {
          id: item.id,
          part: item.part,
          title: item.title,
          description: item.description,
        }
        if (item.submissionClosedAt !== undefined) {
          cls.submissionClosedAt = new Date(
            item.submissionClosedAt * 1000
          ).toLocaleString()
        }
        return cls
      }
    )
    const announcements: Array<Announcement> = announcementResponses.map(
      (item: AnnouncementResponse) => {
        const announce: Announcement = {
          id: item.id,
          courseId: item.courseId,
          courseName: item.courseName,
          title: item.title,
          unread: item.unread,
          createdAt: new Date(item.createdAt * 1000).toLocaleString(),
        }
        return announce
      }
    )
    return {
      course,
      announcements,
      classes,
    }
  },
  data(): CourseData | undefined {
    return undefined
  },
  methods: {
    async openAnnouncement(
      event: { done: () => undefined },
      announcement: Announcement
    ) {
      const announcementDetail: Announcement = await this.$axios.$get(
        `/api/announcements/${announcement.id}`
      )
      const target = this.announcements.find(
        (item) => item.id === announcement.id
      )
      if (target) {
        target.message = announcementDetail.message
      }
      event.done()
    },
    closeAnnouncement(event: { done: () => undefined }) {
      event.done()
    },
  },
})
</script>
