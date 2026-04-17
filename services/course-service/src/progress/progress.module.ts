import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { ProgressService } from './progress.service';
import { StudentProgress, StudentProgressSchema } from './schemas/progress.schema';
import { Course, CourseSchema } from '../course/schemas/course.schema';
import { KafkaModule } from '../kafka/kafka.module';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: StudentProgress.name, schema: StudentProgressSchema },
      { name: Course.name, schema: CourseSchema },
    ]),
    KafkaModule,
  ],
  providers: [ProgressService],
  exports: [ProgressService],
})
export class ProgressModule {}
