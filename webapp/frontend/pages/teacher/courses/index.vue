<template>
  <div>
    <div
      class="py-10 px-8 bg-white shadow-lg w-192 max-w-full mt-8 mb-8 rounded"
    >
      <div class="flex-1 flex-col">
        <InlineNotification type="warn">
          <template #title>本ページは工事中です。</template>
          <template #message>UIは将来的に刷新される可能性があります。</template>
        </InlineNotification>

        <section class="mt-8">
          <div class="flex justify-between">
            <div class="text-2xl">科目一覧</div>
            <Button color="primary" @click="showAddCourseModal"
              >新規登録</Button
            >
          </div>
          <template v-if="hasError">
            <InlineNotification type="error" class="mt-4">
              <template #title>APIエラーがあります</template>
              <template #message>科目一覧の取得に失敗しました。</template>
            </InlineNotification>
          </template>
          <div class="mt-4">
            <CourseTable
              :courses="courses"
              :selected-course-idx="selectedCourseIdx"
              :link="courseLink"
              @paginate="moveCoursePage"
              @setStatus="showSetStatusModal"
              @addAnnouncement="showAddAnnouncementModal"
            />
          </div>
        </section>
      </div>
    </div>
    <AddAnnouncementModal
      :is-shown="visibleModal === 'AddAnnouncement'"
      :course-id="courseId"
      @close="visibleModal = null"
    />
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
import InlineNotification from '~/components/common/InlineNotification.vue'
import CourseTable from '~/components/CourseTable.vue'
import AddCourseModal from '~/components/AddCourseModal.vue'
import SetCourseStatusModal from '~/components/SetCourseStatusModal.vue'
import AddAnnouncementModal from '~/components/AddAnnouncementModal.vue'
import { SyllabusCourse, User } from '~/types/courses'
import { Link, parseLinkHeader } from '~/helpers/link_helper'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'

type modalKinds = 'AddCourse' | 'SetCourseStatus' | 'AddAnnouncement' | null

type DataType = {
  visibleModal: modalKinds
  courses: SyllabusCourse[]
  selectedCourseIdx: number | null
  courseLink: Partial<Link>
  hasError: boolean
}

const initLink = { prev: undefined, next: undefined }

export default Vue.extend({
  components: {
    Button,
    CourseTable,
    AddCourseModal,
    SetCourseStatusModal,
    AddAnnouncementModal,
    InlineNotification,
  },
  middleware: 'is_teacher',
  data(): DataType {
    return {
      visibleModal: null,
      courses: [],
      selectedCourseIdx: null,
      courseLink: { prev: undefined, next: undefined },
      hasError: false,
    }
  },
  head: {
    title: 'ISUCHOLAR - 教員用講義一覧',
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
    async loadCourses(path = `/api/courses`, query?: Record<string, any>) {
      this.hasError = false
      if (!query) {
        this.courseLink = Object.assign({}, initLink)
      }
      try {
        const resUser = await this.$axios.get<User>(`/api/users/me`)
        const user = resUser.data
        const resCourses = await this.$axios.get<SyllabusCourse[]>(path, {
          params: { ...query, teacher: user.name },
        })
        this.courses = resCourses.data ?? []
        this.courseLink = Object.assign(
          {},
          this.courseLink,
          parseLinkHeader(resCourses.headers.link)
        )
      } catch (e) {
        this.hasError = true
        notify('科目一覧の取得に失敗しました')
      }
    },
    async moveCoursePage(path: string | undefined, query: URLSearchParams) {
      await this.loadCourses(path, urlSearchParamsToObject(query))
    },
    showAddCourseModal() {
      this.visibleModal = 'AddCourse'
    },
    showSetStatusModal(courseIdx: number) {
      this.selectedCourseIdx = courseIdx
      this.visibleModal = 'SetCourseStatus'
    },
    showAddAnnouncementModal(courseIdx: number) {
      this.selectedCourseIdx = courseIdx
      this.visibleModal = 'AddAnnouncement'
    },
  },
})
</script>
