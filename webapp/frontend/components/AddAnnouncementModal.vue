<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card class="w-96 max-w-full">
      <p class="text-2xl text-gray-800 font-bold justify-center mb-4">
        お知らせ登録
      </p>
      <div class="flex flex-col space-y-4 mb-4">
        <TextField
          id="params-title"
          v-model="params.title"
          label="お知らせタイトル"
          label-direction="vertical"
          type="text"
          placeholder="お知らせのタイトルを入力してください"
        />
        <TextField
          id="params-message"
          v-model="params.message"
          label="お知らせ内容"
          label-direction="vertical"
          type="text"
          placeholder="お知らせの内容を入力してください"
        />
      </div>
      <template v-if="hasError">
        <InlineNotification type="error">
          <template #title>APIエラーがあります</template>
          <template #message>お知らせの登録に失敗しました。</template>
        </InlineNotification>
      </template>
      <div class="flex justify-center gap-2 mt-4">
        <Button w-class="w-28" @click="close"> 閉じる </Button>
        <Button w-class="w-28" color="primary" @click="submit"> 登録 </Button>
      </div>
    </Card>
  </Modal>
</template>

<script lang="ts">
import Vue from 'vue'
import { ulid } from 'ulid'
import { notify } from '~/helpers/notification_helper'
import Card from '~/components/common/Card.vue'
import Modal from '~/components/common/Modal.vue'
import Button from '~/components/common/Button.vue'
import TextField from '~/components/common/TextField.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'
import { AddAnnouncementRequest } from '~/types/courses'

type SubmitFormData = {
  hasError: boolean
  params: AddAnnouncementRequest
}

const initParams: AddAnnouncementRequest = {
  id: '',
  courseId: '',
  title: '',
  message: '',
}

export default Vue.extend({
  components: {
    Card,
    Modal,
    Button,
    TextField,
    InlineNotification,
  },
  props: {
    isShown: {
      type: Boolean,
      default: false,
      required: true,
    },
    courseId: {
      type: String,
      required: true,
    },
  },
  data(): SubmitFormData {
    return {
      hasError: false,
      params: Object.assign({}, initParams),
    }
  },
  methods: {
    async submit() {
      try {
        const params = {
          ...this.params,
          id: ulid(),
          courseId: this.courseId,
        }
        await this.$axios.post(`/api/announcements`, params)

        notify('お知らせ登録が完了しました')
        this.close()
      } catch (e) {
        this.showAlert()
        notify('お知らせ登録に失敗しました')
      }
    },
    close() {
      this.params = Object.assign({}, initParams)
      this.hideAlert()
      this.$emit('close')
    },
    showAlert() {
      this.hasError = true
    },
    hideAlert() {
      this.hasError = false
    },
  },
})
</script>
