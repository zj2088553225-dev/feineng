<!-- src/App.vue -->
<template>
  <a-layout class="layout">
    <!-- 传递 collapsed 状态给 Sidebar，并接收变化 -->
    <Sidebar  v-if="!hideSidebar" v-model:collapsed="collapsed" />

    <!-- 右侧内容：marginLeft 动态绑定 -->
    <a-layout :style="{ marginLeft: sidebarWidth }">


      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup>
import
{ref, computed, onMounted} from 'vue'
import Sidebar from './components/Sidebar.vue'
import {useRoute} from "vue-router";
import {useStore} from "@/stores/userStore.js";

// 当前路由
const route = useRoute()

// 控制侧边栏展开/折叠
const collapsed = ref(false)

// 判断当前页面是否隐藏侧边栏
const hideSidebar = computed(() => {
  return route.meta?.hideSidebar === true
})

// 动态计算 marginLeft（仅当 Sidebar 显示时）
const sidebarWidth = computed(() => {
  if (hideSidebar.value) return '0px'
  return collapsed.value ? '80px' : '256px'
})
const store = useStore()

onMounted(() => {
  // 页面加载时尝试恢复登录状态
  store.loadUserInfo()
})
</script>

<style scoped>
.layout {
  min-height: 100vh;
}
/*
.header {
  background: #fff;
  padding: 0 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
  position: relative;
  z-index: 998;
}
*/


.content {
  background: #F5F5F5;
  overflow: auto;
}

</style>