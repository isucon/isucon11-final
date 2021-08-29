import { Middleware } from '@nuxt/types'
import axios, {AxiosError} from "axios";

const isStudent: Middleware = async (context) => {
  try {
    const res = await context.$axios.get('/api/users/me')
    if (res.status > 199 && res.status < 300) {
      const { isAdmin } = res.data
      if (!isAdmin) {
        return
      }
    }

    return context.redirect('/')
  } catch (e) {
    if (axios.isAxiosError(e) && (e as AxiosError)?.response?.status === 401) {
      return context.redirect('/')
    }
    console.error(e)
    return context.redirect('/')
  }
}

export default isStudent
