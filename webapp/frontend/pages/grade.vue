<template>
  <div>
    <div
      class="py-10 px-8 bg-white shadow-lg w-192 max-w-full mt-8 mb-8 rounded"
    >
      <div class="flex-1 flex-col">
        <h1 class="text-2xl font-bold text-gray-800">個人成績照会</h1>

        <template v-if="hasError">
          <InlineNotification type="error" class="my-4">
            <template #title>APIエラーがあります</template>
            <template #message>成績の取得に失敗しました。</template>
          </InlineNotification>
        </template>

        <section class="mt-10">
          <h1 class="text-xl text-gray-800">成績概要</h1>

          <div class="my-2 grid grid-cols-6 text-gray-900 max-w-full">
            <div class="px-4 py-2 bg-gray-300">取得単位数</div>
            <div class="px-4 py-2 bg-gray-300">GPA</div>
            <div class="px-4 py-2 bg-gray-300">学内 GPA 偏差値</div>
            <div class="px-4 py-2 bg-gray-300">学内 GPA 平均値</div>
            <div class="px-4 py-2 bg-gray-300">学内 GPA 最高値</div>
            <div class="px-4 py-2 bg-gray-300">学内 GPA 最低値</div>

            <div class="px-4 py-2 border break-words">
              {{ grades.summary.credits }}
            </div>
            <div class="px-4 py-2 border break-words">
              {{ round(grades.summary.gpa, digits) }}
            </div>
            <div class="px-4 py-2 border break-words">
              {{ round(grades.summary.gpaTScore, digits) }}
            </div>
            <div class="px-4 py-2 border break-words">
              {{ round(grades.summary.gpaAvg, digits) }}
            </div>
            <div class="px-4 py-2 border break-words">
              {{ round(grades.summary.gpaMax, digits) }}
            </div>
            <div class="px-4 py-2 border break-words">
              {{ round(grades.summary.gpaMin, digits) }}
            </div>
          </div>
        </section>

        <section class="mt-10">
          <h1 class="text-xl text-gray-800">成績一覧</h1>

          <div class="my-2 grid grid-cols-8 text-gray-900">
            <div class="px-4 py-2 bg-gray-300">科目コード</div>
            <div class="px-4 py-2 bg-gray-300">科目名</div>
            <div class="px-4 py-2 bg-gray-300">成績</div>
            <div class="px-4 py-2 bg-gray-300">偏差値</div>
            <div class="px-4 py-2 bg-gray-300">平均点</div>
            <div class="px-4 py-2 bg-gray-300">最高点</div>
            <div class="px-4 py-2 bg-gray-300">最低点</div>
            <div class="px-4 py-2 bg-gray-300">各講義成績</div>

            <template v-if="grades.courses">
              <template v-for="(r, i) in grades.courses">
                <div
                  :key="`course${i}-code${r.code}`"
                  class="px-4 py-2 border break-words"
                >
                  {{ r.code }}
                </div>
                <div :key="`course${i}-name${r.name}`" class="px-4 py-2 border">
                  {{ r.name }}
                </div>
                <div
                  :key="`course${i}-totalScore${r.totalScore}`"
                  class="px-4 py-2 border break-words"
                >
                  {{ r.totalScore }}
                </div>
                <div
                  :key="`course${i}-totalScoreTScore${r.totalScoreTScore}`"
                  class="px-4 py-2 border break-words"
                >
                  {{ round(r.totalScoreTScore, digits) }}
                </div>
                <div
                  :key="`course${i}-totalScoreAvg${r.totalScoreAvg}`"
                  class="px-4 py-2 border break-words"
                >
                  {{ round(r.totalScoreAvg, digits) }}
                </div>
                <div
                  :key="`course${i}-totalScoreMin${r.totalScoreMax}`"
                  class="px-4 py-2 border break-words"
                >
                  {{ r.totalScoreMax }}
                </div>
                <div
                  :key="`course${i}-totalScoreMax${r.totalScoreMin}`"
                  class="px-4 py-2 border break-words"
                >
                  {{ r.totalScoreMin }}
                </div>
                <div
                  :key="`button${i}`"
                  class="flex justify-center items-center border"
                >
                  <Button
                    color="plain"
                    size="mini"
                    @click="onClickClassDetail(i)"
                  >
                    <template v-if="includeOpenedIndex(i)">閉じる</template>
                    <template v-else>開く</template>
                  </Button>
                </div>
                <template v-if="includeOpenedIndex(i)">
                  <div
                    :key="`courseScore${i}`"
                    class="
                      px-4
                      pt-2
                      pb-6
                      col-start-1 col-end-9
                      grid grid-cols-4
                      bg-gray-100
                    "
                  >
                    <div class="px-4 py-2 bg-gray-300">講義回</div>
                    <div class="px-4 py-2 bg-gray-300">講義名</div>
                    <div class="px-4 py-2 bg-gray-300">採点結果</div>
                    <div class="px-4 py-2 bg-gray-300">課題提出者数</div>
                    <template v-if="r.classScores">
                      <template v-for="(s, j) in r.classScores">
                        <div
                          :key="`courseScore${j}-part${s.part}`"
                          class="px-4 py-2 border bg-white"
                        >
                          {{ s.part }}
                        </div>
                        <div
                          :key="`courseScore${j}-title${s.title}`"
                          class="px-4 py-2 border bg-white"
                        >
                          {{ s.title }}
                        </div>
                        <div
                          :key="`courseScore${j}-score${s.score}`"
                          class="px-4 py-2 border bg-white"
                        >
                          {{ s.score === null ? '未提出 / 未採点' : s.score }}
                        </div>
                        <div
                          :key="`courseScore${j}-submitters${s.submitters}`"
                          class="px-4 py-2 border bg-white"
                        >
                          {{ s.submitters }}
                        </div>
                      </template>
                    </template>
                  </div>
                </template>
              </template>
            </template>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'
import { Grade } from '~/types/courses'
import Button from '~/components/common/Button.vue'
import InlineNotification from '~/components/common/InlineNotification.vue'

type Data = {
  digits: number
  grades: Grade | undefined
  openedIndex: number[]
  hasError: boolean
}

const initGrade: Grade = {
  summary: {
    gpa: 0,
    credits: 0,
    gpaAvg: 0,
    gpaMax: 0,
    gpaMin: 0,
    gpaTScore: 0,
  },
  courses: [],
}

const DIGITS = 2

export default Vue.extend({
  components: { InlineNotification, Button },
  middleware: 'is_student',
  async asyncData(ctx: Context) {
    try {
      const res = await ctx.$axios.get('/api/users/me/grades')
      if (res.status === 200) {
        return { grades: res.data, hasError: false }
      }
    } catch (e) {
      console.error(e)
    }

    return { grades: initGrade, hasError: true }
  },
  data(): Data {
    return {
      digits: DIGITS,
      grades: initGrade,
      openedIndex: [],
      hasError: false,
    }
  },
  head: {
    title: 'ISUCHOLAR - 成績照会',
  },
  methods: {
    onClickClassDetail(index: number): void {
      if (this.openedIndex.includes(index)) {
        this.openedIndex = this.openedIndex.filter((i) => !(i === index))
      } else {
        this.openedIndex = [...this.openedIndex, index]
      }
    },
    includeOpenedIndex(index: number): boolean {
      return this.openedIndex.includes(index)
    },
    round(value: number, digits: number = 0): number {
      const base = 10 ** digits
      return Math.round(value * base) / base
    },
  },
})
</script>
