import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { v4 as uuidv4 } from 'uuid';

@Schema({ _id: false })
export class Lesson {
  @Prop({ default: () => uuidv4() })
  id: string;

  @Prop({ required: true })
  title: string;

  @Prop({ default: 0 })
  order: number;

  @Prop({ enum: ['video', 'pdf', 'link', 'text'], default: 'text' })
  contentType: string;

  @Prop()
  contentUrl: string;

  @Prop({ default: true })
  isVisible: boolean;

  @Prop({ type: Date, default: null })
  visibleAfter: Date | null;
}

export const LessonSchema = SchemaFactory.createForClass(Lesson);

@Schema({ _id: false })
export class Module {
  @Prop({ default: () => uuidv4() })
  id: string;

  @Prop({ required: true })
  title: string;

  @Prop()
  description: string;

  @Prop({ default: 0 })
  order: number;

  @Prop({ type: [LessonSchema], default: [] })
  lessons: Lesson[];

  @Prop({ default: true })
  isVisible: boolean;
}

export const ModuleSchema = SchemaFactory.createForClass(Module);
