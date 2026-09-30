<template>
  <AppLayout>
    <div class="national-day-page">
      <header class="campaign-header">
        <p class="activity-kicker"><Icon name="gift" size="sm" /> 2026 国庆限时礼遇</p>
        <h1>国庆七天乐<span>双重福利，一起享</span></h1>
        <p class="campaign-description">充值赠礼，每日再享免费额度。新老用户均可参与，两重福利可叠加。</p>
        <div class="campaign-toolbar">
          <p class="campaign-date">
            <Icon name="calendar" size="sm" />
            <span><time datetime="2026-09-30T18:00:00+08:00">2026/09/30 18:00</time><span class="date-divider">至</span><time datetime="2026-10-08T00:00:00+08:00">2026/10/08 00:00</time></span>
          </p>
          <RouterLink to="/purchase" class="btn btn-primary recharge-action">
            立即充值 <Icon name="arrowRight" size="sm" />
          </RouterLink>
        </div>
      </header>

      <section class="benefit-grid" aria-label="活动福利">
        <article class="benefit-panel recharge-benefit">
          <div class="panel-topline">
            <span class="panel-icon"><Icon name="creditCard" size="lg" /></span>
            <span class="panel-label">活动一 · 新老同享</span>
          </div>
          <h2>国庆储值赠礼</h2>
          <p class="benefit-amount">50<span>%</span><small>充值即赠</small></p>
          <p class="benefit-description">活动期间，充值即赠送充值金额的 50%，新老用户均可享受。</p>
          <div class="benefit-footer"><Icon name="checkCircle" size="sm" /><span>充值 100 元，到账 150 元额度</span></div>
        </article>

        <article class="benefit-panel daily-benefit">
          <div class="panel-topline">
            <span class="panel-icon"><Icon name="gift" size="lg" /></span>
            <span class="panel-label">活动二 · 每日福利</span>
          </div>
          <h2>10 元解锁国庆七天福利</h2>
          <p class="benefit-amount">10<span>元</span><small>每日免费额度</small></p>
          <p class="benefit-description">单笔充值满 10 元，即可解锁国庆七天福利，每天享受 10 元免费额度。</p>
          <div class="benefit-footer"><Icon name="calendar" size="sm" /><span>每日 0 点刷新，当天有效</span></div>
        </article>

        <article class="benefit-panel loyalty-benefit">
          <div class="panel-topline">
            <span class="panel-icon"><Icon name="badge" size="lg" /></span>
            <span class="panel-label">储值加享 · 老用户专属</span>
          </div>
          <h2>再享限时额度</h2>
          <p class="benefit-amount">20<span>%</span><small>额外赠送</small></p>
          <p class="benefit-description">老用户（历史充值过的用户）同享 50% 储值赠礼，再额外获赠充值金额 20% 的限时额度。</p>
          <div class="benefit-footer"><Icon name="checkCircle" size="sm" /><span>限时额度有效期 1 个月</span></div>
        </article>
      </section>

      <section class="example-section" aria-labelledby="example-heading">
        <div class="section-heading">
          <div>
            <p class="section-kicker">福利叠加享</p>
            <h2 id="example-heading">充多少，享多少</h2>
          </div>
          <p class="section-caption">以充值 100 元为例</p>
        </div>
        <div class="example-table-wrap">
          <table class="example-table">
            <thead>
              <tr><th scope="col">用户类型</th><th scope="col">充值金额</th><th scope="col">到账额度</th><th scope="col">额外限时额度</th><th scope="col">每日免费额度</th></tr>
            </thead>
            <tbody>
              <tr>
                <th scope="row">新用户</th><td>100 元</td><td class="credited-amount">150 元</td><td class="muted-cell">—</td><td><span class="daily-quota">10 元 / 天</span></td>
              </tr>
              <tr>
                <th scope="row">老用户</th><td>100 元</td><td class="credited-amount">150 元</td><td class="limited-amount">+20 元</td><td><span class="daily-quota">10 元 / 天</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <footer class="campaign-footer">
        <div class="footer-copy">
          <Icon name="sparkles" size="lg" />
          <div><p>国庆好礼，充值即享</p><span>赠送额度将在支付成功后自动发放</span></div>
        </div>
        <RouterLink to="/purchase" class="btn btn-primary recharge-action">
          去充值 <Icon name="arrowRight" size="sm" />
        </RouterLink>
      </footer>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
</script>

<style scoped>
.national-day-page {
  @apply text-gray-900 dark:text-gray-100;
  max-width: 1120px;
  margin: 0 auto;
  letter-spacing: 0;
}

.campaign-header {
  @apply border-b border-gray-200 dark:border-dark-700/60;
  padding: 12px 0 28px;
}

.activity-kicker {
  @apply text-red-700 dark:text-red-300;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  font-size: 13px;
  font-weight: 600;
}

.campaign-header h1 {
  margin: 0;
  font-size: 36px;
  line-height: 1.4;
  font-weight: 700;
}

.campaign-header h1 span {
  @apply text-primary-700 dark:text-primary-300;
  display: inline-block;
  margin-left: 20px;
}

.campaign-description {
  @apply text-gray-600 dark:text-dark-300;
  margin-top: 12px;
  font-size: 15px;
  line-height: 1.8;
}

.campaign-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 24px;
}

.campaign-date {
  @apply text-gray-500 dark:text-dark-400;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  line-height: 1.8;
}

.campaign-date svg { flex-shrink: 0; }
.campaign-date time { display: inline-block; }
.date-divider { margin: 0 10px; }
.recharge-action { min-height: 42px; border-radius: 8px; white-space: nowrap; }

.benefit-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
  margin-top: 28px;
}

.benefit-panel {
  @apply border border-gray-200 bg-white shadow-sm dark:border-dark-700/60 dark:bg-dark-800/50;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 24px;
  border-radius: 8px;
}

.panel-topline { display: flex; align-items: center; gap: 10px; }
.panel-icon { display: grid; place-items: center; width: 38px; height: 38px; flex-shrink: 0; border-radius: 8px; }
.panel-label { @apply text-gray-500 dark:text-dark-400; font-size: 12px; line-height: 1.5; }
.recharge-benefit .panel-icon { @apply bg-primary-50 text-primary-700 dark:bg-primary-400/10 dark:text-primary-300; }
.daily-benefit .panel-icon { @apply bg-red-50 text-red-600 dark:bg-red-400/10 dark:text-red-300; }
.loyalty-benefit .panel-icon { @apply bg-amber-50 text-amber-700 dark:bg-amber-400/10 dark:text-amber-300; }

.benefit-panel h2 { margin-top: 20px; font-size: 18px; font-weight: 600; line-height: 1.6; }
.benefit-amount { display: flex; flex-wrap: wrap; align-items: baseline; column-gap: 4px; margin-top: 12px; font-size: 48px; font-weight: 700; line-height: 1.2; font-variant-numeric: tabular-nums; }
.benefit-amount > span { font-size: 24px; }
.benefit-amount small { @apply text-gray-500 dark:text-dark-400; margin-left: 6px; font-size: 12px; font-weight: 400; }
.recharge-benefit .benefit-amount { @apply text-primary-700 dark:text-primary-300; }
.daily-benefit .benefit-amount { @apply text-red-600 dark:text-red-300; }
.loyalty-benefit .benefit-amount { @apply text-amber-700 dark:text-amber-300; }
.benefit-description { @apply text-gray-600 dark:text-dark-300; flex: 1; margin: 18px 0 24px; font-size: 14px; line-height: 1.85; }
.benefit-footer { @apply border-t border-gray-100 text-gray-500 dark:border-dark-700/60 dark:text-dark-400; display: flex; gap: 8px; align-items: center; padding-top: 16px; font-size: 12px; line-height: 1.7; }
.benefit-footer svg { flex-shrink: 0; }

.example-section { margin-top: 36px; }
.section-heading { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 12px; align-items: flex-end; margin-bottom: 18px; }
.section-kicker { @apply text-primary-700 dark:text-primary-300; font-size: 12px; margin-bottom: 6px; font-weight: 600; }
.section-heading h2 { font-size: 20px; font-weight: 600; }
.section-caption { @apply text-gray-500 dark:text-dark-400; font-size: 13px; }
.example-table-wrap { overflow-x: auto; }
.example-table { width: 100%; border-collapse: collapse; text-align: left; font-size: 14px; }
.example-table th, .example-table td { @apply border-b border-gray-200 dark:border-dark-700/60; padding: 18px 16px; line-height: 1.6; }
.example-table thead { @apply bg-gray-100/70 text-gray-500 dark:bg-dark-800/40 dark:text-dark-400; font-size: 12px; }
.example-table th { font-weight: 500; }
.example-table tbody td { font-variant-numeric: tabular-nums; }
.credited-amount { @apply text-primary-700 dark:text-primary-300; font-weight: 700; }
.limited-amount { @apply text-amber-700 dark:text-amber-300; font-weight: 600; }
.muted-cell { @apply text-gray-400 dark:text-dark-500; }
.daily-quota { @apply text-gray-700 dark:text-dark-200; white-space: nowrap; }

.campaign-footer { display: flex; justify-content: space-between; align-items: center; gap: 20px; padding: 28px 0 12px; }
.footer-copy { display: flex; align-items: center; gap: 12px; }
.footer-copy > svg { @apply text-primary-600 dark:text-primary-400; flex-shrink: 0; }
.footer-copy p { font-size: 14px; font-weight: 500; }
.footer-copy span { @apply text-gray-500 dark:text-dark-400; display: block; margin-top: 4px; font-size: 12px; line-height: 1.6; }

@media (max-width: 1279px) {
  .benefit-grid { gap: 12px; }
  .benefit-panel { padding: 20px; }
  .campaign-header h1 { font-size: 30px; }
  .campaign-header h1 span { margin-left: 12px; }
  .benefit-amount small { width: 100%; margin: 8px 0 0; }
}

@media (max-width: 767px) {
  .campaign-header { padding-top: 4px; padding-bottom: 24px; }
  .campaign-header h1 { font-size: 28px; }
  .campaign-header h1 span { display: block; margin: 4px 0 0; font-size: 24px; }
  .campaign-description { font-size: 14px; }
  .campaign-toolbar { align-items: stretch; flex-direction: column; margin-top: 20px; }
  .campaign-date { font-size: 12px; }
  .date-divider { margin: 0 6px; }
  .benefit-grid { grid-template-columns: 1fr; gap: 16px; margin-top: 24px; }
  .benefit-panel { padding: 22px; }
  .benefit-panel h2 { margin-top: 16px; }
  .benefit-amount small { width: auto; margin: 0 0 0 8px; }
  .benefit-description { margin: 14px 0 20px; }
  .example-section { margin-top: 28px; }
  .example-table th, .example-table td { padding: 14px 6px; font-size: 12px; }
  .example-table thead th { max-width: 62px; }
  .daily-quota { white-space: normal; }
  .campaign-footer { align-items: stretch; flex-direction: column; }
}
</style>
