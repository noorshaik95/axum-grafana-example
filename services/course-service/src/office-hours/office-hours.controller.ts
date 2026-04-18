import {
  Controller,
  Get,
  Post,
  Delete,
  Body,
  Param,
  Query,
  HttpCode,
  HttpStatus,
} from '@nestjs/common';
import { OfficeHoursService } from './office-hours.service';

@Controller('office-hours')
export class OfficeHoursController {
  constructor(private readonly officeHoursService: OfficeHoursService) {}

  /**
   * GET /office-hours
   * List available OH slots (student view).
   * Optional query param: professorId
   */
  @Get()
  async listAvailable(
    @Query('professorId') professorId?: string,
    @Query('tenantId') tenantId?: string,
  ) {
    return this.officeHoursService.listAvailableSlots(tenantId ?? '', professorId);
  }

  /**
   * POST /office-hours/book
   * Book a slot { professorId, slotId, question }
   */
  @Post('book')
  @HttpCode(HttpStatus.CREATED)
  async bookSlot(
    @Body() body: { slotId: string; professorId: string; studentId: string; question?: string },
  ) {
    return this.officeHoursService.bookSlot(
      body.slotId,
      body.studentId,
      body.professorId,
      body.question ?? '',
    );
  }

  /**
   * GET /office-hours/hosted
   * List the instructor's OH schedule.
   */
  @Get('hosted')
  async listHosted(@Query('professorId') professorId: string) {
    return this.officeHoursService.listHostedSlots(professorId ?? '');
  }

  /**
   * POST /office-hours/slots
   * Create an availability slot (instructor).
   */
  @Post('slots')
  @HttpCode(HttpStatus.CREATED)
  async createSlot(
    @Body()
    body: {
      professorId: string;
      courseId: string;
      startTime: string;
      endTime: string;
      durationMinutes?: number;
      location?: string;
      meetingUrl?: string;
    },
  ) {
    return this.officeHoursService.createSlot({
      ...body,
      startTime: new Date(body.startTime),
      endTime: new Date(body.endTime),
    });
  }

  /**
   * DELETE /office-hours/slots/:id
   * Remove a slot.
   */
  @Delete('slots/:id')
  @HttpCode(HttpStatus.OK)
  async removeSlot(
    @Param('id') id: string,
    @Query('professorId') professorId: string,
  ) {
    await this.officeHoursService.removeSlot(id, professorId ?? '');
    return { status: 'deleted', id };
  }
}
