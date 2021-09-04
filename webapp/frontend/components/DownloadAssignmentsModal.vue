<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-black font-bold justify-center mb-4">
        提出課題のダウンロード
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
            id="params-classId"
            v-model="classId"
            label="講義"
            :options="classOptions"
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
        <span class="block sm:inline"
          >提出課題のダウンロードに失敗しました</span
        >
        <span class="absolute top-0 bottom-0 right-0 px-4 py-3">
          <CloseIcon :classes="['text-red-500']" @click="hideAlert"></CloseIcon>
        </span>
      </div>
      <div class="px-4 py-3 flex justify-center">
        <Button @click="close"> 閉じる </Button>
        <Button color="primary" @click="submit"> ダウンロード </Button>
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
import { Course, ClassInfo, User } from '~/types/courses'

type SubmitFormData = {
  failed: boolean
  courseId: string
  classId: string
  courses: Course[]
  classes: ClassInfo[]
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
      classId: '',
      courses: [],
      classes: [],
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
      try {
        const resUser = await this.$axios.get<User>(`/api/users/me`)
        const user = resUser.data
        const resCourses = await this.$axios.get<Course[]>(
          `/api/syllabus?teacher=${user.name}`
        )
        this.courses = resCourses.data
      } catch (e) {
        notify('科目の読み込みに失敗しました')
      }
    },
    async loadClasses() {
      try {
        const res = await this.$axios.get<ClassInfo[]>(
          `/api/courses/${this.courseId}/classes`
        )
        this.classes = res.data
      } catch (e) {
        notify('講義の読み込みに失敗しました')
      }
    },
    async submit() {
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
            link.href = window.URL.createObjectURL(response)
            link.download = `${this.classId}.zip`
            link.click()

            notify('ダウンロードに成功しました')
            this.close()
          })
      } catch (e) {
        notify('ダウンロードに失敗しました')
        this.showAlert()
      }
    },
    close() {
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
