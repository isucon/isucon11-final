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
    const course: Course = await $axios.$get(`/api/courses/${params.id}`)
    // not implemented
    // const announcements: Array<Announcement> = (
    //   await $axios.$get(`/api/courses/${params.id}/announcements`)
    // ).map((item: Announcement) => {
    //   item.createdAt = new Date(item.createdAt).toLocaleString()
    //   return item
    // })
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
    const documents: Array<Document> = await $axios.$get(
      `/api/courses/${params.id}/documents`
    )
    const assignments: Array<Assignment> = await $axios.$get(
      `/api/courses/${params.id}/assignments`
    )
    const classInfo: Array<ClassInfo> = await $axios.$get(
      `/api/courses/${params.id}/classes`
    )
    const classworks: Array<Classwork> = classInfo.map((cls) => {
      return {
        ...cls,
        documents: documents.filter((item) => item.classId === cls.id),
        assignments: assignments.filter((item) => item.classId === cls.id),
      }
    })
    // const classworks: Array<Classwork> = [
    //   {
    //     id: 'classid1',
    //     part: 1,
    //     title: '第一回講義 イントロダクション',
    //     description: 'deeeeeeeeeeeeeeeeeeeeeeeeeeeeeescription!!!!!!!!!!!',
    //     documents: [
    //       {
    //         id: 'docid1',
    //         name: 'text1.pdf',
    //       },
    //       {
    //         id: 'docid2',
    //         name: 'text2.pdf',
    //       },
    //     ],
    //     assignments: [
    //       {
    //         id: 'assignmentid1',
    //         name: '課題1',
    //         description: 'kadai 1111111 dayo',
    //       },
    //       {
    //         id: 'assignmentid2',
    //         name: '課題2',
    //         description: 'kadai 2222222 dayo',
    //       },
    //     ],
    //   },
    //   {
    //     id: 'classid2',
    //     part: 2,
    //     title: '第二回講義 椅子の基礎',
    //     description: 'deeeeeeeeeeeeeeeeeeeeeeeeeeeeeescription!!!!!!!!!!!',
    //     documents: [
    //       {
    //         id: 'docid1',
    //         name: 'text1.pdf',
    //       },
    //       {
    //         id: 'docid2',
    //         name: 'text2.pdf',
    //       },
    //     ],
    //     assignments: [
    //       {
    //         id: 'assignmentid1',
    //         name: '課題1',
    //         description: 'kadai 1111111 dayo',
    //       },
    //       {
    //         id: 'assignmentid2',
    //         name: '課題2',
    //         description: 'kadai 2222222 dayo',
    //       },
    //     ],
    //   },
    // ]
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
