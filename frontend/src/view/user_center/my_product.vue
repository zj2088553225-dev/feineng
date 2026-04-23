<template>
  <a-card style="padding: 10px" title="产品中心页">
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
      <a-col :span="8">
        <a-input
            v-model:value="searchKeyword"
            placeholder="输入 seller_sku/jumiasku/产品中英文名搜索"
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
        <a-button style="margin-left: 10px" :loading="exportLoading" @click="exportAllToExcel">导出数据</a-button>
      </a-col>
    </a-row>
    <!-- 表格 -->
    <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        row-key="jumia_sku"
        :pagination="pagination"
        @change="handleTableChange"
        :scroll="scroll"
    >
      <!-- 操作列 -->
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'action'">
        <span>
          <a @click="EditUserProduct(record)">编辑</a>
        </span>
        </template>
      </template>
    </a-table>
  </a-card>

  <!-- 编辑抽屉 -->
  <a-drawer
      title="修改我的产品数据"
      :width="500"
      :open="open"
      :body-style="{ paddingBottom: '80px' }"
      :footer-style="{ textAlign: 'right' }"
      @close="onClose"
  >
    <a-form
        :label-col="labelCol"
        :wrapper-col="wrapperCol"
        layout="horizontal"
        :model="form"
        style="max-width: 400px"
    >
      <a-form-item label="用户名">
        <a-typography-text>{{ form.user_name }}</a-typography-text>
      </a-form-item>
      <a-form-item label="seller_sku">
        <a-typography-text copyable>{{ form.seller_sku }}</a-typography-text>
      </a-form-item>
      <a-form-item label="Jumia SKU">
        <a-typography-text copyable>{{ form.jumia_sku }}</a-typography-text>
      </a-form-item>
      <a-form-item label="英文名称">
        <a-typography-text copyable>{{ form.name_en }}</a-typography-text>
      </a-form-item>
      <a-form-item label="售价 (USD)">
        <a-typography-text copyable>${{ form.price_value }}</a-typography-text>
      </a-form-item>
      <a-form-item label="促销价 (USD)">
        <a-typography-text copyable>${{ form.sale_value }}</a-typography-text>
      </a-form-item>
      <a-form-item label="库存" name="inventory">
        <a-typography-text copyable>{{ form.inventory }}</a-typography-text>
      </a-form-item>

      <a-form-item label="产品中文名称" name="name_zh">
        <a-input v-model:value="form.name_zh" placeholder="请输入产品中文名称" />
      </a-form-item>
      <a-form-item label="1688购买链接" name="buy_url">
        <a-input v-model:value="form.buy_url" placeholder="https://..." />
      </a-form-item>
      <a-form-item label="独立站链接" name="sell_url">
        <a-input v-model:value="form.sell_url" placeholder="https://..." />
      </a-form-item>
    </a-form>
    <template #extra>
      <a-space>
        <a-button @click="onClose">取消修改</a-button>
        <a-button type="primary" @click="onSubmit()">提交修改</a-button>
      </a-space>
    </template>
  </a-drawer>
</template>

<script setup>
import {ref, onMounted, reactive, shallowRef, watch, onBeforeUnmount} from 'vue';
import { message } from 'ant-design-vue';
import request from '@/utils/request';
import * as XLSX from "xlsx";



// ========== 表格列定义（不变）==========
const columns = [
  { title: '合伙人', dataIndex: 'user_name', key: 'user_name', width: 120 },
  { title: '卖家 SKU', dataIndex: 'seller_sku', key: 'seller_sku' },
  { title: 'Jumia SKU', dataIndex: 'jumia_sku', key: 'jumia_sku', width: 180 },
  { title: '出售国家', dataIndex: 'country_name', key: 'country_name', width: 180 },
  // { title: '库存', dataIndex: 'inventory', key: 'inventory' },
  {
    title: '库存',
    dataIndex: 'inventory',
    key: 'inventory',
    // ✅ 添加 sorter 和 sortDirections
    sortDirections: ['ascend', 'descend'], // 显示上下箭头
    sorter: (a, b) => a.inventory - b.inventory, // ✅ 添加这行
    // 可选：设置默认排序
    // defaultSortOrder: 'descend',
  },
  { title: '商品名称 (en)', dataIndex: 'name_en', key: 'name_en', ellipsis: true },
  { title: '商品名称 (zh)', dataIndex: 'name_zh', key: 'name_zh' },
  { title: '售价 (USD)', dataIndex: 'price_value', key: 'price_value', customRender: ({ value }) => `$${value.toFixed(2)}` },
  { title: '促销价 (USD)', dataIndex: 'sale_value', key: 'sale_value', customRender: ({ value }) => `$${value.toFixed(2)}` },
  { title: '售价 (当地货币)', dataIndex: 'local_price_value', key: 'local_price_value', customRender: ({ record }) => {
      // record 中包含当前行所有数据，包括 local_currency
      return `${record.local_price_value.toFixed(2)} ${record.local_currency}`;
    } ,width: 200},
  { title: '促销价 (当地货币)', dataIndex: 'sale_local_value', key: 'sale_local_value', customRender: ({ record }) => {
      // record 中包含当前行所有数据，包括 local_currency
      return `${record.sale_local_value.toFixed(2)} ${record.local_currency}`;
    } ,width: 200},
  { title: '1688购买链接', dataIndex: 'buy_url', key: 'buy_url', ellipsis: true },
  { title: '独立站卖货链接', dataIndex: 'sell_url', key: 'sell_url' , ellipsis: true},
  {
    title: '销售开始时间',
    dataIndex: 'sale_start_at',
    key: 'sale_start_at',
    width: 120,
    customRender: ({ value }) => formatTime(value),
  },
  {
    title: '更新时间',
    dataIndex: 'updated_at',
    key: 'updated_at',
    width: 120,
    customRender: ({ value }) => formatTime(value),
  },
  { title: '操作', key: 'action' },
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

// 分页状态
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
    fetchData(current, size, searchKeyword.value);
  },
})

const loading = ref(false);
const tableData = ref([]);
// 获取数据（带分页）
const fetchData = async (page = 1, pageSize = 10,keyword = searchKeyword.value,sort = "",country_name=selectedCountry.value) => {
  loading.value = true;
  try {
    const res = await request.get('/user/my_product', {
      params: {
        page,
        limit: pageSize,
        key: keyword,
        sort,
        country_name
      },
    });

    console.log('后端返回:', res.data);

    if (res.data.code === 200) {
      const { list, count } = res.data.data;

      if (!list || list.length === 0) {
        message.info('暂无数据');
        tableData.value = [];
        pagination.current = page;
        pagination.total = 0;
        return;
      }

      // ✅ 直接使用 list，字段已包含 user_name
      tableData.value = list;

      // 更新分页器总条数
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
const exportLoading = ref(false); // 控制导出按钮 loading
const exportAllToExcel = async () => {
  const keyword = searchKeyword.value;
  const pageSize = 1000; // 每页取多一点，减少请求数
  let allData = [];
  let currentPage = 1;
  let total = 0;
  exportLoading.value = true
  try {
    message.loading('正在加载全部数据...', 0); // 显示提示

    // 第一次请求：只为了拿到 total 总数
    const firstRes = await request.get('/user/my_product', {
      params: {
        page: 1,
        limit: 1,
        key: keyword,
        sort:currentSort.value,
        country_name:selectedCountry.value,
      },
    });

    if (firstRes.data.code !== 200) {
      message.destroy();
      message.error(firstRes.data.msg || '获取总数失败');
      return;
    }

    total = firstRes.data.data.count;
    if (total === 0) {
      message.destroy();
      message.info('暂无数据可导出');
      return;
    }

    message.info(`共 ${total} 条数据，正在加载...`);

    // 计算总页数
    const totalPages = Math.ceil(total / pageSize);

    // 循环请求所有页
    for (let page = 1; page <= totalPages; page++) {
      const res = await request.get('/user/my_product', {
        params: {
          page,
          limit: pageSize,
          key: keyword,
          sort:currentSort.value,
          country_name:selectedCountry.value,
        },
      });

      if (res.data.code === 200 && Array.isArray(res.data.data.list)) {
        allData = allData.concat(res.data.data.list);
      }

      // 可选：显示进度
      message.loading(`加载中... ${Math.min(page * pageSize, total)}/${total}`, 0);
    }

    // ✅ 全部数据加载完成，开始导出
    message.destroy();

    if (allData.length === 0) {
      message.warning('没有数据可导出');
      return;
    }

    // 📦 格式化数据用于导出
    const exportData = allData.map(item => ({
      '合伙人': item.user_name,
      '卖家 SKU': item.seller_sku,
      '出售国家': item.country_name,
      'Jumia SKU': item.jumia_sku,
      '库存': item.inventory,
      '商品名称 (en)': item.name_en,
      '商品名称 (zh)': item.name_zh,
      '售价 (USD)': `$${Number(item.price_value).toFixed(2)}`,
      '促销价 (USD)': `$${Number(item.sale_value).toFixed(2)}`,
      '1688购买链接': item.buy_url,
      '独立站链接': item.sell_url,
      '销售开始时间': formatTime(item.sale_start_at), // 使用你已有的 formatTime 函数
      '更新时间': formatTime(item.updated_at),
    }));

    // 📄 生成工作表
    const ws = XLSX.utils.json_to_sheet(exportData);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, '用户产品数据');

    // 💾 下载文件
    const fileName = `产品数据_${keyword || '全部'}_${new Date().toLocaleDateString().replace(/\//g, '-')}.xlsx`;
    XLSX.writeFile(wb, fileName);

    message.success('导出成功！');
  } catch (err) {
    message.destroy();
    message.error('导出失败，请检查网络或数据');
    console.error(err);
  } finally{

    exportLoading.value = false
  }
};


// ========== 编辑抽屉逻辑（不变）==========
const labelCol = { span: 6 };
const wrapperCol = { span: 16 };

const form = reactive({
  name_zh: '',
  buy_url: '',
  sell_url: '',
  inventory: 0,
  user_name: "",
  seller_sku: '',
  jumia_sku: "",
  name_en: "",
  price_value: 0,
  sale_value: 0,
});

const open = ref(false);

const onClose = () => {
  open.value = false;
};

const onSubmit = async () => {
  loading.value = true;
  try {
    const res = await request.put('/user/my_product', {
      jumia_sku: form.jumia_sku,
      name_zh: form.name_zh,
      buy_url: form.buy_url,
      sell_url: form.sell_url,
      inventory: form.inventory,
    });

    if (res.data.code === 200) {
      message.success('更新成功');
      fetchData(pagination.current, pagination.pageSize);

      onClose();
    } else {
      message.error(res.data.msg || '更新失败');
    }
  } catch (err) {
    message.error('提交失败');
    console.error(err);
  } finally {
    loading.value = false;
  }
};

const EditUserProduct = (product) => {
  Object.assign(form, {
    user_name: product.user_name,
    seller_sku: product.seller_sku,
    jumia_sku: product.jumia_sku,
    name_en: product.name_en,
    price_value: product.price_value,
    sale_value: product.sale_value,
    name_zh: product.name_zh,
    buy_url: product.buy_url,
    sell_url: product.sell_url,
    inventory: product.inventory,
  });
  open.value = true;
};
const selectedCountry = ref("");
const countryOptions = ref([
  { label: '加纳', value: 'Ghana' },
  { label: '尼日利亚', value: 'Nigeria' },
  { label: '肯尼亚', value: 'Kenya' },
  // 其他国家...
]);
const handleCountryChange = (value) => {
  // value 是 user_id，可能是 undefined
  pagination.current = 1;
  fetchData(1, pagination.pageSize, searchKeyword.value, currentSort.value,value);
};
// 搜索功能
// 搜索功能
const searchKeyword = ref(''); // 搜索关键词
const handleSearch = () => {
  // 搜索时回到第一页
  pagination.current = 1;
  // 搜索时回到第一页
  fetchData(1, pagination.pageSize, searchKeyword.value);
};
const currentSort = ref(''); // 保存当前排序


const handleTableChange = (pag, filters, sorter) => {
  console.log('Sorter:', sorter); // 现在会输出 { field: 'inventory', order: 'ascend' }

  let sort = '';
  if (sorter && sorter.order && sorter.field === 'inventory') {
    const order = sorter.order === 'ascend' ? 'asc' : 'desc';
    sort = `inventory ${order}`;
  }
  currentSort.value = sort;

  fetchData(pag.current, pag.pageSize, searchKeyword.value, sort);
};
watch(searchKeyword, (val) => {
  if (val === '') {
    // 可选：清空搜索框时自动搜索（清空结果）
    fetchData(1, pagination.pageSize, '');
  }
});

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
  fetchData(pagination.current, pagination.pageSize, searchKeyword.value);
});
onBeforeUnmount(() => {
  window.removeEventListener('resize', updateScroll)
})
</script>