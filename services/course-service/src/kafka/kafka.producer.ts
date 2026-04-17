import { Injectable, OnModuleInit, OnModuleDestroy, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Kafka, Producer } from 'kafkajs';

@Injectable()
export class KafkaProducerService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(KafkaProducerService.name);
  private producer: Producer;
  private connected = false;

  constructor(private readonly configService: ConfigService) {
    const brokers = this.configService.get<string>('kafka.brokers') || 'kafka:9092';
    const kafka = new Kafka({
      clientId: 'course-service',
      brokers: brokers.split(','),
    });
    this.producer = kafka.producer();
  }

  async onModuleInit() {
    try {
      await this.producer.connect();
      this.connected = true;
      this.logger.log('Kafka producer connected');
    } catch (error) {
      this.logger.warn(
        `Kafka producer connection failed: ${error.message}. Events will be logged only.`,
      );
    }
  }

  async onModuleDestroy() {
    if (this.connected) {
      await this.producer.disconnect();
    }
  }

  async emit(topic: string, event: string, payload: Record<string, any>) {
    const message = { event, ...payload, timestamp: new Date().toISOString() };

    if (!this.connected) {
      this.logger.debug(
        `Kafka not connected. Would emit ${event} to ${topic}: ${JSON.stringify(message)}`,
      );
      return;
    }

    try {
      await this.producer.send({
        topic,
        messages: [{ key: payload.courseId || payload.tenantId, value: JSON.stringify(message) }],
      });
      this.logger.debug(`Emitted ${event} to ${topic}`);
    } catch (error) {
      this.logger.error(`Failed to emit ${event}: ${error.message}`);
    }
  }

  async emitCourseCreated(data: {
    courseId: string;
    tenantId: string;
    instructorId: string;
    title: string;
  }) {
    await this.emit('course.events', 'course.created', data);
  }

  async emitCourseUpdated(data: { courseId: string; tenantId: string; changes: string[] }) {
    await this.emit('course.events', 'course.updated', data);
  }

  async emitCourseLocked(data: { courseId: string; tenantId: string }) {
    await this.emit('course.events', 'course.locked', data);
  }

  async emitCourseDeleted(data: { courseId: string; tenantId: string }) {
    await this.emit('course.events', 'course.deleted', data);
  }

  async emitLessonCompleted(data: {
    studentId: string;
    courseId: string;
    lessonId: string;
    tenantId: string;
    completionPct: number;
  }) {
    await this.emit('course.events', 'lesson.completed', data);
  }

  async emitUserEnrolled(data: { studentId: string; courseId: string; tenantId: string }) {
    await this.emit('course.events', 'user.enrolled', data);
  }
}
