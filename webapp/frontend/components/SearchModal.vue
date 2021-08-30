<template>
  <Modal :is-shown="isShown" @close="onClose">
    <div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
      <div class="flex flex-col flex-nowrap">
        <h3
          id="modal-title"
          class="text-lg leading-6 font-medium text-gray-900"
        >
          科目検索
        </h3>
        <form class="flex-1 flex-col" @submit.prevent="onSubmitSearch()">
          <div class="flex items-center">
            <TextField
              id="params-keywords"
              v-model="params.keywords"
              label="キーワード"
              type="text"
              placeholder="キーワードを入力してください"
            />
          </div>
          <div class="flex mt-4 space-x-2">
            <label
              class="whitespace-nowrap block text-gray-500 font-bold pr-4 w-1/6"
              >科目</label
            >
            <div class="">
              <TextField
                id="params-teacher"
                v-model="params.teacher"
                label="担当教員"
                label-direction="vertical"
                type="text"
                placeholder="教員名を入力"
              />
            </div>
            <div class="flex items-center">
              <TextField
                id="params-credit"
                v-model="params.credit"
                label="単位数"
                label-direction="vertical"
                type="number"
                min="1"
                placeholder="単位数を入力"
              />
            </div>
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
          <div class="flex mt-4 space-x-2">
            <label
              class="whitespace-nowrap block text-gray-500 font-bold pr-4 w-1/6"
              >開講</label
            >
            <Select
              id="params-day-of-week"
              label="曜日"
              :options="[
                { text: '月曜', value: 'monday' },
                { text: '火曜', value: 'tuesday' },
                { text: '水曜', value: 'wednesday' },
                { text: '木曜', value: 'thursday' },
                { text: '金曜', value: 'friday' },
              ]"
              :selected="params.dayOfWeek || selected.dayOfWeek"
              @change="params.dayOfWeek = $event"
            />
            <Select
              id="params-period"
              label="時限"
              :options="periods"
              :selected="params.period || selected.period"
              @change="params.period = $event"
            />
          </div>
          <div class="flex justify-center">
            <Button type="button" class="mt-6 flex-grow-0" @click="onClickReset"
              >リセット
            </Button>
            <Button type="submit" class="mt-6 flex-grow-0" color="primary"
              >検索
            </Button>
          </div>
        </form>

        <template v-if="isShowSearchResult">
          <hr class="my-6" />
          <div>
            <h3 class="text-xl font-bold">検索結果</h3>
            <table class="table-auto border w-full mt-1">
              <tr class="text-center">
                <th>選択</th>
                <th>科目コード</th>
                <th>科目名</th>
                <th>科目種別</th>
                <th>時間</th>
                <th>単位数</th>
                <th>ステータス</th>
                <th>担当</th>
                <th></th>
              </tr>
              <template v-for="(c, i) in courses">
                <tr
                  :key="`tr-${i}`"
                  class="text-center bg-gray-200 odd:bg-white"
                >
                  <td>
                    <input
                      type="checkbox"
                      class="
                        form-input
                        text-primary-500
                        focus:outline-none focus:ring-primary-200
                        rounded
                      "
                      :checked="isChecked(c.id)"
                      @change="onChangeCheckbox(c)"
                    />
                  </td>
                  <td>{{ c.code }}</td>
                  <td>{{ c.name }}</td>
                  <td>{{ formatType(c.type) }}</td>
                  <td>{{ formatPeriod(c.dayOfWeek, c.period) }}</td>
                  <td>{{ c.credit }}</td>
                  <td>{{ formatStatus(c.status) }}</td>
                  <td>椅子 昆</td>
                  <td>
                    <a
                      :href="`/syllabus/${c.id}`"
                      target="_blank"
                      class="text-primary-500"
                      >詳細を見る
                    </a>
                  </td>
                </tr>
              </template>
            </table>
            <div class="flex justify-between mt-2">
              <Button
                :disabled="checkedCourses.length === 0"
                class="w-28"
                @click="onSubmitTemporaryRegistration"
                >仮登録</Button
              >
              <div class="">
                <Pagination
                  :prev-disabled="!Boolean(link.prev)"
                  :next-disabled="!Boolean(link.next)"
                  @goPrev="onClickPagination(link.prev.query)"
                  @goNext="onClickPagination(link.next.query)"
                />
              </div>
              <span class="opacity-0 w-28"></span>
            </div>
          </div>
        </template>
      </div>
    </div>
  </Modal>
</template>
<script lang="ts">
import Vue, { PropType } from 'vue'
import Modal from './common/Modal.vue'
import TextField from './common/TextField.vue'
import Select from './common/Select.vue'
import Button from '~/components/common/Button.vue'
import {
  CourseStatus,
  CourseType,
  DayOfWeek,
  SearchCourseRequest,
  SyllabusCourse,
} from '~/types/courses'
import { notify } from '~/helpers/notification_helper'
import { formatPeriod, formatStatus, formatType } from '~/helpers/course_helper'
import Pagination from '~/components/common/Pagination.vue'
import { Link, parseLinkHeader } from '~/helpers/link_helper'
import { PeriodCount } from '~/constants/calendar'
import { urlSearchParamsToObject } from '~/helpers/urlsearchparams'

type Selected = {
  dayOfWeek: DayOfWeek | undefined
  period: number | undefined
}

type DataType = {
  courses: SyllabusCourse[]
  checkedCourses: SyllabusCourse[]
  params: SearchCourseRequest
  link: Partial<Link>
}

const initParams = {
  keywords: '',
  type: '',
  credit: undefined,
  teacher: '',
  period: undefined,
  dayOfWeek: '',
}

export default Vue.extend({
  components: { Pagination, Button, Select, TextField, Modal },
  props: {
    isShown: {
      type: Boolean,
      default: false,
      required: true,
    },
    selected: {
      type: Object as PropType<Selected>,
      default: () => ({ dayOfWeek: undefined, period: undefined }),
    },
    value: {
      type: Array as PropType<SyllabusCourse[]>,
      default: () => [],
      required: true,
    },
  },
  data(): DataType {
    return {
      courses: [],
      checkedCourses: this.value,
      params: Object.assign({}, initParams),
      link: { prev: undefined, next: undefined },
    }
  },
  computed: {
    isShowSearchResult(): boolean {
      return this.courses.length > 0
    },
    periods() {
      return new Array(PeriodCount).fill(undefined).map((_, i) => {
        return { text: `${i + 1}`, value: i + 1 }
      })
    },
  },
  methods: {
    formatType(type: CourseType): string {
      return formatType(type)
    },
    formatPeriod(dayOfWeek: DayOfWeek, period: number): string {
      return formatPeriod(dayOfWeek, period)
    },
    formatStatus(status: CourseStatus): string {
      return formatStatus(status)
    },
    isChecked(courseId: string): boolean {
      const course = this.checkedCourses.find((v) => v.id === courseId)
      return course !== undefined
    },
    onClickReset(): void {
      this.reset()
    },
    async onSubmitSearch(query?: Record<string, any>): Promise<void> {
      const params = this.filterParams(this.params)
      try {
        const res = await this.$axios.get<SyllabusCourse[]>('/api/syllabus', {
          params: { ...params, ...query },
        })
        if (res.status === 200) {
          if (res.data.length === 0) {
            notify('検索条件に一致する科目がありません')
          }
          this.courses = res.data
          this.link = Object.assign(
            {},
            this.link,
            parseLinkHeader(res.headers.link)
          )
        }
      } catch (e) {
        notify('検索結果を取得できませんでした')
      }
    },
    onChangeCheckbox(course: SyllabusCourse): void {
      const c = this.checkedCourses.find((v) => v.id === course.id)
      if (c) {
        this.checkedCourses = this.checkedCourses.filter(
          (v) => v.id !== course.id
        )
      } else {
        this.checkedCourses = [...this.checkedCourses, course]
      }
    },
    onSubmitTemporaryRegistration(): void {
      this.$emit('input', this.checkedCourses)
      this.onClose()
    },
    onClose() {
      this.reset()
      this.$emit('close')
    },
    onClickPagination(query: URLSearchParams): void {
      this.onSubmitSearch(urlSearchParamsToObject(query))
    },
    filterParams(params: SearchCourseRequest): SearchCourseRequest {
      return (Object.keys(params) as (keyof SearchCourseRequest)[])
        .filter((k) => params[k] !== undefined && params[k] !== '')
        .reduce(
          (acc, k) => ({ ...acc, [k]: params[k] }),
          {} as SearchCourseRequest
        )
    },
    reset(): void {
      this.courses = []
      this.params = Object.assign({}, this.params, initParams)
    },
  },
})
</script>
