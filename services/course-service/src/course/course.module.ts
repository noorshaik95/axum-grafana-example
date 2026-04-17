import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { CourseController } from './course.controller';
import { CourseRestController } from './course-rest.controller';
import { CourseService } from './course.service';
import { CourseRepository } from './repositories/course.repository';
import { CourseTemplateRepository } from './repositories/course-template.repository';
import { SectionRepository } from './repositories/section.repository';
import { CrossListingRepository } from './repositories/cross-listing.repository';
import { Course, CourseSchema } from './schemas/course.schema';
import { CourseTemplate, CourseTemplateSchema } from './schemas/course-template.schema';
import { Section, SectionSchema } from './schemas/section.schema';
import { CrossListing, CrossListingSchema } from './schemas/cross-listing.schema';
import { EnrollmentModule } from '../enrollment/enrollment.module';
import { ProgressModule } from '../progress/progress.module';
import { KafkaModule } from '../kafka/kafka.module';
import { MetricsService } from '../observability/metrics.service';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: Course.name, schema: CourseSchema },
      { name: CourseTemplate.name, schema: CourseTemplateSchema },
      { name: Section.name, schema: SectionSchema },
      { name: CrossListing.name, schema: CrossListingSchema },
    ]),
    EnrollmentModule,
    ProgressModule,
    KafkaModule,
  ],
  controllers: [CourseController, CourseRestController],
  providers: [
    CourseService,
    CourseRepository,
    CourseTemplateRepository,
    SectionRepository,
    CrossListingRepository,
    MetricsService,
  ],
  exports: [CourseService, CourseRepository, SectionRepository],
})
export class CourseModule {}
