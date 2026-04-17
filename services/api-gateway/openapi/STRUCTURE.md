# OpenAPI Documentation Structure

## Overview

The OpenAPI specification has been modularized to improve maintainability and readability. The main `openapi.yaml` file now references external path definition files.

## File Structure

```
services/api-gateway/openapi/
├── openapi.yaml              # Main OpenAPI specification file
├── openapi-backup.yaml       # Backup of the original monolithic file
├── paths/                    # Modular path definitions
│   ├── README.md            # Documentation for path files
│   ├── auth.yaml
│   ├── users.yaml
│   ├── profile.yaml
│   ├── roles.yaml
│   ├── courses.yaml
│   ├── enrollments.yaml
│   ├── course-templates.yaml
│   ├── course-prerequisites.yaml
│   └── course-co-instructors.yaml
├── schemas/                  # Modular schema definitions
│   ├── README.md            # Documentation for schema files
│   └── user-auth-schemas.yaml
├── README.md                # General OpenAPI documentation
└── STRUCTURE.md             # This file
```

## Main File (openapi.yaml)

The main file contains:

- API metadata (title, description, version)
- Server configurations
- Security schemes
- Component schemas (User, Course, Enrollment, etc.)
- Component responses (error responses)
- Tags
- Path references to external files

## Path Files

### course-templates.yaml

Documents course template management endpoints:

- **POST /api/coursetemplates** - Create a new course template
- **GET /api/coursetemplates** - List all course templates (paginated)
- **GET /api/coursetemplates/{template_id}** - Get a specific template
- **POST /api/coursetemplates/{template_id}/courses** - Create a course from a template

### course-prerequisites.yaml

Documents course prerequisite management endpoints:

- **POST /api/courses/{course_id}/prerequisites** - Add a prerequisite to a course
- **DELETE /api/courses/{course_id}/prerequisites/{prerequisite_id}** - Remove a prerequisite
- **POST /api/courses/{course_id}/prerequisites/check** - Check if a student meets prerequisites

### course-co-instructors.yaml

Documents course co-instructor management endpoints:

- **POST /api/courses/{course_id}/co-instructors** - Add a co-instructor to a course
- **DELETE /api/courses/{course_id}/co-instructors/{co_instructor_id}** - Remove a co-instructor

## Benefits of Modular Structure

1. **Maintainability**: Easier to find and update specific endpoints
2. **Readability**: Smaller files are easier to review
3. **Collaboration**: Multiple developers can work on different path files simultaneously
4. **Organization**: Related endpoints are grouped together
5. **Scalability**: Easy to add new endpoint groups without bloating the main file

## How References Work

The main `openapi.yaml` file uses JSON Pointer syntax to reference paths in external files:

```yaml
paths:
  /api/coursetemplates:
    $ref: './paths/course-templates.yaml#/~1api~1coursetemplates'
```

- `./paths/course-templates.yaml` - Relative path to the file
- `#/` - JSON Pointer root
- `~1` - Encoded `/` character (JSON Pointer encoding)
- `api~1coursetemplates` - The path `/api/coursetemplates` encoded

## Schema References

Path files reference schemas from the main file or schema files:

```yaml
# Reference from main file
schema:
  $ref: '../openapi.yaml#/components/schemas/Course'

# Reference from schema file (via main file)
schema:
  $ref: '../openapi.yaml#/components/schemas/LoginRequest'
```

The main file references schemas from the schemas directory:

```yaml
components:
  schemas:
    LoginRequest:
      $ref: './schemas/user-auth-schemas.yaml#/LoginRequest'
```

This allows:

- Modular schema organization by service
- Centralized schema definitions
- Consistent data models across all endpoints
- Easy schema updates in one location

## Adding New Endpoints

To add new endpoints:

1. Create a new YAML file in `paths/` directory
2. Define your endpoints following the existing structure
3. Add path references in the main `openapi.yaml` file
4. Update `paths/README.md` with the new endpoints
5. Test the complete specification with a validator

## Validation

To validate the complete OpenAPI specification:

```bash
# Using npx and swagger-cli
npx @apidevtools/swagger-cli validate services/api-gateway/openapi/openapi.yaml

# Using Docker and openapi-generator
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli validate \
  -i /local/services/api-gateway/openapi/openapi.yaml
```

## Migration Notes

- The original monolithic file is preserved as `openapi-backup.yaml`
- All existing endpoint definitions remain unchanged
- New endpoints (templates, prerequisites, co-instructors) are in separate files
- Future work: Extract remaining endpoints (auth, users, courses, enrollments) into separate files

## Schemas Directory

The `schemas/` directory contains modular schema definitions organized by service:

### user-auth-schemas.yaml

Contains all User Auth Service schemas generated from `proto/user.proto`:

- Authentication schemas (Login, Register, Token operations)
- OAuth schemas (Provider info, authorization flow)
- SAML schemas (Authentication, metadata)
- MFA schemas (Setup, validation, status)
- Group schemas (Group management, members)
- Parent-Child relationship schemas

## Future Improvements

### Path Files

Consider extracting these endpoint groups into separate files:

- `paths/sections.yaml` - Section management endpoints
- `paths/cross-listing.yaml` - Cross-listing endpoints
- `paths/assignments.yaml` - Assignment management endpoints
- `paths/content.yaml` - Content management endpoints

### Schema Files

Consider creating additional schema files:

- `schemas/course-schemas.yaml` - Course Service schemas from proto/course.proto
- `schemas/assignment-schemas.yaml` - Assignment Service schemas from proto/assignment.proto
- `schemas/content-schemas.yaml` - Content Service schemas from proto/content.proto

This would make the specification even more modular and maintainable.
