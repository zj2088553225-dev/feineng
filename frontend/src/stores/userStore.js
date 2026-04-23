// src/stores/index.js
import { defineStore } from 'pinia'
import { message } from 'ant-design-vue'
import * as jwtDecode from 'jwt-decode'

export const useStore = defineStore('jumia', {
    state: () => ({
        userInfo: {
            token: '',
            user_name: '',
            role: "user", // 0=未登录, 1=管理员, 2=普通用户
            user_id: 0,
        },
    }),

    actions: {
        // 设置用户信息（登录后调用）
        setUserInfo(info) {
            const { token } = info
            try {
                const decoded = jwtDecode.jwtDecode(token) // 注意：jwtDecode.jwtDecode

                // ✅ 将 role 数值映射为字符串
                let roleStr = 'user' // 默认为 user
                if (decoded.role === 1) {
                    roleStr = 'admin'
                } else if (decoded.role === 2) {
                    roleStr = 'user'
                }

                // 假设 JWT payload 结构如下（根据你的 Go 后端 jwts.JwtPayLoad）
                // { user_name: "admin", role: 1, user_id: 1001, exp: ..., iat: ... }
                this.userInfo = {
                    token,
                    user_name: decoded.user_name,
                    role: roleStr ,
                    user_id: decoded.user_id || 0,
                }

                // 持久化保存到 localStorage
                localStorage.setItem('userInfo', JSON.stringify(this.userInfo))
                message.success(`欢迎回来，${decoded.user_name}！`)
            } catch (error) {
                message.error('登录凭证无效')
                console.error('JWT 解码失败:', error)
            }
        },

        loadUserInfo() {
            const saved = localStorage.getItem('userInfo')
            if (!saved) return false

            try {
                const parsed = JSON.parse(saved)
                const token = parsed.token

                // 1. 解码 token
                const decoded = jwtDecode.jwtDecode(token)
                const now = Date.now() / 1000

                // 2. 检查是否过期
                if (decoded.exp < now) {
                    this.logout()
                    return false
                }

                // 3. ✅ 重新映射 role（关键！不能相信 localStorage 的 role）
                let roleStr = 'user'
                if (decoded.role === 1) {
                    roleStr = 'admin'
                } else if (decoded.role === 2) {
                    roleStr = 'user'
                }

                // 4. ✅ 用解码后的数据重建 userInfo（防止 localStorage 被篡改）
                this.userInfo = {
                    token: token,
                    user_name: decoded.user_name,
                    role: roleStr,
                    user_id: decoded.user_id || 0,
                }

                // 5. ✅ 可选：刷新 localStorage（确保本地存储也是最新的）
                localStorage.setItem('userInfo', JSON.stringify(this.userInfo))

                return true
            } catch (e) {
                console.error('恢复登录状态失败:', e)
                this.logout()
                return false
            }
        },

        // 登出
        logout() {
            this.userInfo = {
                token: '',
                user_name: '',
                role: "user",
                user_id: 0,
            }
            localStorage.removeItem('userInfo')
            message.info('您已退出登录')
        },
    },
})