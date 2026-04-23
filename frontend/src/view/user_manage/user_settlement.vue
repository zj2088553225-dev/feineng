
<template>
  <a-card style="padding: 10px" title="合伙人结算明细页">
    <!-- 搜索区域 -->
    <a-row :gutter="16" style="margin-bottom: 16px;">

      <!-- ✅ 新增：合伙人下拉框 -->
      <a-col :span="2">
        <a-select
            v-model:value="selectedPartnerId"
            :options="upUserIDOptions"
            placeholder="选择合伙人"
            allowClear
            :disabled="activeKey === 'total'"
            style="width: 100%;"
            @change="handlePartnerChange"
        />
      </a-col>
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
      <a-col :span="2">
        <a-select
            v-model:value="selectedStatus"
            :options="statusOptions"
            placeholder="选择状态"
            allowClear
            :disabled="activeKey === 'total'"
            style="width: 100%;"
            @change="handleStatusChange"
        />
      </a-col>
      <a-col :span="6">
        <a-range-picker
            v-model:value="Seleteddate"
            show-time
            format="YYYY-MM-DD"
            @change="onDateChange"
        />
      </a-col>
    </a-row>
    <a-tabs v-model:activeKey="activeKey" type="card">
      <a-tab-pane key="user" tab="周期合伙人数据">
        <div style="overflow-x: auto; width: 100%;">
          <a-table
              :columns="columns"
              :data-source="tableData"
              :loading="loading"
              row-key="id"
              :pagination="pagination"
              @change="handleTableChange"
              :expandable="expandable"
              :scroll="{ x: 2000, y: 600 }"
              :key="tableKey"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
        <span>
          <a @click="EditUserSettlementStatus(record)">编辑状态</a>
        </span>
              </template>
            </template>
            <!-- 展开行模板 -->
            <template #expandedRowRender="{ record }">
              <div style="padding: 16px; background: #fafafa; margin: -16px -16px 0;">
                <a-table
                    :columns="itemColumns"
                    :data-source="record.Details"
                    :pagination="false"
                    :show-header="true"
                    size="small"
                    :row-key="item => item.id"
                    :scroll="{ x: 'max-content' }"
                />
              </div>
            </template>
          </a-table>
        </div>
      </a-tab-pane>
      <a-tab-pane key="total" tab="周期汇总数据" force-render>
        <div class="summary-tab-content">
          <!-- 加载状态 -->
          <a-spin :spinning="loading">
            <!-- 无数据 -->
            <div v-if="!tableDataTotal || Object.keys(tableDataTotal).length === 0" class="no-data">
              <a-empty description="暂无汇总数据" />
            </div>

            <!-- 有数据：使用 Descriptions 展示 -->
            <a-card v-else :bordered="true" size="small" :body-style="{ padding: '16px' }">
              <a-descriptions
                  :column="2"
                  bordered
                  size="small"
                  :key="tableDataTotal.country_code + '-' + tableDataTotal.total_signed_count"
              >

                <!-- 结算周期 -->
                <a-descriptions-item label="结算周期" :span="2">
                  {{ formatSettlementPeriod(tableDataTotal) }}
                </a-descriptions-item>

                <!-- 国家 -->
                <a-descriptions-item label="国家">
                  {{ getCountryName(tableDataTotal.country_code) }}
                </a-descriptions-item>

                <!-- 签收数 -->
                <a-descriptions-item label="签收数">
                  {{ tableDataTotal.total_signed_count || 0 }}
                </a-descriptions-item>

                <!-- 总签收金额 -->
                <a-descriptions-item label="总签收金额 (当地货币)">
                  {{ formatCurrency(tableDataTotal.total_signed_amount) }}
                </a-descriptions-item>

                <!-- Jumia总抽佣 -->
                <a-descriptions-item label="Jumia总抽佣 (当地货币)">
                  {{ formatCurrency(tableDataTotal.total_jumia_commission) }}
                </a-descriptions-item>

                <!-- 总出库费 -->
                <a-descriptions-item label="总出库费 (当地货币)">
                  {{ formatCurrency(tableDataTotal.total_outbound_fee) }}
                </a-descriptions-item>

                <!-- 总库存费 -->
                <a-descriptions-item label="总库存费 (当地货币)">
                  {{ formatCurrency(tableDataTotal.total_storage_fee) }}
                </a-descriptions-item>

                <!-- 实际到账总金额 -->
                <a-descriptions-item label="实际到账总金额 (当地货币)">
                <span style="color: #52c41a; font-weight: 500;">
                  {{ formatCurrency(tableDataTotal.received_amount) }}
                </span>
                </a-descriptions-item>

                <!-- 云驰总抽佣 -->
                <a-descriptions-item label="云驰总抽佣 (当地货币)">
                  {{ formatCurrency(tableDataTotal.total_cloud_ride_commission) }}
                </a-descriptions-item>

                <!-- Pyvio总手续费 -->
                <a-descriptions-item label="Pyvio总手续费 (当地货币)">
                  {{ formatCurrency(tableDataTotal.total_pyvio_fee) }}
                </a-descriptions-item>

                <!-- 审单总费用 -->
                <a-descriptions-item label="审单总费用 (当地货币)">
                  {{ formatCurrency(tableDataTotal.total_review_fee) }}
                </a-descriptions-item>

                <!-- 实际结算总金额 (当地货币) -->
                <a-descriptions-item label="实际结算总金额 (当地货币)">
                <span style="color: #1890ff; font-weight: 500;">
                  {{ formatCurrency(tableDataTotal.actual_settle_amount) }}
                </span>
                </a-descriptions-item>

                <!-- 实际结算总金额 (人民币) -->
                <a-descriptions-item label="实际结算总金额 (人民币)">
                <span style="color: #1890ff; font-weight: 500;">
                  {{ formatCurrency(tableDataTotal.actual_settle_cny) }}
                </span>
                </a-descriptions-item>

                <!-- 关联用户数 -->
                <a-descriptions-item label="关联用户数">
                  {{ tableDataTotal.user_count || 0 }}
                </a-descriptions-item>
              </a-descriptions>
              <a-descriptions :column="2" bordered size="small">
                <a-descriptions-item label="每个合营合伙人明细当地货币" :span="2">
                  <div style="display: flex; flex-wrap: wrap; gap: 8px;">
                    <a-tag
                        v-for="(partner, index) in calculationDetail.PerPartnerCooperation"
                        :key="index"
                        color="geekblue"
                        style="margin-bottom: 4px"
                    >
                      <span v-if="partner.note"> - {{ partner.note }}</span>
                      ({{ (partner.rate * 100).toFixed(0) }}%) ：
                      {{ formatCurrency(partner.cooperation_amount) }}
                    </a-tag>
                  </div>
                </a-descriptions-item>

                <!-- 公司收入（当地货币） & 公司收入（人民币） -->
                <a-descriptions-item label="公司收入 (当地货币)">
    <span style="color: #fa541c; font-weight: 600;">
      {{ formatCurrency(tableDataTotal.company_profits) }}
    </span>
                </a-descriptions-item>

                <a-descriptions-item label="公司收入 (人民币)">
    <span style="color: #1890ff; font-weight: 600;">
      {{ formatCurrency(calculationDetail.ResultCNY) }}
    </span>
                </a-descriptions-item>

                <!-- 利润计算过程（当地货币） & 利润计算过程（人民币） -->
                <a-descriptions-item label="公司收入计算过程">
                  实际到账: {{ formatCurrency(calculationDetail.ReceivedAmount) }} -
                  合伙人结算: {{ formatCurrency(calculationDetail.TotalActualSettleAmount) }} -
                  合营扣除: {{ formatCurrency(calculationDetail.TotalCooperationDeduction) }} =
                  <span style="color: #f5222d; font-weight: 500;">
      {{ formatCurrency(calculationDetail.Result) }}
    </span>
                </a-descriptions-item>

                <a-descriptions-item label="计算过程 (人民币)">
                  实际到账: {{ formatCurrency(calculationDetail.ReceivedAmount * calculationDetail.Rate) }} -
                  合伙人结算: {{ formatCurrency(calculationDetail.TotalActualSettleAmount * calculationDetail.Rate) }} -
                  合营扣除: {{ formatCurrency(calculationDetail.TotalCooperationDeduction * calculationDetail.Rate) }} =
                  <span style="color:#f5222d; font-weight:500;">
      {{ formatCurrency(calculationDetail.ResultCNY) }}
    </span>
                </a-descriptions-item>
              </a-descriptions>
            </a-card>
          </a-spin>
        </div>
      </a-tab-pane>
    </a-tabs>
  </a-card>
</template>
<script setup>
// ========== 表格列定义（不变）==========
import {h, onMounted, reactive, ref, watch} from "vue";
import request from "@/utils/request.js";
import {message} from "ant-design-vue";
import dayjs from 'dayjs';
const activeKey = ref('total');
const tableDataTotal = ref({}); // 存储汇总对象
const calculationDetail = ref({}); //存储计算过程数据

// 格式化结算周期
const formatSettlementPeriod = (data) => {
  if (!data?.settlement_start_date || !data?.settlement_end_date) {
    return '-- ~ --';
  }
  const start = dayjs(data.settlement_start_date).format('YYYY-MM-DD');
  const end = dayjs(data.settlement_end_date).format('YYYY-MM-DD');
  return `${start} ~ ${end}`;
};

// 国家代码转中文
const getCountryName = (code) => {
  return { 'GH': '加纳', 'NG': '尼日利亚', 'KE': '肯尼亚' }[code] || code;
};

// 格式化金额，保留2位小数
const formatCurrency = (value) => {
  if (value === undefined || value === null) return '0.00';
  return parseFloat(value).toFixed(2);
};
const fetchDataTotal = async (
    country_code = selectedCountry.value,
    start_date = startDate.value,
    end_date = endDate.value
) => {
  loading.value = true;

  try {
    // ✅ 如果 start_date 或 end_date 为空，则计算本周一和本周日
    if (!start_date || !end_date) {
      const today = dayjs(); // 当前日期

      // 计算本周一 (day(1) 表示周一)
      const monday = today.day(1).startOf('day'); // 00:00:00
      // 计算本周日 (day(7) 表示周日)
      const sunday = today.day(7).endOf('day');   // 23:59:59

      start_date = monday.format('YYYY-MM-DD'); // 格式化为 '2025-08-25'
      end_date = sunday.format('YYYY-MM-DD');   // 格式化为 '2025-09-01'
    }
    const res = await request.get('/user/settlement_total', {
      params: { country_code, start_date, end_date },
    });

    if (res.data.code === 200) {
      const data = res.data.data;
      console.log(data);

      if (!data || Object.keys(data).length === 0) {
        message.info('暂无数据');
        tableDataTotal.value = {};
        calculationDetail.value = {};
      } else {
        message.success('数据获取成功');
        tableDataTotal.value = { ...data.summary };  // 只放 summary
        calculationDetail.value = { ...data.calculationDetail }; // 单独存详情
        console.log('📊 汇总:', tableDataTotal.value);
        console.log('📊 计算详情:', calculationDetail.value);
      }
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
    tableDataTotal.value = {};
  } finally {
    loading.value = false; // ✅ 确保 loading 关闭
  }
};
// 添加监听器
watch(activeKey, (newVal) => {
  if (newVal === 'total') {
    fetchDataTotal(selectedCountry.value, startDate.value, endDate.value);
  }
  fetchData(1, pagination.pageSize, selectedPartnerId.value,selectedCountry.value,selectedStatus.value,startDate.value, endDate.value);
});



const columns = [
  {
    title: '结算周期',
    key: 'settlement_period',
    width: 200,
    fixed: 'left',
    customRender: ({ record }) => {
      const start = record.settlement_start_date ? dayjs(record.settlement_start_date).format('YYYY-MM-DD') : '--';
      const end = record.settlement_end_date ? dayjs(record.settlement_end_date).format('YYYY-MM-DD') : '--';
      return `${start} ~ ${end}`;
    }
  },
  {
    title: '国家',
    dataIndex: 'country_code',
    key: 'country_code',
    width: 80,
    customRender: ({ text }) => ({
      'GH': '加纳',
      'NG': '尼日利亚',
      'KE': '肯尼亚'
    })[text] || text
  },
  {
    title: '签收数',
    dataIndex: 'signed_count',
    key: 'signed_count',
    width: 80
  },
  {
    title: '总签收金额 (当地货币)',
    dataIndex: 'total_signed_amount',
    key: 'total_signed_amount',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: 'Jumia总抽佣 (当地货币)',
    dataIndex: 'total_jumia_commission',
    key: 'total_jumia_commission',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '总出库费 (当地货币)',
    dataIndex: 'total_outbound_fee',
    key: 'total_outbound_fee',
    width: 100,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '总库存费 (当地货币)',
    dataIndex: 'total_storage_fee',
    key: 'total_storage_fee',
    width: 100,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '实际到账总金额 (当地货币)',
    dataIndex: 'received_amount',
    key: 'received_amount',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '云驰总抽佣 (当地货币)',
    dataIndex: 'total_cloud_ride_commission',
    key: 'total_cloud_ride_commission',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: 'Pyvio总手续费  (当地货币)',
    dataIndex: 'total_pyvio_fee',
    key: 'total_pyvio_fee',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '审单总费用 (当地货币)',
    dataIndex: 'total_review_fee',
    key: 'total_review_fee',
    width: 100,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '实际结算总金额 (当地货币)',
    dataIndex: 'actual_settle_amount',
    key: 'actual_settle_amount',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '实际结算总金额 (人民币)',
    dataIndex: 'actual_settle_cny',
    key: 'actual_settle_cny',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '结算状态',
    dataIndex: 'settlement_status',
    key: 'settlement_status',
    width: 100,
    customRender: ({ text }) => {
      const color = text === '已结算' ? 'green' : 'orange';
      return h('span', {
        style: { color }
      }, text);
    }
  },
  {
    title: '更新时间',
    key: 'updated_at',
    width: 160,
    customRender: ({ record }) => formatTime(record.updated_at)
  },
  { title: '操作', key: 'action' },
];
const itemColumns = [
  {
    title: '卖家 SKU',
    dataIndex: 'seller_sku',
    key: 'seller_sku',
    width: 140,
    fixed: 'left'
  },
  {
    title: '签收笔数',
    dataIndex: 'signed_count',
    key: 'signed_count',
    width: 80,
    customRender: ({ value }) => value || 0
  },
  {
    title: '总签收金额 (当地货币)',
    dataIndex: 'total_signed_amount',
    key: 'total_signed_amount',
    width: 100,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: 'Jumia抽佣 (当地货币)',
    dataIndex: 'jumia_commission',
    key: 'jumia_commission',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '妥头出库费 (当地货币)',
    dataIndex: 'outbound_fee',
    key: 'outbound_fee',
    width: 100,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '库存费 (当地货币)',
    dataIndex: 'storage_fee',
    key: 'storage_fee',
    width: 100,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '实际到账金额 (当地货币)',
    dataIndex: 'received_amount',
    key: 'received_amount',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '云驰抽佣 (当地货币)',
    dataIndex: 'cloud_ride_commission',
    key: 'cloud_ride_commission',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '云驰抽佣比例',
    dataIndex: 'cloud_ride_commission_rate',
    key: 'cloud_ride_commission_rate',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: 'Pyvio手续费 (当地货币)',
    dataIndex: 'pyvio_fee',
    key: 'pyvio_fee',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '审单费用 (当地货币)',
    dataIndex: 'review_fee',
    key: 'review_fee',
    width: 100,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },
  {
    title: '实际结算金额 (当地货币)',
    dataIndex: 'actual_settle_amount',
    key: 'actual_settle_amount',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  },{
    title: '结算汇率',
    dataIndex: 'settlement_rate',
    key: 'settlement_rate',
    width: 120,
    customRender: ({ value }) => value?.toFixed(6) || '0.00'
  },
  {
    title: '实际结算金额 (人民币)',
    dataIndex: 'actual_settle_cny',
    key: 'actual_settle_cny',
    width: 120,
    customRender: ({ value }) => value?.toFixed(2) || '0.00'
  }
];
const expandable = ref({
  // 启用展开功能，具体模板在 #expandedRowRender 中定义
});
const loading = ref(false);
const tableData = ref([]);
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
const pagination = reactive({
  current: 1,
  pageSize: 10, // 默认每页10条
  total: 0,
  showSizeChanger: true,           // 显示“每页条数”下拉框
  showQuickJumper: true,           // 显示“快速跳转”
  pageSizeOptions: ['10', '20', '50', '100'], // 可选的每页条数
  showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
  onShowSizeChange: (current, size) => {
    // 更新当前页码和页面大小
    pagination.current = current;
    pagination.pageSize = size;

    // 调用 fetchData 立即加载新数据
    // fetchData(current, size, searchKeyword.value);
    // 👉 根据当前 activeKey 调用不同的请求函数
    if (activeKey.value === 'user') {
    fetchData(current, size,selectedPartnerId.value,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);
    } else if (activeKey.value === 'total') {
      fetchDataTotal(selectedCountry.value, startDate.value, endDate.value);
    }
  },
})

const Seleteddate = ref();
// 用 ref 保存当前选中的时间范围
const startDate = ref("");
const endDate = ref("");
const onDateChange = (dates) => {
  if (dates && dates.length === 2) {
    // 这里用 "YYYY-MM-DD" 就行，因为后端会自动加时间
    startDate.value = dates[0].format("YYYY-MM-DD");
    endDate.value = dates[1].format("YYYY-MM-DD");
  } else {
    startDate.value = "";
    endDate.value = "";
  }
  // console.log(dates)
  // 搜索时回到第一页
  pagination.current = 1;
  // 搜索时回到第一页
  if (activeKey.value === 'user') {
  fetchData(1, pagination.pageSize, selectedPartnerId.value,selectedCountry.value,selectedStatus.value, startDate.value,
      endDate.value);
  } else if (activeKey.value === 'total') {
    fetchDataTotal(selectedCountry.value, startDate.value, endDate.value);
  }

};
const tableKey = ref(1);

// 获取数据
const fetchData = async (page = 1, pageSize = 10,partnerId = selectedPartnerId.value,country_code=selectedCountry.value,paid_status =selectedStatus.value, start_date = startDate.value,
                         end_date = endDate.value) => {
  tableData.value = []
  loading.value = true;
  try {
    const res = await request.get('/user/settlement', {
      params: {
        page,
        limit: pageSize,
        country_code:country_code,
        partner_id: partnerId, // ✅ 传给后端
        paid_status:paid_status,
        start_date, // ✅ 前端传后端可识别的日期
        end_date,
      },
    });

    if (res.data.code === 200) {
      const { list, count } = res.data.data;
      tableKey.value = Date.now(); // ✅ 用时间戳确保唯一
      if (!list || list.length === 0) {
        message.info('暂无数据');
        tableData.value = [];
        pagination.total = 0;
        return;
      }

      // 确保 orderItems 是数组
      tableData.value = list
      pagination.current = page;
      pagination.total = count;
      console.log(res.data.data)
      console.log(tableData.value)
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
  if (activeKey.value === 'user') {
  fetchData(pag.current, pag.pageSize,selectedPartnerId.value,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);
  } else if (activeKey.value === 'total') {
    fetchDataTotal(selectedCountry.value, startDate.value, endDate.value);
  }
};


const upUserIDOptions = ref([]);     // 下拉选项
const selectedPartnerId = ref(); // ✅ 初始为空，不选中任何人
// ========== 获取合伙人列表 ==========
const getbindUser = async () => {
  if (upUserIDOptions.value.length > 0) {
    // ✅ 避免重复请求（可选）
    return;
  }

  try {
    const res = await request.get('/user/user_name_list');
    if (res.data.code === 200 && Array.isArray(res.data.data)) {
      upUserIDOptions.value = res.data.data.map(user => ({
        value: user.id,      // ✅ 假设 user.id 是 number 类型
        label: user.user_name,
      }));
    } else {
      message.info(res.data.msg || '数据为空');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
  }
};
getbindUser()
// 合伙人选择变化
const handlePartnerChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  if (activeKey.value === 'user') {
  fetchData(1, pagination.pageSize, value,selectedCountry.value,selectedStatus.value,startDate.value, endDate.value);
  } else if (activeKey.value === 'total') {
    fetchDataTotal(selectedCountry.value, startDate.value, endDate.value);
  }
};

// 根据国家筛选
const selectedCountry = ref("");
const countryOptions = ref([
  { label: '加纳', value: 'GH' },
  { label: '尼日利亚', value: 'NG' },
  { label: '肯尼亚', value: 'KE' },
]);
const handleCountryChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  if (activeKey.value === 'user') {
  fetchData(1, pagination.pageSize,selectedPartnerId.value,value,selectedStatus.value,startDate.value, endDate.value);
  } else if (activeKey.value === 'total') {
    fetchDataTotal(selectedCountry.value, startDate.value, endDate.value);
  }
};



// 根据状态筛选
const selectedStatus = ref("");
const statusOptions = ref([
  { label: '已结算', value: '已结算' },
  { label: '待结算', value: '待结算' },
  // 其他国家...
]);
const handleStatusChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  if (activeKey.value === 'user') {
  fetchData(1, pagination.pageSize, selectedPartnerId.value,selectedCountry.value,value,startDate.value, endDate.value);
  } else if (activeKey.value === 'total') {
    fetchDataTotal(selectedCountry.value, startDate.value, endDate.value);
  }
};


const EditUserSettlementStatus = async (record) => {
  loading.value = true;
  if (record.settlement_status === "待结算") {
    record.settlement_status = "已结算"
  }else {
    record.settlement_status = "待结算"
  }
  try {
    const res = await request.put('/user/settlement', {
      settlement_id: record.id,
      settlement_status:record.settlement_status
    });

    if (res.data.code === 200) {
      message.info('修改状态成功');
    } else {
      message.error(res.data.msg || '修改状态失败');
    }
  } catch (err) {
    message.error('请求失败，请检查网络');
    console.error(err);
  } finally {
    loading.value = false;
  }
};

// 初始化
// 初始化
onMounted(() => {
  // 第一次加载
  fetchData(pagination.current, pagination.pageSize);
  fetchDataTotal()
});

</script>



<style scoped>

</style>