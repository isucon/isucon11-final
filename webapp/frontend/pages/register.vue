<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg">
      <div class="flex-1 flex-col">
        <h1 class="text-2xl">履修登録</h1>

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

        <div>
          <Button class="mr-2" @click="onClickSearchCourse">科目検索</Button>
          <Button color="primary" @click="onClickConfirm">内容の確定</Button>
        </div>
        <Calendar>
          <template v-for="(c, i) in courses">
            <CalendarCell :key="`course-${i}`">
              <template v-if="c !== undefined">
                <NuxtLink
                  :to="`/courses/${c.id}`"
                  class="flex-grow h-30 py-1 w-full cursor-pointer"
                >
                  <span>プログラミング演習B</span><span>教員名</span
                  ><span>講義場所</span></NuxtLink
                >
              </template>
              <template v-else>
                <button class="flex-grow h-30 w-full cursor-pointer">
                  登録(icon)
                </button>
              </template>
            </CalendarCell>
          </template>
        </Calendar>
      </div>
    </div>
    <SearchModal
      v-model="willRegisterCourses"
      :is-shown="isShownModal"
      @close="onCloseSearchModal"
    />
  </div>
</template>
<script lang="ts">
import Vue from 'vue'
import Button from '../components/common/Button.vue'
import Calendar from '../components/Calendar.vue'
import CalendarCell from '../components/CalendarCell.vue'
import SearchModal from '../components/SearchModal.vue'

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
  isShownModal: boolean
  willRegisterCourses: Course[]
}

export default Vue.extend({
  components: {
    SearchModal,
    CalendarCell,
    Calendar,
    Button,
  },
  middleware: 'is_loggedin',
  data(): DataType {
    return {
      isShownModal: false,
      willRegisterCourses: [],
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
      const course = this.willRegisterCourses.find((c) => {
        const dayOfWeek = DayOfWeek[c.dayOfWeek]
        return idx === dayOfWeek + (c.period - 1) * 5
      })
      return course
    },
    onClickSearchCourse(): void {
      this.isShownModal = true
    },
    onCloseSearchModal(): void {
      this.isShownModal = false
    },
    async onClickConfirm(): Promise<void> {
      const path = `/api/users/me/courses`
      const ids = this.willRegisterCourses.map((c) => ({ id: c.id }))
      const res = await this.$axios.put(path, ids)
      if (res.status === 200) {
        await this.$router.push('mypage')
      }
    },
  },
})
</script>
