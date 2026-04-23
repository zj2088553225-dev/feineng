<template>
  <a-card style="padding: 10px" title="合伙人头程国际物流明细页">
    <!-- 搜索区域 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <!-- 合伙人下拉框 -->
      <a-col :span="2">
        <a-select
            v-model:value="selectedPartnerId"
            :options="upUserIDOptions"
            placeholder="选择合伙人"
            allowClear
            style="width: 100%"
            @change="handlePartnerChange"
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
        <a-button style="margin-left: 10px" @click="handleExport" :loading="exportLoading">
          导出 Excel
        </a-button>
      </a-col>
    </a-row>

    <!-- 数据表格 -->
    <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        row-key="id"
        :pagination="pagination"
        @change="handleTableChange"
        :scroll="{ x: 2000, y: 600 }"
        :expanded-row-keys="expandedRowKeys"
        @expand="handleExpand"
    >
      <!-- 展开行内容 -->
      <template #expandedRowRender="{ record }">
        <div style="padding: 16px; background: #f9f9f9; border-radius: 8px; max-width: 1600px">
          <!-- 商品明细 -->
          <h4 style="margin-bottom: 12px; color: #1890ff">🛍️ 商品明细</h4>
          <a-table
              :columns="cargoColumns"
              :data-source="record._cargoList"
              :pagination="false"
              size="small"
              :row-key="(r, i) => `${r.commoditySku}-${i}`"
          />
          <!-- 物流轨迹 -->
          <h4 style="margin-bottom: 12px; color: #1890ff">📦 物流轨迹</h4>
          <a-timeline mode="left" style="margin-left: 20px; margin-bottom: 24px">
            <a-timeline-item
                v-for="t in record._trajectoryData"
                :key="t.opLink + t.timestamp"
                :label="t.timestamp"
            >
              {{ t.opLink }}
            </a-timeline-item>
          </a-timeline>

        </div>
      </template>
    </a-table>
  </a-card>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue';
import request from '@/utils/request.js';
import { message } from 'ant-design-vue';
import { SearchOutlined } from '@ant-design/icons-vue';
import * as XLSX from 'xlsx'; // npm install xlsx

// ==================== 数据响应式定义 ====================
const loading = ref(false);
const exportLoading = ref(false);
const tableData = ref([]);
const searchKeyword = ref('');
const Seleteddate = ref();
const startDate = ref('');
const endDate = ref('');
const upUserIDOptions = ref([]);
const selectedPartnerId = ref(null);
const expandedRowKeys = ref([]); // 控制展开的行

// ==================== 分页配置 ====================
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
    fetchData(current, size, searchKeyword.value, selectedPartnerId.value, startDate.value, endDate.value);
  },
});

// ==================== 表格列定义 ====================
const columns = [
  { title: '订单ID', dataIndex: 'id', key: 'id', width: 180 },
  { title: 'HBL编号', dataIndex: 'hbl', key: 'hbl', width: 150 },
  { title: '创建时间', dataIndex: 'createTime', key: 'createTime', width: 180 },
  { title: '渠道名称', dataIndex: 'channelName', key: 'channelName', width: 200 },
  { title: '状态', dataIndex: 'statusName', key: 'statusName', width: 120 },
  { title: '总件数', dataIndex: 'totalCount', key: 'totalCount', width: 100 },
  { title: '净重(kg)', dataIndex: 'totalNetWeight', key: 'totalNetWeight', width: 120 },
  { title: '毛重(kg)', dataIndex: 'totalRoughWeight', key: 'totalRoughWeight', width: 120 },
  { title: '体积(CBM)', dataIndex: 'totalCBM', key: 'totalCBM', width: 120 },
  { title: '入仓毛重(kg)', dataIndex: 'totalRoughWeightStorage', key: 'totalRoughWeightStorage', width: 120 },
  { title: '入仓体积(CBM)', dataIndex: 'totalCBMStorage', key: 'totalCBMStorage', width: 120 },
  { title: '商品总数', dataIndex: 'totalCommodityCount', key: 'totalCommodityCount', width: 120 },
];

// 商品子表格列
const cargoColumns = [
  { title: 'PO号', dataIndex: 'po', width: 120 },
  { title: '店铺名称', dataIndex: 'shopName', width: 150 },
  { title: '商品名称', dataIndex: 'commodityCname', width: 180 },
  { title: '商品SKU', dataIndex: 'commoditySku', width: 180 },
  { title: '店铺SKU', dataIndex: 'shopSku', width: 180 },
  { title: '数量', dataIndex: 'count', width: 100 },
  { title: '净重(kg)', dataIndex: 'netWeight', width: 100 },
  { title: '毛重(kg)', dataIndex: 'roughWeight', width: 100 },
  { title: '尺寸(cm)', dataIndex: 'dimensions', width: 130 },
];

// ==================== 日期处理 ====================
const onDateChange = (dates) => {
  if (dates && dates.length === 2) {
    startDate.value = dates[0].format('YYYY-MM-DD');
    endDate.value = dates[1].format('YYYY-MM-DD');
  } else {
    startDate.value = '';
    endDate.value = '';
  }
  pagination.current = 1;
  fetchData();
};

// ==================== 获取数据 ====================
const fetchData = async (
    page = pagination.current,
    pageSize = pagination.pageSize,
    keyword = searchKeyword.value,
    partnerId = selectedPartnerId.value,
    start_date = startDate.value,
    end_date = endDate.value
) => {
  loading.value = true;
  try {
    const res = await request.get('/user/wuliu', {
      params: { page, limit: pageSize, key: keyword, partner_id: partnerId, start_date, end_date },
    });

    if (res.data.code === 200) {
      const { list, count } = res.data.data;
      if (!list || list.length === 0) {
        message.info('暂无数据');
        tableData.value = [];
        pagination.total = 0;
        return;
      }

      // 处理数据：添加 _trajectoryData 和 _cargoList 用于展开行
      const processedData = list.map(item => {
        const trajectoryData = (Array.isArray(item.trajectories) ? item.trajectories : [])
            .filter(t => t && t.timestamp) // 👈 过滤掉 null、undefined 或无 timestamp 的项
            .sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp))
            .map(t => ({
              opLink: t.opLink || '未知操作',
              timestamp: String(t.timestamp) // 转字符串
                  .replace('T', ' ')
                  .replace('+08:00', '')
                  .replace(/\.\d+/, ''), // 可选：去掉毫秒部分
            }));

        const cargoList = item.cargos?.flatMap(cargo =>
            cargo.packages.flatMap(pkg =>
                pkg.commodities.map(commodity => ({
                  po: cargo.po,
                  shopName: cargo.shopName,
                  commodityCname: commodity.commodityCname,
                  commoditySku: commodity.commoditySku,
                  shopSku: commodity.shopSku,
                  count: pkg.totalCommodityCount,
                  netWeight: pkg.netWeight,
                  roughWeight: pkg.roughWeight,
                  dimensions: `${pkg.length}×${pkg.width}×${pkg.high} cm`,
                }))
            )
        ) || [];

        return {
          ...item,
          _trajectoryData: trajectoryData,
          _cargoList: cargoList,
        };
      });

      tableData.value = processedData;
      pagination.total = count;
      pagination.current = page;
    } else {
      message.error(res.data.msg || '获取数据失败');
    }
  } catch (err) {
    console.error('【fetchData 错误详情】', err); // 👈 关键！看具体哪一行报错
    message.error('请求失败，请检查网络');
    console.error(err);
  } finally {
    loading.value = false;
  }
};

// ==================== 搜索与分页 ====================
const handleSearch = () => {
  pagination.current = 1;
  fetchData(1, pagination.pageSize, searchKeyword.value, selectedPartnerId.value, startDate.value, endDate.value);
};

const handleTableChange = (pag) => {
  fetchData(pag.current, pag.pageSize, searchKeyword.value, selectedPartnerId.value, startDate.value, endDate.value);
};

watch(searchKeyword, (val) => {
  if (val === '') {
    fetchData(1, pagination.pageSize, '', selectedPartnerId.value, startDate.value, endDate.value);
  }
});

// ==================== 合伙人下拉 ====================
const getbindUser = async () => {
  if (upUserIDOptions.value.length > 0) return;

  try {
    const res = await request.get('/user/user_name_list');
    if (res.data.code === 200 && Array.isArray(res.data.data)) {
      upUserIDOptions.value = res.data.data.map(user => ({
        value: user.id,
        label: user.user_name,
      }));
    } else {
      message.info(res.data.msg || '数据为空');
    }
  } catch (err) {
    message.error('请求合伙人列表失败');
    console.error(err);
  }
};
getbindUser();

const handlePartnerChange = (value) => {
  selectedPartnerId.value = value;
  pagination.current = 1;
  fetchData( pagination.current, pagination.pageSize,searchKeyword.value, value, startDate.value, endDate.value);
};

// ==================== 导出 Excel ====================
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
  const maxPages = 100;
  let totalCount = null;

  while (page <= maxPages) {
    try {
      const params = {
        page,
        limit: pageSize,
        key: searchKeyword.value || undefined,
        partner_id: selectedPartnerId.value || undefined,
        start_date: startDate.value || undefined,
        end_date: endDate.value || undefined,
      };

      const res = await request.get('/user/wuliu', { params });
      if (res.data.code !== 200) {
        message.error(`第 ${page} 页请求失败：${res.data.msg}`);
        break;
      }

      const { list, count } = res.data.data;
      if (!Array.isArray(list)) break;

      if (totalCount === null) totalCount = count;
      allData.push(...list);

      if (list.length === 0 || allData.length >= totalCount) break;
      page++;
    } catch (err) {
      message.error(`请求异常：${err.message}`);
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
      message.warning('没有可导出的数据');
      exportLoading.value = false;
      return;
    }

    // 构造导出数据
    const exportRows = allTransactions.flatMap(item =>
        item.cargos.flatMap(cargo =>
            cargo.packages.flatMap(pkg =>
                pkg.commodities.map(commodity => ({
                  订单ID: item.id,
                  HBL编号: item.hbl,
                  创建时间: item.createTime,
                  渠道名称: item.channelName,
                  状态: item.statusName,
                  总件数: item.totalCount,
                  净重kg: item.totalNetWeight,
                  毛重kg: item.totalRoughWeight,
                  体积CBM: item.totalCBM,
                  入仓毛重kg: item.totalRoughWeightStorage,
                  入仓体积CBM: item.totalCBMStorage,
                  商品总数: item.totalCommodityCount,
                  PO号: cargo.po,
                  店铺名称: cargo.shopName,
                  商品名称: commodity.commodityCname,
                  商品SKU: commodity.commoditySku,
                  店铺SKU: commodity.shopSku,
                  包裹数量: pkg.totalCommodityCount,
                  尺寸cm: `${pkg.length}×${pkg.width}×${pkg.high}`,
                }))
            )
        )
    );

    const ws = XLSX.utils.json_to_sheet(exportRows);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, '物流订单明细');

    const fileName = `物流订单_${formatDateForFilename(new Date())}.xlsx`;
    XLSX.writeFile(wb, fileName);

    message.success(`导出成功！共 ${allTransactions.length} 单，${exportRows.length} 行商品数据`);
  } catch (error) {
    message.error('导出失败');
    console.error(error);
  } finally {
    exportLoading.value = false;
  }
};

// ==================== 展开行控制 ====================
const handleExpand = (expanded, record) => {
  if (expanded) {
    expandedRowKeys.value = [record.id];
  } else {
    expandedRowKeys.value = [];
  }
};

// ==================== 初始化 ====================
onMounted(() => {
  fetchData(1, pagination.pageSize, '');
});
</script>

<style scoped>

</style>