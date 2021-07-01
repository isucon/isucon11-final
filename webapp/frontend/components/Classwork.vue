<template>
  <div>
    <div class="mt-2">
      <card>
        <div class="mb-8 flex flex-col justify-between leading-normal">
          <p class="text-2xl text-primary-500 font-bold flex items-center">
            {{ classwork.title }}
          </p>
          <div class="text-neutral-300 text-sm mb-4">
            {{ classwork.startedAt }}
          </div>
          <p class="text-black text-base mb-4">{{ classwork.description }}</p>
          <div class="mb-4">
            <p class="text-lg font-bold">講義資料</p>
            <div v-for="doc in classwork.documents" :key="doc.id">
              <a
                class="cursor-pointer text-primary-400"
                @click="download(doc)"
                >{{ doc.name }}</a
              >
            </div>
          </div>
          <div>
            <p class="text-lg font-bold mb-4">課題・レポート</p>
            <table class="table-auto">
              <thead>
                <tr class="bg-gray-300">
                  <th class="px-4 py-2">課題番号</th>
                  <th class="px-4 py-2">課題名</th>
                  <th class="px-4 py-2">提出期限</th>
                  <th class="px-4 py-2">提出日</th>
                  <th class="px-4 py-2"></th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(assignment, index) of classwork.assignments"
                  :key="assignment.id"
                  :class="getRowBgColor(index)"
                >
                  <td class="px-4 py-2">{{ index }}</td>
                  <td class="px-4 py-2">{{ assignment.name }}</td>
                  <td class="px-4 py-2">{{ assignment.deadline }}</td>
                  <td class="px-4 py-2">
                    {{ getAssignmentStatusMessage(assignment) }}
                  </td>
                  <td class="px-4 py-2">
                    <span
                      class="text-primary-400 cursor-pointer"
                      @click="openModal(index)"
                      >課題の提出</span
                    >
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </card>
    </div>
    <submit-modal
      :is-shown="modalVisibility"
      :course-name="course.name"
      :class-title="classwork.title"
      :assignment-name="assignmentName"
      :assignment-id="assignmentId"
      @close="closeModal"
    />
  </div>
</template>

<script lang="ts">
import Vue, { PropOptions } from 'vue'
import { Course, Classwork, Document, Assignment } from '@/interfaces/courses'

interface ClassworkData {
  assignmentName: string
  assignmentId: string
}

export default Vue.extend({
  name: 'Classwork',
  props: {
    course: {
      type: Object,
      required: true,
    } as PropOptions<Course>,
    classwork: {
      type: Object,
      required: true,
    } as PropOptions<Classwork>,
  },
  data(): ClassworkData {
    return {
      assignmentName: '',
      assignmentId: '',
    }
  },
  computed: {
    modalVisibility(): boolean {
      return this.assignmentId !== ''
    },
  },
  methods: {
    getRowBgColor(index: number) {
      return index % 2 === 1 ? 'bg-gray-200' : null
    },
    // `submitted` is not implemented yet
    // getAssignmentStatusMessage(assignment: Assignment) {
    //   return assignment.submitted ? '提出済み' : '提出ファイルはありません'
    // },
    getAssignmentStatusMessage(_assignment: Assignment) {
      return '提出済み'
    },
    openModal(index: number) {
      this.assignmentName = this.classwork.assignments[index].name
      this.assignmentId = this.classwork.assignments[index].id
    },
    closeModal() {
      this.assignmentName = ''
      this.assignmentId = ''
    },
    download(doc: Document) {
      this.$axios
        .$get(`/api/courses/${this.course.id}/documents/${doc.id}`, {
          responseType: 'blob',
        })
        .then((response) => {
          const link = document.createElement('a')
          link.href = window.URL.createObjectURL(response)
          link.download = doc.name
          link.click()
        })
    },
  },
})
</script>
