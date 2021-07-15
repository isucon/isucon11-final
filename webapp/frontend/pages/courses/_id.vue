<template>
  <div>
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
              v-for="(announcement, index) in announcements"
              :key="announcement.id"
              :announcement="announcement"
              @open="openAnnouncement(announcement, index)"
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
</template>

<script lang="ts">
import Vue from 'vue'
import { Course, Announcement, ClassInfo } from '~/types/courses'
import Card from '~/components/common/Card.vue'
import AnnouncementCard from '~/components/Announcement.vue'
import ClassInfoCard from '~/components/ClassInfo.vue'

type CourseData = {
  course: Course
  announcements: Array<Announcement>
  classes: Array<ClassInfo>
}

// TODO: announcement周りは#166~#168あたりの改修が済み次第

export default Vue.extend({
  components: {
    Card,
    AnnouncementCard,
    ClassInfoCard,
  },
  middleware: 'is_loggedin',
  async asyncData({ params, $axios }): Promise<CourseData> {
    const [course, classes, announcements] = await Promise.all([
      $axios.$get(`/api/syllabus/${params.id}`),
      $axios.$get(`/api/courses/${params.id}/classes`),
      // $axios.$get(`/api/announcements?course_id=${params.id}`), // not implemented
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
    ])
    // api is not implemented
    // const announcements = announcements.map((item: Announcement) => {
    //   item.createdAt = new Date(item.createdAt).toLocaleString()
    //   return item
    // })
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
    async openAnnouncement(announcement: Announcement, index: number) {
      const announcementDetail: Announcement = await this.$axios.$get(
        `/api/announcements/${announcement.id}`
      )
      this.announcements[index].message = announcementDetail.message
    },
  },
})
</script>
