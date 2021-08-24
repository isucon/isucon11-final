type LinkData = {
  path: string
  query: URLSearchParams
}

export type Link = {
  prev: LinkData | undefined
  next: LinkData | undefined
}

export function parseLinkHeader(linkHeader: string | undefined): Link {
  const parsedLink: Link = { prev: undefined, next: undefined }
  if (!linkHeader) {
    return parsedLink
  }

  const linkData = linkHeader.split(',')
  for (const link of linkData) {
    const linkInfo = /<([^>]+)>;\s+rel="([^"]+)"/gi.exec(link)
    if (linkInfo && (linkInfo[2] === 'prev' || linkInfo[2] === 'next')) {
      const u = new URL(linkInfo[1], 'http://localhost:3000')
      parsedLink[linkInfo[2]] = { path: u.pathname, query: u.searchParams }
    }
  }
  return parsedLink
}
