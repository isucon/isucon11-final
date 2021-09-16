<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-gray-800 font-bold flex justify-center mb-4">
        {{ title }}
      </p>
      <div class="mb-4">
        <p>{{ description }}</p>
        <p>提出先の課題や提出ファイルが正しいか確認してください。</p>
      </div>
      <div class="flex justify-center items-center my-8">
        <label
          class="
            mr-2
            w-auto
            rounded-md
            border border-transparent
            shadow-sm
            px-4
            py-2
            bg-primary-500
            text-center
            cursor-pointer
          "
        >
          <span class="text-sm font-medium text-white">ファイル選択</span>
          <input type="file" class="hidden" @change="onFileChanged" />
        </label>
        <template v-if="file !== null">
          <div class="flex flex-col justify-center">
            <div class="flex flex-row items-center text-gray-800 text-base">
              <span class="mr=2">{{ file.name }}</span
              ><CloseIcon @click="removeFile"></CloseIcon>
            </div>
          </div>
        </template>
        <template v-else>
          <span class="text-gray-800 text-base"
            >ファイルが選択されていません</span
          >
        </template>
      </div>
      <template v-if="failed">
        <InlineNotification type="error">
          <template #title>APIエラーがあります</template>
          <template #message>課題の提出に失敗しました。</template>
        </InlineNotification>
      </template>
      <div class="flex justify-center">
        <button
          type="button"
          class="
            mr-2
            rounded-md
            border border-primary-500
            shadow-sm
            px-4
            py-2
            bg-white
            text-sm
            font-medium
            text-primary-500
            w-20
          "
          @click="close"
        >
          閉じる
        </button>
        <button
          type="button"
          class="
            rounded-md
            border border-transparent
            shadow-sm
            px-4
            bg-primary-500
            text-sm
            font-medium
            text-white
            w-20
          "
          :disabled="file === null"
          @click="upload"
        >
          提出
        </button>
      </div>
    </Card>
  </Modal>
</template>

<script lang="ts">
import Vue from 'vue'
import { notify } from '~/helpers/notification_helper'
import Card from '~/components/common/Card.vue'
import Modal from '~/components/common/Modal.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'

type SubmitFormData = {
  file: File | null
  failed: boolean
}

export default Vue.extend({
  name: 'SubmitModal',
  components: {
    Card,
    Modal,
    InlineNotification,
  },
  props: {
    isShown: {
      type: Boolean,
      default: false,
      required: true,
    },
    courseName: {
      type: String,
      required: true,
    },
    classTitle: {
      type: String,
      required: true,
    },
    classId: {
      type: String,
      required: true,
    },
  },
  data(): SubmitFormData {
    return {
      file: null,
      failed: false,
    }
  },
  computed: {
    title(): string {
      return `${this.classTitle} 課題提出`
    },
    description(): string {
      return `これは科目 ${this.courseName} の講義 ${this.classTitle} の課題提出フォームです。 `
    },
  },
  methods: {
    onFileChanged(event: Event) {
      if (
        !(event.target instanceof HTMLInputElement) ||
        event.target.files === null
      ) {
        return
      }
      this.file = event.target.files[0]
      event.target.value = ''
    },
    removeFile() {
      this.file = null
    },
    upload() {
      if (this.file === null) {
        return
      }
      const formData = new FormData()
      formData.append('file', this.file)
      this.$axios
        .post(
          `/api/courses/${this.$route.params.id}/classes/${this.classId}/assignments`,
          formData
        )
        .then(() => {
          notify('課題の提出が完了しました')
          this.$emit('submitted')
          this.close()
        })
        .catch(() => {
          notify('課題の提出に失敗しました')
          this.showAlert()
        })
    },
    close() {
      this.file = null
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
