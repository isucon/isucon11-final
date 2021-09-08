<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12 mt-8 mb-8 rounded">
      <div class="flex-1 flex-col">
        <div
          class="
            bg-yellow-100
            border border-yellow-400
            text-yellow-700
            px-4
            py-3
            rounded
            relative
            mb-4
          "
          role="alert"
        >
          <span class="block sm:inline"
            >本ページは工事中であり、UIは将来的に刷新される予定です。</span
          >
        </div>
        <section>
          <h1 class="text-2xl">科目</h1>
          <div class="py-4">
            <Button color="primary" @click="showAddCourseModal"
              >新規登録</Button
            >
          </div>
          <CourseTable
            :courses="courses"
            :selected-course-idx="selectedCourseIdx"
            :link="courseLink"
            @paginate="moveCoursePage"
            @setStatus="showSetStatusModal"
          />
        </section>
      </div>
    </div>
    <AddCourseModal
      :is-shown="visibleModal === 'AddCourse'"
      @close="visibleModal = null"
      @completed="loadCourses"
    />
    <SetCourseStatusModal
      :is-shown="visibleModal === 'SetCourseStatus'"
      :course-id="courseId"
      :course-name="courseName"
      :course-status="courseStatus"
      @close="visibleModal = null"
      @completed="loadCourses"
    />
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { notify } from '~/helpers/notification_helper'
import Button from '~/components/common/Button.vue'
import CourseTable from '~/components/CourseTable.vue'
import AddCourseModal from '~/components/AddCourseModal.vue'
import SetCourseStatusModal from '~/components/SetCourseStatusModal.vue'
import { SyllabusCourse, User } from '~/types/courses'
import { Link, parseLinkHeader } from '~/helpers/link_helper'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'

type modalKinds = 'AddCourse' | 'SetCourseStatus' | null

type DataType = {
  visibleModal: modalKinds
  courses: SyllabusCourse[]
  selectedCourseIdx: number | null
  courseLink: Partial<Link>
}

const initLink = { prev: undefined, next: undefined }

export default Vue.extend({
  components: {
    Button,
    CourseTable,
    AddCourseModal,
    SetCourseStatusModal,
  },
  middleware: 'is_teacher',
  data(): DataType {
    return {
      visibleModal: null,
      courses: [],
      selectedCourseIdx: null,
      courseLink: { prev: undefined, next: undefined },
    }
  },
  computed: {
    courseId(): string {
      return this.selectedCourseIdx !== null
        ? this.courses[this.selectedCourseIdx].id
        : ''
    },
    courseName(): string {
      return this.selectedCourseIdx !== null
        ? this.courses[this.selectedCourseIdx].name
        : ''
    },
    courseStatus(): string {
      return this.selectedCourseIdx !== null
        ? this.courses[this.selectedCourseIdx].status
        : ''
    },
  },
  async created() {
    await this.loadCourses()
  },
  methods: {
    async loadCourses(query?: Record<string, any>) {
      if (!query) {
        this.courseLink = Object.assign({}, initLink)
      }
      try {
        const resUser = await this.$axios.get<User>(`/api/users/me`)
        const user = resUser.data
        const resCourses = await this.$axios.get<SyllabusCourse[]>(
          `/api/courses`,
          { params: { ...query, teacher: user.name } }
        )
        this.courses = resCourses.data
        this.courseLink = Object.assign(
          {},
          this.courseLink,
          parseLinkHeader(resCourses.headers.link)
        )
        this.courses = resCourses.data
      } catch (e) {
        notify('科目の読み込みに失敗しました')
      }
    },
    async moveCoursePage(query: URLSearchParams) {
      await this.loadCourses(urlSearchParamsToObject(query))
    },
    showAddCourseModal() {
      this.visibleModal = 'AddCourse'
    },
    showSetStatusModal(courseIdx: number) {
      this.selectedCourseIdx = courseIdx
      this.visibleModal = 'SetCourseStatus'
    },
  },
})
</script>
