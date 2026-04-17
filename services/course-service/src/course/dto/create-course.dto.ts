import { IsString, IsOptional, IsArray, IsBoolean, ValidateNested, IsEnum } from 'class-validator';
import { Type } from 'class-transformer';

class CourseSettingsDto {
  @IsBoolean()
  @IsOptional()
  allowSelfEnrollment?: boolean;

  @IsBoolean()
  @IsOptional()
  visibleToStudents?: boolean;
}

export class CreateCourseDto {
  @IsString()
  title: string;

  @IsString()
  @IsOptional()
  description?: string;

  @IsString()
  @IsOptional()
  coverImageUrl?: string;

  @IsString()
  @IsOptional()
  category?: string;

  @IsArray()
  @IsString({ each: true })
  @IsOptional()
  tags?: string[];

  @ValidateNested()
  @Type(() => CourseSettingsDto)
  @IsOptional()
  settings?: CourseSettingsDto;
}

export class UpdateCourseDto {
  @IsString()
  @IsOptional()
  title?: string;

  @IsString()
  @IsOptional()
  description?: string;

  @IsString()
  @IsOptional()
  coverImageUrl?: string;

  @IsString()
  @IsOptional()
  category?: string;

  @IsArray()
  @IsString({ each: true })
  @IsOptional()
  tags?: string[];

  @IsEnum(['draft', 'active', 'locked', 'archived'])
  @IsOptional()
  status?: string;

  @ValidateNested()
  @Type(() => CourseSettingsDto)
  @IsOptional()
  settings?: CourseSettingsDto;
}

export class ListCoursesDto {
  @IsOptional()
  @IsString()
  status?: string;

  @IsOptional()
  @IsString()
  instructorId?: string;

  @IsOptional()
  @IsString()
  category?: string;

  @IsOptional()
  page?: number;

  @IsOptional()
  limit?: number;

  @IsOptional()
  @IsString()
  sortBy?: string;

  @IsOptional()
  @IsEnum(['asc', 'desc'])
  sortOrder?: 'asc' | 'desc';
}
