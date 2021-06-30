import { NuxtCookies } from 'cookie-universal-nuxt';

declare module '@nuxt/types' {
  interface Context {
    $cookies: NuxtCookies;
  }
}
