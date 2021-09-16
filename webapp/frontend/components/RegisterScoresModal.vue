<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card class="p-8">
      <p class="text-2xl text-gray-800 font-bold justify-center mb-4">
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
          <div class="grid grid-cols-score gap-2">
            <label class="text-gray-500 font-bold"> 学内コード </label>
            <label class="text-gray-500 font-bold"> 採点結果 </label>
          </div>
          <template v-for="(param, index) in params">
            <div
              :key="`param-${index}`"
              class="grid grid-cols-score gap-2 items-center w-full mt-2"
            >
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
                  text-gray-800
                  leading-tight
                  focus:outline-none focus:border-primary-500 focus:ring-0
                "
                type="text"
                placeholder="学生の学内コードを入力"
                :value="param.userCode"
                @input="$set(param, 'userCode', $event.target.value)"
              />
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
                  text-gray-800
                  leading-tight
                  focus:outline-none focus:border-primary-500 focus:ring-0
                "
                type="number"
                placeholder="採点結果を入力"
                :value="String(param.score)"
                @input="$set(param, 'score', Number($event.target.value))"
              />
              <div
                class="
                  cursor-pointer
                  w-6
                  flex
                  items-center
                  justify-center
                  text-red-400
                "
                @click="removeStudent(index)"
              >
                <fa-icon icon="times" size="lg" />
              </div>
            </div>
          </template>
          <div class="mt-2 float-right">
            <div
              class="
                cursor-pointer
                w-6
                pl-0.5
                flex
                items-center
                justify-center
                text-green-500
              "
              @click="addStudent"
            >
              <fa-icon icon="plus" size="lg" />
            </div>
          </div>
        </div>
      </div>
      <template v-if="failed">
        <InlineNotification type="error">
          <template #title>APIエラーがあります</template>
          <template #message>採点結果の登録に失敗しました。</template>
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
import { notify } from '~/helpers/notification_helper'
import Card from '~/components/common/Card.vue'
import Modal from '~/components/common/Modal.vue'
import Button from '~/components/common/Button.vue'
import LabeledText from '~/components/common/LabeledText.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'
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
    InlineNotification,
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
