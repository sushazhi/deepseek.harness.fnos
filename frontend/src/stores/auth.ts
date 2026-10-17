import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { accountApi } from '../api'
import type { DshAccountStatus, AccountPublicInfo } from '../types/api'

// DeepSeek 官方账号状态管理 Store
export const useAuthStore = defineStore(
  'auth',
  () => {
    const accountStatus = ref<DshAccountStatus>({
      status: 'signed-out',
      waiting_sign_in: false
    })

    const accountPool = ref<AccountPublicInfo[]>([])
    const refreshing = ref(false)
    const loadingPool = ref(false)
    const switchingAccountId = ref<string | null>(null)

    const isAccountLoggedIn = computed(() => accountStatus.value.status === 'credential-stored')

    // 更新账号状态
    function setAccountStatus(status: DshAccountStatus) {
      accountStatus.value = status
    }

    // 重置账号状态为未登录
    function resetAccountStatus() {
      accountStatus.value = {
        status: 'signed-out',
        waiting_sign_in: false
      }
    }

    // 从服务端拉取最新账号状态（SWR 静默同步）
    async function fetchAccountStatus(silent = true): Promise<boolean> {
      if (!silent) refreshing.value = true
      try {
        const res = await accountApi.getStatus()
        if (res.success && res.data) {
          accountStatus.value = res.data
          return true
        }
        return false
      } catch (_) {
        return false
      } finally {
        if (!silent) refreshing.value = false
      }
    }

    // 从服务端拉取已保存账号池列表
    async function fetchAccountPool(silent = true): Promise<boolean> {
      if (!silent) loadingPool.value = true
      try {
        const res = await accountApi.getPool()
        if (res.success && res.data) {
          accountPool.value = res.data
          return true
        }
        return false
      } catch (_) {
        return false
      } finally {
        if (!silent) loadingPool.value = false
      }
    }

    // 切换至指定账号
    async function switchAccount(id: string): Promise<boolean> {
      switchingAccountId.value = id
      try {
        const res = await accountApi.switchAccount(id)
        if (res.success) {
          await Promise.all([fetchAccountStatus(true), fetchAccountPool(true)])
          return true
        }
        return false
      } catch (_) {
        return false
      } finally {
        switchingAccountId.value = null
      }
    }

    // 移除指定账号凭据
    async function deleteAccount(id: string): Promise<boolean> {
      try {
        const res = await accountApi.deleteFromPool(id)
        if (res.success) {
          await Promise.all([fetchAccountStatus(true), fetchAccountPool(true)])
          return true
        }
        return false
      } catch (_) {
        return false
      }
    }

    return {
      accountStatus,
      accountPool,
      refreshing,
      loadingPool,
      switchingAccountId,
      isAccountLoggedIn,
      setAccountStatus,
      resetAccountStatus,
      fetchAccountStatus,
      fetchAccountPool,
      switchAccount,
      deleteAccount
    }
  },
  {
    persist: {
      paths: ['accountStatus', 'accountPool']
    }
  }
)

