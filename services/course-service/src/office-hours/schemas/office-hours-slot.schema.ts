import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type OfficeHoursSlotDocument = OfficeHoursSlot & Document;

@Schema({ timestamps: true, collection: 'office_hours_slots' })
export class OfficeHoursSlot {
  @Prop({ required: true })
  professorId: string;

  @Prop({ required: true })
  courseId: string;

  @Prop({ required: true })
  startTime: Date;

  @Prop({ required: true })
  endTime: Date;

  /** Duration in minutes (default 15) */
  @Prop({ default: 15 })
  durationMinutes: number;

  @Prop({ default: true })
  available: boolean;

  @Prop()
  location: string;

  @Prop()
  meetingUrl: string;
}

export const OfficeHoursSlotSchema = SchemaFactory.createForClass(OfficeHoursSlot);
