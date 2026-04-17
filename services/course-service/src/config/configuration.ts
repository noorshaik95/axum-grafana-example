export default () => ({
  port: parseInt(process.env.PORT, 10) || 3000,
  grpc: {
    host: process.env.GRPC_HOST || '0.0.0.0',
    port: parseInt(process.env.GRPC_PORT, 10) || 50052,
  },
  mongodb: {
    uri: process.env.MONGO_URI || 'mongodb://mongodb:27017/courses',
    dbName: process.env.MONGO_DB_NAME || 'courses',
  },
  kafka: {
    brokers: process.env.KAFKA_BROKERS || 'kafka:9092',
  },
  observability: {
    logLevel: process.env.LOG_LEVEL || 'info',
    metricsPort: parseInt(process.env.METRICS_PORT, 10) || 9090,
    otlpEndpoint: process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://tempo:4317',
  },
  service: {
    name: 'course-service',
    version: process.env.SERVICE_VERSION || '1.0.0',
  },
});
