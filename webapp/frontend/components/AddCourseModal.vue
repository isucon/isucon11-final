<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <Card>
      <p class="text-2xl text-gray-800 font-bold justify-center mb-4">
        科目登録
      </p>
      <div class="flex flex-col space-y-4 mb-4">
        <div class="flex flex-row space-x-2">
          <div class="flex-1">
            <TextField
              id="params-code"
              v-model="params.code"
              label="科目コード"
              label-direction="vertical"
              type="text"
              placeholder="科目コードを入力してください"
            />
          </div>
          <div class="flex-1">
            <Select
              id="params-type"
              v-model="params.type"
              label="科目種別"
              :options="[
                { text: '一般教養', value: 'liberal-arts' },
                { text: '専門', value: 'major-subjects' },
              ]"
            />
          </div>
        </div>
        <TextField
          id="params-name"
          v-model="params.name"
          label="科目名"
          label-direction="vertical"
          type="text"
          placeholder="科目名を入力してください"
        />
        <TextField
          id="params-description"
          v-model="params.description"
          label="科目詳細"
          label-direction="vertical"
          type="text"
          placeholder="科目の詳細を入力してください"
        />
        <div class="flex flex-row space-x-2">
          <div class="flex-1">
            <TextField
              id="params-credit"
              label="単位数"
              label-direction="vertical"
              type="number"
              placeholder="単位数を入力"
              :value="String(params.credit)"
              @input="$set(params, 'credit', Number($event))"
            />
          </div>
          <div class="flex-1">
            <Select
              id="params-day-of-week"
              v-model="params.dayOfWeek"
              label="曜日"
              :options="[
                { text: '月曜', value: 'monday' },
                { text: '火曜', value: 'tuesday' },
                { text: '水曜', value: 'wednesday' },
                { text: '木曜', value: 'thursday' },
                { text: '金曜', value: 'friday' },
              ]"
            />
          </div>
          <div class="flex-1">
            <Select
              id="params-period"
              label="時限"
              :options="periods"
              :selected="String(params.period)"
              @change="$set(params, 'period', Number($event))"
            />
          </div>
        </div>
        <TextField
          id="params-keywords"
          v-model="params.keywords"
          label="キーワード"
          label-direction="vertical"
          type="text"
          placeholder="キーワードを半角スペース区切りで入力してください"
        />
      </div>
      <template v-if="failed">
        <InlineNotification type="error">
          <template #title>APIエラーがあります</template>
          <template #message>科目の登録に失敗しました。</template>
        </InlineNotification>
      </template>
      <div class="flex justify-center gap-2 mt-4">
        <Button w-class="w-28" @click="close"> 閉じる </Button>
        <Button color="primary" w-class="w-28" @click="submit"> 登録 </Button>
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
import TextField from '~/components/common/TextField.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'
import { AddCourseRequest } from '~/types/courses'
import { PeriodCount } from '~/constants/calendar'

type SubmitFormData = {
  failed: boolean
  params: AddCourseRequest
}

const initParams: AddCourseRequest = {
  code: '',
  type: 'liberal-arts',
  name: '',
  description: '',
  credit: 0,
  period: 1,
  dayOfWeek: 'monday',
  keywords: '',
}

export default Vue.extend({
  components: {
    Card,
    Modal,
    InlineNotification,
    Button,
    TextField,
  },
  props: {
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
  computed: {
    periods() {
      return new Array(PeriodCount).fill(undefined).map((_, i) => {
        return { text: `${i + 1}`, value: i + 1 }
      })
    },
  },
  methods: {
    async submit() {
      try {
        await this.$axios.post(`/api/courses`, this.params)
        notify('科目の登録が完了しました')
        this.$emit('completed')
        this.close()
      } catch (e) {
        notify('科目の登録に失敗しました')
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
