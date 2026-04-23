<template>
  <a-card style="padding: 10px" title="合伙人结算配置页">
    <!-- 搜索区域 -->
    <a-row :gutter="16" style="margin-bottom: 16px;">
      <!-- 合伙人下拉框 -->
      <a-col :span="2">
        <a-select
            v-model:value="selectedPartnerId"
            :options="upUserIDOptions"
            placeholder="选择合伙人"
            allowClear
            style="width: 100%;"
            @change="handlePartnerChange"
        />
      </a-col>
      <!-- 国家下拉框 -->
      <a-col :span="2">
        <a-select
            v-model:value="selectedCountry"
            :options="countryOptions"
            placeholder="选择国家"
            allowClear
            style="width: 100%;"
            @change="handleCountryChange"
        />
      </a-col>
      <!-- 时间范围选择器 -->
      <a-col :span="6">
        <a-range-picker
            v-model:value="Seleteddate"
            show-time
            format="YYYY-MM-DD"
            @change="onDateChange"
        />
      </a-col>
      <a-col :span="6">
        <a-button
            type="primary"
            style="margin-right: 8px"
            @click="openAddModal"
        >
          新增配置
        </a-button>

      </a-col>
    </a-row>
    <!-- 表格 -->
    <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        row-key="id"
        :pagination="pagination"
        @change="handleTableChange"
        :scroll="{ x: 1300, y: 600 }"
        :key="tableKey"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'action'">
          <span>
            <a @click="EditUserSettlementConfig(record)">编辑配置</a>
            <a style="color: #ff4d4f;margin-left: 10px" @click="DeleteUserSettlementConfig(record)">删除配置</a>

            <!-- 运行按钮 -->
          </span>
          <a-button
              type="primary"
              size="small"
              @click="runSettlementConfig(record)"
              :loading="record.status === '运行中'"
              :disabled="record.status === '运行中'"
              style="margin-left: 10px; height: 24px; padding: 0 8px; font-size: 12px;"
          >
            {{ record.status === '运行中' ? '运行中...' : '运行' }}
          </a-button>
        </template>
      </template>
    </a-table>
    <!-- 编辑配置弹窗 -->
    <a-modal
        v-model:open="modalVisible"
        title="编辑结算配置"
        @ok="handleOk"
        @cancel="handleCancel"
        okText="保存"
        cancelText="取消"
        :maskClosable="false"
        :confirmLoading="loading"
        width="500px"
    >
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" style="margin-top: 20px">
        <a-form-item label="云骑佣金比例" required>
          <a-input-number
              v-model:value="formState.cloud_ride_commission_rate"
              :step="0.01"
              :min="0"
              :max="1"
              :precision="4"
              placeholder="请输入 0~1 之间的比例"
              style="width: 100%"
          />
          <div style="color: #999; font-size: 12px; margin-top: 4px;">
            例如：0.1 表示 10%
          </div>
        </a-form-item>

        <a-form-item label="结算汇率" required>
          <a-input-number
              v-model:value="formState.settlement_rate"
              :step="0.01"
              :min="0.01"
              :precision="4"
              placeholder="请输入结算汇率"
              style="width: 100%"
          />
          <div style="color: #999; font-size: 12px; margin-top: 4px;">
            例如：0.55 表示 1 CNY = 0.55 GHS
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
    <!-- 新增配置弹窗 -->
    <a-modal
        v-model:open="addModalVisible"
        title="新增结算配置"
        @ok="handleAddSubmit"
        @cancel="handleAddCancel"
        okText="保存"
        cancelText="取消"
        :maskClosable="false"
        :confirmLoading="loading"
        width="500px"
    >
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" style="margin-top: 20px">
        <a-form-item label="合伙人" required>
          <a-select
              v-model:value="addFormState.user_id"
              :options="upUserIDOptions"
              placeholder="请选择合伙人"
              style="width: 100%"
          />
        </a-form-item>

        <a-form-item label="国家" required>
          <a-select
              v-model:value="addFormState.country_code"
              :options="countryOptions"
              placeholder="请选择国家"
              style="width: 100%"
          />
        </a-form-item>
        <a-form-item label="结算周期" required>
          <a-week-picker
              v-model:value="tempWeek"
          placeholder="请选择一周"
          @change="onWeekChange"
          style="width: 100%"
          />
          <div style="color: #999; font-size: 12px; margin-top: 4px;">
            选择后将自动设置为该周的周一到周日
          </div>
        </a-form-item>
        <a-form-item label="云骑佣金比例" required>
          <a-input-number
              v-model:value="addFormState.cloud_ride_commission_rate"
              :step="0.01"
              :min="0"
              :max="1"
              :precision="4"
              placeholder="请输入 0~1 之间的比例"
              style="width: 100%"
          />
          <div style="color: #999; font-size: 12px; margin-top: 4px;">
            例如：0.1 表示 10%
          </div>
        </a-form-item>

        <a-form-item label="结算汇率" required>
          <a-input-number
              v-model:value="addFormState.settlement_rate"
              :step="0.01"
              :min="0.01"
              :precision="4"
              placeholder="请输入结算汇率"
              style="width: 100%"
          />
          <div style="color: #999; font-size: 12px; margin-top: 4px;">
            例如：0.55 表示 1 CNY = 0.55 GHS
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </a-card>
</template>

<script setup>
import dayjs from 'dayjs';
import {h, onMounted, onUnmounted, reactive, ref} from "vue";
import request from "@/utils/request.js";
import {message, Modal} from "ant-design-vue";
const columns = [
  {
    title: '合伙人',
    dataIndex: 'user_id',
    key: 'user_id',
    width: 50,
    customRender: ({ text }) => {
      const user = upUserIDOptions.value.find(u => u.value === text);
      return user ? user.label : '默认配置';
    }
  },
  {
    title: '国家',
    dataIndex: 'country_code',
    key: 'country_code',
    width: 50,
    customRender: ({ text }) => ({
      'GH': '加纳',
      'NG': '尼日利亚',
      'KE': '肯尼亚'
    })[text] || text
  },
  {
    title: '结算开始日期',
    dataIndex: 'settlement_start_date',
    key: 'settlement_start_date',
    width: 50,
    customRender: ({ text }) => formatTime(text)
  },
  {
    title: '结算结束日期',
    dataIndex: 'settlement_end_date',
    key: 'settlement_end_date',
    width: 50,
    customRender: ({ text }) => formatTime(text)
  },
  {
    title: '云驰佣金比例',
    dataIndex: 'cloud_ride_commission_rate',
    key: 'cloud_ride_commission_rate',
    width: 50,
    customRender: ({ value }) => `${(value * 100).toFixed(2)}%`
  },
  {
    title: '结算汇率',
    dataIndex: 'settlement_rate',
    key: 'settlement_rate',
    width: 50,
    customRender: ({ value }) => `${(value ).toFixed(6)}`
  },
  {
    title: '操作',
    key: 'action',
    fixed: 'right',
    width: 120,
    scopedSlots: { customRender: 'action' }
  },
  {
    title: '操作时间',
    key: 'updated_at',
    dataIndex: 'updated_at',
    width: 120,
    customRender: ({ text }) => formatTime(text)
  },
  {
    title: '状态',
    key: 'status',
    width: 120,
    customRender: ({ record }) => {
      const status = record.status?.trim();
      const config = SETTLEMENT_STATUS[status] || { text: '未知', color: 'warning' };

      return h('a-tag', {
        props: {
          color: config.color,
          // 可选：加 title 提示
          title: status
        },
        style: {
          fontSize: '12px',
          fontWeight: 500
        }
      }, config.text);
    }
  }

];
// 🔹 提取状态配置（放在 utils/status.js）
const SETTLEMENT_STATUS = {
  '初始化':    { text: '未运行',   color: 'default' },
  '运行中':    { text: '运行中',   color: 'processing' },
  '运行成功':  { text: '已完成',   color: 'success' },
  '运行失败':  { text: '失败',     color: 'error' }
};

// 数据源
const loading = ref(false);
const tableData = ref([]);
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  pageSizeOptions: ['10', '20', '50', '100'],
  showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
  onShowSizeChange: (current, size) => {
    pagination.current = current;
    pagination.pageSize = size;
    fetchData(current, size, selectedPartnerId.value, selectedCountry.value, startDate.value, endDate.value);
  },
});

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

// 日期选择器
const Seleteddate = ref();
const startDate = ref("");
const endDate = ref("");

const onDateChange = (dates) => {
  if (dates && dates.length === 2) {
    startDate.value = dates[0].format("YYYY-MM-DD");
    endDate.value = dates[1].format("YYYY-MM-DD");
  } else {
    startDate.value = "";
    endDate.value = "";
  }
  pagination.current = 1;
  fetchData(1, pagination.pageSize, selectedPartnerId.value, selectedCountry.value, startDate.value, endDate.value);
};

// 获取数据
const fetchData = async (page = 1, pageSize = 10, partnerId = selectedPartnerId.value, country_code = selectedCountry.value, start_date = startDate.value, end_date = endDate.value) => {
  tableData.value = [];
  loading.value = true;
  try {
    const res = await request.get('/user/settlement_config', {
      params: {
        page,
        limit: pageSize,
        country_code,
        partner_id: partnerId,
        start_date,
        end_date,
      },
    });

    if (res.data.code === 200) {
      const { list, count } = res.data.data;
      // ✅ 在这里给每条记录添加默认状态字段
      tableData.value = list.map(item => ({
        ...item,
      }));
      pagination.current = page;
      pagination.total = count;
      // ✅ 判断是否还有“运行中”的任务
      const hasRunning = res.data.data.list.some(item => item.status === '运行中');

      if (!hasRunning) {
        stopRefreshTable(); // ✅ 自动停止
      }
    } else {
      message.error(res.data.msg || '获取数据失败');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
  } finally {
    loading.value = false;
  }
};

// 切换页面
const handleTableChange = (pag, filters, sorter) => {
  fetchData(pag.current, pag.pageSize, selectedPartnerId.value, selectedCountry.value, startDate.value, endDate.value);
};

// 下拉选项
const upUserIDOptions = ref([]);     // 合伙人下拉选项
// const upUserIDOptions = ref([]);     // 合伙人下拉选项

// 获取合伙人列表
const getbindUser = async () => {
  if (upUserIDOptions.value.length > 0) {
    return;
  }

  try {
    const res = await request.get('/user/user_name_list');
    if (res.data.code === 200 && Array.isArray(res.data.data)) {
      const userList = res.data.data || [];
      // 转换为 { label, value } 格式，并确保 value 是 number
      const options = userList.map(user => ({
        label: user.user_name || user.phone || `用户${user.id}`,
        value: Number(user.id)
      }));

      // ✅ 在最前面插入“默认配置”选项
      upUserIDOptions.value = [
        { label: '默认配置', value: 0 },
        ...options
      ];
    } else {
      message.info(res.data.msg || '数据为空');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
  }
};
getbindUser();

// 合伙人选择变化
const selectedPartnerId = ref(null);
const handlePartnerChange = (value) => {
  pagination.current = 1;
  fetchData(1, pagination.pageSize, value, selectedCountry.value, startDate.value, endDate.value);
};

// 根据国家筛选
const selectedCountry = ref("");
const countryOptions = ref([
  { label: '加纳', value: 'GH' },
  { label: '尼日利亚', value: 'NG' },
  { label: '肯尼亚', value: 'KE' },
]);
const handleCountryChange = (value) => {
  pagination.current = 1;
  fetchData(1, pagination.pageSize, selectedPartnerId.value, value, startDate.value, endDate.value);
};

// ========== 响应式变量：编辑弹窗控制 ==========
const modalVisible = ref(false);
// ========== 表单数据（用于弹窗）==========
const formState = reactive({
  id : 0,
  cloud_ride_commission_rate: undefined,
  settlement_rate: undefined,
});
// ========== 编辑配置：打开弹窗 ==========
const EditUserSettlementConfig = (record) => {
  if (!record || !record.id) {
    message.error('无效的配置记录');
    return;
  }
  console.log(record)
  formState.id = record.id;
  formState.cloud_ride_commission_rate = record.cloud_ride_commission_rate;
  formState.settlement_rate = record.settlement_rate;

  modalVisible.value = true;
};
// ========== 提交表单 ==========
const handleOk = async () => {
  try {
    // 可在此添加表单验证逻辑（使用 a-form 更方便，但我们现在用 reactive + 手动校验）
    if (formState.cloud_ride_commission_rate == null || formState.settlement_rate == null) {
      message.warning('请填写所有字段');
      return;
    }

    if (formState.cloud_ride_commission_rate < 0 || formState.cloud_ride_commission_rate > 1) {
      message.warning('云骑佣金比例必须在 0~1 之间');
      return;
    }

    if (formState.settlement_rate <= 0) {
      message.warning('结算汇率必须大于 0');
      return;
    }

    loading.value = true;

    const res = await request.put('/user/settlement_config', {
      id: formState.id,
      cloud_ride_commission_rate: formState.cloud_ride_commission_rate,
      settlement_rate: formState.settlement_rate,
    });

    if (res.data.code === 200) {
      message.success('更新成功');
      modalVisible.value = false;
      // 刷新表格
      fetchData(pagination.current, pagination.pageSize, selectedPartnerId.value, selectedCountry.value, startDate.value, endDate.value);
    } else {
      message.error(res.data.msg || '更新失败');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
  } finally {
    loading.value = false;
  }
};
// ========== 取消弹窗 ==========
const handleCancel = () => {
  modalVisible.value = false;
};


// 增加配置模块
// ========== 新增配置相关 ==========
const addModalVisible = ref(false);

// 新增表单数据
const addFormState = reactive({
  user_id: undefined,
  country_code: undefined,
  cloud_ride_commission_rate: undefined,
  settlement_rate: undefined,
  settlement_start_date: undefined,  // 新增
  settlement_end_date: undefined     // 新增
});
const tempWeek = ref(dayjs());

const onWeekChange = (date) => {
  if (!date) return;
  const monday = date.startOf('week').add(1, 'day');
  const sunday = date.endOf('week').add(1, 'day');
  addFormState.settlement_start_date = monday.format('YYYY-MM-DDTHH:mm:ss+08:00');
  addFormState.settlement_end_date = sunday.format('YYYY-MM-DDTHH:mm:ss+08:00');
};
// 打开新增弹窗
const openAddModal = () => {
  // 重置表单
  addFormState.user_id = undefined;
  addFormState.country_code = undefined;
  addFormState.cloud_ride_commission_rate = undefined;
  addFormState.settlement_rate = undefined;

  // ✅ 自动计算本周一 00:00:00 到 本周日 23:59:59
  const today = dayjs();
  const monday = today.startOf('week').add(1, 'day'); // 周一（dayjs 0 是周日）
  const sunday = today.endOf('week').add(1, 'day');   // 周日

  addFormState.settlement_start_date = monday.format('YYYY-MM-DDTHH:mm:ss+08:00');
  addFormState.settlement_end_date = sunday.format('YYYY-MM-DDTHH:mm:ss+08:00');

  addModalVisible.value = true;
};

// 提交新增配置
const handleAddSubmit = async () => {
  // 手动校验
  if (!addFormState.user_id == null) {
    message.warning('请选择合伙人');
    return;
  }
  if (!addFormState.country_code) {
    message.warning('请选择国家');
    return;
  }
  if (addFormState.cloud_ride_commission_rate == null || addFormState.cloud_ride_commission_rate < 0 || addFormState.cloud_ride_commission_rate > 1) {
    message.warning('云骑佣金比例必须在 0 ~ 1 之间');
    return;
  }
  if (addFormState.settlement_rate == null || addFormState.settlement_rate <= 0) {
    message.warning('结算汇率必须大于 0');
    return;
  }

  loading.value = true;
  try {
    const res = await request.post('/user/settlement_config', addFormState);

    if (res.data.code === 200) {
      message.success('新增成功');
      addModalVisible.value = false;
      // 刷新表格
      fetchData(pagination.current, pagination.pageSize, selectedPartnerId.value, selectedCountry.value, startDate.value, endDate.value);
    } else {
      message.error(res.data.msg || '新增失败');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
  } finally {
    loading.value = false;
  }
};

// 取消新增
const handleAddCancel = () => {
  addModalVisible.value = false;
};


// 运行配置并且轮询查看配置运行状态
const runSettlementConfig = async (record) => {
  // ✅ 检查：是否已在运行（使用 record.status）
  if (record.status === 'running') {
    message.info('任务已在运行中');
    return;
  }

  Modal.confirm({
    title: '确认运行？',
    onOk: async () => {
      try {
        const res = await request.get(`/user/settlement_config/${record.id}`);
        if (res.data.code === 200) {
          message.success('任务已启动');

          // ✅ 2. 更新本地状态（乐观更新）
          record.status = '运行中';

          // ✅ 3. 开始定时刷新表格（不是轮询单个任务！）
          startRefreshTable();
        }
      } catch (err) {
        message.error('启动失败');
      }
    }
  });
};
let refreshTimer = null;

const startRefreshTable = () => {
  // ✅ 清除旧定时器
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }

  // ✅ 每 3 秒刷新一次表格
  refreshTimer = setInterval(() => {
    fetchData(1, pagination.pageSize, selectedPartnerId.value, selectedCountry.value, startDate.value, endDate.value);
  }, 3000);

  console.log('✅ 开始定时刷新表格（每 3 秒）');
};

const stopRefreshTable = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
    console.log('🛑 停止定时刷新表格');
  }
};

onUnmounted(() => {
  stopRefreshTable();
});

// 删除配置
const DeleteUserSettlementConfig =  async (record) => {
  loading.value = true;
  try {
    const res = await request.delete(`/user/settlement_config/${record.id}`, );

    if (res.data.code === 200) {
      message.info("删除配置成功")
      fetchData(pagination.current, pagination.pageSize, selectedPartnerId.value, selectedCountry.value, startDate.value, endDate.value);
    } else {
      message.error(res.data.msg || '删除配置失败');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
  } finally {
    loading.value = false;
  }
};

// 初始化
onMounted(() => {
  getbindUser();
  fetchData(pagination.current, pagination.pageSize);
});
</script>

<style scoped>
/* 如果有样式需求可以在这里添加 */
</style>