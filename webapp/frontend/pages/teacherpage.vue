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
            <Button color="primary" @click="visibleModal = 'AddCourse'"
              >新規登録</Button
            >
          </div>
          <div class="py-4">
            <Button color="primary" @click="visibleModal = 'SetCourseStatus'"
              >ステータス変更</Button
            >
          </div>
          <CourseTable
            :courses="courses"
            :selected-course-idx="selectedCourseIdx"
            :link="courseLink"
            @change="selectCourse"
            @paginate="moveCoursePage"
            @setStatus="showSetStatusModal"
            @addClass="showAddClassModal"
          />
        </section>

        <section class="mt-10">
          <h1 class="text-2xl">講義</h1>
          <div class="py-4">
            <Button color="primary" @click="visibleModal = 'AddClass'"
              >新規登録</Button
            >
          </div>
          <div class="py-4">
            <Button
              color="primary"
              @click="visibleModal = 'DownloadAssignments'"
              >提出課題のダウンロード</Button
            >
          </div>
          <div class="py-4">
            <Button color="primary" @click="visibleModal = 'RegisterScores'"
              >成績登録</Button
            >
          </div>
          <ClassTable
            :classes="classes"
            :selected-class-idx="selectedClassIdx"
            :link="classLink"
            @paginate="moveClassPage"
          />
        </section>
      </div>
    </div>
    <AddCourseModal
      :is-shown="visibleModal === 'AddCourse'"
      @close="visibleModal = null"
    />
    <SetCourseStatusModal
      :is-shown="visibleModal === 'SetCourseStatus'"
      @close="visibleModal = null"
    />
    <AddClassModal
      :is-shown="visibleModal === 'AddClass'"
      @close="visibleModal = null"
    />
    <RegisterScoresModal
      :is-shown="visibleModal === 'RegisterScores'"
      @close="visibleModal = null"
    />
    <DownloadAssignmentsModal
      :is-shown="visibleModal === 'DownloadAssignments'"
      @close="visibleModal = null"
    />
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { notify } from '~/helpers/notification_helper'
import Button from '~/components/common/Button.vue'
import CourseTable from '~/components/CourseTable.vue'
import ClassTable from '~/components/ClassTable.vue'
import AddCourseModal from '~/components/AddCourseModal.vue'
import SetCourseStatusModal from '~/components/SetCourseStatusModal.vue'
import AddClassModal from '~/components/AddClassModal.vue'
import RegisterScoresModal from '~/components/RegisterScoresModal.vue'
import DownloadAssignmentsModal from '~/components/DownloadAssignmentsModal.vue'
import { SyllabusCourse, ClassInfo, User } from '~/types/courses'
import { Link, parseLinkHeader } from '~/helpers/link_helper'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'

type modalKinds =
  | 'AddCourse'
  | 'SetCourseStatus'
  | 'AddClass'
  | 'RegisterScores'
  | 'DownloadAssignments'
  | null

type FacultyPageData = {
  visibleModal: modalKinds
  courses: SyllabusCourse[]
  selectedCourseIdx: number | null
  courseLink: Partial<Link>
  classes: ClassInfo[]
  selectedClassIdx: number | null
  classLink: Partial<Link>
}

const initLink = { prev: undefined, next: undefined }

export default Vue.extend({
  components: {
    Button,
    CourseTable,
    ClassTable,
    AddCourseModal,
    SetCourseStatusModal,
    AddClassModal,
    RegisterScoresModal,
    DownloadAssignmentsModal,
  },
  middleware: 'is_teacher',
  data(): FacultyPageData {
    return {
      visibleModal: null,
      courses: [],
      selectedCourseIdx: null,
      courseLink: { prev: undefined, next: undefined },
      classes: [],
      selectedClassIdx: null,
      classLink: { prev: undefined, next: undefined },
    }
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
    async loadClasses(query?: Record<string, any>) {
      if (!query) {
        this.classLink = Object.assign({}, initLink)
      }
      try {
        if (
          this.selectedCourseIdx === null ||
          !this.courses[this.selectedCourseIdx]
        ) {
          notify('科目が選択されていないか、存在しません')
          return
        }
        const courseId = this.courses[this.selectedCourseIdx].id
        const resClasses = await this.$axios.get<ClassInfo[]>(
          `/api/courses/${courseId}/classes`,
          { params: { ...query } }
        )
        this.classLink = Object.assign(
          {},
          this.courseLink,
          parseLinkHeader(resClasses.headers.link)
        )
        this.classes = resClasses.data
      } catch (e) {
        notify('講義の読み込みに失敗しました')
      }
    },
    async selectCourse(courseIdx: number) {
      this.selectedCourseIdx = courseIdx
      await this.loadClasses()
    },
    async moveCoursePage(query: URLSearchParams) {
      await this.loadCourses(urlSearchParamsToObject(query))
    },
    showSetStatusModal(courseIdx: number) {
      this.visibleModal = 'SetCourseStatus'
      console.log(courseIdx)
    },
    showAddClassModal(courseIdx: number) {
      this.visibleModal = 'AddClass'
      console.log(courseIdx)
    },
    async moveClassPage(query: URLSearchParams) {
      await this.loadClasses(urlSearchParamsToObject(query))
    },
  },
})
</script>
