package executor

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"time"
)

// DemoRunner simulates script execution with realistic output.
// Used for public demos where real execution is disabled.
type DemoRunner struct{}

func NewDemoRunner() *DemoRunner { return &DemoRunner{} }

func (d *DemoRunner) Run(ctx context.Context, scriptPath string, env map[string]string) (<-chan OutputLine, <-chan error) {
	lines := make(chan OutputLine, 100)
	errs := make(chan error, 1)

	// Derive runbook name from script path for context-aware output.
	name := filepath.Base(filepath.Dir(scriptPath))

	go func() {
		defer close(lines)
		defer close(errs)

		for _, line := range demoOutput(name, env) {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			// Simulate realistic output timing.
			jitter, _ := rand.Int(rand.Reader, big.NewInt(120))
		delay := time.Duration(450+jitter.Int64()) * time.Millisecond
			time.Sleep(delay)

			lines <- OutputLine{Text: line}
		}

		errs <- nil
	}()

	return lines, errs
}

func demoOutput(name string, env map[string]string) []string {
	// Check for specific runbook simulations.
	if out, ok := runbookOutputs[name]; ok {
		return resolveEnv(out, env)
	}

	// Generic fallback.
	return []string{
		fmt.Sprintf("$ %s", name),
		"Initializing...",
		"Running pre-flight checks... OK",
		"Executing task...",
		"",
		"Step 1/3: Preparing environment",
		"  → Loading configuration",
		"  → Validating parameters",
		"Step 2/3: Running operation",
		"  → Processing...",
		"  → Done",
		"Step 3/3: Cleanup",
		"  → Removing temporary files",
		"",
		fmt.Sprintf("\033[32m✓\033[0m %s completed successfully", name),
	}
}

// resolveEnv replaces ${VAR} placeholders with env values.
func resolveEnv(lines []string, env map[string]string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		resolved := line
		for k, v := range env {
			resolved = strings.ReplaceAll(resolved, "${"+k+"}", v)
		}
		out[i] = resolved
	}
	return out
}

var runbookOutputs = map[string][]string{
	"hello-world": {
		"$ hello-world",
		"Hello, ${MESSAGE}!",
	},
	"echo": {
		"$ echo",
		"${MESSAGE}",
	},
	"disk-cleanup": {
		"$ disk-cleanup",
		"Scanning /tmp for files older than ${DAYS} days...",
		"",
		"  /tmp/build-cache-a8f3e2    1.2 MB   12 days old",
		"  /tmp/session-9d2c41        0.3 MB   8 days old",
		"  /tmp/upload-tmp-fe2910     4.7 MB   15 days old",
		"  /tmp/npm-cache-2847af      2.1 MB   22 days old",
		"",
		"Found 4 files (8.3 MB total)",
		"Removing files...",
		"",
		"\033[32m✓\033[0m Cleaned up 8.3 MB from /tmp",
	},
	"health-check": {
		"$ health-check",
		"Pinging ${URL}...",
		"",
		"  HTTP/1.1 200 OK",
		"  Content-Type: application/json",
		"  X-Response-Time: 42ms",
		"",
		"\033[32m✓\033[0m Endpoint is healthy (200 OK, 42ms)",
	},
	"network-test": {
		"$ network-test",
		"Testing TCP connection to ${HOST}:${PORT}...",
		"",
		"  Resolving ${HOST}... 93.184.216.34",
		"  Connecting to 93.184.216.34:${PORT}...",
		"  Connected in 23ms",
		"  TLS handshake completed (TLS 1.3)",
		"",
		"\033[32m✓\033[0m Connection successful (23ms)",
	},
	"encode-base64": {
		"$ encode-base64",
		"Input:  ${INPUT}",
		"Mode:   ${MODE}",
		"",
		"Output: ZGVtbyBvdXRwdXQ=",
	},
	"generate-password": {
		"$ generate-password",
		"Generating password (length: ${LENGTH}, charset: ${CHARSET})...",
		"",
		"  k9#mP2$xR7!vN4@qW8&bL5",
		"",
		"\033[32m✓\033[0m Password generated (entropy: 142 bits)",
	},
	"json-format": {
		"$ json-format",
		"Formatting JSON...",
		"",
		"{",
		`  "name": "dops",`,
		`  "version": "1.0.0",`,
		`  "status": "running"`,
		"}",
	},
	"git-summary": {
		"$ git-summary",
		"Repository: ${REPO_PATH}",
		"Branch:     main",
		"",
		"Recent commits:",
		"  a8f3e2c  Add live demo playground",
		"  d92c41b  Fix catalog config for demo",
		"  fe2910a  Update README with demo links",
		"  2847afc  Add animated hero logo",
		"",
		"4 commits in the last 7 days",
		"2 contributors",
	},
	"dns-lookup": {
		"$ dns-lookup",
		"Querying ${RECORD_TYPE} records for ${DOMAIN}...",
		"",
		"  ${DOMAIN}    300    IN    A       93.184.216.34",
		"  ${DOMAIN}    300    IN    AAAA    2606:2800:220:1:248:1893:25c8:1946",
		"",
		"\033[32m✓\033[0m 2 records found",
	},
	"ssl-check": {
		"$ ssl-check",
		"Checking SSL certificate for ${DOMAIN}...",
		"",
		"  Subject:    ${DOMAIN}",
		"  Issuer:     Let's Encrypt Authority X3",
		"  Valid from: 2026-01-15",
		"  Expires:    2026-04-15 (17 days remaining)",
		"  Protocol:   TLS 1.3",
		"",
		"\033[32m✓\033[0m Certificate is valid",
	},
	"disk-usage": {
		"$ disk-usage",
		"Checking disk usage...",
		"",
		"  Filesystem      Size    Used    Avail   Use%    Mounted on",
		"  /dev/sda1       50G     32G     16G     67%     /",
		"  /dev/sda2       200G    89G     101G    47%     /home",
		"  tmpfs           8G      0.2G    7.8G    3%      /tmp",
		"  /dev/sdb1       500G    312G    163G    66%     /var",
		"",
		"\033[32m✓\033[0m All filesystems within thresholds",
	},
	"service-status": {
		"$ service-status",
		"Checking service: ${SERVICE}...",
		"",
		"  Status:     \033[32mactive (running)\033[0m",
		"  PID:        2847",
		"  Uptime:     14 days, 6 hours",
		"  Memory:     128 MB",
		"  CPU:        0.3%",
	},
	"restart-pods": {
		"$ restart-pods",
		"Rolling restart: ${DEPLOYMENT} in namespace ${NAMESPACE}",
		"",
		"  Pod ${DEPLOYMENT}-7f8d9-abc12   \033[33mTerminating\033[0m",
		"  Pod ${DEPLOYMENT}-7f8d9-def34   \033[32mRunning\033[0m",
		"  Pod ${DEPLOYMENT}-7f8d9-abc12   \033[32mRunning\033[0m      (new)",
		"  Pod ${DEPLOYMENT}-7f8d9-ghi56   \033[33mTerminating\033[0m",
		"  Pod ${DEPLOYMENT}-7f8d9-ghi56   \033[32mRunning\033[0m      (new)",
		"",
		"\033[32m✓\033[0m Rolling restart complete (2/2 pods restarted)",
	},
	"log-tail": {
		"$ log-tail",
		"Tailing last ${LINES} lines from ${SERVICE}...",
		"",
		"  2026-03-29 14:22:01 [INFO]  Request processed in 12ms",
		"  2026-03-29 14:22:03 [INFO]  Health check passed",
		"  2026-03-29 14:22:05 [WARN]  Connection pool at 80% capacity",
		"  2026-03-29 14:22:08 [INFO]  Cache hit ratio: 94.2%",
		"  2026-03-29 14:22:10 [INFO]  Request processed in 8ms",
	},
	"canary-deploy": {
		"$ canary-deploy",
		"Deploying v${VERSION} to ${ENVIRONMENT} (canary ${PERCENTAGE}%)",
		"",
		"  Building image... done",
		"  Pushing to registry... done",
		"  Updating canary deployment...",
		"    → Routing ${PERCENTAGE}% traffic to v${VERSION}",
		"    → Monitoring error rate... 0.02% (threshold: 1%)",
		"    → Monitoring latency p99... 145ms (threshold: 500ms)",
		"",
		"\033[32m✓\033[0m Canary deployment healthy",
	},
	"deploy-app": {
		"$ deploy-app",
		"Deploying ${APP_NAME} v${VERSION} to ${ENVIRONMENT}",
		"",
		"  Step 1/4: Building...",
		"    → Compiling application",
		"    → Build successful (12.4s)",
		"  Step 2/4: Testing...",
		"    → Running test suite",
		"    → 142 passed, 0 failed (8.1s)",
		"  Step 3/4: Deploying...",
		"    → Pushing image to registry",
		"    → Updating deployment manifest",
		"    → Rolling out pods (3/3 ready)",
		"  Step 4/4: Verifying...",
		"    → Health check passed",
		"    → Smoke tests passed",
		"",
		"\033[32m✓\033[0m Deployment complete",
	},
	"rollback": {
		"$ rollback",
		"Rolling back ${SERVICE} to ${TARGET_VERSION}",
		"",
		"  Current version: v1.2.3",
		"  Target version:  ${TARGET_VERSION}",
		"",
		"  → Scaling down current deployment",
		"  → Restoring ${TARGET_VERSION} manifest",
		"  → Rolling out pods (3/3 ready)",
		"  → Health check passed",
		"",
		"\033[32m✓\033[0m Rollback to ${TARGET_VERSION} complete",
	},
	"scale-deployment": {
		"$ scale-deployment",
		"Scaling ${DEPLOYMENT} to ${REPLICAS} replicas in namespace ${NAMESPACE}",
		"",
		"  Current replicas: 2",
		"  Target replicas:  ${REPLICAS}",
		"",
		"  → Updating replica count",
		"  → Waiting for pods to be ready...",
		"  → ${REPLICAS}/${REPLICAS} pods running",
		"",
		"\033[32m✓\033[0m Scaled to ${REPLICAS} replicas",
	},
	"drain-node": {
		"$ drain-node",
		"Draining node ${NODE}",
		"",
		"  → Cordoning ${NODE}",
		"  → Evicting pods...",
		"    pod/api-server-7f8d9-abc12       evicted",
		"    pod/worker-5c4e8-def34           evicted",
		"    pod/cache-3b2a1-ghi56            evicted",
		"  → Waiting for graceful shutdown...",
		"",
		"\033[32m✓\033[0m Node ${NODE} drained (3 pods evicted)",
	},
	"build-image": {
		"$ build-image",
		"Building Docker image: ${IMAGE_NAME}:${TAG}",
		"",
		"  Step 1/6: FROM golang:1.26-alpine",
		"  Step 2/6: COPY go.mod go.sum ./",
		"  Step 3/6: RUN go mod download",
		"  Step 4/6: COPY . .",
		"  Step 5/6: RUN go build -o /app",
		"  Step 6/6: ENTRYPOINT [\"/app\"]",
		"",
		"  Image: ${IMAGE_NAME}:${TAG}",
		"  Size:  24.7 MB",
		"",
		"\033[32m✓\033[0m Image built successfully",
	},
	"backup-database": {
		"$ backup-database",
		"Backing up ${DB_NAME} (format: ${FORMAT})",
		"",
		"  → Connecting to ${DB_HOST}:${DB_PORT}...",
		"  → Starting backup...",
		"  → Dumping tables... 14/14",
		"  → Compressing backup...",
		"",
		"  Backup: /backups/${DB_NAME}_2026-03-29.dump",
		"  Size:   42.8 MB",
		"",
		"\033[32m✓\033[0m Database backup complete",
	},
	"rotate-certs": {
		"$ rotate-certs",
		"Rotating TLS certificates for ${CLUSTER}",
		"",
		"  → Generating new certificate (${KEY_SIZE}-bit RSA)",
		"  → Signing with CA",
		"  → Deploying to cluster nodes...",
		"    node-1: \033[32mupdated\033[0m",
		"    node-2: \033[32mupdated\033[0m",
		"    node-3: \033[32mupdated\033[0m",
		"  → Verifying TLS handshake... OK",
		"",
		"\033[32m✓\033[0m Certificates rotated (expires: 2027-03-29)",
	},
	"rotate-secrets": {
		"$ rotate-secrets",
		"Rotating secrets in ${VAULT_PATH}",
		"",
		"  → Generating new API key",
		"  → Encrypting with vault seal",
		"  → Writing to secret store",
		"  → Invalidating old key",
		"",
		"\033[32m✓\033[0m Secret rotated successfully",
	},
	"count-lines": {
		"$ count-lines",
		"Counting lines from ${FROM} to ${TO}...",
		"",
		"  Total lines: 42",
	},
	"calculate-cost": {
		"$ calculate-cost",
		"Estimating compute cost...",
		"",
		"  Instance type: ${INSTANCE_TYPE}",
		"  vCPUs:         ${VCPUS}",
		"  Hours/month:   ${HOURS}",
		"",
		"  Estimated cost: $127.40/month",
	},
	"check-path": {
		"$ check-path",
		"Checking path: ${FILE_PATH}",
		"",
		"  Type:     regular file",
		"  Size:     4.2 KB",
		"  Modified: 2026-03-28 09:14:22",
		"  Perms:    -rw-r--r--",
		"",
		"\033[32m✓\033[0m Path exists",
	},
	"describe-resource": {
		"$ describe-resource",
		"Looking up resource: ${RESOURCE_ID}",
		"",
		"  Type:     ec2/instance",
		"  Region:   us-east-1",
		"  State:    running",
		"  Created:  2026-02-10",
		"  Tags:     env=production, team=platform",
	},
	"quick-note": {
		"$ quick-note",
		"Appending note to log...",
		"",
		"  [2026-03-29 14:30:00] ${NOTE}",
		"",
		"\033[32m✓\033[0m Note saved",
	},
	"audit-permissions": {
		"$ audit-permissions",
		"Scanning IAM policies for ${ACCOUNT}...",
		"",
		"  Checking 23 policies...",
		"",
		"  \033[33m⚠\033[0m  policy/admin-full-access    — wildcard (*) resource",
		"  \033[32m✓\033[0m  policy/deploy-role           — scoped correctly",
		"  \033[32m✓\033[0m  policy/read-only             — scoped correctly",
		"  \033[33m⚠\033[0m  policy/legacy-lambda         — unused for 90+ days",
		"",
		"21 policies OK, 2 warnings, 0 critical",
	},
}
