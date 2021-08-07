import camelCaseKeys from 'camelcase-keys'
import snakecaseKeys from 'snakecase-keys'

export default function ({ $axios }) {
  $axios.onRequest((request) => {
    if (request.data instanceof FormData) {
      return
    }

    if (request.params) {
      request.params = snakecaseKeys(request.params)
    }
    if (request.data) {
      request.data = snakecaseKeys(request.data)
    }
  })

  $axios.onResponse((response) => {
    if (
      !response.headers['content-type'] ||
      !response.headers['content-type'].includes('application/json')
    ) {
      return
    }

    response.data = camelCaseKeys(response.data, {
      deep: true,
    })
  })
}
