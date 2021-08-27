<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12 mt-8 mb-8 rounded">
      <div class="flex-1 flex-col">
        <h1 class="text-2xl">個人成績照会</h1>

        <section class="mt-10">
          <h1 class="text-xl">成績概要</h1>

          <table class="table-auto my-2">
            <thead>
              <tr class="bg-gray-300">
                <th class="px-4 py-2">取得単位数</th>
                <th class="px-4 py-2">GPT</th>
                <th class="px-4 py-2">全学生GPT平均値</th>
                <th class="px-4 py-2">全学生GPT偏差値</th>
                <th class="px-4 py-2">全学生GPT最低値</th>
                <th class="px-4 py-2">全学生GPT最大値</th>
              </tr>
            </thead>
            <tbody>
              <tr class="bg-gray-200 odd:bg-white">
                <td class="px-4 py-2 border">{{ grades.summary.credits }}</td>
                <td class="px-4 py-2 border">{{ grades.summary.gpt }}</td>
                <td class="px-4 py-2 border">{{ grades.summary.gptAvg }}</td>
                <td class="px-4 py-2 border">{{ grades.summary.gptTScore }}</td>
                <td class="px-4 py-2 border">{{ grades.summary.gptMin }}</td>
                <td class="px-4 py-2 border">{{ grades.summary.gptMax }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <section class="mt-10">
          <h1 class="text-xl">成績一覧</h1>

          <div class="my-2 grid grid-cols-8 items-stretch">
            <div class="px-4 py-2 border bg-gray-300">科目コード</div>
            <div class="px-4 py-2 border bg-gray-300">科目名</div>
            <div class="px-4 py-2 border bg-gray-300">成績</div>
            <div class="px-4 py-2 border bg-gray-300">平均点</div>
            <div class="px-4 py-2 border bg-gray-300">偏差値</div>
            <div class="px-4 py-2 border bg-gray-300">最低点</div>
            <div class="px-4 py-2 border bg-gray-300">最高点</div>
            <div class="px-4 py-2 border bg-gray-300">各講義成績</div>

            <template v-for="(r, i) in grades.courses">
              <div :key="`course${i}-code${r.code}`" class="px-4 py-2 border">
                {{ r.code }}
              </div>
              <div :key="`course${i}-name${r.name}`" class="px-4 py-2 border">
                {{ r.name }}
              </div>
              <div
                :key="`course${i}-totalScore${r.totalScore}`"
                class="px-4 py-2 border"
              >
                {{ r.totalScore }}
              </div>
              <div
                :key="`course${i}-totalScoreAvg${r.totalScoreAvg}`"
                class="px-4 py-2 border"
              >
                {{ r.totalScoreAvg }}
              </div>
              <div
                :key="`course${i}-totalScoreTScore${r.totalScoreTScore}`"
                class="px-4 py-2 border"
              >
                {{ r.totalScoreTScore }}
              </div>
              <div
                :key="`course${i}-totalScoreMin${r.totalScoreMin}`"
                class="px-4 py-2 border"
              >
                {{ r.totalScoreMin }}
              </div>
              <div
                :key="`course${i}-totalScoreMax${r.totalScoreMax}`"
                class="px-4 py-2 border"
              >
                {{ r.totalScoreMax }}
              </div>
              <div :key="`button${i}`" class="px-2 py-2 border">
                <Button color="plain" @click="onClickClassDetail(i)">
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
                  <div class="px-4 py-2 bg-gray-300">成績</div>
                  <div class="px-4 py-2 bg-gray-300">課題提出者数</div>
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
                      {{ s.score }}
                    </div>
                    <div
                      :key="`courseScore${j}-submitters${s.submitters}`"
                      class="px-4 py-2 border bg-white"
                    >
                      {{ s.submitters }}
                    </div>
                  </template>
                </div>
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

type Data = {
  grades: Grade | undefined
  openedIndex: number[]
}

export default Vue.extend({
  components: { Button },
  middleware: 'is_student',
  async asyncData(ctx: Context) {
    try {
      const res = await ctx.$axios.get('/api/users/me/grades')
      if (res.status === 200) {
        return { grades: res.data }
      }
    } catch (e) {
      console.error(e)
    }

    return { grades: undefined }
  },
  data(): Data {
    return {
      grades: undefined,
      openedIndex: [],
    }
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
  },
})
</script>
