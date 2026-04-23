<template>
  <a-card style="padding: 10px" title="合伙人独立站订单审核明细页">
      <a-row :gutter="16" style="margin-bottom: 16px;">
        <!-- ✅ 新增：合伙人下拉框 -->
        <a-col :span="2">
          <a-select
              v-model:value="selectedPartnerId"
              :options="upUserIDOptions"
              placeholder="合伙人"
              allowClear
              style="width: 100%;"
              @change="handlePartnerChange"
          />
        </a-col>
        <a-col :span="2">
          <a-select
              v-model:value="selectedOrderStatus"
              :options="orderStatusOptions"
              placeholder="电话结果"
              allowClear
              style="width: 100%;"
              @change="handleOrderStatusChange"
          />
        </a-col>
        <a-col :span="2">
          <a-select
              v-model:value="selectedStatus"
              :options="statusOptions"
              placeholder="订单状态"
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
              placeholder="支持 jumiasku/订单编号/手机号 搜索"
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
            <!-- 上传按钮 -->
        </a-col>
          <a-col :span="4">
            <a-upload
                :before-upload="beforeUpload"
                :show-upload-list="false"
                :multiple="false"
            >
              <a-button style="margin-left: 10px" type="primary">
                上传CSV文件
              </a-button>
            </a-upload>

            <!-- 状态展示 -->
          <transition name="fade">
            <div v-if="uploadStatus" style="margin-top: 20px">
              <a-alert
                  :type="uploadStatus.type"
                  :message="uploadStatus.message"
                  :description="uploadStatus.description"
                  show-icon
                  closable
                  @close="uploadStatus = null"
              />
              <a-progress
                  v-if="uploadStatus.progress < 100 && uploadStatus.type === 'info'"
                  :percent="uploadStatus.progress"
                  status="active"
                  style="margin: 10px 0"
              />
            </div>
          </transition>
        </a-col>
      </a-row>
    <a-tabs v-model:activeKey="activeKey" type="card">
      <a-tab-pane key="gh" tab="加纳">
      <div style="overflow-x: auto; width: 100%;">
        <a-table
          :columns="columnsGH"
          :data-source="tableData"
          :loading="loading"
          row-key="gh_id"
          :pagination="pagination"
          @change="handleTableChange"
          :scroll="{ x: 2000 , y: 500}"
      >
      </a-table>
        </div>
      </a-tab-pane>
      <a-tab-pane key="ke" tab="肯尼亚" force-render><a-table
          :columns="columnsKE"
          :data-source="tableData"
          :loading="loading"
          row-key="ke_id"
          :pagination="pagination"
          @change="handleTableChange"
          :scroll="{ x: 2000 , y: 500}"
      >
      </a-table></a-tab-pane>
      <a-tab-pane key="ng" tab="尼日利亚"><a-table
          :columns="columnsNG"
          :data-source="tableData"
          :loading="loading"
          row-key="ng_id"
          :pagination="pagination"
          @change="handleTableChange"
          :scroll="{ x: 2000 , y: 500}"
      >
      </a-table></a-tab-pane>
    </a-tabs>
  </a-card>
</template>

<script setup>
import {onBeforeUnmount, onMounted, reactive, ref, watch} from 'vue';
import request from "@/utils/request.js";
import {message} from "ant-design-vue";
import {SearchOutlined} from "@ant-design/icons-vue";
import * as XLSX from "xlsx";




const exportLoading = ref(false);



// 获取所有自定义订单数据（自动翻页）
const fetchAllCustomizeOrderData = async (customize_order_type = activeKey.value) => {
  const result = [];
  const pageSize = 200;
  let page = 1;
  const maxPages = 500; // 最多 100 页，即 100000 条
  let totalCount = null;

  while (page <= maxPages) {
    try {
      const params = {
        page,
        limit: pageSize,
        customize_order_type,
        key: searchKeyword.value || undefined,
        person: selectedPartnerId.value || undefined,
        status: selectedStatus.value || undefined,
        start_date: startDate.value || undefined,
        end_date: endDate.value || undefined,
        order_status: selectedOrderStatus.value || undefined,
      };

      console.log(`[导出] 请求 ${customize_order_type} 第 ${page} 页`, params);
      const res = await request.get('/user/customize_order', { params });

      if (res.data.code !== 200) {
        message.error(`第 ${page} 页失败: ${res.data.msg}`);
        break;
      }

      const { list, count } = res.data.data;

      if (!Array.isArray(list)) {
        message.error('list 不是数组');
        break;
      }

      if (totalCount === null) {
        totalCount = count;
        console.log(`[导出] ${customize_order_type} 总计数据条数:`, totalCount);
      }

      result.push(...list);

      console.log(`[导出] 第 ${page} 页，返回 ${list.length} 条，累计 ${result.length}/${totalCount}`);

      // 结束条件：无数据或已达总数
      if (list.length === 0 || result.length >= totalCount) {
        break;
      }

      page++;
    } catch (err) {
      message.error(`请求失败: ${err.message}`);
      console.error(err);
      break;
    }
  }

  if (page > maxPages) {
    message.warning(`数据过多，仅导出前 ${maxPages * pageSize} 条`);
  }

  console.log(`✅ fetchAllCustomizeOrderData(${customize_order_type}) 完成，共获取 ${result.length} 条`);
  return result;
};
const handleExport = async () => {
  exportLoading.value = true;
  const type = activeKey.value; // 当前 tab 类型：gh / ke / ng

  if (!['gh', 'ke', 'ng'].includes(type)) {
    message.error('无效的订单类型');
    exportLoading.value = false;
    return;
  }

  try {
    const columns = exportColumns[type];
    const allData = await fetchAllCustomizeOrderData(type);

    console.log(`📊 共获取 ${type} 数据条数:`, allData.length);

    if (allData.length === 0) {
      message.warning('没有可导出的数据');
      exportLoading.value = false;
      return;
    }

    // 生成导出行：使用列的 title 作为 Excel 表头
    const exportRows = allData.map((record) => {
      const row = {};

      columns.forEach((col) => {
        if (!col.dataIndex) return; // 跳过无 dataIndex 的列（如操作列）

        let value = record[col.dataIndex];

        // 特殊字段处理（模拟 customRender）
        if (col.customRender) {
          const render = col.customRender({ value, record });

          // 如果返回的是字符串或数字，直接使用
          if (render !== null && render !== undefined && typeof render !== 'object') {
            value = render;
          } else {
            // 否则根据字段特殊处理
            if (col.dataIndex === 'full_name') {
              const first = record.first_name || '';
              const last = record.last_name || '';
              value = first || last ? `${first} ${last}`.trim() : '-';
            } else if (['date', 'order_date', 'time', 'call_date'].includes(col.dataIndex)) {
              value = value ? formatTime(value) : '';
            } else if (['amount', 'price'].some(f => col.dataIndex.includes(f))) {
              const num = parseFloat(value);
              value = isNaN(num) ? '0.00' : num.toFixed(2);
            } else if (col.dataIndex === 'ordered') {
              value = value === 'TRUE' ? '是' : '否';
            } else {
              value = render; // fallback
            }
          }
        } else {
          // 没有 customRender 的字段，直接取值
          value = value !== null && value !== undefined ? value : '';
        }

        // 使用列的 title 作为 Excel 列名
        row[col.title] = value;
      });

      return row;
    });

    console.log('📝 准备导出 Excel 行数:', exportRows.length);

    // 生成 Excel
    const ws = XLSX.utils.json_to_sheet(exportRows);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, `${type} 订单数据`);

    // ✅ 修复：正确使用 toUpperCase()
    const fileName = `${type}_订单数据_${formatDateForFilename(new Date())}.xlsx`;
    XLSX.writeFile(wb, fileName);

    message.success(`导出成功！共 ${allData.length} 条数据`);
  } catch (err) {
    message.error('导出失败');
    console.error(err);
  } finally {
    exportLoading.value = false;
  }
};
const formatDateForFilename = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};









// 响应式状态
const uploadStatus = ref(null) // { type, message, description, progress }
const taskID = ref(null)
const pollingInterval = ref(null)

// 文件上传前处理（阻止默认上传，手动发送）
const beforeUpload = (file) => {
  const type = getFileType(file.name)
  if (!type) {
    setUploadStatus('error', '上传失败', '无法识别文件类型，请使用 gh/ke/ng 命名文件，如 orders_ke.csv')
    return false
  }

  const formData = new FormData()
  formData.append('file', file)
  formData.append('type', type)

  startUpload(formData, type)
  return false // 阻止默认行为
}

// 发起上传请求
const startUpload = async (formData, fileType) => {
  setUploadStatus('info', '文件上传中...', '正在上传文件到服务器...', 10)

  try {
    const res = await request.post('/service/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })

    const data = res.data
    if (data.code === 200) {
      taskID.value = data.data.taskID
      setUploadStatus(
          'info',
          '后台处理中...',
          `任务ID: ${taskID.value}，正在异步处理数据...`,
          20
      )
      pollTaskStatus()
    } else {
      setUploadStatus('error', '上传失败', data.msg || '未知错误')
    }
  } catch (error) {
    const msg = error.response?.data?.msg || error.message
    setUploadStatus('error', '请求失败', `上传出错: ${msg}`)
  }

  return false
}

// 轮询任务状态
const pollTaskStatus = () => {
  if (pollingInterval.value) clearInterval(pollingInterval.value)

  pollingInterval.value = setInterval(async () => {
    try {
      const res = await request.get(`/service/status/${taskID.value}`)
      const task = res.data.data

      let progress = 0
      if (task.status === 'pending') progress = 10
      else if (task.status === 'running') progress = 50
      else if (task.status === 'completed') progress = 100
      console.log(res.data.data)
      let description = ''
      if (task.status === 'completed') {
        description = `✅ 导入 ${task.total} 条| 上传国家 ${task.type}  | 更新负责人 ${task.person_updated} 条 | 更新订单状态 ${task.status_updated} 条 | 订单号空 ${task.order_number_is_null} 条`
        message.success('数据处理完成！')
      } else if (task.status === 'failed') {
        description = `❌ 错误: ${task.errorMessage}`
      } else {
        description = `🕒 开始时间: ${new Date(task.startTime).toLocaleString()}`
      }

      setUploadStatus(
          task.status === 'completed'
              ? 'success'
              : task.status === 'failed'
                  ? 'error'
                  : 'info',
          `状态: ${formatStatus(task.status)}`,
          description,
          progress
      )

      // 停止轮询
      if (task.status === 'completed' || task.status === 'failed') {
        clearInterval(pollingInterval.value)
        pollingInterval.value = null
      }
    } catch (error) {
      const msg = error.response?.data?.msg || '任务不存在或网络错误'
      setUploadStatus('error', '查询状态失败', msg)
      clearInterval(pollingInterval.value)
      pollingInterval.value = null
    }
  }, 1000)
}

// 设置上传状态
const setUploadStatus = (type, message, description, progress = 0) => {
  uploadStatus.value = { type, message, description, progress }
}

// 格式化状态显示
const formatStatus = (status) => {
  const map = {
    pending: '等待中',
    running: '运行中',
    completed: '已完成',
    failed: '失败',
  }
  return map[status] || status
}

// 根据文件名推断国家类型
const getFileType = (filename) => {
  const lower = filename.toLowerCase()
  if (lower.includes('_gh') || lower.includes('gh_')) return 'gh'
  if (lower.includes('_ke') || lower.includes('ke_')) return 'ke'
  if (lower.includes('_ng') || lower.includes('ng_')) return 'ng'
  return null
}

// 组件销毁前清理定时器
onBeforeUnmount(() => {
  if (pollingInterval.value) {
    clearInterval(pollingInterval.value)
  }
})













const activeKey = ref('gh');

const columnsGH = [
  { title: 'ID', dataIndex: 'gh_id', key: 'gh_id',width: 90 },
  { title: '周数', dataIndex: 'week', key: 'week'  ,width: 90,},
  {
    title: '下单日期',
    dataIndex: 'date',
    key: 'date',
    width: 150,
    customRender: ({ value }) => formatTime(value)
  },
  { title: '合伙人', dataIndex: 'person',  width: 100,key: 'person' },
  { title: '订单编号', dataIndex: 'order_numb', key: 'order_numb',  width: 180,ellipsis: false },
  { title: '商品名称', dataIndex: 'product_name', key: 'product_name', ellipsis: false ,width: 280,},
  { title: 'Jumia SKU', dataIndex: 'jumia_sku', width: 280, key: 'jumia_sku', ellipsis: false },
  { title: '数量', dataIndex: 'qty', key: 'qty',width: 90, },
  {
    title: '金额 (GHS)',
    dataIndex: 'amount',
    key: 'amount',
    width: 80,
    customRender: ({ value }) => {
      const num = parseFloat(value);
      return isNaN(num) ? '0.00' : num.toFixed(2);
    }
  },
  {
    title: '商品链接',
    dataIndex: 'order_shop',
    key: 'order_shop',
    width: 280,
    ellipsis: true,
  },
  { title: '客户姓名',
    key: 'full_name',
    dataIndex: 'full_name',
    width: 100,
    customRender: ({ record }) => {
      const first = record.first_name || '';
      const last = record.last_name || '';
      return first || last ? `${first} ${last}`.trim() : '-';
    }
  },
  { title: '电话', dataIndex: 'phone_number', width: 100, key: 'phone_number' },
  { title: '邮箱', dataIndex: 'email_addr', width: 100, key: 'email_addr' },
  { title: '地址', dataIndex: 'address', width: 100, key: 'address' },
  { title: '城市', dataIndex: 'city', width: 100, key: 'city' },
  { title: '客服人员', dataIndex: 'agents', width: 100, key: 'agents' },
  { title: '是否已致电', dataIndex: 'called', width: 100, key: 'called' },
  { title: '电话结果', dataIndex: 'order_done', width: 100, key: 'order_done' },
  { title: '通话备注', dataIndex: 'call_comment', width: 100, key: 'call_comment' },
  { title: '最近自提点', dataIndex: 'closest_pus',  width: 100,key: 'closest_pus' },
  { title: 'Jumia订单号', dataIndex: 'order_number', width: 120, key: 'order_number' },
  { title: '客服备注', dataIndex: 'agent_comments', width: 100, key: 'agent_comments' },
  { title: '卖家备注', dataIndex: 'seller_comments', width: 100, key: 'seller_comments' },
  { title: 'WhatsApp联系', dataIndex: 'wa_contact_made', width: 100, key: 'wa_contact_made' },
  { title: '状态', dataIndex: 'status', width: 100, key: 'status' },
  {
    title: '物流跟踪',
    dataIndex: 'tracking_url',
    key: 'tracking_url',
    width: 300,
    ellipsis: false,
  },
];
const columnsKE = [
  { title: 'ID', dataIndex: 'ke_id', key: 'ke_id', width: 80 },
  { title: '周数', dataIndex: 'first', key: 'first', width: 100 },
  {
    title: '订单日期',
    dataIndex: 'order_date',
    key: 'order_date',
    width: 160,
    customRender: ({ value }) => formatTime(value)
  },
  { title: '合伙人', dataIndex: 'person', key: 'person', width: 120 },
  { title: '订单编号', dataIndex: 'id', key: 'id', width: 200 },
  { title: '商品名称', dataIndex: 'item_name', key: 'item_name', width: 200, ellipsis: false },
  { title: 'Jumia SKU', dataIndex: 'jumia_sku', key: 'jumia_sku', width: 280, ellipsis: false },
  {
    title: '单价 (KES)',
    dataIndex: 'price',
    key: 'price',
    width: 120,
    customRender: ({ value }) => {
      const num = parseFloat(value);
      return isNaN(num) ? '0.00' : num.toFixed(2);
    }
  },
  { title: '数量', dataIndex: 'qty', key: 'qty', width: 100 },
  {
    title: '客户姓名',
    dataIndex: 'customer_name',
    key: 'customer_name',
    width: 150
  },
  { title: '电话 1', dataIndex: 'phone_number', key: 'phone_number', width: 140 },
  { title: '电话 2', dataIndex: 'phone_number_2', key: 'phone_number_2', width: 140 },
  { title: '地址', dataIndex: 'address', key: 'address', width: 180, ellipsis: false },
  { title: '城市', dataIndex: 'city', key: 'city', width: 150 },
  { title: '地区', dataIndex: 'region', key: 'region', width: 120 },
  { title: '邮箱', dataIndex: 'email', key: 'email', width: 180, ellipsis: false },
  { title: '自提点', dataIndex: 'pick_up_stations', key: 'pick_up_stations', width: 180, ellipsis: false },
  { title: '销售客服', dataIndex: 'seller_agent', key: 'seller_agent', width: 120 },
  { title: '是否已致电', dataIndex: 'called', key: 'called', width: 120 },
  {
    title: '呼叫日期',
    dataIndex: 'call_date',
    key: 'call_date',
    width: 160,
    customRender: ({ value }) => formatTime(value)
  },
  { title: '是否接通', dataIndex: 'reached', key: 'reached', width: 120 },
  { title: '通话状态', dataIndex: 'order_status', key: 'order_status', width: 140 },
  { title: '配送方式', dataIndex: 'shipping_method', key: 'shipping_method', width: 140 },
  { title: 'Jumia 销售员', dataIndex: 'jumia_sales_agent_name', key: 'jumia_sales_agent_name', width: 150 },
  { title: '是否下单', dataIndex: 'order_placed', key: 'order_placed', width: 120 },
  { title: '正式订单号', dataIndex: 'order_number', key: 'order_number', width: 160 },
  { title: '卖家备注', dataIndex: 'seller_comment', key: 'seller_comment', width: 260 },
  {
    title: '是否已下单',
    dataIndex: 'ordered',
    key: 'ordered',
    width: 120,
    customRender: ({ value }) => value === 'TRUE' ? '是' : '否'
  },
  { title: '状态', dataIndex: 'status', key: 'status', width: 120 },
  {
    title: '物流跟踪',
    dataIndex: 'tracking_url',
    key: 'tracking_url',
    width: 300,
    slots: { customRender: 'tracking_url' } // 使用 slot 渲染链接
  },
];
const columnsNG = [
  { title: 'ID', dataIndex: 'ng_id', key: 'ng_id', width: 80 },
  { title: '周数', dataIndex: 'week', key: 'week', width: 100 },
  {
    title: '订单时间',
    dataIndex: 'time',
    key: 'time',
    width: 160,
    customRender: ({ value }) => formatTime(value)
  },
  { title: '合伙人', dataIndex: 'person', key: 'person', width: 120 },
  { title: '订单编号', dataIndex: 'id', key: 'id', width: 120 },
  { title: '商品名称', dataIndex: 'item_name', key: 'item_name', width: 250, ellipsis: true },
  { title: 'Jumia SKU', dataIndex: 'jumia_sku', key: 'jumia_sku', width: 280, ellipsis: true },
  {
    title: '单价 (NGN)',
    dataIndex: 'price',
    key: 'price',
    width: 120,
    customRender: ({ value }) => {
      const num = parseFloat(value);
      return isNaN(num) ? '0.00' : num.toFixed(2);
    }
  },
  { title: '数量', dataIndex: 'qty', key: 'qty', width: 100 },
  {
    title: '客户姓名',
    dataIndex: 'customer_name',
    key: 'customer_name',
    width: 150
  },
  { title: '电话 1', dataIndex: 'phone_number', key: 'phone_number', width: 140 },
  { title: '电话 2', dataIndex: 'phone_number_2', key: 'phone_number_2', width: 140 },
  { title: '地址', dataIndex: 'address', key: 'address', width: 200, ellipsis: true },
  { title: '城市', dataIndex: 'city', key: 'city', width: 150 },
  { title: '地区', dataIndex: 'region', key: 'region', width: 120 },
  { title: '邮箱', dataIndex: 'email', key: 'email', width: 180, ellipsis: true },
  { title: '自提点地址', dataIndex: 'pus_address', key: 'pus_address', width: 200, ellipsis: true },
  { title: '销售客服', dataIndex: 'seller_agent', key: 'seller_agent', width: 120 },
  { title: '是否已致电', dataIndex: 'called', key: 'called', width: 120 },
  {
    title: '呼叫日期',
    dataIndex: 'date',
    key: 'date',
    width: 160,
    customRender: ({ value }) => formatTime(value)
  },
  { title: '通话状态', dataIndex: 'order_status', key: 'order_status', width: 140 },
  { title: '是否接通', dataIndex: 'reached', key: 'reached', width: 120 },
  { title: '配送方式', dataIndex: 'shipping_method', key: 'shipping_method', width: 140 },
  { title: 'Jumia 销售员', dataIndex: 'jumia_agent_name', key: 'jumia_agent_name', width: 150 },
  { title: '是否下单', dataIndex: 'order_placed', key: 'order_placed', width: 120 },
  { title: '正式订单号', dataIndex: 'order_number', key: 'order_number', width: 160 },
  { title: '卖家备注', dataIndex: 'seller_comment', key: 'seller_comment', width: 200, ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 120 },
  {
    title: '物流跟踪',
    dataIndex: 'tracking_url',
    key: 'tracking_url',
    width: 300,
    slots: { customRender: 'tracking_url' } // 使用 slot 渲染链接
  },
];
// 列配置映射（用于导出）
const exportColumns = {
  gh: columnsGH,
  ke: columnsKE,
  ng: columnsNG,
};
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
    fetchData(activeKey.value,current, size, searchKeyword.value);
  },
})

// 切换页面
const handleTableChange = (pag, filters, sorter) => {


  fetchData(activeKey.value,pag.current, pag.pageSize, searchKeyword.value,selectedPartnerId.value,selectedOrderStatus.value,selectedStatus.value,startDate.value,endDate.value);
};

// 获取数据
const fetchData = async (customize_order_type=activeKey.value,page = 1, pageSize = 10, keyword = searchKeyword.value,person = selectedPartnerId.value,order_status=selectedOrderStatus.value,status =selectedStatus.value, start_date = startDate.value,
                         end_date = endDate.value) => {
  loading.value = true;
  try {
    const res = await request.get('/user/customize_order', {
      params: {
        page,
        limit: pageSize,
        key: keyword,
        person,
        order_status:order_status,
        status:status,
        start_date, // ✅ 前端传后端可识别的日期
        end_date,
        customize_order_type
      },
    });

    if (res.data.code === 200) {
      const { list, count } = res.data.data;
      // 👇 在这里打印！这是最源头的数据
      // console.log('Raw list from API:', JSON.parse(JSON.stringify(list))); // 使用 JSON 序列化避免引用问题
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



const upUserIDOptions = ref([]);     // 下拉选项
const selectedPartnerId = ref(null); // ✅ 初始为空，不选中任何人
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
        value: user.user_name,      // ✅ 假设 user.id 是 number 类型
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
  fetchData(activeKey.value,1, pagination.pageSize, searchKeyword.value, value,selectedOrderStatus.value,selectedStatus.value,startDate.value, endDate.value);
};

// 根据国家筛选
const selectedOrderStatus = ref("");
const orderStatusOptions = ref([
  { label: '确认', value: 'YES' },
  { label: '没有回复', value: 'No Answer' },
  { label: '其他', value: 'other' },
]);
const handleOrderStatusChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  fetchData(activeKey.value,1, pagination.pageSize, searchKeyword.value,selectedPartnerId.value,value,selectedStatus.value,startDate.value, endDate.value);
};

// 根据状态筛选
const selectedStatus = ref("");
const statusOptions = ref([
  { label: '运输中', value: 'SHIPPED' },
  { label: '已签收', value: 'DELIVERED' },
  { label: '已拒收', value: 'FAILED' },
  { label: '取消订单', value: 'CANCELED' },
  { label: '退货', value: 'RETURNED' },
  { label: '待定', value: 'PENDING' },
]);
const handleStatusChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  fetchData(activeKey.value,1, pagination.pageSize, searchKeyword.value, selectedPartnerId.value,selectedOrderStatus.value,value,startDate.value, endDate.value);
};

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
  fetchData(activeKey.value,1, pagination.pageSize, searchKeyword.value, selectedPartnerId.value,selectedOrderStatus.value,selectedStatus.value, startDate.value,
      endDate.value);
};

// 搜索功能
const searchKeyword = ref(''); // 搜索关键词
const handleSearch = () => {
  // 搜索时回到第一页
  pagination.current = 1;
  // 搜索时回到第一页
  fetchData(activeKey.value,1, pagination.pageSize, searchKeyword.value,selectedPartnerId.value,selectedOrderStatus.value,selectedStatus.value,startDate.value,endDate.value);
};
watch(searchKeyword, (val) => {
  if (val === '') {
    // 可选：清空搜索框时自动搜索（清空结果）
    fetchData(activeKey.value,1, pagination.pageSize, '',selectedPartnerId.value,selectedOrderStatus.value,selectedStatus.value,startDate.value,endDate.value);
  }
});
watch(activeKey, (val) => {
    // 可选：清空搜索框时自动搜索（清空结果）
    fetchData(activeKey.value,1, pagination.pageSize, '',selectedPartnerId.value,selectedOrderStatus.value,selectedStatus.value,startDate.value,endDate.value);
});
onMounted(() => {
  // 第一次加载
  fetchData(activeKey.value,pagination.current, pagination.pageSize, searchKeyword.value);
});
</script>

<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>