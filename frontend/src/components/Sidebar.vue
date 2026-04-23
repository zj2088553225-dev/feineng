<!-- src/components/Sidebar.vue -->
<template>
  <a-layout-sider
      v-model:collapsed="collapsed"
      :width="256"
      collapsible
      class="sidebar"
  >
    <a-menu
        v-model:openKeys="openKeys"
        v-model:selectedKeys="selectedKeys"
        mode="inline"
        theme="dark"
        :inline-collapsed="collapsed"
        @click="handleMenuClick"
    >
      <template v-for="item in menuItems" :key="item.key">
        <template v-if="!item.children">
          <a-menu-item :key="item.key">
            <component :is="item.icon" />
            <span>{{ item.label }}</span>
          </a-menu-item>
        </template>
        <template v-else>
          <a-sub-menu :key="item.key">
            <template #title>
              <span>
                <component :is="item.icon" />
                <span>{{ item.label }}</span>
              </span>
            </template>
            <template v-for="child in item.children" :key="child.key">
              <a-menu-item v-if="!child.children" :key="child.key">
                {{ child.label }}
              </a-menu-item>
              <sub-menu v-else :menu-info="child" />
            </template>
          </a-sub-menu>
        </template>
      </template>
    </a-menu>
  </a-layout-sider>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useStore } from '@/stores/userStore'
import {
  PieChartOutlined,
  DesktopOutlined,
  InboxOutlined,
  UserOutlined,
  LoginOutlined,
  HomeOutlined,
  MailOutlined,
  AppstoreOutlined,
} from '@ant-design/icons-vue'

// 递归子菜单组件
const SubMenu = {
  name: 'SubMenu',
  props: ['menuInfo'],
  template: `
    <a-sub-menu :key="menuInfo.key">
      <template #title>{{ menuInfo.label }}</template>
      <template v-for="item in menuInfo.children" :key="item.key">
        <a-menu-item v-if="!item.children" :key="item.key">
          {{ item.label }}
        </a-menu-item>
        <sub-menu v-else :menu-info="item" />
      </template>
    </a-sub-menu>
  `,
}

const router = useRouter()
const route = useRoute() // 获取当前路由
const userStore = useStore()

// 🔐 所有原始菜单（含权限控制）
const allMenuItems = [
  {
    key: 'home',
    icon: HomeOutlined,
    label: '首页',
    path: '/', // 显式添加 path 映射
    roles: ['admin', 'user']
  },
  {
    key: 'user_center',
    icon: UserOutlined,
    label: '合伙人中心',
    roles: ['admin', 'user'],
    children: [
      { key: 'my_home', label: '个人信息', path: '/user_center/my_home', roles: [ 'user'] },
      { key: 'my_product', label: '产品中心', path: '/user_center/my_product', roles: ['admin', 'user'] },
      { key: 'my_order', label: 'Jumia订单', path: '/user_center/my_order', roles: ['admin', 'user'] },
      { key: 'my_transaction', label: 'Jumia订单明细', path: '/user_center/my_transaction', roles: ['admin', 'user'] },
      { key: 'my_customize_order', label: '独立站订单审核明细', path: '/user_center/my_customize_order', roles: ['admin', 'user'] },
      { key: 'my_wuliu_order', label: '头程国际物流明细', path: '/user_center/my_wuliu_order', roles: ['admin', 'user'] },
      { key: 'my_settlement', label: '结算明细', path: '/user_center/my_settlement', roles: ['admin', 'user'] }
    ]
  },
  {
    key: 'user_manage',
    icon: PieChartOutlined,
    label: '合伙人管理',
    roles: ['admin'],
    children: [
      { key: 'user_list', label: '合伙人列表', path: '/user_manage/user_list', roles: ['admin'] },
      { key: 'user_product', label: '合伙人产品中心', path: '/user_manage/user_product', roles: ['admin'] },
      { key: 'user_order', label: '合伙人Jumia订单', path: '/user_manage/user_order', roles: ['admin'] },
      { key: 'user_transaction', label: '合伙人Jumia订单明细', path: '/user_manage/user_transaction', roles: ['admin'] },
      { key: 'user_customize_order', label: '合伙人独立站订单审核明细', path: '/user_manage/user_customize_order', roles: ['admin'] },
      { key: 'user_wuliu_order', label: '合伙人头程国际物流明细', path: '/user_manage/user_wuliu_order', roles: ['admin'] },
      { key: 'user_settlement', label: '合伙人结算明细', path: '/user_manage/user_settlement', roles: ['admin'] },
      { key: 'user_settlement_config', label: '合伙人结算配置', path: '/user_manage/user_settlement_config', roles: ['admin'] }
    ]
  },
  {
    key: 'logout',
    icon: LoginOutlined,
    label: '退出登录',
    roles: ['admin', 'user'],
  },
]

// 路由路径到菜单 key 的映射表
const routeToMenuKey = {}
// 递归构建映射
const buildRouteMap = (items) => {
  items.forEach(item => {
    if (item.path) {
      routeToMenuKey[item.path] = item.key
    }
    if (item.children) {
      buildRouteMap(item.children)
    }
  })
}
buildRouteMap(allMenuItems)

// 🔑 动态计算：根据用户角色过滤菜单
const menuItems = computed(() => {
  const userRole = userStore.userInfo.role || 'guest'

  const filterMenu = (items) => {
    return items
        .filter(item => item.roles.includes(userRole))
        .map(item => {
          if (item.children) {
            return {
              ...item,
              children: filterMenu(item.children)
            }
          }
          return item
        })
        .filter(item => {
          if (item.children) {
            return item.children.length > 0
          }
          return true
        })
  }

  return filterMenu(allMenuItems)
})

// 状态
const collapsed = ref(false)
const selectedKeys = ref([]) // ✅ 移除默认 ['home']
const openKeys = ref([])
const preOpenKeys = ref([])

// 初始化菜单选中和展开状态
const initMenuKeys = () => {
  const path = route.path
  const menuKey = routeToMenuKey[path]

  if (menuKey) {
    selectedKeys.value = [menuKey]

    // 查找父级 key（用于展开）
    let parentKey = null
    for (const item of allMenuItems) {
      if (item.children?.some(child => child.key === menuKey)) {
        parentKey = item.key
        break
      }
    }
    if (parentKey) {
      openKeys.value = [parentKey]
    } else {
      openKeys.value = []
    }
  } else {
    selectedKeys.value = []
    openKeys.value = []
  }
}

// 组件挂载后初始化
onMounted(() => {
  initMenuKeys()
})

// 路由变化时更新菜单
watch(
    () => route.path,
    () => {
      initMenuKeys()
    }
)

// 监听 openKeys 变化（保持 ant-design 逻辑）
watch(
    openKeys,
    (newKeys) => {
      const latestOpenKey = newKeys.find((key) => !preOpenKeys.value.includes(key))
      if (latestOpenKey && !allMenuItems.find((item) => item.key === latestOpenKey)?.children) {
        openKeys.value = preOpenKeys.value
      } else {
        preOpenKeys.value = newKeys
      }
    },
    { immediate: true }
)

// 菜单点击跳转
const handleMenuClick = ({ key }) => {
  if (key !== 'logout') {
    router.push({ name: key })
  } else {
    userStore.logout()
    router.push({ name: 'login' })
  }
}
</script>

<style scoped>
.sidebar {
  height: 100vh;
  overflow: auto;
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 999;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
}

.sidebar .ant-menu {
  border-right: none;
  height: calc(100vh - 64px);
}
</style>