import {
  Controller,
  Get,
  Post,
  Put,
  Patch,
  Delete,
  Body,
  Param,
  Query,
  HttpCode,
  HttpStatus,
  BadRequestException,
  NotFoundException,
  Logger,
} from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { TenantId } from '../auth/decorators/tenant.decorator';
import { Course, CourseDocument } from './schemas/course.schema';
import { KafkaProducerService } from '../kafka/kafka.producer';
import { ProgressService } from '../progress/progress.service';
import { CreateCourseDto, UpdateCourseDto, ListCoursesDto } from './dto/create-course.dto';
import {
  CreateModuleDto,
  UpdateModuleDto,
  CreateLessonDto,
  UpdateLessonDto,
  ReorderModulesDto,
} from './dto/module.dto';
import { UpdateProgressDto } from '../progress/dto/progress.dto';
import { v4 as uuidv4 } from 'uuid';

@Controller()
export class CourseRestController {
  private readonly logger = new Logger(CourseRestController.name);

  constructor(
    @InjectModel(Course.name) private readonly courseModel: Model<CourseDocument>,
    private readonly kafkaProducer: KafkaProducerService,
    private readonly progressService: ProgressService,
  ) {}

  // ── Course CRUD ──────────────────────────────────────────

  @Post('courses')
  async createCourse(@TenantId() tenantId: string, @Body() dto: CreateCourseDto) {
    this.requireTenant(tenantId);
    const course = new this.courseModel({
      ...dto,
      tenantId,
      instructorId: 'system', // would come from JWT in production
      status: 'draft',
    });
    const saved = await course.save();

    await this.kafkaProducer.emitCourseCreated({
      courseId: saved._id.toString(),
      tenantId,
      instructorId: saved.instructorId,
      title: saved.title,
    });

    return { data: saved };
  }

  @Get('courses')
  async listCourses(@TenantId() tenantId: string, @Query() query: ListCoursesDto) {
    this.requireTenant(tenantId);
    const filter: any = { tenantId, status: { $ne: 'archived' } };

    if (query.status) filter.status = query.status;
    if (query.instructorId) filter.instructorId = query.instructorId;
    if (query.category) filter.category = query.category;

    const page = Math.max(1, Number(query.page) || 1);
    const limit = Math.min(100, Math.max(1, Number(query.limit) || 20));
    const skip = (page - 1) * limit;
    const sortBy = query.sortBy || 'createdAt';
    const sortOrder = query.sortOrder === 'asc' ? 1 : -1;

    const [courses, total] = await Promise.all([
      this.courseModel
        .find(filter)
        .sort({ [sortBy]: sortOrder })
        .skip(skip)
        .limit(limit)
        .exec(),
      this.courseModel.countDocuments(filter).exec(),
    ]);

    return { data: courses, meta: { page, limit, total } };
  }

  @Get('courses/:id')
  async getCourse(@TenantId() tenantId: string, @Param('id') id: string) {
    this.requireTenant(tenantId);
    const course = await this.findCourseOrThrow(id, tenantId);
    return { data: course };
  }

  @Put('courses/:id')
  async updateCourse(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Body() dto: UpdateCourseDto,
  ) {
    this.requireTenant(tenantId);
    const course = await this.courseModel
      .findOneAndUpdate({ _id: id, tenantId }, { $set: dto }, { new: true })
      .exec();
    if (!course) throw new NotFoundException('Course not found');

    const changes = Object.keys(dto);
    await this.kafkaProducer.emitCourseUpdated({
      courseId: id,
      tenantId,
      changes,
    });

    return { data: course };
  }

  @Patch('courses/:id/lock')
  async lockCourse(@TenantId() tenantId: string, @Param('id') id: string) {
    this.requireTenant(tenantId);
    const course = await this.courseModel
      .findOneAndUpdate(
        { _id: id, tenantId, status: { $ne: 'archived' } },
        { $set: { status: 'locked' } },
        { new: true },
      )
      .exec();
    if (!course) throw new NotFoundException('Course not found');

    await this.kafkaProducer.emitCourseLocked({ courseId: id, tenantId });
    return { data: course };
  }

  @Patch('courses/:id/unlock')
  async unlockCourse(@TenantId() tenantId: string, @Param('id') id: string) {
    this.requireTenant(tenantId);
    const course = await this.courseModel
      .findOneAndUpdate(
        { _id: id, tenantId, status: 'locked' },
        { $set: { status: 'active' } },
        { new: true },
      )
      .exec();
    if (!course) throw new NotFoundException('Course not found or not locked');
    return { data: course };
  }

  @Delete('courses/:id')
  @HttpCode(HttpStatus.OK)
  async deleteCourse(@TenantId() tenantId: string, @Param('id') id: string) {
    this.requireTenant(tenantId);
    const course = await this.courseModel
      .findOneAndUpdate({ _id: id, tenantId }, { $set: { status: 'archived' } }, { new: true })
      .exec();
    if (!course) throw new NotFoundException('Course not found');

    await this.kafkaProducer.emitCourseDeleted({ courseId: id, tenantId });
    return { data: { message: 'Course archived' } };
  }

  // ── Modules ──────────────────────────────────────────────

  @Post('courses/:id/modules')
  async addModule(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Body() dto: CreateModuleDto,
  ) {
    this.requireTenant(tenantId);
    const course = await this.findCourseOrThrow(id, tenantId);

    const maxOrder = course.modules.reduce((max, m) => Math.max(max, m.order), -1);
    const newModule = {
      id: uuidv4(),
      title: dto.title,
      description: dto.description || '',
      order: dto.order ?? maxOrder + 1,
      lessons: [],
      isVisible: dto.isVisible ?? true,
    };

    const updated = await this.courseModel
      .findOneAndUpdate({ _id: id, tenantId }, { $push: { modules: newModule } }, { new: true })
      .exec();

    return { data: newModule, course: updated };
  }

  @Get('courses/:id/modules')
  async listModules(@TenantId() tenantId: string, @Param('id') id: string) {
    this.requireTenant(tenantId);
    const course = await this.findCourseOrThrow(id, tenantId);
    const sorted = [...course.modules].sort((a, b) => a.order - b.order);
    return { data: sorted };
  }

  @Put('courses/:id/modules/:moduleId')
  async updateModule(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Param('moduleId') moduleId: string,
    @Body() dto: UpdateModuleDto,
  ) {
    this.requireTenant(tenantId);

    const setFields: Record<string, any> = {};
    if (dto.title !== undefined) setFields['modules.$.title'] = dto.title;
    if (dto.description !== undefined) setFields['modules.$.description'] = dto.description;
    if (dto.order !== undefined) setFields['modules.$.order'] = dto.order;
    if (dto.isVisible !== undefined) setFields['modules.$.isVisible'] = dto.isVisible;

    const course = await this.courseModel
      .findOneAndUpdate(
        { _id: id, tenantId, 'modules.id': moduleId },
        { $set: setFields },
        { new: true },
      )
      .exec();

    if (!course) throw new NotFoundException('Course or module not found');
    const mod = course.modules.find((m) => m.id === moduleId);
    return { data: mod };
  }

  @Delete('courses/:id/modules/:moduleId')
  @HttpCode(HttpStatus.OK)
  async deleteModule(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Param('moduleId') moduleId: string,
  ) {
    this.requireTenant(tenantId);
    const course = await this.courseModel
      .findOneAndUpdate(
        { _id: id, tenantId },
        { $pull: { modules: { id: moduleId } } },
        { new: true },
      )
      .exec();

    if (!course) throw new NotFoundException('Course not found');
    return { data: { message: 'Module deleted' } };
  }

  @Patch('courses/:id/modules/reorder')
  async reorderModules(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Body() dto: ReorderModulesDto,
  ) {
    this.requireTenant(tenantId);
    const course = await this.findCourseOrThrow(id, tenantId);

    for (const item of dto.items) {
      const mod = course.modules.find((m) => m.id === item.moduleId);
      if (mod) {
        mod.order = item.order;
      }
    }

    course.markModified('modules');
    await course.save();

    const sorted = [...course.modules].sort((a, b) => a.order - b.order);
    return { data: sorted };
  }

  // ── Lessons ──────────────────────────────────────────────

  @Post('courses/:id/modules/:moduleId/lessons')
  async addLesson(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Param('moduleId') moduleId: string,
    @Body() dto: CreateLessonDto,
  ) {
    this.requireTenant(tenantId);
    const course = await this.findCourseOrThrow(id, tenantId);
    const mod = course.modules.find((m) => m.id === moduleId);
    if (!mod) throw new NotFoundException('Module not found');

    const maxOrder = mod.lessons.reduce((max, l) => Math.max(max, l.order), -1);
    const newLesson = {
      id: uuidv4(),
      title: dto.title,
      order: dto.order ?? maxOrder + 1,
      contentType: dto.contentType || 'text',
      contentUrl: dto.contentUrl || '',
      isVisible: dto.isVisible ?? true,
      visibleAfter: dto.visibleAfter ? new Date(dto.visibleAfter) : null,
    };

    const updated = await this.courseModel
      .findOneAndUpdate(
        { _id: id, tenantId, 'modules.id': moduleId },
        { $push: { 'modules.$.lessons': newLesson } },
        { new: true },
      )
      .exec();

    return { data: newLesson, course: updated };
  }

  @Put('courses/:id/modules/:moduleId/lessons/:lid')
  async updateLesson(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Param('moduleId') moduleId: string,
    @Param('lid') lessonId: string,
    @Body() dto: UpdateLessonDto,
  ) {
    this.requireTenant(tenantId);
    const course = await this.findCourseOrThrow(id, tenantId);
    const mod = course.modules.find((m) => m.id === moduleId);
    if (!mod) throw new NotFoundException('Module not found');

    const lesson = mod.lessons.find((l) => l.id === lessonId);
    if (!lesson) throw new NotFoundException('Lesson not found');

    if (dto.title !== undefined) lesson.title = dto.title;
    if (dto.order !== undefined) lesson.order = dto.order;
    if (dto.contentType !== undefined) lesson.contentType = dto.contentType;
    if (dto.contentUrl !== undefined) lesson.contentUrl = dto.contentUrl;
    if (dto.isVisible !== undefined) lesson.isVisible = dto.isVisible;
    if (dto.visibleAfter !== undefined)
      lesson.visibleAfter = dto.visibleAfter ? new Date(dto.visibleAfter) : null;

    course.markModified('modules');
    await course.save();

    return { data: lesson };
  }

  @Delete('courses/:id/modules/:moduleId/lessons/:lid')
  @HttpCode(HttpStatus.OK)
  async deleteLesson(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Param('moduleId') moduleId: string,
    @Param('lid') lessonId: string,
  ) {
    this.requireTenant(tenantId);
    const course = await this.findCourseOrThrow(id, tenantId);
    const mod = course.modules.find((m) => m.id === moduleId);
    if (!mod) throw new NotFoundException('Module not found');

    mod.lessons = mod.lessons.filter((l) => l.id !== lessonId);
    course.markModified('modules');
    await course.save();

    return { data: { message: 'Lesson deleted' } };
  }

  // ── Student Progress & Enrollment ────────────────────────

  @Patch('students/:studentId/courses/:courseId/progress')
  async updateProgress(
    @TenantId() tenantId: string,
    @Param('studentId') studentId: string,
    @Param('courseId') courseId: string,
    @Body() dto: UpdateProgressDto,
  ) {
    this.requireTenant(tenantId);
    const progress = await this.progressService.updateProgress(
      tenantId,
      studentId,
      courseId,
      dto.lessonId,
      dto.timeSpentMinutes,
    );
    return { data: progress };
  }

  @Get('students/:studentId/courses')
  async getStudentCourses(@TenantId() tenantId: string, @Param('studentId') studentId: string) {
    this.requireTenant(tenantId);
    const results = await this.progressService.getStudentCourses(tenantId, studentId);
    return { data: results };
  }

  @Get('courses/:id/students')
  async getCourseStudents(
    @TenantId() tenantId: string,
    @Param('id') id: string,
    @Query('filter') filter?: string,
  ) {
    this.requireTenant(tenantId);
    await this.findCourseOrThrow(id, tenantId);

    if (filter === 'at-risk') {
      const students = await this.progressService.getAtRiskStudents(tenantId, id);
      return { data: students };
    }

    const students = await this.progressService.getCourseStudents(tenantId, id);
    return { data: students };
  }

  // ── Helpers ──────────────────────────────────────────────

  private requireTenant(tenantId: string) {
    if (!tenantId) {
      throw new BadRequestException('X-Tenant-ID header or JWT tenantId is required');
    }
  }

  private async findCourseOrThrow(id: string, tenantId: string): Promise<CourseDocument> {
    const course = await this.courseModel.findOne({ _id: id, tenantId }).exec();
    if (!course) throw new NotFoundException('Course not found');
    return course;
  }
}
