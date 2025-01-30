import polars as pl
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
        filtered_df = df.filter((pl.col("success")))
        resampled_df = filtered_df.group_by_dynamic(
            "timestamp", every=sample_rate, start_by="window"
        ).agg(
            [
                pl.col("latency_ms").quantile(0.99).alias("latency"),
            ]
        )

        min_timestamp = resampled_df["timestamp"].min()
        return resampled_df.with_columns(
            (pl.col("timestamp") - min_timestamp)
            .dt.total_seconds()
            .clip(0, 1500)
            .alias("relative_time")
        )

    def calculate_throughput_metrics(
        self,
        df: pl.DataFrame,
        sla_threshold: int = 100,
        rolling_window="1s",
        sample_rate="1m",
    ) -> pl.DataFrame:
        """Calculate throughput metrics with SLA threshold."""
        filtered_df = df.filter(
            (pl.col("success")) & (pl.col("latency_ms") < sla_threshold)
        )

        rolling_throughput_df = filtered_df.rolling(
            index_column="timestamp", period=rolling_window
        ).agg([pl.len().alias("rolling_throughput")])

        # Resample rolling throughput into 1-minute intervals
        resampled_df = rolling_throughput_df.group_by_dynamic(
            "timestamp", every=sample_rate, start_by="window"
        ).agg(
            [
                pl.col("rolling_throughput").mean().alias("throughput"),
            ]
        )
        min_timestamp = resampled_df["timestamp"].min()
        return resampled_df.with_columns(
            (pl.col("timestamp") - min_timestamp)
            .dt.total_seconds()
            .clip(0, 1500)
            .alias("relative_time")
        )

    def plot_latency_comparison(
        self, data: Dict, workload_type: str, sample_rate: str, output_dir: str
    ):
        """Generate latency comparison plot."""
        plt.figure(figsize=(20, 6))
        print(f"Analyzing latency comparison for {workload_type}")
        for i, df in enumerate(data[workload_type]["original_dfs"]):
            metrics = self.calculate_latency_metrics(df, sample_rate=sample_rate)
            plt.plot(
                metrics["relative_time"],
                metrics["latency"],
                label=f"{self.node_configs[i]} P99",
            )

        # Set x-axis ticks at 60-second intervals
        max_time = 60 * 25
        ticks = list(range(0, int(max_time) + 60, 60))
        plt.xticks(ticks, rotation=45)

        plt.xlabel("Benchmark Duration (seconds)")
        plt.ylabel("Latency (ms)")
        plt.title(f"Latency Comparison - {workload_type}")
        plt.legend()
        plt.grid(True)
        plt.savefig(f"{output_dir}/latency_{workload_type}.png")
        plt.close()

    def plot_throughput_comparison(
        self, data: Dict, workload_type: str, sample_rate: str, output_dir: str
    ):
        """Generate throughput comparison plot."""
        plt.figure(figsize=(20, 6))
        print(f"Analyzing throughput comparison for {workload_type}")
        for i, df in enumerate(data[workload_type]["original_dfs"]):
            metrics = self.calculate_throughput_metrics(
                df, sla_threshold=100, rolling_window="1s", sample_rate=sample_rate
            )
            plt.plot(
                metrics["relative_time"],
                metrics["throughput"],
                label=self.node_configs[i],
            )

        # Set x-axis ticks at 60-second intervals
        max_time = 60 * 25
        ticks = list(range(0, int(max_time) + 60, 60))
        plt.xticks(ticks, rotation=45)

        plt.xlabel("Benchmark Duration (seconds)")
        plt.ylabel("Throughput (req/s)")
        plt.title(f"Throughput Comparison - {workload_type}, (Latency < 100ms)")
        plt.legend()
        plt.grid(True)
        plt.savefig(f"{output_dir}/throughput_{workload_type}.png")
        plt.close()

    def compute_baseline_throughput(self, df_1node: pl.DataFrame) -> pl.DataFrame:
        """
        Compute cumulative requests for the 1-node setup to use as the baseline.
        """
        return (
            df_1node.filter(pl.col("success"))
            .sort("timestamp")
            .with_row_index(name="cumulative_requests")
        )

    def extract_matching_requests(
        self, df_other: pl.DataFrame, baseline_df: pl.DataFrame
    ) -> pl.DataFrame:
        """
        Extract requests from df_other at the same cumulative request milestones as the baseline.
        """
        df_other = (
            df_other.filter(pl.col("success"))
            .sort("timestamp")
            .with_row_index(name="cumulative_requests")
        )

        # Use nearest match to find timestamps where other setups reached the same request count
        aligned_df = baseline_df.join(
            df_other, on="cumulative_requests", how="left"
        ).select(
            [
                pl.col("timestamp_right").alias("aligned_timestamp"),
                pl.col("latency_ms"),
                pl.col("cumulative_requests"),
            ]
        )

        return aligned_df

    def normalize_progress_time(
        self, df_aligned: pl.DataFrame, total_requests: int
    ) -> pl.DataFrame:
        """
        Normalize progress time as a percentage (0% to 100%) of workload completion.
        """
        return df_aligned.with_columns(
            (pl.col("cumulative_requests") / total_requests * 100).alias(
                "progress_time"
            )
        )

    def resample_latency(self, df: pl.DataFrame, sample_rate) -> pl.DataFrame:
        """
        Resample the latency data into 1-minute intervals after extracting matching requests.
        """
        if "timestamp" in df.columns:
            df = df.rename({"timestamp": "aligned_timestamp"})

        # Forward fill any remaining nulls in other columns
        df = df.fill_null(strategy="forward")

        # Sort the dataframe by aligned_timestamp before grouping
        df = df.sort("aligned_timestamp")

        return df.group_by_dynamic("aligned_timestamp", every=sample_rate).agg(
            pl.col("latency_ms").quantile(0.99).alias("latency_p99"),
            pl.col("cumulative_requests").max(),
        )

    def normalize_latency_progress(
        self, df_1node, df_3node, df_5node, workload_type, sample_rate, output_dir
    ) -> Dict:
        """
        Normalize progress time and plot latency in progress time for different node setups.
        """

        # Compute baseline throughput from 1-node
        baseline_df = self.compute_baseline_throughput(df_1node)
        total_requests = baseline_df[
            "cumulative_requests"
        ].max()  # Total number of requests for normalization

        # Extract equivalent requests for 3-node and 5-node
        aligned_3node = self.extract_matching_requests(df_3node, baseline_df)
        aligned_5node = self.extract_matching_requests(df_5node, baseline_df)

        # Resample latency values
        baseline_df = self.resample_latency(baseline_df, sample_rate)
        aligned_3node = self.resample_latency(aligned_3node, sample_rate)
        aligned_5node = self.resample_latency(aligned_5node, sample_rate)

        # Normalize progress time
        baseline_df = self.normalize_progress_time(baseline_df, total_requests)
        aligned_3node = self.normalize_progress_time(aligned_3node, total_requests)
        aligned_5node = self.normalize_progress_time(aligned_5node, total_requests)

        data_dict = {
            "1-node": baseline_df,
            "3-node": aligned_3node,
            "5-node": aligned_5node,
        }

        return data_dict

    def plot_latency_fixed_throughput_comparison(
        self, data_dict, workload_type, output_dir
    ):
        """
        Plot latency in progress time with the same throughput level for different node setups.
        """
        plt.figure(figsize=(20, 6))
        print(
            f"Analyzing latency with the same level of throughput for {workload_type}"
        )
        for node_config, df in data_dict.items():
            plt.plot(
                df["progress_time"],
                df["latency_p99"],
                label=f"{node_config} P99",
            )

        # Extract the throughput level for the title
        max_throughput = max(
            df["cumulative_requests"].max() for df in data_dict.values()
        )

        plt.xlabel("Progress (%)")
        plt.ylabel("Latency (ms)")
        plt.title(f"Latency Comparison - {workload_type} with {max_throughput} reqs")
        plt.legend()
        plt.grid(True)
        plt.savefig(f"{output_dir}/latency_{workload_type}_fixed_throughput.png")
        plt.close()

    def plot_error_rate_comparison(
        self, data: Dict, workload_type: str, sample_rate: str, output_dir: str
    ):
        """Generate error rate comparison plot over time."""
        plt.figure(figsize=(20, 6))
        print(f"Analyzing error rate comparison for {workload_type}")
        for i, df in enumerate(data[workload_type]["original_dfs"]):
            # Calculate error rate per minute
            error_metrics = df.group_by_dynamic("timestamp", every=sample_rate).agg(
                [
                    (
                        pl.col("status_code")
                        .filter(
                            (pl.col("status_code") != 0) & (pl.col("status_code") != -1)
                        )
                        .len()
                        / pl.len()
                    ).alias("error_rate")
                ]
            )

            min_timestamp = error_metrics["timestamp"].min()
            error_metrics = error_metrics.with_columns(
                (pl.col("timestamp") - min_timestamp)
                .dt.total_seconds()
                .clip(0, 1500)
                .alias("relative_time")
            )

            plt.plot(
                error_metrics["relative_time"],
                error_metrics["error_rate"] * 100,  # Convert to percentage
                label=f"{self.node_configs[i]}",
            )

        # Set x-axis ticks at 60-second intervals
        max_time = 60 * 25
        ticks = list(range(0, int(max_time) + 60, 60))
        plt.xticks(ticks, rotation=45)

        plt.xlabel("Benchmark Duration (minutes)")
        plt.ylabel("Error Rate (%)")
        plt.title(f"Error Rate Comparison - {workload_type}")
        plt.legend()
        plt.grid(True)
        plt.savefig(f"{output_dir}/error_rate_{workload_type}.png")
        plt.close()

    def plot_latency_distribution(
        self, data: Dict, workload_type: str, output_dir: str
    ):
        """Generate violin plots to show latency distribution."""
        plt.figure(figsize=(12, 6))
        print(f"Analyzing latency distribution for {workload_type}")
        plot_data = []

        for df in data[workload_type]["original_dfs"]:
            filtered_df = df.filter(pl.col("success"))
            latencies = filtered_df["latency_ms"].to_numpy()
            plot_data.append(latencies)

        violin_parts = plt.violinplot(
            plot_data,
            positions=range(len(self.node_configs)),
            showmeans=True,
            showmedians=True,
        )

        for pc in violin_parts["bodies"]:
            pc.set_facecolor("skyblue")
            pc.set_edgecolor("black")
            pc.set_alpha(0.7)

        violin_parts["cmeans"].set_color("red")
        violin_parts["cmedians"].set_color("black")

        plt.xticks(range(len(self.node_configs)), self.node_configs)
        plt.xlabel("Node Configuration")
        plt.ylabel("Latency (ms)")
        plt.title(f"Latency Distribution - {workload_type}")
        plt.grid(True, axis="y")

        # Add legend for mean and median
        from matplotlib.lines import Line2D

        legend_elements = [
            Line2D([0], [0], color="red", label="Mean"),
            Line2D([0], [0], color="black", label="Median"),
        ]
        plt.legend(handles=legend_elements)

        plt.savefig(f"{output_dir}/latency_distribution_{workload_type}.png")
        plt.close()

    def infer_num_clients(self, df: pl.DataFrame) -> pl.DataFrame:
        """Infer number of clients from timestamp for lock-mixed workloads."""
        # Get the start timestamp
        start_time = df["timestamp"].min()

        # Calculate minutes elapsed since start
        df = df.with_columns(
            ((pl.col("timestamp") - start_time).dt.total_seconds() / 60)
            .floor()
            .alias("minutes_elapsed")
        )

        # Calculate number of clients (starting from 10, adding 10 every minute)
        df = df.with_columns(
            (10 + pl.col("minutes_elapsed") * 10)
            .clip(10, 250)  # Max 250 clients
            .alias("num_clients")
        )

        return df

    def analyze_load_response(self, data: Dict, workload_type: str, output_dir: str):
        """Analyze load response between number of clients and latency."""
        print(f"Analyzing load response for {workload_type}")
        plt.figure(figsize=(16, 9))

        colors = ["blue", "orange", "green"]  # One color for each node configuration

        for i, df in enumerate(data[workload_type]["original_dfs"]):
            filtered_df = df.filter(pl.col("success"))

            # Check if we need to infer num_clients
            if (
                workload_type.startswith("lock-mixed")
                and filtered_df["num_clients"].max() == 0
            ):
                print(
                    f"Inferring num_clients for {workload_type} in {self.node_configs[i]}"
                )
                filtered_df = self.infer_num_clients(filtered_df)
            # Calculate average latency and standard deviation for each client count
            agg_df = (
                filtered_df.group_by("num_clients")
                .agg(
                    [
                        pl.col("latency_ms").mean().alias("avg_latency"),
                        pl.col("latency_ms").std().alias("std_latency"),
                    ]
                )
                .sort("num_clients")
            )

            x = agg_df["num_clients"].to_numpy()
            y = agg_df["avg_latency"].to_numpy()
            yerr = agg_df["std_latency"].to_numpy()

            # Plot with error bars
            plt.errorbar(
                x,
                y,
                yerr=yerr,
                label=self.node_configs[i],
                marker="o",
                linestyle="-",
                capsize=5,
                color=colors[i],
            )

        plt.xlabel("Number of Clients")
        plt.ylabel("Mean Latency (ms)")
        plt.title(f"Load Response - {workload_type}")
        plt.legend()
        plt.grid(True)

        # Adjust layout and save
        plt.tight_layout()
        plt.savefig(f"{output_dir}/load_response_{workload_type}.png")
        plt.close()

    def analyze_kv_workloads(self, workloads: List[str], output_dir: str):
        """Analyze and plot metrics for given workloads in kv store use case."""
        results = {workload: {"original_dfs": []} for workload in workloads}
        for plot_type in [
            "latency",
            "throughput",
            "error_rate",
            "distribution",
            "scalability",
        ]:
            os.makedirs(Path(output_dir) / plot_type, exist_ok=True)
        print("Analyzing KV-Store workloads: ", workloads)
        for workload in workloads:
            print(f"Analyzing {workload}")
            for node_config in self.node_configs:
                csv_filepath = (
                    self.base_path / f"kv/{node_config}/{workload}/metrics.csv"
                )
                if csv_filepath.exists():
                    raw_df = self.load_metrics(csv_filepath, "kv")
                    results[workload]["original_dfs"].append(raw_df)

            if results[workload]["original_dfs"]:
                data_dict = self.normalize_latency_progress(
                    results[workload]["original_dfs"][0],
                    results[workload]["original_dfs"][1],
                    results[workload]["original_dfs"][2],
                    workload,
                    "5s",
                    output_dir,
                )
                self.plot_latency_fixed_throughput_comparison(
                    data_dict, workload, Path(output_dir) / "latency"
                )
                self.plot_latency_comparison(
                    results, workload, "5s", Path(output_dir) / "latency"
                )
                self.plot_throughput_comparison(
                    results, workload, "5s", Path(output_dir) / "throughput"
                )
                self.plot_latency_distribution(
                    results, workload, Path(output_dir) / "distribution"
                )
                self.plot_error_rate_comparison(
                    results, workload, "5s", Path(output_dir) / "error_rate"
                )
                self.analyze_load_response(
                    results, workload, Path(output_dir) / "scalability"
                )

    def analyze_lock_workloads(self, workloads: List[str], output_dir: str):
        """Analyze and plot metrics for given workloads in lock service use case."""
        results = {workload: {"original_dfs": []} for workload in workloads}

        for plot_type in [
            "latency",
            "throughput",
            "error_rate",
            "distribution",
            "scalability",
        ]:
            os.makedirs(Path(output_dir) / plot_type, exist_ok=True)

        print("Analyzing Lock-Service workloads: ", workloads)
        for workload in workloads:
            print(f"Analyzing {workload}")
            for node_config in self.node_configs:
                csv_filepath = (
                    self.base_path / f"lock/{node_config}/{workload}/metrics.csv"
                )
                if csv_filepath.exists():
                    raw_df = self.load_metrics(csv_filepath, "lock")
                    results[workload]["original_dfs"].append(raw_df)
            if results[workload]["original_dfs"]:
                data_dict = self.normalize_latency_progress(
                    results[workload]["original_dfs"][0],
                    results[workload]["original_dfs"][1],
                    results[workload]["original_dfs"][2],
                    workload,
                    "5s",
                    output_dir,
                )
                self.plot_latency_fixed_throughput_comparison(
                    data_dict, workload, Path(output_dir) / "latency"
                )
                self.plot_latency_comparison(
                    results, workload, "5s", Path(output_dir) / "latency"
                )
                self.plot_throughput_comparison(
                    results, workload, "5s", Path(output_dir) / "throughput"
                )
                self.plot_latency_distribution(
                    results, workload, Path(output_dir) / "distribution"
                )
                self.plot_error_rate_comparison(
                    results, workload, "5s", Path(output_dir) / "error_rate"
                )
                self.analyze_load_response(
                    results, workload, Path(output_dir) / "scalability"
                )


class SystemMetricsAnalyzer:
    node_configs = ["1-node", "3-node", "5-node"]

    def __init__(self, base_path: str):
        self.base_path = Path(base_path)

    def load_metrics(self, cpu_path, mem_path):
        """
        Load CPU and Memory utilization metrics from Google Cloud Monitoring exported CSVs.
        """
        cpu_df = pl.read_csv(
            cpu_path, try_parse_dates=True, has_header=True, skip_rows=4
        )
        mem_df = pl.read_csv(
            mem_path, try_parse_dates=True, has_header=True, skip_rows=4
        )
        return cpu_df, mem_df

    def process_metrics(self, df):
        """
        Process CPU/Memory metrics: convert timestamps and aggregate by node.
        """
        df = df.rename({"system_labels.name": "timestamp"})
        df = df.with_columns(
            pl.col("timestamp").str.replace(r" \(.*\)$", "").alias("timestamp")
        ).with_columns(
            pl.col("timestamp")
            .str.strptime(pl.Datetime, format="%a %b %d %Y %H:%M:%S GMT%z")
            .alias("timestamp"),
            pl.col(df.columns[1:]).cast(pl.Float64, strict=False),
        )
        # Compute relative time in minutes
        start_time = df["timestamp"].min()
        df = df.with_columns(
            ((pl.col("timestamp") - start_time).dt.total_seconds() / 60).alias(
                "timestamp"
            )
        )
        utilization_columns = df.columns[1:]  # Exclude timestamp
        for col in utilization_columns:
            if df[col].max() <= 1.0:  # Convert to percentage
                df = df.with_columns((pl.col(col) * 100).alias(col))
        return df

    def plot_utilization(self, df, metric_name, workload, node_config, output_path):
        """
        Plot CPU or Memory utilization for different nodes over time.
        """
        plt.figure(figsize=(12, 6))
        print(f"Analyzing {metric_name} utilization for {workload} in {node_config}")
        nodes = df.columns[1:]  # Exclude timestamp
        colors = plt.get_cmap("tab10").colors  # Use a standard color map

        for i, node in enumerate(nodes):
            color = colors[i % len(colors)]

            plt.plot(
                df["timestamp"],
                df[node],
                color=color,
                linestyle="-",
                label=f"{node}",
            )

        plt.xlabel("Time")
        plt.ylabel(f"{metric_name} Utilization (%)")
        plt.title(f"{metric_name} Utilization Over Time")
        plt.legend()
        plt.grid(True)
        plt.savefig(
            f"{output_path}/{node_config}_{workload}_{metric_name.lower()}_utilization.png"
        )
        plt.close()

    def analyze_and_plot(self, workloads, scenario, output_path):
        """
        Analyze and plot CPU & Memory utilization together.
        """
        output_path = Path(output_path) / "system"
        os.makedirs(output_path, exist_ok=True)
        print(f"Analyzing system metrics for {scenario} scenario")
        for workload in workloads:
            for node_config in self.node_configs:
                cpu_csv = (
                    self.base_path
                    / f"{scenario}/{node_config}/{workload}/CPU_Utilization.csv"
                )
                mem_csv = (
                    self.base_path
                    / f"{scenario}/{node_config}/{workload}/Memory_Utilization.csv"
                )
                cpu_df, mem_df = self.load_metrics(cpu_csv, mem_csv)
                cpu_df = self.process_metrics(cpu_df)
                mem_df = self.process_metrics(mem_df)
                self.plot_utilization(cpu_df, "CPU", workload, node_config, output_path)
                self.plot_utilization(
                    mem_df, "Memory", workload, node_config, output_path
                )


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

    print("Start analyzing Etcd performance benchmarks")
    os.makedirs(args.output, exist_ok=True)
    analyzer = EtcdPerfAnalyzer(args.root)
    sys_analyzer = SystemMetricsAnalyzer(args.root)

    # KV Store Analysis
    kv_workloads = ["read-only", "read-heavy", "update-heavy"]
    analyzer.analyze_kv_workloads(kv_workloads, args.output)
    sys_analyzer.analyze_and_plot(kv_workloads, "kv", args.output)

    # Lock Service Analysis
    lock_workloads = [
        "lock-only",
        "lock-mixed-read",
        "lock-mixed-write",
        "lock-contention",
    ]
    analyzer.analyze_lock_workloads(lock_workloads, args.output)
    sys_analyzer.analyze_and_plot(lock_workloads, "lock", args.output)
    print("Analysis completed. Plots saved in ", args.output)


if __name__ == "__main__":
    main()
