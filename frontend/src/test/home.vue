<template>
  <a-card title="首页" style="padding: 10px">
    <a-card title="工具箱">
      <!-- 条形码生成 -->
      <a-card-grid style="width: 25%; text-align: center" :hoverable="false">
        <div style="font-size: 16px; font-weight: bold; margin-bottom: 10px">条形码生成</div>
        <a-input v-model:value="barcodeValue" placeholder="请输入条形码内容" @input="onBarcodeInput"/>
        <a-button type="primary" style="margin-top: 10px" @click="openBarcodeModal" :disabled="!barcodeValue">生成条形码</a-button>
      </a-card-grid>

<!--      &lt;!&ndash; 二维码生成 &ndash;&gt;-->
<!--      <a-card-grid style="width: 25%; text-align: center" :hoverable="false">-->
<!--        <div style="font-size: 16px; font-weight: bold; margin-bottom: 10px">二维码生成</div>-->
<!--        <a-input v-model:value="qrcodeValue" placeholder="请输入二维码内容" @input="onQrCodeInput"/>-->
<!--        <a-button type="primary" style="margin-top: 10px" @click="openQrcodeModal" :disabled="!qrcodeValue">生成二维码</a-button>-->
<!--      </a-card-grid>-->

      <a-card-grid style="width: 25%; text-align: center" :hoverable="false">
        <div style="font-size: 16px; font-weight: bold; margin-bottom: 10px">TikTok视频下载</div>
        <a-input v-model:value="tiktokUrl" placeholder="请输入TikTok原视频链接"/>
        <div style="margin: 10px 0">
          <a-spin :spinning="tiktokLoading" tip="解析中...">
          </a-spin>
        </div>
        <a-button type="primary" @click="fetchTikTokVideo()" :loading="tiktokLoading" :disabled="!tiktokUrl">
          <template #icon><DownloadOutlined /></template>
          下载高清无水印视频
        </a-button>


      </a-card-grid>

      <!-- 占位格子 -->
      <a-card-grid style="width: 25%; text-align: center">功能开发中...</a-card-grid>
      <a-card-grid style="width: 25%; text-align: center">功能开发中...</a-card-grid>
      <a-card-grid style="width: 25%; text-align: center">功能开发中...</a-card-grid>
      <a-card-grid style="width: 25%; text-align: center">功能开发中...</a-card-grid>
      <a-card-grid style="width: 25%; text-align: center">功能开发中...</a-card-grid>
      <a-card-grid style="width: 25%; text-align: center">功能开发中...</a-card-grid>
    </a-card>

    <!-- 条形码浮窗 -->
    <a-modal v-model:visible="showBarcodeModal" title="条形码预览" width="800px" @cancel="closeBarcodeModal">
      <div style="text-align: center">
        <svg id="modal-barcode" style="max-width: 100%; height: auto;"></svg>
      </div>
      <template #footer>
        <a-button key="back" @click="closeBarcodeModal">关闭</a-button>
        <a-button key="png" @click="downloadBarcodePng">下载PNG</a-button>
        <a-button key="submit" type="primary" @click="downloadBarcodePdf">下载PDF (106.26×21.45mm)</a-button>
      </template>
    </a-modal>

    <!-- 二维码浮窗 -->
<!--    <a-modal v-model:visible="showQrcodeModal" title="二维码预览" width="400px" @cancel="closeQrcodeModal">-->
<!--      <div style="text-align: center">-->
<!--        <canvas id="modal-qrcode" width="200" height="200"></canvas>-->
<!--      </div>-->
<!--      <template #footer>-->
<!--        <a-button key="back" @click="closeQrcodeModal">关闭</a-button>-->
<!--        <a-button key="png" @click="downloadQrCodePng">下载PNG</a-button>-->
<!--        <a-button key="submit" type="primary" @click="downloadQrCodePdf">下载PDF (19.0×19.0mm)</a-button>-->
<!--      </template>-->
<!--    </a-modal>-->
  </a-card>
</template>

<script setup>
import { ref, nextTick } from 'vue';
import { DownloadOutlined } from '@ant-design/icons-vue';
import JsBarcode from 'jsbarcode';
import QRCode from 'qrcode';
import { jsPDF } from 'jspdf';
import request from "@/utils/request.js";

// 条形码
const barcodeValue = ref('');
const showBarcodeModal = ref(false);

// 二维码
const qrcodeValue = ref('');
const showQrcodeModal = ref(false);

// TikTok
const tiktokUrl = ref('');
const tiktokLoading = ref(false);

// 输入处理，长度限制
const onBarcodeInput = () => {
  barcodeValue.value = barcodeValue.value.replace(/[^A-Za-z0-9\-_]/g, '').substring(0, 100);
};
const onQrCodeInput = () => {
  qrcodeValue.value = qrcodeValue.value.substring(0, 1000);
};

//  打开条形码浮窗
const openBarcodeModal = async () => {
  showBarcodeModal.value = true;
  await nextTick();
  const svg = document.getElementById('modal-barcode');
  if (svg && barcodeValue.value) {
    try {
      JsBarcode(svg, barcodeValue.value, {
        format: 'CODE128B',
        displayValue: true,
        fontSize: 16,
        height: 60,
        width: 2,
        margin: 10,
        text: barcodeValue.value,
        flat: true,
        quietZone: 15,
      });
    } catch (e) {
      console.error('条形码生成失败:', e);
    }
  }
};

// 打开二维码浮窗
const openQrcodeModal = async () => {
  showQrcodeModal.value = true;
  await nextTick();
  const canvas = document.getElementById('modal-qrcode');
  if (canvas && qrcodeValue.value) {
    QRCode.toCanvas(canvas, qrcodeValue.value, {
      width: 200,
      height: 200,
      margin: 2,
      color: { dark: '#000000', light: '#ffffff' },
    });
  }
};

// 关闭浮窗
const closeBarcodeModal = () => {
  showBarcodeModal.value = false;
};
const closeQrcodeModal = () => {
  showQrcodeModal.value = false;
};

//  下载条形码 PDF（106.26 × 21.45mm）
const downloadBarcodePdf = async () => {
  const svg = document.getElementById('modal-barcode');
  if (!svg || !barcodeValue.value) return;
  // 打印尺寸，可以自定义修改
  const targetWidthMm = 106.26;
  const targetHeightMm = 21.45;
  const dpi = 300;
  const mmToInch = 1 / 25.4;
  const widthPx = Math.ceil(targetWidthMm * mmToInch * dpi);
  const heightPx = Math.ceil(targetHeightMm * mmToInch * dpi);

  const svgData = new XMLSerializer().serializeToString(svg);
  const img = new Image();
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  canvas.width = widthPx;
  canvas.height = heightPx;

  await new Promise((resolve, reject) => {
    img.onload = () => {
      ctx.clearRect(0, 0, widthPx, heightPx);
      ctx.drawImage(img, 0, 0, widthPx, heightPx);
      const imgData = canvas.toDataURL('image/png', 1.0);

      const doc = new jsPDF({
        orientation: 'landscape',
        unit: 'mm',
        format: [targetWidthMm, targetHeightMm],
      });
      doc.addImage(imgData, 'PNG', 0, 0, targetWidthMm, targetHeightMm);
      doc.save(`barcode-${sanitizeFilename(barcodeValue.value)}.pdf`);
      resolve();
    };
    img.onerror = reject;
    img.src = 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svgData)));
  });
};

// 下载条形码 PNG
const downloadBarcodePng = async () => {
  const svg = document.getElementById('modal-barcode');
  if (!svg) return;

  const scale = 4;
  const width = svg.width.baseVal.value * scale;
  const height = svg.height.baseVal.value * scale;

  const svgData = new XMLSerializer().serializeToString(svg);
  const img = new Image();
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  canvas.width = width;
  canvas.height = height;

  await new Promise((resolve) => {
    img.onload = () => {
      ctx.drawImage(img, 0, 0, width, height);
      const link = document.createElement('a');
      link.download = `barcode-${sanitizeFilename(barcodeValue.value)}.png`;
      link.href = canvas.toDataURL('image/png', 1.0);
      link.click();
      resolve();
    };
    img.src = 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svgData)));
  });
};

// 下载二维码 PDF（19.0 × 19.0mm）
const downloadQrCodePdf = async () => {
  const canvas = document.getElementById('modal-qrcode');
  if (!canvas || !qrcodeValue.value) return;
  // 打印尺寸，可以自定义修改
  const targetWidthMm = 19.0;
  const targetHeightMm = 19.0;
  const dpi = 300;
  const mmToInch = 1 / 25.4;
  const widthPx = Math.ceil(targetWidthMm * mmToInch * dpi);
  const heightPx = Math.ceil(targetHeightMm * mmToInch * dpi);

  const tmpCanvas = document.createElement('canvas');
  tmpCanvas.width = widthPx;
  tmpCanvas.height = heightPx;
  const ctx = tmpCanvas.getContext('2d');
  ctx.drawImage(canvas, 0, 0, widthPx, heightPx);

  const imgData = tmpCanvas.toDataURL('image/png', 1.0);

  const doc = new jsPDF({
    orientation: 'portrait',
    unit: 'mm',
    format: [targetWidthMm, targetHeightMm], // 固定为 19.0mm × 19.0mm
  });
  doc.addImage(imgData, 'PNG', 0, 0, targetWidthMm, targetHeightMm);
  doc.save(`qrcode-${sanitizeFilename(qrcodeValue.value.substring(0, 10))}.pdf`);
};

// 下载二维码 PNG
const downloadQrCodePng = async () => {
  const canvas = document.getElementById('modal-qrcode');
  if (!canvas) return;

  const scale = 4;
  const width = canvas.width * scale;
  const height = canvas.height * scale;

  const tmpCanvas = document.createElement('canvas');
  tmpCanvas.width = width;
  tmpCanvas.height = height;
  const ctx = tmpCanvas.getContext('2d');
  ctx.drawImage(canvas, 0, 0, width, height);

  const link = document.createElement('a');
  link.download = `qrcode-${sanitizeFilename(qrcodeValue.value.substring(0, 10))}.png`;
  link.href = tmpCanvas.toDataURL('image/png', 1.0);
  link.click();
};
const fetchTikTokVideo = async () => {
  // 1. 安全获取输入链接
  const url = tiktokUrl.value;
  if (!url || typeof url !== 'string' || !url.trim()) {
    alert('请输入有效的 TikTok 视频链接');
    return;
  }

  tiktokLoading.value = true;
  try {
    const res = await request.post('/user/tiktok', {
      url: url.trim()
    });

    // 2. 统一处理响应结构
    const responseData = res.data ? res.data : res;

    if (responseData.code === 200 && responseData.data?.hd) {
      const videoURL = responseData.data.hd;

      // 3. 触发下载
      triggerDownload(videoURL);
    } else {
      alert('解析失败: ' + (responseData.msg || '无法获取下载链接'));
    }
  } catch (error) {
    console.error('下载请求失败:', error);
    alert('网络错误，请检查接口是否可用');
  } finally {
    tiktokLoading.value = false;
  }
};

// 工具函数：触发文件下载
async function triggerDownload(videoURL) {
  try {
    const response = await fetch(videoURL, {
      method: 'GET',
      mode: 'cors',
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'
      }
    });

    if (!response.ok) {
      throw new Error('HTTP ' + response.status);
    }

    const blob = await response.blob();
    const blobURL = URL.createObjectURL(blob);

    const videoId = extractVideoIdFromURL(videoURL) || 'tiktok_video';
    const filename = `tiktok_${videoId}.mp4`;

    const a = document.createElement('a');
    a.href = blobURL;
    a.download = filename;
    a.style.display = 'none';
    document.body.appendChild(a);
    a.click();

    // ✅ 关键：不要立即 revoke，给浏览器时间启动下载
    // 延迟清理：30秒后释放内存（足够大文件开始下载）
    setTimeout(() => {
      document.body.removeChild(a);
    }, 100);

    // ✅ 单独延迟 revoke，避免中断下载
    setTimeout(() => {
      URL.revokeObjectURL(blobURL);
    }, 30000); // 30秒

    // ✅ 成功触发下载，不再走 catch
  } catch (err) {
    console.warn('fetch 失败，尝试直接跳转:', err);

    // ⚠️ 如果 fetch 失败，直接跳转到视频页（用户可右键另存为）
    window.open(videoURL, '_blank');

    // ✅ 可选：提示用户“已打开页面，请手动保存”
    // alert('无法直接下载，已打开视频页面，可右键 → 另存为');
  }
}
// 文件名安全
const sanitizeFilename = (str) => {
  return str.replace(/[<>:"/\\|?*\x00-\x1F]/g, '_');
};
</script>

<style scoped>
#modal-barcode, #modal-qrcode {
  margin: 20px auto;
  display: block;
}
</style>