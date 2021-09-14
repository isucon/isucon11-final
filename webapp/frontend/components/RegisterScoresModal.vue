<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-black font-bold justify-center mb-4">
        採点結果の登録
      </p>
      <div class="flex flex-col space-y-4 mb-4">
        <div class="flex-1">
          <LabeledText label="科目名" :value="courseName" />
        </div>
        <div class="flex-1">
          <LabeledText label="講義タイトル" :value="classTitle" />
        </div>
        <div>
          <div class="flex flex-row space-x-2">
            <div class="flex-1 w-full">
              <label class="text-gray-500 font-bold text-right">
                学籍番号
              </label>
            </div>
            <div class="flex-1 w-full">
              <label class="text-gray-500 font-bold text-right">
                採点結果
              </label>
            </div>
          </div>
          <template v-for="(param, index) in params">
            <div
              :key="`param-${index}`"
              class="flex flex-row space-x-2 items-center"
            >
              <div class="flex-1 mb-1">
                <input
                  :id="`params-usercode-${index}`"
                  class="
                    w-full
                    bg-white
                    appearance-none
                    border-2 border-gray-200
                    rounded
                    py-2
                    px-4
                    text-gray-700
                    leading-tight
                    focus:outline-none focus:bg-white focus:border-purple-500
                  "
                  type="text"
                  placeholder="学生の学籍番号を入力"
                  :value="param.userCode"
                  @input="$set(param, 'userCode', $event.target.value)"
                />
              </div>
              <div class="flex-1 mb-1">
                <input
                  :id="`params-score-${index}`"
                  class="
                    w-full
                    bg-white
                    appearance-none
                    border-2 border-gray-200
                    rounded
                    py-2
                    px-4
                    text-gray-700
                    leading-tight
                    focus:outline-none focus:bg-white focus:border-purple-500
                  "
                  type="number"
                  placeholder="採点結果を入力"
                  :value="String(param.score)"
                  @input="$set(param, 'score', Number($event.target.value))"
                />
              </div>
              <div class="flex-2 mb-1 cursor-pointer">
                <fa-icon icon="times" size="lg" @click="removeStudent(index)" />
              </div>
            </div>
          </template>
          <div class="grid">
            <div class="mt-1 place-self-center cursor-pointer">
              <fa-icon icon="plus" size="lg" @click="addStudent" />
            </div>
          </div>
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
        <span class="block sm:inline">採点結果の登録に失敗しました</span>
        <span class="absolute top-0 bottom-0 right-0 px-4 py-3">
          <CloseIcon :classes="['text-red-500']" @click="hideAlert"></CloseIcon>
        </span>
      </div>
      <div class="px-4 py-3 flex justify-center">
        <Button @click="close"> 閉じる </Button>
        <Button color="primary" @click="submit"> 登録 </Button>
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
import LabeledText from '~/components/common/LabeledText.vue'
import { RegisterScoreRequest } from '~/types/courses'

type SubmitFormData = {
  failed: boolean
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
    LabeledText,
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
    classId: {
      type: String,
      required: true,
    },
    classTitle: {
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
      params: initParams.map((o) => Object.assign({}, o)),
    }
  },
  methods: {
    async submit() {
      try {
        await this.$axios.put(
          `/api/courses/${this.courseId}/classes/${this.classId}/assignments/scores`,
          this.params
        )
        notify('採点結果の登録が完了しました')
        this.close()
      } catch (e) {
        notify('採点結果の登録に失敗しました')
        this.showAlert()
      }
    },
    close() {
      this.params = initParams.map((o) => Object.assign({}, o))
      this.hideAlert()
      this.$emit('close')
    },
    showAlert() {
      this.failed = true
    },
    hideAlert() {
      this.failed = false
    },
    addStudent() {
      this.params.push(Object.assign({}, initParams[0]))
    },
    removeStudent(index: number) {
      this.params.splice(index, 1)
    },
  },
})
</script>
