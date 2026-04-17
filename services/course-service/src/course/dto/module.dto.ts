import {
  IsString,
  IsOptional,
  IsNumber,
  IsBoolean,
  IsEnum,
  IsDateString,
  IsArray,
  ValidateNested,
} from 'class-validator';
import { Type } from 'class-transformer';

export class CreateModuleDto {
  @IsString()
  title: string;

  @IsString()
  @IsOptional()
  description?: string;

  @IsNumber()
  @IsOptional()
  order?: number;

  @IsBoolean()
  @IsOptional()
  isVisible?: boolean;
}

export class UpdateModuleDto {
  @IsString()
  @IsOptional()
  title?: string;

  @IsString()
  @IsOptional()
  description?: string;

  @IsNumber()
  @IsOptional()
  order?: number;

  @IsBoolean()
  @IsOptional()
  isVisible?: boolean;
}

export class CreateLessonDto {
  @IsString()
  title: string;

  @IsNumber()
  @IsOptional()
  order?: number;

  @IsEnum(['video', 'pdf', 'link', 'text'])
  @IsOptional()
  contentType?: string;

  @IsString()
  @IsOptional()
  contentUrl?: string;

  @IsBoolean()
  @IsOptional()
  isVisible?: boolean;

  @IsDateString()
  @IsOptional()
  visibleAfter?: string;
}

export class UpdateLessonDto {
  @IsString()
  @IsOptional()
  title?: string;

  @IsNumber()
  @IsOptional()
  order?: number;

  @IsEnum(['video', 'pdf', 'link', 'text'])
  @IsOptional()
  contentType?: string;

  @IsString()
  @IsOptional()
  contentUrl?: string;

  @IsBoolean()
  @IsOptional()
  isVisible?: boolean;

  @IsDateString()
  @IsOptional()
  visibleAfter?: string;
}

class ReorderItem {
  @IsString()
  moduleId: string;

  @IsNumber()
  order: number;
}

export class ReorderModulesDto {
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => ReorderItem)
  items: ReorderItem[];
}
