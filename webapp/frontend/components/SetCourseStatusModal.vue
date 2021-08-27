<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-black font-bold justify-center mb-4">
        ステータス変更
      </p>
      <div class="flex flex-col space-y-4 mb-4">
        <div class="flex-1">
          <Select
            id="params-courseId"
            v-model="courseId"
            label="科目"
            :options="courseOptions"
          />
        </div>
        <div class="flex-1">
          <Select
            id="params-status"
            v-model="params.status"
            label="ステータス"
            :options="statusOptions"
          />
        </div>
      </div>
      <div
        v-if="failed"
        class="
          bg-red-100
          border border-red-400
          text-red-700
          px-4
          py-3
          rounded
          relative
        "
        role="alert"
      >
        <strong class="font-bold">エラー</strong>
        <span class="block sm:inline">ステータスの変更に失敗しました</span>
        <span class="absolute top-0 bottom-0 right-0 px-4 py-3">
          <CloseIcon :classes="['text-red-500']" @click="hideAlert"></CloseIcon>
        </span>
      </div>
      <div class="px-4 py-3 flex justify-center">
        <Button @click="close"> 閉じる </Button>
        <Button color="primary" @click="submit"> 変更 </Button>
      </div>
    </Card>
  </Modal>
</template>

<script lang="ts">
import Vue from 'vue'
import { notify } from '~/helpers/notification_helper'
import Card from '~/components/common/Card.vue'
import Modal from '~/components/common/Modal.vue'
import CloseIcon from '~/components/common/CloseIcon.vue'
import Button from '~/components/common/Button.vue'
import Select, { Option } from '~/components/common/Select.vue'
import { Course, SetCourseStatusRequest, User } from '~/types/courses'

type SubmitFormData = {
  failed: boolean
  courseId: string
  courses: Course[]
  params: SetCourseStatusRequest
}

const statusOptions = [
  {
    text: '履修登録受付中',
    value: 'registration',
  },
  {
    text: '開講中',
    value: 'in-progress',
  },
  {
    text: '閉講済み',
    value: 'closed',
  },
]

const initParams: SetCourseStatusRequest = {
  status: 'registration',
}

export default Vue.extend({
  components: {
    Card,
    Modal,
    CloseIcon,
    Button,
    Select,
  },
  props: {
    isShown: {
      type: Boolean,
      default: false,
      required: true,
    },
  },
  data(): SubmitFormData {
    return {
      failed: false,
      courseId: '',
      courses: [],
      params: Object.assign({}, initParams),
    }
  },
  computed: {
    statusOptions() {
      return statusOptions
    },
    courseOptions(): Option[] {
      return this.courses.map((course) => {
        return {
          text: course.name,
          value: course.id,
        }
      })
    },
  },
  watch: {
    async isShown(newval: boolean) {
      if (newval) {
        await this.loadCourses()
      }
    },
  },
  methods: {
    async loadCourses() {
      const user: User = await this.$axios.$get(`/api/users/me`)
      const courses: Course[] = await this.$axios.$get(
        `/api/syllabus?teacher=${user.name}`
      )
      this.courses = courses
    },
    async submit() {
      try {
        await this.$axios.put(
          `/api/courses/${this.courseId}/status`,
          this.params
        )
        notify('ステータス変更が完了しました')
        this.close()
      } catch (e) {
        notify('ステータス変更に失敗しました')
        this.showAlert()
      }
    },
    close() {
      this.params = Object.assign({}, initParams)
      this.hideAlert()
      this.$emit('close')
    },
    showAlert() {
      this.failed = true
    },
    hideAlert() {
      this.failed = false
    },
  },
})
</script>