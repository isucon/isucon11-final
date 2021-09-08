import { Plugin } from '@nuxt/types'
import Axios from 'axios'
import snakecaseKeys from 'snakecase-keys'
import camelCaseKeys from 'camelcase-keys'

const axios: Plugin = (context, inject) => {
  Axios.interceptors.request.use((request) => {
    if (request.data instanceof FormData) {
      return request
    }

    if (request.params) {
      request.params = snakecaseKeys(request.params)
    }
    if (request.data) {
      request.data = snakecaseKeys(request.data)
    }

    return request
  })

  Axios.interceptors.response.use((response) => {
    if (
      !response.headers['content-type'] ||
      !response.headers['content-type'].includes('application/json')
    ) {
      return response
    }

    response.data = camelCaseKeys(response.data, {
      deep: true,
    })

    return response
  }, err => {
    err.response.data = camelCaseKeys(err.response.data, {
      deep: true,
    })
    return Promise.reject(err)
  })

  context.$axios = Axios
  inject('axios', Axios)
}

export default axios
