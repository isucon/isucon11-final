import { AxiosStatic } from 'axios'

declare module '@nuxt/vue-app' {
  interface Context {
    $axios: AxiosStatic
  }

  interface NuxtAppOptions {
    $axios: AxiosStatic
  }
}

// Nuxt 2.9+
declare module '@nuxt/types' {
  interface Context {
    $axios: AxiosStatic
  }

  interface NuxtAppOptions {
    $axios: AxiosStatic
  }
}

declare module 'vue/types/vue' {
  interface Vue {
    $axios: AxiosStatic
  }
}
