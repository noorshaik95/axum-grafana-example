import { Test, TestingModule } from '@nestjs/testing';
import { getModelToken } from '@nestjs/mongoose';
import { NotFoundException } from '@nestjs/common';
import { ProgressService } from './progress.service';
import { StudentProgress } from './schemas/progress.schema';
import { Course } from '../course/schemas/course.schema';
import { KafkaProducerService } from '../kafka/kafka.producer';

describe('ProgressService', () => {
  let service: ProgressService;
  let progressModel: any;
  let courseModel: any;
  let kafkaProducer: any;

  const mockCourse = {
    _id: 'course-1',
    tenantId: 'tenant-1',
    modules: [
      {
        id: 'mod-1',
        lessons: [{ id: 'lesson-1' }, { id: 'lesson-2' }, { id: 'lesson-3' }, { id: 'lesson-4' }],
      },
    ],
  };

  const mockProgress = {
    tenantId: 'tenant-1',
    studentId: 'student-1',
    courseId: 'course-1',
    completedLessonIds: [],
    completionPercentage: 0,
    lastActiveAt: new Date(),
    enrolledAt: new Date(),
    totalTimeMinutes: 0,
    save: jest.fn(),
    markModified: jest.fn(),
  };

  beforeEach(async () => {
    const mockProgressModelInstance = {
      ...mockProgress,
      save: jest.fn().mockResolvedValue(mockProgress),
    };

    const mockProgressModel = Object.assign(
      jest.fn().mockImplementation(() => mockProgressModelInstance),
      {
        findOne: jest.fn().mockReturnValue({ exec: jest.fn().mockResolvedValue(null) }),
        find: jest.fn().mockReturnValue({ exec: jest.fn().mockResolvedValue([]) }),
      },
    );

    const mockCourseModel = Object.assign(jest.fn(), {
      findOne: jest.fn().mockReturnValue({ exec: jest.fn().mockResolvedValue(mockCourse) }),
      find: jest.fn().mockReturnValue({ exec: jest.fn().mockResolvedValue([mockCourse]) }),
      findOneAndUpdate: jest
        .fn()
        .mockReturnValue({ exec: jest.fn().mockResolvedValue(mockCourse) }),
    });

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        ProgressService,
        { provide: getModelToken(StudentProgress.name), useValue: mockProgressModel },
        { provide: getModelToken(Course.name), useValue: mockCourseModel },
        {
          provide: KafkaProducerService,
          useValue: {
            emitLessonCompleted: jest.fn().mockResolvedValue(undefined),
            emitUserEnrolled: jest.fn().mockResolvedValue(undefined),
          },
        },
      ],
    }).compile();

    service = module.get<ProgressService>(ProgressService);
    progressModel = module.get(getModelToken(StudentProgress.name));
    courseModel = module.get(getModelToken(Course.name));
    kafkaProducer = module.get<KafkaProducerService>(KafkaProducerService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('updateProgress', () => {
    it('should create new progress record and emit event', async () => {
      const result = await service.updateProgress('tenant-1', 'student-1', 'course-1', 'lesson-1');

      expect(result).toBeDefined();
      expect(kafkaProducer.emitLessonCompleted).toHaveBeenCalledWith(
        expect.objectContaining({
          studentId: 'student-1',
          courseId: 'course-1',
          lessonId: 'lesson-1',
          tenantId: 'tenant-1',
        }),
      );
    });

    it('should throw NotFoundException when course not found', async () => {
      courseModel.findOne.mockReturnValueOnce({
        exec: jest.fn().mockResolvedValue(null),
      });

      await expect(
        service.updateProgress('tenant-1', 'student-1', 'nonexistent', 'lesson-1'),
      ).rejects.toThrow(NotFoundException);
    });
  });

  describe('getStudentCourses', () => {
    it('should return student courses', async () => {
      const result = await service.getStudentCourses('tenant-1', 'student-1');
      expect(result).toBeDefined();
      expect(Array.isArray(result)).toBe(true);
    });
  });

  describe('getAtRiskStudents', () => {
    it('should query with at-risk criteria', async () => {
      const result = await service.getAtRiskStudents('tenant-1', 'course-1');
      expect(result).toBeDefined();
      expect(progressModel.find).toHaveBeenCalledWith(
        expect.objectContaining({
          tenantId: 'tenant-1',
          courseId: 'course-1',
          $or: expect.arrayContaining([
            expect.objectContaining({ completionPercentage: { $lt: 30 } }),
          ]),
        }),
      );
    });
  });

  describe('enrollStudent', () => {
    it('should create progress and emit enrolled event', async () => {
      const result = await service.enrollStudent('tenant-1', 'student-1', 'course-1');
      expect(result).toBeDefined();
      expect(kafkaProducer.emitUserEnrolled).toHaveBeenCalledWith({
        studentId: 'student-1',
        courseId: 'course-1',
        tenantId: 'tenant-1',
      });
    });
  });
});
