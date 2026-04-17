import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Types } from 'mongoose';
import { ModuleSchema, Module } from './module.schema';

export type CourseDocument = Course & Document;

@Schema({ collection: 'course_metadata' })
export class CourseMetadata {
  @Prop()
  maxStudents?: number;

  @Prop()
  department?: string;

  @Prop()
  courseCode?: string;

  @Prop()
  credits?: number;

  @Prop([String])
  tags?: string[];
}

export const CourseMetadataSchema = SchemaFactory.createForClass(CourseMetadata);

@Schema({ _id: false })
export class CourseSettings {
  @Prop({ default: false })
  allowSelfEnrollment: boolean;

  @Prop({ default: true })
  visibleToStudents: boolean;
}

export const CourseSettingsSchema = SchemaFactory.createForClass(CourseSettings);

@Schema({ timestamps: true })
export class Course {
  @Prop({ required: true })
  title: string;

  @Prop()
  description: string;

  @Prop()
  term: string;

  @Prop({ type: String })
  syllabus?: string;

  @Prop({ required: true })
  instructorId: string;

  @Prop()
  tenantId: string;

  @Prop({ type: [String], default: [] })
  coInstructorIds: string[];

  @Prop({ default: false })
  isPublished: boolean;

  @Prop({ enum: ['draft', 'active', 'locked', 'archived'], default: 'draft' })
  status: string;

  @Prop()
  coverImageUrl: string;

  @Prop()
  category: string;

  @Prop({ type: [String], default: [] })
  tags: string[];

  @Prop({ type: [ModuleSchema], default: [] })
  modules: Module[];

  @Prop({ type: [String], default: [] })
  enrolledStudentIds: string[];

  @Prop({ type: CourseSettingsSchema, default: {} })
  settings: CourseSettings;

  @Prop({ type: [String], default: [] })
  prerequisiteCourseIds: string[];

  @Prop({ type: Types.ObjectId, ref: 'CourseTemplate' })
  templateId?: string;

  @Prop()
  crossListingGroupId?: string;

  @Prop({ type: CourseMetadataSchema })
  metadata?: CourseMetadata;
}

export const CourseSchema = SchemaFactory.createForClass(Course);

// Indexes
CourseSchema.index({ tenantId: 1, status: 1 });
CourseSchema.index({ tenantId: 1, instructorId: 1 });
CourseSchema.index({ instructorId: 1, term: 1 });
CourseSchema.index({ isPublished: 1 });
CourseSchema.index({ term: 1 });
CourseSchema.index({ 'metadata.department': 1, 'metadata.courseCode': 1 });
CourseSchema.index({ crossListingGroupId: 1 });
CourseSchema.index({ instructorId: 1, isPublished: 1 });
CourseSchema.index({ title: 'text', description: 'text' });
