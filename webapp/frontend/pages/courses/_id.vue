<template>
  <div>
    <card>
      <div class="flex-1 flex-col">
        <h1 class="text-2xl mb-4 font-bold">{{ course.name }}</h1>
        <tabs
          :tabs="[
            { id: 'announcements', label: 'お知らせ' },
            { id: 'classworks', label: '講義情報' },
          ]"
        >
          <template slot="announcements">
            <announcement
              v-for="(announcement, index) in announcements"
              :key="announcement.id"
              :announcement="announcement"
              @open="openAnnouncement(announcement, index)"
            />
          </template>
          <template slot="classworks">
            <classwork
              v-for="classwork in classworks"
              :key="classwork.id"
              :course="course"
              :classwork="classwork"
            />
          </template>
        </tabs>
      </div>
    </card>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  Course,
  Announcement,
  ClassInfo,
  Classwork,
  Document,
  Assignment,
} from '@/interfaces/courses'

type CourseData = {
  course: Course
  announcements: Array<Announcement>
  classworks: Array<Classwork>
}

export default Vue.extend({
  async asyncData({ params, $axios }): Promise<CourseData> {
    const [course, announcements, documents, assignments, classInfo] =
      await Promise.all([
        $axios.$get(`/api/courses/${params.id}`),
        // $axios.$get(`/api/courses/${params.id}/announcements`), // not implemented
        // tentative
        (() => {
          const announcements: Array<Announcement> = [
            {
              id: '01234567-89ab-cdef-0010-000000000001',
              courseName: '微分積分基礎',
              title: 'The third class will be cancelled',
              createdAt: new Date(1625573684000).toLocaleString(),
            },
            {
              id: '01234567-89ab-cdef-0010-000000000002',
              courseName: '微分積分基礎',
              title: 'Comments for your assignments',
              createdAt: new Date(1625573684000).toLocaleString(),
            },
          ]
          return announcements
        })(),
        $axios.$get(`/api/courses/${params.id}/documents`),
        $axios.$get(`/api/courses/${params.id}/assignments`),
        $axios.$get(`/api/courses/${params.id}/classes`),
      ])
    // api is not implemented
    // const announcements = announcements.map((item: Announcement) => {
    //   item.createdAt = new Date(item.createdAt).toLocaleString()
    //   return item
    // })
    const classworks: Array<Classwork> = classInfo.map((cls: ClassInfo) => {
      return {
        ...cls,
        documents: documents.filter(
          (item: Document) => item.classId === cls.id
        ),
        assignments: assignments.filter(
          (item: Assignment) => item.classId === cls.id
        ),
      }
    })
    return {
      course,
      announcements,
      classworks,
    }
  },
  data(): CourseData | undefined {
    return undefined
  },
  methods: {
    async openAnnouncement(announcement: Announcement, index: number) {
      const announcementDetail: Announcement = await this.$axios.$get(
        `/api/announcements/${announcement.id}`
      )
      this.announcements[index].message = announcementDetail.message
    },
  },
})
</script>
