# Help & Support Center

We are here to help you get the most out of the EntSaaS framework.

---

## 1. Frequently Asked Questions

### 1.1 How do I sync new plans in billing.yaml?
Simply execute the admin synchronization CLI tool:
```bash
./entsaasctl billing sync
```
This parses your local YAML plan metadata and safely updates active PostgreSQL structures without severing current customer subscriptions.

### 1.2 How do I add a new API route?
1. Write your database migrations in `/internal/migrations/postgres`.
2. Add store interfaces in `/internal/store/interfaces.go` and implement the methods in `postgres.go`.
3. Create your handler routing logic in `/internal/handlers`.
4. Register the HTTP endpoint routes inside `/internal/api`.

---

## 2. Developer Resources

- **GitHub Repository**: [https://github.com/angelhvargas/entsaas](https://github.com/angelhvargas/entsaas)
- **Technical Inquiries**: Support tickets and operational escalations can be managed directly by contacting support representatives inside your commercial portal.
