<template>
  <div>
    <div
      class="py-10 px-8 bg-white shadow-lg w-192 max-w-full mt-8 mb-8 rounded"
    >
      <div class="flex-1 flex-col">
        <section>
          <h1 class="text-2xl">科目概要</h1>

          <template v-if="hasError">
            <InlineNotification type="error" class="my-4">
              <template #title>APIエラーがあります</template>
              <template #message>科目概要の取得に失敗しました。</template>
            </InlineNotification>
          </template>

          <div
            class="
              grid grid-cols-syllabus
              justify-items-stretch
              items-stretch
              mt-4
            "
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
              科目の状態
            </div>
            <div class="px-2 py-2 border">
              {{ courseStatus }}
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
import { SyllabusCourse } from '~/types/courses'
import { formatType, formatPeriod, formatStatus } from '~/helpers/course_helper'
import InlineNotification from '~/components/common/InlineNotification.vue'

type SyllabusData = {
  title: string
  course: SyllabusCourse
  hasError: boolean
}

const initCourse: SyllabusCourse = {
  id: '',
  code: '',
  type: 'liberal-arts',
  name: '',
  description: '',
  credit: 0,
  period: 0,
  dayOfWeek: 'monday',
  teacher: '',
  keywords: '',
  status: 'registration',
}

export default Vue.extend({
  components: { InlineNotification },
  middleware: 'is_logged_in',
  async asyncData(ctx: Context): Promise<SyllabusData> {
    try {
      const id = ctx.params.id
      const res = await ctx.$axios.get(`/api/courses/${id}`)
      const course: SyllabusCourse = res.data

      return {
        title: `ISUCHOLAR - 科目概要:${course.name}`,
        course,
        hasError: false,
      }
    } catch (e) {
      console.error(e)
    }

    return { title: '', course: initCourse, hasError: true }
  },
  data(): SyllabusData {
    return {
      title: '',
      course: initCourse,
      hasError: false,
    }
  },
  head(): any {
    return {
      title: this.title,
    }
  },
  computed: {
    courseType(): string {
      return formatType(this.course.type)
    },
    coursePeriod(): string {
      return formatPeriod(this.course.dayOfWeek, this.course.period)
    },
    courseStatus(): string {
      return formatStatus(this.course.status)
    },
  },
})
</script>
