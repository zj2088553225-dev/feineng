
<template>
  <a-card style="padding: 10px" title="结算明细页">
    <!-- 搜索区域 -->
    <a-row :gutter="16" style="margin-bottom: 16px;">
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
    <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        row-key="id"
        :pagination="pagination"
        @change="handleTableChange"
        :expandable="expandable"
        :scroll="scroll"
        :key="tableKey"
    >
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
  </a-card>
</template>
<script setup>
// ========== 表格列定义（不变）==========
import {h, onBeforeUnmount, onMounted, reactive, ref, watch} from "vue";
import request from "@/utils/request.js";
import {message} from "ant-design-vue";
import dayjs from 'dayjs';
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
    fetchData(current, size,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);

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
  fetchData(1, pagination.pageSize,selectedCountry.value,selectedStatus.value, startDate.value,
      endDate.value);
};
const tableKey = ref(1);
// 获取数据
const fetchData = async (page = 1, pageSize = 10,country_code=selectedCountry.value,paid_status =selectedStatus.value, start_date = startDate.value,
                         end_date = endDate.value) => {
  tableData.value = []
  loading.value = true;
  try {
    const res = await request.get('/user/my_settlement', {
      params: {
        page,
        limit: pageSize,
        country_code:country_code,
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


  fetchData(pag.current, pag.pageSize,selectedCountry.value,selectedStatus.value,startDate.value,endDate.value);
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
  fetchData(1, pagination.pageSize,value,selectedStatus.value,startDate.value, endDate.value);
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
  fetchData(1, pagination.pageSize, selectedCountry.value,value,startDate.value, endDate.value);
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
  // 第一次加载
  fetchData(pagination.current, pagination.pageSize);
});
onBeforeUnmount(() => {
  window.removeEventListener('resize', updateScroll)
})


</script>



<style scoped>

</style>