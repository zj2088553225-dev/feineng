
<template>
  <a-card style="padding: 10px" title="Jumia订单明细页">
    <!-- 搜索区域 -->
    <a-row :gutter="16" style="margin-bottom: 16px;">
      <a-col :span="3">
        <a-select
            v-model:value="selectedCountry"
            :options="countryOptions"
            placeholder="选择国家"
            allowClear
            style="width: 100%;"
            @change="handleCountryChange"
        />
      </a-col>
      <a-col :span="3">
        <a-select
            v-model:value="selectedTransactionType"
            :options="transactionTypeOptions"
            placeholder="选择交易类型"
            allowClear
            style="width: 100%;"
            @change="handleTransactionTypeChange"
        />
      </a-col>
      <a-col :span="3">
        <a-select
            v-model:value="selectedStatus"
            :options="statusOptions"
            placeholder="选择状态"
            allowClear
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
      <a-col :span="4">
        <a-input
            v-model:value="searchKeyword"
            placeholder="支持 seller_sku/jumiasku 搜索"
            @pressEnter="handleSearch"
            allowClear
        >
          <template #suffix>
            <SearchOutlined />
          </template>
        </a-input>
      </a-col>
      <a-col :span="4">
        <a-button type="primary" @click="handleSearch">搜索</a-button>
                <a-button style="margin-left: 10px" @click="handleExport" :loading="exportLoading">导出 Excel</a-button>
      </a-col>
    </a-row>
    <!--    交易表单-->
    <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        row-key="transaction_number"
        :pagination="pagination"
        @change="handleTableChange"
        :scroll="scroll"
        :key="tableKey"
    >
    </a-table>
  </a-card>
</template>
<script setup>

import * as XLSX from 'xlsx'; // 需要安装 xlsx 包：npm install xlsx

const exportLoading = ref(false);

const formatDateForFilename = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const fetchAllTransactionData = async () => {
  const allData = [];
  const pageSize = 200;
  let page = 1;
  const maxPages = 100; // 最多支持 2万条数据

  let totalCount = null;

  while (page <= maxPages) {
    try {
      const params = {
        page,
        limit: pageSize,
        key: searchKeyword.value || undefined,
        country_code: selectedCountry.value || undefined,
        paid_status: selectedStatus.value || undefined,
        start_date: startDate.value || undefined,
        end_date: endDate.value || undefined,
        transaction_type: selectedTransactionType.value || undefined,
      };

      const res = await request.get('/user/my_transactions', { params });
      if (res.data.code !== 200) {
        message.error(`导出时，第 ${page} 页请求失败：${res.data.msg || '未知错误'}`);
        break;
      }

      const { list, count } = res.data.data;
      if (!Array.isArray(list)) {
        message.error('导出时获取数据格式错误');
        break;
      }

      if (totalCount === null) {
        totalCount = count;
      }

      allData.push(...list);

      if (list.length === 0 || allData.length >= totalCount) {
        break;
      }

      page++;
    } catch (err) {
      message.error(`导出时请求异常：${err.message}`);
      break;
    }
  }

  if (page > maxPages) {
    message.warning(`数据量过大，仅导出前 ${maxPages * pageSize} 条`);
  }

  return allData;
};

const handleExport = async () => {
  exportLoading.value = true;
  try {
    const allTransactions = await fetchAllTransactionData();

    if (allTransactions.length === 0) {
      message.warning('没有可导出的交易数据');
      exportLoading.value = false;
      return;
    }

    // 转换数据为 Excel 行格式（根据你的列结构调整）
    const exportRows = allTransactions.map(item => ({
      '交易日期': formatTime(item.transaction_date),
      '交易类型': item.transaction_type,
      // '交易单号': item.transaction_number,
      '交易状态': item.transaction_state,
      '交易详情': item.details,
      '卖家 SKU': item.seller_sku,
      'Jumia SKU': item.jumia_sku,
      '订单号': item.order_no,
      '金额': Number(item.amount.toFixed(2)),
      '结算开始日期': formatTime(item.statement_start_date),
      '结算结束日期': formatTime(item.statement_end_date),
      '支付状态': item.paid_status ? '已支付' : '未支付',
      '备注': item.comment?.String || '',
      // '汇率': item.local_exchange_rate,
      '国家代码': item.country_code,
      // '结算单号': item.statement_number,
    }));

    const ws = XLSX.utils.json_to_sheet(exportRows);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, '交易明细');

    const fileName = `合伙人交易明细_${formatDateForFilename(new Date())}.xlsx`;
    XLSX.writeFile(wb, fileName);

    message.success(`导出成功！共导出 ${allTransactions.length} 条数据`);
  } catch (error) {
    message.error('导出失败，请重试');
    console.error(error);
  } finally {
    exportLoading.value = false;
  }
};









// ========== 表格列定义（不变）==========
import {onBeforeUnmount, onMounted, reactive, ref, watch} from "vue";
import request from "@/utils/request.js";
import {message} from "ant-design-vue";
import {SearchOutlined} from "@ant-design/icons-vue";

const columns = [
  { title: '交易日期', dataIndex: 'transaction_date', key: 'transaction_date', customRender: ({ value }) => formatTime(value) },
  { title: '交易类型', dataIndex: 'transaction_type', key: 'transaction_type' },
  // { title: '交易单号', dataIndex: 'transaction_number', key: 'transaction_number' },
  { title: '交易状态', dataIndex: 'transaction_state', key: 'transaction_state' },
  { title: '交易详情', dataIndex: 'details', key: 'details', ellipsis: true },
  { title: '卖家 SKU', dataIndex: 'seller_sku', key: 'seller_sku' },
  { title: 'Jumia SKU', dataIndex: 'jumia_sku', key: 'jumia_sku', ellipsis: true },
  { title: '订单号', dataIndex: 'order_no', key: 'order_no', ellipsis: true },
  { title: '金额', dataIndex: 'amount', key: 'amount', customRender: ({ value }) => value.toFixed(2) },
  { title: '结算开始日期', dataIndex: 'statement_start_date', key: 'statement_start_date', customRender: ({ value }) => formatTime(value) },
  { title: '结算结束日期', dataIndex: 'statement_end_date', key: 'statement_end_date', customRender: ({ value }) => formatTime(value) },
  { title: '支付状态', dataIndex: 'paid_status', key: 'paid_status', customRender: ({ value }) => value ? '已支付' : '未支付' },
  { title: '备注', dataIndex: ['comment', 'String'], key: 'comment', ellipsis: true },
  // { title: '汇率', dataIndex: 'local_exchange_rate', key: 'local_exchange_rate' },
  { title: '国家代码', dataIndex: 'country_code', key: 'country_code' },
  // { title: '结算单号', dataIndex: 'statement_number', key: 'statement_number' }
];
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
    fetchData(current, size, searchKeyword.value,selectedCountry.value,selectedStatus.value, startDate.value,
        endDate.value,selectedTransactionType.value);
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
  console.log(dates)
  // 搜索时回到第一页
  pagination.current = 1;
  // 搜索时回到第一页
  fetchData(1, pagination.pageSize, searchKeyword.value,selectedCountry.value,selectedStatus.value, startDate.value,
      endDate.value,selectedTransactionType.value);
};
const tableKey = ref(1);
// 获取数据
const fetchData = async (page = 1, pageSize = 10, keyword = searchKeyword.value,country_code=selectedCountry.value,paid_status =selectedStatus.value, start_date = startDate.value,
                         end_date = endDate.value,transaction_type = selectedTransactionType.value) => {
  tableData.value = []
  loading.value = true;
  try {
    const res = await request.get('/user/my_transactions', {
      params: {
        page,
        limit: pageSize,
        key: keyword,
        country_code:country_code,
        paid_status:paid_status,
        start_date, // ✅ 前端传后端可识别的日期
        end_date,
        transaction_type
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
// 搜索功能
const searchKeyword = ref(''); // 搜索关键词
const handleSearch = () => {
  // 搜索时回到第一页
  pagination.current = 1;
  // 搜索时回到第一页
  fetchData(1, pagination.pageSize, searchKeyword.value,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value,selectedTransactionType.value);
};

// 切换页面
const handleTableChange = (pag, filters, sorter) => {


  fetchData(pag.current, pag.pageSize, searchKeyword.value,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value,selectedTransactionType.value);
};
watch(searchKeyword, (val) => {
  if (val === '') {
    // 可选：清空搜索框时自动搜索（清空结果）
    fetchData(1, pagination.pageSize, '',selectedCountry.value,selectedStatus.value,startDate.value,endDate.value,selectedTransactionType.value);
  }
});
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
  fetchData(1, pagination.pageSize, searchKeyword.value,value,selectedStatus.value,startDate.value, endDate.value,selectedTransactionType.value);
};

// 根据国家筛选
const selectedTransactionType = ref("");
const transactionTypeOptions = ref([
  { label: 'Jumia佣金', value: 'Commission' },
  { label: '佣金退款', value: 'Commission Credit' },
  { label: '商品总价', value: 'Item Price' },
  { label: '商品总价退款', value: 'Item Price Credit' },
  { label: '丢失或损坏', value: 'Lost or Damaged (Product Level) Credit' },
  { label: '出库费', value: 'Outbound Fee' },
  { label: '出库费退款', value: 'Outbound Fee Credit' },
  { label: '仓储费', value: 'Storage Fee' },
  { label: '补贴', value: 'Subsidy' },
  { label: '补贴退款', value: 'Subsidy Refund' },
]);
const handleTransactionTypeChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  fetchData(1, pagination.pageSize, searchKeyword.value,selectedCountry.value,selectedStatus.value,startDate.value, endDate.value,value);
};
// 根据状态筛选
const selectedStatus = ref("");
const statusOptions = ref([
  { label: '已支付', value: '1' },
  { label: '未支付', value: '0' },
  // 其他国家...
]);
const handleStatusChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  fetchData(1, pagination.pageSize, searchKeyword.value,selectedCountry.value,value,startDate.value, endDate.value,selectedTransactionType.value);
};
// 初始化
// 初始化

const scroll = ref({ x: 2000, y: 600 })

// 根据窗口大小动态计算 scroll
function updateScroll() {
  const width = window.innerWidth
  const height = window.innerHeight

  // 这里你可以根据自己的布局调整比例或固定值
  scroll.value = {
    x: width * 1.2,  // 表格横向滚动宽度（超出屏幕）
    y: height * 0.45  // 表格纵向高度占窗口的 60%
  }
}
onMounted(() => {
  updateScroll()
  window.addEventListener('resize', updateScroll)
  // 第一次加载
  fetchData(pagination.current, pagination.pageSize, searchKeyword.value);
});
onBeforeUnmount(() => {
  window.removeEventListener('resize', updateScroll)
})

</script>



<style scoped>

</style>