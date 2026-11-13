import { useSearchParams, type Location, type To } from 'react-router-dom'

const LABEL_PARAM = 'tag'
const FILTERABLE_PATHS = ['/board', '/list']
const FALLBACK_PATH = '/list'

export function useLabelFilter() {
  const [searchParams, setSearchParams] = useSearchParams()
  const label = searchParams.get(LABEL_PARAM)
  function setLabel(value: string | null) {
    const next = new URLSearchParams(searchParams)
    if (value) next.set(LABEL_PARAM, value)
    else next.delete(LABEL_PARAM)
    setSearchParams(next, { replace: true })
  }
  return [label, setLabel] as const
}

export function filtersInPlace(location: Location) {
  return FILTERABLE_PATHS.includes(location.pathname)
}

export function activeLabel(location: Location) {
  return filtersInPlace(location) ? new URLSearchParams(location.search).get(LABEL_PARAM) : null
}

export function labelFilterLink(location: Location, label: string | null): To {
  const inPlace = filtersInPlace(location)
  const params = new URLSearchParams(inPlace ? location.search : '')
  if (label) params.set(LABEL_PARAM, label)
  else params.delete(LABEL_PARAM)
  return { pathname: inPlace ? location.pathname : FALLBACK_PATH, search: params.toString() }
}
