<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-black font-bold flex justify-center mb-4">
        {{ title }}
      </p>
      <div class="mb-4">
        <p>{{ description }}</p>
        <p>提出先の課題や提出ファイルが正しいか確認してください。</p>
      </div>
      <div class="flex justify-center items-center mb-4">
        <span class="text-black text-base font-bold mr-2">提出ファイル</span>
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
        <template v-if="files.length > 0">
          <div class="flex flex-col justify-center overflow-x-scroll">
            <div
              v-for="(file, index) in files"
              :key="file.name"
              class="flex flex-row items-center text-black text-base"
            >
              <span class="mr=2">{{ file.name }}</span
              ><CloseIcon @click="removeFile(index)"></CloseIcon>
            </div>
          </div>
        </template>
        <template v-else>
          <span class="text-black text-base">ファイルが選択されていません</span>
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
        <span class="block sm:inline">課題の提出に失敗しました</span>
        <span class="absolute top-0 bottom-0 right-0 px-4 py-3">
          <CloseIcon :classes="['text-red-500']" @click="hideAlert"></CloseIcon>
        </span>
      </div>
      <div class="px-4 py-3 flex justify-center">
        <button
          type="button"
          class="
            mr-2
            w-auto
            rounded-md
            border border-primary-500
            shadow-sm
            px-4
            py-2
            bg-white
            text-sm
            font-medium
            text-primary-500
          "
          @click="close"
        >
          閉じる
        </button>
        <button
          type="button"
          class="
            w-auto
            rounded-md
            border border-transparent
            shadow-sm
            px-4
            py-2
            bg-primary-500
            text-sm
            font-medium
            text-white
          "
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
import CloseIcon from '~/components/common/CloseIcon.vue'

type SubmitFormData = {
  files: File[]
  failed: boolean
}

export default Vue.extend({
  name: 'SubmitModal',
  components: {
    Card,
    Modal,
    CloseIcon,
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
      files: [],
      failed: false,
    }
  },
  computed: {
    title(): string {
      return `${this.classTitle} 課題提出`
    },
    description(): string {
      return `これは科目 ${this.courseName} の授業 ${this.classTitle} の課題提出フォームです。 `
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
      const files: FileList = event.target.files
      for (let i = 0; i < files.length; i++) {
        this.files.push(files.item(i) as File)
      }
    },
    removeFile(index: number) {
      this.files.splice(index, 1)
    },
    upload() {
      const formData = new FormData()
      for (const file of this.files) {
        formData.append('file', file)
      }
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
      this.files = []
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
