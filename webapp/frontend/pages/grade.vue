<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12">
      <div class="flex-1 flex-col">
        <section>
          <h1 class="text-2xl">個人成績照会</h1>

          <div class="py-4">
            <table class="table-auto border">
              <tr>
                <th class="bg-primary-500 text-white border py-2.5 px-2.5">
                  氏名
                </th>
                <td class="border px-2.5">椅子 近</td>
              </tr>
              <tr>
                <th class="bg-primary-500 text-white border py-2.5 px-2.5">
                  学籍番号
                </th>
                <td class="border px-2.5">s123456789</td>
              </tr>
              <tr>
                <th class="bg-primary-500 text-white border py-2.5 px-2.5">
                  所属
                </th>
                <td class="border px-2.5">椅子学部椅子解析工学科</td>
              </tr>
            </table>
          </div>
        </section>

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
                <td class="px-4 py-2 border">{{ grades.summary.gptStd }}</td>
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
              <div :key="r.code" class="px-4 py-2 border">
                {{ r.code }}
              </div>
              <div :key="r.name" class="px-4 py-2 border">
                {{ r.name }}
              </div>
              <div :key="r.totalScore" class="px-4 py-2 border">
                {{ r.totalScore }}
              </div>
              <div :key="r.totalScoreAvg" class="px-4 py-2 border">
                {{ r.totalScoreAvg }}
              </div>
              <div :key="r.totalScoreStd" class="px-4 py-2 border">
                {{ r.totalScoreStd }}
              </div>
              <div :key="r.totalScoreMin" class="px-4 py-2 border">
                {{ r.totalScoreMin }}
              </div>
              <div :key="r.totalScoreMax" class="px-4 py-2 border">
                {{ r.totalScoreMax }}
              </div>
              <div :key="`button-${i}`" class="px-4 py-2 border">
                <Button color="plain" @click="onClickClassDetail(i)"
                  >開く
                </Button>
              </div>
              <template v-if="includeOpenedIndex(i)">
                <div
                  :key="`cousre-score-${i}`"
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
                  <template v-for="s in r.classScores">
                    <div :key="s.part" class="px-4 py-2 border bg-white">
                      {{ s.part }}
                    </div>
                    <div :key="s.title" class="px-4 py-2 border bg-white">
                      {{ s.title }}
                    </div>
                    <div :key="s.score" class="px-4 py-2 border bg-white">
                      {{ s.score }}
                    </div>
                    <div :key="s.submitters" class="px-4 py-2 border bg-white">
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
import { Grade } from '~/types/courses'
import Button from '~/components/common/Button.vue'

type Data = {
  grades: Grade | undefined
  openedIndex: number[]
}

export default Vue.extend({
  components: { Button },
  asyncData() {
    // const grades = await ctx.$axios.get('/api/users/me/grades')
    const grades = {
      summary: {
        gpt: 1,
        credits: 200,
        gptAvg: 50,
        gptStd: 50,
        gptMax: 50,
        gptMin: 50,
      },
      courses: [
        {
          name: 'コース1',
          code: 'course1',
          totalScore: 50,
          totalScoreAvg: 50,
          totalScoreStd: 50,
          totalScoreMax: 50,
          totalScoreMin: 50,
          classScores: [
            {
              title: '講義1',
              part: 1,
              score: 50,
              submitters: 1,
            },
            {
              title: '講義2',
              part: 2,
              score: 50,
              submitters: 1,
            },
          ],
        },
        {
          name: 'コース2',
          code: 'course2',
          totalScore: 50,
          totalScoreAvg: 50,
          totalScoreStd: 50,
          totalScoreMax: 50,
          totalScoreMin: 50,
          classScores: [
            {
              title: '講義1',
              part: 1,
              score: 50,
              submitters: 1,
            },
            {
              title: '講義2',
              part: 2,
              score: 50,
              submitters: 1,
            },
          ],
        },
      ],
    }

    return { grades }
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
