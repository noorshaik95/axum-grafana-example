import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Types } from 'mongoose';

export type EnrollmentDocument = Enrollment & Document;

export enum EnrollmentType {
  SELF = 'SELF',
  INSTRUCTOR = 'INSTRUCTOR',
  ADMIN = 'ADMIN',
}

export enum EnrollmentStatus {
  ACTIVE = 'ACTIVE',
  DROPPED = 'DROPPED',
  COMPLETED = 'COMPLETED',
  WAITLISTED = 'WAITLISTED',
}

@Schema({ timestamps: true })
export class Enrollment {
  @Prop({ type: Types.ObjectId, ref: 'Course', required: true })
  courseId: string;

  @Prop({ required: true })
  studentId: string;

  @Prop({ type: String, enum: EnrollmentType, required: true })
  enrollmentType: EnrollmentType;

  @Prop({ type: String, enum: EnrollmentStatus, default: EnrollmentStatus.ACTIVE })
  status: EnrollmentStatus;

  @Prop({ required: true })
  enrolledBy: string;

  @Prop({ type: Date, default: Date.now })
  enrolledAt: Date;

  @Prop({ type: Types.ObjectId, ref: 'Section' })
  sectionId?: string;
}

export const EnrollmentSchema = SchemaFactory.createForClass(Enrollment);

// Indexes
EnrollmentSchema.index({ courseId: 1, studentId: 1 }, { unique: true });
EnrollmentSchema.index({ studentId: 1, status: 1 });
EnrollmentSchema.index({ courseId: 1, status: 1 });
EnrollmentSchema.index({ sectionId: 1 });
EnrollmentSchema.index({ enrolledBy: 1 });
EnrollmentSchema.index({ enrolledAt: -1 });
