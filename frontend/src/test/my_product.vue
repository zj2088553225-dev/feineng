<template>
  <a-card title="首页" style="padding: 10px">
    <a-card title="工具箱">
      <!-- 条形码生成 -->
      <a-card-grid style="width: 25%; text-align: center" :hoverable="false">
        <div style="font-size: 16px; font-weight: bold; margin-bottom: 10px">条形码生成</div>
        <a-input v-model:value="barcodeValue" placeholder="请输入条形码内容" @input="onBarcodeInput"/>
        <a-button type="primary" style="margin-top: 10px" @click="openBarcodeModal" :disabled="!barcodeValue">生成条形码</a-button>
      </a-card-grid>

      <!-- 二维码生成 -->
      <a-card-grid style="width: 25%; text-align: center" :hoverable="false">
        <div style="font-size: 16px; font-weight: bold; margin-bottom: 10px">二维码生成</div>
        <a-input v-model:value="qrcodeValue" placeholder="请输入二维码内容" @input="onQrCodeInput"/>
        <a-button type="primary" style="margin-top: 10px" @click="openQrcodeModal" :disabled="!qrcodeValue">生成二维码</a-button>
      </a-card-grid>

      <!-- TikTok 视频下载 -->
      <a-card-grid style="width: 25%; text-align: center" :hoverable="false">
        <div style="font-size: 16px; font-weight: bold; margin-bottom: 10px">TikTok视频下载</div>
        <a-input v-model:value="tiktokUrl" placeholder="请输入TikTok原视频链接"/>
        <div style="margin: 10px 0">
          <a-spin :spinning="tiktokLoading" tip="解析中...">
            <video v-if="videoSrc" :src="videoSrc" controls style="max-width: 100%; height: auto; margin-top: 10px"></video>
          </a-spin>
        </div>
        <a-button type="primary" @click="fetchTikTokVideo" :loading="tiktokLoading" :disabled="!tiktokUrl">
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
    <a-modal v-model:visible="showQrcodeModal" title="二维码预览" width="400px" @cancel="closeQrcodeModal">
      <div style="text-align: center">
        <canvas id="modal-qrcode" width="200" height="200"></canvas>
      </div>
      <template #footer>
        <a-button key="back" @click="closeQrcodeModal">关闭</a-button>
        <a-button key="png" @click="downloadQrCodePng">下载PNG</a-button>
        <a-button key="submit" type="primary" @click="downloadQrCodePdf">下载PDF (19.0×19.0mm)</a-button>
      </template>
    </a-modal>
  </a-card>
</template>

<script setup>
import { ref, nextTick } from 'vue';
import { DownloadOutlined } from '@ant-design/icons-vue';
import JsBarcode from 'jsbarcode';
import QRCode from 'qrcode';
import { jsPDF } from 'jspdf';

// 条形码
const barcodeValue = ref('');
const showBarcodeModal = ref(false);

// 二维码
const qrcodeValue = ref('');
const showQrcodeModal = ref(false);

// TikTok
const tiktokUrl = ref('');
const tiktokLoading = ref(false);
const videoSrc = ref('');

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

// TikTok 模拟
const fetchTikTokVideo = () => {
  tiktokLoading.value = true;
  setTimeout(() => {
    videoSrc.value = 'https://www.w3schools.com/html/mov_bbb.mp4';
    tiktokLoading.value = false;
  }, 2000);
};

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