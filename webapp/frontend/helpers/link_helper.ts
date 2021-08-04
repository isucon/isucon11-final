export type Link = {
  prev: string
  next: string
}

export function parseLinkHeader(linkHeader: string | undefined): Link {
  const parsedLink = { prev: '', next: '' }
  if (!linkHeader) {
    return parsedLink
  }

  const linkData = linkHeader.split(',')
  for (const link of linkData) {
    const linkInfo = /<([^>]+)>;\s+rel="([^"]+)"/gi.exec(link)
    if (linkInfo && (linkInfo[2] === 'prev' || linkInfo[2] === 'next')) {
      const url = new URL(linkInfo[1])
      parsedLink[linkInfo[2]] = `${url.pathname}${url.search}`
    }
  }
  return parsedLink
}
