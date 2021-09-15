<template>
  <div>
    <table class="table-auto border w-full mt-1">
      <thead>
        <tr class="text-center">
          <th>科目コード</th>
          <th>科目名</th>
          <th>科目種別</th>
          <th>時間</th>
          <th>単位数</th>
          <th>ステータス</th>
          <th></th>
        </tr>
      </thead>

      <tbody>
        <template v-if="isShowSearchResult">
          <template v-for="(c, i) in courses">
            <tr
              :key="`course-tr-${i}`"
              class="text-center bg-gray-200 odd:bg-white"
            >
              <td>{{ c.code }}</td>
              <td>{{ c.name }}</td>
              <td>{{ formatType(c.type) }}</td>
              <td>{{ formatPeriod(c.dayOfWeek, c.period) }}</td>
              <td>{{ c.credit }}</td>
              <td>{{ formatStatus(c.status) }}</td>
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
                      :href="`/syllabus/${c.id}`"
                      target="_blank"
                      class="
                        block
                        px-4
                        py-2
                        text-black text-sm
                        hover:bg-primary-300 hover:text-white
                      "
                      @click.stop="closeDropdown()"
                      >科目の詳細を確認
                    </a>
                    <a
                      :href="`/teacher/courses/${c.id}`"
                      target="_blank"
                      class="
                        block
                        px-4
                        py-2
                        text-black text-sm
                        hover:bg-primary-300 hover:text-white
                      "
                      @click.stop="closeDropdown()"
                      >講義一覧を確認
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
                      >科目の状態を変更
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
                      @click.prevent.stop="onClickAddAnnouncement(i)"
                      >お知らせを送信
                    </a>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </template>
        <template v-else>
          <tr>
            <td colspan="9">
              <div class="text-center">登録済みの科目が存在しません</div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
    <div class="flex justify-center mt-2">
      <Pagination
        :prev-disabled="!Boolean(link.prev)"
        :next-disabled="!Boolean(link.next)"
        @goPrev="onClickPagination(link.prev.path, link.prev.query)"
        @goNext="onClickPagination(link.next.path, link.next.query)"
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
    document.addEventListener('click', this.closeDropdown)
  },
  beforeDestroy() {
    document.removeEventListener('click', this.closeDropdown)
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
    onClickPagination(path: string | undefined, query: URLSearchParams): void {
      this.$emit('paginate', path, query)
    },
    onClickCourseDropdown(courseIdx: number): void {
      if (this.openDropdownIdx !== null && this.openDropdownIdx === courseIdx) {
        this.closeDropdown()
        return
      }
      this.openDropdownIdx = courseIdx
    },
    onClickSetStatus(courseIdx: number): void {
      this.$emit('setStatus', courseIdx)
      this.closeDropdown()
    },
    onClickAddAnnouncement(courseIdx: number): void {
      this.$emit('addAnnouncement', courseIdx)
      this.closeDropdown()
    },
    closeDropdown() {
      this.openDropdownIdx = null
    },
  },
})
</script>
