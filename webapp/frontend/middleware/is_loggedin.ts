import { Middleware } from '@nuxt/types'

const isLoggedIn: Middleware = async (context) => {
  if (!context.$cookies.get('session')) {
    return context.redirect('/')
  }
}

export default isLoggedIn
