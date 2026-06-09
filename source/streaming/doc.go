// Package streaming provides streaming sources for real-time data ingestion.
package streaming

// Streaming sources follow the same intake.Source interface but are designed
// for continuous data feeds rather than finite files.
//
// Available sources:
//   - FileTailSource: tails files for new lines (log monitoring)
//
// Future sources (planned):
//   - KafkaSource: Apache Kafka integration
//   - KinesisSource: AWS Kinesis streams
//   - WebSocketSource: WebSocket feeds
//   - HTTPStreamSource: Server-sent events
