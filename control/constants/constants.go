package constants

const (
	// config
	DEFAULT_CONFIG_DIR               = ".benchctl"
	DEFAULT_CONFIG_FILE              = "config.json"
	DEFAULT_KEY_FILE                 = "keys.txt"
	DEFAULT_SEED               int64 = 0x207B096061CDA310
	DEFAULT_NUM_KEYS           int   = 1_000_000 // 1 million keys
	MIN_KEY_SIZE               int   = 12
	SCENARIO_KV_STORE                = "kv-store"
	SCENARIO_LOCK_SERVICE            = "lock-service"
	WORKLOAD_TYPE_READ_HEAVY         = "read-heavy"   // 95% reads, 5% writes
	WORKLOAD_TYPE_UPDATE_HEAVY       = "update-heavy" // 50% reads, 50% writes
	WORKLOAD_TYPE_READ_ONLY          = "read-only"    // 100% reads

	// grpc
	DEFAULT_GRPC_SERVER_PORT = 50051

	// metrics
	DEFAULT_METRICS_BATCH_SIZE = 1000
)
