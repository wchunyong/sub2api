export default {
  activity: {
    kicker: 'Campaign management', title: 'Campaigns',
    description: 'Set the start time and benefit group. Each round lasts exactly 7×24 hours.',
    campaignName: 'Ten Yuan Seven Days',
    rule: 'A single recharge of at least ¥10 unlocks the benefit. One claim per user per round.',
    start: 'Starts', end: 'Ends', group: 'Benefit group', dailyLimit: 'Actual daily limit', startBeijing: 'Start time (Beijing, seconds)',
    derivedEnd: 'Calculated end time', day: 'day', selectGroup: 'Select a group',
    missingDefault: 'The active “Ten Yuan Seven Days” group was not found. Select a group manually.',
    partialDays: 'Daily quota resets at midnight Beijing time; the first and last calendar days may be partial.',
    activeLocked: 'The start time and group cannot be changed while the campaign is active.',
    closedNote: 'New claims have stopped. Existing benefits remain until the original end time.',
    schedule: 'Start campaign', edit: 'Edit schedule', save: 'Save campaign', close: 'Close campaign',
    closeWarning: 'Closing stops new claims. Existing benefits remain until the original end time. Close this campaign?',
    status: { off: 'Off', scheduled: 'Scheduled', active: 'Active', closed: 'Closed', ended: 'Ended' }
  }
}
