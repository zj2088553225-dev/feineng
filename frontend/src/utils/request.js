// src/utils/request.js
import axios from 'axios'
import { useStore } from '@/stores/userStore.js' // 确保路径正确
// 测试环境
// 创建 axios 实例
// const request = axios.create({
//     baseURL: 'http://localhost:8080/api', // 请根据实际后端地址修改
//     timeout: 100000, // 请求超时时间（毫秒）
// })
// 生产环境
// const request = axios.create({
//     baseURL: 'https://hunanyunchi.com/api',
//     timeout: 5000000,
// })
// 本地开发
const request = axios.create({
    baseURL: 'http://localhost:8080/api',
    timeout: 100000,
})
// http://yunchi.wukong.wales/login

// 请求拦截器：在每次请求前自动添加 token
request.interceptors.request.use(
    (config) => {
        const store = useStore()

        // ✅ 动态获取 store，避免初始化问题
        // ✅ 检查 token 是否存在
        if (store?.userInfo?.token) {
            // 设置自定义 header（根据后端要求）
            config.headers['token'] = store.userInfo.token

            // 🔔 如果后端使用标准 Authorization，建议改为：
            // config.headers['Authorization'] = `Bearer ${store.userInfo.token}`
        }

        return config
    },
    (error) => {
        // 处理请求发送前的错误
        console.error('[Request Error]', error)
        return Promise.reject(error)
    }
)

// 响应拦截器：统一处理响应和错误
request.interceptors.response.use(
    (response) => {
        // 可在此统一处理响应数据，例如检查 code
        const { code, msg } = response.data || {}

        // 例如：如果后端返回 code !== 200 表示业务错误
        // if (code !== 200) {
        //   // 可使用 message 提示错误
        //   // import { message } from 'ant-design-vue'
        //   // message.error(msg || '请求失败')
        // }

        return response // 返回完整的响应对象
    },
    (error) => {
        // 统一处理响应错误
        const status = error.response?.status

        if (status === 401) {
            // token 过期或无效，自动登出
            const store = useStore()
            store.logout() // 清除状态并跳转登录页
            console.warn('登录已过期，正在退出...')
        } else if (status === 403) {
            console.warn('权限不足')
        } else if (!error.response) {
            console.error('网络错误或服务器无响应')
        } else {
            console.error(`[HTTP ${status}] 请求失败`, error.message)
        }

        return Promise.reject(error)
    }
)

export default request