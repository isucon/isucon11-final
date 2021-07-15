<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg">
      <div class="flex-1 flex-col">
        <section>
          <h1 class="text-2xl">現在開講中の講義</h1>

          <div class="py-4">
            <Card>
              <div class="flex justify-between items-center">
                <div>
                  <h2 class="text-xl mb-2 text-primary-500">線形代数</h2>
                  <ul class="list-none text-gray-500">
                    <li>教員氏名 xxx</li>
                    <li>講義場所 xxx</li>
                  </ul>
                </div>
                <div>
                  <Button>講義情報</Button>
                  <Button color="primary">出席入力</Button>
                </div>
              </div>
            </Card>
          </div>
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
            <p>履修登録期間は xx/xx から xx/xx までです。</p>
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
              <CalendarCell :key="`course-${i}`">
                <template v-if="c !== undefined">
                  <NuxtLink
                    :to="`/courses/${c.id}`"
                    class="flex-grow h-30 py-1 w-full cursor-pointer"
                  >
                    <span>{{ c.name }}</span
                    ><span>教員名</span><span>講義場所</span></NuxtLink
                  >
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

const DayOfWeek = {
  monday: 1,
  tuesday: 2,
  wednesday: 3,
  thursday: 4,
  friday: 5,
}

type Course = {
  id: string
  name: string
  credit: number
  dayOfWeek: keyof typeof DayOfWeek
  period: number
}

type DataType = {
  registeredCourses: Course[]
}

export default Vue.extend({
  components: { Button, Card, CalendarCell, Calendar },
  middleware: 'is_loggedin',
  async asyncData(ctx: Context): Promise<{ registeredCourses: Course[] }> {
    const registeredCourses = await ctx.$axios.$get<Course[]>(
      `/api/users/me/courses`
    )

    return { registeredCourses }
  },
  data(): DataType {
    return {
      registeredCourses: [],
    }
  },
  computed: {
    courses(): (Course | undefined)[] {
      return new Array(25).fill(undefined).map((_, i) => {
        return this.getCourse(i + 1)
      })
    },
  },
  methods: {
    getCourse(idx: number): Course | undefined {
      const course = this.registeredCourses.find((c) => {
        const dayOfWeek = DayOfWeek[c.dayOfWeek]
        return idx === dayOfWeek + (c.period - 1) * 5
      })
      return course
    },
  },
})
</script>
