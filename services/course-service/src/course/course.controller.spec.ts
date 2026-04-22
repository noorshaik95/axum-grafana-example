import { Test, TestingModule } from '@nestjs/testing';
import { getModelToken } from '@nestjs/mongoose';
import { CourseController } from './course.controller';
import { CourseService } from './course.service';
import { EnrollmentService } from '../enrollment/enrollment.service';
import { ProgressService } from '../progress/progress.service';
import { AnalyticsService } from '../analytics/analytics.service';
import { Course } from './schemas/course.schema';
import { MetricsService } from '../observability/metrics.service';
import { EnrollmentType, EnrollmentStatus } from '../enrollment/schemas/enrollment.schema';

describe('CourseController', () => {
  let controller: CourseController;
  let courseService: jest.Mocked<CourseService>;
  let enrollmentService: jest.Mocked<EnrollmentService>;
  let progressService: { getCompletedLessonIds: jest.Mock };
  let analyticsService: { getCourseAnalytics: jest.Mock };
  let courseModel: { findOne: jest.Mock };

  const mockCourse = {
    _id: { toString: () => 'course-1' },
    title: 'Test Course',
    description: 'Test Description',
    term: 'Fall 2024',
    syllabus: 'Syllabus content',
    instructorId: 'instructor-1',
    coInstructorIds: [],
    isPublished: false,
    prerequisiteCourseIds: [],
    templateId: null,
    crossListingGroupId: null,
    createdAt: new Date(),
    updatedAt: new Date(),
    metadata: {},
  };

  const mockEnrollment = {
    _id: { toString: () => 'enrollment-1' },
    courseId: { toString: () => 'course-1' },
    studentId: 'student-1',
    enrollmentType: EnrollmentType.SELF,
    status: EnrollmentStatus.ACTIVE,
    enrolledBy: 'student-1',
    enrolledAt: new Date(),
    sectionId: null,
  };

  beforeEach(async () => {
    const mockCourseService = {
      createCourse: jest.fn(),
      getCourse: jest.fn(),
      updateCourse: jest.fn(),
      deleteCourse: jest.fn(),
      listCourses: jest.fn(),
      publishCourse: jest.fn(),
      unpublishCourse: jest.fn(),
      createTemplate: jest.fn(),
      getTemplate: jest.fn(),
      listTemplates: jest.fn(),
      createCourseFromTemplate: jest.fn(),
      addPrerequisite: jest.fn(),
      removePrerequisite: jest.fn(),
      checkPrerequisites: jest.fn(),
      addCoInstructor: jest.fn(),
      removeCoInstructor: jest.fn(),
      createSection: jest.fn(),
      getSection: jest.fn(),
      updateSection: jest.fn(),
      deleteSection: jest.fn(),
      listCourseSections: jest.fn(),
      crossListCourses: jest.fn(),
      removeCrossListing: jest.fn(),
      getCrossListedCourses: jest.fn(),
    };

    const mockEnrollmentService = {
      selfEnroll: jest.fn(),
      instructorAddStudent: jest.fn(),
      removeEnrollment: jest.fn(),
      getCourseRoster: jest.fn(),
      getStudentEnrollments: jest.fn(),
    };

    const mockMetricsService = {
      incrementCoursesCreated: jest.fn(),
      incrementEnrollments: jest.fn(),
      observeGrpcRequestDuration: jest.fn(),
    };

    progressService = { getCompletedLessonIds: jest.fn().mockResolvedValue([]) };
    analyticsService = { getCourseAnalytics: jest.fn() };
    courseModel = {
      findOne: jest.fn().mockReturnValue({
        lean: () => ({ exec: () => Promise.resolve(null) }),
      }),
    };

    const module: TestingModule = await Test.createTestingModule({
      controllers: [CourseController],
      providers: [
        { provide: CourseService, useValue: mockCourseService },
        { provide: EnrollmentService, useValue: mockEnrollmentService },
        { provide: ProgressService, useValue: progressService },
        { provide: AnalyticsService, useValue: analyticsService },
        { provide: getModelToken(Course.name), useValue: courseModel },
        { provide: MetricsService, useValue: mockMetricsService },
      ],
    }).compile();

    controller = module.get<CourseController>(CourseController);
    courseService = module.get(CourseService);
    enrollmentService = module.get(EnrollmentService);
  });

  describe('createCourse', () => {
    it('should create a course and return proto format', async () => {
      courseService.createCourse.mockResolvedValue(mockCourse as any);

      const result = await controller.createCourse({
        title: 'Test Course',
        description: 'Test Description',
        term: 'Fall 2024',
        instructor_id: 'instructor-1',
        syllabus: 'Syllabus content',
      });

      expect(courseService.createCourse).toHaveBeenCalledWith({
        title: 'Test Course',
        description: 'Test Description',
        term: 'Fall 2024',
        instructorId: 'instructor-1',
        syllabus: 'Syllabus content',
        metadata: undefined,
      });
      expect(result.course.id).toBe('course-1');
      expect(result.course.title).toBe('Test Course');
    });
  });

  describe('getCourse', () => {
    it('should get a course and return proto format', async () => {
      courseService.getCourse.mockResolvedValue(mockCourse as any);

      const result = await controller.getCourse({ courseId: 'course-1' });

      expect(courseService.getCourse).toHaveBeenCalledWith('course-1');
      expect(result.course.id).toBe('course-1');
    });
  });

  describe('listCourses', () => {
    it('should list courses with pagination', async () => {
      courseService.listCourses.mockResolvedValue({
        courses: [mockCourse] as any,
        total: 1,
      });

      const result = await controller.listCourses({
        instructor_id: 'instructor-1',
        page: 1,
        pageSize: 20,
      });

      expect(courseService.listCourses).toHaveBeenCalledWith({
        instructorId: 'instructor-1',
        term: undefined,
        isPublished: undefined,
        page: 1,
        pageSize: 20,
      });
      expect(result.courses).toHaveLength(1);
      expect(result.totalCount).toBe(1);
      expect(result.page).toBe(1);
    });
  });

  describe('publishCourse', () => {
    it('should publish a course', async () => {
      const publishedCourse = { ...mockCourse, isPublished: true };
      courseService.publishCourse.mockResolvedValue(publishedCourse as any);

      const result = await controller.publishCourse({ courseId: 'course-1' });

      expect(courseService.publishCourse).toHaveBeenCalledWith('course-1');
      expect(result.course.isPublished).toBe(true);
    });
  });

  describe('selfEnroll', () => {
    it('should allow self-enrollment', async () => {
      enrollmentService.selfEnroll.mockResolvedValue(mockEnrollment as any);

      const result = await controller.selfEnroll({
        courseId: 'course-1',
        studentId: 'student-1',
      });

      expect(enrollmentService.selfEnroll).toHaveBeenCalledWith('course-1', 'student-1', undefined);
      expect(result.enrollment.id).toBe('enrollment-1');
      expect(result.enrollment.enrollmentType).toBe(1); // SELF
    });
  });

  describe('instructorAddStudent', () => {
    it('should allow instructor to add student', async () => {
      const instructorEnrollment = {
        ...mockEnrollment,
        enrollmentType: EnrollmentType.INSTRUCTOR,
      };
      enrollmentService.instructorAddStudent.mockResolvedValue(instructorEnrollment as any);

      const result = await controller.instructorAddStudent({
        courseId: 'course-1',
        studentId: 'student-1',
        instructor_id: 'instructor-1',
      });

      expect(enrollmentService.instructorAddStudent).toHaveBeenCalledWith(
        'course-1',
        'student-1',
        'instructor-1',
        undefined,
      );
      expect(result.enrollment.enrollmentType).toBe(2); // INSTRUCTOR
    });
  });

  describe('getCourseRoster', () => {
    it('should return course roster', async () => {
      enrollmentService.getCourseRoster.mockResolvedValue({
        enrollments: [mockEnrollment] as any,
        totalCount: 1,
      });

      const result = await controller.getCourseRoster({
        courseId: 'course-1',
      });

      expect(enrollmentService.getCourseRoster).toHaveBeenCalledWith(
        'course-1',
        undefined,
        undefined,
      );
      expect(result.enrollments).toHaveLength(1);
      expect(result.totalCount).toBe(1);
    });
  });

  describe('getStudentEnrollments', () => {
    it('should return student enrollments with courses', async () => {
      enrollmentService.getStudentEnrollments.mockResolvedValue([
        { enrollment: mockEnrollment, course: mockCourse },
      ] as any);

      const result = await controller.getStudentEnrollments({
        studentId: 'student-1',
      });

      expect(enrollmentService.getStudentEnrollments).toHaveBeenCalledWith(
        'student-1',
        undefined,
        undefined,
      );
      expect(result.enrollments).toHaveLength(1);
      expect(result.enrollments[0].enrollment.id).toBe('enrollment-1');
      expect(result.enrollments[0].course.id).toBe('course-1');
    });
  });

  describe('addPrerequisite', () => {
    it('should add prerequisite to course', async () => {
      const withPrereq = {
        ...mockCourse,
        prerequisiteCourseIds: ['prereq-1'],
      };
      courseService.addPrerequisite.mockResolvedValue(withPrereq as any);

      const result = await controller.addPrerequisite({
        courseId: 'course-1',
        prerequisiteCourseId: 'prereq-1',
      });

      expect(courseService.addPrerequisite).toHaveBeenCalledWith('course-1', 'prereq-1');
      expect(result.course.prerequisiteCourseIds).toContain('prereq-1');
    });
  });

  describe('addCoInstructor', () => {
    it('should add co-instructor to course', async () => {
      const withCoInstructor = {
        ...mockCourse,
        coInstructorIds: ['co-instructor-1'],
      };
      courseService.addCoInstructor.mockResolvedValue(withCoInstructor as any);

      const result = await controller.addCoInstructor({
        courseId: 'course-1',
        co_instructor_id: 'co-instructor-1',
      });

      expect(courseService.addCoInstructor).toHaveBeenCalledWith('course-1', 'co-instructor-1');
      expect(result.course.co_instructor_ids).toContain('co-instructor-1');
    });
  });

  // W8 gRPC wrappers
  describe('listModules', () => {
    it('returns ordered modules for course+tenant', async () => {
      const course = {
        _id: 'course-1',
        tenantId: 'tenant-a',
        modules: [
          { id: 'm2', title: 'Second', order: 2, isVisible: true, lessons: [] },
          { id: 'm1', title: 'First', order: 1, isVisible: true, lessons: [] },
        ],
      };
      courseModel.findOne.mockReturnValueOnce({
        lean: () => ({ exec: () => Promise.resolve(course) }),
      });

      const result = await controller.listModules({
        course_id: 'course-1',
        tenant_id: 'tenant-a',
      });

      expect(courseModel.findOne).toHaveBeenCalledWith({ _id: 'course-1', tenantId: 'tenant-a' });
      expect(result.modules).toHaveLength(2);
      expect(result.modules[0].id).toBe('m1');
      expect(result.modules[1].id).toBe('m2');
    });

    it('returns empty modules when course not found', async () => {
      const result = await controller.listModules({
        course_id: 'nope',
        tenant_id: 'tenant-a',
      });
      expect(result.modules).toEqual([]);
    });
  });

  describe('getNextUp', () => {
    it('returns first uncompleted lesson', async () => {
      const course = {
        _id: 'course-1',
        tenantId: 'tenant-a',
        modules: [
          {
            id: 'm1',
            order: 1,
            lessons: [
              {
                id: 'l1',
                title: 'L1',
                order: 1,
                contentType: 'video',
                contentUrl: '',
                isVisible: true,
              },
              {
                id: 'l2',
                title: 'L2',
                order: 2,
                contentType: 'video',
                contentUrl: '',
                isVisible: true,
              },
            ],
          },
        ],
      };
      courseModel.findOne.mockReturnValueOnce({
        lean: () => ({ exec: () => Promise.resolve(course) }),
      });
      progressService.getCompletedLessonIds.mockResolvedValueOnce(['l1']);

      const result = await controller.getNextUp({
        course_id: 'course-1',
        user_id: 'user-1',
        tenant_id: 'tenant-a',
      });

      expect(result.has_next).toBe(true);
      expect(result.module_id).toBe('m1');
      expect(result.lesson.id).toBe('l2');
    });

    it('returns has_next=false when all lessons complete', async () => {
      const course = {
        _id: 'course-1',
        tenantId: 'tenant-a',
        modules: [{ id: 'm1', order: 1, lessons: [{ id: 'l1', order: 1, title: 'L1' }] }],
      };
      courseModel.findOne.mockReturnValueOnce({
        lean: () => ({ exec: () => Promise.resolve(course) }),
      });
      progressService.getCompletedLessonIds.mockResolvedValueOnce(['l1']);

      const result = await controller.getNextUp({
        course_id: 'course-1',
        user_id: 'user-1',
        tenant_id: 'tenant-a',
      });

      expect(result.has_next).toBe(false);
    });
  });

  describe('getCourseAnalytics', () => {
    it('returns analytics shape mapped to proto field names', async () => {
      analyticsService.getCourseAnalytics.mockResolvedValueOnce({
        courseId: 'course-1',
        engagementPct: 72,
        completionPct: 45,
        medianGrade: 82.5,
        atRiskCount: 3,
        updatedAt: '2026-04-22T00:00:00Z',
      });

      const result = await controller.getCourseAnalytics({
        course_id: 'course-1',
        tenant_id: 'tenant-a',
      });

      expect(analyticsService.getCourseAnalytics).toHaveBeenCalledWith('tenant-a', 'course-1');
      expect(result.course_id).toBe('course-1');
      expect(result.engagement_pct).toBe(72);
      expect(result.completion_pct).toBe(45);
      expect(result.median_grade).toBe(82.5);
      expect(result.at_risk_count).toBe(3);
      expect(result.updated_at).toBe('2026-04-22T00:00:00Z');
    });
  });

  describe('getNextLecture', () => {
    it('returns has_next=false default (schedule not yet modeled)', async () => {
      const result = await controller.getNextLecture({
        user_id: 'user-1',
        tenant_id: 'tenant-a',
      });
      expect(result.has_next).toBe(false);
    });
  });
});
