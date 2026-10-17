<template>
  <div v-auto-animate class="w-full flex-1 flex flex-col gap-4 sm:gap-6">
    <!-- 页头标题 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2 sm:pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3 w-full min-w-0 transition-all duration-200">
      <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight shrink-0">授权管理</h1>

      <!-- 右侧刷新状态按钮 -->
      <n-button size="small" secondary :loading="refreshing" @click="handleManualRefresh"
        class="!h-8 !px-3 rounded-lg text-xs font-medium transition-transform duration-150 active:scale-95">
        <template #icon>
          <n-icon :size="15">
            <Refresh />
          </n-icon>
        </template>
        刷新
      </n-button>
    </div>

    <!-- 主授权卡片网格 -->
    <div class="flex flex-col gap-4 sm:gap-6 w-full">
      <!-- 1. DeepSeek 官方账号授权核心卡片 -->
      <n-card :bordered="false"
        class="shadow-sm group rounded-2xl overflow-hidden hover:shadow-md transition-all duration-200">
        <div class="flex flex-col gap-4">
          <!-- 卡片头部：标识与状态徽章 -->
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-4">
            <div class="flex items-start sm:items-center gap-2.5 sm:gap-3.5 min-w-0 flex-1">
              <!-- 图标容器 -->
              <div
                class="w-10 h-10 rounded-xl bg-slate-100 dark:bg-white/[0.06] group-hover:bg-blue-50 dark:group-hover:bg-blue-950/40 text-slate-500 dark:text-slate-400 group-hover:text-fnos-blue dark:group-hover:text-blue-400 flex items-center justify-center transition-all duration-200 group-hover:scale-105 shrink-0 mt-0.5 sm:mt-0">
                <n-avatar v-if="isAccountLoggedIn && accountStatus.avatar_url" round :size="32"
                  :src="accountStatus.avatar_url" />
                <n-icon v-else :size="20">
                  <BrandNytimes />
                </n-icon>
              </div>

              <div class="min-w-0 flex-1 space-y-1">
                <div class="flex items-center gap-2 flex-wrap min-w-0">
                  <span class="text-sm sm:text-base font-semibold text-slate-800 dark:text-slate-100 leading-tight">
                    DeepSeek 官方账号
                  </span>
                  <n-tag v-if="isAccountLoggedIn" type="success" size="small" round :bordered="false"
                    class="font-medium text-xs">
                    已授权
                  </n-tag>
                  <n-tag v-else type="warning" size="small" round :bordered="false" class="font-medium text-xs">
                    未登录
                  </n-tag>
                </div>
                <p class="text-xs text-slate-400 dark:text-slate-500 truncate">
                  {{ isAccountLoggedIn && accountStatus.name ? `已绑定用户：${accountStatus.name} (${accountStatus.contact ||
                    '已验证'})` : '绑定 DeepSeek 官方账号，同步账号信息与调用配额' }}
                </p>
              </div>
            </div>

            <!-- 操作按钮组 -->
            <div class="flex items-center gap-2 shrink-0">
              <n-button v-if="!isAccountLoggedIn" size="medium" type="primary" secondary @click="openLoginModal"
                class="w-full sm:w-auto !h-9 sm:!h-10 px-5 font-medium rounded-xl text-sm transition-transform duration-150 active:scale-95">
                登录授权
              </n-button>
              <template v-else>
                <n-button size="medium" secondary @click="openSwitchAccountModal"
                  class="!h-9 sm:!h-10 px-4 rounded-xl text-xs sm:text-sm font-medium transition-transform duration-150 active:scale-95">
                  切换账号
                </n-button>
                <n-button size="medium" quaternary type="error" :loading="loggingOutAccount"
                  @click="handleAccountLogout"
                  class="!h-9 sm:!h-10 px-3 rounded-xl text-xs sm:text-sm font-medium transition-transform duration-150 active:scale-95">
                  退出登录
                </n-button>
              </template>
            </div>
          </div>

          <!-- 已登录态：账户额度与官方直达快捷指标 -->
          <div v-if="isAccountLoggedIn"
            class="grid grid-cols-2 sm:grid-cols-4 gap-2 sm:gap-3 text-center pt-2 border-t border-slate-100 dark:border-white/[0.06]">
            <!-- 账户余额 -->
            <div
              class="py-2.5 sm:py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col justify-center items-center gap-0.5">
              <span class="text-xs text-slate-400 dark:text-slate-500">充值余额</span>
              <div class="text-base sm:text-lg font-bold text-slate-700 dark:text-slate-200 font-mono">
                {{ accountStatus.balance ? `${accountStatus.currency === 'USD' ? '$' : '¥'}
                ${formatAmount(accountStatus.balance)}` :
                  '-'
                }}
              </div>
            </div>

            <!-- 赠送额度 -->
            <div
              class="py-2.5 sm:py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col justify-center items-center gap-0.5">
              <span class="text-xs text-slate-400 dark:text-slate-500">赠送额度</span>
              <div class="text-base sm:text-lg font-bold text-slate-700 dark:text-slate-200 font-mono">
                {{ accountStatus.bonus_balance ? `${accountStatus.currency === 'USD' ? '$' : '¥'}
                ${formatAmount(accountStatus.bonus_balance)}` : '-' }}
              </div>
            </div>

            <!-- 消耗用量直达 -->
            <div
              class="py-2.5 sm:py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col justify-center items-center gap-0.5">
              <span class="text-xs text-slate-400 dark:text-slate-500">用量中心</span>
              <n-button text type="primary" size="small" class="font-medium mt-0.5"
                @click="openExternal(accountStatus.usage_url || 'https://platform.deepseek.com/usage')">
                查看明细 ↗
              </n-button>
            </div>

            <!-- 余额充值直达 -->
            <div
              class="py-2.5 sm:py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col justify-center items-center gap-0.5">
              <span class="text-xs text-slate-400 dark:text-slate-500">官方充值</span>
              <n-button text type="primary" size="small" class="font-medium mt-0.5"
                @click="openExternal(accountStatus.top_up_url || 'https://platform.deepseek.com/top_up')">
                立即充值 ↗
              </n-button>
            </div>
          </div>
        </div>
      </n-card>

      <!-- 2. 飞牛官方技能命令行授权卡片 -->
      <n-card :bordered="false"
        class="shadow-sm group rounded-2xl overflow-hidden hover:shadow-md transition-all duration-200">
        <div class="flex flex-col gap-4">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-4">
            <div class="flex items-start sm:items-center gap-2.5 sm:gap-3.5 min-w-0 flex-1">
              <!-- 图标容器 -->
              <div
                class="w-10 h-10 rounded-xl bg-slate-100 dark:bg-white/[0.06] group-hover:bg-blue-50 dark:group-hover:bg-blue-950/40 text-slate-500 dark:text-slate-400 group-hover:text-fnos-blue dark:group-hover:text-blue-400 flex items-center justify-center transition-all duration-200 group-hover:scale-105 shrink-0 mt-0.5 sm:mt-0">
                <n-icon :size="20">
                  <Terminal2 />
                </n-icon>
              </div>

              <div class="min-w-0 flex-1 space-y-1">
                <div class="flex items-center gap-2 flex-wrap min-w-0">
                  <span class="text-sm sm:text-base font-semibold text-slate-800 dark:text-slate-100 leading-tight">
                    飞牛官方技能 (fnOS CLI)
                  </span>
                  <n-tag v-if="!isSkillEnabled" type="default" size="small" round :bordered="false"
                    class="font-medium text-xs text-slate-500 bg-slate-100 dark:bg-slate-800">
                    未启用技能
                  </n-tag>
                  <n-tag v-else-if="skillAuthStatus?.authorized" type="success" size="small" round :bordered="false"
                    class="font-medium text-xs">
                    已授权 {{ skillAuthStatus.username ? `(${skillAuthStatus.username})` : '' }}
                  </n-tag>
                  <n-tag v-else type="warning" size="small" round :bordered="false" class="font-medium text-xs">
                    未授权
                  </n-tag>
                </div>
                <p class="text-xs text-slate-400 dark:text-slate-500 truncate">
                  {{ !isSkillEnabled ? '未在应用设置中开启飞牛官方技能，开启后可进行账号授权' : '向 DSH 提供 NAS 文件管理、相册索引、磁盘监控等本地操作能力' }}
                </p>
              </div>
            </div>

            <!-- 操作按钮组 -->
            <div class="flex items-center gap-2 shrink-0">
              <n-button v-if="!isSkillEnabled" size="medium" secondary @click="goToSettings"
                class="w-full sm:w-auto !h-9 sm:!h-10 px-4 font-medium rounded-xl text-sm transition-transform duration-150 active:scale-95">
                <template #icon>
                  <n-icon :size="16">
                    <Settings />
                  </n-icon>
                </template>
                前往开启
              </n-button>
              <n-button v-else-if="!skillAuthStatus?.authorized" size="medium" type="primary" secondary
                @click="openSkillAuthModal"
                class="w-full sm:w-auto !h-9 sm:!h-10 px-5 font-medium rounded-xl text-sm transition-transform duration-150 active:scale-95">
                登录授权
              </n-button>
              <n-button v-else size="medium" quaternary type="error" :loading="loggingOutSkill"
                @click="handleSkillLogout"
                class="!h-9 sm:!h-10 px-3 rounded-xl text-xs sm:text-sm font-medium transition-transform duration-150 active:scale-95">
                解除授权
              </n-button>
            </div>
          </div>
        </div>
      </n-card>
    </div>

    <!-- 切换 DeepSeek 账号弹窗 -->
    <n-modal v-model:show="showSwitchModal" preset="card" title="切换 DeepSeek 账号"
      class="w-full max-w-md rounded-2xl shadow-xl" :mask-closable="!switchingAccountId">
      <div class="space-y-4 py-1">
        <!-- 账号列表容器 -->
        <n-spin :show="loadingPool">
          <div v-if="accountPool.length === 0" class="py-8 text-center text-xs text-slate-400">
            暂无已保存的历史账号凭据
          </div>
          <div v-else class="space-y-2.5 max-h-80 overflow-y-auto pr-0.5">
            <div v-for="item in accountPool" :key="item.id"
              class="p-3 rounded-xl border transition-all duration-150 flex items-center justify-between gap-3"
              :class="item.is_active
                ? 'bg-blue-50/70 dark:bg-blue-950/30 border-blue-200/80 dark:border-blue-900/60 shadow-xs'
                : 'bg-slate-50 dark:bg-slate-800/60 border-slate-100 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700'">
              
              <!-- 账号信息展示 -->
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <n-avatar v-if="item.avatar_url" round :size="36" :src="item.avatar_url" class="shrink-0" />
                <div v-else class="w-9 h-9 rounded-full bg-slate-200 dark:bg-slate-700 flex items-center justify-center shrink-0 text-slate-500 dark:text-slate-400">
                  <n-icon :size="18"><User /></n-icon>
                </div>
                <div class="min-w-0 flex-1 space-y-0.5">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    <span class="text-xs sm:text-sm font-semibold text-slate-800 dark:text-slate-100 truncate">
                      {{ item.name || 'DeepSeek 用户' }}
                    </span>
                    <n-tag v-if="item.is_active" type="success" size="tiny" round :bordered="false" class="font-medium text-[10px]">
                      当前使用
                    </n-tag>
                  </div>
                  <p class="text-[11px] text-slate-400 dark:text-slate-500 truncate">
                    {{ item.contact || '已验证' }}
                    <span v-if="item.balance" class="ml-1 text-slate-600 dark:text-slate-300 font-mono">
                      (¥{{ formatAmount(item.balance) }})
                    </span>
                  </p>
                </div>
              </div>

              <!-- 切换与删除操作 -->
              <div class="flex items-center gap-1.5 shrink-0">
                <n-button v-if="!item.is_active" size="tiny" type="primary" secondary
                  :loading="switchingAccountId === item.id"
                  :disabled="!!switchingAccountId"
                  @click="handleSwitchAccount(item.id, item.name || item.contact)"
                  class="rounded-lg !h-7 !px-2.5 text-xs font-medium">
                  切换
                </n-button>
                <n-popconfirm @positive-click="handleDeleteAccount(item.id)">
                  <template #trigger>
                    <n-button size="tiny" quaternary type="error" :disabled="!!switchingAccountId"
                      class="rounded-lg !h-7 !w-7 !p-0">
                      <template #icon>
                        <n-icon :size="14"><Trash /></n-icon>
                      </template>
                    </n-button>
                  </template>
                  确认从本机移除此账号凭据？下次切换需重新授权。
                </n-popconfirm>
              </div>
            </div>
          </div>
        </n-spin>

        <!-- 底部添加新账号 -->
        <div class="pt-2 border-t border-slate-100 dark:border-white/[0.06] flex items-center justify-between gap-2">
          <n-button block type="primary" dashed @click="handleAddAccountFromSwitch"
            class="!h-9 rounded-xl text-xs font-medium">
            <template #icon>
              <n-icon :size="15"><Plus /></n-icon>
            </template>
            添加新账号
          </n-button>
        </div>
      </div>
    </n-modal>

    <!-- DeepSeek 账号授权弹窗 -->
    <n-modal v-model:show="showLoginModal" preset="card" title="DeepSeek 账号授权"
      class="w-full max-w-lg rounded-2xl shadow-xl" :mask-closable="!startingLogin && !confirmingLogin">
      <div class="space-y-4 py-1">
        <!-- 步骤 1：前往官方授权页面 -->
        <div
          class="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800 space-y-2.5">
          <div class="text-xs font-medium text-slate-700 dark:text-slate-300">
            步骤一：前往官方授权页面
          </div>
          <n-button type="primary" block :loading="startingLogin" @click="handleStartLogin"
            class="!h-10 rounded-xl font-medium shadow-xs shadow-blue-500/20">
            <template #icon>
              <n-icon>
                <ExternalLink />
              </n-icon>
            </template>
            {{ loginAuthUrl ? '重新打开授权页面' : '前往授权页面' }}
          </n-button>
        </div>

        <!-- 步骤 2：填入回调网址 -->
        <div
          class="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800 space-y-2.5">
          <div class="text-xs font-medium text-slate-700 dark:text-slate-300">
            步骤二：填入回调地址
          </div>

          <p class="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
            在官方页面完成授权后，浏览器将跳转至回调地址。请复制地址栏中的完整链接，粘贴至下方输入框并点击确认绑定。
          </p>

          <!-- 粘贴输入框 -->
          <n-input v-model:value="callbackUrlInput" type="text"
            placeholder="http://127.0.0.1:2298/oauth/callback?code=xxx" clearable :disabled="confirmingLogin" autofocus
            @keydown.enter="handleConfirmLogin" class="rounded-xl font-mono text-xs" />
        </div>

        <div class="flex justify-end gap-2 pt-1">
          <n-button :disabled="confirmingLogin" @click="showLoginModal = false">取消</n-button>
          <n-button type="primary" :loading="confirmingLogin" :disabled="!callbackUrlInput.trim()"
            @click="handleConfirmLogin">
            确认绑定
          </n-button>
        </div>
      </div>
    </n-modal>

    <!-- 飞牛技能授权弹窗 -->
    <n-modal v-model:show="showSkillModal" :auto-focus="false" preset="card" title="飞牛官方技能授权"
      class="w-full max-w-lg rounded-2xl shadow-xl" :mask-closable="!startingSkillAuth && !confirmingSkillAuth">
      <div class="space-y-4 py-1">
        <!-- 访问端口 (可选) -->
        <div
          class="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800 space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-slate-700 dark:text-slate-300">访问端口 (可选)</span>
            <span class="text-[11px] text-slate-400">如系统使用非默认端口，可在此指定</span>
          </div>
          <n-input v-model:value="skillAuthPort" type="text" placeholder="默认端口" clearable
            :disabled="startingSkillAuth || confirmingSkillAuth" class="rounded-xl text-xs" />
        </div>

        <!-- 步骤 1：前往官方授权页面 -->
        <div
          class="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800 space-y-2.5">
          <div class="text-xs font-medium text-slate-700 dark:text-slate-300">
            步骤一：前往官方授权页面
          </div>
          <n-button type="primary" block :loading="startingSkillAuth" @click="handleOpenSkillAuthPage"
            class="!h-10 rounded-xl font-medium shadow-xs shadow-blue-500/20">
            <template #icon>
              <n-icon>
                <ExternalLink />
              </n-icon>
            </template>
            前往授权页面
          </n-button>
        </div>

        <!-- 步骤 2：填入授权代码 -->
        <div
          class="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800 space-y-2.5">
          <div class="text-xs font-medium text-slate-700 dark:text-slate-300">
            步骤二：填入授权代码
          </div>
          <p class="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
            在官方页面完成登录后，复制页面中展示的授权代码，粘贴至下方输入框并点击确认绑定。
          </p>
          <n-input v-model:value="skillAuthCode" type="text" placeholder="在此粘贴获取到的授权代码" clearable
            :disabled="confirmingSkillAuth" @keydown.enter="handleConfirmSkillAuth"
            class="rounded-xl font-mono text-xs" />
        </div>

        <div class="flex justify-end gap-2 pt-1">
          <n-button :disabled="confirmingSkillAuth" @click="showSkillModal = false">取消</n-button>
          <n-button type="primary" :loading="confirmingSkillAuth" :disabled="!skillAuthCode.trim()"
            @click="handleConfirmSkillAuth">
            确认绑定
          </n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated } from 'vue'
import {
  NCard,
  NButton,
  NTag,
  NIcon,
  NModal,
  NInput,
  NAvatar,
  NSpin,
  NPopconfirm,
  useMessage
} from 'naive-ui'
import {
  BrandNytimes,
  Terminal2,
  ExternalLink,
  Refresh,
  Settings,
  Plus,
  Trash,
  User
} from '@vicons/tabler'
import { accountApi, skillAuthApi } from '../api'
import { useAuthStore } from '../stores/auth'
import { useConfigStore } from '../stores/config'
import { useAppStore } from '../stores/app'
import { trimSdk } from '../utils/trimSdk'

const message = useMessage()
const authStore = useAuthStore()
const configStore = useConfigStore()
const appStore = useAppStore()

const isSkillEnabled = computed(() => configStore.config.enable_builtin_skill !== false)

function goToSettings() {
  appStore.setTab('settings')
}

const refreshing = computed(() => authStore.refreshing)
const accountStatus = computed(() => authStore.accountStatus)
const isAccountLoggedIn = computed(() => authStore.isAccountLoggedIn)
const skillAuthStatus = computed(() => configStore.skillAuthStatus)

// 账号池相关状态
const accountPool = computed(() => authStore.accountPool)
const loadingPool = computed(() => authStore.loadingPool)
const switchingAccountId = computed(() => authStore.switchingAccountId)
const showSwitchModal = ref(false)

// 打开多账号切换弹窗
function openSwitchAccountModal() {
  showSwitchModal.value = true
  authStore.fetchAccountPool(false)
}

// 切换至目标账号
async function handleSwitchAccount(id: string, name?: string) {
  const ok = await authStore.switchAccount(id)
  if (ok) {
    message.success(`已切换至账号 ${name || ''}`)
    showSwitchModal.value = false
  } else {
    message.error('切换账号失败，请重试')
  }
}

// 从本机移除历史账号凭据
async function handleDeleteAccount(id: string) {
  const ok = await authStore.deleteAccount(id)
  if (ok) {
    message.success('已从本机移除该账号凭据')
  } else {
    message.error('移除账号凭据失败')
  }
}

// 从切换弹窗跳转到添加新账号
function handleAddAccountFromSwitch() {
  showSwitchModal.value = false
  openLoginModal()
}

// DeepSeek 登录相关状态
const showLoginModal = ref(false)
const startingLogin = ref(false)
const confirmingLogin = ref(false)
const loggingOutAccount = ref(false)
const loginAuthUrl = ref('')
const callbackUrlInput = ref('')

// 飞牛技能授权相关状态
const showSkillModal = ref(false)
const startingSkillAuth = ref(false)
const confirmingSkillAuth = ref(false)
const loggingOutSkill = ref(false)
const skillAuthCode = ref('')
const skillAuthPort = ref('')

// 手动刷新账号状态
async function handleManualRefresh() {
  const [statusOk, poolOk] = await Promise.all([
    authStore.fetchAccountStatus(false),
    authStore.fetchAccountPool(true)
  ])
  if (statusOk || poolOk) {
    message.success('账号状态已更新')
  } else {
    message.error('获取账号状态失败')
  }
}

// 格式化金额显示（保留两位小数）
function formatAmount(val?: string) {
  if (!val) return '-'
  const num = Number(val)
  if (isNaN(num)) return val
  return num.toFixed(2)
}

// 打开外部链接
function openExternal(url: string) {
  if (url) trimSdk.openURL(url, '_blank')
}

// 打开 DeepSeek 登录弹窗
function openLoginModal() {
  callbackUrlInput.value = ''
  loginAuthUrl.value = ''
  showLoginModal.value = true
}

// 发起 DeepSeek 官方登录
async function handleStartLogin() {
  startingLogin.value = true
  try {
    const res = await accountApi.startLogin()
    if (res.success && res.data?.authorize_url) {
      loginAuthUrl.value = res.data.authorize_url
      trimSdk.openURL(res.data.authorize_url, '_blank')
      message.info('已在新窗口打开 DeepSeek 授权页')
    } else {
      message.error(res.message || '发起登录失败')
    }
  } catch (err: any) {
    message.error(err?.message || '发起登录异常')
  } finally {
    startingLogin.value = false
  }
}

// 确认并提交回调绑定
async function submitCallback(urlText: string) {
  if (!urlText || !urlText.trim()) return
  confirmingLogin.value = true
  try {
    const res = await accountApi.confirmLogin({ url: urlText.trim() })
    if (res.success) {
      message.success('DeepSeek 账号授权绑定成功')
      showLoginModal.value = false
      callbackUrlInput.value = ''
      await Promise.all([
        authStore.fetchAccountStatus(true),
        authStore.fetchAccountPool(true)
      ])
    } else {
      message.error(res.message || '授权绑定失败')
    }
  } catch (err: any) {
    message.error(err?.message || '授权绑定请求失败')
  } finally {
    confirmingLogin.value = false
  }
}

// 确认绑定
function handleConfirmLogin() {
  submitCallback(callbackUrlInput.value)
}

// DeepSeek 退出登录
async function handleAccountLogout() {
  loggingOutAccount.value = true
  try {
    const res = await accountApi.logout()
    if (res.success) {
      message.success('已退出 DeepSeek 账号')
      authStore.resetAccountStatus()
      await Promise.all([
        authStore.fetchAccountStatus(true),
        authStore.fetchAccountPool(true)
      ])
    } else {
      message.error(res.message || '退出登录失败')
    }
  } catch (err: any) {
    message.error(err?.message || '退出登录异常')
  } finally {
    loggingOutAccount.value = false
  }
}

// 打开飞牛技能授权弹窗
async function openSkillAuthModal() {
  if (!isSkillEnabled.value) {
    message.warning('请先在应用设置中开启飞牛官方内置技能')
    return
  }
  skillAuthPort.value = ''
  skillAuthCode.value = ''
  showSkillModal.value = true
}

// 打开飞牛技能授权页面
async function handleOpenSkillAuthPage() {
  startingSkillAuth.value = true
  try {
    const res = await skillAuthApi.startAuth(skillAuthPort.value.trim())
    if (res.success && res.data?.url) {
      let targetUrl = res.data.url
      if (targetUrl.includes('/signin?')) {
        targetUrl = targetUrl.slice(targetUrl.indexOf('/signin?'))
      }
      trimSdk.openURL(targetUrl, '_blank')
    } else {
      message.error(res.message || '获取授权链接失败')
    }
  } catch (err: any) {
    message.error(err?.message || '获取授权链接异常')
  } finally {
    startingSkillAuth.value = false
  }
}

// 确认飞牛技能授权
async function handleConfirmSkillAuth() {
  if (!skillAuthCode.value.trim()) return
  confirmingSkillAuth.value = true
  try {
    const res = await skillAuthApi.confirmAuth(skillAuthCode.value.trim())
    if (res.success) {
      message.success('飞牛技能授权绑定成功')
      showSkillModal.value = false
      skillAuthCode.value = ''
      configStore.updateSkillAuthStatus({ authorized: true })
    } else {
      message.error(res.message || '技能授权验证失败')
    }
  } catch (err: any) {
    message.error(err?.message || '技能授权验证异常')
  } finally {
    confirmingSkillAuth.value = false
  }
}

// 解除飞牛技能授权
async function handleSkillLogout() {
  loggingOutSkill.value = true
  try {
    const res = await skillAuthApi.logout()
    if (res.success) {
      message.success('已解除技能授权')
      configStore.updateSkillAuthStatus({ authorized: false })
    } else {
      message.error(res.message || '解除授权失败')
    }
  } catch (err: any) {
    message.error(err?.message || '解除授权异常')
  } finally {
    loggingOutSkill.value = false
  }
}

onMounted(() => {
  authStore.fetchAccountStatus(true)
  authStore.fetchAccountPool(true)
  configStore.fetchConfig()
})

onActivated(() => {
  authStore.fetchAccountStatus(true)
  authStore.fetchAccountPool(true)
  configStore.fetchConfig()
})
</script>
