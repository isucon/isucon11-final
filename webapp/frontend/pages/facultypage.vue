<template>
  <div>
    <div class="py-10 px-8 bg-white shadow-lg w-8/12">
      <div class="flex-1 flex-col">
        <section>
          <h1 class="text-2xl">科目</h1>
          <div class="py-4">
            <Button @click="visibleModal = 'AddCourse'">新規登録</Button>
          </div>
          <div class="py-4">
            <Button>ステータス変更</Button>
          </div>
        </section>

        <section class="mt-10">
          <h1 class="text-2xl">講義</h1>
          <div class="py-4">
            <Button>新規登録</Button>
          </div>
          <div class="py-4">
            <Button>成績登録</Button>
          </div>
          <div class="py-4">
            <Button>提出課題のダウンロード</Button>
          </div>
        </section>
      </div>
    </div>
    <AddCourseModal
      :is-shown="visibleModal === 'AddCourse'"
      @close="visibleModal = null"
    />
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import Button from '~/components/common/Button.vue'
import AddCourseModal from '~/components/AddCourseModal.vue'

type modalKinds =
  | 'AddCourse'
  | 'SetCourseStatus'
  | 'AddClass'
  | 'RegisterScores'
  | null

type FacultyPageData = {
  visibleModal: modalKinds
}

export default Vue.extend({
  components: {
    Button,
    AddCourseModal,
  },
  middleware: 'is_loggedin',
  data(): FacultyPageData {
    return {
      visibleModal: null,
    }
  },
})
</script>
