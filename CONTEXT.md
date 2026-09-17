# Domain context

## Task

A unit of intended work. A Task has acceptance criteria, dependencies and a lifecycle. It is not an execution attempt.

## Run

One logical execution of a Task. A Run owns progress, cancellation, review outcome and repair lineage. A repaired Run points to the previous Run with `retry_of`.

## Attempt

One provider invocation or corrective pass inside a Run. Retries increase the Attempt number but do not create additional completed Tasks.

## Evidence

An immutable observation produced while validating a Task or Run. Evidence supports review but does not itself approve the work.

## Preview

A validated presentation of a Run result awaiting an explicit human decision. Approving a Preview and its Run closes the Task.
