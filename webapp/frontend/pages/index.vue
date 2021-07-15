<template>
  <div class="">
    <div class="py-10 px-8 bg-white shadow-lg">
      <form
        class="flex-1 flex-col w-full max-w-sm"
        @submit.prevent="onSubmitLogin"
      >
        <TextInput
          id="inline-name"
          v-model="name"
          class="mb-2"
          label="学籍番号"
          type="text"
          placeholder="学籍番号"
        />
        <TextInput
          id="inline-password"
          v-model="password"
          label="パスワード"
          type="password"
          placeholder="********"
        />

        <Button type="submit" color="primary">ログイン</Button>
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
    console.log(app.$cookies.get('session'), localStorage.getItem('user'))
    if (app.$cookies.get('session') && localStorage.getItem('user')) {
      return redirect('mypage')
    }
  },
  data() {
    return {
      name: '',
      password: '',
    }
  },
  methods: {
    async onSubmitLogin() {
      await this.$axios.post('/login', {
        name: this.name,
        password: this.password,
      })
      localStorage.setItem('user', this.name)
      await this.$router.push('mypage')
    },
  },
})
</script>
