<template>
  <a-card title="个人信息页" style="padding: 24px">
    <a-form >
      <!-- 用户名 -->
      <!-- 用户名 + 密码 + 修改按钮 在同一行 -->
      <a-form-item label="用户名">
        <div style="display: flex; align-items: center; gap: 16px; flex-wrap: wrap">
          <!-- 用户名输入框 -->
          <a-input
              v-model:value="userInfo.user_name"
              disabled
              style="width: 220px"
          />

          <!-- 密码标签与输入框 -->
          <span style="margin-right: 8px; white-space: nowrap">密码：</span>
          <a-input-password
              v-model:value="userInfo.pass_word"
              :visibility-toggle="true"
              placeholder="••••••••"
              style="width: 220px"
          />

          <!-- 修改按钮 -->
          <a-button
              type="primary"
              @click="onEdit"
              style="margin-left: 24px"
              :loading="loading"
          >
            修改
          </a-button>
        </div>
      </a-form-item>

      <!-- Seller SKU 列表 -->
      <a-form-item label="Seller SKU" style="margin-bottom: 0">
        <!-- 提示文字 -->
        <template #label>
          <span style="font-weight: 500">Seller SKU</span>
          <div style="font-size: 12px; color: #999; margin-top: 4px;">
            共 {{ userInfo.seller_skus?.length || 0 }} 个 SKU
          </div>
        </template>

        <!-- SKU 标签列表 -->
        <div style="min-height: 48px; padding: 8px 0;">
          <a-tag
              v-for="item in userInfo.seller_skus"
              :key="item.seller_sku"
              color="geekblue"
              style="margin: 4px 6px 4px 0; font-size: 14px; padding: 4px 10px"
          >
            {{ item.seller_sku }}
          </a-tag>
          <span v-if="!userInfo.seller_skus || userInfo.seller_skus.length === 0" class="text-muted">
            暂无 SKU
          </span>
        </div>
      </a-form-item>

    </a-form>
  </a-card>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import request from '@/utils/request';

// 单个用户信息
const userInfo = ref({
  user_name: '',
  pass_word: '',
  seller_skus: [],
});

const loading = ref(false);

// 获取我的信息
const fetchMyInfo = async () => {
  loading.value = true;
  try {
    const res = await request.get('/user/my_sku'); // 假设接口返回 { code: 200, data: { user_name, pass_word, skus: [...] } }

    if (res.data.code === 200) {
      const data = res.data.data;

      userInfo.value = {
        user_name: data.user_name,
        pass_word: data.pass_word,
        role: data.role || 'seller',
        created_at: data.created_at || new Date().toISOString(),
        seller_skus: Array.isArray(data.skus)
            ? data.skus.map(sku => ({ seller_sku: sku.seller_sku || sku }))
            : [],
      };
    } else {
      message.error(res.data.msg || '获取信息失败');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error('Error fetching my info:', err);
  } finally {
    loading.value = false;
  }
};
// 修改账户密码
const onEdit = async() => {
  loading.value = true;
  try {
    const res = await request.post('/user/userinfo',{
      password: userInfo.value.pass_word
    });

    if (res.data.code === 200) {
      message.info("更新密码成功")
    } else {
      message.error(res.data.msg || '更新密码失败');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error('Error fetching my info:', err);
  } finally {
    loading.value = false;
  }
}
onMounted(() => {
  fetchMyInfo();
});
</script>

<style scoped>
.text-muted {
  color: #999;
  font-style: italic;
}
</style>