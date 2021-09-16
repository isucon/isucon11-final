<template>
  <nav
    class="
      flex
      items-center
      justify-between
      flex-wrap
      bg-primary-500
      p-6
      shadow
    "
  >
    <div class="flex-no-shrink text-white">
      <span class="text-xl tracking-tight">
        <NuxtLink to="/"
          ><img
            src="/image/header_logo_white.svg"
            alt="isucholar logo"
            class="h-12"
        /></NuxtLink>
      </span>
    </div>

    <div :class="stateIsLoggedIn" class="text-white flex items-center gap-x-6">
      <template v-if="!isAdmin">
        <NuxtLink to="/register" class="block hover:text-gray-200"
          ><fa-icon icon="pen" class="mr-0.5" />履修登録</NuxtLink
        >
        <NuxtLink to="/grade" class="block hover:text-gray-200">
          <fa-icon icon="graduation-cap" class="mr-0.5" />成績照会</NuxtLink
        >
        <NuxtLink to="/announce" class="block hover:text-gray-200">
          <fa-icon icon="bell" class="mr-0.5" />お知らせ</NuxtLink
        >
      </template>
      <div class="relative">
        <span
          class="pl-2 cursor-pointer hover:text-gray-200"
          @click="onClickDropdown"
          >学内コード: {{ code }}</span
        >
        <div
          class="
            absolute
            right-0
            mt-2
            py-1
            rounded
            z-20
            w-40
            bg-white
            shadow-2xl
          "
          :class="stateDropdown"
        >
          <a
            v-click-outside="onOutsideClickDropdown"
            href="#"
            class="
              block
              px-4
              py-2
              text-gray-800 text-sm
              hover:bg-primary-300 hover:text-white
            "
            @click="onClickLogout"
            >ログアウト</a
          >
        </div>
      </div>
    </div>
  </nav>
</template>
<script lang="ts">
import Vue from 'vue'
import axios, { AxiosError } from 'axios'
// @ts-ignore
import ClickOutside from 'vue-click-outside'
import { notify } from '~/helpers/notification_helper'

type Data = {
  code: string
  isAdmin: boolean
  isOpenDropdown: boolean
}

export default Vue.extend({
  directives: {
    ClickOutside,
  },
  data(): Data {
    return {
      code: '',
      isAdmin: false,
      isOpenDropdown: false,
    }
  },
  async fetch() {
    try {
      const res = await this.$axios.get('/api/users/me')
      const me = res.data
      this.code = me.code
      this.isAdmin = me.isAdmin
    } catch (e) {
      if (
        axios.isAxiosError(e) &&
        (e as AxiosError)?.response?.status === 401
      ) {
        return
      }
      console.error(e)
      notify('自身の情報の取得に失敗しました')
    }
  },
  computed: {
    stateIsLoggedIn(): string {
      return this.code !== '' ? 'show' : 'hidden'
    },
    stateDropdown(): string {
      return this.isOpenDropdown ? 'show' : 'hidden'
    },
  },
  methods: {
    onOutsideClickDropdown(e: Event) {
      e.stopPropagation()
      this.isOpenDropdown = false
    },
    onClickDropdown(e: Event) {
      e.stopPropagation()
      this.isOpenDropdown = !this.isOpenDropdown
    },
    async onClickLogout(event: Event) {
      event.preventDefault()
      await this.$axios.post('/logout')
      this.code = ''
      this.isAdmin = false
      await this.$router.push('/')
    },
  },
})
</script>
