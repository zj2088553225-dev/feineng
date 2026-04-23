<template>
  <a-card title="数据统计" style="padding: 10px">
    <!-- 筛选区域 -->
    <a-row :gutter="16" style="margin-bottom: 16px;">
      <a-col :span="2">
        <a-select
            v-model:value="selectedCountry"
            :options="countryOptions"
            placeholder="选择国家"
            allowClear
            style="width: 100%;"
            @change="onFilterChange"
        />
      </a-col>
      <a-col :span="8">
        <a-range-picker
            v-model:value="selectedDate"
            :show-time="false"
            format="YYYY-MM-DD"
            :ranges="{
            '最近7天': [dayjs().subtract(6, 'day'), dayjs()],
            '最近30天': [dayjs().subtract(29, 'day'), dayjs()]
          }"
            @change="onFilterChange"
            style="width: 100%;"
        />
      </a-col>
    </a-row>

    <!-- 管理员：图表模式 -->
    <template v-if="isAdmin">
      <a-card title="平台订单分布" size="small" style="margin-bottom: 20px;">
        <div ref="platformChartRef" style="width: 100%; height: 400px;"></div>
      </a-card>

      <a-card title="社媒销量分布" size="small">
        <div ref="socialChartRef" style="width: 100%; height: 400px;"></div>
      </a-card>
    </template>

    <!-- 普通用户：数据卡片 -->
    <template v-else>
      <a-row :gutter="16">
        <a-col :span="12">
          <a-card :title="`社媒销售额 (${currencySymbol})`" style="text-align: center;">
            <h1 style="color: #1890ff; margin: 0;">
              {{ formatCurrency(socialAmount) }}
            </h1>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card title="今日订单数" style="text-align: center;">
            <h1 style="color: #52c41a; margin: 0;">
              {{ todaySalesCount }} <span style="font-size: 16px; color: #999;">单</span>
            </h1>
          </a-card>
        </a-col>
      </a-row>
    </template>
  </a-card>
</template>

<script setup>
import { ref, onMounted, nextTick, computed } from 'vue'
import request from '@/utils/request.js'
import { message } from 'ant-design-vue'
import * as echarts from 'echarts'
import dayjs from 'dayjs'
import { useStore } from '@/stores/userStore'
const userStore = useStore()
const userRole = userStore.userInfo?.role || 'user'
const isAdmin = ref(userRole === 'admin')

// ========== 国家与货币映射 ==========
const countryCurrencyMap = {
  Kenya: 'KES',
  Nigeria: 'NGN',
  Ghana: 'GHS'
}

const currencySymbolMap = {
  KES: 'KES',
  NGN: 'NGN',
  GHS: 'GHS'
}

const selectedCountry = ref('Ghana')
const countryOptions = ref([
  { label: '加纳', value: 'Ghana' },
  { label: '尼日利亚', value: 'Nigeria' },
  { label: '肯尼亚', value: 'Kenya' },
])

// 当前货币
const currentCurrencyCode = computed(() => countryCurrencyMap[selectedCountry.value] || 'GHS')
const currencySymbol = computed(() => currencySymbolMap[currentCurrencyCode.value] || '₵')

// ========== 日期选择 ==========
// ✅ 默认为今天
const today = dayjs()
const selectedDate = ref([today, today]) // 默认单日：今天
const startDate = ref(today.format('YYYY-MM-DD'))
const endDate = ref(today.format('YYYY-MM-DD'))

const onFilterChange = () => {
  if (selectedDate.value && selectedDate.value.length === 2) {
    startDate.value = selectedDate.value[0].format('YYYY-MM-DD')
    endDate.value = selectedDate.value[1].format('YYYY-MM-DD')
  } else {
    // 清空后也默认为今天
    const today = dayjs()
    selectedDate.value = [today, today]
    startDate.value = today.format('YYYY-MM-DD')
    endDate.value = today.format('YYYY-MM-DD')
  }
  fetchData()
}

// ========== 普通用户数据 ==========
const socialAmount = ref(0)
const todaySalesCount = ref(0)

const formatCurrency = (value) => {
  const num = Number(value) || 0
  const formattedNumber = new Intl.NumberFormat().format(num)
  return `${currencySymbol.value}${formattedNumber}`
}

// ========== 图表相关 ==========
const platformChartRef = ref(null)
const socialChartRef = ref(null)
let platformChartInstance = null
let socialChartInstance = null

const chartData = ref({
  platform: { legendData: [], seriesData: [] },
  social: { legendData: [], seriesData: [] }
})

const initCharts = () => {
  nextTick(() => {
    if (platformChartRef.value && !platformChartInstance) {
      platformChartInstance = echarts.init(platformChartRef.value)
    }
    if (socialChartRef.value && !socialChartInstance) {
      socialChartInstance = echarts.init(socialChartRef.value)
    }
    renderPlatformChart()
    renderSocialChart()
  })
}

const renderPlatformChart = () => {
  if (!platformChartInstance) return
  const option = {
    title: {
      text: '平台订单分布',
      subtext: `${selectedCountry.value} | ${startDate.value} 至 ${endDate.value}`,
      left: 'center'
    },
    tooltip: { trigger: 'item', formatter: '{b}: {c} 单' },
    legend: {
      type: 'scroll',
      orient: 'vertical',
      right: 10,
      top: 20,
      bottom: 20,
      data: chartData.value.platform.legendData
    },
    series: [
      {
        name: '订单数',
        type: 'pie',
        radius: '55%',
        center: ['40%', '50%'],
        data: chartData.value.platform.seriesData,
        emphasis: {
          itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' }
        },
        label: { show: true, formatter: '{b}: {c} 单' }
      }
    ]
  }
  platformChartInstance.setOption(option, true)
}

const renderSocialChart = () => {
  if (!socialChartInstance) return
  const option = {
    title: {
      text: '社媒销量分布',
      subtext: `${selectedCountry.value} | ${currencySymbol.value}`,
      left: 'center'
    },
    tooltip: { trigger: 'item', formatter: `{b}: {c} (${currencySymbol.value})` },
    legend: {
      type: 'scroll',
      orient: 'vertical',
      right: 10,
      top: 20,
      bottom: 20,
      data: chartData.value.social.legendData
    },
    series: [
      {
        name: '销售额',
        type: 'pie',
        radius: '55%',
        center: ['40%', '50%'],
        data: chartData.value.social.seriesData,
        emphasis: {
          itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' }
        },
        label: { show: true, formatter: '{b}: ' + currencySymbol.value + '{c}' }
      }
    ]
  }
  socialChartInstance.setOption(option, true)
}

// ========== 获取数据 ==========
const fetchData = async () => {
  try {
    const endpoint = isAdmin.value ? '/service/dashboard' : '/service/my_dashboard'

    const params = {
      country_name: selectedCountry.value,
      start_date: startDate.value,
      end_date: endDate.value
    }

    const res = await request.get(endpoint, { params })

    if (res.data.code === 200) {
      const data = res.data.data

      if (isAdmin.value) {
        const platform_orders = data.platform_orders || []
        chartData.value.platform.legendData = platform_orders.map(i => i.name)
        chartData.value.platform.seriesData = platform_orders.map(i => ({
          name: i.name,
          value: i.value
        }))
        todaySalesCount.value = platform_orders.reduce((sum, item) => sum + item.value, 0)
        renderPlatformChart()

        const social_sales = data.social_sales || []
        chartData.value.social.legendData = social_sales.map(i => i.name)
        chartData.value.social.seriesData = social_sales.map(i => ({
          name: i.name,
          value: i.value
        }))
        renderSocialChart()
      } else {
        socialAmount.value = data.social_amount || 0
        todaySalesCount.value = data.today_sales_count || 0
      }
    } else {
      message.error(res.data.msg || '数据获取失败')
    }
  } catch (err) {
    message.error('请求失败，请检查网络')
    console.error(err)
  }
}

// ========== 页面初始化 ==========
onMounted(() => {
  if (isAdmin.value) {
    initCharts()
  }
  fetchData()
})

// ========== 窗口大小适配 ==========
window.addEventListener('resize', () => {
  platformChartInstance?.resize()
  socialChartInstance?.resize()
})
</script>

<style scoped>
/* 可添加自定义样式 */
</style>