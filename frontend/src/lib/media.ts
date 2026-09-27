const API_URL = import.meta.env?.VITE_API_URL || '/api/v1'

export function getMediaUrl(path?: string): string {
  if (!path) return ''
  if (/^(https?:|data:|blob:)/i.test(path)) return path

  const apiOrigin = new URL(API_URL, window.location.origin).origin
  return `${apiOrigin}${path.startsWith('/') ? path : `/${path}`}`
}
