# CI failure receipt — PR #150

Workflow run: `41337051368`
Job: `unit tests` (`124939394874`)
Head: `74b687c4627a52f861660108f8d7b86af398795a`

The selected Go tests all passed, including the scheduler resource,
scheduler assignment, and resource URI reconciliation tests. The job then
failed closed in `scripts/offline_receipts.py` while validating the committed
semantic selector plan:

```text
OFFLINE_SEMANTIC {"stage": 22, "valid": false, "reason": "EVIDENCE_REFUSED"}
ValueError: test selection mismatch
```

Root cause: `scripts/offline.py` selected the provider tests in the order
`SchedulerResource`, `SchedulerAssignment`, `DriveSchedulerUUID`, while
`scripts/offline_plan.json` listed the assignment and URI tests before the
scheduler resource. The test implementation itself did not fail. The plan
ordering is corrected in the follow-up commit and the exact GitHub log is
retained in workflow job `124939394874`.
