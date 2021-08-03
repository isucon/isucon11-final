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
              {{ course.code }}
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
              {{ courseType }}
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
              時限
            </div>
            <div class="px-2 py-2 border">
              {{ coursePeriod }}
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
import { Course } from '~/types/courses'
import { formatType, formatPeriod } from '~/helpers/course_helper'

type SyllabusData = {
  course: Course
}

export default Vue.extend({
  middleware: 'is_loggedin',
  async asyncData(ctx: Context): Promise<SyllabusData> {
    const id = ctx.params.id
    const res = await ctx.$axios.get(`/api/syllabus/${id}`)
    const course: Course = res.data

    return { course }
  },
  data(): SyllabusData {
    return {
      course: {
        id: '',
        code: '',
        type: 'liberal-arts',
        name: '',
        description: '',
        credit: 0,
        period: 0,
        dayOfWeek: 'sunday',
        teacher: '',
        keywords: '',
      },
    }
  },
  computed: {
    courseType(): String {
      return formatType(this.course.type)
    },
    coursePeriod(): String {
      return formatPeriod(this.course.dayOfWeek, this.course.period)
    },
  },
})
</script>
