import { Controller, UseInterceptors } from '@nestjs/common';
import { GrpcMethod } from '@nestjs/microservices';
import { CourseService } from './course.service';
import { EnrollmentService } from '../enrollment/enrollment.service';
import { MetricsInterceptor } from '../common/interceptors/metrics.interceptor';

@Controller()
@UseInterceptors(MetricsInterceptor)
export class CourseController {
  constructor(
    private readonly courseService: CourseService,
    private readonly enrollmentService: EnrollmentService,
  ) {}

  // Course CRUD
  @GrpcMethod('CourseService', 'CreateCourse')
  async createCourse(data: any) {
    // The gateway injects the authenticated caller's identity as `user_id`.
    // Clients may also supply `instructor_id` explicitly (e.g. admin portals
    // creating a course on behalf of another instructor).  `instructor_id`
    // wins when present; otherwise fall back to the authenticated `user_id`.
    const course = await this.courseService.createCourse({
      title: data.title,
      description: data.description,
      term: data.term,
      syllabus: data.syllabus,
      instructorId: data.instructor_id || data.user_id,
      metadata: data.metadata,
    });

    return { course: this.toCourseProto(course) };
  }

  @GrpcMethod('CourseService', 'GetCourse')
  async getCourse(data: { courseId: string }) {
    const course = await this.courseService.getCourse(data.courseId);
    return { course: this.toCourseProto(course) };
  }

  @GrpcMethod('CourseService', 'UpdateCourse')
  async updateCourse(data: any) {
    const updates: any = {};
    if (data.title) updates.title = data.title;
    if (data.description) updates.description = data.description;
    if (data.term) updates.term = data.term;
    if (data.syllabus) updates.syllabus = data.syllabus;
    if (data.metadata) updates.metadata = data.metadata;

    const course = await this.courseService.updateCourse(data.courseId, updates);
    return { course: this.toCourseProto(course) };
  }

  @GrpcMethod('CourseService', 'DeleteCourse')
  async deleteCourse(data: { courseId: string }) {
    await this.courseService.deleteCourse(data.courseId);
    return {};
  }

  @GrpcMethod('CourseService', 'ListCourses')
  async listCourses(data: any) {
    const { courses, total } = await this.courseService.listCourses({
      instructorId: data.instructor_id,
      term: data.term,
      isPublished: data.isPublished,
      page: data.page || 1,
      pageSize: data.pageSize || 20,
    });

    return {
      courses: courses.map((c) => this.toCourseProto(c)),
      totalCount: total,
      page: data.page || 1,
      pageSize: data.pageSize || 20,
    };
  }

  // Publishing
  @GrpcMethod('CourseService', 'PublishCourse')
  async publishCourse(data: { courseId: string }) {
    const course = await this.courseService.publishCourse(data.courseId);
    return { course: this.toCourseProto(course) };
  }

  @GrpcMethod('CourseService', 'UnpublishCourse')
  async unpublishCourse(data: { courseId: string }) {
    const course = await this.courseService.unpublishCourse(data.courseId);
    return { course: this.toCourseProto(course) };
  }

  // Enrollment
  @GrpcMethod('CourseService', 'SelfEnroll')
  async selfEnroll(data: any) {
    // Gateway path is `/api/courses/:id/enroll`.  The `:id` segment arrives
    // as `data.id` (path-param injection in conversion.rs).  The authenticated
    // student's identity is injected as `data.user_id` by the auth context.
    // Legacy callers that send explicit `courseId` / `studentId` fields are
    // still supported as fallbacks.
    const enrollment = await this.enrollmentService.selfEnroll(
      data.courseId || data.id,
      data.studentId || data.user_id,
      data.sectionId,
    );
    return { enrollment: this.toEnrollmentProto(enrollment) };
  }

  @GrpcMethod('CourseService', 'InstructorAddStudent')
  async instructorAddStudent(data: any) {
    const enrollment = await this.enrollmentService.instructorAddStudent(
      data.courseId,
      data.studentId,
      data.instructor_id,
      data.sectionId,
    );
    return { enrollment: this.toEnrollmentProto(enrollment) };
  }

  @GrpcMethod('CourseService', 'RemoveEnrollment')
  async removeEnrollment(data: { enrollmentId: string }) {
    await this.enrollmentService.removeEnrollment(data.enrollmentId);
    return {};
  }

  @GrpcMethod('CourseService', 'GetCourseRoster')
  async getCourseRoster(data: any) {
    const { enrollments, totalCount } = await this.enrollmentService.getCourseRoster(
      data.courseId,
      data.sectionId,
      data.status,
    );

    return {
      enrollments: enrollments.map((e) => this.toEnrollmentProto(e)),
      totalCount,
    };
  }

  @GrpcMethod('CourseService', 'GetStudentEnrollments')
  async getStudentEnrollments(data: any) {
    const enrollments = await this.enrollmentService.getStudentEnrollments(
      data.studentId,
      data.term,
      data.status,
    );

    return {
      enrollments: enrollments.map((e) => ({
        enrollment: this.toEnrollmentProto(e.enrollment),
        course: this.toCourseProto(e.course),
      })),
    };
  }

  // Templates
  @GrpcMethod('CourseService', 'CreateCourseTemplate')
  async createCourseTemplate(data: any) {
    const template = await this.courseService.createTemplate({
      name: data.name,
      description: data.description,
      syllabusTemplate: data.syllabusTemplate,
      createdBy: data.createdBy,
      defaultMetadata: data.defaultMetadata,
    });

    return { template: this.toTemplateProto(template) };
  }

  @GrpcMethod('CourseService', 'GetCourseTemplate')
  async getCourseTemplate(data: { template_id: string }) {
    const template = await this.courseService.getTemplate(data.template_id);
    return { template: this.toTemplateProto(template) };
  }

  @GrpcMethod('CourseService', 'ListCourseTemplates')
  async listCourseTemplates(data: any) {
    const { templates, total } = await this.courseService.listTemplates({
      page: data.page || 1,
      pageSize: data.pageSize || 20,
    });

    return {
      templates: templates.map((t) => this.toTemplateProto(t)),
      totalCount: total,
    };
  }

  @GrpcMethod('CourseService', 'CreateCourseFromTemplate')
  async createCourseFromTemplate(data: any) {
    const course = await this.courseService.createCourseFromTemplate(data.template_id, {
      title: data.title,
      term: data.term,
      instructorId: data.instructor_id,
      description: data.description,
    });

    return { course: this.toCourseProto(course) };
  }

  // Prerequisites
  @GrpcMethod('CourseService', 'AddPrerequisite')
  async addPrerequisite(data: any) {
    const course = await this.courseService.addPrerequisite(
      data.courseId,
      data.prerequisiteCourseId,
    );
    return { course: this.toCourseProto(course) };
  }

  @GrpcMethod('CourseService', 'RemovePrerequisite')
  async removePrerequisite(data: any) {
    const course = await this.courseService.removePrerequisite(
      data.courseId,
      data.prerequisiteCourseId,
    );
    return { course: this.toCourseProto(course) };
  }

  @GrpcMethod('CourseService', 'CheckPrerequisites')
  async checkPrerequisites(data: any) {
    const result = await this.courseService.checkPrerequisites(data.courseId, data.studentId);
    return {
      hasPrerequisites: result.hasPrerequisites,
      missingPrerequisiteIds: result.missingPrerequisiteIds,
      missingPrerequisites: result.missingPrerequisites.map((c) => this.toCourseProto(c)),
    };
  }

  // Co-teaching
  @GrpcMethod('CourseService', 'AddCoInstructor')
  async addCoInstructor(data: any) {
    const course = await this.courseService.addCoInstructor(data.courseId, data.co_instructor_id);
    return { course: this.toCourseProto(course) };
  }

  @GrpcMethod('CourseService', 'RemoveCoInstructor')
  async removeCoInstructor(data: any) {
    const course = await this.courseService.removeCoInstructor(
      data.courseId,
      data.co_instructor_id,
    );
    return { course: this.toCourseProto(course) };
  }

  // Sections
  @GrpcMethod('CourseService', 'CreateSection')
  async createSection(data: any) {
    const section = await this.courseService.createSection({
      courseId: data.courseId,
      sectionNumber: data.sectionNumber,
      instructorId: data.instructor_id,
      schedule: data.schedule,
      location: data.location,
      maxStudents: data.maxStudents,
    });

    return { section: this.toSectionProto(section) };
  }

  @GrpcMethod('CourseService', 'GetSection')
  async getSection(data: { sectionId: string }) {
    const section = await this.courseService.getSection(data.sectionId);
    return { section: this.toSectionProto(section) };
  }

  @GrpcMethod('CourseService', 'UpdateSection')
  async updateSection(data: any) {
    const updates: any = {};
    if (data.sectionNumber) updates.sectionNumber = data.sectionNumber;
    if (data.instructor_id) updates.instructorId = data.instructor_id;
    if (data.schedule) updates.schedule = data.schedule;
    if (data.location) updates.location = data.location;
    if (data.maxStudents !== undefined) updates.maxStudents = data.maxStudents;

    const section = await this.courseService.updateSection(data.sectionId, updates);
    return { section: this.toSectionProto(section) };
  }

  @GrpcMethod('CourseService', 'DeleteSection')
  async deleteSection(data: { sectionId: string }) {
    await this.courseService.deleteSection(data.sectionId);
    return {};
  }

  @GrpcMethod('CourseService', 'ListCourseSections')
  async listCourseSections(data: { courseId: string }) {
    const sections = await this.courseService.listCourseSections(data.courseId);
    return {
      sections: sections.map((s) => this.toSectionProto(s)),
    };
  }

  // Cross-listing
  @GrpcMethod('CourseService', 'CrossListCourses')
  async crossListCourses(data: any) {
    const crossListing = await this.courseService.crossListCourses(data.courseIds, data.createdBy);
    return {
      crossListing: {
        id: crossListing._id.toString(),
        groupId: crossListing.groupId,
        courseIds: crossListing.courseIds,
        createdBy: crossListing.createdBy,
        createdAt: crossListing.createdAt,
      },
    };
  }

  @GrpcMethod('CourseService', 'RemoveCrossListing')
  async removeCrossListing(data: { groupId: string }) {
    await this.courseService.removeCrossListing(data.groupId);
    return {};
  }

  @GrpcMethod('CourseService', 'GetCrossListedCourses')
  async getCrossListedCourses(data: { courseId: string }) {
    const { courses, groupId } = await this.courseService.getCrossListedCourses(data.courseId);
    return {
      courses: courses.map((c) => this.toCourseProto(c)),
      groupId,
    };
  }

  // Helper methods to convert Mongoose documents to proto messages
  private toCourseProto(course: any) {
    return {
      id: course._id.toString(),
      title: course.title,
      description: course.description,
      term: course.term,
      syllabus: course.syllabus || '',
      instructor_id: course.instructorId,
      co_instructor_ids: course.coInstructorIds || [],
      isPublished: course.isPublished,
      prerequisiteCourseIds: course.prerequisiteCourseIds || [],
      template_id: course.templateId?.toString() || '',
      crossListingGroupId: course.crossListingGroupId || '',
      createdAt: course.createdAt,
      updatedAt: course.updatedAt,
      metadata: course.metadata || {},
    };
  }

  private toEnrollmentProto(enrollment: any) {
    return {
      id: enrollment._id.toString(),
      courseId: enrollment.courseId.toString(),
      studentId: enrollment.studentId,
      enrollmentType: this.mapEnrollmentType(enrollment.enrollmentType),
      status: this.mapEnrollmentStatus(enrollment.status),
      enrolledBy: enrollment.enrolledBy,
      enrolledAt: enrollment.enrolledAt,
      sectionId: enrollment.sectionId?.toString() || '',
    };
  }

  private toTemplateProto(template: any) {
    return {
      id: template._id.toString(),
      name: template.name,
      description: template.description,
      syllabusTemplate: template.syllabusTemplate || '',
      createdBy: template.createdBy,
      createdAt: template.createdAt,
      defaultMetadata: template.defaultMetadata || {},
    };
  }

  private toSectionProto(section: any) {
    return {
      id: section._id.toString(),
      courseId: section.courseId.toString(),
      sectionNumber: section.sectionNumber,
      instructor_id: section.instructorId,
      schedule: section.schedule,
      location: section.location || '',
      maxStudents: section.maxStudents,
      enrolledCount: section.enrolledCount,
      createdAt: section.createdAt,
    };
  }

  private mapEnrollmentType(type: string): number {
    const mapping = {
      SELF: 1,
      INSTRUCTOR: 2,
      ADMIN: 3,
    };
    return mapping[type] || 0;
  }

  private mapEnrollmentStatus(status: string): number {
    const mapping = {
      ACTIVE: 1,
      DROPPED: 2,
      COMPLETED: 3,
      WAITLISTED: 4,
    };
    return mapping[status] || 0;
  }

  // Health Check
  @GrpcMethod('CourseService', 'HealthCheck')
  async healthCheck() {
    return {
      status: 'healthy',
      service: 'course-service',
      timestamp: new Date().toISOString(),
    };
  }
}
