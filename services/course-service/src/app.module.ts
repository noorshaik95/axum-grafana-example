import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { MongooseModule } from '@nestjs/mongoose';
import { LoggerModule } from 'nestjs-pino';
import { CourseModule } from './course/course.module';
import { EnrollmentModule } from './enrollment/enrollment.module';
import { ProgressModule } from './progress/progress.module';
import { KafkaModule } from './kafka/kafka.module';
import { HealthController } from './health/health.controller';
import { MetricsController } from './observability/metrics.controller';
import { MetricsService } from './observability/metrics.service';
import { loggerConfig } from './observability/logger.config';
import configuration from './config/configuration';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      load: [configuration],
    }),
    LoggerModule.forRoot(loggerConfig),
    MongooseModule.forRootAsync({
      imports: [ConfigModule],
      useFactory: async (configService: ConfigService) => ({
        uri: configService.get<string>('mongodb.uri'),
        dbName: configService.get<string>('mongodb.dbName'),
        connectionFactory: (connection) => {
          connection.on('connected', () => {
            console.log('MongoDB connected successfully');
          });
          connection.on('error', (error: Error) => {
            console.error('MongoDB connection error:', error);
          });
          return connection;
        },
      }),
      inject: [ConfigService],
    }),
    CourseModule,
    EnrollmentModule,
    ProgressModule,
    KafkaModule,
  ],
  controllers: [HealthController, MetricsController],
  providers: [MetricsService],
})
export class AppModule {}
