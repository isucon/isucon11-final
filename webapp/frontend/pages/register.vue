<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg">
      <div class="flex-1 flex-col">
        <h1 class="text-2xl">履修登録</h1>
        <div class="mt-2 mb-6">
          <Button class="mr-2" @click="onClickSearchCourse()">科目検索</Button>
          <Button color="primary" @click="onClickConfirm">内容の確定</Button>
        </div>
        <Calendar :period-count="periodCount">
          <template v-for="(periodCourses, p) in courses">
            <template v-for="(weekdayCourses, w) in periodCourses">
              <CalendarCell :key="`course-${p}-${w}`">
                <template v-for="(course, i) in weekdayCourses">
                  <template v-if="course.id">
                    <NuxtLink
                      :key="`link-${p}-${w}-${i}`"
                      :to="`/courses/${course.id}`"
                      class="flex-grow h-30 py-1 w-full cursor-pointer"
                    >
                      <div class="flex flex-col">
                        <span class="text-primary-500">
                          <template
                            v-if="
                              course.code &&
                              course.displayType === 'will_register'
                            "
                          >
                            <span>{{ course.code }}</span>
                          </template>
                          <template
                            v-else-if="course.displayType === 'registered'"
                          >
                            <span>履修済</span>
                          </template>
                          <span class="font-bold">{{ course.name }}</span>
                        </span>
                        <span class="text-sm">{{ course.teacher }}</span>
                      </div>
                    </NuxtLink>
                  </template>
                  <template v-else>
                    <button
                      :key="`button-${p}-${w}-${i}`"
                      class="h-full w-full cursor-pointer"
                      @click="onClickSearchCourse(course)"
                    >
                      <fa-icon icon="pen" size="lg" class="text-primary-500" />
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
import Button from '../components/common/Button.vue'
import Calendar from '../components/Calendar.vue'
import CalendarCell from '../components/CalendarCell.vue'
import SearchModal from '../components/SearchModal.vue'
import { notify } from '~/helpers/notification_helper'
import { Course, DayOfWeek } from '~/types/courses'
import { DayOfWeekMap, PeriodCount, WeekdayCount } from '~/constants/calendar'

type DisplayType = 'registered' | 'will_register' | 'none'
type DisplayCourse = Partial<Course> & { displayType: DisplayType }
type CalendarCourses = DisplayCourse[][][]

type DataType = {
  isShownModal: boolean
  selected: { dayOfWeek: DayOfWeek | undefined; period: number | undefined }
  willRegisterCourses: Course[]
  registeredCourses: Course[]
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
  async asyncData(ctx: Context) {
    try {
      const registeredCourses = await ctx.$axios.$get<Course[]>(
        `/api/users/me/courses`
      )
      return { registeredCourses }
    } catch (e) {
      console.error(e)
      notify('履修登録済み科目の取得に失敗しました')
    }

    return { registeredCourses: [] }
  },
  data(): DataType {
    return {
      isShownModal: false,
      selected: { dayOfWeek: undefined, period: undefined },
      willRegisterCourses: [],
      registeredCourses: [],
      periodCount: PeriodCount,
    }
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
    async onClickConfirm(): Promise<void> {
      try {
        const ids = this.willRegisterCourses.flat().map((c) => ({ id: c.id }))
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
