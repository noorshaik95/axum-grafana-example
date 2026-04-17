import { NestFactory } from '@nestjs/core';
import { ValidationPipe } from '@nestjs/common';
import { MicroserviceOptions, Transport } from '@nestjs/microservices';
import { Logger } from 'nestjs-pino';
import { AppModule } from './app.module';
import { initializeTracing } from './observability/tracing';
import { join } from 'path';
import { Server } from '@grpc/grpc-js';
import { ReflectionService } from '@grpc/reflection';
import * as protoLoader from '@grpc/proto-loader';

// Initialize OpenTelemetry tracing before anything else
initializeTracing();

async function bootstrap() {
  // Create HTTP app for health checks and metrics
  const app = await NestFactory.create(AppModule, { bufferLogs: true });

  // Use Pino logger
  app.useLogger(app.get(Logger));

  // Enable validation
  app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));

  // Enable graceful shutdown
  app.enableShutdownHooks();

  const port = process.env.PORT || 3000;
  await app.listen(port);
  console.log(`HTTP server listening on port ${port} (health and metrics)`);

  const grpcHost = process.env.GRPC_HOST || '0.0.0.0';
  const grpcPort = process.env.GRPC_PORT || 50052;

  // Use environment variable for proto path, fallback to relative path for local development
  const protoPath = process.env.PROTO_PATH || join(__dirname, '../../../proto/course.proto');
  const protoDir = process.env.PROTO_DIR || join(__dirname, '../../../proto');

  // Load proto for reflection
  const packageDefinition = await protoLoader.load(protoPath, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
    includeDirs: [protoDir],
  });

  const grpcApp = await NestFactory.createMicroservice<MicroserviceOptions>(AppModule, {
    transport: Transport.GRPC,
    options: {
      package: 'course',
      protoPath: protoPath,
      url: `${grpcHost}:${grpcPort}`,
      loader: {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true,
        includeDirs: [protoDir],
      },
      // Enable reflection by adding it to the server options
      onLoadPackageDefinition: (pkg: any, server: Server) => {
        const reflectionService = new ReflectionService(packageDefinition);
        reflectionService.addToServer(server);
        console.log('gRPC Server Reflection enabled');
      },
    },
  });

  grpcApp.useLogger(grpcApp.get(Logger));

  await grpcApp.listen();
  console.log(`gRPC server listening on ${grpcHost}:${grpcPort}`);

  // Graceful shutdown handlers
  const shutdown = async (signal: string) => {
    console.log(`Received ${signal}, starting graceful shutdown...`);
    try {
      await app.close();
      await grpcApp.close();
      console.log('Graceful shutdown completed');
      process.exit(0);
    } catch (error) {
      console.error('Error during shutdown:', error);
      process.exit(1);
    }
  };

  process.on('SIGTERM', () => shutdown('SIGTERM'));
  process.on('SIGINT', () => shutdown('SIGINT'));
}

bootstrap().catch((error) => {
  console.error('Failed to start application:', error);
  process.exit(1);
});
