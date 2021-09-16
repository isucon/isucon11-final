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
          <h1 class="text-2xl">講義</h1>
          <div class="mt-4">
            <Button color="primary" @click="showAddClassModal">新規登録</Button>
          </div>
          <template v-if="hasLoadCourseError">
            <InlineNotification type="error" class="mt-4">
              <template #title>APIエラーがあります</template>
              <template #message>科目情報の取得に失敗しました。</template>
            </InlineNotification>
          </template>
          <template v-if="hasLoadClassesError">
            <InlineNotification type="error" class="mt-4">
              <template #title>APIエラーがあります</template>
              <template #message>講義一覧の取得に失敗しました。</template>
            </InlineNotification>
          </template>
          <template v-if="hasDownloadSubmissionsError">
            <InlineNotification type="error" class="mt-4">
              <template #title>APIエラーがあります</template>
              <template #message
                >提出課題のダウンロードに失敗しました。</template
              >
            </InlineNotification>
          </template>
          <div class="mt-4">
            <ClassTable
              :classes="classes"
              :selected-class-idx="selectedClassIdx"
              @downloadSubmissions="downloadSubmissions"
              @registerScores="showRegisterScoresModal"
            />
          </div>
        </section>
      </div>
    </div>
    <AddClassModal
      :is-shown="visibleModal === 'AddClass'"
      :course-id="course.id"
      :course-name="course.name"
      @close="visibleModal = null"
      @completed="loadClasses"
    />
    <RegisterScoresModal
      :is-shown="visibleModal === 'RegisterScores'"
      :course-id="course.id"
      :course-name="course.name"
      :class-id="classId"
      :class-title="classTitle"
      @close="visibleModal = null"
    />
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'
import { notify } from '~/helpers/notification_helper'
import Button from '~/components/common/Button.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'
import ClassTable from '~/components/ClassTable.vue'
import AddClassModal from '~/components/AddClassModal.vue'
import RegisterScoresModal from '~/components/RegisterScoresModal.vue'
import { SyllabusCourse, ClassInfo } from '~/types/courses'

type modalKinds = 'AddClass' | 'RegisterScores' | null

type AsyncData = {
  course: SyllabusCourse
  hasLoadCourseError: boolean
}

type DataType = AsyncData & {
  visibleModal: modalKinds
  classes: ClassInfo[]
  selectedClassIdx: number | null
  hasLoadClassesError: boolean
  hasDownloadSubmissionsError: boolean
}

const initCourse: SyllabusCourse = {
  id: '',
  code: '',
  type: 'liberal-arts',
  name: '',
  description: '',
  credit: 0,
  period: 0,
  dayOfWeek: 'monday',
  teacher: '',
  keywords: '',
  status: 'registration',
}

export default Vue.extend({
  components: {
    Button,
    ClassTable,
    AddClassModal,
    RegisterScoresModal,
    InlineNotification,
  },
  middleware: 'is_teacher',
  async asyncData(ctx: Context): Promise<AsyncData> {
    try {
      const id = ctx.params.id
      const res = await ctx.$axios.get(`/api/courses/${id}`)
      const course: SyllabusCourse = res.data
      return { course, hasLoadCourseError: false }
    } catch (e) {
      console.error(e)
    }

    return { course: initCourse, hasLoadCourseError: true }
  },
  data(): DataType {
    return {
      visibleModal: null,
      course: initCourse,
      classes: [],
      selectedClassIdx: null,
      hasLoadCourseError: false,
      hasLoadClassesError: false,
      hasDownloadSubmissionsError: false,
    }
  },
  head: {
    title: 'ISUCHOLAR - 教員用科目詳細',
  },
  computed: {
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
    await this.loadClasses()
  },
  methods: {
    async loadClasses() {
      this.hasLoadClassesError = false
      try {
        const resClasses = await this.$axios.get<ClassInfo[]>(
          `/api/courses/${this.$route.params.id}/classes`
        )
        this.classes = resClasses.data ?? []
      } catch (e) {
        this.hasLoadClassesError = true
        notify('講義一覧の取得に失敗しました')
      }
    },
    async downloadSubmissions(classIdx: number) {
      this.selectedClassIdx = classIdx
      this.hasDownloadSubmissionsError = false
      try {
        await this.$axios
          .get(
            `/api/courses/${this.$route.params.id}/classes/${this.classId}/assignments/export`,
            {
              responseType: 'blob',
            }
          )
          .then((response) => {
            const link = document.createElement('a')
            link.href = window.URL.createObjectURL(response.data)
            link.download = `${this.classId}.zip`
            link.click()

            notify('提出課題のダウンロードに成功しました')
          })
      } catch (e) {
        this.hasDownloadSubmissionsError = true
        notify('提出課題のダウンロードに失敗しました')
      }
    },
    showAddClassModal() {
      this.visibleModal = 'AddClass'
    },
    showRegisterScoresModal(classIdx: number) {
      this.selectedClassIdx = classIdx
      this.visibleModal = 'RegisterScores'
    },
  },
})
</script>
