import { Injectable } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { OfficeHoursSlot, OfficeHoursSlotDocument } from './schemas/office-hours-slot.schema';
import { OfficeHoursBooking, OfficeHoursBookingDocument } from './schemas/office-hours-booking.schema';

@Injectable()
export class OfficeHoursService {
  constructor(
    @InjectModel(OfficeHoursSlot.name)
    private readonly slotModel: Model<OfficeHoursSlotDocument>,
    @InjectModel(OfficeHoursBooking.name)
    private readonly bookingModel: Model<OfficeHoursBookingDocument>,
  ) {}

  /** List available OH slots for a student (upcoming, available=true) */
  async listAvailableSlots(tenantId: string, professorId?: string): Promise<OfficeHoursSlotDocument[]> {
    const query: Record<string, unknown> = {
      available: true,
      startTime: { $gte: new Date() },
    };
    if (professorId) query.professorId = professorId;
    return this.slotModel.find(query).sort({ startTime: 1 }).limit(50).exec();
  }

  /** Book a slot for a student */
  async bookSlot(
    slotId: string,
    studentId: string,
    professorId: string,
    question: string,
  ): Promise<OfficeHoursBookingDocument> {
    // Mark slot unavailable
    await this.slotModel.findByIdAndUpdate(slotId, { available: false }).exec();
    const booking = new this.bookingModel({ slotId, studentId, professorId, question });
    return booking.save();
  }

  /** List OH schedule for an instructor */
  async listHostedSlots(professorId: string): Promise<OfficeHoursSlotDocument[]> {
    return this.slotModel
      .find({ professorId })
      .sort({ startTime: 1 })
      .exec();
  }

  /** Create an availability slot (instructor) */
  async createSlot(data: Partial<OfficeHoursSlot>): Promise<OfficeHoursSlotDocument> {
    const slot = new this.slotModel({ ...data, available: true });
    return slot.save();
  }

  /** Remove a slot */
  async removeSlot(slotId: string, professorId: string): Promise<void> {
    await this.slotModel.findOneAndDelete({ _id: slotId, professorId }).exec();
  }
}
