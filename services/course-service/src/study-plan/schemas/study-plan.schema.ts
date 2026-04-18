import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type StudyPlanDocument = StudyPlan & Document;

export interface WeekPlan {
  weekNum: number;
  theme: string;
  tasks: string[];
  courseIds: string[];
  estimatedHours: number;
}

@Schema({ timestamps: true, collection: 'study_plans' })
export class StudyPlan {
  @Prop({ required: true })
  studentId: string;

  @Prop({ required: true })
  tenantId: string;

  /** 16-week breakdown */
  @Prop({ type: [Object], default: [] })
  weeks: WeekPlan[];

  @Prop()
  generatedAt: Date;
}

export const StudyPlanSchema = SchemaFactory.createForClass(StudyPlan);
StudyPlanSchema.index({ studentId: 1, tenantId: 1 }, { unique: true });
