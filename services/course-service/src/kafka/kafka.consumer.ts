import { Injectable, OnModuleInit, OnModuleDestroy, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Kafka, Consumer } from 'kafkajs';
import { Course, CourseDocument } from '../course/schemas/course.schema';

@Injectable()
export class KafkaConsumerService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(KafkaConsumerService.name);
  private consumer: Consumer;
  private connected = false;

  constructor(
    private readonly configService: ConfigService,
    @InjectModel(Course.name) private readonly courseModel: Model<CourseDocument>,
  ) {
    const brokers = this.configService.get<string>('kafka.brokers') || 'kafka:9092';
    const kafka = new Kafka({
      clientId: 'course-service',
      brokers: brokers.split(','),
    });
    this.consumer = kafka.consumer({ groupId: 'course-service-group' });
  }

  async onModuleInit() {
    try {
      await this.consumer.connect();
      this.connected = true;
      await this.consumer.subscribe({ topic: 'tenant.events', fromBeginning: false });
      await this.consumer.run({
        eachMessage: async ({ message }) => {
          try {
            const payload = JSON.parse(message.value.toString());
            await this.handleMessage(payload);
          } catch (error) {
            this.logger.error(`Failed to process message: ${error.message}`);
          }
        },
      });
      this.logger.log('Kafka consumer connected and subscribed to tenant.events');
    } catch (error) {
      this.logger.warn(
        `Kafka consumer connection failed: ${error.message}. Tenant events will not be processed.`,
      );
    }
  }

  async onModuleDestroy() {
    if (this.connected) {
      await this.consumer.disconnect();
    }
  }

  private async handleMessage(payload: { event: string; tenantId: string }) {
    switch (payload.event) {
      case 'tenant.disabled':
        await this.handleTenantDisabled(payload.tenantId);
        break;
      case 'tenant.enabled':
        await this.handleTenantEnabled(payload.tenantId);
        break;
      default:
        this.logger.debug(`Unhandled event: ${payload.event}`);
    }
  }

  private async handleTenantDisabled(tenantId: string) {
    this.logger.log(`Locking all courses for disabled tenant: ${tenantId}`);
    const result = await this.courseModel.updateMany(
      { tenantId, status: { $in: ['draft', 'active'] } },
      { $set: { status: 'locked' } },
    );
    this.logger.log(`Locked ${result.modifiedCount} courses for tenant ${tenantId}`);
  }

  private async handleTenantEnabled(tenantId: string) {
    this.logger.log(`Restoring courses for enabled tenant: ${tenantId}`);
    const result = await this.courseModel.updateMany(
      { tenantId, status: 'locked' },
      { $set: { status: 'active' } },
    );
    this.logger.log(`Restored ${result.modifiedCount} courses for tenant ${tenantId}`);
  }
}
