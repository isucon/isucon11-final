<template>
  <div>
    <table class="table-auto w-full">
      <thead>
        <tr class="text-left">
          <th class="px-1 py-0.5">科目コード</th>
          <th class="px-1 py-0.5">科目名</th>
          <th class="px-1 py-0.5">科目種別</th>
          <th class="px-1 py-0.5">時間</th>
          <th class="px-1 py-0.5">単位数</th>
          <th class="px-1 py-0.5">科目の状態</th>
          <th class="px-1 py-0.5"></th>
        </tr>
      </thead>

      <tbody>
        <template v-if="isShowSearchResult">
          <template v-for="(c, i) in courses">
            <tr
              :key="`course-tr-${i}`"
              class="text-left bg-gray-200 odd:bg-white"
            >
              <td class="px-1 py-0.5">{{ c.code }}</td>
              <td class="px-1 py-0.5">{{ c.name }}</td>
              <td class="px-1 py-0.5">{{ formatType(c.type) }}</td>
              <td class="px-1 py-0.5">
                {{ formatPeriod(c.dayOfWeek, c.period) }}
              </td>
              <td class="px-1 py-0.5">{{ c.credit }}</td>
              <td class="px-1 py-0.5">{{ formatStatus(c.status) }}</td>
              <td class="px-1 py-0.5">
                <div class="relative">
                  <fa-icon
                    icon="ellipsis-v"
                    class="min-w-min px-2 cursor-pointer rounded float-right"
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
                        text-gray-800 text-sm
                        hover:bg-primary-300 hover:text-white
                      "
                      @click.stop="closeDropdown()"
                      >科目の詳細を確認
                    </a>
                    <NuxtLink
                      :to="`/teacher/courses/${c.id}`"
                      class="
                        block
                        px-4
                        py-2
                        text-gray-800 text-sm
                        hover:bg-primary-300 hover:text-white
                      "
                      @click.stop="closeDropdown()"
                      >講義一覧を確認
                    </NuxtLink>
                    <a
                      href="#"
                      class="
                        block
                        px-4
                        py-2
                        text-gray-800 text-sm
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
                        text-gray-800 text-sm
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
