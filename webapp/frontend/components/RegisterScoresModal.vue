<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-black font-bold justify-center mb-4">成績登録</p>
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
            id="params-classId"
            v-model="classId"
            label="講義"
            :options="classOptions"
          />
        </div>
        <template v-for="(param, index) in params">
          <div :key="`score-${index}`" class="flex flex-row space-x-2">
            <div class="flex-1">
              <TextField
                id="params-usercode"
                v-model="param.userCode"
                label="生徒の学籍番号"
                label-direction="vertical"
                type="text"
                placeholder="生徒の学籍番号を入力"
              />
            </div>
            <div class="flex-1">
              <TextField
                id="params-score"
                label="成績"
                label-direction="vertical"
                type="number"
                placeholder="成績を入力"
                :value="String(param.score)"
                @input="$set(param, 'score', Number($event))"
              />
            </div>
          </div>
        </template>
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
        <span class="block sm:inline">成績の登録に失敗しました</span>
        <span class="absolute top-0 bottom-0 right-0 px-4 py-3">
          <CloseIcon :classes="['text-red-500']" @click="hideAlert"></CloseIcon>
        </span>
      </div>
      <div class="px-4 py-3 flex justify-center">
        <Button @click="close"> 閉じる </Button>
        <Button @click="submit"> 登録 </Button>
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
import { Course, RegisterScoreRequest, ClassInfo, User } from '~/types/courses'

type SubmitFormData = {
  failed: boolean
  courseId: string
  classId: string
  courses: Course[]
  classes: ClassInfo[]
  params: RegisterScoreRequest
}

const initParams: RegisterScoreRequest = [
  {
    userCode: '',
    score: 0,
  },
]

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
      classId: '',
      courses: [],
      classes: [],
      params: initParams.map((o) => Object.assign({}, o)),
    }
  },
  computed: {
    courseOptions(): Option[] {
      return this.courses.map((course) => {
        return {
          text: course.name,
          value: course.id,
        }
      })
    },
    classOptions(): Option[] {
      return this.classes.map((cls) => {
        return {
          text: `第${cls.part}回 ${cls.title}`,
          value: cls.id,
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
    async courseId(newval: string) {
      if (newval) {
        await this.loadClasses()
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
    async loadClasses() {
      const classes: ClassInfo[] = await this.$axios.$get(
        `/api/courses/${this.courseId}/classes`
      )
      this.classes = classes
    },
    async submit() {
      try {
        await this.$axios.post(
          `/api/courses/${this.courseId}/classes/${this.classId}/assignments`,
          this.params
        )
        notify('成績の登録が完了しました')
        this.close()
      } catch (e) {
        notify('成績の登録に失敗しました')
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
