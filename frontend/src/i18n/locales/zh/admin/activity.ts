export default {
  activity: {
    kicker: '活动管理', title: '活动',
    description: '设置十元七天乐的开始时间与福利分组。每轮持续精确 7×24 小时。',
    campaignName: '十元七天乐',
    rule: '活动期间单笔充值满 10 元即可解锁；同一用户每轮限领一次。',
    start: '开始时间', end: '结束时间', group: '福利分组', dailyLimit: '实际日限额',
    startBeijing: '开始时间（北京时间，精确到秒）', derivedEnd: '自动计算的结束时间', day: '天',
    selectGroup: '请选择分组',
    missingDefault: '未找到名为“十元七天乐”的可用分组，请手动选择。',
    partialDays: '每日额度在北京时间 0 点刷新；首尾日期可能只有部分时段。',
    activeLocked: '活动进行中，开始时间和分组不可修改。',
    closedNote: '已停止新用户解锁；已领取的福利保留到原结束时间。',
    schedule: '开启活动', edit: '修改预约', save: '保存活动', close: '关闭活动',
    closeWarning: '关闭后停止新用户解锁。已领取的福利继续保留到本轮原结束时间。确认关闭吗？',
    status: { off: '未开启', scheduled: '待开始', active: '进行中', closed: '已关闭', ended: '已结束' }
  }
}
