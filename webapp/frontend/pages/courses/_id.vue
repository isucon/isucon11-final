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
              v-for="announcement in announcements"
              :key="announcement.id"
              :announcement="announcement"
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
import { Course, Announcement, Classwork } from '@/interfaces/courses'

interface CourseData {
  course: Course
  announcements: Array<Announcement>
  classworks: Array<Classwork>
}

export default Vue.extend({
  // async asyncData({ params, $axios }): Promise<CourseData> {
  asyncData(): CourseData {
    // document.cookie =
    //   'session=MTYyNTEyMDY4N3xEdi1CQkFFQ180SUFBUkFCRUFBQV80bl9nZ0FEQm5OMGNtbHVad3dLQUFoMWMyVnlUbUZ0WlFaemRISnBibWNNRGdBTTVMMlE2SmVrNWFTcTZZT09Cbk4wY21sdVp3d0pBQWRwYzBGa2JXbHVCR0p2YjJ3Q0FnQUFCbk4wY21sdVp3d0lBQVoxYzJWeVNVUUdjM1J5YVc1bkRDWUFKREF4TWpNME5UWTNMVGc1WVdJdFkyUmxaaTB3TURBeExUQXdNREF3TURBd01EQXdNUT09fJIlIfeWK_QtzxaoXqG_u9sdAibbR_KMtMNpw2APz-4g; Path=/; Expires=Thu, 01 Jul 2021 07:24:47 GMT; Max-Age=3600'
    // const course: Course = await $axios.$get(`/api/courses/${params.id}`)
    const course: Course = {
      id: '01234567-89ab-cdef-0002-000000000001',
      code: 'ISU.F117',
      type: 'liberal-arts',
      name: '微分積分基礎',
      description: '微積分の基礎を学びます。',
      credit: 2,
      classroom: 'A101講義室',
      capacity: 100,
      teacher: '椅子昆',
      keywords: '数学 微分 積分 基礎',
      period: 1,
      day_of_week: 'monday',
      semester: 'first',
      year: 2021,
      required_courses: [],
    }
    // not implemented
    // const announcements: Array<Announcement> = await $axios.$get(`/api/courses/${params.id}/announcements`)
    const announcements: Array<Announcement> = [
      {
        id: 'announce2',
        title: '椅子概論 第一回課題の訂正',
        message:
          'Lorem ipsum dolor sit amet, ut alii voluptaria est, ad illum inimicus deterruisset eam. His eu bonorum adipisci definiebas, no vis nostrud conclusionemque. Ad his virtute accusata, pro habemus singulis temporibus ut, ne bonorum dolores euripidis quo. No nam amet erant intellegebat. Rationibus instructior id pri, vis case abhorreant ea, id sea meis feugiat.',
        createdAt: '6/17 10:00',
      },
      {
        id: 'announce1',
        title: '椅子概論 第一回講義日時 変更のお知らせ',
        message:
          'Lorem ipsum dolor sit amet, ut alii voluptaria est, ad illum inimicus deterruisset eam. His eu bonorum adipisci definiebas, no vis nostrud conclusionemque. Ad his virtute accusata, pro habemus singulis temporibus ut, ne bonorum dolores euripidis quo. No nam amet erant intellegebat. Rationibus instructior id pri, vis case abhorreant ea, id sea meis feugiat.',
        createdAt: '6/10 10:00',
      },
    ]
    // const classworks: Array<Classwork> = await $axios.$get(
    //   `/api/courses/${params.id}/classes`
    // )
    const classworks: Array<Classwork> = [
      {
        id: 'classid1',
        title: '第一回講義 イントロダクション',
        // startedAt: '6/10 18:00',
        description: 'deeeeeeeeeeeeeeeeeeeeeeeeeeeeeescription!!!!!!!!!!!',
        documents: [
          {
            id: 'docid1',
            name: 'text1.pdf',
          },
          {
            id: 'docid2',
            name: 'text2.pdf',
          },
        ],
        assignments: [
          {
            id: 'assignmentid1',
            name: '課題1',
            description: 'kadai 1111111 dayo',
            deadline: '6/17 18:00',
            // submitted: false,
          },
          {
            id: 'assignmentid2',
            name: '課題2',
            description: 'kadai 2222222 dayo',
            deadline: '6/17 18:00',
            // submitted: true,
          },
        ],
      },
      {
        id: 'classid2',
        title: '第二回講義 椅子の基礎',
        // startedAt: '6/17 18:00',
        description: 'deeeeeeeeeeeeeeeeeeeeeeeeeeeeeescription!!!!!!!!!!!',
        documents: [
          {
            id: 'docid1',
            name: 'text1.pdf',
          },
          {
            id: 'docid2',
            name: 'text2.pdf',
          },
        ],
        assignments: [
          {
            id: 'assignmentid1',
            name: '課題1',
            description: 'kadai 1111111 dayo',
            deadline: '6/17 18:00',
            // submitted: false,
          },
          {
            id: 'assignmentid2',
            name: '課題2',
            description: 'kadai 2222222 dayo',
            deadline: '6/17 18:00',
            // submitted: true,
          },
        ],
      },
    ]
    return {
      course,
      announcements,
      classworks,
    }
  },
})
</script>
