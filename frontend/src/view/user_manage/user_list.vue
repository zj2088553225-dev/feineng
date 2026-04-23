<template>
  <a-card style="padding: 10px" title="合伙人列表页">
    <!-- 搜索区域 -->
    <a-form layout="inline" :model="searchForm" style="margin-bottom: 16px">
      <a-form-item label="用户名">
        <a-input v-model:value="searchForm.user_name" placeholder="输入用户名" @change="handleSearch" style="width: 140px" />
      </a-form-item>
      <a-form-item label="权限">
        <a-input v-model:value="searchForm.role" placeholder="输入权限" @change="handleSearch" style="width: 120px" />
      </a-form-item>
      <a-form-item label="Seller SKU">
        <a-input v-model:value="searchForm.seller_sku" placeholder="输入Seller SKU" @change="handleSearch" style="width: 180px" />
      </a-form-item>
      <a-form-item>
        <a-space>
          <a-button @click="resetSearch">重置查询条件</a-button>
        </a-space>
      </a-form-item>
      <!-- 在 <a-button type="primary">创建新用户</a-button> 后面添加弹窗 -->
      <a-button type="primary" @click="showCreateModal = true">创建新用户</a-button>

      <a-button type="primary" style="margin-left: 10px" @click="showCooperationModal = true">设置合营合伙人</a-button>
    </a-form>
<!-- 编辑合营合伙人弹窗   -->
    <a-modal
        v-model:open="showEditCoopModal"
        title="编辑合营合伙人"
        :okText="'保存'"
        :cancelText="'取消'"
        :confirm-loading="editCoopLoading"
        @ok="handleEditCoop"
    >
      <a-form layout="vertical" :model="editCoopForm">
        <a-form-item label="合营比例">
          <a-input-number v-model:value="editCoopForm.rate" :min="0" :max="1" :step="0.01" />
        </a-form-item>
        <a-form-item label="备注">
          <a-input v-model:value="editCoopForm.note" placeholder="可选" />
        </a-form-item>
      </a-form>
    </a-modal>


    <!-- 合营合伙人管理弹窗 -->
    <a-modal
        v-model:open="showCooperationModal"
        title="合营合伙人管理"
        :width="650"
        :footer="null"
    >
      <!-- 当前合营合伙人 -->
      <div style="margin-bottom: 24px">
        <div style="font-weight: 500; margin-bottom: 8px">当前合营合伙人：</div>
        <div style="display: flex; flex-wrap: wrap; gap: 8px;">
          <a-tag
              v-for="item in cooperationPartners"
              :key="item.id"
              closable
              @close="handleDeleteCooperation(item)"
              style="display: flex; align-items: center;"
          >
        <span>
          {{ item.user_name }} ({{ (item.rate * 100).toFixed(0) }}%)
          <span v-if="item.note">- {{ item.note }}</span>
        </span>
            <a @click.stop="showEditCoopModalFn(item)" style="margin-left: 8px; color:#1890ff">
              编辑
            </a>
          </a-tag>
        </div>
      </div>

      <!-- 添加合营合伙人表单 -->
      <a-form layout="inline" :model="addCooperationForm" style="gap: 16px; flex-wrap: wrap;">
        <a-form-item label="选择用户">
          <a-select
              v-model:value="addCooperationForm.user_id"
              style="width: 220px"
              placeholder="选择用户"
          >
            <a-select-option
                v-for="user in upUserIDOptions"
                :key="user.value"
                :value="user.value"
            >
              {{ user.label }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="合营比例">
          <a-input-number
              v-model:value="addCooperationForm.rate"
              :min="0"
              :max="1"
              :step="0.01"
              style="width: 120px;"
          />
        </a-form-item>

        <a-form-item label="设置备注">
          <a-input
              v-model:value="addCooperationForm.note"
              style="width: 220px"
              placeholder="可选"
          />
        </a-form-item>

        <a-form-item style="margin-top: 4px;">
          <a-button type="primary" @click="handleAddCooperation">添加</a-button>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 表格 -->
    <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        row-key="id"
    >
      <!-- 使用 #bodyCell 渲染 seller_skus 列 -->
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'seller_skus'">
          <template v-if="record.seller_skus && record.seller_skus.length > 0">
            <a-tag
                v-for="item in record.seller_skus"
                :key="item.seller_sku"
                color="geekblue"
                style="margin-bottom: 4px; display: inline-block"
            >
              {{ item.seller_sku }}
            </a-tag>
          </template>
          <span v-else class="text-muted">—</span>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a @click="() => handleEdit(record)">编辑</a>
            <a @click="() => handleDelete(record)" style="color: #ff4d4f; font-weight: 500;">删除</a>
          </a-space>
        </template>
      </template>
    </a-table>

    <!-- ========== 编辑用户弹窗 ========== -->
    <a-modal v-model:open="editModalVisible" title="编辑用户" @ok="submitEdit" :confirm-loading="editLoading"  :okText="'确定'"
             :cancelText="'取消'">
      <a-form :model="editForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="用户名">
          <a-input v-model:value="editForm.user_name" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="editForm.pass_word" placeholder="请输入新密码" />
        </a-form-item>
        <a-form-item label="权限">
          <a-select v-model:value="editForm.role">
            <a-select-option :value="1">管理员</a-select-option>
            <a-select-option :value="2">用户</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>


    <!-- ========== 创建用户弹窗 ========== -->
    <a-modal
        v-model:open="showCreateModal"
        title="创建新用户"
        @ok="handleCreateUser"
        :confirm-loading="createLoading"
        :okText="'确定'"
        :cancelText="'取消'"
    >
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="用户名" required>
          <a-input v-model:value="createForm.user_name" placeholder="请输入用户名" />
        </a-form-item>
        <a-form-item label="密码" required>
          <a-input-password v-model:value="createForm.password" placeholder="请输入密码" />
        </a-form-item>
        <a-form-item label="权限" required>
          <a-select v-model:value="createForm.role">
            <a-select-option :value="1">管理员</a-select-option>
            <a-select-option :value="2">用户</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </a-card>
</template>

<script setup>
import { ExclamationCircleOutlined } from '@ant-design/icons-vue';
import {ref, onMounted, reactive, createVNode, watch} from 'vue';
import {message, Modal} from 'ant-design-vue';
import request from '@/utils/request';
const showCooperationModal = ref(false);

// 当前合营合伙人列表
const cooperationPartners = ref([]);

// 下拉选择可绑定用户
const upUserIDOptions = ref([]);

// 新增合营合伙人表单
const addCooperationForm = ref({
  user_id: null,
  rate: 0.8,
  note: ''
});

// 获取合营合伙人列表
const fetchCooperationPartners = async () => {
  try {
    const res = await request.get('/user/cooperation_partner');
    if (res.data.code === 200) {
      cooperationPartners.value = res.data.data;
    } else {
      message.error(res.data.msg || '获取合营合伙人列表失败');
    }
  } catch (err) {
    message.error('请求失败');
    console.error(err);
  }
};

// 获取用户列表用于下拉
const getbindUser = async () => {
  if (upUserIDOptions.value.length > 0) return; // 避免重复请求
  try {
    const res = await request.get('/user/user_name_list');
    if (res.data.code === 200) {
      upUserIDOptions.value = res.data.data.map(user => ({
        value: user.id,
        label: user.user_name
      }));
    }
  } catch (err) {
    console.error(err);
  }
};

// 添加合营合伙人
const handleAddCooperation = async () => {
  try {
    const { user_id, rate, note } = addCooperationForm.value;
    if (!user_id) {
      message.warning('请选择用户');
      return;
    }
    const res = await request.post('/user/cooperation_partner', { user_id, rate, note });
    if (res.data.code === 200) {
      message.success('添加成功');
      fetchCooperationPartners();
      addCooperationForm.value = { user_id: null, rate: 0.8, note: '' };
    } else {
      message.error(res.data.msg || '添加失败');
    }
  } catch (err) {
    message.error('请求失败');
  }
};

// 删除合营合伙人
const handleDeleteCooperation = async (item) => {
  try {
    const res = await request.delete('/user/cooperation_partner', { data: { id: item.id } });
    if (res.data.code === 200) {
      message.success('删除成功');
      fetchCooperationPartners();
    } else {
      message.error(res.data.msg || '删除失败');
    }
  } catch (err) {
    message.error('请求失败');
  }
};
const showEditCoopModal = ref(false);
const editCoopLoading = ref(false);
const editCoopForm = ref({ id: null, rate: 0.8, note: '' });

// 打开编辑弹窗
const showEditCoopModalFn = (item) => {
  editCoopForm.value = { ...item }; // 拷贝数据
  showEditCoopModal.value = true;
};

// 提交编辑
const handleEditCoop = async () => {
  editCoopLoading.value = true;
  try {
    const { id, rate, note } = editCoopForm.value;
    const res = await request.put('/user/cooperation_partner', { id, rate, note });
    if (res.data.code === 200) {
      message.success('编辑成功');
      showEditCoopModal.value = false;
      fetchCooperationPartners(); // 刷新列表
    } else {
      message.error(res.data.msg || '编辑失败');
    }
  } catch (err) {
    message.error('请求失败');
  } finally {
    editCoopLoading.value = false;
  }
};


// 弹窗打开时加载数据
watch(showCooperationModal, (visible) => {
  if (visible) {
    fetchCooperationPartners();
    getbindUser();
  }
});
// ========== 搜索表单 ==========
const searchForm = reactive({
  user_name: '',
  role: '',
  seller_sku: '',
});

// ========== 表格数据 ==========
const originalData = ref([]);
const tableData = ref([]);
const loading = ref(false);

// ========== 表格列定义 ==========
const columns = [
  { title: '用户ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '用户名', dataIndex: 'user_name', key: 'user_name', width: 120 },
  // ❌ 移除密码列（安全）：你原本加了密码列，但不应该显示
  { title: '密码', dataIndex: 'pass_word', key: 'pass_word', width: 100 },
  { title: '权限', dataIndex: 'role', key: 'role', width: 100 },
  { title: 'Seller SKU', key: 'seller_skus', width: 500  ,ellipsis: true},
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180,
    customRender: ({ value }) => formatTime(value),
  },
  { title: '操作', key: 'action' }, // 添加操作列
];

// 时间格式化
const formatTime = (timeStr) => {
  if (!timeStr) return '';
  const date = new Date(timeStr);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
};

// ========== 获取原始数据 ==========
const fetchData = async () => {
  loading.value = true;
  try {
    const res = await request.get('/user/user_list');

    if (res.data.code === 200) {
      originalData.value = res.data.data.map(user => ({
        ...user,
        seller_skus: Array.isArray(user.seller_skus) ? user.seller_skus : [],
      }));
      tableData.value = [...originalData.value];
      // console.log('第一个用户的第一个 SKU:', res.data.data[0].seller_skus[0].seller_sku);
    } else {
      message.error(res.data.msg || '获取用户列表失败');
    }
  } catch (err) {
    message.error('请求失败');
    console.error('Error fetching user list:', err);
  } finally {
    loading.value = false;
  }
};

// ========== 搜索逻辑 ==========
const handleSearch = () => {
  const { user_name, role, seller_sku } = searchForm;
  tableData.value = originalData.value.filter(user => {
    const matchesName = !user_name || user.user_name.includes(user_name.trim());
    const matchesRole = !role || user.role.toString().includes(role.trim());
    const matchesSku = !seller_sku || user.seller_skus.some(sku => sku.seller_sku.includes(seller_sku.trim()));
    return matchesName && matchesRole && matchesSku;
  });
};

// ========== 重置搜索 ==========
const resetSearch = () => {
  searchForm.user_name = '';
  searchForm.role = '';
  searchForm.seller_sku = '';
  tableData.value = [...originalData.value];
};

// ========== 编辑用户 ==========
const editModalVisible = ref(false);
const editLoading = ref(false);
const editForm = ref({ id: null, user_name: '默认用户名', pass_word: '', role: 2 });

// 角色映射表
const ROLE_MAP = {
  '管理员': 1,
  '用户': 2,
  // 其他角色...
};

// 反向映射（用于显示）
const ROLE_LABEL = {
  1: '管理员',
  2: '用户',
};

const handleEdit = (record) => {
  console.log('原始 record:', record);

  // ✅ 把中文 role 转成数字
  const roleValue = ROLE_MAP[record.role] || 2; // 默认为用户

  editForm.value = {
    ...record,
    role: roleValue,     // ✅ 存数字 1 或 2
  };

  editModalVisible.value = true;
};

const submitEdit = async () => {
  editLoading.value = true;
  try {
    const { id, user_name, pass_word, role } = editForm.value;

    // ✅ role 已经是 1 或 2
    console.log('提交 role:', role); // 应该是 number

    const res = await request.put('/user/create_user', {
      user_id: id,
      user_name,
      password: pass_word,
      role, // 直接传数字
    });

    if (res.data.code === 200) {
      message.success('更新成功');
      editModalVisible.value = false;
      fetchData();
    } else {
      message.error(res.data.msg || '失败');
    }
  } catch (err) {
    message.error('请求失败');
    console.error(err);
  } finally {
    editLoading.value = false;
  }
};




// ========== 创建用户 ==========
const showCreateModal = ref(false);
const createLoading = ref(false);
const createForm = ref({
  user_name: '',
  password: '',
  role: 2, // 默认为管理员
});

const handleCreateUser = async () => {
  const { user_name, password, role } = createForm.value;

  if (!user_name.trim()) {
    message.warning('请输入用户名');
    return;
  }
  if (!password) {
    message.warning('请输入密码');
    return;
  }
  if (![1, 2].includes(role)) {
    message.warning('请选择有效权限');
    return;
  }

  createLoading.value = true;
  try {
    const res = await request.post('/user/create_user', {
      user_name: user_name.trim(),
      password,
      role,
    });

    if (res.data.code === 200) {
      message.success('用户创建成功');
      showCreateModal.value = false;
      // 重置表单
      createForm.value = { user_name: '', password: '', role: 1 };
      // 刷新列表
      fetchData();
    } else {
      message.error(res.data.msg || '创建失败');
    }
  } catch (err) {
    message.error('请求失败');
    console.error(err);
  } finally {
    createLoading.value = false;
  }
};
const handleDelete = (user) => {
  if (!user || !user.id) {
    message.warning('无效的用户信息');
    return;
  }

  // ✅ 手动定义一个 loading 变量
  const modalConfig = {
    title: '确认删除？',
    icon: createVNode(ExclamationCircleOutlined),
    content: `确认删除用户 "${user.user_name}" 吗？删除后无法恢复。`,
    okText: '确认删除',
    okType: 'danger',
    cancelText: '取消',
    // ✅ 直接绑定 confirmLoading
    confirmLoading: true,
    onOk() {
      return new Promise(async (resolve, reject) => {
        try {
          const res = await request.post('/user/delete_user', {
            user_id: user.id,
          });

          if (res.data.code === 200) {
            message.success('用户删除成功');
            fetchData();
            resolve(); // 关闭弹窗
          } else {
            message.error(res.data.msg || '删除失败');
            reject(); // 不关闭弹窗
          }
        } catch (error) {
          message.error('请求失败，请检查网络或重试');
          reject(); // 不关闭弹窗
        } finally {
          // ✅ 手动更新 confirmLoading（通过修改配置项）
          // Ant Design Vue 会自动关闭 loading
        }
      });
    },
    onCancel() {
      console.log('取消删除');
    },
  };

  // ✅ 显示弹窗
  Modal.confirm(modalConfig);
};

// 初始化
onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.text-muted {
  color: #999;
  font-style: italic;
}
</style>