import { Test, TestingModule } from '@nestjs/testing';
import { getModelToken } from '@nestjs/mongoose';
import { CourseRestController } from './course-rest.controller';
import { KafkaProducerService } from '../kafka/kafka.producer';
import { ProgressService } from '../progress/progress.service';
import { Course } from './schemas/course.schema';
import { NotFoundException, BadRequestException } from '@nestjs/common';

describe('CourseRestController', () => {
  let controller: CourseRestController;
  let courseModel: any;
  let kafkaProducer: any;
  let progressService: any;

  const mockCourse = {
    _id: '507f1f77bcf86cd799439011',
    tenantId: 'tenant-1',
    instructorId: 'instructor-1',
    title: 'Test Course',
    description: 'A test course',
    status: 'draft',
    modules: [],
    enrolledStudentIds: [],
    tags: [],
    settings: { allowSelfEnrollment: false, visibleToStudents: true },
    save: jest.fn().mockResolvedValue(undefined),
    markModified: jest.fn(),
    toObject: jest.fn().mockReturnThis(),
  };

  const mockCourseModelInstance = {
    ...mockCourse,
    save: jest.fn().mockResolvedValue(mockCourse),
  };

  const mockCourseModel = Object.assign(
    jest.fn().mockImplementation(() => mockCourseModelInstance),
    {
      find: jest.fn().mockReturnValue({
        sort: jest.fn().mockReturnValue({
          skip: jest.fn().mockReturnValue({
            limit: jest.fn().mockReturnValue({
              exec: jest.fn().mockResolvedValue([mockCourse]),
            }),
          }),
        }),
      }),
      findOne: jest.fn().mockReturnValue({ exec: jest.fn().mockResolvedValue(mockCourse) }),
      findOneAndUpdate: jest
        .fn()
        .mockReturnValue({ exec: jest.fn().mockResolvedValue(mockCourse) }),
      countDocuments: jest.fn().mockReturnValue({ exec: jest.fn().mockResolvedValue(1) }),
    },
  );

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [CourseRestController],
      providers: [
        { provide: getModelToken(Course.name), useValue: mockCourseModel },
        {
          provide: KafkaProducerService,
          useValue: {
            emitCourseCreated: jest.fn().mockResolvedValue(undefined),
            emitCourseUpdated: jest.fn().mockResolvedValue(undefined),
            emitCourseLocked: jest.fn().mockResolvedValue(undefined),
            emitCourseDeleted: jest.fn().mockResolvedValue(undefined),
          },
        },
        {
          provide: ProgressService,
          useValue: {
            updateProgress: jest.fn().mockResolvedValue({ completionPercentage: 50 }),
            getStudentCourses: jest.fn().mockResolvedValue([]),
            getCourseStudents: jest.fn().mockResolvedValue([]),
            getAtRiskStudents: jest.fn().mockResolvedValue([]),
          },
        },
      ],
    }).compile();

    controller = module.get<CourseRestController>(CourseRestController);
    kafkaProducer = module.get<KafkaProducerService>(KafkaProducerService);
    progressService = module.get<ProgressService>(ProgressService);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });

  describe('createCourse', () => {
    it('should create a course and emit kafka event', async () => {
      const result = await controller.createCourse('tenant-1', {
        title: 'New Course',
        description: 'Description',
      });

      expect(result.data).toBeDefined();
      expect(kafkaProducer.emitCourseCreated).toHaveBeenCalled();
    });

    it('should throw if no tenantId', async () => {
      await expect(controller.createCourse('', { title: 'New Course' })).rejects.toThrow(
        BadRequestException,
      );
    });
  });

  describe('listCourses', () => {
    it('should return paginated courses', async () => {
      const result = await controller.listCourses('tenant-1', {});

      expect(result.data).toBeDefined();
      expect(result.meta).toBeDefined();
      expect(result.meta.page).toBe(1);
      expect(result.meta.limit).toBe(20);
    });
  });

  describe('getCourse', () => {
    it('should return a course', async () => {
      const result = await controller.getCourse('tenant-1', '507f1f77bcf86cd799439011');
      expect(result.data).toBeDefined();
    });

    it('should throw NotFoundException when course not found', async () => {
      mockCourseModel.findOne.mockReturnValueOnce({
        exec: jest.fn().mockResolvedValue(null),
      });

      await expect(controller.getCourse('tenant-1', 'nonexistent')).rejects.toThrow(
        NotFoundException,
      );
    });
  });

  describe('updateCourse', () => {
    it('should update a course and emit event', async () => {
      const result = await controller.updateCourse('tenant-1', '507f1f77bcf86cd799439011', {
        title: 'Updated Title',
      });

      expect(result.data).toBeDefined();
      expect(kafkaProducer.emitCourseUpdated).toHaveBeenCalledWith(
        expect.objectContaining({ changes: ['title'] }),
      );
    });
  });

  describe('lockCourse', () => {
    it('should lock a course and emit event', async () => {
      const result = await controller.lockCourse('tenant-1', '507f1f77bcf86cd799439011');
      expect(result.data).toBeDefined();
      expect(kafkaProducer.emitCourseLocked).toHaveBeenCalled();
    });
  });

  describe('deleteCourse', () => {
    it('should soft-delete (archive) a course', async () => {
      const result = await controller.deleteCourse('tenant-1', '507f1f77bcf86cd799439011');
      expect(result.data.message).toBe('Course archived');
      expect(kafkaProducer.emitCourseDeleted).toHaveBeenCalled();
    });
  });

  describe('updateProgress', () => {
    it('should update student progress', async () => {
      const result = await controller.updateProgress('tenant-1', 'student-1', 'course-1', {
        lessonId: 'lesson-1',
      });
      expect(result.data.completionPercentage).toBe(50);
      expect(progressService.updateProgress).toHaveBeenCalledWith(
        'tenant-1',
        'student-1',
        'course-1',
        'lesson-1',
        undefined,
      );
    });
  });

  describe('getCourseStudents', () => {
    it('should return all students', async () => {
      const result = await controller.getCourseStudents('tenant-1', '507f1f77bcf86cd799439011');
      expect(result.data).toBeDefined();
    });

    it('should return at-risk students when filter is at-risk', async () => {
      await controller.getCourseStudents('tenant-1', '507f1f77bcf86cd799439011', 'at-risk');
      expect(progressService.getAtRiskStudents).toHaveBeenCalledWith(
        'tenant-1',
        '507f1f77bcf86cd799439011',
      );
    });
  });
});
