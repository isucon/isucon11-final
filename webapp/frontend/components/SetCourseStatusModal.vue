<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card class="w-96 max-w-full">
      <p class="text-2xl text-gray-800 font-bold justify-center mb-4">
        科目の状態を変更
      </p>
      <div class="flex flex-col space-y-4 mb-4">
        <div class="flex-1">
          <LabeledText label="科目名" :value="courseName"> </LabeledText>
        </div>
        <div class="flex-1">
          <Select
            id="params-status"
            v-model="params.status"
            label="科目の状態"
            :options="statusOptions"
          />
        </div>
      </div>
      <template v-if="failed">
        <InlineNotification type="error">
          <template #title>APIエラーがあります</template>
          <template #message>科目の状態の変更に失敗しました。</template>
        </InlineNotification>
      </template>
      <div class="flex justify-center gap-2 mt-4">
        <Button w-class="w-28" @click="close"> 閉じる </Button>
        <Button
          w-class="w-28"
          color="primary"
          :disable="params.status !== ''"
          @click="submit"
        >
          変更
        </Button>
      </div>
    </Card>
  </Modal>
</template>

<script lang="ts">
import Vue, { PropType } from 'vue'
import { notify } from '~/helpers/notification_helper'
import Card from '~/components/common/Card.vue'
import Modal from '~/components/common/Modal.vue'
import Button from '~/components/common/Button.vue'
import Select from '~/components/common/Select.vue'
import LabeledText from '~/components/common/LabeledText.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'
import { CourseStatus, SetCourseStatusRequest } from '~/types/courses'

type SetCourseStatusRequestWithDefault = {
  status: SetCourseStatusRequest['status'] | ''
}

type Data = {
  failed: boolean
  params: SetCourseStatusRequestWithDefault
}

const statusOptions = [
  {
    text: '履修登録期間',
    value: 'registration',
  },
  {
    text: '講義期間',
    value: 'in-progress',
  },
  {
    text: '終了済み',
    value: 'closed',
  },
]

const initParams: { status: CourseStatus | '' } = {
  status: 'registration',
}

export default Vue.extend({
  components: {
    Card,
    Modal,
    InlineNotification,
    Button,
    Select,
    LabeledText,
  },
  props: {
    courseName: {
      type: String,
      required: true,
    },
    courseId: {
      type: String,
      required: true,
    },
    courseStatus: {
      type: String as PropType<CourseStatus>,
      required: true,
    },
    isShown: {
      type: Boolean,
      default: false,
      required: true,
    },
  },
  data(): Data {
    return {
      failed: false,
      params: Object.assign({}, initParams),
    }
  },
  computed: {
    statusOptions() {
      return statusOptions
    },
  },
  watch: {
    courseStatus(newval) {
      this.params.status = newval
    },
  },
  methods: {
    async submit() {
      if (this.params.status === '') return

      try {
        await this.$axios.put(
          `/api/courses/${this.courseId}/status`,
          this.params
        )
        notify('科目の状態変更が完了しました')
        this.$emit('completed')
        this.close()
      } catch (e) {
        notify('科目の状態変更に失敗しました')
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
