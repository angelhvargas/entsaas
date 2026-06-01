# REST API Reference

Welcome to the EntSaaS REST API documentation. The framework includes a high-performance REST API supporting JWT authentication, dynamic JSON payloads, and streaming endpoints.

---

## 1. Authentication

All requests to non-public endpoints must include a valid JWT Access Token inside the standard `Authorization` HTTP header:

```http
Authorization: Bearer <your_access_token>
```

---

## 2. API Endpoints

### 2.1 Registration
- **URL**: `/v1/auth/register`
- **Method**: `POST`
- **Exposure**: `public-ga`
- **Description**: Registers a new user account and creates their primary organization context.

### 2.2 Login
- **URL**: `/v1/auth/login`
- **Method**: `POST`
- **Exposure**: `public-ga`
- **Description**: Verifies credentials and returns access/refresh JWT tokens.

### 2.3 Current Session Profile
- **URL**: `/v1/auth/me`
- **Method**: `GET`
- **Exposure**: `public-ga`
- **Description**: Fetches current user claims and organization billing statuses.

### 2.4 List Projects
- **URL**: `/v1/projects`
- **Method**: `GET`
- **Exposure**: `public-preview`
- **Description**: Fetches an array of projects active under the tenant organization.

### 2.5 Create Project
- **URL**: `/v1/projects`
- **Method**: `POST`
- **Exposure**: `public-preview`
- **Description**: Creates a new project database record. Subject to maximum plan limits.

### 2.6 Streaming AI Chat
- **URL**: `/v1/ai/chat`
- **Method**: `POST`
- **Exposure**: `public-preview`
- **Description**: Starts an interactive Server-Sent Events stream for AI prompts.
