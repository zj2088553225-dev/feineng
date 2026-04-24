<!-- src/App.vue -->
<template>
  <a-layout class="layout">
    <Sidebar v-if="!hideSidebar" v-model:collapsed="collapsed" />

    <a-layout :style="{ marginLeft: sidebarWidth }">
      <a-layout-content class="content">
        <a-alert
          v-if="showKilimallAuthAlert && !hideSidebar"
          class="kilimall-alert"
          type="error"
          banner
          message="Kilimall 授权已过期，请抓取最新 Seller-SID Cookie 和 AccessToken 后更新授权。"
        >
          <template #action>
            <a-button size="small" type="primary" danger @click="openKilimallAuthModal">
              更新 Kilimall 授权
            </a-button>
          </template>
        </a-alert>

        <a-modal
          v-model:open="kilimallAuthModalVisible"
          title="更新 Kilimall 授权"
          ok-text="保存"
          cancel-text="取消"
          :confirm-loading="savingKilimallAuth"
          @ok="saveKilimallAuth"
        >
          <a-form layout="vertical">
            <a-form-item label="Kilimall Seller-SID (Cookie)">
              <a-textarea
                v-model:value="kilimallAuthForm.cookie"
                :rows="4"
                placeholder="请粘贴包含 seller-sid=... 的 Cookie"
                allow-clear
              />
            </a-form-item>
            <a-form-item label="Kilimall AccessToken">
              <a-textarea
                v-model:value="kilimallAuthForm.token"
                :rows="4"
                placeholder="请粘贴请求头 accesstoken 的值"
                allow-clear
              />
            </a-form-item>
          </a-form>
        </a-modal>

        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { useRoute } from 'vue-router'
import Sidebar from './components/Sidebar.vue'
import { useStore } from '@/stores/userStore.js'
import request from '@/utils/request.js'

const route = useRoute()
const collapsed = ref(false)
const showKilimallAuthAlert = ref(false)
const kilimallAuthModalVisible = ref(false)
const savingKilimallAuth = ref(false)
const kilimallAuthForm = reactive({
  cookie: '',
  token: '',
})
const store = useStore()

const hideSidebar = computed(() => route.meta?.hideSidebar === true)

const sidebarWidth = computed(() => {
  if (hideSidebar.value) return '0px'
  return collapsed.value ? '80px' : '256px'
})

const fetchServiceStatus = async () => {
  if (hideSidebar.value || store.userInfo?.role !== 'admin') {
    showKilimallAuthAlert.value = false
    return
  }

  try {
    const res = await request.get('/service')
    if (res.data.code !== 200 || !Array.isArray(res.data.data)) {
      showKilimallAuthAlert.value = false
      return
    }

    const kilimallStatus = res.data.data.find((item) => item?.ID === 13)
    const statusText = String(kilimallStatus?.Status || '').trim().toLowerCase()
    showKilimallAuthAlert.value = statusText === '错误' || statusText === 'error'
  } catch (error) {
    showKilimallAuthAlert.value = false
    console.error(error)
  }
}

const openKilimallAuthModal = () => {
  kilimallAuthForm.cookie = ''
  kilimallAuthForm.token = ''
  kilimallAuthModalVisible.value = true
}

const saveKilimallAuth = async () => {
  const cookie = kilimallAuthForm.cookie.trim()
  const token = kilimallAuthForm.token.trim()
  if (!cookie || !token) {
    message.warning('请填写 Kilimall Seller-SID Cookie 和 AccessToken')
    return
  }

  savingKilimallAuth.value = true
  try {
    const res = await request.post('/system/kilimall-cookie', { cookie, token })
    if (res.data.code !== 200) {
      message.error(res.data.msg || '更新失败')
      return
    }

    message.success('更新成功')
    kilimallAuthModalVisible.value = false
    showKilimallAuthAlert.value = false
    await fetchServiceStatus()
  } catch (error) {
    message.error('更新失败，请检查网络')
    console.error(error)
  } finally {
    savingKilimallAuth.value = false
  }
}

let pollTimer = null

onMounted(async () => {
  store.loadUserInfo()
  await fetchServiceStatus()
  pollTimer = setInterval(async () => {
    await fetchServiceStatus()
    if (!showKilimallAuthAlert.value) clearInterval(pollTimer)
  }, 30000)
})

onUnmounted(() => {
  clearInterval(pollTimer)
})

watch(
  () => route.fullPath,
  async () => {
    await fetchServiceStatus()
  }
)
</script>

<style scoped>
.layout {
  min-height: 100vh;
}

.content {
  background: #f5f5f5;
  overflow: auto;
}

.kilimall-alert {
  margin: 16px 16px 0;
}
</style>
