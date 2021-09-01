<template>
  <div class="h-screen items-center">
    <div class="py-10 px-8 bg-white shadow-lg w-1/3">
      <h1 class="text-center text-2xl mb-6">ISUCHOLAR ログイン</h1>
      <form
        class="grid grid-cols-3 place-content-center gap-y-2"
        @submit.prevent="onSubmitLogin"
      >
        <TextInput
          id="login-code"
          v-model="code"
          class="col-span-3"
          label="学籍番号"
          type="text"
          placeholder="学籍番号"
        />
        <TextInput
          id="login-password"
          v-model="password"
          class="col-span-3"
          label="パスワード"
          type="password"
          placeholder="********"
          autocomplete="current-password"
        />

        <Button type="submit" color="primary" class="mt-4 col-start-2"
          >ログイン</Button
        >
      </form>
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import { Context } from '@nuxt/types'
import axios from 'axios'
import type { AxiosError } from 'axios'
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
      if (res.status > 199 && res.status < 300) {
        const { isAdmin } = res.data
        if (isAdmin) {
          return context.redirect('/teacherpage')
        } else {
          return context.redirect('/mypage')
        }
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
    }
  },
  methods: {
    async onSubmitLogin() {
      try {
        await this.$axios.post('/login', {
          code: this.code,
          password: this.password,
        })
        const user: User = await this.$axios.$get(`/api/users/me`)
        if (user.isAdmin) {
          return this.$router.push('/teacherpage')
        } else {
          return this.$router.push('/mypage')
        }
      } catch (e) {
        notify('学籍番号またはパスワードが誤っています')
      }
    },
  },
})
</script>
