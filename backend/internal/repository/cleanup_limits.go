package repository

// expiredRecordBatchSize bounds each scheduled cleanup statement. Remaining
// expired records wait for the next run; valid records are never selected.
const expiredRecordBatchSize = 100
