<template>
  <div>
    <Card>
      <div class="flex flex-col justify-between leading-normal">
        <p class="text-2xl text-primary-500 font-bold flex items-center">
          {{ classinfo.title }}
        </p>
        <p class="text-gray-800 text-base mb-4">{{ classinfo.description }}</p>
        <div class="flex flex-row items-center">
          <Button :disabled="submissionDisabled" @click="openModal">
            課題を提出する
          </Button>
          <div class="text-neutral-300 text-sm ml-4">
            {{ submissionStatus }}
          </div>
        </div>
      </div>
    </Card>
    <SubmitModal
      :is-shown="showModal"
      :course-name="course.name"
      :class-title="classinfo.title"
      :class-id="classinfo.id"
      @submitted="$emit('submitted')"
      @close="closeModal"
    />
  </div>
</template>

<script lang="ts">
import Vue, { PropOptions } from 'vue'
import { Course, ClassInfo } from '~/types/courses'
import Card from '~/components/common/Card.vue'
import SubmitModal from '~/components/SubmitModal.vue'

type ClassInfoData = {
  showModal: boolean
}

export default Vue.extend({
  name: 'ClassInfo',
  components: {
    Card,
    SubmitModal,
  },
  props: {
    course: {
      type: Object,
      required: true,
    } as PropOptions<Course>,
    classinfo: {
      type: Object,
      required: true,
    } as PropOptions<ClassInfo>,
  },
  data(): ClassInfoData {
    return {
      showModal: false,
    }
  },
  computed: {
    submissionStatus(): string {
      if (this.classinfo.submitted) {
        return '提出済み'
      } else {
        return this.classinfo.submissionClosed ? '締切済み' : '受付中'
      }
    },
    submissionDisabled(): boolean {
      return this.classinfo.submissionClosed
    },
  },
  methods: {
    openModal() {
      this.showModal = true
    },
    closeModal() {
      this.showModal = false
    },
    download(name: string, data: Blob) {
      const link = document.createElement('a')
      link.href = window.URL.createObjectURL(data)
      link.download = name
      link.click()
    },
  },
})
</script>
