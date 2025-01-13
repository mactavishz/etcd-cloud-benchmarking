package constants

const (
	// config
	DEFAULT_CONFIG_DIR        = ".benchctl"
	DEFAULT_CONFIG_FILE       = "config.json"
	DEFAULT_KEY_FILE          = "keys.txt"
	DEFAULT_SEED        int64 = 0x207B096061CDA310
	DEFAULT_NUM_KEYS    int   = 1_000_000 // 1 million keys
	MIN_KEY_SIZE        int   = 12

	// Different scenarios that can be run with corresponding workload types
	SCENARIO_KV_STORE     = "kv-store"
	SCENARIO_LOCK_SERVICE = "lock-service"

	// The following workload types are specific to the kv-store scenario
	WORKLOAD_TYPE_READ_HEAVY   = "read-heavy"   // 95% reads, 5% writes
	WORKLOAD_TYPE_UPDATE_HEAVY = "update-heavy" // 50% reads, 50% writes
	WORKLOAD_TYPE_READ_ONLY    = "read-only"    // 100% reads

	// The following workload types are specific to the lock-service scenario
	WORKLOAD_TYPE_LOCK_ONLY        = "lock-only"        // 100% lock operations
	WORKLOAD_TYPE_LOCK_MIXED_READ  = "lock-mixed-read"  // all read/write opeartions performed under lock
	WORKLOAD_TYPE_LOCK_MIXED_WRITE = "lock-mixed-write" // all read/write opeartions performed under lock
	WORKLOAD_TYPE_LOCK_CONTENTION  = "lock-contention"  // all clients contending for a  set of locks

	// grpc
	DEFAULT_GRPC_SERVER_PORT   = 50051
	DEFAULT_BENCH_RUN_LOG_FILE = "run.log"

	// metrics
	DEFAULT_METRICS_BATCH_SIZE = 1000
)
