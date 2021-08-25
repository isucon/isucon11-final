export function urlSearchParamsToObject(
  params: URLSearchParams
): Record<string, string> {
  let o: Record<string, string> = {}
  params.forEach((v, k) => {
    o[k] = v
  })
  return o
}
