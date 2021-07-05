<template>
  <Modal :is-shown="isShown" @close="$emit('close')">
    <div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
      <div class="flex flex-col flex-nowrap">
        <h3
          id="modal-title"
          class="text-lg leading-6 font-medium text-gray-900"
        >
          科目検索
        </h3>
        <form class="flex-1 flex-col" @submit.prevent="onSubmitSearch">
          <div class="flex items-center">
            <TextField
              id="inline-query"
              label="キーワード"
              type="text"
              placeholder="キーワードを入力してください"
            />
          </div>
          <div class="flex mt-4 space-x-1">
            <label class="whitespace-nowrap block text-gray-500 font-bold pr-4"
              >対象</label
            >
            <Select
              id="faculty"
              label="学部・研究科"
              :value="[{ text: '工学部', value: 'Engineering' }]"
            />
            <Select
              id="department"
              label="学科・専攻"
              :value="[{ text: 'xx学科', value: 'xxx' }]"
            />
            <Select
              id="year"
              label="年次"
              :value="[{ text: '1', value: '1' }]"
            />
          </div>
          <div class="flex mt-4 space-x-1">
            <label class="whitespace-nowrap block text-gray-500 font-bold pr-4"
              >開講</label
            >
            <Select
              id="semester"
              label="学期"
              :value="[{ text: '工学部', value: 'Engineering' }]"
            />
            <Select
              id="day-of-week"
              label="曜日"
              :value="[
                { text: '月曜', value: '1' },
                { text: '火曜', value: '2' },
                { text: '水曜', value: '3' },
                { text: '木曜', value: '4' },
                { text: '金曜', value: '5' },
              ]"
            />
            <Select
              id="period"
              label="時限"
              :value="[
                { text: '1', value: '1' },
                { text: '2', value: '2' },
                { text: '3', value: '3' },
                { text: '4', value: '4' },
                { text: '5', value: '5' },
              ]"
            />
          </div>
          <Button type="submit" class="mt-6 flex-grow-0" color="primary"
            >検索
          </Button>
        </form>

        <div v-if="isShowSearchResult">
          <h3 class="text-xl">検索結果: xx件</h3>
          <table class="table-auto border w-full">
            <tr class="text-center">
              <th>選択</th>
              <th>科目コード</th>
              <th>科目名</th>
              <th>学期</th>
              <th>時間</th>
              <th>単位数</th>
              <th>担当</th>
              <th></th>
            </tr>
            <template v-for="(c, i) in courses">
              <tr :key="`tr-${i}`" class="text-center">
                <td>
                  <input
                    type="checkbox"
                    class="
                      form-input
                      text-primary-500
                      focus:outline-none focus:ring-primary-200
                    "
                    :checked="isChecked(c.id)"
                    @change="onChangeCheckbox(c)"
                  />
                </td>
                <td>{{ c.id }}</td>
                <td>{{ c.name }}</td>
                <td>前期</td>
                <td>{{ c.dayOfWeek }}{{ c.period }}</td>
                <td>{{ c.credit }}</td>
                <td>椅子 昆</td>
                <td>
                  <NuxtLink :to="`/course/${c.id}`" class="text-primary-500"
                    >詳細を見る
                  </NuxtLink>
                </td>
              </tr>
            </template>
          </table>
          <Button @click="onSubmitTemporaryRegistration">仮登録</Button>
        </div>
      </div>
    </div>
  </Modal>
</template>
<script lang="ts">
import Vue, { PropType } from 'vue'
import Modal from './common/Modal.vue'
import TextField from './common/TextField.vue'
import Select from './common/Select.vue'
import Button from '~/components/common/Button.vue'

type Course = {
  id: string
  name: string
  credit: number
  dayOfWeek: string
  period: number
}

type DataType = {
  courses: Course[]
  checkedCourses: Course[]
}

export default Vue.extend({
  components: { Button, Select, TextField, Modal },
  props: {
    isShown: {
      type: Boolean,
      default: false,
      required: true,
    },
    value: {
      type: Array as PropType<Course[]>,
      default: () => [],
      required: true,
    },
  },
  data(): DataType {
    return {
      courses: [],
      checkedCourses: this.value,
    }
  },
  computed: {
    isShowSearchResult(): boolean {
      return this.courses.length > 0
    },
  },
  methods: {
    isChecked(courseId: string): boolean {
      const course = this.checkedCourses.find((v) => v.id === courseId)
      return course !== undefined
    },
    async onSubmitSearch(): Promise<void> {
      await Promise.resolve()
      this.courses = [
        {
          id: '01234567-89ab-cdef-0002-000000000001',
          name: '微分積分基礎',
          credit: 2,
          dayOfWeek: 'monday',
          period: 1,
        },
        {
          id: '01234567-89ab-cdef-0002-000000000002',
          name: '線形代数基礎',
          credit: 2,
          dayOfWeek: 'monday',
          period: 3,
        },
        {
          id: '01234567-89ab-cdef-0002-000000000003',
          name: 'アルゴリズム基礎',
          credit: 2,
          dayOfWeek: 'thursday',
          period: 2,
        },
        {
          id: '01234567-89ab-cdef-0002-000000000011',
          name: '微分積分応用',
          credit: 2,
          dayOfWeek: 'thursday',
          period: 4,
        },
        {
          id: '01234567-89ab-cdef-0002-000000000012',
          name: '線形代数応用',
          credit: 2,
          dayOfWeek: 'wednesday',
          period: 3,
        },
        {
          id: '01234567-89ab-cdef-0002-000000000013',
          name: 'プログラミング',
          credit: 2,
          dayOfWeek: 'wednesday',
          period: 5,
        },
        {
          id: '01234567-89ab-cdef-0002-000000000014',
          name: 'プログラミング演習A',
          credit: 2,
          dayOfWeek: 'friday',
          period: 1,
        },
        {
          id: '01234567-89ab-cdef-0002-000000000015',
          name: 'プログラミング演習B',
          credit: 1,
          dayOfWeek: 'friday',
          period: 2,
        },
      ]
      // const res = await this.$axios.get('/api/syllabus')
      // if (res.status === 200) {
      //   this.courses = res.data
      // }
    },
    onChangeCheckbox(course: Course): void {
      const c = this.checkedCourses.find((v) => v.id === course.id)
      if (c) {
        this.checkedCourses = this.checkedCourses.filter(
          (v) => v.id !== course.id
        )
      } else {
        this.checkedCourses = [...this.checkedCourses, course]
      }
    },
    onSubmitTemporaryRegistration(): void {
      this.$emit('input', this.checkedCourses)
      this.$emit('close')
    },
  },
})
</script>
