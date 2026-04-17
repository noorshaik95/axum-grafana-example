import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type StudentProgressDocument = StudentProgress & Document;

@Schema({ timestamps: true })
export class StudentProgress {
  @Prop({ required: true })
  tenantId: string;

  @Prop({ required: true })
  studentId: string;

  @Prop({ required: true })
  courseId: string;

  @Prop({ type: [String], default: [] })
  completedLessonIds: string[];

  @Prop({ default: 0 })
  completionPercentage: number;

  @Prop({ type: Date })
  lastActiveAt: Date;

  @Prop({ default: 0 })
  totalTimeMinutes: number;

  @Prop({ type: Date })
  enrolledAt: Date;
}

export const StudentProgressSchema = SchemaFactory.createForClass(StudentProgress);

StudentProgressSchema.index({ tenantId: 1, courseId: 1 });
StudentProgressSchema.index({ tenantId: 1, studentId: 1 });
StudentProgressSchema.index({ tenantId: 1, courseId: 1, studentId: 1 }, { unique: true });
StudentProgressSchema.index({ completionPercentage: 1 });
StudentProgressSchema.index({ lastActiveAt: 1 });
