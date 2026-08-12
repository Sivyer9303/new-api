## ADDED Requirements

### Requirement: Role-aware stored video access
Ordinary users SHALL access only their own stored videos. Administrators and super administrators SHALL be able to access any user's unexpired stored video from task logs without gaining access to an upstream result address.

#### Scenario: Owner previews a video
- **WHEN** an ordinary authenticated user requests an unexpired completed task that they own
- **THEN** the system serves the stored video

#### Scenario: User requests another user's video
- **WHEN** an ordinary authenticated user requests a task owned by another user
- **THEN** the system returns not found or forbidden without disclosing task details

#### Scenario: Administrator previews another user's video
- **WHEN** an administrator opens an unexpired completed video from the administrator task log
- **THEN** the system resolves the task without owner scoping, serves the stored video, and records an audit event

### Requirement: Administrative storage recovery
The administrator task-log interface SHALL provide actions to retry a failed transfer and inspect enough administrator-private diagnostic information to confirm whether the upstream generation succeeded.

#### Scenario: Retry failed storage
- **WHEN** an administrator invokes retry for a non-refunded delivery-failure task
- **THEN** the system resets the storage attempt state safely, queues a new transfer attempt, and keeps the public task non-completed until storage is ready

#### Scenario: Confirm upstream success
- **WHEN** an administrator inspects a delivery-failure task
- **THEN** the interface shows the upstream task identifier, upstream status, retry count, last storage error, and private result address only to authorized administrators

#### Scenario: Retry succeeds
- **WHEN** an administrative retry stores and verifies the result successfully
- **THEN** the task becomes completed and remains charged

### Requirement: Idempotent full manual refund
Administrators SHALL be able to issue one full refund for a non-refunded terminal delivery failure. The operation SHALL be transactional, idempotent, auditable, and SHALL not allow a custom refund amount.

#### Scenario: Refund an undeliverable task
- **WHEN** an administrator confirms that a terminal delivery failure cannot be recovered and invokes refund
- **THEN** the system refunds the task's full charged quota exactly once and records the administrator and reason

#### Scenario: Repeat the refund request
- **WHEN** the same or another administrator repeats a refund request for the task
- **THEN** the system reports that the task was already refunded and does not credit the user again

#### Scenario: Attempt a partial refund
- **WHEN** a client attempts to provide a custom refund amount
- **THEN** the system rejects the request because only a full task refund is supported

### Requirement: Refunded tasks cannot later deliver
Once a delivery-failure task has been manually refunded, the system SHALL prevent later storage retries, completion, or content delivery for that task.

#### Scenario: Retry after refund
- **WHEN** an administrator or worker attempts to retry storage after the task was refunded
- **THEN** the system rejects or ignores the retry and retains the refunded terminal state

#### Scenario: Transfer races with refund
- **WHEN** a storage transfer and manual refund race
- **THEN** a transactional state transition ensures the task ends either delivered-and-charged or undelivered-and-refunded, never delivered-and-refunded

### Requirement: User-visible delivery failure guidance
Users SHALL be able to identify a storage delivery failure in their task logs without seeing private provider details.

#### Scenario: User views failed delivery
- **WHEN** storage retries are exhausted after upstream success
- **THEN** the user's task log states that video result transfer failed, that the charge has not been automatically refunded, and that the user should contact an administrator with the task identifier

### Requirement: Administrative audit trail
Every cross-user preview, retry, upstream confirmation, and manual refund SHALL create an audit event containing the acting administrator, target task, action, timestamp, and outcome without writing secrets or upstream media URLs to general logs.

#### Scenario: Administrative action succeeds
- **WHEN** an administrator performs a video recovery or access action
- **THEN** the system records a request-correlated audit event and exposes it to authorized audit viewers

#### Scenario: Administrative action fails
- **WHEN** an administrative action is rejected or encounters an error
- **THEN** the system records the attempted action and safe failure reason
