<template>
  <div class="h-screen items-center">
    <div class="py-10 px-8 bg-white shadow-lg w-1/3">
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
import Button from '~/components/common/Button.vue'
import TextField from '~/components/common/TextField.vue'

export default Vue.extend({
  components: { TextInput: TextField, Button },
  middleware({ app, redirect }) {
    if (app.$cookies.get('session')) {
      return redirect('mypage')
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
        await this.$router.push('mypage')
      } catch (e) {
        // TODO: 通知を出すなど適切に処理する
        console.error(e)
      }
    },
  },
})
</script>
