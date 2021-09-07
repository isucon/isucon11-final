<template>
  <div v-if="isShowSearchResult">
    <table class="table-auto border w-full mt-1">
      <thead>
        <tr class="text-center">
          <th></th>
          <th>科目コード</th>
          <th>科目名</th>
          <th>科目種別</th>
          <th>時間</th>
          <th>単位数</th>
          <th>ステータス</th>
          <th>担当</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <template v-for="(c, i) in courses">
          <tr
            :key="`course-tr-${i}`"
            class="text-center bg-gray-200 odd:bg-white cursor-pointer"
            @click="onChangeRadioButton(i)"
          >
            <td>
              <input
                name="course-select"
                type="radio"
                class="
                  form-input
                  text-primary-500
                  focus:outline-none focus:ring-primary-200
                  rounded
                "
                :checked="selectedCourseIdx === i"
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
              <div class="relative">
                <fa-icon
                  icon="ellipsis-v"
                  class="
                    min-w-min
                    px-2
                    cursor-pointer
                    rounded
                    hover:bg-primary-300
                  "
                  @click.stop="onClickCourseDropdown(i)"
                />
                <div
                  class="
                    absolute
                    right-0
                    mt-2
                    py-1
                    rounded
                    z-20
                    w-40
                    bg-white
                    shadow-2xl
                  "
                  :class="openDropdownIdx === i ? 'show' : 'hidden'"
                >
                  <a
                    :href="`/courses/${c.id}`"
                    target="_blank"
                    class="
                      block
                      px-4
                      py-2
                      text-black text-sm
                      hover:bg-primary-300 hover:text-white
                    "
                    @click.stop=""
                    >詳細を見る
                  </a>
                  <a
                    href="#"
                    class="
                      block
                      px-4
                      py-2
                      text-black text-sm
                      hover:bg-primary-300 hover:text-white
                    "
                    @click.prevent.stop="onClickSetStatus(i)"
                    >ステータス変更
                  </a>
                  <a
                    href="#"
                    class="
                      block
                      px-4
                      py-2
                      text-black text-sm
                      hover:bg-primary-300 hover:text-white
                    "
                    @click.prevent.stop="onClickAddClass(i)"
                    >講義追加
                  </a>
                </div>
              </div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
    <div class="flex justify-center mt-2">
      <Pagination
        :prev-disabled="!Boolean(link.prev)"
        :next-disabled="!Boolean(link.next)"
        @goPrev="onClickPagination(link.prev.query)"
        @goNext="onClickPagination(link.next.query)"
      />
    </div>
  </div>
</template>

<script lang="ts">
import Vue, { PropType } from 'vue'
import {
  CourseStatus,
  CourseType,
  DayOfWeek,
  SyllabusCourse,
} from '~/types/courses'
import { formatPeriod, formatStatus, formatType } from '~/helpers/course_helper'
import Pagination from '~/components/common/Pagination.vue'
import { Link } from '~/helpers/link_helper'

type DataType = {
  openDropdownIdx: number | null
}

export default Vue.extend({
  components: { Pagination },
  props: {
    courses: {
      type: Array as PropType<SyllabusCourse[]>,
      default: () => [],
    },
    selectedCourseIdx: {
      type: Number as PropType<number | null>,
      default: null,
    },
    link: {
      type: Object as PropType<Partial<Link>>,
      default: () => {
        return { prev: undefined, next: undefined }
      },
    },
  },
  data(): DataType {
    return {
      openDropdownIdx: null,
    }
  },
  computed: {
    isShowSearchResult(): boolean {
      return this.courses.length > 0
    },
  },
  beforeMount() {
    document.addEventListener('click', this.outsideClick)
  },
  beforeDestroy() {
    document.removeEventListener('click', this.outsideClick)
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
    onChangeRadioButton(courseIdx: number): void {
      this.$emit('change', courseIdx)
    },
    onClickPagination(query: URLSearchParams): void {
      this.$emit('paginate', query)
    },
    onClickCourseDropdown(courseIdx: number): void {
      if (this.openDropdownIdx !== null && this.openDropdownIdx === courseIdx) {
        this.openDropdownIdx = null
        return
      }
      this.openDropdownIdx = courseIdx
    },
    onClickSetStatus(courseIdx: number): void {
      this.$emit('setStatus', courseIdx)
      this.openDropdownIdx = null
    },
    onClickAddClass(courseIdx: number): void {
      this.$emit('addClass', courseIdx)
      this.openDropdownIdx = null
    },
    outsideClick() {
      this.openDropdownIdx = null
    },
  },
})
</script>
