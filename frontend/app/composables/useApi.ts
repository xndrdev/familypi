const TOKEN_KEY = 'familydisplay.token'

function readToken(): string {
  try {
    return localStorage.getItem(TOKEN_KEY) ?? ''
  } catch {
    return ''
  }
}

export function useApi() {
  const token = useState('api-token', readToken)

  function setToken(value: string) {
    token.value = value
    try {
      localStorage.setItem(TOKEN_KEY, value)
    } catch {
      // Ohne localStorage gilt das Token nur fuer diese Sitzung.
    }
  }

  function request<T>(path: string, opts: { method?: 'GET' | 'POST' | 'PUT' | 'DELETE', body?: unknown } = {}) {
    return $fetch<T>(path, {
      method: opts.method ?? 'GET',
      body: opts.body as Record<string, unknown> | undefined,
      headers: token.value ? { Authorization: `Bearer ${token.value}` } : {},
    })
  }

  return { token, setToken, request }
}

// errorMessage holt die deutsche Fehlermeldung aus der API-Antwort.
export function errorMessage(e: unknown): string {
  const data = (e as { data?: { error?: string } })?.data
  return data?.error ?? (e instanceof Error ? e.message : 'Unbekannter Fehler')
}

// useLiveUpdates ruft onChange auf, sobald das Backend eine Aenderung meldet (gebuendelt).
export function useLiveUpdates(onChange: () => void) {
  let source: EventSource | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  const connected = ref(false)

  const trigger = () => {
    clearTimeout(timer)
    timer = setTimeout(onChange, 250)
  }

  onMounted(() => {
    source = new EventSource('/api/stream')
    source.addEventListener('hello', () => {
      // Nach einem Reconnect koennen Aenderungen verpasst worden sein.
      connected.value = true
      trigger()
    })
    source.addEventListener('change', trigger)
    source.onerror = () => { connected.value = false }
  })
  onBeforeUnmount(() => {
    source?.close()
    clearTimeout(timer)
  })
  return { connected }
}
