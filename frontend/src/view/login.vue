<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { UserOutlined, LockOutlined } from '@ant-design/icons-vue'
import {useStore} from "@/stores/userStore.js";
import {message} from "ant-design-vue";
import request from "@/utils/request.js";

const router = useRouter()

// 表单数据
const form = ref({
  username: '',
  password: '',
  remember: true
})

// 表单验证
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const formRef = ref()
const store = useStore()
// 登录提交
const handleSubmit = async () => {
  try {
    const res = await request.post('/user/login', {
      user_name: form.value.username,
      password: form.value.password,
    })

    // 假设后端返回格式：{ code: 200, data: "token...", msg: "操作成功" }
    if (res.data.code === 200) {
      // ✅ 登录成功，保存 token 并解析
      store.setUserInfo({ token: res.data.data })

      // 跳转首页
      await router.push('/')
    } else {
      message.error(res.data.msg || '登录失败')
    }
  } catch (err) {
    const msg = err.response?.data?.msg || '登录失败，请检查用户名或密码'
    message.error(msg)
    console.error('登录请求异常:', err)
  }
}
</script>

<template>
  <!-- 全屏容器，居中 -->
  <div class="login-container">
    <!-- 登录表单卡片 -->
    <div class="login-card">
      <!-- 标题 -->
      <div class="login-header">
        <h1>后台管理系统</h1>
        <h2>欢迎登录</h2>
      </div>

      <!-- 表单 -->
      <a-form
          ref="formRef"
          :model="form"
          :rules="rules"
          @finish="handleSubmit"
          layout="vertical"
      >
        <!-- 用户名 -->
        <a-form-item name="username">
          <a-input
              v-model:value="form.username"
              placeholder="请输入用户名"
              size="large"
              style="height: 50px"
          >
            <template #prefix>
              <UserOutlined style="color: #1890ff" />
            </template>
          </a-input>
        </a-form-item>

        <!-- 密码 -->
        <a-form-item name="password">
          <a-input-password
              v-model:value="form.password"
              placeholder="请输入密码"
              size="large"
              style="height: 50px"
          >
            <template #prefix>
              <LockOutlined style="color: #1890ff" />
            </template>
          </a-input-password>
        </a-form-item>



        <!-- 登录按钮 -->
        <a-form-item>
          <a-button
              type="primary"
              html-type="submit"
              block
              size="large"
              :loading="false"
              style="height: 50px; font-size: 16px"
          >
            登 录
          </a-button>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
/* 全屏容器：居中对齐 */
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f0f2f5;
  padding: 20px;
  box-sizing: border-box;
  font-family: 'Segoe UI', 'Microsoft YaHei', Arial, sans-serif;
}

/* 登录卡片：响应式宽度 */
.login-card {
  width: 100%;
  max-width: 480px; /* 最大宽度 480px */
  padding: 50px 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  box-sizing: border-box;
}

.login-header h2 {
  margin: 0 0 8px 0;
  font-size: 20px;
  font-weight: 500;
  color: #333;
  text-align: center;
  margin-bottom: 20px;
}

.login-header h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 600;
  color: #333;
  text-align: center;
  margin-bottom: 20px;
}

/* 表单项间距 */
:deep(.ant-form-item) {
  margin-bottom: 20px;
}

/* 记住我 + 忘记密码 */
.form-extra {
  display: flex;
  justify-content: space-between;
  font-size: 14px;
  margin-bottom: 24px;
}

/* 忘记密码链接 */
.form-extra a {
  color: #1890ff;
}

/* 按钮悬停动效 */
:deep(.ant-btn-primary:hover) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.25);
}
</style>