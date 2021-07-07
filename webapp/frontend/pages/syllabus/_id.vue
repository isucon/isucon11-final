<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12">
      <div class="flex-1 flex-col">
        <section>
          <h1 class="text-2xl">科目概要</h1>

          <div
            class="grid grid-cols-syllabus justify-items-stretch items-stretch"
          >
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              <span class="text-center">科目名</span>
            </div>
            <div class="px-2 py-2 border">
              {{ course.name }}
            </div>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              科目番号
            </div>
            <div class="px-2 py-2 border">
              {{ course.id }}
            </div>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              科目種別
            </div>
            <div class="px-2 py-2 border">
              {{ course.type }}
            </div>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              単位数
            </div>
            <div class="px-2 py-2 border">
              {{ course.credit }}
            </div>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              開講期
            </div>
            <div class="px-2 py-2 border">
              {{ course.year }} {{ course.semester }}
            </div>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              時限
            </div>
            <div class="px-2 py-2 border">
              {{ course.day_of_week }}{{ course.period }}
            </div>

            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              前提科目
            </div>
            <template v-if="course.required_courses.id">
              <NuxtLink
                class="px-2 py-2 border"
                :to="`/syllabus/${course.required_courses.id}`"
                >{{ course.required_courses.name }}
              </NuxtLink>
            </template>
            <template v-else>
              <div class="px-2 py-2 border">なし</div>
            </template>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              担当教員
            </div>
            <div class="px-2 py-2 border">
              {{ course.teacher }}
            </div>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              講義内容
            </div>
            <div class="px-2 py-2 border">
              {{ course.description }}
            </div>
            <div
              class="
                px-2
                py-2
                bg-primary-500
                text-white
                flex flex-col
                justify-center
                items-center
                border
              "
            >
              キーワード
            </div>
            <div class="px-2 py-2 border">
              {{ course.keywords }}
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'

export default Vue.extend({
  async asyncData(ctx: Context): Promise<object> {
    const id = ctx.params.id
    const res = await ctx.$axios.get(`/api/syllabus/${id}`)

    let course = {}
    if (res.status === 200) {
      course = res.data
    }

    return { course }
  },
  data() {
    return {
      course: {},
    }
  },
})
</script>
