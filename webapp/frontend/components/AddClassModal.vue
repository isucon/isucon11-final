<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-black font-bold justify-center mb-4">講義登録</p>
      <div class="flex flex-col space-y-4 mb-4">
        <div class="flex-1">
          <Select
            id="params-courseId"
            v-model="courseId"
            label="科目"
            :options="courseOptions"
          />
        </div>
        <div class="flex flex-row space-x-2">
          <div class="flex-2">
            <TextField
              id="params-part"
              label="講義回"
              label-direction="vertical"
              type="number"
              placeholder="講義回を入力"
              :value="String(params.part)"
              @input="updateNumberParam('part', $event)"
            />
          </div>
          <div class="flex-1">
            <TextField
              id="params-title"
              v-model="params.title"
              label="講義タイトル"
              label-direction="vertical"
              type="text"
              placeholder="タイトルを入力"
            />
          </div>
        </div>
        <div class="flex-1">
          <TextField
            id="params-description"
            v-model="params.description"
            label="講義詳細"
            label-direction="vertical"
            type="text"
            placeholder="講義詳細を入力"
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
        <span class="block sm:inline">講義の登録に失敗しました</span>
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
import { Course, AddClassRequest } from '~/types/courses'

type MyCourseData = {
  courses: Course[]
}

type SubmitFormData = {
  failed: boolean
  courseId: string
  params: AddClassRequest
} & MyCourseData

const initParams: AddClassRequest = {
  part: 1,
  title: '',
  description: '',
  createdAt: 0,
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
  // async asyncData(): Promise<MyCourseData> {
  //   const courses: Course[] = await $axios.$get('/api/syllabus', { teacher:  })
  //   return courses
  // },
  data(): SubmitFormData {
    return {
      failed: false,
      courseId: '',
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
      params: Object.assign({}, initParams),
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
  },
  methods: {
    updateNumberParam(fieldname: string, value: string) {
      this.$set(this.params, fieldname, Number(value))
    },
    async submit() {
      this.params.createdAt = new Date().getTime()
      try {
        await this.$axios.post(
          `/api/courses/${this.courseId}/classes`,
          this.params
        )
        notify('講義の登録が完了しました')
        this.close()
      } catch (e) {
        notify('講義の登録に失敗しました')
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
