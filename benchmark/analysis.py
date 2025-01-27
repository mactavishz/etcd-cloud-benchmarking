import polars as pl

# import numpy as np
import seaborn as sns
import matplotlib.pyplot as plt
import argparse
import os
from pathlib import Path
from typing import List, Dict


class EtcdPerfAnalyzer:
    node_configs = ["1-node", "3-node", "5-node"]
    kv_schema = pl.Schema(
        {
            "unix_timestamp_nano": pl.UInt64(),
            "key": pl.String(),
            "opeartion": pl.String(),
            "latency_ms": pl.UInt32(),
            "success": pl.Boolean(),
            "status_code": pl.Int16(),
            "status_text": pl.String(),
            "num_clients": pl.UInt16(),
            "client_id": pl.UInt16(),
            "run_phase": pl.String(),
        }
    )

    # Define lock_dtypes schema, extending kv_schema
    lock_schema = pl.Schema(
        {
            "unix_timestamp_nano": pl.UInt64(),
            "key": pl.String(),
            "opeartion": pl.String(),
            "latency_ms": pl.UInt32(),
            "success": pl.Boolean(),
            "status_code": pl.Int16(),
            "status_text": pl.String(),
            "num_clients": pl.UInt16(),
            "client_id": pl.UInt16(),
            "run_phase": pl.String(),
            "lock_name": pl.String(),
            "aquire_latency_ms": pl.UInt32(),
            "release_latency_ms": pl.UInt32(),
            "lock_op_status_code": pl.Int16(),
            "lock_op_status_text": pl.String(),
            "contention_level": pl.Int16(),
        }
    )

    def __init__(self, base_path: str):
        self.base_path = Path(base_path)
        sns.set_theme(style="whitegrid")
        plt.rcParams["figure.figsize"] = [12, 6]

    def load_metrics(self, path: Path, scenario: str) -> pl.DataFrame:
        df: pl.DataFrame = pl.read_csv(
            path,
            schema=EtcdPerfAnalyzer.lock_schema
            if scenario == "lock"
            else EtcdPerfAnalyzer.kv_schema,
        )
        df = (
            df.with_columns(
                (pl.col("unix_timestamp_nano"))
                .cast(pl.Datetime(time_unit="ns"))
                .alias("timestamp")
            )
            .filter(pl.col("run_phase") == "main")
            .select(
                pl.all().exclude(["unix_timestamp_nano", "status_text", "run_phase"])
            )
            .sort("timestamp")
        )
        return df

    def calculate_latency_metrics(
        self, df: pl.DataFrame, sample_rate: str = "1m"
    ) -> pl.DataFrame:
        """Calculate latency metrics with specified window."""
        return (
            df.group_by_dynamic("timestamp", every=sample_rate)
            .agg(
                [
                    pl.col("latency_ms").quantile(0.99).alias("latency"),
                ]
            )
            .with_columns(
                (
                    (pl.col("timestamp") - df["timestamp"].min()).dt.total_seconds()
                    / 60
                ).alias("relative_time")
            )
        )

    def calculate_throughput_metrics(
        self,
        df: pl.DataFrame,
        sla_threshold: int = 100,
        rolling_window="1s",
    ) -> pl.DataFrame:
        """Calculate throughput metrics with SLA threshold."""
        filtered_df = df.filter(
            (pl.col("success")) & (pl.col("latency_ms") < sla_threshold)
        )
        return (
            filtered_df.rolling(index_column="timestamp", period=rolling_window)
            .agg([pl.len().alias("requests")])
            .with_columns(
                [
                    (
                        (pl.col("timestamp") - df["timestamp"].min()).dt.total_seconds()
                        / 60
                    ).alias("relative_time"),
                ]
            )
            .with_columns([pl.col("requests").alias("throughput")])
        )

    def plot_latency_comparison(
        self, data: Dict, workload_type: str, raw: bool, output_dir: str
    ):
        """Generate latency comparison plot."""
        plt.figure()

        for i, df in enumerate(data[workload_type]["original_dfs"]):
            metrics = self.calculate_latency_metrics(df, sample_rate="10s")
            plt.plot(
                metrics["relative_time"],
                metrics["latency"],
                label=f"{self.node_configs[i]} P99",
            )

        plt.xlabel("Benchmark Duration (minutes)")
        plt.ylabel("Latency (ms)")
        plt.title(f"Latency Comparison - {workload_type}")
        plt.legend()
        plt.grid(True)
        plt.savefig(f"{output_dir}/latency_{workload_type}.png")
        plt.close()

    def plot_throughput_comparison(
        self, data: Dict, workload_type: str, output_dir: str
    ):
        """Generate throughput comparison plot."""
        plt.figure()
        for i, df in enumerate(data[workload_type]["original_dfs"]):
            metrics = self.calculate_throughput_metrics(
                df, sla_threshold=100, rolling_window="1s"
            )
            plt.plot(
                metrics["relative_time"],
                metrics["throughput"],
                label=self.node_configs[i],
            )

        plt.xlabel("Benchmark Duration (minutes)")
        plt.ylabel("Throughput (req/s)")
        plt.title(f"Throughput Comparison - {workload_type}")
        plt.legend()
        plt.grid(True)
        plt.savefig(f"{output_dir}/throughput_{workload_type}.png")
        plt.close()

    def analyze_workloads(self, workloads: List[str], scenario: str, output_dir: str):
        """Analyze and plot metrics for given workloads."""
        results = {workload: {"original_dfs": []} for workload in workloads}

        for workload in workloads:
            for node_config in self.node_configs:
                csv_filepath = (
                    self.base_path / f"{scenario}/{node_config}/{workload}/metrics.csv"
                )
                if csv_filepath.exists():
                    raw_df = self.load_metrics(csv_filepath, scenario)
                    results[workload]["original_dfs"].append(raw_df)

            if results[workload]["original_dfs"]:
                self.plot_latency_comparison(results, workload, True, output_dir)
                self.plot_throughput_comparison(results, workload, output_dir)


def main():
    parser = argparse.ArgumentParser(description="Analyze Etcd performance benchmarks")
    parser.add_argument(
        "--root",
        type=str,
        required=True,
        help="Root directory containing benchmark results",
    )
    parser.add_argument(
        "--output", type=str, default="plots", help="Output directory for plots"
    )
    args = parser.parse_args()

    os.makedirs(args.output, exist_ok=True)
    analyzer = EtcdPerfAnalyzer(args.root)

    # KV Store Analysis
    kv_workloads = ["read-only", "read-heavy", "update-heavy"]
    analyzer.analyze_workloads(kv_workloads, "kv", args.output)
    # data = analyzer.preprocess_kv_metrics(workloads)
    # for workload_type, workload_data in data.items():
    #     print(workload_type)
    #     for df in workload_data["original_dfs"]:
    #         print(df.head(5))

    # Lock Store Analysis
    # workloads = ["lock-only", "lock-mixed-read", "lock-mixed-write", "lock-contention"]


if __name__ == "__main__":
    main()
