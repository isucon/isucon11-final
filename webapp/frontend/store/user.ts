export type State = {
  name: string
}
export const state = (): State => ({
  name: ''
})

export const mutations = {
  set(state: State, name: string): void {
    state.name = name
  }
}
