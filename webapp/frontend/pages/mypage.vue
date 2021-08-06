<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg">
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

        <section class="mt-10">
          <h1 class="text-2xl">成績照会</h1>

          <p class="py-2">今期の成績は成績照会ページで確認してください</p>

          <div class="py-4">
            <NuxtLink to="grade">
              <Button>成績照会</Button>
            </NuxtLink>
          </div>
        </section>

        <section class="mt-10">
          <h1 class="text-2xl">時間割</h1>

          <div class="py-2">
            <h2 class="text-xl">履修登録</h2>
            <p>履修登録ページから履修登録を行ってください。</p>
          </div>

          <div class="py-4">
            <NuxtLink to="register">
              <Button color="primary">履修登録</Button>
            </NuxtLink>
          </div>
        </section>

        <section>
          <h2 class="text-lg">2021年度前期</h2>
          <Calendar>
            <template v-for="(c, i) in courses">
              <CalendarCell
                :key="`course-${i}`"
                :cursor="c !== undefined ? 'pointer' : 'default'"
              >
                <template v-if="c !== undefined">
                  <NuxtLink
                    :to="`/courses/${c.id}`"
                    class="flex-grow h-30 py-1 w-full"
                  >
                    <div class="flex flex-col">
                      <span class="text-primary-500 font-bold">{{
                        c.name
                      }}</span>
                      <span class="text-sm">{{ c.teacher }}</span>
                    </div>
                  </NuxtLink>
                </template>
                <template v-else></template>
              </CalendarCell>
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

type MinimalCourse = Pick<
  Course,
  'id' | 'name' | 'teacher' | 'period' | 'dayOfWeek'
>

type DataType = {
  registeredCourses: MinimalCourse[]
  current: {
    dayOfWeek: number
    period: number
  }
}

export default Vue.extend({
  components: { Button, Card, CalendarCell, Calendar },
  middleware: 'is_loggedin',
  async asyncData(
    ctx: Context
  ): Promise<{ registeredCourses: MinimalCourse[] }> {
    try {
      const registeredCourses = await ctx.$axios.$get<MinimalCourse[]>(
        `/api/users/me/courses`
      )
      return { registeredCourses }
    } catch (e) {
      console.error(e)
    }

    return { registeredCourses: [] }
  },
  data(): DataType {
    return {
      registeredCourses: [],
      current: {
        dayOfWeek: -1,
        period: -1,
      },
    }
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
  computed: {
    courses(): (MinimalCourse | undefined)[] {
      return new Array(WeekdayCount * PeriodCount)
        .fill(undefined)
        .map((_, i) => {
          return this.getCourse(i)
        })
    },
    currentCourse(): MinimalCourse | undefined {
      const idx =
        this.current.dayOfWeek + (this.current.period - 1) * WeekdayCount
      return this.courses[idx]
    },
  },
  methods: {
    getCourse(idx: number): MinimalCourse | undefined {
      const course = this.registeredCourses.find((c) => {
        const dayOfWeek = DayOfWeekMap[c.dayOfWeek as DayOfWeek]
        return idx === dayOfWeek + (c.period - 1) * WeekdayCount
      })
      return course
    },
  },
})
</script>
