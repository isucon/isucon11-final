<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-gray-800 font-bold justify-center mb-4">
        講義登録
      </p>
      <div class="flex flex-col space-y-4 mb-4">
        <div class="flex-1">
          <LabeledText label="科目名" :value="courseName" />
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
              @input="$set(params, 'part', Number($event))"
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
      <template v-if="failed">
        <InlineNotification type="error">
          <template #title>APIエラーがあります</template>
          <template #message>講義の登録に失敗しました。</template>
        </InlineNotification>
      </template>
      <div class="flex justify-center gap-2 mt-4">
        <Button w-class="w-28" @click="close">閉じる</Button>
        <Button color="primary" w-class="w-28" @click="submit">登録</Button>
      </div>
    </Card>
  </Modal>
</template>

<script lang="ts">
import Vue from 'vue'
import { notify } from '~/helpers/notification_helper'
import Card from '~/components/common/Card.vue'
import Modal from '~/components/common/Modal.vue'
import Button from '~/components/common/Button.vue'
import LabeledText from '~/components/common/LabeledText.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'
import { AddClassRequest } from '~/types/courses'

type SubmitFormData = {
  failed: boolean
  params: AddClassRequest
}

const initParams: AddClassRequest = {
  part: 1,
  title: '',
  description: '',
}

export default Vue.extend({
  components: {
    Card,
    Modal,
    Button,
    LabeledText,
    InlineNotification,
  },
  props: {
    courseId: {
      type: String,
      required: true,
    },
    courseName: {
      type: String,
      required: true,
    },
    isShown: {
      type: Boolean,
      default: false,
      required: true,
    },
  },
  data(): SubmitFormData {
    return {
      failed: false,
      params: Object.assign({}, initParams),
    }
  },
  methods: {
    async submit() {
      try {
        await this.$axios.post(
          `/api/courses/${this.courseId}/classes`,
          this.params
        )
        notify('講義の登録が完了しました')
        this.$emit('completed')
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
