export default {
  activity: {
    kicker: 'Campaign management', title: 'Campaigns',
    description: 'Manage Ten Yuan Seven Days, the Programmers\' Day offer and the new user first recharge benefit.',
    guideTitle: 'New User Guide',
    guideRule: 'Stays in the activity list permanently. The user popup offers a video tutorial and a written guide, with no time limit.',
    guideVideoUrl: 'Video tutorial link', guidePdfUrl: 'Written guide (PDF) link',
    guideHint: 'Links must be http or https URLs. Saving a blank link restores the default.',
    newUserTitle: 'New User First Recharge Benefit',
    newUserRule: 'Within 24 hours after registration, the first successful recharge that reaches the minimum amount receives the bonus credit. Once per user.',
    newUserEnabled: 'Enable new user first recharge benefit', newUserBonusAmount: 'Bonus amount', newUserMinAmount: 'Minimum recharge amount',
    newUserDisabledNote: 'Currently disabled. The new user benefit is hidden from users.',
    programmersDayTitle: "Programmers' Day",
    programmersDayRule: 'Active each year from October 24 at 00:00 to October 27 at 00:00 Beijing time, when enabled.',
    programmersDayEnabled: 'Enable annual offer', programmersDayPercent: 'Extra credit percentage (%)',
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
    status: { off: 'Off', scheduled: 'Scheduled', active: 'Active', closed: 'Closed', ended: 'Ended', permanent: 'Always on' }
  }
}
