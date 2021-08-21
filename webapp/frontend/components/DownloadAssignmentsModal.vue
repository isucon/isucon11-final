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
        <Button @click="submit"> ダウンロード </Button>
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
import { Course, ClassInfo } from '~/types/courses'

type AsyncLoadData = {
  courses: Course[]
  classes: ClassInfo[]
}

type SubmitFormData = {
  failed: boolean
  courseId: string
  classId: string
} & AsyncLoadData

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
  // async asyncData(): Promise<MyCourseData> {
  //   const courses: Course[] = await $axios.$get('/api/syllabus', { teacher:  })
  //   return courses
  // },
  data(): SubmitFormData {
    return {
      failed: false,
      courseId: '',
      classId: '',
      courses: [
        {
          id: 'ididididididid',
          code: 'X9991',
          type: 'liberal-arts',
          name: 'testtest',
          description: 'deeeeeeeeeeeescription',
          credit: 2,
          period: 4,
          dayOfWeek: 'tuesday',
          teacher: '伊藤 翔',
          keywords: 'hoge',
        },
        {
          id: 'IDIDIDIDIDIDIDIDID',
          code: 'X9992',
          type: 'liberal-arts',
          name: 'hogehogefugafuga',
          description: 'deeeeeeeeeeeescriiiiiiiiiiiiiiiiiiption',
          credit: 2,
          period: 4,
          dayOfWeek: 'tuesday',
          teacher: '伊藤 翔',
          keywords: 'fuga',
        },
      ],
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
  methods: {
    async submit() {
      try {
        await this.$axios.get(
          `/api/courses/${this.courseId}/classes/${this.classId}/assignments/export`
        )
        notify('ダウンロードに成功しました')
        this.close()
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
