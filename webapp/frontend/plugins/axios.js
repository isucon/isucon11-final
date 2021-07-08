import camelCaseKeys from 'camelcase-keys'

export default function ({ $axios }) {
  $axios.onResponse((response) => {
    response.data = camelCaseKeys(response.data)
  })
}
