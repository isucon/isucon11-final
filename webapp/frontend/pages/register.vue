<template>
  <div>
    <div
      class="py-10 px-8 bg-white shadow-lg mt-8 mb-8 rounded w-192 max-w-full"
    >
      <div class="flex-1 flex-col">
        <h1 class="text-2xl font-bold text-gray-800">履修登録</h1>
        <div class="flex mt-2 mb-6 gap-2">
          <Button @click="onClickSearchCourse">科目検索</Button>
          <Button color="primary" @click="onClickConfirm">内容の確定</Button>
        </div>

        <template v-if="hasError">
          <InlineNotification type="error" class="my-4">
            <template #title>APIエラーがあります</template>
            <template #message>{{
              errorMessage || '履修済み科目の取得に失敗しました。'
            }}</template>
          </InlineNotification>
        </template>

        <Calendar :period-count="periodCount">
          <template v-for="(periodCourses, p) in courses">
            <template v-for="(weekdayCourses, w) in periodCourses">
              <CalendarCell :key="`course-${p}-${w}`">
                <template v-for="(course, i) in weekdayCourses">
                  <template v-if="course.id">
                    <a
                      :key="`link-${p}-${w}-${i}`"
                      :href="`/syllabus/${course.id}`"
                      target="_blank"
                      class="
                        flex-grow
                        h-30
                        px-2
                        py-2
                        w-full
                        cursor-pointer
                        transition
                        duration-500
                        ease
                        hover:bg-primary-100
                      "
                      :class="
                        course.displayType === 'will_register'
                          ? 'border-2 border-primary-700 border-opacity-40'
                          : ''
                      "
                    >
                      <div class="relative flex flex-col w-full h-full">
                        <span class="text-primary-500">
                          <template
                            v-if="
                              course.code &&
                              course.displayType === 'will_register'
                            "
                          >
                            <span>{{ course.code }}</span>
                          </template>
                          <span class="font-bold">{{ course.name }}</span>
                        </span>
                        <span class="text-sm">{{ course.teacher }}</span>
                        <template
                          v-if="
                            course.code &&
                            course.displayType === 'will_register'
                          "
                        >
                          <button
                            title="仮登録解除"
                            class="absolute right-0 -bottom-0.5"
                            @click="onClickCancelCourse($event, course.id)"
                          >
                            <fa-icon
                              icon="times"
                              size="lg"
                              class="text-primary-500"
                            />
                          </button>
                        </template>
                      </div>
                    </a>
                  </template>
                  <template v-else>
                    <button
                      :key="`button-${p}-${w}-${i}`"
                      class="
                        h-full
                        w-full
                        cursor-pointer
                        transition
                        duration-500
                        ease
                        hover:bg-primary-100
                      "
                      @click="onClickSearchCourse(course)"
                    >
                      <div
                        class="
                          relative
                          h-full
                          w-full
                          opacity-0
                          hover:opacity-70
                          transition
                          duration-500
                          ease
                        "
                      >
                        <fa-icon
                          icon="pen"
                          size="lg"
                          class="absolute bottom-4 right-4 text-primary-500"
                        />
                      </div>
                    </button>
                  </template>
                </template>
              </CalendarCell>
            </template>
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
import { Context } from '@nuxt/types'
import axios from 'axios'
import Button from '../components/common/Button.vue'
import Calendar from '../components/Calendar.vue'
import CalendarCell from '../components/CalendarCell.vue'
import SearchModal from '../components/SearchModal.vue'
import { notify } from '~/helpers/notification_helper'
import { Course, DayOfWeek } from '~/types/courses'
import { DayOfWeekMap, PeriodCount, WeekdayCount } from '~/constants/calendar'
import InlineNotification from '~/components/common/InlineNotification.vue'
import {
  formatRegistrationError,
  isRegistrationError,
} from '~/helpers/course_helper'

type DisplayType = 'registered' | 'will_register' | 'none'
type DisplayCourse = Partial<Course> & { displayType: DisplayType }
type CalendarCourses = DisplayCourse[][][]

type DataType = {
  isShownModal: boolean
  selected: { dayOfWeek: DayOfWeek | undefined; period: number | undefined }
  willRegisterCourses: Course[]
  registeredCourses: Course[]
  periodCount: number
  hasError: boolean
  errorMessage: string | undefined
}

export default Vue.extend({
  components: {
    InlineNotification,
    SearchModal,
    CalendarCell,
    Calendar,
    Button,
  },
  middleware: 'is_student',
  async asyncData(ctx: Context) {
    try {
      const res = await ctx.$axios.get<Course[]>(`/api/users/me/courses`)
      return { registeredCourses: res.data ?? [], hasError: false }
    } catch (e) {
      console.error(e)
      notify('履修登録済み科目の取得に失敗しました')
    }

    return { registeredCourses: [], hasError: true }
  },
  data(): DataType {
    return {
      isShownModal: false,
      selected: { dayOfWeek: undefined, period: undefined },
      willRegisterCourses: [],
      registeredCourses: [],
      periodCount: PeriodCount,
      hasError: false,
      errorMessage: undefined,
    }
  },
  head: {
    title: 'ISUCHOLAR - 履修登録',
  },
  computed: {
    courses(): CalendarCourses {
      const periodCourses: CalendarCourses = []
      for (let period = 1; period <= PeriodCount; period++) {
        const weekdayCourses = []
        for (let weekday = 1; weekday <= WeekdayCount; weekday++) {
          const dayOfWeek = (Object.keys(DayOfWeekMap) as DayOfWeek[]).find(
            (k) => DayOfWeekMap[k] === weekday
          )

          const courses: DisplayCourse[] = []
          const willRegisterCourse = this.getWillRegisterCourse(period, weekday)
          if (willRegisterCourse) {
            willRegisterCourse.forEach((c) => {
              courses.push({
                ...c,
                displayType: 'will_register',
              })
            })
          }

          const registeredCourse = this.getRegisteredCourse(period, weekday)
          if (registeredCourse) {
            courses.push({ ...registeredCourse, displayType: 'registered' })
          }

          if (courses.length === 0) {
            courses.push({
              id: undefined,
              period,
              dayOfWeek,
              displayType: 'none',
            })
          }

          weekdayCourses.push(courses)
        }
        periodCourses.push(weekdayCourses)
      }

      return periodCourses
    },
  },
  methods: {
    getWillRegisterCourse(
      period: number,
      weekday: number
    ): Course[] | undefined {
      const course = this.willRegisterCourses.filter((c) => {
        const dayOfWeek = DayOfWeekMap[c.dayOfWeek as DayOfWeek]
        return period === c.period && weekday === dayOfWeek
      })

      return course.length > 0 ? course : undefined
    },
    getRegisteredCourse(period: number, weekday: number): Course | undefined {
      return this.registeredCourses.find((c) => {
        const dayOfWeek = DayOfWeekMap[c.dayOfWeek as DayOfWeek]
        return period === c.period && weekday === dayOfWeek
      })
    },
    onClickSearchCourse(c: Partial<Course> | undefined): void {
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
    formatRegistrationError(err: any): string {
      if (axios.isAxiosError(err) && isRegistrationError(err?.response?.data)) {
        return `(失敗理由: ${formatRegistrationError(err?.response?.data)})`
      }

      return ''
    },
    async onClickConfirm(): Promise<void> {
      try {
        const ids = this.willRegisterCourses.flat().map((c) => ({ id: c.id }))
        const res = await this.$axios.put(`/api/users/me/courses`, ids)
        if (res.status === 200) {
          await this.$router.push('/mypage')
        }
      } catch (e) {
        this.hasError = true
        this.errorMessage = `履修登録に失敗しました。${this.formatRegistrationError(
          e
        )}`
        notify('履修登録に失敗しました')
      }
    },
    onClickCancelCourse(e: Event, id: string): void {
      e.preventDefault()
      this.willRegisterCourses = this.willRegisterCourses.filter(
        (c) => c.id !== id
      )
    },
  },
})
</script>
