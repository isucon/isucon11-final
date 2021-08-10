<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12">
      <div class="flex-1 flex-col">
        <h1 class="text-2xl">履修登録</h1>
        <div class="mt-2 mb-6">
          <Button class="mr-2" @click="onClickSearchCourse()">科目検索</Button>
          <Button color="primary" @click="onClickConfirm">内容の確定</Button>
        </div>
        <Calendar :period-count="periodCount">
          <template v-for="(c, i) in courses">
            <CalendarCell :key="`course-${i}`">
              <template v-if="c.id !== undefined">
                <NuxtLink
                  :to="`/courses/${c.id}`"
                  class="flex-grow h-30 py-1 w-full cursor-pointer"
                >
                  <div class="flex flex-col">
                    <span class="text-primary-500">
                      <span>{{ c.code }}</span>
                      <span class="font-bold">{{ c.name }}</span>
                    </span>
                    <span class="text-sm">{{ c.teacher }}</span>
                  </div>
                </NuxtLink>
              </template>
              <template v-else>
                <button
                  class="flex-grow h-30 w-full cursor-pointer"
                  @click="onClickSearchCourse(c)"
                >
                  <fa-icon icon="pen" size="lg" class="text-primary-500" />
                </button>
              </template>
            </CalendarCell>
          </template>
        </Calendar>
      </div>
    </div>
    <SearchModal
      v-model="willRegisterCourses"
      :is-shown="isShownModal"
      :period-count="periodCount"
      :selected="selected"
      @close="onCloseSearchModal"
    />
  </div>
</template>
<script lang="ts">
import Vue from 'vue'
import Button from '../components/common/Button.vue'
import Calendar from '../components/Calendar.vue'
import CalendarCell from '../components/CalendarCell.vue'
import SearchModal from '../components/SearchModal.vue'
import { notify } from '~/helpers/notification_helper'
import { Course, DayOfWeek } from '~/types/courses'
import { DayOfWeekMap, WeekdayCount, PeriodCount } from '~/constants/calendar'

type PartialCourse = Partial<Course>

type DataType = {
  isShownModal: boolean
  selected: { dayOfWeek: DayOfWeek | undefined; period: number | undefined }
  willRegisterCourses: Course[]
  periodCount: number
}

export default Vue.extend({
  components: {
    SearchModal,
    CalendarCell,
    Calendar,
    Button,
  },
  middleware: 'is_loggedin',
  data(): DataType {
    return {
      isShownModal: false,
      selected: { dayOfWeek: undefined, period: undefined },
      willRegisterCourses: [],
      periodCount: PeriodCount,
    }
  },
  computed: {
    courses(): PartialCourse[] {
      return new Array(WeekdayCount * PeriodCount)
        .fill(undefined)
        .map((_, i) => {
          const course = this.getCourse(i)
          if (!course) {
            const dayOfWeek = (Object.keys(DayOfWeekMap) as DayOfWeek[]).find(
              (k) => DayOfWeekMap[k] === i % WeekdayCount
            )
            const period = Math.floor(i / WeekdayCount) + 1

            return { id: undefined, dayOfWeek, period }
          }

          return course
        })
    },
  },
  methods: {
    getCourse(idx: number): Course | undefined {
      return this.willRegisterCourses.find((c) => {
        const dayOfWeek = DayOfWeekMap[c.dayOfWeek as DayOfWeek]
        return idx === dayOfWeek + (c.period - 1) * WeekdayCount
      })
    },
    onClickSearchCourse(c: PartialCourse | undefined): void {
      if (c) {
        this.selected = Object.assign({}, this.selected, {
          dayOfWeek: c?.dayOfWeek,
          period: c?.period,
        })
      }
      this.isShownModal = true
    },
    onCloseSearchModal(): void {
      this.isShownModal = false
      this.selected = Object.assign({}, this.selected, {
        dayOfWeek: undefined,
        period: undefined,
      })
    },
    async onClickConfirm(): Promise<void> {
      try {
        const ids = this.willRegisterCourses.map((c) => ({ id: c.id }))
        const res = await this.$axios.put(`/api/users/me/courses`, ids)
        if (res.status === 200) {
          await this.$router.push('/mypage')
        }
      } catch (e) {
        notify('履修登録に失敗しました')
      }
    },
  },
})
</script>
