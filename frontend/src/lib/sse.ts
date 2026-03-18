import { ApiRequestError } from '@/api/client'
import type { ApiError } from '@/types'

type MessageHandler<T> = (message: T) => void

interface SubscribeSSEOptions<T> {
  path: string
  onMessage: MessageHandler<T>
  signal?: AbortSignal
}

interface SSEFrame {
  event: string
  data: string
}

function parseFrame(chunk: string): SSEFrame | null {
  const lines = chunk.split(/\r?\n/)
  let event = 'message'
  const data: string[] = []

  for (const line of lines) {
    if (!line || line.startsWith(':')) {
      continue
    }
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
      continue
    }
    if (line.startsWith('data:')) {
      data.push(line.slice(5).trimStart())
    }
  }

  if (data.length === 0) {
    return null
  }

  return {
    event,
    data: data.join('\n'),
  }
}

async function readStream<T>(
  response: Response,
  onMessage: MessageHandler<T>,
  signal?: AbortSignal
) {
  const reader = response.body?.getReader()
  if (!reader) {
    return
  }

  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    if (signal?.aborted) {
      await reader.cancel()
      return
    }

    buffer += decoder.decode(value, { stream: true })
    const frames = buffer.split(/\r?\n\r?\n/)
    buffer = frames.pop() ?? ''

    for (const frame of frames) {
      const parsed = parseFrame(frame)
      if (!parsed || parsed.event === 'ping') {
        continue
      }
      onMessage(JSON.parse(parsed.data) as T)
    }
  }
}

function handleUnauthorized() {
  localStorage.removeItem('auth-token')
  localStorage.removeItem('auth-storage')
  if (window.location.pathname !== '/login' && window.location.pathname !== '/register') {
    window.location.href = '/login'
  }
}

async function connectOnce<T>(
  options: SubscribeSSEOptions<T>,
  signal?: AbortSignal
) {
  const token = localStorage.getItem('auth-token')
  const response = await fetch(`/api/v1/${options.path}`, {
    headers: {
      Accept: 'text/event-stream',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    signal,
  })

  if (!response.ok) {
    if (response.status === 401) {
      handleUnauthorized()
    }

    const body = await response.json().catch(() => null) as { error?: ApiError } | null
    if (body?.error) {
      throw new ApiRequestError(body.error, response.status)
    }
    throw new Error(`SSE request failed with status ${response.status}`)
  }

  await readStream(response, options.onMessage, signal)
}

export function subscribeSSE<T>(options: SubscribeSSEOptions<T>) {
  let stopped = false
  let currentAbort: AbortController | null = null

  const stop = () => {
    stopped = true
    currentAbort?.abort()
  }

  if (options.signal) {
    options.signal.addEventListener('abort', stop, { once: true })
  }

  const run = async () => {
    while (!stopped) {
      currentAbort = new AbortController()
      const abortListener = () => currentAbort?.abort()
      options.signal?.addEventListener('abort', abortListener, { once: true })

      try {
        await connectOnce(options, currentAbort.signal)
      } catch (error) {
        if (stopped || currentAbort.signal.aborted) {
          return
        }
        if (error instanceof ApiRequestError && error.status < 500 && error.status !== 429) {
          return
        }
      } finally {
        options.signal?.removeEventListener('abort', abortListener)
      }

      await new Promise((resolve) => window.setTimeout(resolve, 2000))
    }
  }

  void run()

  return stop
}
