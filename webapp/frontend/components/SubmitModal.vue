<template>
  <modal :is-shown="isShown" @close="emit('close')">
    <card>
      <p class="text-2xl text-black font-bold flex justify-center mb-4">
        {{ title }}
      </p>
      <p class="text-black text-base mb-4">{{ description }}</p>
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
              class="text-black text-base"
            >
              {{ file.name }}&nbsp;&nbsp;<span
                class="cursor-pointer"
                @click="removeFile(index)"
              >
                X
              </span>
            </div>
          </div>
        </template>
        <template v-else>
          <span class="text-black text-base">ファイルが選択されていません</span>
        </template>
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
          Cancel
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
          Submit
        </button>
      </div>
    </card>
  </modal>
</template>

<script lang="ts">
import Vue from 'vue'

interface SubmitFormData {
  files: File[]
}

export default Vue.extend({
  name: 'SubmitModal',
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
    assignmentName: {
      type: String,
      required: true,
    },
    assignmentId: {
      type: String,
      required: true,
    },
  },
  data(): SubmitFormData {
    return {
      files: [],
    }
  },
  computed: {
    title(): string {
      return `${this.assignmentName} 課題提出`
    },
    description(): string {
      return `これは科目名 ${this.courseName} の ${this.classTitle} の課題 ${this.assignmentName} の提出用です。 提出先の課題や提出ファイルが正しいか確認してください。`
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
          `/${this.$route.params.courseId}/assignments/${this.assignmentId}`,
          formData
        )
        .then((response) => {
          console.log(response.data)
        })
        .catch((error) => {
          console.log(error)
        })
      this.close()
    },
    close() {
      this.files = []
      this.$emit('close')
    },
  },
})
</script>
