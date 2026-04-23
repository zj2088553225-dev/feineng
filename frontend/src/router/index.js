import { createRouter, createWebHistory } from 'vue-router'
import { useStore } from '@/stores/userStore.js'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      meta: { hideSidebar: true },
      component: () => import('../view/login.vue')
    },
    {
      path: '/',
      name: 'home',
      component: () => import('../view/home.vue')
    },
    {
      path: '/user_center',
      name: 'user_center',
      redirect: '/user_center/my_product',
      // ✅ 移除这里 meta.requiresAuth
      children: [
        {
          path: 'my_home',
          name: 'my_home',
          component: () => import('../view/user_center/my_home.vue'),
          meta: { requiresAuth: true } // ✅ 每个子路由单独设置
        },
        {
          path: 'my_product',
          name: 'my_product',
          component: () => import('../view/user_center/my_product.vue'),
          meta: { requiresAuth: true } // ✅ 每个子路由单独设置
        },
        {
          path: 'my_order',
          name: 'my_order',
          component: () => import('../view/user_center/my_order.vue'),
          meta: { requiresAuth: true }
        },
        {
          path: 'my_transaction',
          name: 'my_transaction',
          component: () => import('../view/user_center/my_transaction.vue'),
          meta: { requiresAuth: true }
        }, {
          path: 'my_customize_order',
          name: 'my_customize_order',
          component: () => import('../view/user_center/my_customize_order.vue'),
          meta: { requiresAuth: true }
        },
        {
          path: 'my_wuliu_order',
          name: 'my_wuliu_order',
          component: () => import('../view/user_center/my_wuliu_order.vue'),
          meta: { requiresAuth: true }
        },
        {
          path: 'my_settlement',
          name: 'my_settlement',
          component: () => import('../view/user_center/my_settlement.vue'),
          meta: { requiresAuth: true }
        }
      ]
    },
    {
      path: '/user_manage',
      name: 'user_manage',
      redirect: '/user_manage/user_info',
      children: [
        {
          path: 'user_list',
          name: 'user_list',
          component: () => import('../view/user_manage/user_list.vue'),
          meta: { requiresAuth: true }
        },
        {
          path: 'user_product',
          name: 'user_product',
          component: () => import('../view/user_manage/user_product.vue'),
          meta: { requiresAuth: true }
        },
        {
          path: 'user_order',
          name: 'user_order',
          component: () => import('../view/user_manage/user_order.vue'),
          meta: { requiresAuth: true }
        },
        {
          path: 'user_transaction',
          name: 'user_transaction',
          component: () => import('../view/user_manage/user_transaction.vue'),
          meta: { requiresAuth: true }
        },{
          path: 'user_customize_order',
          name: 'user_customize_order',
          component: () => import('../view/user_manage/user_customize_order.vue'),
          meta: { requiresAuth: true }
        },{
          path: 'user_wuliu_order',
          name: 'user_wuliu_order',
          component: () => import('../view/user_manage/user_wuliu_order.vue'),
          meta: { requiresAuth: true }
        },{
          path: 'user_settlement',
          name: 'user_settlement',
          component: () => import('../view/user_manage/user_settlement.vue'),
          meta: { requiresAuth: true }
        },{
          path: 'user_settlement_config',
          name: 'user_settlement_config',
          component: () => import('../view/user_manage/user_settlement_config.vue'),
          meta: { requiresAuth: true }
        }
      ]
    }
  ]
})

// ✅ 全局前置守卫
router.beforeEach(async (to, from, next) => {
  const store = useStore()

  // ✅ 确保用户信息已加载（同步从 localStorage 读取或异步请求）
  await store.loadUserInfo() // 如果是异步，请用 await；如果是同步，可去掉 await

  const hasToken = !!store.userInfo?.token

  if (to.meta.requiresAuth) {
    if (hasToken) {
      next() // 已登录，放行
    } else {
      next('/login') // 未登录，跳转登录
    }
  } else {
    // 不需要认证的页面（如首页、登录页）
    if (to.path === '/login') {
      if (hasToken) {
        // 已登录用户访问登录页，跳回首页
        next('/')
      } else {
        next() // 未登录，允许访问登录页
      }
    } else {
      next() // 其他非保护页面（如 /）直接放行
    }
  }
})

export default router