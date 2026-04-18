import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { OfficeHoursController } from './office-hours.controller';
import { OfficeHoursService } from './office-hours.service';
import { OfficeHoursSlot, OfficeHoursSlotSchema } from './schemas/office-hours-slot.schema';
import { OfficeHoursBooking, OfficeHoursBookingSchema } from './schemas/office-hours-booking.schema';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: OfficeHoursSlot.name, schema: OfficeHoursSlotSchema },
      { name: OfficeHoursBooking.name, schema: OfficeHoursBookingSchema },
    ]),
  ],
  controllers: [OfficeHoursController],
  providers: [OfficeHoursService],
  exports: [OfficeHoursService],
})
export class OfficeHoursModule {}
