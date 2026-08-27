# Launch review

All findings from the pre-release review are resolved. The regression tests named below are part of the normal Go or frontend suite.

- [x] Bundle `cash-updater.exe` in the custom Windows NSIS installer and verify its presence in the release workflow.
- [x] Preserve tags and splits in generated and confirmed recurring occurrences (`TestRecurringOccurrences_PreserveDetailsAndExtendRollingWindow`).
- [x] Maintain a rolling twelve-month occurrence window instead of ending recurrences after one year (`TestRecurringOccurrences_PreserveDetailsAndExtendRollingWindow`).
- [x] Preserve existing category limits while editing a monthly budget (`preserva limites existentes ao adicionar categoria no orçamento`).
- [x] Carry accumulated category rollover through consecutive months (`TestPlanning_RolloverCarriesAccumulatedBalance`).
- [x] Revalidate savings-goal allocations on every ledger or opening-balance mutation (`TestLedgerMutations_RejectGoalOverallocation`).
- [x] Warn when any eligible account is negative, independently of the combined balance (`TestCalculateDashboard_WarnsWhenAnyCashAccountIsNegative`).
- [x] Hydrate tags and splits before CSV export (`TestBackupExportAndRestoreRoundTrip`).
- [x] Back up and migrate older encrypted databases opened with a recovery key (`TestRecoverPassword_AppliesPendingMigrations`).
- [x] Support creating, renaming, archiving, and restoring categories in the service and desktop UI (`TestCategoryWorkflow_CreateRenameArchiveAndRestore`, `cria e arquiva categorias pela interface`).

Additional blockers found during the completion audit:

- [x] Keep month-end recurrence dates anchored to their configured day after short months.
- [x] Propagate recurring-transaction edits to pending occurrences and allow the recurrence to be stopped.
- [x] Reject new use of archived categories while keeping historical and already-scheduled records usable.
