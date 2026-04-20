# HTTP API Endpoints

etcd-backup-restore exposes HTTP endpoints for managing and monitoring the backup-restore process. All endpoints are served on the configured port (default: 8080).

## Member Management

### Remove Member from Cluster

**Endpoint:** `/member/remove`

**Method:** GET

**Purpose:** Remove a member from the etcd cluster. This is useful for rolling updates, live migrations, and cluster maintenance operations where a member needs to be gracefully removed before it is brought down.

**Use Cases:**
- Rolling updates: Remove a member before updating its etcd instance
- Live migration: Remove a member before migrating it to a different node
- Cluster rebalancing: Remove members as part of scaling down the cluster

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | The name of the etcd cluster member to remove |

**Request Examples:**

```bash
# Remove a member named "etcd-1"
curl http://localhost:8080/member/remove?name=etcd-1

# Remove a member with HTTPS
curl https://localhost:8080/member/remove?name=etcd-2 \
  --cacert /path/to/ca.crt
```

**Response Codes:**

| Status Code | Description |
|-------------|-------------|
| 200 OK | Member successfully removed or was already not present (idempotent operation) |
| 400 Bad Request | Missing or invalid query parameter (e.g., `name` parameter not provided) |
| 500 Internal Server Error | Failed to remove member due to an etcd error or connection issue |

**Response Body (Success - 200):**

```json
{
  "removed": true,
  "memberName": "etcd-1"
}
```

**Response Body (Error - 400):**

```
missing required query parameter: name
```

**Response Body (Error - 500):**

```
failed to remove member: <error details>
```

**Notes:**
- The operation is idempotent. Attempting to remove a member that is not present in the cluster will return HTTP 200 with a successful response.
- TLS support is available when the etcd-backup-restore server is configured with TLS certificates.
- The member name should match exactly with the member's name in the etcd cluster configuration.

**Example Workflow:**

```bash
# Step 1: Remove member from cluster
curl http://localhost:8080/member/remove?name=etcd-1

# Step 2: Stop the etcd instance corresponding to the member
# (perform necessary maintenance or migration)

# Step 3: Optionally, add the member back to the cluster if needed
# (this would be done through etcd's API or cluster management tools)
```
