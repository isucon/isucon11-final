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
          <h1 class="text-2xl">講義</h1>
          <div class="py-4">
            <Button color="primary" @click="showAddClassModal">新規登録</Button>
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
import { notify } from '~/helpers/notification_helper'
import Button from '~/components/common/Button.vue'
import ClassTable from '~/components/ClassTable.vue'
import AddClassModal from '~/components/AddClassModal.vue'
import RegisterScoresModal from '~/components/RegisterScoresModal.vue'
import { SyllabusCourse, ClassInfo } from '~/types/courses'

type modalKinds = 'AddClass' | 'RegisterScores' | null

type DataType = {
  visibleModal: modalKinds
  course: SyllabusCourse
  classes: ClassInfo[]
  selectedClassIdx: number | null
}

export default Vue.extend({
  components: {
    Button,
    ClassTable,
    AddClassModal,
    RegisterScoresModal,
  },
  middleware: 'is_teacher',
  data(): DataType {
    return {
      visibleModal: null,
      course: {
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
      },
      classes: [],
      selectedClassIdx: null,
    }
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
    await this.loadCourseDetail()
    await this.loadClasses()
  },
  methods: {
    async loadCourseDetail() {
      try {
        const resCourse = await this.$axios.get<SyllabusCourse>(
          `/api/courses/${this.$route.params.id}`
        )
        this.course = resCourse.data
      } catch (e) {
        notify('科目の読み込みに失敗しました')
      }
    },
    async loadClasses() {
      try {
        const resClasses = await this.$axios.get<ClassInfo[]>(
          `/api/courses/${this.$route.params.id}/classes`
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

            notify('ダウンロードに成功しました')
          })
      } catch (e) {
        notify('ダウンロードに失敗しました')
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
