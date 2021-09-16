<template>
  <div>
    <div
      class="py-10 px-8 bg-white shadow-lg mt-8 mb-8 rounded w-192 max-w-full"
    >
      <div class="flex-1 flex-col">
        <section>
          <h1 class="text-2xl">現在開講中の講義</h1>

          <template v-if="currentCourse">
            <div class="py-4">
              <Card>
                <div class="flex justify-between items-center">
                  <div>
                    <h2 class="text-xl mb-2 font-bold text-primary-500">
                      {{ currentCourse.name }}
                    </h2>
                    <ul class="list-none text-gray-500">
                      <li>教員氏名 {{ currentCourse.teacher }}</li>
                    </ul>
                  </div>
                  <div>
                    <NuxtLink :to="`/courses/${currentCourse.id}`">
                      <Button>講義情報</Button>
                    </NuxtLink>
                  </div>
                </div>
              </Card>
            </div>
          </template>
          <template v-else>
            <p>現在の時間は履修している開講中の講義はありません。</p>
          </template>
        </section>

        <template v-if="hasError">
          <InlineNotification type="error" class="my-4">
            <template #title>APIエラーがあります</template>
            <template #message>履修済み科目の取得に失敗しました。</template>
          </InlineNotification>
        </template>

        <section>
          <h2 class="text-lg mt-4">2021年度前期</h2>
          <Calendar>
            <template v-for="(periodCourses, p) in courses">
              <template v-for="(course, w) in periodCourses">
                <CalendarCell
                  :key="`course-${p}-${w}`"
                  :cursor="course !== undefined ? 'pointer' : 'default'"
                >
                  <template v-if="course !== undefined">
                    <NuxtLink
                      :to="`/courses/${course.id}`"
                      class="
                        flex-grow
                        h-30
                        px-2
                        py-2
                        w-full
                        transition
                        duration-500
                        ease
                        hover:bg-primary-100
                      "
                    >
                      <div class="flex flex-col">
                        <span class="text-primary-500 font-bold">{{
                          course.name
                        }}</span>
                        <span class="text-sm">{{ course.teacher }}</span>
                      </div>
                    </NuxtLink>
                  </template>
                  <template v-else></template>
                </CalendarCell>
              </template>
            </template>
          </Calendar>
        </section>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'
import Calendar from '../components/Calendar.vue'
import CalendarCell from '../components/CalendarCell.vue'
import Card from '~/components/common/Card.vue'
import Button from '~/components/common/Button.vue'
import { DayOfWeekMap, PeriodCount, WeekdayCount } from '~/constants/calendar'
import { Course, DayOfWeek } from '~/types/courses'
import InlineNotification from '~/components/common/InlineNotification.vue'

type MinimalCourse = Pick<
  Course,
  'id' | 'name' | 'teacher' | 'period' | 'dayOfWeek'
>
type CalendarCourses = (MinimalCourse | undefined)[][]

type DataType = {
  registeredCourses: MinimalCourse[]
  current: {
    dayOfWeek: number
    period: number
  }
  hasError: boolean
}

export default Vue.extend({
  components: { InlineNotification, Button, Card, CalendarCell, Calendar },
  middleware: 'is_student',
  async asyncData(
    ctx: Context
  ): Promise<{ registeredCourses: MinimalCourse[]; hasError: boolean }> {
    try {
      const res = await ctx.$axios.get<MinimalCourse[]>(`/api/users/me/courses`)
      return { registeredCourses: res.data ?? [], hasError: false }
    } catch (e) {
      console.error(e)
    }

    return { registeredCourses: [], hasError: true }
  },
  data(): DataType {
    return {
      registeredCourses: [],
      current: {
        dayOfWeek: -1,
        period: -1,
      },
      hasError: false,
    }
  },
  head: {
    title: 'ISUCHOLAR - マイページ',
  },
  computed: {
    courses(): CalendarCourses {
      const periodCourses: CalendarCourses = []
      for (let period = 1; period <= PeriodCount; period++) {
        const weekdayCourses = []
        for (let weekday = 1; weekday <= WeekdayCount; weekday++) {
          const course = this.getCourse(period, weekday)
          weekdayCourses.push(course)
        }
        periodCourses.push(weekdayCourses)
      }

      return periodCourses
    },
    currentCourse(): MinimalCourse | undefined {
      return this.courses?.[this.current.period - 1]?.[
        this.current.dayOfWeek - 1
      ]
    },
  },
  created() {
    setInterval(() => {
      const now = new Date()
      const minute = now.getMinutes()
      const second = now.getSeconds()

      this.current = Object.assign({}, this.current, {
        dayOfWeek: minute % 5,
        period: Math.floor(second / 10) + 1,
      })
    }, 1000 /* 10秒ごと */)
  },
  methods: {
    getCourse(period: number, weekday: number): MinimalCourse | undefined {
      const course = this.registeredCourses.find((c) => {
        const dayOfWeek = DayOfWeekMap[c.dayOfWeek as DayOfWeek]
        return period === c.period && weekday === dayOfWeek
      })
      return course
    },
  },
})
</script>
