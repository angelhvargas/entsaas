# Authentication & Role-Based Access Control

EntSaaS features a comprehensive, secure identity layer supporting JWT access tokens, token rotation, and robust Role-Based Access Control (RBAC).

---

## 1. Authentication Lifecycle

The authentication system runs entirely on stateless JSON Web Tokens (JWT) signed with HS256:

### 1.1 Double-Token Issuance
Upon successful login or registration, the backend issues two tokens:
1. **Access Token**: A short-lived (e.g. 15-minute) signature containing the user's ID, active `org_id`, and role claim.
2. **Refresh Token**: A long-lived (e.g. 7-day) cryptographically random token stored in the database, used to renew access tokens automatically.

### 1.2 Client-Side Axios Interceptor Rotation
The Vue 3 client Axios wrapper ([client.js](web/src/api/client.js)) handles token expirations seamlessly:
- When an API request fails with a `401 Unauthorized` status, the client intercepts the failure.
- It pauses the request queue, dispatches an asynchronous token refresh call (`POST /v1/auth/refresh`), updates the local token, and transparently re-fires the pending requests.

---

## 2. Role-Based Access Control (RBAC)

Permission states are evaluated by the backend middleware and represent a strict hierarchical scale defined in [rbac.go](internal/auth/rbac.go):

| Role | Access Scope | Code Constant |
| :--- | :--- | :--- |
| **Owner** | Full organization control, ownership transfers, billing portals. | `RoleOwner` |
| **Admin** | Managing user accounts, team invitations, and general settings. | `RoleAdmin` |
| **Member** | Standard database resource CRUD (create projects, view data). | `RoleMember` |
| **Viewer** | Read-only observation of workspace resources. | `RoleViewer` |

### Enforcing Roles in Endpoints:
```go
// Register team invite endpoints, restricted strictly to Admins and Owners
adminGroup := r.Group("/v1/admin")
adminGroup.Use(middleware.RequireRole(auth.RoleAdmin))
{
    adminGroup.POST("/invite", invitesHandler.Create)
}
```
