import { Test, TestingModule } from '@nestjs/testing';
import { ConfigService } from '@nestjs/config';
import { KafkaProducerService } from './kafka.producer';

jest.mock('kafkajs', () => {
  const mockProducer = {
    connect: jest.fn().mockResolvedValue(undefined),
    disconnect: jest.fn().mockResolvedValue(undefined),
    send: jest.fn().mockResolvedValue(undefined),
  };
  return {
    Kafka: jest.fn().mockImplementation(() => ({
      producer: jest.fn().mockReturnValue(mockProducer),
    })),
  };
});

describe('KafkaProducerService', () => {
  let service: KafkaProducerService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        KafkaProducerService,
        {
          provide: ConfigService,
          useValue: {
            get: jest.fn().mockReturnValue('localhost:9092'),
          },
        },
      ],
    }).compile();

    service = module.get<KafkaProducerService>(KafkaProducerService);
    await service.onModuleInit();
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  it('should emit course created event', async () => {
    await service.emitCourseCreated({
      courseId: 'course-1',
      tenantId: 'tenant-1',
      instructorId: 'instructor-1',
      title: 'Test Course',
    });
    // No error means success (producer is mocked)
  });

  it('should emit course updated event', async () => {
    await service.emitCourseUpdated({
      courseId: 'course-1',
      tenantId: 'tenant-1',
      changes: ['title'],
    });
  });

  it('should emit course locked event', async () => {
    await service.emitCourseLocked({
      courseId: 'course-1',
      tenantId: 'tenant-1',
    });
  });

  it('should emit course deleted event', async () => {
    await service.emitCourseDeleted({
      courseId: 'course-1',
      tenantId: 'tenant-1',
    });
  });

  it('should emit lesson completed event', async () => {
    await service.emitLessonCompleted({
      studentId: 'student-1',
      courseId: 'course-1',
      lessonId: 'lesson-1',
      tenantId: 'tenant-1',
      completionPct: 25,
    });
  });

  it('should emit user enrolled event', async () => {
    await service.emitUserEnrolled({
      studentId: 'student-1',
      courseId: 'course-1',
      tenantId: 'tenant-1',
    });
  });
});
