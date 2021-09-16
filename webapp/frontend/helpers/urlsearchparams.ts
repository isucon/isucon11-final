export function urlSearchParamsToObject(
  params: URLSearchParams,
  omits: string[] = []
): Record<string, string> {
  let o: Record<string, string> = {}
  params.forEach((v, k) => {
    if (!omits.includes(k)) {
      o[k] = v
    }
  })
  return o
}
