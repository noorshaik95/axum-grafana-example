import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { KafkaProducerService } from './kafka.producer';
import { KafkaConsumerService } from './kafka.consumer';
import { Course, CourseSchema } from '../course/schemas/course.schema';

@Module({
  imports: [MongooseModule.forFeature([{ name: Course.name, schema: CourseSchema }])],
  providers: [KafkaProducerService, KafkaConsumerService],
  exports: [KafkaProducerService],
})
export class KafkaModule {}
