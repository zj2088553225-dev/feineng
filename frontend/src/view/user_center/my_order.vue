<template>
  <a-card style="padding: 10px" title="Jumia订单页">
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
            placeholder="输入 seller_sku/jumiasku 搜索"
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

    <!-- 表格 -->
    <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        row-key="id"
        :pagination="pagination"
        @change="handleTableChange"
        :expandable="expandable"
        :scroll="scroll"
    >
      <!-- 自定义展开内容 -->
      <template #expandedRowRender="{ record }">
        <div style="padding: 16px; background: #fafafa; margin: -16px -16px 0;">
          <a-table
              :columns="itemColumns"
              :data-source="record.orderItems"
              :pagination="false"
              :show-header="true"
              size="small"
              :row-key="item => item.id"
          >
            <!-- 自定义金额显示 -->
            <template #bodyCell="{ column, text, record: item }">
              <template v-if="column.dataIndex === 'paidPriceLocal' || column.dataIndex === 'shippingAmountLocal'">
                {{ formatCurrency(text, `${record.CountryCurrency}`) }}
              </template>
              <template v-else-if="column.dataIndex === 'productName'">
                <div style="display: flex; align-items: center; gap: 8px;">
                  <img
                      v-if="item.imageUrl"
                      :src="item.imageUrl"
                      :alt="item.productName"
                      style="width: 40px; height: 40px; object-fit: cover; border-radius: 4px;"
                  />
                  <span>{{ text }}</span>
                </div>
              </template>
              <template v-else>
                {{ text }}
              </template>
            </template>
          </a-table>
        </div>
      </template>
    </a-table>
  </a-card>
</template>

<script setup>
import {onBeforeUnmount, onMounted, reactive, ref, watch} from 'vue';
import request from '@/utils/request';
import { message } from 'ant-design-vue';
import { SearchOutlined } from '@ant-design/icons-vue';
import * as XLSX from 'xlsx';
const exportLoading = ref(false); // 控制导出按钮 loading

// 导出 Excel
// 导出 Excel（包含明细）

const handleExport = async () => {
  exportLoading.value = true;
  try {
    const allOrders = await fetchAllOrderData();
    console.log('📊 共获取订单数量:', allOrders.length);

    // 统计总 item 数
    const totalItems = allOrders.reduce((sum, order) => {
      return sum + (Array.isArray(order.orderItems) ? order.orderItems.length : 0);
    }, 0);
    console.log('📦 共计订单项（SKU）数量:', totalItems);

    if (totalItems === 0) {
      message.warning('没有可导出的商品明细');
      return;
    }

    // 💰 定义累计变量
    let totalOrderAmount = 0;
    let totalPaidAmount = 0;
    let totalShippingAmount = 0;

    // 生成导出行
    const exportRows = [];
    allOrders.forEach(order => {
      const items = Array.isArray(order.orderItems) ? order.orderItems : [];

      if (items.length === 0) {
        exportRows.push({
          '订单号': order.number,
          '国家': order.CountryName,
        });
      } else {
        items.forEach((item, index) => {
          if (index === 0) {
            totalOrderAmount += Number(order.TotalAmountLocalValue) || 0;
          }
          totalPaidAmount += Number(item.paidPriceLocal) || 0;
          totalShippingAmount += Number(item.shippingAmountLocal) || 0;

          exportRows.push({
            '订单号': index === 0 ? order.number : '',
            [`订单金额(${order.TotalAmountLocalCurrency})`]: index === 0 ? Number(order.TotalAmountLocalValue) : '',
            '国家': index === 0 ? order.CountryName : '',
            '产品名': item.productName || '-',
            'Seller SKU': item.sellerSku || '-',
            'Jumia SKU': item.jumia_sku || '-',
            [`支付金额(${order.TotalAmountLocalCurrency})`]: Number(item.paidPriceLocal) || 0,
            [`运费金额(${order.TotalAmountLocalCurrency})`]: Number(item.shippingAmountLocal) || 0,
            '订单状态': item.status || '',
            '收货人': index === 0 ? `${order.ShippingFirstName} ${order.ShippingLastName}`.trim() : '',
            '创建时间': index === 0 ? formatTime(order.createdAt) : '',
            '更新时间': index === 0 ? formatTime(order.updatedAt) : '',
          });
        });
      }
    });

    console.log('📝 最终导出 Excel 行数:', exportRows.length);

    // 生成 Excel
    const ws = XLSX.utils.json_to_sheet(exportRows);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, '订单明细');

    const fileName = `订单明细_${formatDateForFilename(new Date())}.xlsx`;
    XLSX.writeFile(wb, fileName);

    // ✅ 成功提示（带统计信息）
    message.success(
        `导出成功！共 ${allOrders.length} 个订单，${totalItems} 个商品项\n` +
        `订单金额总和: ${totalOrderAmount.toFixed(2)} ${allOrders[0]?.TotalAmountLocalCurrency || ''}, ` +
        `支付金额总和: ${totalPaidAmount.toFixed(2)} ${allOrders[0]?.TotalAmountLocalCurrency || ''}, ` +
        `运费金额总和: ${totalShippingAmount.toFixed(2)} ${allOrders[0]?.TotalAmountLocalCurrency || ''}`
    );

    console.log('💰 汇总结果:', {
      订单金额总和: totalOrderAmount,
      支付金额总和: totalPaidAmount,
      运费金额总和: totalShippingAmount
    });
  } catch (err) {
    message.error('导出失败');
    console.error(err);
  } finally {
    exportLoading.value = false;
  }
};


// 获取所有订单数据（自动翻页）
const fetchAllOrderData = async () => {
  const result = [];
  const pageSize = 200;
  let page = 1;
  const maxPages = 100; // 最多支持 10000 条
  let totalCount = null;

  while (page <= maxPages) {
    try {
      const params = {
        page,
        limit: pageSize,
        key: searchKeyword.value || undefined,
        country_name: selectedCountry.value || undefined,
        status: selectedStatus.value || undefined,
        start_date: startDate.value || undefined,
        end_date: endDate.value || undefined,
      };

      console.log('[导出] 请求页码:', page, '参数:', params);

      const res = await request.get('/user/my_order', { params });

      if (res.data.code !== 200) {
        message.error(`第 ${page} 页失败: ${res.data.msg}`);
        break;
      }

      const { list, count } = res.data.data;

      if (!Array.isArray(list)) {
        message.error('list 不是数组');
        break;
      }

      // 如果是第一次请求，记录 totalCount
      if (totalCount === null) {
        totalCount = count;
        console.log('[导出] 总计订单数:', totalCount);
      }

      result.push(...list);
      console.log(`[导出] 第 ${page} 页，返回 ${list.length} 个订单，累计 ${result.length}/${totalCount}`);

      // 核心修复：如果当前页返回的数量为 0 或累计数量已达 totalCount，则停止
      if (list.length === 0 || result.length >= totalCount) {
        console.log('[导出] 结束：当前页无数据或累计数量已达 total count');
        break;
      }

      // 继续下一页
      page++;
    } catch (err) {
      message.error(`请求失败: ${err.message}`);
      console.error(err);
      break;
    }
  }

  if (page > maxPages) {
    message.warning(`数据过多，仅导出前 ${maxPages * 100} 条`);
  }

  console.log('✅ fetchAllOrderData 完成，共获取订单数:', result.length);
  return result;
};


// 格式化时间用于文件名
const formatDateForFilename = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};
// ========== 主表列定义 ==========
const columns = [
  // ✅ 展开控制器列（必须放在第一列）
  {
    key: 'action',
    width: 20,
    // 不设置 title，避免表头多出一列文字
  },
  { title: '订单号', dataIndex: 'number', key: 'number', width: 120 },
  { title: '国家', dataIndex: 'CountryName', key: 'country', width: 100 },
  {
    title: '订单金额',
    dataIndex: 'TotalAmountValue',
    key: 'amount',
    width: 120,
    customRender: ({ record }) => `${record.TotalAmountLocalValue} ${record.TotalAmountLocalCurrency}`.trim(),
  },
  // { title: '订单状态', dataIndex: 'status', key: 'status', width: 100 },
  {
    title: '收货人',
    dataIndex: 'ShippingFirstName',
    key: 'shippingName',
    width: 120,
    customRender: ({ record }) => `${record.ShippingFirstName} ${record.ShippingLastName}`.trim(),
  },
  {
    title: '创建时间',
    dataIndex: 'createdAt',
    key: 'createdAt',
    width: 160,
    customRender: ({ value }) => formatTime(value),
  },
  {
    title: '更新时间',
    dataIndex: 'updatedAt',
    key: 'updatedAt',
    width: 160,
    customRender: ({ value }) => formatTime(value),
  },
];

// ========== 子表列定义（订单项）==========
const itemColumns = [
  { title: '产品名', dataIndex: 'productName', width: 300 },
  { title: 'Seller SKU', dataIndex: 'sellerSku', width: 150 },
  { title: 'Jumia SKU', dataIndex: 'jumia_sku', width: 180 },
  { title: 'trackingUrl', dataIndex: 'trackingUrl', width: 180 },
  {
    title: '支付金额',
    dataIndex: 'paidPriceLocal',
    width: 100,
  },
  {
    title: '运费金额',
    dataIndex: 'shippingAmountLocal',
    width: 100,
  },
  { title: '状态', dataIndex: 'status', width: 100 },
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

// 货币格式化
const formatCurrency = (value, currency = 'USD') => {
  if (typeof value !== 'number') return '-';
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: currency,
    minimumFractionDigits: 2,
  }).format(value);
};

// 分页状态
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
    fetchData(current, size, searchKeyword.value, currentSort.value);
  },
});

// ✅ 启用展开功能（只启用，不定义渲染逻辑）
const expandable = {};

const loading = ref(false);
const tableData = ref([]);
const searchKeyword = ref('');
const currentSort = ref('');

// 获取数据
const fetchData = async (page = 1, pageSize = 10, keyword = searchKeyword.value, sort = '',country_name=selectedCountry.value,status =selectedStatus.value,start_date = startDate.value,end_date = endDate.value) => {
  loading.value = true;
  try {
    const res = await request.get('/user/my_order', {
      params: {
        page,
        limit: pageSize,
        key: keyword,
        sort,
        country_name:country_name,
        status:status,
        start_date,
        end_date
      },
    });

    if (res.data.code === 200) {
      const { list, count } = res.data.data;

      if (!list || list.length === 0) {
        message.info('暂无数据');
        tableData.value = [];
        pagination.total = 0;
        return;
      }

      // 确保 orderItems 是数组
      tableData.value = list.map(order => ({
        ...order,
        orderItems: Array.isArray(order.orderItems) ? order.orderItems : []
      }));
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

// 搜索
const handleSearch = () => {
  pagination.current = 1;
  fetchData(1, pagination.pageSize, searchKeyword.value, currentSort.value,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);
};

// 表格变化（分页/排序）
const handleTableChange = (pag, filters, sorter) => {
  let sort = '';
  // if (sorter?.order && sorter.field === 'paid_price') {
  //   sort = `paid_price  ${sorter.order === 'ascend' ? 'asc' : 'desc'}`;
  // }
  currentSort.value = sort;

  fetchData(pag.current, pag.pageSize, searchKeyword.value, sort,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);
};


// 根据国家筛选
const selectedCountry = ref(null);
const countryOptions = ref([
  { label: '加纳', value: 'Ghana' },
  { label: '尼日利亚', value: 'Nigeria' },
  { label: '肯尼亚', value: 'Kenya' },
  // 其他国家...
]);
const handleCountryChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  fetchData(1, pagination.pageSize, searchKeyword.value, currentSort.value,value,selectedStatus.value,startDate.value,endDate.value);
};

// 根据状态筛选
const selectedStatus = ref(null);
const statusOptions = ref([
  { label: '已发货', value: 'SHIPPED' },
  { label: '交付', value: 'DELIVERED' },
  { label: '失败', value: 'FAILED' },
  { label: '取消', value: 'CANCELED' },
  { label: '退货', value: 'RETURNED' },
  // 其他国家...
]);
const handleStatusChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  fetchData(1, pagination.pageSize, searchKeyword.value, currentSort.value,selectedCountry.value,value,startDate.value,endDate.value);
};
// 监听搜索框
watch(searchKeyword, (val) => {
  if (val === '') {
    fetchData(1, pagination.pageSize, '', currentSort.value,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);
  }
});


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
  fetchData(1, pagination.pageSize, searchKeyword.value, currentSort.value,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);
};

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
// 初始化
onMounted(() => {
  updateScroll()
  window.addEventListener('resize', updateScroll)
  fetchData(pagination.current, pagination.pageSize, searchKeyword.value, currentSort.value);
});
onBeforeUnmount(() => {
  window.removeEventListener('resize', updateScroll)
})
</script>

<style scoped>
/* 优化展开行样式 */
.ant-table-expanded-row .ant-table {
  margin: 16px 0;
  background-color: #f9f9f9;
}

/* 可选：展开按钮 hover 效果 */
.ant-table-row-expand-icon {
  transition: transform 0.3s ease;
}
.ant-table-row-expand-icon-expanded {
  transform: rotate(90deg);
}
</style>