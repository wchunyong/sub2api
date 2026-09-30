<template>
  <AppLayout>
    <main class="national-day-page">
      <section class="hero-section">
        <div class="hero-copy">
          <p class="activity-kicker">2026 国庆限时活动</p>
          <h1>国庆七天乐，双重福利限时开启</h1>
          <p class="hero-subtitle">储值赠送到账额度，单笔满 10 元解锁每日福利。</p>
          <div class="hero-meta">
            <span><Icon name="calendar" size="sm" /> 2026/09/30 18:00 - 2026/10/08 00:00</span>
            <span><Icon name="clock" size="sm" /> 上海时区</span>
          </div>
          <div class="hero-actions">
            <button class="primary-action" @click="goRecharge">立即充值</button>
            <a href="#rules" class="secondary-action">查看活动规则</a>
          </div>
        </div>
        <div class="festival-visual" aria-hidden="true">
          <div class="sunburst"></div>
          <div class="gate">
            <div class="gate-roof"></div>
            <div class="gate-body">
              <span>50%</span>
              <small>储值赠送</small>
            </div>
          </div>
          <div class="benefit-chip chip-one">每日 10 元</div>
          <div class="benefit-chip chip-two">老用户 +20%</div>
        </div>
      </section>

      <section class="benefit-grid" aria-label="活动福利">
        <article class="benefit-panel">
          <div class="panel-icon"><Icon name="creditCard" size="lg" /></div>
          <p class="panel-label">活动一</p>
          <h2>国庆储值赠送</h2>
          <p>活动期内任意固定档位充值均可参与，充值页展示的到账额度已包含 50% 赠送。</p>
          <div class="panel-stat"><strong>到账 150</strong><span>充值 100 示例</span></div>
        </article>

        <article class="benefit-panel">
          <div class="panel-icon"><Icon name="gift" size="lg" /></div>
          <p class="panel-label">活动二</p>
          <h2>10 元解锁七天福利</h2>
          <p>单笔余额充值大于等于 10 元即可解锁，每天 0 点刷新 10 元额度，当天有效。</p>
          <div class="panel-stat"><strong>10 元/天</strong><span>每日刷新，不累计</span></div>
        </article>

        <article class="benefit-panel">
          <div class="panel-icon"><Icon name="badge" size="lg" /></div>
          <p class="panel-label">老用户专享</p>
          <h2>额外限时额度</h2>
          <p>2026/09/30 18:00 前有成功余额充值记录的用户，额外获得充值金额 20% 的限时额度。</p>
          <div class="panel-stat"><strong>+20%</strong><span>有效期 1 个月</span></div>
        </article>
      </section>

      <section class="example-section" aria-label="充值示例">
        <div class="section-heading">
          <p>充值示例</p>
          <h2>两个活动独立判定，可以叠加享受</h2>
        </div>
        <div class="example-table">
          <div class="example-row header">
            <span>用户类型</span>
            <span>充值</span>
            <span>活动权益</span>
          </div>
          <div class="example-row">
            <span>新用户</span>
            <span>100</span>
            <span>到账 150，并解锁每日 10 元七天福利</span>
          </div>
          <div class="example-row">
            <span>老用户</span>
            <span>100</span>
            <span>到账 150，另享 20 限时额度，并解锁每日 10 元七天福利</span>
          </div>
          <div class="example-row">
            <span>任意用户</span>
            <span>10</span>
            <span>到账 15，并解锁每日 10 元七天福利</span>
          </div>
        </div>
      </section>

      <section id="rules" class="rules-section" aria-label="活动规则">
        <div class="section-heading">
          <p>活动规则</p>
          <h2>关键口径</h2>
        </div>
        <div class="rules-list">
          <div v-for="rule in rules" :key="rule" class="rule-item">
            <Icon name="checkCircle" size="sm" />
            <span>{{ rule }}</span>
          </div>
        </div>
      </section>

      <div class="sticky-cta">
        <button @click="goRecharge">去充值</button>
        <span>赠送额度将在支付成功后自动发放</span>
      </div>
    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'

const router = useRouter()

const rules = [
  '储值赠送不设 10 元门槛，活动期内余额充值均可参与。',
  '10 元七天福利需单笔余额充值大于等于 10 元。',
  '两个活动分开判定，满足条件时可以叠加。',
  '老用户以 2026/09/30 18:00 前成功余额充值记录为准，管理员赠送不计入。',
  '活动资格以支付成功时间为准。',
]

function goRecharge() {
  router.push('/purchase')
}
</script>

<style scoped>
.national-day-page {
  --festival-red: #b91c1c;
  --festival-deep: #7f1d1d;
  --festival-gold: #f59e0b;
  --festival-ink: #1f2937;
  min-height: calc(100vh - 4rem);
  padding: 20px 16px 96px;
  color: var(--festival-ink);
  background:
    linear-gradient(180deg, #fff7ed 0%, #fff 42%, #fff7ed 100%);
}

.hero-section {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(300px, 0.95fr);
  gap: 28px;
  overflow: hidden;
  max-width: 1120px;
  margin: 0 auto;
  padding: 46px;
  border-radius: 8px;
  background:
    radial-gradient(circle at 80% 18%, rgba(245, 158, 11, 0.24), transparent 30%),
    linear-gradient(135deg, #991b1b 0%, #dc2626 48%, #f97316 100%);
  box-shadow: 0 24px 60px rgba(153, 27, 27, 0.22);
}

.hero-section::before,
.hero-section::after {
  position: absolute;
  content: "";
  width: 220px;
  height: 220px;
  border: 1px solid rgba(255, 237, 213, 0.36);
  transform: rotate(28deg);
}

.hero-section::before {
  right: -80px;
  top: -82px;
}

.hero-section::after {
  left: -130px;
  bottom: -150px;
}

.hero-copy {
  position: relative;
  z-index: 1;
  align-self: center;
  color: #fff7ed;
}

.activity-kicker {
  margin: 0 0 12px;
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0;
  color: #fde68a;
}

.hero-copy h1 {
  max-width: 620px;
  margin: 0;
  font-size: clamp(34px, 7vw, 64px);
  line-height: 1.05;
  letter-spacing: 0;
}

.hero-subtitle {
  max-width: 560px;
  margin: 18px 0 0;
  font-size: 18px;
  line-height: 1.7;
  color: #ffedd5;
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 24px;
}

.hero-meta span {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 34px;
  padding: 7px 11px;
  border: 1px solid rgba(254, 243, 199, 0.3);
  border-radius: 8px;
  background: rgba(127, 29, 29, 0.28);
  font-size: 13px;
  font-weight: 700;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 30px;
}

.primary-action,
.secondary-action,
.sticky-cta button {
  min-height: 44px;
  border-radius: 8px;
  font-weight: 800;
  transition: transform 0.18s ease, box-shadow 0.18s ease;
}

.primary-action {
  padding: 0 22px;
  color: #7f1d1d;
  background: #fef3c7;
  box-shadow: 0 12px 24px rgba(120, 53, 15, 0.24);
}

.secondary-action {
  display: inline-flex;
  align-items: center;
  padding: 0 18px;
  color: #fff7ed;
  border: 1px solid rgba(255, 247, 237, 0.55);
}

.primary-action:hover,
.secondary-action:hover,
.sticky-cta button:hover {
  transform: translateY(-1px);
}

.festival-visual {
  position: relative;
  z-index: 1;
  min-height: 360px;
}

.sunburst {
  position: absolute;
  inset: 18px 42px auto auto;
  width: 260px;
  height: 260px;
  border-radius: 50%;
  background: repeating-conic-gradient(from 0deg, rgba(254, 243, 199, 0.95) 0deg 8deg, rgba(251, 191, 36, 0.18) 8deg 16deg);
  opacity: 0.82;
}

.gate {
  position: absolute;
  right: 28px;
  bottom: 16px;
  width: min(320px, 84%);
  color: #7f1d1d;
}

.gate-roof {
  height: 68px;
  border-radius: 8px 8px 2px 2px;
  background: linear-gradient(180deg, #fde68a, #f59e0b);
  clip-path: polygon(8% 100%, 20% 16%, 80% 16%, 92% 100%);
}

.gate-body {
  display: grid;
  place-items: center;
  min-height: 178px;
  border: 2px solid rgba(253, 230, 138, 0.8);
  border-radius: 0 0 8px 8px;
  background: rgba(255, 247, 237, 0.92);
}

.gate-body span {
  font-size: 76px;
  line-height: 1;
  font-weight: 900;
}

.gate-body small {
  margin-top: -34px;
  font-size: 18px;
  font-weight: 800;
}

.benefit-chip {
  position: absolute;
  min-width: 124px;
  padding: 10px 14px;
  border-radius: 8px;
  color: #7f1d1d;
  background: #fff7ed;
  box-shadow: 0 12px 30px rgba(127, 29, 29, 0.2);
  font-size: 15px;
  font-weight: 900;
  text-align: center;
}

.chip-one {
  left: 28px;
  top: 76px;
}

.chip-two {
  right: 4px;
  top: 160px;
}

.benefit-grid,
.example-section,
.rules-section {
  max-width: 1120px;
  margin: 24px auto 0;
}

.benefit-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.benefit-panel {
  min-height: 260px;
  padding: 24px;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 14px 36px rgba(127, 29, 29, 0.08);
}

.panel-icon {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border-radius: 8px;
  color: #991b1b;
  background: #ffedd5;
}

.panel-label {
  margin: 18px 0 6px;
  font-size: 12px;
  font-weight: 900;
  color: #b45309;
}

.benefit-panel h2,
.section-heading h2 {
  margin: 0;
  color: #111827;
  letter-spacing: 0;
}

.benefit-panel h2 {
  font-size: 22px;
}

.benefit-panel > p:not(.panel-label) {
  margin: 12px 0 0;
  min-height: 74px;
  color: #4b5563;
  line-height: 1.7;
}

.panel-stat {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid #ffedd5;
}

.panel-stat strong {
  color: #b91c1c;
  font-size: 26px;
}

.panel-stat span {
  color: #6b7280;
  font-size: 13px;
}

.example-section,
.rules-section {
  padding: 28px;
  border-radius: 8px;
  background: #fff;
  border: 1px solid #fed7aa;
}

.section-heading p {
  margin: 0 0 6px;
  color: #b45309;
  font-size: 13px;
  font-weight: 900;
}

.section-heading h2 {
  font-size: 28px;
}

.example-table {
  margin-top: 20px;
  overflow: hidden;
  border: 1px solid #ffedd5;
  border-radius: 8px;
}

.example-row {
  display: grid;
  grid-template-columns: 0.8fr 0.6fr 1.8fr;
  gap: 12px;
  align-items: center;
  min-height: 56px;
  padding: 12px 16px;
  border-top: 1px solid #ffedd5;
}

.example-row:first-child {
  border-top: 0;
}

.example-row.header {
  min-height: 46px;
  background: #fff7ed;
  color: #7f1d1d;
  font-weight: 900;
}

.example-row span {
  min-width: 0;
  line-height: 1.55;
  word-break: break-word;
}

.rules-list {
  display: grid;
  gap: 12px;
  margin-top: 18px;
}

.rule-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 8px;
  background: #fff7ed;
  color: #374151;
  line-height: 1.6;
}

.rule-item svg {
  flex: 0 0 auto;
  margin-top: 3px;
  color: #b91c1c;
}

.sticky-cta {
  position: fixed;
  z-index: 20;
  right: 22px;
  bottom: 22px;
  display: flex;
  align-items: center;
  gap: 14px;
  max-width: calc(100vw - 44px);
  padding: 10px;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 16px 40px rgba(127, 29, 29, 0.14);
  backdrop-filter: blur(10px);
}

.sticky-cta button {
  padding: 0 20px;
  color: #fff7ed;
  background: #dc2626;
}

.sticky-cta span {
  color: #6b7280;
  font-size: 13px;
  line-height: 1.4;
}

@media (max-width: 900px) {
  .hero-section {
    grid-template-columns: 1fr;
    padding: 34px 24px 24px;
  }

  .festival-visual {
    min-height: 280px;
  }

  .benefit-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .national-day-page {
    padding: 12px 12px 112px;
  }

  .hero-copy h1 {
    font-size: 38px;
  }

  .hero-subtitle {
    font-size: 16px;
  }

  .hero-actions,
  .hero-actions > * {
    width: 100%;
  }

  .primary-action,
  .secondary-action {
    justify-content: center;
  }

  .festival-visual {
    min-height: 240px;
  }

  .sunburst {
    width: 190px;
    height: 190px;
  }

  .gate {
    right: 12px;
    width: 82%;
  }

  .gate-body {
    min-height: 132px;
  }

  .gate-body span {
    font-size: 52px;
  }

  .gate-body small {
    margin-top: -24px;
    font-size: 15px;
  }

  .chip-one {
    left: 0;
    top: 42px;
  }

  .chip-two {
    right: 0;
    top: 126px;
  }

  .example-section,
  .rules-section,
  .benefit-panel {
    padding: 20px;
  }

  .example-row {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .example-row.header {
    display: none;
  }

  .sticky-cta {
    right: 12px;
    left: 12px;
    bottom: 12px;
    justify-content: space-between;
  }
}
</style>
