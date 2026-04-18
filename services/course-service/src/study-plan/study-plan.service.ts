import { Injectable } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { StudyPlan, StudyPlanDocument, WeekPlan } from './schemas/study-plan.schema';

@Injectable()
export class StudyPlanService {
  constructor(
    @InjectModel(StudyPlan.name)
    private readonly studyPlanModel: Model<StudyPlanDocument>,
  ) {}

  /** Get a student's generated study plan */
  async getPlan(studentId: string, tenantId: string): Promise<StudyPlanDocument | null> {
    return this.studyPlanModel.findOne({ studentId, tenantId }).exec();
  }

  /** Update a specific week's tasks */
  async updateWeek(
    studentId: string,
    tenantId: string,
    weekNum: number,
    data: Partial<WeekPlan>,
  ): Promise<StudyPlanDocument | null> {
    const plan = await this.studyPlanModel.findOne({ studentId, tenantId }).exec();
    if (!plan) return null;

    const weekIndex = plan.weeks.findIndex((w) => w.weekNum === weekNum);
    if (weekIndex === -1) {
      // Append a new week entry
      plan.weeks.push({ weekNum, theme: '', tasks: [], courseIds: [], estimatedHours: 0, ...data });
    } else {
      plan.weeks[weekIndex] = { ...plan.weeks[weekIndex], ...data };
    }

    plan.markModified('weeks');
    return plan.save();
  }

  /** Regenerate study plan from syllabi (stub — real impl would call AI/syllabus service) */
  async regeneratePlan(studentId: string, tenantId: string): Promise<StudyPlanDocument> {
    const weeks: WeekPlan[] = Array.from({ length: 16 }, (_, i) => ({
      weekNum: i + 1,
      theme: `Week ${i + 1}`,
      tasks: [],
      courseIds: [],
      estimatedHours: 8,
    }));

    const plan = await this.studyPlanModel.findOneAndUpdate(
      { studentId, tenantId },
      { studentId, tenantId, weeks, generatedAt: new Date() },
      { upsert: true, new: true },
    );
    return plan!;
  }
}
