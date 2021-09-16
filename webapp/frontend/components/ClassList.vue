<template>
  <div class="py-4">
    <div v-if="classList.length" class="flex flex-col gap-4">
      <ClassInfoCard
        v-for="(cls, index) in classList"
        :key="cls.id"
        :course="course"
        :classinfo="cls"
        @submitted="submissionComplete(index)"
      />
    </div>
    <div v-else class="min-h-28">
      <div>講義情報が登録されていません</div>
    </div>
  </div>
</template>
<script lang="ts">
import Vue, { PropOptions } from 'vue'
import { Course, ClassInfo } from '~/types/courses'
import ClassInfoCard from '~/components/ClassInfo.vue'

export default Vue.extend({
  components: {
    ClassInfoCard,
  },
  props: {
    course: {
      type: Object,
      required: true,
    } as PropOptions<Course>,
    classes: {
      type: Array,
      required: true,
    } as PropOptions<ClassInfo[]>,
  },
  data(): { classList: ClassInfo[] } {
    return { classList: this.classes }
  },
  methods: {
    submissionComplete(classIdx: number) {
      this.classList[classIdx].submitted = true
    },
  },
})
</script>
