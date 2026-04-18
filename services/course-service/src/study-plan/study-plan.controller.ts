import {
  Controller,
  Get,
  Put,
  Post,
  Body,
  Param,
  Query,
  HttpCode,
  HttpStatus,
} from '@nestjs/common';
import { StudyPlanService } from './study-plan.service';
import { WeekPlan } from './schemas/study-plan.schema';

@Controller('study-plan')
export class StudyPlanController {
  constructor(private readonly studyPlanService: StudyPlanService) {}

  /**
   * GET /study-plan
   * Get the student's generated 16-week plan.
   */
  @Get()
  async getPlan(
    @Query('studentId') studentId: string,
    @Query('tenantId') tenantId: string,
  ) {
    const plan = await this.studyPlanService.getPlan(studentId, tenantId);
    if (!plan) {
      return { weeks: [], message: 'No study plan found. POST /study-plan/regenerate to generate one.' };
    }
    return plan;
  }

  /**
   * PUT /study-plan/week/:weekNum
   * Update a specific week's tasks/theme.
   */
  @Put('week/:weekNum')
  @HttpCode(HttpStatus.OK)
  async updateWeek(
    @Param('weekNum') weekNum: string,
    @Query('studentId') studentId: string,
    @Query('tenantId') tenantId: string,
    @Body() body: Partial<WeekPlan>,
  ) {
    return this.studyPlanService.updateWeek(
      studentId,
      tenantId,
      parseInt(weekNum, 10),
      body,
    );
  }

  /**
   * POST /study-plan/regenerate
   * Regenerate the plan from syllabi.
   */
  @Post('regenerate')
  @HttpCode(HttpStatus.OK)
  async regenerate(
    @Body() body: { studentId: string; tenantId: string },
  ) {
    return this.studyPlanService.regeneratePlan(body.studentId, body.tenantId);
  }
}
