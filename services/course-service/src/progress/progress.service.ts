import { Injectable, Logger, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { StudentProgress, StudentProgressDocument } from './schemas/progress.schema';
import { Course, CourseDocument } from '../course/schemas/course.schema';
import { KafkaProducerService } from '../kafka/kafka.producer';

@Injectable()
export class ProgressService {
  private readonly logger = new Logger(ProgressService.name);

  constructor(
    @InjectModel(StudentProgress.name) private progressModel: Model<StudentProgressDocument>,
    @InjectModel(Course.name) private courseModel: Model<CourseDocument>,
    private readonly kafkaProducer: KafkaProducerService,
  ) {}

  async updateProgress(
    tenantId: string,
    studentId: string,
    courseId: string,
    lessonId: string,
    timeSpentMinutes?: number,
  ) {
    const course = await this.courseModel.findOne({ _id: courseId, tenantId }).exec();
    if (!course) {
      throw new NotFoundException(`Course not found: ${courseId}`);
    }

    // Count total lessons across all modules
    const totalLessons = course.modules.reduce((sum, m) => sum + m.lessons.length, 0);

    // Upsert progress record
    let progress = await this.progressModel.findOne({ tenantId, studentId, courseId }).exec();

    if (!progress) {
      progress = new this.progressModel({
        tenantId,
        studentId,
        courseId,
        completedLessonIds: [],
        completionPercentage: 0,
        lastActiveAt: new Date(),
        enrolledAt: new Date(),
        totalTimeMinutes: 0,
      });
    }

    // Add lessonId if not already completed
    if (!progress.completedLessonIds.includes(lessonId)) {
      progress.completedLessonIds.push(lessonId);
    }

    // Recalculate completion percentage
    progress.completionPercentage =
      totalLessons > 0 ? Math.round((progress.completedLessonIds.length / totalLessons) * 100) : 0;

    progress.lastActiveAt = new Date();
    if (timeSpentMinutes) {
      progress.totalTimeMinutes += timeSpentMinutes;
    }

    await progress.save();

    // Emit Kafka event
    await this.kafkaProducer.emitLessonCompleted({
      studentId,
      courseId,
      lessonId,
      tenantId,
      completionPct: progress.completionPercentage,
    });

    this.logger.log(
      `Progress updated for student ${studentId} in course ${courseId}: ${progress.completionPercentage}%`,
    );

    return progress;
  }

  async getStudentCourses(tenantId: string, studentId: string) {
    const progressRecords = await this.progressModel.find({ tenantId, studentId }).exec();

    const courseIds = progressRecords.map((p) => p.courseId);
    const courses = await this.courseModel.find({ _id: { $in: courseIds }, tenantId }).exec();

    return progressRecords.map((progress) => {
      const course = courses.find((c) => c._id.toString() === progress.courseId);
      return { progress, course };
    });
  }

  async getCourseStudents(tenantId: string, courseId: string) {
    return this.progressModel.find({ tenantId, courseId }).exec();
  }

  async getAtRiskStudents(tenantId: string, courseId: string) {
    const fourteenDaysAgo = new Date(Date.now() - 14 * 24 * 60 * 60 * 1000);
    return this.progressModel
      .find({
        tenantId,
        courseId,
        $or: [{ completionPercentage: { $lt: 30 } }, { lastActiveAt: { $lt: fourteenDaysAgo } }],
      })
      .exec();
  }

  async enrollStudent(tenantId: string, studentId: string, courseId: string) {
    const existing = await this.progressModel.findOne({ tenantId, studentId, courseId }).exec();
    if (existing) {
      return existing;
    }

    const progress = new this.progressModel({
      tenantId,
      studentId,
      courseId,
      completedLessonIds: [],
      completionPercentage: 0,
      lastActiveAt: new Date(),
      enrolledAt: new Date(),
      totalTimeMinutes: 0,
    });

    await progress.save();

    // Also add to course's enrolledStudentIds
    await this.courseModel.findOneAndUpdate(
      { _id: courseId, tenantId },
      { $addToSet: { enrolledStudentIds: studentId } },
    );

    await this.kafkaProducer.emitUserEnrolled({ studentId, courseId, tenantId });

    return progress;
  }
}
