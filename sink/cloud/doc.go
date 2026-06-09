// Package cloud provides cloud destination sinks.
package cloud

// Cloud sinks write processed data to cloud storage services.
// All sinks use the intake.Sink interface and can be composed
// into pipelines alongside file-based sinks.
//
// Available sinks:
//   - S3Sink: AWS S3 object storage
//
// Future sinks (planned):
//   - GCSink: Google Cloud Storage
//   - AzureBlobSink: Azure Blob Storage
//   - BigQuerySink: Google BigQuery
//   - SnowflakeSink: Snowflake data warehouse
