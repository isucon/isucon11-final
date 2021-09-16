<template>
  <Modal :is-shown="isShown" @close="onClose">
    <div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4 rounded">
      <div class="flex flex-col flex-nowrap">
        <h3
          id="modal-title"
          class="text-2xl leading-6 font-medium text-gray-500 mb-4"
        >
          科目検索
        </h3>
        <form class="flex-1 flex-col" @submit.prevent="onSubmitSearch()">
          <TextField
            id="params-keywords"
            v-model="params.keywords"
            label="キーワード"
            label-direction="vertical"
            type="text"
            placeholder="キーワードを入力してください"
          />
          <div class="flex mt-4">
            <div class="flex gap-1">
              <TextField
                id="params-teacher"
                v-model="params.teacher"
                class="flex-1"
                label="担当教員"
                label-direction="vertical"
                type="text"
                placeholder="教員名を入力"
              />
              <TextField
                id="params-credit"
                v-model="params.credit"
                class="flex-1"
                label="単位数"
                label-direction="vertical"
                type="number"
                min="1"
                placeholder="単位数を入力"
              />
              <Select
                id="params-type"
                v-model="params.type"
                class="flex-1"
                label="科目種別"
                :options="[
                  { text: '一般教養', value: 'liberal-arts' },
                  { text: '専門', value: 'major-subjects' },
                ]"
              />
            </div>
          </div>
          <div class="flex mt-4 space-x-2 flex-wrap">
            <div class="flex flex-auto gap-1">
              <Select
                id="params-day-of-week"
                class="flex-1"
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
                class="flex-1"
                label="時限"
                :options="periods"
                :selected="params.period || selected.period"
                @change="params.period = $event"
              />
              <Select
                id="params-period"
                class="flex-1"
                label="科目の状態"
                :options="[
                  { text: '履修登録期間', value: 'registration' },
                  { text: '講義期間', value: 'in-progress' },
                  { text: '終了済み', value: 'closed' },
                ]"
                :selected="params.status || selected.status"
                @change="params.status = $event"
              />
            </div>
          </div>
          <div class="flex justify-center gap-2">
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
                <th>科目の状態</th>
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
                  <td>{{ c.teacher }}</td>
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
                  @goPrev="onClickPagination(link.prev.path, link.prev.query)"
                  @goNext="onClickPagination(link.next.path, link.next.query)"
                />
              </div>
              <span class="opacity-0 w-28"></span>
            </div>
          </div>
        </template>
        <template v-else-if="courses && !hasError">
          <hr class="my-6" />
          <div class="">検索条件にマッチする科目がありませんでした。</div>
        </template>
        <template v-else-if="hasError">
          <hr class="my-6" />
          <div class="">検索結果を取得できませんでした。</div>
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
  courses: SyllabusCourse[] | null
  checkedCourses: SyllabusCourse[]
  params: SearchCourseRequest
  link: Partial<Link>
  hasError: boolean
}

const initParams = {
  keywords: '',
  type: '',
  credit: undefined,
  teacher: '',
  period: undefined,
  dayOfWeek: '',
  status: '',
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
      courses: null,
      checkedCourses: this.value,
      params: Object.assign({}, initParams),
      link: { prev: undefined, next: undefined },
      hasError: false,
    }
  },
  computed: {
    isShowSearchResult(): boolean {
      if (this.courses) {
        return this.courses.length > 0
      }
      return false
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
    async onSubmitSearch(
      path = '/api/courses',
      query?: Record<string, any>
    ): Promise<void> {
      this.hasError = false
      const params = this.filterParams(this.params)
      try {
        const res = await this.$axios.get<SyllabusCourse[]>(path, {
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
        this.hasError = true
        this.courses = null
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
    onClickPagination(path: string, query: URLSearchParams): void {
      this.onSubmitSearch(path, urlSearchParamsToObject(query))
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
      this.hasError = false
      this.courses = null
      this.params = Object.assign({}, this.params, initParams)
    },
  },
})
</script>
