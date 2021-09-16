<template>
  <div>
    <table class="table-fixed w-full">
      <thead>
        <tr class="text-left">
          <th class="px-1 py-1 w-1/8">講義回</th>
          <th class="px-1 py-1 w-1/4">講義タイトル</th>
          <th class="px-1 py-1 w-1/2">講義詳細</th>
          <th class="px-1 py-1 w-1/8"></th>
        </tr>
      </thead>

      <tbody>
        <template v-if="isShowSearchResult">
          <template v-for="(c, i) in classes">
            <tr
              :key="`class-tr-${i}`"
              class="text-left bg-gray-200 odd:bg-white"
            >
              <td class="px-1 py-0.5">{{ c.part }}</td>
              <td class="px-1 py-0.5">
                <p class="truncate">{{ c.title }}</p>
              </td>
              <td class="px-1 py-0.5">
                <p class="truncate">{{ c.description }}</p>
              </td>
              <td class="px-1 py-0.5">
                <div class="relative">
                  <fa-icon
                    icon="ellipsis-v"
                    class="min-w-min px-2 cursor-pointer rounded float-right"
                    @click.stop="onClickClassDropdown(i)"
                  />
                  <div
                    class="
                      absolute
                      right-0
                      mt-2
                      py-1
                      rounded
                      z-20
                      w-52
                      bg-white
                      shadow-2xl
                    "
                    :class="openDropdownIdx === i ? 'show' : 'hidden'"
                  >
                    <a
                      href="#"
                      class="
                        block
                        px-4
                        py-2
                        text-gray-800 text-sm
                        hover:bg-primary-300 hover:text-white
                      "
                      @click.prevent.stop="onClickDownloadSubmissions(i)"
                      >提出課題のダウンロード
                    </a>
                    <a
                      href="#"
                      class="
                        block
                        px-4
                        py-2
                        text-gray-800 text-sm
                        hover:bg-primary-300 hover:text-white
                      "
                      @click.prevent.stop="onClickRegisterScores(i)"
                      >採点結果の入力
                    </a>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </template>
        <template v-else>
          <tr>
            <td colspan="4">
              <div class="text-center">登録済みの講義が存在しません</div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts">
import Vue, { PropType } from 'vue'
import { ClassInfo } from '~/types/courses'

type DataType = {
  openDropdownIdx: number | null
}

export default Vue.extend({
  props: {
    classes: {
      type: Array as PropType<ClassInfo[]>,
      default: () => [],
    },
    selectedClassIdx: {
      type: Number as PropType<number | null>,
      default: null,
    },
  },
  data(): DataType {
    return {
      openDropdownIdx: null,
    }
  },
  computed: {
    isShowSearchResult(): boolean {
      return this.classes.length > 0
    },
  },
  beforeMount() {
    document.addEventListener('click', this.closeDropdown)
  },
  beforeDestroy() {
    document.removeEventListener('click', this.closeDropdown)
  },
  methods: {
    onClickClassDropdown(classIdx: number): void {
      if (this.openDropdownIdx !== null && this.openDropdownIdx === classIdx) {
        this.closeDropdown()
        return
      }
      this.openDropdownIdx = classIdx
    },
    onClickDownloadSubmissions(classIdx: number): void {
      this.$emit('downloadSubmissions', classIdx)
      this.closeDropdown()
    },
    onClickRegisterScores(classIdx: number): void {
      this.$emit('registerScores', classIdx)
      this.closeDropdown()
    },
    closeDropdown() {
      this.openDropdownIdx = null
    },
  },
})
</script>
