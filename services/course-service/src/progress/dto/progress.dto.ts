import { IsString, IsOptional, IsNumber } from 'class-validator';

export class UpdateProgressDto {
  @IsString()
  lessonId: string;

  @IsNumber()
  @IsOptional()
  timeSpentMinutes?: number;
}
