import type { PiniaPluginContext } from 'pinia'

export interface PersistOptions {
  key?: string
  paths?: string[]
}

declare module 'pinia' {
  export interface DefineStoreOptionsBase<S, Store> {
    persist?: boolean | PersistOptions
  }
}

const STORAGE_PREFIX = 'dsh_cache_'

// Pinia 全局持久化插件：负责 Store 状态的本地水合与防抖保存
export function piniaPersistPlugin({ store, options }: PiniaPluginContext) {
  if (!options.persist) return

  const persistOpts: PersistOptions = options.persist === true ? {} : options.persist
  const storageKey = persistOpts.key || `${STORAGE_PREFIX}${store.$id}`
  const paths = persistOpts.paths

  // 1. 初始化时从本地存储恢复状态（水合 Hydration）
  try {
    const raw = localStorage.getItem(storageKey)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (parsed && typeof parsed === 'object') {
        const stateToPatch: Record<string, any> = {}
        if (paths && paths.length > 0) {
          paths.forEach((p) => {
            if (p in parsed) {
              stateToPatch[p] = parsed[p]
            }
          })
        } else {
          Object.assign(stateToPatch, parsed)
        }
        store.$patch(stateToPatch)
      }
    }
  } catch (_) {
    // 忽略存储读取异常
  }

  // 2. 状态变动时防抖写入本地存储
  let timer: ReturnType<typeof setTimeout> | null = null
  store.$subscribe(
    (_, state) => {
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => {
        try {
          let dataToSave: any = state
          if (paths && paths.length > 0) {
            dataToSave = {}
            paths.forEach((p) => {
              dataToSave[p] = (state as any)[p]
            })
          }
          localStorage.setItem(storageKey, JSON.stringify(dataToSave))
        } catch (_) {
          // 忽略存储写入限制异常
        }
      }, 100)
    },
    { detached: true }
  )
}
