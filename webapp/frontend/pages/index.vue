<template>
  <div class="h-screen items-center">
    <div class="py-10 px-8 bg-white shadow-lg xl:w-1/3">
      <img
        src="/image/hero_logo_green.svg"
        alt="login logo"
        class="w-96 mx-auto"
      />
      <form
        class="grid grid-cols-3 place-content-center gap-y-2"
        @submit.prevent="onSubmitLogin"
      >
        <TextInput
          id="login-code"
          v-model="code"
          class="col-span-3"
          label="学内コード"
          type="text"
          placeholder="学内コード"
          :required="true"
          :invalid="loginError"
          invalid-text="学内コードまたはパスワードに間違いがあります。"
        />
        <TextInput
          id="login-password"
          v-model="password"
          class="col-span-3"
          label="パスワード"
          type="password"
          placeholder="********"
          autocomplete="current-password"
          :required="true"
          :invalid="loginError"
          invalid-text="学内コードまたはパスワードに間違いがあります。"
        />

        <Button type="submit" color="primary" class="mt-4 col-start-2"
          >ログイン
        </Button>
      </form>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'
import type { AxiosError } from 'axios'
import axios from 'axios'
import { notify } from '~/helpers/notification_helper'
import Button from '~/components/common/Button.vue'
import TextField from '~/components/common/TextField.vue'
import { User } from '~/types/courses'

export default Vue.extend({
  components: { TextInput: TextField, Button },
  layout: 'empty',
  async middleware(context: Context) {
    try {
      const res = await context.$axios.get<User>(`/api/users/me`)
      const { isAdmin } = res.data
      if (isAdmin) {
        return context.redirect('/teacher/courses')
      } else {
        return context.redirect('/mypage')
      }
    } catch (e) {
      if (
        axios.isAxiosError(e) &&
        (e as AxiosError)?.response?.status === 401
      ) {
        return
      }
      console.error(e)
      notify('ログイン周りの処理に失敗しました')
    }
  },
  data() {
    return {
      code: '',
      password: '',
      loginError: false,
    }
  },
  head: {
    title: 'ISUCHOLAR - ログイン',
  },
  methods: {
    async onSubmitLogin() {
      try {
        await this.$axios.post('/login', {
          code: this.code,
          password: this.password,
        })
        const res = await this.$axios.get(`/api/users/me`)
        const user: User = res.data
        if (user.isAdmin) {
          return this.$router.push('/teacher/courses')
        } else {
          return this.$router.push('/mypage')
        }
      } catch (e) {
        this.loginError = true
        notify('学内コードまたはパスワードが誤っています')
      }
    },
  },
})
</script>
