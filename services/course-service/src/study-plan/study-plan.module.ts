import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { StudyPlanController } from './study-plan.controller';
import { StudyPlanService } from './study-plan.service';
import { StudyPlan, StudyPlanSchema } from './schemas/study-plan.schema';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: StudyPlan.name, schema: StudyPlanSchema },
    ]),
  ],
  controllers: [StudyPlanController],
  providers: [StudyPlanService],
  exports: [StudyPlanService],
})
export class StudyPlanModule {}
