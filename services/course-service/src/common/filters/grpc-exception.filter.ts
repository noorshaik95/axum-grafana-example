import { Catch, ArgumentsHost, Logger } from '@nestjs/common';
import {
  BadRequestException,
  ConflictException,
  NotFoundException,
  ForbiddenException,
  UnauthorizedException,
} from '@nestjs/common';
import { BaseRpcExceptionFilter, RpcException } from '@nestjs/microservices';
import { Observable, throwError } from 'rxjs';
import { status as GrpcStatus } from '@grpc/grpc-js';

/**
 * Global gRPC exception filter for course-service.
 *
 * NestJS's default gRPC error handler wraps every unhandled exception as
 * `Unknown` with no diagnostic message.  This filter converts:
 *   - NotFoundException          → NOT_FOUND
 *   - BadRequestException        → INVALID_ARGUMENT
 *   - ConflictException          → ALREADY_EXISTS
 *   - ForbiddenException         → PERMISSION_DENIED
 *   - UnauthorizedException      → UNAUTHENTICATED
 *   - Mongoose ValidationError   → INVALID_ARGUMENT (required-field failures)
 *   - Mongoose CastError         → INVALID_ARGUMENT (bad ObjectId)
 *   - RpcException (pass-through)
 *   - everything else            → INTERNAL (with the real message)
 *
 * Registered in main.ts via `grpcApp.useGlobalFilters(...)`.
 */
@Catch()
export class GrpcExceptionFilter extends BaseRpcExceptionFilter {
  private readonly logger = new Logger(GrpcExceptionFilter.name);

  override catch(exception: unknown, _host: ArgumentsHost): Observable<never> {
    const rpcError = this.toRpcError(exception);
    return throwError(() => rpcError);
  }

  private toRpcError(exception: unknown): { code: number; message: string } {
    // Already a structured RpcException — unwrap and pass through.
    if (exception instanceof RpcException) {
      const err = exception.getError();
      if (typeof err === 'object' && err !== null && 'code' in err && 'message' in err) {
        return err as { code: number; message: string };
      }
      return { code: GrpcStatus.INTERNAL, message: String(err) };
    }

    if (exception instanceof NotFoundException) {
      return { code: GrpcStatus.NOT_FOUND, message: exception.message };
    }

    if (exception instanceof ConflictException) {
      return { code: GrpcStatus.ALREADY_EXISTS, message: exception.message };
    }

    if (exception instanceof BadRequestException) {
      const res = exception.getResponse();
      const msg =
        typeof res === 'object' && res !== null && 'message' in res
          ? Array.isArray((res as any).message)
            ? (res as any).message.join('; ')
            : String((res as any).message)
          : exception.message;
      return { code: GrpcStatus.INVALID_ARGUMENT, message: msg };
    }

    if (exception instanceof ForbiddenException) {
      return { code: GrpcStatus.PERMISSION_DENIED, message: exception.message };
    }

    if (exception instanceof UnauthorizedException) {
      return { code: GrpcStatus.UNAUTHENTICATED, message: exception.message };
    }

    // Mongoose ValidationError (required fields, enum violations, etc.)
    if (exception instanceof Error && exception.constructor.name === 'ValidationError') {
      return {
        code: GrpcStatus.INVALID_ARGUMENT,
        message: `Validation failed: ${exception.message}`,
      };
    }

    // Mongoose CastError (bad ObjectId passed as a required field)
    if (exception instanceof Error && exception.constructor.name === 'CastError') {
      return {
        code: GrpcStatus.INVALID_ARGUMENT,
        message: `Invalid value: ${exception.message}`,
      };
    }

    // Unknown / unhandled — surface as INTERNAL with the real message so
    // the gateway can log something useful instead of a bare "Internal server
    // error".
    const message = exception instanceof Error ? exception.message : String(exception);

    this.logger.error(
      `Unhandled gRPC exception (surfaced as INTERNAL): ${message}`,
      exception instanceof Error ? exception.stack : undefined,
    );

    return { code: GrpcStatus.INTERNAL, message };
  }
}
