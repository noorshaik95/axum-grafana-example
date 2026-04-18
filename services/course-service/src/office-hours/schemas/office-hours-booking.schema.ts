import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type OfficeHoursBookingDocument = OfficeHoursBooking & Document;

@Schema({ timestamps: true, collection: 'office_hours_bookings' })
export class OfficeHoursBooking {
  @Prop({ required: true })
  slotId: string;

  @Prop({ required: true })
  studentId: string;

  @Prop({ required: true })
  professorId: string;

  @Prop()
  question: string;

  /** Status: pending | confirmed | cancelled | completed */
  @Prop({ default: 'confirmed' })
  status: string;
}

export const OfficeHoursBookingSchema = SchemaFactory.createForClass(OfficeHoursBooking);
