<script setup lang="ts">
import { ElMessage } from 'element-plus'

function handleSave() {
  ElMessage.success('设置保存成功（演示页面）')
}
</script>

<template>
  <div class="settings-page">
    <h1 class="page-title">系统设置</h1>

    <div class="page-card">
      <div class="page-card-title">基本设置</div>
      <el-form label-width="120px" label-position="left">
        <el-form-item label="系统名称">
          <el-input value="FullStack 微服务管理平台" style="width: 400px" />
        </el-form-item>
        <el-form-item label="默认语言">
          <el-select :model-value="'zh-CN'" style="width: 200px">
            <el-option label="简体中文" value="zh-CN" />
            <el-option label="English" value="en-US" />
          </el-select>
        </el-form-item>
        <el-form-item label="时区">
          <el-select :model-value="'Asia/Shanghai'" style="width: 200px">
            <el-option label="Asia/Shanghai (UTC+8)" value="Asia/Shanghai" />
            <el-option label="UTC" value="UTC" />
            <el-option label="America/New_York" value="America/New_York" />
          </el-select>
        </el-form-item>
      </el-form>
    </div>

    <div class="page-card">
      <div class="page-card-title">AI 设置</div>
      <el-form label-width="120px" label-position="left">
        <el-form-item label="默认模型">
          <el-select :model-value="'gpt-4'" style="width: 200px">
            <el-option label="GPT-4" value="gpt-4" />
            <el-option label="GPT-4o" value="gpt-4o" />
            <el-option label="GPT-3.5 Turbo" value="gpt-3.5-turbo" />
          </el-select>
        </el-form-item>
        <el-form-item label="温度">
          <el-slider :model-value="0.7" :min="0" :max="2" :step="0.1" show-input style="width: 300px" />
        </el-form-item>
        <el-form-item label="最大 Token">
          <el-input-number :model-value="4096" :min="256" :max="32768" style="width: 200px" />
        </el-form-item>
      </el-form>
    </div>

    <div class="page-card">
      <div class="page-card-title">通知设置</div>
      <el-form label-width="120px" label-position="left">
        <el-form-item label="邮件通知">
          <el-switch :model-value="true" />
        </el-form-item>
        <el-form-item label="系统消息">
          <el-switch :model-value="true" />
        </el-form-item>
        <el-form-item label="订单提醒">
          <el-switch :model-value="false" />
        </el-form-item>
      </el-form>
    </div>

    <div style="display: flex; justify-content: flex-end">
      <el-button type="primary" class="btn-gradient" @click="handleSave">
        <el-icon><Check /></el-icon> 保存设置
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 800px;
}
</style>
