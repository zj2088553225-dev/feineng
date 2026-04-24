<template>
  <a-card title="Kilimall 物流订单" style="padding: 10px">
    <a-form layout="inline" style="margin-bottom: 16px">
      <a-form-item>
        <a-input
          v-model:value="searchKeyword"
          allow-clear
          placeholder="支持订单号 / 运单号搜索"
          style="width: 280px"
          @pressEnter="handleSearch"
        />
      </a-form-item>
      <a-form-item>
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </a-form-item>
      <a-form-item>
        <a-button @click="handleReset">重置</a-button>
      </a-form-item>
      <a-form-item>
        <a-button @click="fetchData">刷新</a-button>
      </a-form-item>
    </a-form>

    <a-table
      :columns="columns"
      :data-source="tableData"
      :loading="loading"
      :pagination="pagination"
      row-key="id"
      :scroll="{ x: 1400 }"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'productInfo'">
          <div class="product-cell">
            <img v-if="record.imageUrl" :src="record.imageUrl" alt="product" class="product-image" />
            <div>
              <div class="product-name">{{ record.productName || '-' }}</div>
              <div class="product-sku">SKU：{{ record.sellerSku || '-' }}</div>
            </div>
          </div>
        </template>
        <template v-else-if="column.key === 'trackingNumber'">
          <a :href="record.trackingUrl" target="_blank" rel="noreferrer" v-if="record.trackingUrl">
            {{ record.trackingNumber || '-' }}
          </a>
          <span v-else>{{ record.trackingNumber || '-' }}</span>
        </template>
        <template v-else-if="column.key === 'region'">
          {{ formatRegion(record) }}
        </template>
        <template v-else-if="column.key === 'netProfit'">
          {{ formatMoney(record.netProfit, record.currency) }}
        </template>
        <template v-else-if="column.key === 'updatedAt'">
          {{ formatDate(record.updatedAt) }}
        </template>
      </template>
    </a-table>
  </a-card>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import request from '@/utils/request.js'

const loading = ref(false)
const searchKeyword = ref('')
const tableData = ref([])

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  pageSizeOptions: ['10', '20', '50', '100'],
  showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
})

const columns = [
  { title: '订单号', dataIndex: 'orderNumber', key: 'orderNumber', width: 180 },
  { title: '运单号', dataIndex: 'trackingNumber', key: 'trackingNumber', width: 220 },
  { title: '商品信息', key: 'productInfo', width: 340 },
  { title: '当前物流状态', dataIndex: 'status', key: 'status', width: 160 },
  { title: '订单状态', dataIndex: 'orderStatus', key: 'orderStatus', width: 160 },
  { title: '净利润', dataIndex: 'netProfit', key: 'netProfit', width: 140 },
  { title: '国家 / 区域', key: 'region', width: 240 },
  { title: '更新时间', dataIndex: 'updatedAt', key: 'updatedAt', width: 180 },
]

const formatDate = (value) => {
  if (!value) return '-'
  return dayjs(value).isValid() ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : value
}

const formatRegion = (record) => {
  return [record.shippingCountryName || record.countryName, record.shippingRegion, record.shippingCity]
    .filter(Boolean)
    .join(' / ') || '-'
}

const formatMoney = (value, currency) => {
  const amount = Number(value || 0).toFixed(2)
  return currency ? `${amount} ${currency}` : amount
}

const fetchData = async (
  page = pagination.current,
  pageSize = pagination.pageSize,
  key = searchKeyword.value,
) => {
  loading.value = true
  try {
    const res = await request.get('/logistics/list', {
      params: {
        page,
        limit: pageSize,
        key,
      },
    })

    if (res.data.code !== 200) {
      message.error(res.data.msg || '获取物流订单失败')
      return
    }

    const payload = res.data.data || {}
    tableData.value = Array.isArray(payload.list) ? payload.list : []
    pagination.total = payload.count || 0
    pagination.current = page
    pagination.pageSize = pageSize
  } catch (error) {
    message.error('请求失败，请检查网络')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.current = 1
  fetchData(1, pagination.pageSize, searchKeyword.value)
}

const handleReset = () => {
  searchKeyword.value = ''
  pagination.current = 1
  fetchData(1, pagination.pageSize, '')
}

const handleTableChange = (pag) => {
  fetchData(pag.current, pag.pageSize, searchKeyword.value)
}

watch(searchKeyword, (value) => {
  if (value === '') {
    fetchData(1, pagination.pageSize, '')
  }
})

onMounted(() => {
  fetchData(1, pagination.pageSize, '')
})
</script>

<style scoped>
.product-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.product-image {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: 4px;
  border: 1px solid #f0f0f0;
}

.product-name {
  color: #222;
  line-height: 1.4;
}

.product-sku {
  margin-top: 4px;
  color: #999;
  font-size: 12px;
}
</style>
