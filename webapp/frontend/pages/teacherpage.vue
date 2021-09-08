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
            <Button color="primary" @click="visibleModal = 'RegisterScores'"
              >成績登録</Button
            >
          </div>
          <ClassTable
            :classes="classes"
            :selected-class-idx="selectedClassIdx"
            @downloadSubmissions="downloadSubmissions"
            @registerScores="showRegisterScoresModal"
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
    <AddClassModal
      :is-shown="visibleModal === 'AddClass'"
      :course-id="courseId"
      :course-name="courseName"
      @close="visibleModal = null"
      @completed="loadClasses"
    />
    <RegisterScoresModal
      :is-shown="visibleModal === 'RegisterScores'"
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
import { SyllabusCourse, ClassInfo, User } from '~/types/courses'
import { Link, parseLinkHeader } from '~/helpers/link_helper'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'

type modalKinds =
  | 'AddCourse'
  | 'SetCourseStatus'
  | 'AddClass'
  | 'RegisterScores'
  | null

type FacultyPageData = {
  visibleModal: modalKinds
  courses: SyllabusCourse[]
  selectedCourseIdx: number | null
  courseLink: Partial<Link>
  classes: ClassInfo[]
  selectedClassIdx: number | null
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
    classId(): string {
      return this.selectedClassIdx !== null
        ? this.classes[this.selectedClassIdx].id
        : ''
    },
    classTitle(): string {
      return this.selectedClassIdx !== null
        ? this.classes[this.selectedClassIdx].title
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
    async loadClasses() {
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
          `/api/courses/${courseId}/classes`
        )
        this.classes = resClasses.data
      } catch (e) {
        notify('講義の読み込みに失敗しました')
      }
    },
    async downloadSubmissions(classIdx: number) {
      this.selectedClassIdx = classIdx
      try {
        await this.$axios
          .get(
            `/api/courses/${this.courseId}/classes/${this.classId}/assignments/export`,
            {
              responseType: 'blob',
            }
          )
          .then((response) => {
            const link = document.createElement('a')
            link.href = window.URL.createObjectURL(response.data)
            link.download = `${this.classId}.zip`
            link.click()

            notify('ダウンロードに成功しました')
          })
      } catch (e) {
        notify('ダウンロードに失敗しました')
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
      this.selectedCourseIdx = courseIdx
      this.visibleModal = 'SetCourseStatus'
    },
    showAddClassModal(courseIdx: number) {
      this.selectedCourseIdx = courseIdx
      this.visibleModal = 'AddClass'
    },
    showRegisterScoresModal(classIdx: number) {
      this.visibleModal = 'RegisterScores'
      console.log(classIdx)
    },
  },
})
</script>
