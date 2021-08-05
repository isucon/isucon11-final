import camelCaseKeys from 'camelcase-keys'
import snakecaseKeys from 'snakecase-keys'

export default function ({ $axios }) {
  $axios.onRequest((request) => {
    if (request.params) {
      request.params = snakecaseKeys(request.params)
    }
    if (request.data) {
      request.data = snakecaseKeys(request.data)
    }
  })

  $axios.onResponse((response) => {
    response.data = camelCaseKeys(response.data, {
      deep: true,
    })
  })
}
